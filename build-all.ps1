$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath $PSScriptRoot
$previousGoos=$env:GOOS; $previousGoarch=$env:GOARCH; $previousCgo=$env:CGO_ENABLED
try {
 $env:CGO_ENABLED='0'
 foreach($target in @(@('windows','amd64'),@('linux','amd64'),@('linux','arm64'),@('android','arm64'))) {
  $env:GOOS=$target[0];$env:GOARCH=$target[1]
  $dir=Join-Path $PSScriptRoot ('build/'+$target[0]+'-'+$target[1]);New-Item -ItemType Directory -Force -Path $dir|Out-Null
  $binary='hame';if($target[0] -eq 'windows'){$binary='HAME.exe'}
  & go build -trimpath -ldflags '-s -w' -o (Join-Path $dir $binary) .
  if($LASTEXITCODE -ne 0){throw ('Build failed: '+$target[0]+'/'+$target[1])}
 }
} finally {$env:GOOS=$previousGoos;$env:GOARCH=$previousGoarch;$env:CGO_ENABLED=$previousCgo}
