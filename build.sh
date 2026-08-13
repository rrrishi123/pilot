#!/usr/bin/env bash
# build.sh — canonical build for pilot. Each main package gets its own binary in
# .bin/ so none can clobber another. Build with THIS, not ad-hoc `go build -o`.
set -euo pipefail
root="$(cd "$(dirname "$0")" && pwd)"
cd "$root"
mkdir -p .bin
go build -o .bin/pilot  .            # the pilot HOST
go build -o .bin/broker ./cmd/broker # the broker
go build -o .bin/castle ./cmd/castle # the castle (living-world view)
echo "built: $root/.bin/{pilot,broker,castle}"
