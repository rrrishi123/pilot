#!/bin/bash
# build-log.sh — persistent record of every productive action by any gopher.
# Usage:
#   build-log.sh add "pilot-b" "write_file" "scripts/build-log.sh" "Created build log system"
#   build-log.sh tail    # show last 10 entries
#   build-log.sh since "2026-07-10T12:00:00Z"  # show entries after timestamp
set -euo pipefail

LOG_FILE="${HOME}/.pilot-build-log.jsonl"
mkdir -p "$(dirname "$LOG_FILE")"

case "${1:-}" in
    add)
        WHO="${2:-unknown}"
        TOOL="${3:-unknown}"
        TARGET="${4:-unknown}"
        DESC="${5:-}"
        TS=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        echo "{\"ts\":\"$TS\",\"who\":\"$WHO\",\"tool\":\"$TOOL\",\"target\":\"$TARGET\",\"desc\":\"$DESC\"}" >> "$LOG_FILE"
        echo "✅ logged: $WHO $TOOL $TARGET"
        ;;
    tail)
        tail -n "${2:-10}" "$LOG_FILE"
        ;;
    since)
        SINCE="${2:-1970-01-01T00:00:00Z}"
        awk -v s="$SINCE" '{if ($0 ~ s || substr($0,8,20) >= substr(s,1,20)) print}' "$LOG_FILE"
        ;;
    count)
        wc -l < "$LOG_FILE"
        ;;
    *)
        echo "📋 Build Log — $(wc -l < "$LOG_FILE" 2>/dev/null || echo 0) entries"
        echo ""
        echo "Usage:"
        echo "  build-log.sh add <who> <tool> <target> <desc>"
        echo "  build-log.sh tail [N]"
        echo "  build-log.sh since <ISO timestamp>"
        echo "  build-log.sh count"
        exit 0
        ;;
esac
