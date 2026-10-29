package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
)

//go:embed hame-shim.js
var hameShim []byte

func shimHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Write(hameShim)
}

type UIAsset struct {
	Path        string `json:"path"`
	ContentType string `json:"content_type"`
	SHA256      string `json:"sha256"`
	Body        []byte `json:"body"`
}

func (e *Engine) uiCachePath(asset string) string {
	identity := e.cfg.Llama + "|" + e.cfg.MainURL + "|" + Version
	if e.profile != nil {
		identity += "|" + e.profile.Version
	}
	if info, err := os.Stat(e.cfg.Llama); err == nil {
		identity += fmt.Sprintf("|%d|%d", info.Size(), info.ModTime().UnixNano())
	}
	return filepath.Join(e.cfg.Data, "webui-cache", fmt.Sprintf("%x", sha256.Sum256([]byte(identity))), fmt.Sprintf("%x.json", sha256.Sum256([]byte(asset))))
}

func (e *Engine) cachedUIAsset(asset string) (UIAsset, bool) {
	var value UIAsset
	b, err := os.ReadFile(e.uiCachePath(asset))
	if err != nil || json.Unmarshal(b, &value) != nil || value.Path != asset || value.SHA256 != fmt.Sprintf("%x", sha256.Sum256(value.Body)) {
		return value, false
	}
	return value, true
}

func (e *Engine) upstreamAsset(ctx context.Context, asset string) (UIAsset, error) {
	if cached, ok := e.cachedUIAsset(asset); ok {
		return cached, nil
	}
	e.uiMu.Lock()
	defer e.uiMu.Unlock()
	if cached, ok := e.cachedUIAsset(asset); ok {
		return cached, nil
	}
	if err := e.acquire(ctx); err != nil {
		return UIAsset{}, err
	}
	defer e.mu.Unlock()
	w, err := e.ensure(ctx, false)
	if err != nil {
		return UIAsset{}, err
	}
	if !loopbackWorker(w) {
		return UIAsset{}, errors.New("WebUI upstream exige worker HTTP em 127.0.0.1")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.url+asset, nil)
	if err != nil {
		return UIAsset{}, err
	}
	client := *e.client
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	r, err := client.Do(req)
	if err != nil {
		return UIAsset{}, err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return UIAsset{}, fmt.Errorf("asset %s: HTTP %d", asset, r.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, (32<<20)+1))
	if err != nil {
		return UIAsset{}, err
	}
	if len(body) > 32<<20 {
		return UIAsset{}, errors.New("asset WebUI excede 32 MiB")
	}
	value := UIAsset{Path: asset, ContentType: r.Header.Get("Content-Type"), SHA256: fmt.Sprintf("%x", sha256.Sum256(body)), Body: body}
	if err = atomicJSON(e.uiCachePath(asset), value); err != nil {
		return UIAsset{}, fmt.Errorf("cache da WebUI: %w", err)
	}
	return value, nil
}

func injectHAMEShim(html []byte) ([]byte, error) {
	lower := bytes.ToLower(html)
	start := bytes.Index(lower, []byte("<head"))
	if start < 0 {
		return nil, errors.New("HTML upstream sem head")
	}
	end := bytes.IndexByte(html[start:], '>')
	if end < 0 {
		return nil, errors.New("head upstream inválido")
	}
	end += start + 1
	// Relative asset paths from the native UI must also work on SPA chat routes.
	injection := []byte(`<meta charset="utf-8"><base href="/"><script src="/hame/shim.js"></script>`)
	out := make([]byte, 0, len(html)+len(injection))
	out = append(out, html[:end]...)
	out = append(out, injection...)
	out = append(out, html[end:]...)
	out = bytes.ReplaceAll(out, []byte("base: new URL('.', location).pathname.slice(0, -1)"), []byte("base: ''"))
	return out, nil
}

func uiAssetRoute(raw string) (string, bool) {
	decoded, err := url.PathUnescape(raw)
	if err != nil || strings.Contains(decoded, "\\") || strings.Contains(decoded, "..") || strings.ContainsRune(decoded, 0) {
		return "", false
	}
	p := path.Clean("/" + strings.TrimPrefix(decoded, "/"))
	if p == "/props" || p == "/tools" || p == "/models" || p == "/health" || p == "/slots" {
		return "", false
	}
	if p == "/" || p == "/index.html" || strings.HasPrefix(p, "/chat/") {
		return "/", true
	}
	if p == "/service-worker.js" {
		return p, true
	}
	switch strings.ToLower(path.Ext(p)) {
	case ".js", ".css", ".svg", ".ico", ".png", ".jpg", ".jpeg", ".webp", ".woff", ".woff2", ".ttf", ".map", ".webmanifest":
		return p, true
	case ".json":
		if strings.HasPrefix(p, "/_app/") || p == "/manifest.json" {
			return p, true
		}
	}
	if !strings.Contains(p, ".") && !strings.HasPrefix(p, "/v1/") && !strings.HasPrefix(p, "/hame/") && !strings.HasPrefix(p, "/models/") {
		return "/", true
	}
	return "", false
}

func (e *Engine) upstreamUIHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "GET or HEAD required", 405)
		return
	}
	asset, ok := uiAssetRoute(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if asset == "/service-worker.js" || asset == "/sw.js" {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		io.WriteString(w, `self.addEventListener('install',()=>self.skipWaiting());self.addEventListener('activate',event=>event.waitUntil(self.clients.claim()));`)
		return
	}
	value, err := e.upstreamAsset(r.Context(), asset)
	if err != nil {
		http.Error(w, "HAME: "+err.Error(), http.StatusBadGateway)
		return
	}
	body := value.Body
	if asset == "/" {
		body, err = injectHAMEShim(body)
		if err != nil {
			http.Error(w, err.Error(), 502)
			return
		}
		value.ContentType = "text/html; charset=utf-8"
	}
	if value.ContentType != "" {
		w.Header().Set("Content-Type", value.ContentType)
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method == http.MethodGet {
		w.Write(body)
	}
}
