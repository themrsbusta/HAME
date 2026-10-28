package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func validateManualConfig(c Config) error {
	if c.Backend != "server" && c.Backend != "cli" {
		return errors.New("Linux/Termux: informe backend server ou cli no config.json")
	}
	if c.executable() == "" {
		return errors.New("Linux/Termux: informe o caminho explícito do executável llama no config.json")
	}
	for _, d := range []string{c.MainDevice, c.CompilerDevice} {
		if d == "" || d == "auto" {
			return errors.New("Linux/Termux: main_device e compiler_device explícitos são obrigatórios; use none para CPU")
		}
	}
	if c.GPULayers == "" || c.GPULayers == "auto" {
		return errors.New("Linux/Termux: gpu_layers explícito é obrigatório; use 0 para CPU")
	}
	if len(c.Capabilities) == 0 {
		return errors.New("Linux/Termux: declare llama_capabilities conforme --help do seu llama; HAME não executa sondagem automática")
	}
	return nil
}
func manualProfile(c Config) (*Profile, error) {
	if err := validateManualConfig(c); err != nil {
		return nil, err
	}
	info, err := os.Stat(c.executable())
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("executável não encontrado: %s", c.executable())
	}
	if info.Mode()&0111 == 0 {
		return nil, fmt.Errorf("executável sem permissão de execução: %s", c.executable())
	}
	p := &Profile{Executable: c.executable(), Backend: c.Backend, Version: c.RunnerVersion, Flags: map[string]bool{}, Devices: []Device{}}
	if p.Version == "" {
		p.Version = "capacidades declaradas pelo usuário; sem sondagem"
	}
	for _, flag := range c.Capabilities {
		if !strings.HasPrefix(flag, "--") {
			return nil, fmt.Errorf("capacidade inválida %q; use nomes longos de --help", flag)
		}
		p.Flags[flag] = true
	}
	p.Resident = c.Backend == "server" && p.Flags["--slots"] && p.Flags["--slot-save-path"] && p.Flags["--cache-ram"] && p.Flags["--no-cache-idle-slots"]
	return p, nil
}
