# Portability 1.01

| Target | GOOS | GOARCH | Configuration |
|---|---|---|---|
| Windows x64 | windows | amd64 | GUI/JSON, capabilities probed by llama |
| Linux x64 | linux | amd64 | Explicit JSON, no automatic probing |
| Linux ARM64 | linux | arm64 | Explicit JSON, no automatic probing |
| Termux ARM64 | android | arm64 | Explicit JSON, no automatic probing |

This release does not support 32-bit targets. Linux ARM64 and Android ARM64 use different binaries. HAME is built with CGO=0; the llama.cpp engine must be compatible with the target's ABI, CPU, libraries, devices, and model. The `llama_capabilities` fields reflect the native contract required by the adapter; they are not a translation of flags or a replacement for `--help`.

The server protocol requires `/health`, `/props`, `/apply-template`, `/tokenize`, and `/v1/chat/completions`. Resident mode also requires `/slots` and verifiable slot erasure. The CLI protocol depends on `--output-file` and the parameters declared in the example. If the operator switches builds, the capability list must be updated manually. Declared capabilities do not certify compatibility; the engine still has to accept them in practice.

The Windows copy retains the configurator. Shell scripts do not install dependencies, choose a GPU, search for models, or modify JSON. The examples target CPU and contain sample paths that must be edited. Missing budget options still use the core's compatibility defaults; use the fully populated examples for complete control.
