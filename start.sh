#!/bin/sh
set -eu
cd "$(dirname "$0")"
exec ./hame -config ./config.json -open=false "$@"
