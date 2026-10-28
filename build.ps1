$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath $PSScriptRoot
if (-not (Test-Path -LiteralPath 'config.json')) { Copy-Item -LiteralPath 'config.example.json' -Destination 'config.json' }
if (-not (Test-Path -LiteralPath 'hame_windows_amd64.syso') -or (Get-Item -LiteralPath 'HAME.ico').LastWriteTime -gt (Get-Item -LiteralPath 'hame_windows_amd64.syso').LastWriteTime -or (Get-Item -LiteralPath 'build_icon_resources.py').LastWriteTime -gt (Get-Item -LiteralPath 'hame_windows_amd64.syso').LastWriteTime) {
 python build_icon_resources.py
 if ($LASTEXITCODE -ne 0) { throw 'Icon resource build failed: Python 3 and clang/LLVM are required.' }
}
go test ./...
if ($LASTEXITCODE -ne 0) { throw 'Tests failed' }
go vet ./...
if ($LASTEXITCODE -ne 0) { throw 'Static analysis failed' }
go build -trimpath -ldflags '-s -w' -o HAME.exe .
if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
Write-Host 'HAME 0.9 RC compiled. Run Configurar-HAME.cmd or Iniciar-HAME.cmd.'
