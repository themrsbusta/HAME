package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func validateManualConfig(c Config) error {
	if c.Backend != "server" && c.Backend != "cli" {
		return errors.New("Linux/Termux: set backend to server or cli in config.json")
	}
	if c.executable() == "" {
		return errors.New("Linux/Termux: specify the explicit path to the llama executable in config.json")
	}
	for _, d := range []string{c.MainDevice, c.CompilerDevice} {
		if d == "" || d == "auto" {
			return errors.New("Linux/Termux: main_device and compiler_device must be explicit; use none for CPU")
		}
	}
	if c.GPULayers == "" || c.GPULayers == "auto" {
		return errors.New("Linux/Termux: gpu_layers must be explicit; use 0 for CPU")
	}
	if len(c.Capabilities) == 0 {
		return errors.New("Linux/Termux: declare llama_capabilities according to your llama --help output; HAME does not probe automatically")
	}
	return nil
}
func manualProfile(c Config) (*Profile, error) {
	if err := validateManualConfig(c); err != nil {
		return nil, err
	}
	info, err := os.Stat(c.executable())
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("executable not found: %s", c.executable())
	}
	if info.Mode()&0111 == 0 {
		return nil, fmt.Errorf("executable permission denied: %s", c.executable())
	}
	p := &Profile{Executable: c.executable(), Backend: c.Backend, Version: c.RunnerVersion, Flags: map[string]bool{}, Devices: []Device{}}
	if p.Version == "" {
		p.Version = "capabilities declared by the user; not probed"
	}
	for _, flag := range c.Capabilities {
		if !strings.HasPrefix(flag, "--") {
			return nil, fmt.Errorf("invalid capability %q; use long names from --help", flag)
		}
		p.Flags[flag] = true
	}
	p.Resident = c.Backend == "server" && p.Flags["--slots"] && p.Flags["--slot-save-path"] && p.Flags["--cache-ram"] && p.Flags["--no-cache-idle-slots"]
	return p, nil
}
