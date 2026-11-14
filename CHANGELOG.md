# Changelog

## 1.02 — 2026-10-29

- Fixed corrupted accents and separators in the WebUI and this changelog.
- Added a UTF-8 BOM to PowerShell scripts for correct reading by Windows PowerShell 5.1.
- Declared the UTF-8 charset before HTML scripts; checks protect interface text and script encoding.

## 0.92 RC — 2026-10-27

- Fixed the trigger: compaction now occurs on saturation or explicit recovery, not after every response.
- Counted the formatted server prompt; added an auxiliary tokenizer and an explicit CLI reserve.
- Constrained keeps the main model between interactions and switches only during compaction.
- Resident keeps both processes; clears and verifies a slot without unloading the main model.
- `context_state` is again produced semantically by the main model.
- Added incremental compilation for new/changed sources and optional full recovery.
- Organized the GUI into tabs with policies, reserves, memory limit, prefill, manual actions, and a real mode test.
- Added indicators and controls as an adaptation to the official WebUI.
- Replaced the lightning logo with the HAME icon in native components; kept the favicon.
- Added configuration migration and preference preservation; old data is not modified.
- Added a separate rollback directory for 0.9 RC.

The architectural version of the original specification remains 1.3; the application version is 0.92 RC.

### Photos and YAML memory

- Added server images with mmproj: original preserved, YAML description, and local retrieval route.
- The compiler reads YAML; selected references are preserved outside the summary, within the budget.
- A real test after compaction and restart recovered FOTO-091 and the link to the original.

## Portability in 0.92 RC

- Preserved Windows x64 with WinForms and detection.
- Linux x64/ARM64 and Termux Android ARM64 use manual configuration, without probing at startup.
- Added Unix process groups and SIGTERM/SIGHUP/Ctrl+C handling.
- Added explicit JSON examples and build/run scripts for each target.
