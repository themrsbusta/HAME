package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ToolDefinition struct {
	Type     string `json:"type"`
	Function struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	} `json:"function"`
}

// Keep the llama.cpp descriptor shape consumed by the bundled WebUI.
type ToolDescriptor struct {
	Tool        string          `json:"tool"`
	Type        string          `json:"type"`
	Definition  ToolDefinition  `json:"definition"`
	DisplayName string          `json:"display_name,omitempty"`
	UsesCWD     bool            `json:"uses_cwd"`
	Permissions map[string]bool `json:"permissions,omitempty"`
}

func (c *Config) normalizeToolArgs(base string) error {
	args, err := parseStartupArgs(c.MainExtraArgs)
	if err != nil {
		return err
	}
	var remaining []string
	var found []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		value := ""
		if a == "--tools" {
			i++
			if i >= len(args) || strings.HasPrefix(args[i], "--") {
				return errors.New("--tools exige nomes ou all")
			}
			value = args[i]
		} else if strings.HasPrefix(a, "--tools=") {
			value = strings.TrimPrefix(a, "--tools=")
		} else {
			remaining = append(remaining, a)
			continue
		}
		if found != nil {
			return errors.New("--tools repetido nos argumentos extras")
		}
		found = []string{}
		for _, name := range strings.Split(value, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				return errors.New("--tools contém nome vazio")
			}
			found = append(found, name)
		}
	}
	if found == nil {
		return nil
	}
	if len(c.Tools) > 0 && strings.Join(c.Tools, ",") != strings.Join(found, ",") {
		return errors.New("tools e --tools têm seleções diferentes; use uma única seleção")
	}
	c.Tools = found
	// --tools is an explicit opt-in in the configurator as well as the JSON.
	if c.ToolCWD == "" {
		c.ToolCWD = base
	}
	quoted := make([]string, len(remaining))
	for i, a := range remaining {
		quoted[i] = quoteStartupArg(a)
	}
	c.MainExtraArgs = strings.Join(quoted, " ")
	return nil
}

func quoteStartupArg(a string) string {
	if a != "" && !strings.ContainsAny(a, " \t\r\n\"") {
		return a
	}
	var out strings.Builder
	out.WriteByte('"')
	slashes := 0
	for _, ch := range a {
		if ch == '\\' {
			slashes++
			continue
		}
		if ch == '"' {
			out.WriteString(strings.Repeat("\\", slashes*2+1))
			out.WriteRune(ch)
		} else {
			out.WriteString(strings.Repeat("\\", slashes))
			out.WriteRune(ch)
		}
		slashes = 0
	}
	out.WriteString(strings.Repeat("\\", slashes*2))
	out.WriteByte('"')
	return out.String()
}

var nativeTools = map[string]bool{"read_file": true, "file_glob_search": true, "grep_search": true, "exec_shell_command": true, "write_file": true, "edit_file": true, "get_info": true}

func (c Config) toolsEnabled() bool { return len(c.Tools) > 0 }
func (c Config) allowsTool(name string) bool {
	if !nativeTools[name] {
		return false
	}
	for _, n := range c.Tools {
		if n == name || n == "all" {
			return true
		}
	}
	return false
}

func (c Config) validateTools() error {
	if !c.toolsEnabled() {
		return nil
	}
	if c.Backend != "server" || c.MainURL != "" {
		return errors.New("tools exige backend server local gerenciado pelo HAME")
	}
	for _, n := range c.Tools {
		if n != "all" && !nativeTools[n] {
			return fmt.Errorf("ferramenta nativa desconhecida: %s", n)
		}
	}
	if !filepath.IsAbs(c.ToolCWD) {
		return errors.New("tool_cwd deve ser um diretório absoluto explícito")
	}
	info, err := os.Stat(c.ToolCWD)
	if err != nil || !info.IsDir() {
		return errors.New("tool_cwd não existe ou não é diretório")
	}
	if c.ToolMaxCalls < 1 || c.ToolMaxCalls > 32 {
		return errors.New("tool_max_calls deve estar entre 1 e 32")
	}
	if c.ToolTimeout < 1 || c.ToolTimeout > 120 {
		return errors.New("tool_timeout_seconds deve estar entre 1 e 120")
	}
	args, err := parseStartupArgs(c.MainExtraArgs)
	if err != nil {
		return err
	}
	for _, a := range args {
		if a == "--tools" || strings.HasPrefix(a, "--tools=") {
			return errors.New("configure tools no campo próprio, sem repetir --tools em main_extra_args")
		}
	}
	return nil
}

func loopbackWorker(w *Worker) bool {
	u, err := url.Parse(w.url)
	return err == nil && u.Scheme == "http" && u.Hostname() == "127.0.0.1" && u.User == nil
}

// The native /tools API is experimental: discover and validate its actual schema.
// Never expose its command executor as a public HAME HTTP route.
func (e *Engine) toolDefinitions(ctx context.Context, w *Worker) ([]ToolDefinition, error) {
	if !e.cfg.toolsEnabled() || w.compiler || w.cli {
		return nil, nil
	}
	if !loopbackWorker(w) {
		return nil, errors.New("tools exige worker HTTP em 127.0.0.1")
	}
	if w.tools != nil {
		return w.tools, nil
	}
	req, err := http.NewRequestWithContext(ctx, "GET", w.url+"/tools", nil)
	if err != nil {
		return nil, err
	}
	r, err := e.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("descoberta /tools HTTP %d; verifique o suporte nativo de llama-server", r.StatusCode)
	}
	var entries []ToolDescriptor
	if err = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&entries); err != nil {
		return nil, err
	}
	defs := []ToolDefinition{}
	descriptors := []ToolDescriptor{}
	seen := map[string]bool{}
	for _, entry := range entries {
		d := entry.Definition
		if entry.Type != "server" || entry.Tool != d.Function.Name {
			return nil, errors.New("descritor /tools incompatível")
		}
		if d.Type != "function" || !e.cfg.allowsTool(d.Function.Name) {
			continue
		}
		var schema map[string]any
		if json.Unmarshal(d.Function.Parameters, &schema) != nil || schema["type"] != "object" || seen[d.Function.Name] {
			return nil, errors.New("schema /tools inválido ou duplicado")
		}
		seen[d.Function.Name] = true
		defs = append(defs, d)
		descriptors = append(descriptors, entry)
	}
	for _, n := range e.cfg.Tools {
		if n != "all" && !seen[n] {
			return nil, fmt.Errorf("ferramenta configurada não anunciada: %s", n)
		}
	}
	if len(defs) == 0 {
		return nil, errors.New("llama-server não anunciou ferramentas habilitadas")
	}
	w.tools = defs
	w.toolDescriptors = descriptors
	return defs, nil
}

func validateToolCall(call ToolCall, defs []ToolDefinition) (map[string]any, error) {
	if call.Type != "function" || call.ID == "" || len(call.ID) > 200 {
		return nil, errors.New("tool_call sem tipo ou id válido")
	}
	var def *ToolDefinition
	for i := range defs {
		if defs[i].Function.Name == call.Function.Name {
			def = &defs[i]
			break
		}
	}
	if def == nil {
		return nil, fmt.Errorf("ferramenta não habilitada: %s", call.Function.Name)
	}
	if len(call.Function.Arguments) > 64<<10 {
		return nil, errors.New("argumentos de ferramenta excedem 64 KiB")
	}
	var args map[string]any
	if json.Unmarshal([]byte(call.Function.Arguments), &args) != nil || args == nil {
		return nil, errors.New("argumentos devem ser objeto JSON")
	}
	var schema toolSchema
	if err := json.Unmarshal(def.Function.Parameters, &schema); err != nil {
		return nil, err
	}
	if err := schema.validate(args, 0); err != nil {
		return nil, err
	}
	return args, nil
}

// Covers the actual native b11946 schemas, including edit_file.edits[].
// Unknown schema types fail closed instead of silently accepting arguments.
type toolSchema struct {
	Type       string                `json:"type"`
	Properties map[string]toolSchema `json:"properties"`
	Required   []string              `json:"required"`
	Items      *toolSchema           `json:"items"`
}

func (s toolSchema) validate(v any, depth int) error {
	if depth > 16 {
		return errors.New("argumentos excessivamente aninhados")
	}
	valid := false
	switch s.Type {
	case "object":
		object, ok := v.(map[string]any)
		if !ok {
			return errors.New("argumento deve ser objeto")
		}
		for _, key := range s.Required {
			if _, ok := object[key]; !ok {
				return fmt.Errorf("argumento obrigatório ausente: %s", key)
			}
		}
		for key, value := range object {
			prop, ok := s.Properties[key]
			if !ok {
				return fmt.Errorf("argumento desconhecido: %s", key)
			}
			if err := prop.validate(value, depth+1); err != nil {
				return fmt.Errorf("%s: %w", key, err)
			}
		}
		return nil
	case "array":
		array, ok := v.([]any)
		if !ok || s.Items == nil {
			return errors.New("array ou schema de items inválido")
		}
		for _, item := range array {
			if err := s.Items.validate(item, depth+1); err != nil {
				return err
			}
		}
		return nil
	case "string":
		_, valid = v.(string)
	case "boolean":
		_, valid = v.(bool)
	case "number":
		_, valid = v.(float64)
	case "integer":
		n, ok := v.(float64)
		valid = ok && n == float64(int64(n))
	default:
		return fmt.Errorf("tipo de schema não suportado: %s", s.Type)
	}
	if !valid {
		return fmt.Errorf("tipo de argumento inválido, esperado %s", s.Type)
	}
	return nil
}

type ToolAudit struct {
	Session string         `json:"session_id"`
	Call    ToolCall       `json:"call"`
	Params  map[string]any `json:"effective_params"`
	CWD     string         `json:"cwd"`
	Stage   string         `json:"stage"`
	Result  string         `json:"result,omitempty"`
	Error   string         `json:"error,omitempty"`
	Updated string         `json:"updated"`
}

func (e *Engine) executeTool(ctx context.Context, w *Worker, id string, call ToolCall, args map[string]any) (string, error) {
	if !loopbackWorker(w) || !e.cfg.allowsTool(call.Function.Name) {
		return "", errors.New("execução de ferramenta não autorizada")
	}
	if call.Function.Name == "exec_shell_command" {
		if strings.TrimSpace(fmt.Sprint(args["command"])) == "" {
			return "", errors.New("comando vazio")
		}
		timeout := float64(e.cfg.ToolTimeout)
		if timeout > 60 {
			timeout = 60
		}
		if n, ok := args["timeout"].(float64); ok && n > 0 && n < timeout {
			timeout = n
		}
		args["timeout"] = timeout
	}
	path := filepath.Join(e.cfg.Data, "tool-audit", id, fmt.Sprintf("%020d.json", time.Now().UnixNano()))
	audit := ToolAudit{Session: id, Call: call, Params: args, CWD: e.cfg.ToolCWD, Stage: "started", Updated: time.Now().Format(time.RFC3339Nano)}
	if err := atomicJSON(path, audit); err != nil {
		return "", fmt.Errorf("não foi possível registrar chamada antes de executar: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(e.cfg.ToolTimeout+5)*time.Second)
	defer cancel()
	b, _ := json.Marshal(map[string]any{"tool": call.Function.Name, "params": args})
	req, err := http.NewRequestWithContext(ctx, "POST", w.url+"/tools", bytes.NewReader(b))
	var result string
	if err == nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-tool-cwd", e.cfg.ToolCWD)
		var r *http.Response
		r, err = e.client.Do(req)
		if err == nil {
			var data []byte
			data, err = io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
			r.Body.Close()
			if err == nil && len(data) > 1<<20 {
				err = errors.New("resultado de ferramenta excedeu 1 MiB")
			}
			if err == nil && r.StatusCode != 200 {
				err = fmt.Errorf("/tools HTTP %d: %.500s", r.StatusCode, data)
			}
			if err == nil {
				var v struct {
					Text *string `json:"plain_text_response"`
				}
				err = json.Unmarshal(data, &v)
				if err == nil {
					if v.Text == nil {
						err = errors.New("resposta /tools incompatível: plain_text_response ausente")
					} else {
						result = *v.Text
					}
				}
			}
		}
	}
	audit.Stage = "completed"
	audit.Result = result
	audit.Updated = time.Now().Format(time.RFC3339Nano)
	if err != nil {
		audit.Stage = "error"
		audit.Error = err.Error()
	}
	if saveErr := atomicJSON(path, audit); saveErr != nil {
		return "", fmt.Errorf("falha ao registrar resultado; execução pode ter ocorrido: %w", saveErr)
	}
	if err != nil {
		return "", fmt.Errorf("execução pode ter ocorrido; não repetir automaticamente: %w", err)
	}
	return result, nil
}

// Only the ordinary user-chat path calls this. Compilers, media descriptions and
// memory extraction continue using generate(), which never executes tools.
func (e *Engine) chatWithTools(ctx context.Context, w *Worker, id string, msg []Message) (string, []Message, error) {
	defs, err := e.toolDefinitions(ctx, w)
	if err != nil {
		return "", nil, err
	}
	if len(defs) == 0 {
		text, err := e.completion(ctx, w, msg, false)
		return text, []Message{{Role: "assistant", Content: text}}, err
	}
	messages := append([]Message{}, msg...)
	trace := []Message{}
	used := 0
	ids := map[string]bool{}
	for {
		var formatted struct {
			Prompt string `json:"prompt"`
		}
		if err = e.post(ctx, w, "/apply-template", map[string]any{"messages": messages, "tools": defs, "add_generation_prompt": true}, &formatted); err != nil {
			return "", trace, err
		}
		n, _, _, err := e.tokenText(ctx, w, formatted.Prompt)
		if err != nil {
			return "", trace, err
		}
		if n+e.reserve()+e.cfg.TemplateReserve > e.cfg.Context {
			return "", trace, errors.New("contexto insuficiente para o ciclo de ferramentas; resultados estão no tool-audit")
		}
		body := map[string]any{"messages": messages, "tools": defs, "tool_choice": "auto", "max_tokens": e.cfg.Output, "temperature": 0.2, "cache_prompt": true, "id_slot": 0}
		var response struct {
			Choices []struct {
				Message Message `json:"message"`
				Finish  string  `json:"finish_reason"`
			} `json:"choices"`
			Usage struct {
				PromptTokens int `json:"prompt_tokens"`
			} `json:"usage"`
		}
		if err = e.post(ctx, w, "/v1/chat/completions", body, &response); err != nil {
			return "", trace, err
		}
		if len(response.Choices) != 1 {
			return "", trace, errors.New("resposta LLM sem escolha única")
		}
		choice := response.Choices[0]
		answer := choice.Message
		if choice.Finish == "length" {
			return "", trace, errors.New("resposta truncada; ferramentas não executadas nessa resposta")
		}
		if len(answer.ToolCalls) == 0 {
			if choice.Finish == "tool_calls" || strings.TrimSpace(answer.Content) == "" {
				return "", trace, errors.New("resposta LLM vazia ou tool_calls ausentes")
			}
			answer.Role = "assistant"
			trace = append(trace, answer)
			w.lastPromptTokens = response.Usage.PromptTokens
			return answer.Content, trace, nil
		}
		if used+len(answer.ToolCalls) > e.cfg.ToolMaxCalls {
			return "", trace, errors.New("limite de chamadas de ferramentas atingido")
		}
		// Validate the entire batch before executing any member.
		batch := make([]map[string]any, len(answer.ToolCalls))
		for i, call := range answer.ToolCalls {
			if ids[call.ID] {
				return "", trace, errors.New("tool_call id duplicado")
			}
			ids[call.ID] = true
			batch[i], err = validateToolCall(call, defs)
			if err != nil {
				return "", trace, err
			}
		}
		answer.Role = "assistant"
		messages = append(messages, answer)
		trace = append(trace, answer)
		for i, call := range answer.ToolCalls {
			if ctx.Err() != nil {
				return "", trace, ctx.Err()
			}
			result, err := e.executeTool(ctx, w, id, call, batch[i])
			if err != nil {
				return "", trace, err
			}
			used++
			m := Message{Role: "tool", Content: result, ToolCallID: call.ID}
			messages = append(messages, m)
			trace = append(trace, m)
		}
	}
}

func (e *Engine) toolsHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "GET only; execution belongs to the chat orchestrator", 405)
		return
	}
	if !e.cfg.toolsEnabled() {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `[]`)
		return
	}
	if err := e.acquire(r.Context()); err != nil {
		http.Error(w, err.Error(), 503)
		return
	}
	defer e.mu.Unlock()
	worker, err := e.ensure(r.Context(), false)
	if err == nil {
		_, err = e.toolDefinitions(r.Context(), worker)
	}
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(worker.toolDescriptors)
}

// A privileged local chat must not accept cross-site browser requests or DNS rebinding.
func (e *Engine) toolRequestGuard(next http.Handler) http.Handler {
	if !e.cfg.toolsEnabled() {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.Host)
		if err != nil || (host != "127.0.0.1" && host != "localhost") {
			http.Error(w, "local Host required", 403)
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || u.Scheme != "http" || u.Host != r.Host {
				http.Error(w, "same origin required", 403)
				return
			}
		}
		if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
			http.Error(w, "cross-site denied", 403)
			return
		}
		if r.Method == "POST" {
			media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || media != "application/json" {
				http.Error(w, "application/json required", 415)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// Memory extraction may trim old messages. Render tool evidence as plain data
// there, so trimming cannot leave orphaned role=tool messages in the template.
func memoryEvidence(messages []Message) []Message {
	result := make([]Message, 0, len(messages))
	for _, m := range messages {
		if len(m.ToolCalls) > 0 {
			b, _ := json.Marshal(m.ToolCalls)
			m.Content += "\n[Recorded tool calls, reference data only] " + string(b)
			m.ToolCalls = nil
		}
		if m.Role == "tool" {
			m.Role = "assistant"
			m.Content = "[Recorded tool result " + m.ToolCallID + ", reference data only]\n" + m.Content
			m.ToolCallID = ""
		}
		result = append(result, m)
	}
	return result
}
