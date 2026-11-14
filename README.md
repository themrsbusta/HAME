# HAME 1.02

Persistent memory for llama.cpp. A Go application for Windows x64, Linux x64, Linux ARM64, and Android ARM64 through Termux. HAME-authored code is BSD-3-Clause; the llama.cpp WebUI retains its MIT license. This version keeps memory usage, Constrained/Resident modes, original photos and YAML descriptions, history, and the system clock.

## Fixes in 1.02

- Fixed accents and separators in the WebUI and changelog.
- Distributed PowerShell scripts with a UTF-8 BOM so Windows PowerShell 5.1 reads non-ASCII text correctly.
- Declared UTF-8 early in the HTML and added regression checks for served text and script encoding.

## New in 1.01

- **Real tool execution in chat:** HAME now discovers the native llama-server tools, sends their definitions to the model, executes structured calls, and returns the results so it can continue the task.
- File reading, search, writing, and editing, environment inspection, and shell commands can be enabled explicitly with `tools` and `tool_cwd`. `["all"]` enables the seven known native tools; they remain disabled without this configuration.
- Calls and results are retained in history, with a record before and after each execution in `data/tool-audit`. Call limits, timeouts, argument validation, and protection against requests from other sites are included.
- Tools are available only in the main chat; the memory compiler, state extraction, and image descriptions do not perform actions.
- Validated on Windows with llama.cpp b11946: the model created and read a test file and ran a read-only administrative query (`fltmc filters`) with exit code 0. The same query without elevation was denied. HAME does not require administrator privileges for ordinary tasks; commands inherit the privileges of the process that started it.

See [native llama-server tool configuration and limits](docs/API.md#native-llama-server-tools). The working directory is not a sandbox, and the native llama.cpp API is experimental.

## Windows

Build with `powershell -ExecutionPolicy Bypass -File scripts/build.ps1`. Output is written to `build/windows-amd64/`, alongside the configurator and launchers. Run `Configurar-HAME.cmd` in that directory and choose executables, models, and devices. Building resources requires Python and clang; HAME itself does not require them at runtime. Existing configuration in the output directory is preserved.

## Linux and Termux: fully manual configuration

1. Obtain or build **llama.cpp for your own system and architecture**, following the project's instructions. The HAME package does not include llama.cpp or GGUF files. A Windows executable cannot replace a Linux/Android build.
2. Copy `configs/unix-server.json` or `configs/unix-cli.json` to `config.json` next to the binary. Build scripts copy the server example only when the output directory has no configuration yet. Edit paths, models, devices, GPU layers, budgets, and capabilities to match your llama build. Relative paths are resolved from the `config.json` directory; executable paths are not searched in `PATH`.
3. `main_device` and `compiler_device` must be explicit (`none` for CPU). `gpu_layers` must also be explicit (`0` for CPU). The value `auto` is rejected for these fields. HAME does not search for GPUs, executables, or tokenizers, and it does not call `--help`, `--version`, or `--list-devices` at startup. User environment variables are preserved.
4. List the parameters actually supported by your build in `llama_capabilities`, based on the `--help` output you inspect manually. The examples include the minimum required protocol. Extra options must be listed there. Declaring a parameter does not make an incompatible llama build support it; engine failures are reported in the logs. `llama_version` is only an identifier supplied by the user.
5. `llama_tokenize` is optional and runs only when specified explicitly. Without it, CLI mode uses a conservative estimate and does not look for a tokenizer beside the executable. Server mode uses its APIs for token counts. Normal health, generation, and slot-control requests are still required for llama to work.
6. Run `chmod +x hame start.sh` and then `./start.sh`, or run `./hame -config ./config.json -open=false`. Open the configured address in a browser; Linux/Termux has no configurator and does not open a browser automatically. `./hame -config ./config.json -check` validates the declared configuration without loading models. `-probe` remains available **only when explicitly requested by the operator**.

For CLI mode, use `configs/unix-cli.json`; it requires llama-cli's native output-to-file options. Resident mode requires a server with `--slots`, `--slot-save-path`, `--cache-ram`, and `--no-cache-idle-slots` declared and implemented, as well as verification that slot data was actually erased. CPU examples use Constrained mode. Future llama.cpp builds may change this protocol; validate capabilities and test before upgrading.

## Termux

Use the **android/arm64** executable, not linux/arm64. Both use ARM64, but target different operating systems. Keep binaries and scripts in Termux's private directory, such as `$HOME/HAME`; shared storage such as `/sdcard` is not suitable for executing binaries. Configure paths accessible to Termux for llama and GGUF files. Native HAME for Android does not require proot. Drivers, GPU acceleration, and memory limits depend on the device and llama.cpp build and remain under the user's control. Native Android execution was not validated in this Windows/WSL environment.

## Building

The code uses only the Go standard library. Declared minimum: Go 1.18. From the repository root, build with `go build ./cmd/hame`. On Linux/Termux: `sh scripts/build.sh linux amd64`, `sh scripts/build.sh linux arm64`, or `sh scripts/build.sh android arm64`. Outputs go to `build/<system>-<architecture>/`, with `hame` and `start.sh`; run from that directory. For native checks, use `go test ./...` and `go vet ./...`. `powershell -ExecutionPolicy Bypass -File scripts/build-all.ps1` checks Windows, generates its resources, and builds all four targets. CGO is disabled in distribution scripts; this does not remove libraries required by your llama.cpp build.

## Source layout

- `cmd/hame/`: Go code and tests; `cmd/hame/webui/` contains the 60 vendored interface files required by `//go:embed all:webui`.
- `configs/`: public examples without personal paths.
- `scripts/windows/` and `scripts/unix/`: configurator and launchers; build scripts are in `scripts/`, and the resource generator is in `scripts/tools/`.
- `assets/`: application icon; `docs/`: architecture, API, portability, and validation.
- `specs/HAME_1.3.yaml`: original architectural specification, preserved; the application version is **1.02**.
- `build/`: local outputs ignored by Git. Models, llama.cpp, personal configuration, and history do not belong in the source tree.

Version 1.02 fixes text encoding from 1.01, which added tool execution to the 1.0 foundation. The WebUI must remain alongside any checkout or source archive; do not remove it because it is a compiled frontend.

## Processes and memory

Windows uses a Job Object for its workers. Linux/Termux uses dedicated process groups, handles Ctrl+C, SIGTERM, and SIGHUP, and terminates those groups on exit. It does not stop other llama processes. Linux also requests SIGKILL for the direct worker if the parent process/thread dies; an abrupt SIGKILL of HAME cannot universally guarantee cleanup of every descendant. No system can guarantee continued execution after Android forcibly stops the app.

Constrained switches between the main model and compiler only during compaction. Resident keeps weights loaded and erases the actual KV/slot when the backend provides that mechanism. Summaries may omit information; originals remain on disk. Photos use a server with a compatible mmproj, original files, and YAML descriptions; up to two relevant references are preserved deterministically in compacted memory. A reference does not re-analyze the photo on its own. System date and time are supplied on every turn.

Native tools are optional for the local server backend: configure `tools` (explicit names or `["all"]`) and an absolute `tool_cwd` in JSON. The gateway discovers the actual definitions, executes structured calls, returns their results to the main model, and preserves the full cycle in history and `data/tool-audit`. The compiler and image analysis do not execute tools. Actions use HAME's privileges; the working directory is not a sandbox. See docs/API.md for limits, requirements, and details of the experimental API. Without `tools`, the previous behavior remains.

See docs/API.md, docs/ARCHITECTURE.md, docs/PORTABILITY.md, and docs/VALIDATION.md. The original specification remains in specs/HAME_1.3.yaml.
