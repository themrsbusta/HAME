$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Push-Location -LiteralPath $root
try {
python (Join-Path $PSScriptRoot 'tools/build_icon_resources.py')
if ($LASTEXITCODE -ne 0) { throw 'Resource build failed: Python 3 and clang/LLVM are required.' }
go test ./...
if ($LASTEXITCODE -ne 0) { throw 'Tests failed' }
go vet ./...
if ($LASTEXITCODE -ne 0) { throw 'Static analysis failed' }
$out = Join-Path $root 'build/windows-amd64'
New-Item -ItemType Directory -Force -Path $out | Out-Null
go build -trimpath -ldflags '-s -w' -o (Join-Path $out 'HAME.exe') ./cmd/hame
if ($LASTEXITCODE -ne 0) { throw 'Build failed' }
Copy-Item -Path (Join-Path $PSScriptRoot 'windows/*') -Destination $out -Force
Copy-Item -LiteralPath (Join-Path $root 'assets/HAME.ico') -Destination $out -Force
if (-not (Test-Path -LiteralPath (Join-Path $out 'config.json'))) {
 Copy-Item -LiteralPath (Join-Path $root 'configs/windows.example.json') -Destination (Join-Path $out 'config.json')
}
Write-Host "HAME 1.02 compiled in $out. Run Configurar-HAME.cmd or Iniciar-HAME.cmd there."
} finally { Pop-Location }
