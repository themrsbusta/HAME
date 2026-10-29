package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestBrowserShimContractWhenNodeAvailable(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js unavailable; run node --test hame-shim.test.cjs separately")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, node, "--test", "hame-shim.test.cjs").CombinedOutput()
	if err != nil {
		t.Fatalf("browser shim tests: %v\n%s", err, out)
	}
}

func TestInterfaceUTF8(t *testing.T) {
	data, err := assets.ReadFile("webui/index.html")
	if err != nil {
		t.Fatal(err)
	}
	if !utf8.Valid(data) {
		t.Fatal("interface is not UTF-8")
	}
	for _, text := range []string{"memória por ocupação", "Recuperar histórico", "Falha na compactação", "Memória renovada; histórico preservado.", "orçamento"} {
		if !bytes.Contains(data, []byte(text)) {
			t.Errorf("missing correctly encoded UI text %q", text)
		}
	}
	for _, bad := range []string{"\u00c3", "\u00c2\u00b7", "\ufffd"} {
		if bytes.Contains(data, []byte(bad)) {
			t.Errorf("corrupted UI text contains %q", bad)
		}
	}
	charset := bytes.Index(data, []byte(`<meta charset="utf-8"`))
	if charset < 0 || charset > 1024 || charset > bytes.Index(data, []byte("<script>")) {
		t.Fatal("UTF-8 must be declared early, before inline scripts")
	}
	e := &Engine{}
	for _, path := range []string{"/", "/chat/encoding-test"} {
		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		e.handler().ServeHTTP(w, r)
		if !strings.Contains(w.Header().Get("Content-Type"), "charset=utf-8") {
			t.Errorf("%s lacks HTTP UTF-8 charset", path)
		}
		if !strings.Contains(w.Body.String(), "memória por ocupação") {
			t.Errorf("%s lost accents", path)
		}
	}
}

func TestWindowsPowerShellUTF8BOM(t *testing.T) {
	root := filepath.Join("..", "..", "scripts")
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || filepath.Ext(path) != ".ps1" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.HasPrefix(data, []byte{0xef, 0xbb, 0xbf}) || !utf8.Valid(data) {
			t.Errorf("%s needs UTF-8 BOM for Windows PowerShell 5.1", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
