#!/bin/bash
# gopher-summary.sh — produce a JSON summary of all gopher activity.
# This is the coordination hub: reads build-log, gopher-status, and gopher-memory
# to produce a single unified view. Gophers can call this to orient themselves.
# Usage:
#   gopher-summary.sh        # print full summary as JSON
#   gopher-summary.sh --short  # print compact one-liner
set -euo pipefail

STATUS_DIR="${HOME}/.pilot-gopher-status"
MEMORY_DIR="${HOME}/.pilot-gopher-memory"
LOG_FILE="${HOME}/.pilot-build-log.jsonl"

echo "{"
echo "  \"ts\": \"$(date -u +"%Y-%m-%dT%H:%M:%SZ")\","

# Active gophers from status
echo "  \"active_gophers\": ["
first=true
for f in "$STATUS_DIR"/*.json; do
    [ -f "$f" ] || continue
    $first || echo ","
    first=false
    cat "$f"
done
echo ""
echo "  ],"

# Build log count
LOG_COUNT=0
[ -f "$LOG_FILE" ] && LOG_COUNT=$(wc -l < "$LOG_FILE")
echo "  \"build_log_entries\": $LOG_COUNT,"

# Recent build log (last 5)
echo "  \"recent_actions\": ["
if [ -f "$LOG_FILE" ]; then
    tail -5 "$LOG_FILE" | while IFS= read -r line; do
        echo "    $line,"
    done
fi
echo "    null"
echo "  ],"

# Memory keys
echo "  \"memories\": ["
first=true
for f in "$MEMORY_DIR"/*; do
    [ -f "$f" ] || continue
    $first || echo ","
    first=false
    name=$(basename "$f")
    val=$(cat "$f" | head -c 200)
    echo "    {\"key\": \"$name\", \"value\": \"$val\"}"
done
echo ""
echo "  ],"

# Presence
echo "  \"presence\": ["
first=true
for f in "${HOME}/.pilot-castle-presence"/*; do
    [ -f "$f" ] || continue
    $first || echo ","
    first=false
    cat "$f"
done
echo ""
echo "  ]"

echo "}"
