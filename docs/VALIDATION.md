# Validation — HAME 1.02

## 1.02 checks — 2026-10-29

- Repaired text that had been stored with corrupted encoding in the WebUI and CHANGELOG. The vendored JavaScript was inspected and its valid Unicode tables were preserved.
- Distributed PowerShell scripts with a UTF-8 BOM; the launcher uses Windows PowerShell 5.1, which does not automatically recognize UTF-8 without a BOM.
- Regression checks cover accents, HTTP charset, UTF-8 declaration before scripts, and BOMs in all PowerShell scripts. Go tests and vet passed; Mica built four targets through HAME.
- Ran the configurator self-test in Windows PowerShell 5.1 and rendered both tabs; visually checked accents in memory, configuration, execution, session, and recovery.

## 1.01 checks — 2026-10-29

- Base: a fresh checkout of the public repository at `e1bcd4d`, with the tool cycle implemented in the main chat. Complete WebUI and licenses were preserved.
- Windows x64, Go 1.27.3: `go test ./...`, `go vet ./...`, and the build passed. New cases cover the actual `/tools` schema from build b11946, `role=tool` results, persisted history, invalid/nested arguments, disabled names, duplicate IDs, batched calls, truncation, per-turn limits, errors without automatic retry, and blocking requests from other sites.
- Real test with Gemma E4B Q4_K_M and llama.cpp b11946 Vulkan: the model requested `exec_shell_command`, created a marked file, and used `read_file` to verify its contents. The actual file and both tool records were checked outside the model.
- Elevation: the model ran `fltmc filters`, an administrative read operation, and received a real listing with `[exit code: 0]`. The same operation in a non-elevated session was denied. Running as administrator is a local setting; public packages do not force elevation.
- Windows build delegated to Mica by HAME and checked by Gepeto: the native tool returned exit code 0; `-version`, FileVersion, and ProductVersion identify `1.01`. Icon resources were included.
- Cross-compilation for Linux amd64, Linux arm64, and Android arm64 completed. These binaries were not run locally on their target systems in this update; inference and tool validation were done on Windows. GitHub automation tests Windows and Ubuntu on every push.
- The llama.cpp `/tools` API is internal and experimental. The fixture in `cmd/hame/testdata/llama-b11946-tools.json` contains only the seven public tool definitions, with no history or personal configuration.

## HAME 1.0 history (preserved report)

## Checks for this update — 2026-10-28

- Functional base: commit `b6531bb3283f3df9988095338108808460ecf6e4` from 0.92 RC; reorganized and labeled 1.0, with no new engine features.
- WebUI: 60 files recovered from the local 0.92 RC executable's `embed.FS`. Contents were checked against the embedded hashes before updating the version in the HTML and service worker. Assets, including those referenced by the index, are in `cmd/hame/webui/`.
- Windows x64, Go 1.27.3: `go test ./...`, `go vet ./...`, and build passed. `scripts/build-all.ps1` built Windows amd64, Linux amd64, Linux arm64, and Android arm64 with CGO disabled. Windows resources: FileVersion and ProductVersion `1.0`; `HAME.exe -version` returns `HAME 1.0`.
- Configurator: device-selection/configuration UI self-test passed. No new real llama/GPU probe or inference was performed in this update.
- Clean local checkout: a new clone of the reorganized commit, without `.syso` resources or build outputs; `go test ./...`, `go vet ./...`, `go build ./cmd/hame`, and `hame.exe -version` passed. The 60 local HTML/manifest references were checked; both licenses retain their original blobs. The only functional change in Go source is the version constant.
- The supplied original specification was preserved byte for byte in `specs/HAME_1.3.yaml`; SHA-256 `d2b67e83eee312e1411d6a08782562a101cd7bab711d3c3301555ce3d2087422`.
- No new Release packages were generated or published. The source machine's 1.0 ZIP was not used: this update was rebuilt from the 0.92 RC base, as requested by the author.

Linux ARM64 and Android/Termux were not run on target hardware. The WSL2 results below are inherited from the 0.92 RC documentation and were not re-run locally. Generic tools remain absent; photo references do not re-analyze originals, and summaries may omit information.

## 0.92 RC validation history (preserved report)

- Windows x64: Go tests and go vet passed; the binary runs `-version` and has FileVersion/ProductVersion 0.92 RC. The configurator passed with real asynchronous llama/GPU detection and validation using an isolated configuration.
- Linux x64: the suite was cross-compiled for linux/amd64 and ran successfully on Ubuntu-24.04 WSL2. It covers memory/compaction, history, photos/YAML, clock, API, and invalid inputs. Process-group tests confirmed that owned descendants were terminated while an independent process survived. Manual validation confirmed that the fake executable was not run for autodetection. Automatic migration of old configuration is Windows-only and was correctly skipped in the Unix test.
- Linux x64: confirmed the static ELF and ran `-version` on Ubuntu. Missing configuration reports an error; no configurator opens. No inference with a real Linux llama was performed because that engine is not installed on the Ubuntu used for tests.
- Linux ARM64: cross-compilation completed; the static AArch64 ELF was checked. Not run on ARM64 hardware.
- Termux ARM64: android/arm64 cross-compilation completed; the AArch64 PIE with `/system/bin/linker64` interpreter was checked. Not run on an Android/Termux device. Do not confuse this target with linux/arm64.

Installed compiler: Go 1.18 windows/amd64, CGO disabled. Real inference and photo-recovery tests from 0.91 are documented in that version; they are not presented as new inference validation for every 0.92 target.

`HAME_1.3.yaml` retains the bytes from the previous version. Packages do not include llama.cpp, models, or personal data. Source is shared across the four targets, with target-specific process-control files and manual Unix behavior.

The Linux x64 package was extracted and ran successfully on Ubuntu; missing-configuration errors and non-probing/process-group checks were confirmed. Unix scripts were normalized to LF, and tar.gz archives preserve mode 0755 for `hame/start.sh`. The Android ARM64 test suite also compiled; it was not run on a device.
