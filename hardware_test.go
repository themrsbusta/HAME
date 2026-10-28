package main

import (
	"testing"
)

func TestAutomaticGPUSelection(t *testing.T) {
	if automaticDevice(nil).ID != "none" {
		t.Fatal("CPU fallback")
	}
	devices := []Device{{ID: "Vulkan1", Name: "integrated", Memory: "2048 MiB, 1024 MiB free"}, {ID: "Vulkan0", Name: "discrete", Memory: "8192 MiB, 7000 MiB free"}}
	if automaticDevice(devices).ID != "Vulkan1" {
		t.Fatal("backend device priority not preserved")
	}
}
func TestProjectorPrincipalOnly(t *testing.T) {
	e := &Engine{cfg: Config{MainMMProj: "projector with spaces.gguf", MainModel: "main", CompilerModel: "compiler", Context: 8192, GPULayers: "auto"}}
	args := e.commonArgs(false)
	i := -1
	for j, arg := range args {
		if arg == "--mmproj" {
			i = j
		}
	}
	if i < 0 || args[i+1] != e.cfg.MainMMProj {
		t.Fatal(args)
	}
	for _, arg := range e.commonArgs(true) {
		if arg == "--mmproj" {
			t.Fatal("compiler received principal projector")
		}
	}
}
