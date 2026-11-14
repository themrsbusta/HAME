# API — HAME 1.01

POST `/v1/chat/completions` preserves the previous format and adds optional `compile` and `retrieve` fields. `session_id` or `X-Conversation-Id` identifies the session. Only the last user message from the WebUI is used; HAME owns the active context.

```json
{"session_id":"example","messages":[{"role":"user","content":"What was the code?"}],"retrieve":true}
```

`compile=true` requests manual compaction before the response. `retrieve=true` requests full recovery under the On-demand/Hybrid policies. The existing `/recuperar` message prefix is equivalent to `retrieve=true`.

POST `/hame/compact`: `{"session_id":"example","recover":false}`. If blank, the active session is used. `recover=true` rereads the original history. The operation is synchronous and does not delete the conversation.

GET `/hame/runtime`: version, mode, policy, active session, telemetry, and operation events. GET `/hame/status?session_id=...` preserves the previous fields and adds `mode/context_usage`. Counts include `prompt_tokens`, `reserved_response_tokens`, `template_reserve_tokens`, `budget_tokens`, `budget_percent`, `exact_prompt`, and `method`. `exact_prompt=false` identifies the CLI template reserve or the fallback used without a tokenizer.

GET `/hame/history` preserves original messages. `/health` does not guarantee that models are ready. `/props` and `/v1/models` continue to report the main model's actual identity. SSE still returns the complete text at the end.

Local clients: `HAME.exe -memory-status`; `HAME.exe -compact -session example`; `HAME.exe -compact -recover`. They check the service version so they do not silently operate on another installation.

Generation/compaction errors return an HTTP error and preserve original files. When the On-demand policy runs out of space, manual action is required before inference.

## Images and YAML

With a server configured using `main_mmproj`, the last user message may contain `content: [{"type":"text","text":"Remember this photo"},{"type":"image_url","image_url":{"url":"data:image/png;base64,..."}}]`. The main model generates a brief description before the text conversation; the photo and YAML are stored in `context/<session>/media`. Up to four images per message, 10 MiB/40 MP per image, and 32 MiB total JSON body. PNG/JPEG/GIF are supported; external URLs and other types are rejected. CLI does not provide this flow.

`GET /hame/media?session_id=<session>&file=<sha256>.png` retrieves the original bytes; `.yaml` retrieves the metadata. Only valid hash names and sessions are accepted. This is a local route, like the chat API; there is no authentication between conversations.

The compiler includes `.yaml`/`.yml` in its incremental sources. The catalog stays outside the KV cache and summary. Up to two relevant references are deterministically re-injected into compacted memory. Retrieving a description/link does not perform a new visual analysis of the original.

GET `/hame/get_datetime`: system clock, ISO 8601/time zone/offset/Unix time; no cache is used. The main model automatically receives an updated reading on each turn. This is not a generic tool-calling cycle.

## Native llama-server tools

Opt-in in `config.json` (disabled when `tools` is absent):

```json
{
  "tools": ["exec_shell_command", "read_file"],
  "tool_cwd": "C:\\HAME\\workspace",
  "tool_max_calls": 8,
  "tool_timeout_seconds": 30
}
```

`["all"]` enables the known tools advertised by the server: `read_file`, `file_glob_search`, `grep_search`, `exec_shell_command`, `write_file`, `edit_file`, and `get_info`. This requires a local server backend managed by HAME and actual `--tools` support. HAME adds this argument only to the main model; do not repeat it in `main_extra_args`. `tool_cwd` is required and must be absolute; it defines the working directory, **not** a sandbox. Commands inherit process privileges; enable this only on a trusted local workstation.

`GET /tools` discovers and lists the worker's actual tool definitions. `POST /tools` on HAME is rejected: the native executor is not exposed as an HTTP proxy. Chat discovers the definitions, sends `tools`, validates IDs, names, and structured arguments, executes each call, and returns `role=tool`/`tool_call_id` before continuing generation. Free text is never interpreted as a command. Image descriptions, state extraction, and the compiler do not run tools. Chat clients continue to receive the final textual response, including over SSE; history contains the complete cycle.

Per-turn limit: 1–32 calls (default 8). Timeout: 1–120 s (default 30); the native shell timeout is limited by this setting. The HTTP deadline for other tools does not guarantee that server-side effects are cancelled. Transport failure does not trigger automatic retry: an operation may have completed. Each call is logged before and after execution in `data_dir/tool-audit/<session>/*.json`; if a failure occurs after execution, check that record before retrying the task. A command's nonzero exit code remains a tool result and is not treated as presumed success.

The cycle counts tool definitions against the context budget and stops before another inference if there is not enough room. Results from completed calls remain in the audit log even if the turn fails. The service stays on loopback, rejects external Host headers, cross-site Origins and requests, and POST requests without `application/json` when tools are enabled. There is no authentication against other local programs: any process that can call chat can request actions with HAME's privileges. The llama.cpp internal `/tools` API is experimental and must be revalidated after changing builds; it was validated locally with b11946.
