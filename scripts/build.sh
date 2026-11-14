#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
if [ "$#" -ne 2 ]; then echo 'Usage: sh scripts/build.sh linux amd64 | linux arm64 | android arm64' >&2; exit 2; fi
case "$1/$2" in linux/amd64|linux/arm64|android/arm64) ;; *) echo 'Unsupported target' >&2; exit 2;; esac
out="build/$1-$2"
mkdir -p "$out"
CGO_ENABLED=0 GOOS="$1" GOARCH="$2" go build -trimpath -ldflags '-s -w' -o "$out/hame" ./cmd/hame
cp scripts/unix/start.sh "$out/start.sh"
chmod +x "$out/hame" "$out/start.sh"
if [ ! -e "$out/config.json" ]; then cp configs/unix-server.json "$out/config.json"; fi
printf 'Built: %s/hame\nEdit %s/config.json before starting.\n' "$out" "$out"
