package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testCall(id, name, args string) ToolCall {
	var c ToolCall
	c.ID = id
	c.Type = "function"
	c.Function.Name = name
	c.Function.Arguments = args
	return c
}
func testDefinition() ToolDefinition {
	var d ToolDefinition
	d.Type = "function"
	d.Function.Name = "exec_shell_command"
	d.Function.Parameters = json.RawMessage(`{"type":"object","properties":{"command":{"type":"string"},"timeout":{"type":"integer"}},"required":["command"]}`)
	return d
}

func TestToolLoopExecutionAndHistory(t *testing.T) {
	root := t.TempDir()
	calls, turns := 0, 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tools":
			if r.Method == "GET" {
				ioJSON(w, []any{map[string]any{"tool": "exec_shell_command", "type": "server", "definition": testDefinition()}})
				return
			}
			calls++
			if r.Header.Get("x-tool-cwd") != root {
				t.Error("wrong cwd")
			}
			var q struct {
				Tool   string         `json:"tool"`
				Params map[string]any `json:"params"`
			}
			json.NewDecoder(r.Body).Decode(&q)
			if q.Tool != "exec_shell_command" || q.Params["timeout"] != float64(5) {
				t.Errorf("unexpected executor request: %+v", q)
			}
			ioJSON(w, map[string]string{"plain_text_response": "marker\n[exit code: 0]"})
		case "/apply-template":
			ioJSON(w, map[string]string{"prompt": "prompt"})
		case "/tokenize":
			ioJSON(w, map[string]any{"tokens": []int{1, 2}})
		case "/v1/chat/completions":
			turns++
			var q struct {
				Messages []Message        `json:"messages"`
				Tools    []ToolDefinition `json:"tools"`
			}
			json.NewDecoder(r.Body).Decode(&q)
			if len(q.Tools) != 1 {
				t.Error("definitions not sent")
			}
			answer := Message{Role: "assistant", ToolCalls: []ToolCall{testCall("call-1", "exec_shell_command", `{"command":"echo marker","timeout":999}`)}}
			finish := "tool_calls"
			if turns == 2 {
				last := q.Messages[len(q.Messages)-1]
				if last.Role != "tool" || last.ToolCallID != "call-1" || !strings.Contains(last.Content, "exit code: 0") {
					t.Errorf("tool result not returned: %+v", last)
				}
				answer = Message{Role: "assistant", Content: "executed"}
				finish = "stop"
			}
			ioJSON(w, map[string]any{"choices": []any{map[string]any{"message": answer, "finish_reason": finish}}})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	e := &Engine{cfg: Config{Tools: []string{"exec_shell_command"}, ToolCWD: root, ToolTimeout: 5, ToolMaxCalls: 2, Context: 4096, Output: 128, Data: root}, client: server.Client()}
	text, trace, err := e.chatWithTools(context.Background(), &Worker{url: server.URL}, "test", []Message{{Role: "user", Content: "run"}})
	if err != nil || text != "executed" || calls != 1 || len(trace) != 3 {
		t.Fatalf("text=%q calls=%d trace=%+v err=%v", text, calls, trace, err)
	}
	files, _ := filepath.Glob(filepath.Join(root, "tool-audit", "test", "*.json"))
	if len(files) != 1 {
		t.Fatal("missing durable tool audit")
	}
	b, _ := os.ReadFile(files[0])
	var audit ToolAudit
	json.Unmarshal(b, &audit)
	if audit.Stage != "completed" || !strings.Contains(audit.Result, "exit code: 0") {
		t.Fatalf("bad audit: %s", b)
	}
	// Tool messages and IDs survive persisted session replay.
	if err = atomicJSON(e.path("test", ".json"), Session{Messages: trace}); err != nil {
		t.Fatal(err)
	}
	s, err := e.loadSession("test")
	if err != nil || s.Messages[1].ToolCallID != "call-1" || s.Messages[0].ToolCalls[0].Function.Name != "exec_shell_command" {
		t.Fatalf("history lost: %+v %v", s, err)
	}
}

func TestToolCallsRejectMalformedOrDisabled(t *testing.T) {
	for _, c := range []ToolCall{testCall("id", "unlisted", `{}`), testCall("", "exec_shell_command", `{"command":"echo"}`), testCall("id", "exec_shell_command", `null`), testCall("id", "exec_shell_command", `{"command":5}`), testCall("id", "exec_shell_command", `{"command":"echo","extra":true}`), testCall("id", "exec_shell_command", `{}`)} {
		if _, err := validateToolCall(c, []ToolDefinition{testDefinition()}); err == nil {
			t.Fatalf("accepted invalid call: %+v", c)
		}
	}
}

func TestToolLoopLimitsAndNeverExecutesFreeText(t *testing.T) {
	for _, mode := range []string{"limit", "duplicate", "truncated", "text", "batch-invalid"} {
		t.Run(mode, func(t *testing.T) {
			calls, turns := 0, 0
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/tools":
					if r.Method == "POST" {
						calls++
						ioJSON(w, map[string]string{"plain_text_response": "ok"})
					} else {
						ioJSON(w, []any{map[string]any{"tool": "exec_shell_command", "type": "server", "definition": testDefinition()}})
					}
				case "/apply-template":
					ioJSON(w, map[string]string{"prompt": "p"})
				case "/tokenize":
					ioJSON(w, map[string]any{"tokens": []int{1}})
				case "/v1/chat/completions":
					turns++
					call := testCall("same", "exec_shell_command", `{"command":"echo"}`)
					if mode == "limit" {
						call.ID = string(rune('a' + turns))
					}
					m := Message{Role: "assistant", ToolCalls: []ToolCall{call}}
					finish := "tool_calls"
					if mode == "truncated" {
						finish = "length"
					}
					if mode == "text" {
						m = Message{Role: "assistant", Content: "cmd /c echo pretend"}
						finish = "stop"
					}
					if mode == "batch-invalid" {
						m.ToolCalls = append(m.ToolCalls, testCall("invalid", "other", `{}`))
					}
					ioJSON(w, map[string]any{"choices": []any{map[string]any{"message": m, "finish_reason": finish}}})
				}
			}))
			defer s.Close()
			e := &Engine{cfg: Config{Tools: []string{"exec_shell_command"}, ToolCWD: t.TempDir(), ToolTimeout: 1, ToolMaxCalls: 2, Context: 4096, Output: 128, Data: t.TempDir()}, client: s.Client()}
			_, _, err := e.chatWithTools(context.Background(), &Worker{url: s.URL}, "test", nil)
			want := 0
			if mode == "limit" {
				want = 2
			}
			if mode == "duplicate" {
				want = 1
			}
			if calls != want {
				t.Fatalf("executed %d commands, want %d", calls, want)
			}
			if mode != "text" && err == nil {
				t.Fatal("missing error")
			}
			if mode == "text" && err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestPrivilegedChatRejectsCrossSite(t *testing.T) {
	e := &Engine{cfg: Config{Tools: []string{"all"}}}
	called := false
	h := e.toolRequestGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	for _, origin := range []string{"https://evil.example", "null", "http://127.0.0.1:9999"} {
		r := httptest.NewRequest("POST", "http://127.0.0.1:8088/v1/chat/completions", strings.NewReader(`{}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if called || w.Code != 403 {
			t.Fatal("cross-site request accepted")
		}
	}
	r := httptest.NewRequest("POST", "http://127.0.0.1:8088/v1/chat/completions", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if !called {
		t.Fatalf("local request rejected: %d", w.Code)
	}
}

func TestToolsOnlyOnPrincipal(t *testing.T) {
	e := &Engine{cfg: Config{Tools: []string{"all"}}, profile: &Profile{Flags: map[string]bool{}}}
	if !strings.Contains(strings.Join(e.commonArgs(false), " "), "--tools all") {
		t.Fatal("principal tools missing")
	}
	if strings.Contains(strings.Join(e.commonArgs(true), " "), "--tools") {
		t.Fatal("compiler received tools")
	}
}

func TestInstalledNativeToolSchema(t *testing.T) {
	data, err := os.ReadFile("testdata/llama-b11946-tools.json")
	if err != nil {
		t.Fatal(err)
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer s.Close()
	e := &Engine{cfg: Config{Tools: []string{"all"}}, client: s.Client()}
	defs, err := e.toolDefinitions(context.Background(), &Worker{url: s.URL})
	if err != nil || len(defs) != 7 {
		t.Fatalf("native discovery: %d, %v", len(defs), err)
	}
	if _, err = validateToolCall(testCall("edit", "edit_file", `{"path":"a.txt","edits":[{"old_text":"a","new_text":"b"}]}`), defs); err != nil {
		t.Fatal(err)
	}
	if _, err = validateToolCall(testCall("edit", "edit_file", `{"path":"a.txt","edits":[{"old_text":"a"}]}`), defs); err == nil {
		t.Fatal("accepted incomplete nested edit")
	}
	if _, err = validateToolCall(testCall("info", "get_info", `{}`), defs); err != nil {
		t.Fatal(err)
	}
}

func TestToolTransportFailureNotRetried(t *testing.T) {
	calls := 0
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		http.Error(w, "failure after possible execution", 500)
	}))
	defer s.Close()
	root := t.TempDir()
	e := &Engine{cfg: Config{Tools: []string{"exec_shell_command"}, ToolCWD: root, ToolTimeout: 1, Data: root}, client: s.Client()}
	_, err := e.executeTool(context.Background(), &Worker{url: s.URL}, "test", testCall("id", "exec_shell_command", `{"command":"echo"}`), map[string]any{"command": "echo"})
	if err == nil || calls != 1 || !strings.Contains(err.Error(), "não repetir automaticamente") {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
	files, _ := filepath.Glob(filepath.Join(root, "tool-audit", "test", "*.json"))
	if len(files) != 1 {
		t.Fatal("missing audit")
	}
	data, _ := os.ReadFile(files[0])
	var audit ToolAudit
	json.Unmarshal(data, &audit)
	if audit.Stage != "error" || audit.Params["timeout"] != float64(1) {
		t.Fatalf("bad audit: %s", data)
	}
}
