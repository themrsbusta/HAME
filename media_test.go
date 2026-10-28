package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImageArchiveAndReference(t *testing.T) {
	var b bytes.Buffer
	png.Encode(&b, image.NewRGBA(image.Rect(0, 0, 8, 8)))
	data, ext, err := decodeImageURL("data:image/png;base64," + base64.StdEncoding.EncodeToString(b.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	e := &Engine{cfg: Config{Data: t.TempDir(), MainModel: "model.gguf", MainMMProj: "projector.gguf"}}
	ref, err := e.saveImage("photo_test", data, ext, "A red object\nuncertain: true")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ref, "/hame/media?") {
		t.Fatal(ref)
	}
	files, _ := filepath.Glob(filepath.Join(e.cfg.Data, "context", "photo_test", "media", "*.yaml"))
	if len(files) != 1 {
		t.Fatal(files)
	}
	raw, _ := os.ReadFile(files[0])
	if !strings.Contains(string(raw), "description_origin: model_generated") {
		t.Fatal(string(raw))
	}
	refs := e.mediaReferences("photo_test", "red object")
	if len(refs) != 1 || !strings.Contains(refs[0].Fact, "/hame/media?") {
		t.Fatal(refs)
	}
	name := strings.TrimSuffix(filepath.Base(files[0]), ".yaml") + ".png"
	response := httptest.NewRecorder()
	e.mediaHTTP(response, httptest.NewRequest("GET", "/hame/media?session_id=photo_test&file="+name, nil))
	if response.Code != 200 || !bytes.Equal(response.Body.Bytes(), data) {
		t.Fatal(response.Code)
	}
	response = httptest.NewRecorder()
	e.mediaHTTP(response, httptest.NewRequest("GET", "/hame/media?session_id=../outside&file="+name, nil))
	if response.Code != http.StatusNotFound {
		t.Fatal(response.Code)
	}
	if mediaFilename("../../config.json") {
		t.Fatal("path traversal accepted")
	}
}
func TestImageInputs(t *testing.T) {
	for _, bad := range []string{"https://example.com/image.png", "file:///c:/secret.png", "data:image/png;base64,aW52YWxpZA=="} {
		if _, _, err := decodeImageURL(bad); err == nil {
			t.Fatal(bad)
		}
	}
	e := &Engine{}
	text, err := e.acceptMedia(context.Background(), "test", json.RawMessage(`"hello"`))
	if err != nil || text != "hello" {
		t.Fatal(text, err)
	}
	_, err = e.acceptMedia(context.Background(), "test", json.RawMessage(`[{"type":"image_url","image_url":{"url":"https://example.com/image.png"}}]`))
	if err == nil {
		t.Fatal("image accepted without server/mmproj")
	}
}
