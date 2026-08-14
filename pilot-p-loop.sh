#!/usr/bin/env bash
# pilot-p-loop.sh — keeps a kosaten-bridged pilot alive as a persistent agent.
# pilot -p is one-shot (stdin → turn → exit), so we loop it with periodic nudges.
# Bounded: exits after 12h or if /tmp/pilot-p-loop.stop exists. Restarted by systemd.
set -uo pipefail

PILOT="${PILOT:-$HOME/.local/bin/pilot}"  # #359: $HOME-derived, env-overridable
CASTLE="$HOME/.pilot-castle.jsonl"
KOSATEN="http://localhost:3942"
STOP=/tmp/pilot-p-loop.stop
MAX_AGE=$(( 12 * 3600 ))  # 12h max
START=$(date +%s)

log() { echo "$(date -u +%FT%TZ) pilot-p-loop: $*"; }

cleanup() {
    log "exiting (signal or bounded end)"
    rm -f "$STOP"
    exit 0
}
trap cleanup SIGTERM SIGINT

log "started — will run for 12h max, checking kosaten every 5 min"

# Initial prompt — survey the organism state
PROMPT="You are pilot-p, the autonomous kosaten agent. Survey the organism: check universe_status, get_health, digest any findings. Report what you see and take ONE action if something needs attention."

while true; do
    now=$(date +%s)
    if [ -f "$STOP" ] || [ $(( now - START )) -gt "$MAX_AGE" ]; then
        log "bounded end reached"
        rm -f "$STOP"
        exit 0
    fi

    log "dispatching turn..."

    # Feed prompt via stdin, capture output to journal
    echo "$PROMPT" | timeout 300 "$PILOT" -p -yes \
        -kosaten "$KOSATEN" \
        -max-steps 100 \
        -castle "$CASTLE" \
        -server "${HTTP_MCP:-$HOME/Work/http-mcp/.bin/http-mcp}" 2>&1

    rc=$?
    log "turn completed (rc=$rc)"

    # If kosaten is unhealthy, the next prompt asks about it
    if ! curl -s -m5 "$KOSATEN/health" 2>/dev/null | grep -q 'avg_confidence'; then
        PROMPT="Kosaten may be down — check and report."
    else
        PROMPT="Survey kosaten again: check universe_status, review the digest, the pulse, and any open limitations. Report what changed and take ONE action if needed."
    fi

    sleep 300  # 5 min between turns
done
