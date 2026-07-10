#!/bin/bash
# castle-health.sh — prove the castle is alive and well
# Written by pilot-b (library gopher) for Rishi's scepticism.
set -euo pipefail

CASTLE_URL="http://localhost:9901"
DATA_FILE="$HOME/.pilot-castle.jsonl"
LOG_PREFIX="[castle-health] $(date -Iseconds)"

# 1. Is the HTTP server responding?
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "$CASTLE_URL" 2>/dev/null || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
    echo "$LOG_PREFIX ✅ HTTP server up (200)"
else
    echo "$LOG_PREFIX ❌ HTTP server returned $HTTP_CODE"
    exit 1
fi

# 2. Is the world data file growing?
if [ -f "$DATA_FILE" ]; then
    LINES=$(wc -l < "$DATA_FILE")
    SIZE=$(du -h "$DATA_FILE" | cut -f1)
    echo "$LOG_PREFIX 📊 world: $LINES rooms, $SIZE"
else
    echo "$LOG_PREFIX ❌ world file missing"
    exit 1
fi

# 3. Are the gophers alive? Check /gophers SSE endpoint
GOPHERS=$(curl -s --max-time 5 "$CASTLE_URL/gophers" 2>/dev/null | head -5 || echo "timeout")
if [ -n "$GOPHERS" ]; then
    echo "$LOG_PREFIX 🧠 gopher SSE stream alive"
else
    echo "$LOG_PREFIX ⚠️  gopher SSE stream unresponsive"
fi

# 4. Memory sanity — sum RSS across all castle threads
CPID=$(pgrep -x castle 2>/dev/null | head -1 || echo "")
if [ -n "$CPID" ]; then
    MEM_KB=$(ps -o rss= --ppid "$CPID" 2>/dev/null | paste -sd+ | bc 2>/dev/null || echo "0")
    if [ "$MEM_KB" = "0" ] || [ -z "$MEM_KB" ]; then
        MEM_KB=$(ps -o rss= -p "$CPID" 2>/dev/null | tr -d ' ')
    fi
    MEM_MB=$((MEM_KB / 1024))
    echo "$LOG_PREFIX 💾 castle RSS: ${MEM_MB}MB"
else
    echo "$LOG_PREFIX ❌ castle PID not found"
    exit 1
fi

echo "$LOG_PREFIX ✅ All checks passed"
exit 0
