package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func ioJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

type fixture struct {
	invalidOnce                               bool
	mu                                        sync.Mutex
	tokens, answers, states, compiles, erases int
	invalidState, invalidCompiler             bool
	last                                      []Message
	server                                    *httptest.Server
}

func newFixture(t *testing.T, compiler bool) *fixture {
	t.Helper()
	f := &fixture{}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		switch r.URL.Path {
		case "/health":
			ioJSON(w, map[string]string{"status": "ok"})
		case "/props":
			ioJSON(w, map[string]string{"model_alias": "test-model.gguf"})
		case "/slots":
			ioJSON(w, []any{map[string]any{"id": 0, "is_processing": false, "n_prompt_tokens": f.tokens}})
		case "/slots/0":
			n := f.tokens
			f.tokens = 0
			f.erases++
			ioJSON(w, map[string]int{"n_erased": n})
		case "/apply-template":
			var q struct {
				Messages []Message `json:"messages"`
			}
			json.NewDecoder(r.Body).Decode(&q)
			b, _ := json.Marshal(q.Messages)
			ioJSON(w, map[string]string{"prompt": "<BOS>" + string(b) + "<ASSISTANT>"})
		case "/tokenize":
			var q struct {
				Content string `json:"content"`
			}
			json.NewDecoder(r.Body).Decode(&q)
			ids := make([]int, len([]byte(q.Content))/4+1)
			ioJSON(w, map[string]any{"tokens": ids})
		case "/completion":
			var q struct {
				Prompt string `json:"prompt"`
			}
			json.NewDecoder(r.Body).Decode(&q)
			f.tokens = len(q.Prompt)/4 + 1
			ioJSON(w, map[string]string{"content": ""})
		case "/v1/chat/completions":
			var q struct {
				Messages []Message `json:"messages"`
			}
			json.NewDecoder(r.Body).Decode(&q)
			f.last = q.Messages
			b, _ := json.Marshal(q.Messages)
			f.tokens = len(b)/4 + 20
			content := "ORIGINAL-739"
			if compiler {
				f.compiles++
				content = `{"system_instruction":"Continue accurately","context_bundle":{"retrieved_knowledge":[{"source":"archive","fact":"ORIGINAL-739"}],"conversation_lineage":["user continuity"]},"response_format":"text"}`
				if f.invalidCompiler || (f.invalidOnce && f.compiles == 1) {
					content = "invalid"
				}
			} else if len(q.Messages) > 0 && strings.Contains(q.Messages[0].Content, "extracting semantic") {
				f.states++
				content = `{"session_id":"test","current_goal":"Remember the original code","unresolved_question":"","last_summary":"The code is ORIGINAL-739","search_intent":"Retrieve relevant original information"}`
				if f.invalidState {
					content = `{"current_goal":"missing fields"}`
				}
			} else {
				f.answers++
			}
			ioJSON(w, map[string]any{"choices": []any{map[string]any{"message": Message{Role: "assistant", Content: content}}}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.server.Close)
	return f
}
func engineFixture(t *testing.T, mode string) (*Engine, *fixture, *fixture) {
	a := newFixture(t, false)
	b := newFixture(t, true)
	yes := true
	e := &Engine{cfg: Config{Backend: "server", Mode: mode, MainURL: a.server.URL, CompilerURL: b.server.URL, Data: t.TempDir(), Context: 8192, Output: 128, Reserve: 256, TemplateReserve: 64, Threshold: 90, Policy: "hybrid", MemoryMax: 512, Prefill: &yes}, client: a.server.Client()}
	t.Cleanup(e.close)
	return e, a, b
}
func TestBelowThresholdKeepsMainAndDoesNotCompile(t *testing.T) {
	e, a, b := engineFixture(t, "constrained")
	for i := 0; i < 3; i++ {
		if _, err := e.chat(context.Background(), "test", "small request", false); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			continue
		}
	}
	main := e.main
	if main == nil || b.compiles != 0 || a.states != 0 || e.compiler != nil {
		t.Fatalf("unnecessary compaction: %d/%d", b.compiles, a.states)
	}
	if _, err := e.chat(context.Background(), "test", "again", false); err != nil {
		t.Fatal(err)
	}
	if e.main != main {
		t.Fatal("main restarted below threshold")
	}
	h, _ := e.history("test")
	if len(h) != 8 {
		t.Fatal(len(h))
	}
}
func TestResidentCompactionClearsSlotAndPreservesWorker(t *testing.T) {
	e, a, b := engineFixture(t, "resident")
	if err := e.initialize(context.Background(), false); err != nil {
		t.Fatal(err)
	}
	if e.main == nil || e.compiler == nil {
		t.Fatal("both workers not resident")
	}
	main, compiler := e.main, e.compiler
	if _, err := e.chat(context.Background(), "test", "remember code", false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.chat(context.Background(), "test", "retrieve", true); err != nil {
		t.Fatal(err)
	}
	if e.main != main || e.compiler != compiler {
		t.Fatal("resident worker replaced")
	}
	if a.states != 1 || a.erases != 2 || b.compiles == 0 {
		t.Fatal(a.states, a.erases, b.compiles)
	}
	h, _ := e.history("test")
	if len(h) != 4 {
		t.Fatal("history changed")
	}
	var s Session
	data, _ := os.ReadFile(e.path("test", ".json"))
	json.Unmarshal(data, &s)
	if s.Compactions != 1 || s.Compiled == nil || len(s.Messages) != 3 {
		t.Fatal(s)
	}
	if !strings.Contains(a.last[0].Content, "compiled memory") {
		t.Fatal("compiled prompt not injected")
	}
	found := false
	for _, event := range e.audit {
		if event.Kind == "slot_erased" && event.After == 0 && event.Erased > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("erase evidence absent")
	}
}
func TestConstrainedSwapsOnlyWhenCompacting(t *testing.T) {
	e, a, b := engineFixture(t, "constrained")
	if _, err := e.chat(context.Background(), "test", "first", false); err != nil {
		t.Fatal(err)
	}
	main := e.main
	if _, err := e.chat(context.Background(), "test", "second", false); err != nil {
		t.Fatal(err)
	}
	if e.main != main || b.compiles != 0 {
		t.Fatal("swapped too early")
	}
	if _, err := e.chat(context.Background(), "test", "manual", true); err != nil {
		t.Fatal(err)
	}
	if e.main == main || e.compiler != nil || a.states != 1 || b.compiles == 0 {
		t.Fatal("incorrect constrained lifecycle")
	}
}
func TestSaturationAndOnDemandPolicies(t *testing.T) {
	for _, policy := range []string{"automatic", "hybrid", "on_demand"} {
		t.Run(policy, func(t *testing.T) {
			e, a, b := engineFixture(t, "constrained")
			e.cfg.Context = 2048
			e.cfg.Threshold = 40
			e.cfg.MemoryMax = 256
			e.cfg.Policy = policy
			for i := 0; i < 3; i++ {
				if _, err := e.chat(context.Background(), "test", strings.Repeat("payload ", 90), false); err != nil {
					t.Fatal(err)
				}
			}
			if policy == "on_demand" {
				if b.compiles != 0 || a.states != 0 {
					t.Fatal("on demand auto-compiled")
				}
			} else if b.compiles == 0 {
				t.Fatal("threshold never triggered")
			}
		})
	}
}
func TestCompilerFailurePreservesSessionAndOriginals(t *testing.T) {
	e, _, b := engineFixture(t, "resident")
	if _, err := e.chat(context.Background(), "test", "keep original", false); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(e.path("test", ".json"))
	b.invalidCompiler = true
	if _, err := e.chat(context.Background(), "test", "compact", true); err == nil {
		t.Fatal("invalid compiler accepted")
	}
	after, _ := os.ReadFile(e.path("test", ".json"))
	if string(before) != string(after) {
		t.Fatal("active session replaced on failure")
	}
	h, _ := e.history("test")
	if len(h) != 2 {
		t.Fatal("history lost")
	}
}
func TestInvalidSemanticStateRejected(t *testing.T) {
	e, a, b := engineFixture(t, "constrained")
	if _, err := e.chat(context.Background(), "test", "first", false); err != nil {
		t.Fatal(err)
	}
	a.invalidState = true
	if _, err := e.chat(context.Background(), "test", "compact", true); err == nil {
		t.Fatal("invalid state accepted")
	}
	if b.compiles != 0 {
		t.Fatal("compiler ran without semantic state")
	}
}
func TestOversizedMessageNeverSent(t *testing.T) {
	e, a, _ := engineFixture(t, "constrained")
	if _, err := e.chat(context.Background(), "test", strings.Repeat("X", 40000), false); err == nil {
		t.Fatal("oversized accepted")
	}
	if a.answers != 0 || a.states != 0 {
		t.Fatal("overflow sent to model")
	}
	h, _ := e.history("test")
	if len(h) != 0 {
		t.Fatal("invalid history written")
	}
}
func TestManualRecoveryAndIncrementalSources(t *testing.T) {
	e, _, _ := engineFixture(t, "resident")
	if _, err := e.chat(context.Background(), "test", "remember", false); err != nil {
		t.Fatal(err)
	}
	call := func(full bool) {
		t.Helper()
		b := strings.NewReader(`{"session_id":"test","recover":` + map[bool]string{true: "true", false: "false"}[full] + `}`)
		w := httptest.NewRecorder()
		e.handler().ServeHTTP(w, httptest.NewRequest("POST", "/hame/compact", b))
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	call(false)
	call(false)
	foundDelta := false
	for _, x := range e.audit {
		if x.Kind == "compiled" && x.Before == 0 {
			foundDelta = true
		}
	}
	if !foundDelta {
		t.Fatal("unchanged sources recompiled")
	}
	call(true)
	last := 0
	for _, x := range e.audit {
		if x.Kind == "compiled" {
			last = x.Before
		}
	}
	if last == 0 {
		t.Fatal("recovery skipped original archive")
	}
}
func TestReloadAfterInterruptionPreservesLineage(t *testing.T) {
	e, _, _ := engineFixture(t, "constrained")
	if _, err := e.chat(context.Background(), "test", "original", false); err != nil {
		t.Fatal(err)
	}
	e.close()
	e.main = nil
	e.compiler = nil
	if _, err := e.chat(context.Background(), "test", "continue", false); err != nil {
		t.Fatal(err)
	}
	h, _ := e.history("test")
	if len(h) != 4 {
		t.Fatal(len(h))
	}
}
func TestAPIAndOfficialUI(t *testing.T) {
	e, _, _ := engineFixture(t, "constrained")
	for _, body := range []string{`{}`, `{"session_id":"../x","messages":[{"role":"user","content":"x"}]}`} {
		w := httptest.NewRecorder()
		e.handler().ServeHTTP(w, httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body)))
		if w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	for _, url := range []string{"/", "/chat/example", "/props", "/v1/models", "/hame/runtime"} {
		w := httptest.NewRecorder()
		e.handler().ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		if w.Code != 200 {
			t.Fatal(url, w.Code)
		}
	}
}
func TestResidentUnavailableIfSlotEraseUnsupported(t *testing.T) {
	e, _, _ := engineFixture(t, "resident")
	unsupported := httptest.NewServer(http.NotFoundHandler())
	defer unsupported.Close()
	e.cfg.MainURL = unsupported.URL
	if err := e.initialize(context.Background(), false); err == nil {
		t.Fatal("fake Resident support")
	}
}

func TestAutomaticPolicyRequiresManualRecoveryAction(t *testing.T) {
	e, _, b := engineFixture(t, "constrained")
	e.cfg.Policy = "automatic"
	if _, err := e.chatRequest(context.Background(), "test", "recover prior facts", false, true); err == nil {
		t.Fatal("automatic policy silently performed semantic retrieval")
	}
	if b.compiles != 0 {
		t.Fatal("unexpected recovery")
	}
}

func TestCommittedExchangeReplayedAfterSnapshotInterruption(t *testing.T) {
	e, _, _ := engineFixture(t, "constrained")
	if _, err := e.chat(context.Background(), "test", "first", false); err != nil {
		t.Fatal(err)
	}
	s, err := e.loadSession("test")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(e.cfg.Data, "context", "test")
	name := "99999999999999999999.json"
	pair := []Message{{Role: "user", Content: "committed before crash"}, {Role: "assistant", Content: "saved reply"}}
	if err = atomicJSON(filepath.Join(dir, name), pair); err != nil {
		t.Fatal(err)
	}
	restored, err := e.loadSession("test")
	if err != nil || len(restored.Messages) != len(s.Messages)+2 || restored.LastArchive != name {
		t.Fatal(restored, err)
	}
	h, _ := e.history("test")
	if len(h) != 4 {
		t.Fatal("original exchange lost")
	}
}

func TestLiteralIdentifiersCannotBeTranslated(t *testing.T) {
	if len(unknownIDs("RESIDENT-091", "RESIDENTE-091")) != 1 {
		t.Fatal("changed identifier not detected")
	}
	if len(unknownIDs("RESIDENTE-091", "source RESIDENTE-091")) != 0 {
		t.Fatal("valid identifier rejected")
	}
}

func TestCompilerRetriesInvalidContractWithoutChangingHistory(t *testing.T) {
	e, _, b := engineFixture(t, "resident")
	if _, err := e.chat(context.Background(), "test", "first", false); err != nil {
		t.Fatal(err)
	}
	b.invalidOnce = true
	if _, err := e.chat(context.Background(), "test", "manual", true); err != nil {
		t.Fatal(err)
	}
	if b.compiles != 2 {
		t.Fatal("expected one corrective attempt", b.compiles)
	}
	h, _ := e.history("test")
	if len(h) != 4 {
		t.Fatal("history lost")
	}
}

func TestRecoveryCommandPreservesOriginalText(t *testing.T) {
	e, _, _ := engineFixture(t, "resident")
	if _, err := e.chat(context.Background(), "test", "first", false); err != nil {
		t.Fatal(err)
	}
	text := "/recuperar original code"
	if _, err := e.chat(context.Background(), "test", text, false); err != nil {
		t.Fatal(err)
	}
	h, _ := e.history("test")
	if h[len(h)-2].Content != text {
		t.Fatal("original command text lost")
	}
}
