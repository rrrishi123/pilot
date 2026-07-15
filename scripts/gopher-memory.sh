#!/bin/bash
# gopher-memory.sh — persistent key-value memory for gopher minds.
# Each gopher can store and retrieve memories across restarts.
# Usage:
#   gopher-memory.sh set <key> <value>
#   gopher-memory.sh get <key>
#   gopher-memory.sh list [prefix]
#   gopher-memory.sh delete <key>
set -euo pipefail

MEM_DIR="${HOME}/.pilot-gopher-memory"
mkdir -p "$MEM_DIR"

case "${1:-}" in
    set)
        KEY="${2:-}"
        VALUE="${3:-}"
        if [ -z "$KEY" ] || [ -z "$VALUE" ]; then
            echo "Usage: gopher-memory.sh set <key> <value>"
            exit 1
        fi
        TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        echo "{\"key\":\"$KEY\",\"value\":\"$VALUE\",\"ts\":\"$TS\"}" > "$MEM_DIR/$(echo "$KEY" | md5sum | cut -c1-16).json"
        echo "✅ saved: $KEY"
        ;;
    get)
        KEY="${2:-}"
        if [ -z "$KEY" ]; then
            echo "Usage: gopher-memory.sh get <key>"
            exit 1
        fi
        F="$MEM_DIR/$(echo "$KEY" | md5sum | cut -c1-16).json"
        if [ -f "$F" ]; then
            cat "$F"
        else
            echo "{\"key\":\"$KEY\",\"value\":\"\",\"ts\":\"\"}"
        fi
        ;;
    list)
        PREFIX="${2:-}"
        for f in "$MEM_DIR"/*.json; do
            [ -f "$f" ] || continue
            if [ -n "$PREFIX" ]; then
                grep -q "\"$PREFIX\"" "$f" && cat "$f" && echo "---"
            else
                cat "$f"
                echo "---"
            fi
        done
        ;;
    delete)
        KEY="${2:-}"
        rm -f "$MEM_DIR/$(echo "$KEY" | md5sum | cut -c1-16).json"
        echo "✅ deleted: $KEY"
        ;;
    *)
        echo "📝 Gopher Memory — $(ls "$MEM_DIR"/*.json 2>/dev/null | wc -l) memories stored"
        echo ""
        echo "Usage:"
        echo "  gopher-memory.sh set <key> <value>"
        echo "  gopher-memory.sh get <key>"
        echo "  gopher-memory.sh list [prefix]"
        echo "  gopher-memory.sh delete <key>"
        exit 0
        ;;
esac
