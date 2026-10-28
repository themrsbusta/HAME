package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Device struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Memory string `json:"memory"`
}
type Profile struct {
	Executable string          `json:"executable"`
	Backend    string          `json:"backend"`
	Version    string          `json:"version"`
	Devices    []Device        `json:"devices"`
	Flags      map[string]bool `json:"flags"`
}

func cleanEnv() []string {
	var env []string
	for _, item := range os.Environ() {
		key := strings.ToUpper(strings.SplitN(item, "=", 2)[0])
		if key == "GGML_VK_VISIBLE_DEVICES" || strings.HasPrefix(key, "LLAMA_ARG_") {
			continue
		}
		env = append(env, item)
	}
	return env
}
func capture(ctx context.Context, path string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Env = cleanEnv()
	var b bytes.Buffer
	cmd.Stdout = &b
	cmd.Stderr = &b
	if err := startOwned(cmd); err != nil {
		return nil, err
	}
	err := cmd.Wait()
	if err != nil {
		return b.Bytes(), fmt.Errorf("%s: %w: %.700s", filepath.Base(path), err, b.Bytes())
	}
	return b.Bytes(), nil
}

var devicePattern = regexp.MustCompile(`^\s*([A-Za-z][A-Za-z0-9_.-]*):\s*(.+)\s+\(([^()]*)\)\s*$`)

func parseDevices(text string) []Device {
	devices := []Device{}
	for _, line := range strings.Split(text, "\n") {
		m := devicePattern.FindStringSubmatch(strings.TrimSpace(line))
		if m != nil {
			devices = append(devices, Device{m[1], strings.TrimSpace(m[2]), m[3]})
		}
	}
	return devices
}
func probe(ctx context.Context, path, backend string) (*Profile, error) {
	if backend != "server" && backend != "cli" {
		return nil, errors.New("backend must be server or cli")
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("executável não encontrado: %s", path)
	}
	help, err := capture(ctx, path, "--help")
	if err != nil {
		return nil, err
	}
	version, err := capture(ctx, path, "--version")
	if err != nil {
		return nil, err
	}
	profile := &Profile{Executable: path, Backend: backend, Version: strings.TrimSpace(string(version)), Flags: map[string]bool{}, Devices: []Device{}}
	for _, flag := range regexp.MustCompile(`--[a-z][a-z0-9-]+`).FindAllString(string(help), -1) {
		profile.Flags[flag] = true
	}
	if profile.Flags["--list-devices"] {
		data, er := capture(ctx, path, "--list-devices")
		if er != nil {
			return nil, er
		}
		profile.Devices = parseDevices(string(data))
	}
	return profile, nil
}
func validateProfile(p *Profile) error {
	required := []string{"--model", "--ctx-size", "--gpu-layers"}
	if p.Backend == "server" {
		required = append(required, "--host", "--port", "--parallel")
	} else {
		required = append(required, "--file", "--system-prompt-file", "--single-turn", "--simple-io", "--no-display-prompt", "--log-disable", "--no-show-timings", "--output-file", "--n-predict", "--no-escape")
	}
	for _, f := range required {
		if !p.Flags[f] {
			return fmt.Errorf("llama %s incompatível: parâmetro necessário %s ausente; versão: %s", p.Backend, f, p.Version)
		}
	}
	return nil
}
func load(path string) (Config, error) {
	var c Config
	b, err := os.ReadFile(path)
	if err != nil {
		return c, err
	}
	if err = json.Unmarshal(b, &c); err != nil {
		return c, err
	}
	if c.Backend == "" {
		c.Backend = "server"
	}
	if c.CLI == "" && c.Llama != "" {
		c.CLI = filepath.Join(filepath.Dir(c.Llama), "llama-cli.exe")
	}
	if c.Data == "" {
		c.Data = "data"
	}
	if c.MainDevice == "" {
		c.MainDevice = "auto"
	}
	if c.CompilerDevice == "" {
		c.CompilerDevice = "auto"
	}
	if c.Mode == "" {
		c.Mode = "constrained"
	}
	if c.Listen == "" {
		c.Listen = "127.0.0.1:8088"
	}
	if c.GPULayers == "" {
		c.GPULayers = "auto"
	}
	if !regexp.MustCompile(`^(auto|all|[0-9]+)$`).MatchString(c.GPULayers) {
		return c, errors.New("gpu_layers deve ser auto, all ou um inteiro não negativo")
	}
	if c.Context == 0 {
		c.Context = 8192
	}
	if c.Output == 0 {
		c.Output = 1024
	}
	if c.Backend != "server" && c.Backend != "cli" {
		return c, errors.New("backend deve ser server ou cli")
	}
	if c.Backend == "cli" && c.Mode == "resident" {
		return c, errors.New("CLI executa por tarefa; selecione o modo constrained")
	}
	if c.Mode != "resident" && c.Mode != "constrained" {
		return c, errors.New("modo deve ser resident ou constrained")
	}
	if c.Context < 2048 || c.Output < 128 || c.Output > c.Context/2 {
		return c, errors.New("orçamento de contexto inválido")
	}
	if c.Backend == "cli" && (c.MainURL != "" || c.CompilerURL != "") {
		return c, errors.New("CLI não aceita URLs de servidores")
	}
	if c.Mode == "constrained" && (c.MainURL != "" || c.CompilerURL != "") {
		return c, errors.New("constrained exige processos locais")
	}
	host, _, err := net.SplitHostPort(c.Listen)
	if err != nil || host != "127.0.0.1" {
		return c, errors.New("listen deve ser 127.0.0.1:porta")
	}
	base, _ := filepath.Abs(filepath.Dir(path))
	resolve := func(value string) string {
		if value != "" && !filepath.IsAbs(value) {
			return filepath.Join(base, value)
		}
		return value
	}
	c.Llama = resolve(c.Llama)
	c.CLI = resolve(c.CLI)
	c.MainModel = resolve(c.MainModel)
	c.CompilerModel = resolve(c.CompilerModel)
	c.Data = resolve(c.Data)
	return c, nil
}
func (c Config) executable() string {
	if c.Backend == "cli" {
		return c.CLI
	}
	return c.Llama
}
func (e *Engine) validate(ctx context.Context) error {
	pctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	profile, err := probe(pctx, e.cfg.executable(), e.cfg.Backend)
	if err != nil {
		return err
	}
	if err = validateProfile(profile); err != nil {
		return err
	}
	for _, path := range []string{e.cfg.MainModel, e.cfg.CompilerModel} {
		if path == "" {
			return errors.New("selecione os dois modelos GGUF")
		}
		info, er := os.Stat(path)
		if er != nil || info.IsDir() {
			return fmt.Errorf("modelo não encontrado: %s", path)
		}
	}
	for i, id := range []string{e.cfg.MainDevice, e.cfg.CompilerDevice} {
		if id == "auto" {
			continue
		}
		if id == "none" {
			continue
		}
		if !profile.Flags["--device"] {
			return errors.New("este llama não suporta seleção explícita --device")
		}
		if id == "none" {
			continue
		}
		found := false
		for _, device := range profile.Devices {
			if device.ID == id {
				found = true
				expected := e.cfg.MainDeviceName
				if i == 1 {
					expected = e.cfg.CompilerDeviceName
				}
				if expected != "" && device.Name != expected {
					return fmt.Errorf("a GPU %s mudou de identidade; selecione novamente no configurador", id)
				}
			}
		}
		if !found {
			return fmt.Errorf("GPU %s não foi encontrada pelo llama selecionado", id)
		}
	}
	e.profile = profile
	return nil
}
func configuratorPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "Configurar-HAME.ps1")
}
