package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func explicitConfig() Config {
	return Config{Backend: "server", Llama: "llama-server", MainDevice: "none", CompilerDevice: "none", GPULayers: "0", Capabilities: []string{"--model", "--ctx-size", "--gpu-layers", "--host", "--port", "--parallel"}}
}
func TestManualConfigurationRejectsDiscovery(t *testing.T) {
	c := explicitConfig()
	if err := validateManualConfig(c); err != nil {
		t.Fatal(err)
	}
	cases := []Config{c, c, c, c, c}
	cases[0].MainDevice = "auto"
	cases[1].CompilerDevice = ""
	cases[2].GPULayers = "auto"
	cases[3].Capabilities = nil
	cases[4].Llama = ""
	for _, v := range cases {
		if validateManualConfig(v) == nil {
			t.Fatal(v)
		}
	}
}
func TestManualValidationDoesNotRunExecutable(t *testing.T) {
	if !manualPlatform {
		t.Skip("Unix manual startup")
	}
	dir := t.TempDir()
	marker := filepath.Join(dir, "was-executed")
	runner := filepath.Join(dir, "llama-server")
	os.WriteFile(runner, []byte("#!/bin/sh\necho unexpected > "+marker+"\nexit 99\n"), 0700)
	model := filepath.Join(dir, "model.gguf")
	os.WriteFile(model, []byte("test"), 0600)
	c := explicitConfig()
	c.Llama = runner
	c.MainModel = model
	c.CompilerModel = model
	c.Mode = "constrained"
	e := &Engine{cfg: c}
	if err := e.validate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("automatic executable invocation occurred")
	}
}
