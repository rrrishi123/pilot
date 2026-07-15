#!/bin/bash
# gopher-status.sh — real-time status coordination between gopher minds.
# Each gopher writes its current task and position to a shared file.
# Usage:
#   gopher-status.sh set <name> <status> <description>
#   gopher-status.sh get <name>
#   gopher-status.sh all
#   gopher-status.sh clear <name>
set -euo pipefail

STATUS_DIR="${HOME}/.pilot-gopher-status"
mkdir -p "$STATUS_DIR"

case "${1:-}" in
    set)
        NAME="${2:-unknown}"
        STATUS="${3:-idle}"
        DESC="${4:-}"
        TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        cat > "$STATUS_DIR/$NAME.json" <<EOF
{"name":"$NAME","status":"$STATUS","desc":"$DESC","ts":"$TS"}
EOF
        echo "✅ $NAME → $STATUS"
        ;;
    get)
        NAME="${2:-unknown}"
        if [ -f "$STATUS_DIR/$NAME.json" ]; then
            cat "$STATUS_DIR/$NAME.json"
        else
            echo "{\"name\":\"$NAME\",\"status\":\"unknown\",\"desc\":\"\",\"ts\":\"\"}"
        fi
        ;;
    all)
        echo "["
        first=true
        for f in "$STATUS_DIR"/*.json; do
            [ -f "$f" ] || continue
            $first || echo ","
            first=false
            cat "$f"
        done
        echo "]"
        ;;
    clear)
        NAME="${2:-unknown}"
        rm -f "$STATUS_DIR/$NAME.json"
        echo "✅ $NAME cleared"
        ;;
    *)
        echo "📋 Gopher Status — $(ls "$STATUS_DIR"/*.json 2>/dev/null | wc -l) active gophers"
        echo ""
        echo "Usage:"
        echo "  gopher-status.sh set <name> <status> <desc>"
        echo "  gopher-status.sh get <name>"
        echo "  gopher-status.sh all"
        echo "  gopher-status.sh clear <name>"
        exit 0
        ;;
esac
