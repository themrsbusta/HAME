package main

import (
	"crypto/sha256"
	"fmt"
	"os"
)

func fileRevision(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", sha256.Sum256(b))
}

// Report the configuration loaded by this service, independently of saved edits
// that have not been applied by restarting it.
func (e *Engine) configurationState() map[string]any {
	changed := false
	if e.configPath != "" {
		changed = fileRevision(e.configPath) != e.configRevision
	}
	tools := append([]string{}, e.cfg.Tools...)
	return map[string]any{"model_file": modelFileName(e.cfg.MainModel), "model_path": e.cfg.MainModel, "config_path": e.configPath, "config_revision": e.configRevision, "restart_required": changed, "tools_enabled": e.cfg.toolsEnabled(), "tools": tools, "tool_cwd": e.cfg.ToolCWD, "listen": e.cfg.Listen, "tool_execution": "hame", "continuous_history": e.continuousHistory(), "real_context_tokens": e.cfg.Context, "webui": e.cfg.WebUI, "retrieval_tokens": e.cfg.RetrievalTokens}
}
