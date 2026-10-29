package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNativeUICacheSurvivesWorkerUnavailable(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch r.URL.Path {
		case "/":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte(`<html><head><script type="module" src="/_app/native.js"></script></head><body>Native UI</body></html>`))
		case "/_app/native.js":
			w.Header().Set("Content-Type", "text/javascript")
			w.Write([]byte(`console.log('native')`))
		default:
			http.NotFound(w, r)
		}
	}))
	e := &Engine{cfg: Config{WebUI: "upstream", Backend: "server", MainURL: server.URL, Data: t.TempDir()}, client: server.Client(), main: &Worker{url: server.URL}}
	for _, path := range []string{"/", "/_app/native.js"} {
		r := httptest.NewRequest("GET", path, nil)
		w := httptest.NewRecorder()
		e.handler().ServeHTTP(w, r)
		if w.Code != 200 {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		if path == "/" && !strings.Contains(w.Body.String(), `/hame/shim.js`) {
			t.Fatal("shim not injected")
		}
	}
	server.Close()
	e.main = nil
	// Simulate a new HAME process with the same persistent cache and no worker.
	copy := &Engine{cfg: e.cfg, client: server.Client()}
	for _, path := range []string{"/chat/conversation-1", "/_app/native.js"} {
		w := httptest.NewRecorder()
		copy.handler().ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Code != 200 {
			t.Fatalf("cached %s unavailable: %d %s", path, w.Code, w.Body.String())
		}
	}
	if requests != 2 {
		t.Fatalf("unexpected upstream access: %d", requests)
	}
	for _, path := range []string{"/tools", "/models/load", "/hame/reload", "/../config.json", "/a%2f..%2fconfig.json"} {
		if _, ok := uiAssetRoute(path); ok {
			t.Errorf("API or traversal treated as asset: %s", path)
		}
	}
}

func TestNativeUIInjectionPrecedesModules(t *testing.T) {
	input := []byte(`<HTML><HEAD><script type="module" src="app.js"></script></HEAD></HTML>`)
	out, err := injectHAMEShim(input)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Index(s, "/hame/shim.js") > strings.Index(s, `type="module"`) {
		t.Fatal("shim runs too late")
	}
	if !strings.Contains(s, `<base href="/">`) {
		t.Fatal("SPA asset base missing")
	}
	if _, err = injectHAMEShim([]byte("not HTML")); err == nil {
		t.Fatal("invalid HTML accepted")
	}
}

func TestLegacyToolsArgsEnableGateway(t *testing.T) {
	root := t.TempDir()
	c := Config{MainExtraArgs: `--threads 8 --tools all --chat-template-file "C:\models with spaces\template.jinja"`}
	if err := c.normalizeToolArgs(root); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.Tools, []string{"all"}) || c.ToolCWD != root {
		t.Fatalf("tools not enabled: %+v", c)
	}
	args, err := parseStartupArgs(c.MainExtraArgs)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(args, []string{"--threads", "8", "--chat-template-file", `C:\models with spaces\template.jinja`}) {
		t.Fatal(args)
	}
	for _, raw := range []string{`--tools`, `--tools=`, `--tools all --tools read_file`} {
		v := Config{MainExtraArgs: raw}
		if v.normalizeToolArgs(root) == nil {
			t.Errorf("accepted %s", raw)
		}
	}
	v := Config{Tools: []string{"read_file"}, MainExtraArgs: "--tools all"}
	if v.normalizeToolArgs(root) == nil {
		t.Fatal("conflicting tools accepted")
	}
}

func TestModelIdentityUsesSelectedGGUFFilename(t *testing.T) {
	e := &Engine{cfg: Config{MainModel: `C:\Jewel\Jewel-Uncensored-Q4_K_M.gguf`, Context: 4096}, modelName: "gemma-4-E4B_q4_0-it", props: map[string]any{"model_alias": "gemma-4-E4B_q4_0-it"}}
	for _, route := range []string{"/props", "/v1/models", "/models"} {
		data, _ := json.Marshal(e.metadata(route))
		if !strings.Contains(string(data), "Jewel-Uncensored-Q4_K_M.gguf") || strings.Contains(string(data), "gemma-4-E4B_q4_0-it") {
			t.Fatalf("wrong identity in %s: %s", route, data)
		}
	}
}

func TestToolDiscoveryMatchesNativeUIArrayContract(t *testing.T) {
	data, err := os.ReadFile("testdata/llama-b11946-tools.json")
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	defer server.Close()
	e := &Engine{cfg: Config{Tools: []string{"all"}}, client: server.Client(), main: &Worker{url: server.URL}}
	w := httptest.NewRecorder()
	e.toolsHTTP(w, httptest.NewRequest("GET", "/tools", nil))
	var descriptors []ToolDescriptor
	if json.Unmarshal(w.Body.Bytes(), &descriptors) != nil || len(descriptors) != 7 {
		t.Fatalf("not a native descriptor array: %s", w.Body.String())
	}
	for _, d := range descriptors {
		if !d.UsesCWD || d.Definition.Function.Name != d.Tool {
			t.Fatalf("missing UI metadata: %+v", d)
		}
	}
	blocked := httptest.NewRecorder()
	e.toolsHTTP(blocked, httptest.NewRequest("POST", "/tools", nil))
	if blocked.Code != 405 {
		t.Fatal("direct native executor exposed")
	}
}

func TestRetrieveOriginalFactAfterSummaryOmittedIt(t *testing.T) {
	e, a, _ := engineFixture(t, "constrained")
	enabled := true
	e.cfg.ContinuousHistory = &enabled
	e.cfg.RetrievalTokens = 512
	if err := atomicJSON(filepath.Join(e.cfg.Data, "context", "test", "0001.json"), []Message{{Role: "user", Content: "O codinome de Aurora é PANTERA-719."}, {Role: "assistant", Content: "Anotado."}}); err != nil {
		t.Fatal(err)
	}
	if err := atomicJSON(filepath.Join(e.cfg.Data, "context", "other", "0001.json"), []Message{{Role: "user", Content: "Aurora secreto OTHER-SESSION-999"}}); err != nil {
		t.Fatal(err)
	}
	messages := []Message{{Role: "system", Content: "Resumo sem o codinome."}, {Role: "user", Content: "Qual era o codinome de Aurora?"}}
	worker := &Worker{url: a.server.URL}
	got, err := e.withRetrievedHistory(context.Background(), worker, "test", messages[1].Content, messages)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(got)
	if !strings.Contains(string(data), "PANTERA-719") || strings.Contains(string(data), "OTHER-SESSION-999") {
		t.Fatal(string(data))
	}
	usage, err := e.usage(context.Background(), worker, got, e.reserve())
	if err != nil || usage.Total > e.cfg.Context {
		t.Fatal(usage, err)
	}
	if len(messages) != 2 {
		t.Fatal("retrieval mutated persisted session")
	}
	files, _ := filepath.Glob(filepath.Join(e.cfg.Data, "retrieval-audit", "test", "*.json"))
	if len(files) != 1 {
		t.Fatal("missing retrieval evidence")
	}
}

func TestContinuousHistoryCompactsEvenWithManualPolicy(t *testing.T) {
	e, _, compiler := engineFixture(t, "constrained")
	enabled := true
	e.cfg.ContinuousHistory = &enabled
	e.cfg.Policy = "on_demand"
	e.cfg.Context = 2048
	e.cfg.Threshold = 40
	if _, err := e.chat(context.Background(), "test", strings.Repeat("small detail ", 80), false); err != nil {
		t.Fatal(err)
	}
	if _, err := e.chat(context.Background(), "test", strings.Repeat("another detail ", 80), false); err != nil {
		t.Fatal(err)
	}
	if compiler.compiles == 0 {
		t.Fatal("continuous history hit manual policy instead of compacting")
	}
	history, err := e.history("test")
	if err != nil || len(history) != 4 {
		t.Fatal("original history lost", len(history), err)
	}
	before := compiler.compiles
	if _, err = e.chat(context.Background(), "test", strings.Repeat("oversize ", 10000), false); err == nil {
		t.Fatal("oversized individual input accepted")
	}
	if compiler.compiles != before {
		t.Fatal("oversize input triggered unnecessary compaction")
	}
}
