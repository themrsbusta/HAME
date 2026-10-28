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
	"time"
)

func waitCycle(t *testing.T, e *Engine) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := e.acquire(ctx); err != nil {
		t.Fatal(err)
	}
	e.mu.Unlock()
}
func TestEveryTurnResetCompileAndFullHistory(t *testing.T) {
	var mu sync.Mutex
	order := []string{}
	mainRequests := [][]Message{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		switch r.URL.Path {
		case "/slots/0":
			order = append(order, "erase")
			ioJSON(w, map[string]any{"n_erased": 42})
			return
		case "/apply-template":
			order = append(order, "template")
			ioJSON(w, map[string]string{"prompt": "compiled memory"})
			return
		case "/completion":
			order = append(order, "prefill")
			ioJSON(w, map[string]string{"content": ""})
			return
		}
		var q struct {
			Messages []Message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&q)
		content := "AZUL-739"
		if strings.Contains(q.Messages[0].Content, "Context Compiler") {
			order = append(order, "compile")
			content = `{"system_instruction":"answer","context_bundle":{"retrieved_knowledge":[{"source":"context","fact":"AZUL-739"}],"conversation_lineage":["AZUL-739"]},"response_format":"text"}`
		} else {
			order = append(order, "answer")
			mainRequests = append(mainRequests, q.Messages)
		}
		ioJSON(w, map[string]any{"choices": []any{map[string]any{"message": Message{"assistant", content}}}})
	}))
	defer srv.Close()
	e := &Engine{cfg: Config{Mode: "resident", MainURL: srv.URL, CompilerURL: srv.URL, Data: t.TempDir(), Context: 8192, Output: 1024}, client: srv.Client()}
	defer e.close()
	for _, question := range []string{"Memorize AZUL-739", "Qual código?"} {
		reply, err := e.chat(context.Background(), "test", question, false)
		if err != nil || reply != "AZUL-739" {
			t.Fatalf("%s %v", reply, err)
		}
		waitCycle(t, e)
	}
	mu.Lock()
	defer mu.Unlock()
	if strings.Join(order, ",") != "answer,erase,compile,template,prefill,answer,erase,compile,template,prefill" {
		t.Fatal(order)
	}
	if len(mainRequests[1]) != 2 || mainRequests[1][0].Role != "system" || mainRequests[1][1].Content != "Qual código?" {
		t.Fatal("raw history forwarded", mainRequests[1])
	}
	h, err := e.history("test")
	if err != nil || len(h) != 4 {
		t.Fatalf("history %v %v", h, err)
	}
	var s Session
	b, _ := os.ReadFile(e.path("test", ".json"))
	json.Unmarshal(b, &s)
	if len(s.Messages) != 1 {
		t.Fatal("active context not replaced")
	}
	entries, _ := os.ReadDir(filepath.Join(e.cfg.Data, "context", "test"))
	if len(entries) != 2 {
		t.Fatal("exchange not persisted atomically")
	}
}
func ioJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
func TestAnswerArrivesBeforeCompilation(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/slots/") {
			ioJSON(w, map[string]int{"n_erased": 1})
			return
		}
		var q struct {
			Messages []Message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&q)
		content := "saved answer"
		if len(q.Messages) > 0 && strings.Contains(q.Messages[0].Content, "Context Compiler") {
			close(started)
			<-release
			content = "invalid JSON"
		}
		ioJSON(w, map[string]any{"choices": []any{map[string]any{"message": Message{"assistant", content}}}})
	}))
	defer srv.Close()
	e := &Engine{cfg: Config{Mode: "resident", MainURL: srv.URL, CompilerURL: srv.URL, Data: t.TempDir(), Context: 8192, Output: 512}, client: srv.Client()}
	reply, err := e.chat(context.Background(), "test", "question", false)
	if err != nil || reply != "saved answer" {
		t.Fatal(reply, err)
	}
	<-started
	h, _ := e.history("test")
	if len(h) != 2 {
		t.Fatal("response not saved before compile")
	}
	close(release)
	waitCycle(t, e)
	b, _ := os.ReadFile(e.path("test", ".status.json"))
	if !strings.Contains(string(b), "error") {
		t.Fatal("compiler failure hidden")
	}
	e.close()
}
func TestAPIAndOfficialUI(t *testing.T) {
	e := &Engine{}
	for _, body := range []string{`{}`, `{"session_id":"../x","messages":[{"role":"user","content":"x"}]}`} {
		w := httptest.NewRecorder()
		e.handler().ServeHTTP(w, httptest.NewRequest("POST", "/v1/chat/completions", strings.NewReader(body)))
		if w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	for _, url := range []string{"/", "/chat/example", "/props", "/v1/models"} {
		w := httptest.NewRecorder()
		e.handler().ServeHTTP(w, httptest.NewRequest("GET", url, nil))
		if w.Code != 200 {
			t.Fatal(url, w.Code)
		}
	}
	w := httptest.NewRecorder()
	e.handler().ServeHTTP(w, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(w.Body.String(), "_app/immutable/") {
		t.Fatal("official UI missing")
	}
}
func TestInvalidSession(t *testing.T) {
	e := &Engine{}
	if _, err := e.chat(context.Background(), "../escape", "x", false); err == nil {
		t.Fatal("path traversal")
	}
}

func TestShutdownCancelsBackgroundCompiler(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/slots/") {
			ioJSON(w, map[string]int{"n_erased": 1})
			return
		}
		var q struct {
			Messages []Message `json:"messages"`
		}
		json.NewDecoder(r.Body).Decode(&q)
		if len(q.Messages) > 0 && strings.Contains(q.Messages[0].Content, "Context Compiler") {
			close(started)
			select {
			case <-r.Context().Done():
			case <-release:
			}
			return
		}
		ioJSON(w, map[string]any{"choices": []any{map[string]any{"message": Message{"assistant", "OK"}}}})
	}))
	defer srv.Close()
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e := &Engine{cfg: Config{Mode: "resident", MainURL: srv.URL, CompilerURL: srv.URL, Data: t.TempDir(), Context: 8192, Output: 1024}, client: srv.Client(), lifecycle: ctx}
	if _, err := e.chat(context.Background(), "shutdown", "question", false); err != nil {
		t.Fatal(err)
	}
	<-started
	cancel()
	closed := make(chan struct{})
	go func() { e.close(); close(closed) }()
	select {
	case <-closed:
	case <-time.After(3 * time.Second):
		t.Fatal("shutdown blocked on compiler")
	}
	waitCycle(t, e)
}
