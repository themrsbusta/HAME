# Architecture — HAME 1.01

HAME is a Go gateway between clients compatible with chat completions and external llama.cpp processes. The vendored WebUI is embedded in the binary by `cmd/hame/main.go` using `//go:embed all:webui`. The gateway serves those files and adapts each conversation to the session's active context.

The main model responds to the user and produces the conversation's semantic state. The compiler summarizes persisted sources when the context budget requires compaction or when full recovery is explicitly requested. Originals, history, and metadata remain in the configured data directory; summaries may omit information.

In Constrained mode, the main model stays active between turns and the models switch during compaction. In Resident mode, weights remain loaded and the gateway requires compatible, verifiable slot/KV mechanisms. Configuration, memory, media, clock, API, and process control are separated across files in `cmd/hame/`.

Windows retains the configurator and explicit probing of engine capabilities. Linux and Android/Termux use manual configuration, with paths, devices, GPU layers, and capabilities declared by the user. Engines and models are not distributed in this repository.

The original specification in `specs/HAME_1.3.yaml` describes the intended architecture; its numbering is not the application version and does not prove that every objective has been implemented. Native tools now have an optional cycle in the main chat: discover actual definitions through `/tools`, validate them, execute structured calls, return `role=tool` results, and run inference again, with limits and durable logging. The compiler, state extraction, and image descriptions do not execute tools. Photo references do not trigger a new visual analysis, and summarized memory may lose details. See `API.md`, `PORTABILITY.md`, and `VALIDATION.md`.
