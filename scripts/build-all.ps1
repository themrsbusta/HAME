$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
Push-Location -LiteralPath $root
$previousGoos=$env:GOOS; $previousGoarch=$env:GOARCH; $previousCgo=$env:CGO_ENABLED
try {
 $env:CGO_ENABLED='0'
 $env:GOOS='windows'; $env:GOARCH='amd64'
 & (Join-Path $PSScriptRoot 'build.ps1')
 foreach($target in @(@('linux','amd64'),@('linux','arm64'),@('android','arm64'))) {
  $env:GOOS=$target[0];$env:GOARCH=$target[1]
  $dir=Join-Path $root ('build/'+$target[0]+'-'+$target[1]);New-Item -ItemType Directory -Force -Path $dir|Out-Null
  $binary='hame';if($target[0] -eq 'windows'){$binary='HAME.exe'}
  & go build -trimpath -ldflags '-s -w' -o (Join-Path $dir $binary) ./cmd/hame
  if($LASTEXITCODE -ne 0){throw ('Build failed: '+$target[0]+'/'+$target[1])}
  Copy-Item -LiteralPath (Join-Path $PSScriptRoot 'unix/start.sh') -Destination $dir -Force
  if (-not (Test-Path -LiteralPath (Join-Path $dir 'config.json'))) {
   Copy-Item -LiteralPath (Join-Path $root 'configs/unix-server.json') -Destination (Join-Path $dir 'config.json')
  }
 }
} finally {$env:GOOS=$previousGoos;$env:GOARCH=$previousGoarch;$env:CGO_ENABLED=$previousCgo;Pop-Location}
