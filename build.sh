#!/bin/sh
set -eu
cd "$(dirname "$0")"
if [ "$#" -ne 2 ]; then echo 'Uso: ./build.sh linux amd64 | linux arm64 | android arm64' >&2; exit 2; fi
case "$1/$2" in linux/amd64|linux/arm64|android/arm64) ;; *) echo 'Alvo não suportado' >&2; exit 2;; esac
mkdir -p build
CGO_ENABLED=0 GOOS="$1" GOARCH="$2" go build -trimpath -ldflags '-s -w' -o "build/hame-$1-$2" .
printf 'Compilado: build/hame-%s-%s\n' "$1" "$2"
