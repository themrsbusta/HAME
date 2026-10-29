package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

func (e *Engine) runtimeHTTP(w http.ResponseWriter, r *http.Request) {
	e.metaMu.RLock()
	id := e.activeSession
	audit := append([]AuditEvent{}, e.audit...)
	e.metaMu.RUnlock()
	var status CycleStatus
	if id != "" {
		b, _ := os.ReadFile(e.path(id, ".status.json"))
		_ = json.Unmarshal(b, &status)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"version": Version, "backend": e.cfg.Backend, "mode": e.cfg.Mode, "memory_policy": e.cfg.Policy, "threshold_percent": e.cfg.Threshold, "session_id": id, "status": status, "audit": audit, "compiled_memory_max_tokens": e.memoryLimit(), "prefill": e.prefillEnabled() && e.cfg.Backend != "cli"})
}
func (e *Engine) compactHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "POST required", 405)
		return
	}
	var q struct {
		ID      string `json:"session_id"`
		Recover bool   `json:"recover"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&q); err != nil {
		http.Error(w, "invalid JSON", 400)
		return
	}
	if q.ID == "" {
		e.metaMu.RLock()
		q.ID = e.activeSession
		e.metaMu.RUnlock()
	}
	if !safeID.MatchString(q.ID) {
		http.Error(w, "selecione uma conversa existente", 400)
		return
	}
	if err := e.acquire(r.Context()); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer e.mu.Unlock()
	s, err := e.loadSession(q.ID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if len(s.Messages) == 0 {
		http.Error(w, "sessão não encontrada", 404)
		return
	}
	err = e.compactLocked(r.Context(), q.ID, "", &s, q.Recover, "manual")
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		e.status(q.ID, "error", err)
		w.WriteHeader(502)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"session_id": q.ID, "compactions": s.Compactions, "usage": s.Usage, "mode": e.cfg.Mode})
}
func memoryClient(c Config, id string, compact, recover bool) error {
	client := &http.Client{Timeout: 20 * time.Minute}
	health, err := client.Get("http://" + c.Listen + "/health")
	if err != nil {
		return err
	}
	var h struct {
		Version string `json:"version"`
	}
	err = json.NewDecoder(health.Body).Decode(&h)
	health.Body.Close()
	if err != nil || h.Version != Version {
		return fmt.Errorf("o serviço em %s não é HAME %s", c.Listen, Version)
	}
	url := "http://" + c.Listen + "/hame/runtime"
	var req *http.Request
	if compact {
		url = "http://" + c.Listen + "/hame/compact"
		b, _ := json.Marshal(map[string]any{"session_id": id, "recover": recover})
		req, _ = http.NewRequest("POST", url, strings.NewReader(string(b)))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, _ = http.NewRequest("GET", url, nil)
	}
	r, err := client.Do(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if r.StatusCode != 200 {
		return fmt.Errorf("HAME HTTP %d: %s", r.StatusCode, b)
	}
	fmt.Println(string(b))
	return nil
}
func (e *Engine) testExecutionMode(ctx context.Context) error {
	if e.cfg.Backend == "cli" {
		w, err := e.ensure(ctx, false)
		if err != nil {
			return err
		}
		if _, _, _, err = e.countMessages(ctx, w, []Message{{Role: "user", Content: "HAME tokenizer compatibility check."}}); err != nil {
			return err
		}
		_, err = e.generate(ctx, w, []Message{{Role: "user", Content: "Reply only OK."}}, 16, "")
		return err
	}
	if e.cfg.Mode == "resident" {
		if e.main == nil || e.compiler == nil {
			return fmt.Errorf("Resident não manteve os dois workers")
		}
		return nil
	}
	w, err := e.ensure(ctx, false)
	if err != nil {
		return err
	}
	if _, _, _, err = e.countMessages(ctx, w, []Message{{Role: "user", Content: "HAME constrained check."}}); err != nil {
		return err
	}
	e.main.stop()
	e.main = nil
	e.record("main_stopped", "mode_test", 0, 0, 0)
	compiler, err := e.ensure(ctx, true)
	if err != nil {
		return err
	}
	if _, _, _, err = e.countMessages(ctx, compiler, []Message{{Role: "user", Content: "HAME compiler check."}}); err != nil {
		return err
	}
	e.compiler.stop()
	e.compiler = nil
	e.record("compiler_stopped", "mode_test", 0, 0, 0)
	_, err = e.ensure(ctx, false)
	return err
}
