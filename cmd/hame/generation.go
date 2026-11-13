package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type GenerationPolicy struct {
	OutputTokens          string `json:"output_tokens"`
	ReserveTokens         string `json:"reserve_tokens"`
	ArtificialOutputLimit bool   `json:"artificial_output_limit"`
}

type ContextPolicy struct {
	Capacity          string           `json:"capacity"`
	KVCacheMonitoring bool             `json:"kv_cache_monitoring"`
	Compaction        CompactionPolicy `json:"compaction"`
}

type CompactionPolicy struct {
	Enabled  bool              `json:"enabled"`
	Trigger  CompactionTrigger `json:"trigger"`
	Strategy []string          `json:"strategy"`
}

type CompactionTrigger struct {
	Type             string `json:"type"`
	ThresholdPercent int    `json:"threshold_percent"`
}

type CompletionPolicy struct {
	SuccessRequires   []string        `json:"success_requires"`
	TruncatedResponse TruncatedPolicy `json:"truncated_response"`
}

type TruncatedPolicy struct {
	MarkCompleted   bool `json:"mark_completed"`
	AttemptRecovery bool `json:"attempt_recovery"`
}

type SafetyPolicy struct {
	ExplicitCancellation       bool `json:"explicit_cancellation"`
	ResourceExhaustionHandling bool `json:"resource_exhaustion_handling"`
	DetectStalledGeneration    bool `json:"detect_stalled_generation"`
}

var (
	errCompactionBoundary = errors.New("KV occupancy reached compaction boundary")
	errStalledGeneration  = errors.New("llama-server generation stalled without progress")
)

const generationStallTimeout = 5 * time.Minute

type chatCompletionResult struct {
	Message            Message
	FinishReason       string
	PromptTokens       int
	CompletionTokens   int
	CompactionBoundary bool
}

type generationRecovery func(messages []Message, partial Message, responseSoFar, reason string) (*Worker, []Message, error)

type slotSnapshot struct {
	Context             int
	PromptTokens        int
	PromptProcessed     int
	Decoded             int
	Processing          bool
	HasProcessing       bool
	HasProgressCounters bool
}

func (e *Engine) compactionThreshold() int {
	if e.cfg.ContextRuntime.Compaction.Trigger.ThresholdPercent > 0 {
		return e.cfg.ContextRuntime.Compaction.Trigger.ThresholdPercent
	}
	return 75
}

func (e *Engine) contextLimit() int {
	e.metaMu.RLock()
	n := e.runtimeContext
	e.metaMu.RUnlock()
	if e.cfg.ContextRuntime.Capacity == "model_runtime" && n > 0 {
		return n
	}
	return e.cfg.Context
}

func numberField(value any) int {
	switch n := value.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		v, _ := n.Int64()
		return int(v)
	default:
		return 0
	}
}

func (e *Engine) readSlotSnapshot(ctx context.Context, w *Worker) (slotSnapshot, bool) {
	var result slotSnapshot
	if w == nil || w.cli {
		return result, false
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.url+"/slots", nil)
	if err != nil {
		return result, false
	}
	client := e.client
	if client == nil {
		client = http.DefaultClient
	}
	r, err := client.Do(req)
	if err != nil {
		return result, false
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return result, false
	}
	var slots []map[string]any
	if json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&slots) != nil {
		return result, false
	}
	for _, slot := range slots {
		if id, ok := slot["id"]; ok && numberField(id) != 0 {
			continue
		}
		result.Context = numberField(slot["n_ctx"])
		result.PromptTokens = numberField(slot["n_prompt_tokens"])
		result.PromptProcessed = numberField(slot["n_prompt_tokens_processed"])
		result.Decoded = numberField(slot["n_decoded"])
		result.HasProgressCounters = slot["n_prompt_tokens_processed"] != nil || slot["n_decoded"] != nil
		result.Processing, result.HasProcessing = slot["is_processing"].(bool)
		if result.Context > 0 {
			e.metaMu.Lock()
			e.runtimeContext = result.Context
			e.runtimeContextFrom = "llama-server /slots"
			e.metaMu.Unlock()
		}
		return result, true
	}
	return result, false
}

func (e *Engine) generationBudget(ctx context.Context, w *Worker, msg []Message) (int, bool, TokenUsage, error) {
	if e.cfg.Generation.ArtificialOutputLimit || !e.cfg.ContextRuntime.Compaction.Enabled {
		return e.cfg.Output, false, TokenUsage{}, nil
	}
	u, err := e.usage(ctx, w, msg, 0)
	if err != nil {
		return 0, false, TokenUsage{}, err
	}
	capacity := e.contextLimit()
	if e.cfg.ContextRuntime.Capacity == "model_runtime" {
		if snapshot, ok := e.readSlotSnapshot(ctx, w); ok && snapshot.Context > 0 {
			capacity = snapshot.Context
		}
	}
	if capacity < 1 {
		return 0, false, u, errors.New("capacidade de contexto do runtime não foi informada")
	}
	threshold := e.compactionThreshold()
	if threshold < 10 || threshold > 99 {
		return 0, false, u, errors.New("limite de ocupação inválido")
	}
	occupied := u.Prompt
	if !u.Exact {
		occupied += u.Overhead
	}
	boundary := capacity * threshold / 100
	remaining := boundary - occupied
	if remaining < 1 {
		return 0, true, u, errCompactionBoundary
	}
	if e.cfg.Generation.ArtificialOutputLimit {
		if e.cfg.Output < 1 {
			return 0, false, u, errors.New("artificial_output_limit requer output_tokens numérico positivo")
		}
		if e.cfg.Output < remaining {
			return e.cfg.Output, false, u, nil
		}
	}
	return remaining, true, u, nil
}

func (e *Engine) chatCompletion(ctx context.Context, w *Worker, id string, msg []Message, defs []ToolDefinition) (chatCompletionResult, error) {
	limit, boundary, usage, err := e.generationBudget(ctx, w, msg)
	if err != nil {
		return chatCompletionResult{}, err
	}
	if w.cli {
		return chatCompletionResult{}, errors.New("geração automática de resposta exige backend server; use limite fixo no CLI")
	}
	body := map[string]any{"messages": msg, "max_tokens": limit, "temperature": 0.2, "cache_prompt": true, "id_slot": 0}
	if len(defs) > 0 {
		body["tools"] = defs
		body["tool_choice"] = "auto"
	}
	b, err := json.Marshal(body)
	if err != nil {
		return chatCompletionResult{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.url+"/v1/chat/completions", bytes.NewReader(b))
	if err != nil {
		return chatCompletionResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	data, err := e.monitoredGeneration(ctx, w, id, req)
	if err != nil {
		return chatCompletionResult{}, err
	}
	var response struct {
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Choices []struct {
			Message      Message `json:"message"`
			FinishReason string  `json:"finish_reason"`
		} `json:"choices"`
	}
	if err = json.Unmarshal(data, &response); err != nil {
		return chatCompletionResult{}, err
	}
	if len(response.Choices) != 1 {
		return chatCompletionResult{}, errors.New("resposta LLM sem escolha única")
	}
	choice := response.Choices[0]
	choice.Message.Role = "assistant"
	result := chatCompletionResult{Message: choice.Message, FinishReason: choice.FinishReason, PromptTokens: response.Usage.PromptTokens, CompletionTokens: response.Usage.CompletionTokens, CompactionBoundary: boundary}
	if usage.Prompt > 0 {
		e.record("generation_segment", "limit derived from runtime KV occupancy", usage.Prompt, usage.Prompt+response.Usage.CompletionTokens, 0)
	}
	return result, nil
}

func (e *Engine) monitoredGeneration(ctx context.Context, w *Worker, id string, req *http.Request) ([]byte, error) {
	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	req = req.WithContext(requestCtx)
	type responseResult struct {
		data []byte
		err  error
	}
	done := make(chan responseResult, 1)
	go func() {
		r, err := e.inferenceHTTPClient().Do(req)
		if err != nil {
			done <- responseResult{err: err}
			return
		}
		defer r.Body.Close()
		data, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			done <- responseResult{err: readErr}
			return
		}
		if r.StatusCode != http.StatusOK {
			done <- responseResult{data: data, err: fmt.Errorf("LLM HTTP %d: %.500s", r.StatusCode, data)}
			return
		}
		done <- responseResult{data: data}
	}()

	if !e.cfg.Safety.DetectStalledGeneration && !e.cfg.ContextRuntime.KVCacheMonitoring {
		result := <-done
		return result.data, result.err
	}
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	lastProgress := time.Now()
	lastCount := -1
	var stallDetected bool
	for {
		select {
		case result := <-done:
			if stallDetected {
				return nil, errStalledGeneration
			}
			return result.data, result.err
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			snapshot, ok := e.readSlotSnapshot(requestCtx, w)
			if !ok {
				continue
			}
			if snapshot.Context > 0 && e.cfg.ContextRuntime.KVCacheMonitoring {
				used := snapshot.PromptTokens + snapshot.Decoded
				usage := TokenUsage{Prompt: used, Limit: snapshot.Context, Total: used, Exact: true, Method: "llama-server /slots KV occupancy"}
				usage.Percent = float64(used) * 100 / float64(snapshot.Context)
				e.publishUsage(id, usage)
			}
			progress := snapshot.PromptProcessed + snapshot.Decoded
			if snapshot.HasProgressCounters && progress != lastCount {
				lastCount = progress
				lastProgress = time.Now()
			}
			if e.cfg.Safety.DetectStalledGeneration && snapshot.HasProcessing && time.Since(lastProgress) >= generationStallTimeout {
				stallDetected = true
				cancel()
			}
		}
	}
}

func isResourceExhaustion(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	for _, marker := range []string{"context is full", "context full", "exceed context", "kv cache", "no available space", "out of memory", "allocation failed"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func tailRunes(text string, limit int) string {
	if limit < 1 {
		return ""
	}
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[len(runes)-limit:])
}
