package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeviceParsing(t *testing.T) {
	d := parseDevices("Available devices:\n Vulkan0: AMD Radeon RX 6600 (8176 MiB, 7000 MiB free)\n Vulkan1: AMD Radeon(TM) Graphics (32487 MiB, 30000 MiB free)\n CUDA0: NVIDIA GPU (8192 MiB)\n SYCL0.1: Intel GPU (16384 MiB)\n")
	if len(d) != 4 || d[1].Name != "AMD Radeon(TM) Graphics" || !strings.HasPrefix(d[1].Memory, "32487") {
		t.Fatal(d)
	}
}
func TestPortableConfig(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	b, _ := json.Marshal(map[string]any{"backend": "cli", "llama_cli": "llama/llama-cli.exe", "main_model": "models/main.gguf", "compiler_model": "models/small.gguf", "data_dir": "data"})
	os.WriteFile(p, b, 0600)
	c, err := load(p)
	if err != nil || c.CLI != filepath.Join(dir, "llama", "llama-cli.exe") || c.Data != filepath.Join(dir, "data") {
		t.Fatal(c, err)
	}
}
func TestCLIOutputIsolation(t *testing.T) {
	prompt := "line one\nline two\\n"
	raw := "Loading model...\nbuild : b123\nmodel : test\n\n> " + prompt + "\nanswer\n\nExiting...\n"
	answer, err := cleanCLIOutput(raw, prompt)
	if err != nil || answer != "answer" {
		t.Fatal(answer, err)
	}
	if _, err = cleanCLIOutput("a new incompatible banner", "question"); err == nil {
		t.Fatal("unknown CLI format accepted")
	}
}
func TestRequiredCapabilityRejection(t *testing.T) {
	p := &Profile{Backend: "server", Version: "future", Flags: map[string]bool{}}
	if validateProfile(p) == nil {
		t.Fatal("incompatible executable accepted")
	}
}
func TestModelIdentity(t *testing.T) {
	e := &Engine{cfg: Config{MainModel: filepath.Join("models", "my-model.gguf"), Context: 8192}}
	if e.displayModel() != "my-model.gguf" {
		t.Fatal(e.displayModel())
	}
	models := e.metadata("/v1/models")
	b, _ := json.Marshal(models)
	if strings.Contains(string(b), `"id":"HAME"`) || !strings.Contains(string(b), "my-model.gguf") {
		t.Fatal(string(b))
	}
}

func TestCLIFileIsolation(t *testing.T) {
	prompt := "multi\nline text\\n"
	answer, err := cleanCLIFile("User:\n"+prompt+"\n\nAssistant:\nresponse\n\n", prompt)
	if err != nil || answer != "response" {
		t.Fatal(answer, err)
	}
	if _, err = cleanCLIFile("changed protocol", prompt); err == nil {
		t.Fatal("unknown file format accepted")
	}
}
