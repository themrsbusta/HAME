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
	ShortFlags []string        `json:"short_flags,omitempty"`
	Executable string          `json:"executable"`
	Backend    string          `json:"backend"`
	Version    string          `json:"version"`
	Devices    []Device        `json:"devices"`
	Flags      map[string]bool `json:"flags"`
	Resident   bool            `json:"resident_candidate"`
}

func cleanEnv() []string {
	if manualPlatform {
		return os.Environ()
	}
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
	err := waitOwned(cmd)
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
	for _, match := range regexp.MustCompile(`(?m)(?:^|[ ,\t])(-[A-Za-z][A-Za-z0-9-]*)(?:[ ,\t]|$)`).FindAllStringSubmatch(string(help), -1) {
		profile.ShortFlags = append(profile.ShortFlags, match[1])
	}
	if profile.Flags["--list-devices"] {
		data, er := capture(ctx, path, "--list-devices")
		if er != nil {
			return nil, er
		}
		profile.Devices = parseDevices(string(data))
	}
	profile.Resident = backend == "server" && profile.Flags["--slots"] && profile.Flags["--cache-ram"] && profile.Flags["--no-cache-idle-slots"] && profile.Flags["--slot-save-path"]
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
	if manualPlatform {
		if err := validateManualConfig(c); err != nil {
			return c, err
		}
	}
	if c.Backend == "" {
		c.Backend = "server"
	}
	if !manualPlatform && c.CLI == "" && c.Llama != "" {
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
	if c.Threshold == 0 {
		c.Threshold = 90
	}
	if c.Policy == "" {
		c.Policy = "hybrid"
	}
	if c.Reserve == 0 {
		c.Reserve = c.Output
	}
	if c.TemplateReserve == 0 {
		c.TemplateReserve = 128
	}
	if c.MemoryMax == 0 {
		c.MemoryMax = 1024
		if c.MemoryMax > c.Context/4 {
			c.MemoryMax = c.Context / 4
		}
	}
	if c.Threshold < 10 || c.Threshold > 99 {
		return c, errors.New("gatilho deve estar entre 10 e 99%")
	}
	if c.Policy != "automatic" && c.Policy != "on_demand" && c.Policy != "hybrid" {
		return c, errors.New("política inválida")
	}
	if c.Reserve < c.Output || c.Reserve+c.TemplateReserve >= c.Context || c.TemplateReserve < 0 {
		return c, errors.New("reserva deve cobrir a saída e deixar espaço no contexto")
	}
	if c.MemoryMax < 128 || c.MemoryMax+c.Reserve+c.TemplateReserve >= c.Context {
		return c, errors.New("orçamento da memória compilada inválido")
	}
	if _, err := parseStartupArgs(c.MainExtraArgs); err != nil {
		return c, err
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
	c.MainMMProj = resolve(c.MainMMProj)
	c.CompilerModel = resolve(c.CompilerModel)
	c.Data = resolve(c.Data)
	if c.ToolMaxCalls == 0 {
		c.ToolMaxCalls = 8
	}
	if c.ToolTimeout == 0 {
		c.ToolTimeout = 30
	}
	if err := c.validateTools(); err != nil {
		return c, err
	}
	if c.Tokenizer == "" && !manualPlatform {
		c.Tokenizer = filepath.Join(filepath.Dir(c.executable()), "llama-tokenize.exe")
	} else {
		c.Tokenizer = resolve(c.Tokenizer)
	}
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
	var profile *Profile
	var err error
	if manualPlatform {
		profile, err = manualProfile(e.cfg)
	} else {
		profile, err = probe(pctx, e.cfg.executable(), e.cfg.Backend)
	}
	if err != nil {
		return err
	}
	if err = validateProfile(profile); err != nil {
		return err
	}
	if err = e.cfg.validateTools(); err != nil {
		return err
	}
	if e.cfg.toolsEnabled() && !profile.Flags["--tools"] {
		return errors.New("llama-server não anuncia --tools")
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

	if e.cfg.MainMMProj != "" {
		if !profile.Flags["--mmproj"] {
			return errors.New("o llama selecionado não anuncia suporte a --mmproj")
		}
		info, err := os.Stat(e.cfg.MainMMProj)
		if err != nil || info.IsDir() {
			return fmt.Errorf("projetor mmproj não encontrado: %s", e.cfg.MainMMProj)
		}
		for _, arg := range strings.Fields(e.cfg.MainExtraArgs) {
			if arg == "--mmproj" || strings.HasPrefix(arg, "--mmproj=") {
				return errors.New("configure mmproj no campo próprio, sem repetir nos argumentos extras")
			}
		}
	}
	for _, choice := range []struct{ id, name *string }{{&e.cfg.MainDevice, &e.cfg.MainDeviceName}, {&e.cfg.CompilerDevice, &e.cfg.CompilerDeviceName}} {
		if *choice.id == "auto" {
			if profile.Flags["--device"] {
				d := automaticDevice(profile.Devices)
				*choice.id = d.ID
				*choice.name = d.Name
				fmt.Printf("GPU automática: %s %s\n", d.ID, d.Name)
			} else {
				fmt.Println("GPU automática: seleção delegada ao llama (sem --device)")
			}
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
		if !found && !manualPlatform {
			return fmt.Errorf("GPU %s não foi encontrada pelo llama selecionado", id)
		}
	}
	if e.cfg.Mode == "resident" && !profile.Resident {
		return errors.New("Resident incompatível: exige server com slots, slot-save-path, cache-ram e controle de cache idle; use Testar modo para validar os endpoints reais")
	}
	extra, err := validateStartupArgs(e.cfg.MainExtraArgs, profile)
	if err != nil {
		return err
	}
	e.extraMain = extra
	e.profile = profile
	return nil
}
func configuratorPath() string {
	exe, _ := os.Executable()
	return filepath.Join(filepath.Dir(exe), "Configurar-HAME.ps1")
}

func (p *Profile) hasStartupFlag(flag string) bool {
	if p.Flags[flag] {
		return true
	}
	for _, alias := range p.ShortFlags {
		if alias == flag {
			return true
		}
	}
	return false
}

// Preserve the backend device priority; shared memory size does not measure GPU performance.
func automaticDevice(devices []Device) Device {
	if len(devices) > 0 {
		return devices[0]
	}
	return Device{ID: "none", Name: "CPU"}
}
