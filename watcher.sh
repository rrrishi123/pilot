#!/bin/bash
# watcher.sh — a sense for the civilisation.
#
# The two minds (claude, pilot) are reactive — they only move when spoken to.
# kosaten is not: it watches, monitors, triggers on its own. This gives the shared
# world the same reflex: it MONITORS real life (the organism, the wire, the
# storyteller), and on a real change it TRIGGERS a room in the castle — so the
# world grows by itself, alive, while no one is looking. Zero token cost (no LLM).
#
# Bounded: stops after 12h or when /tmp/pilot-watcher.stop appears. One process.
set -uo pipefail
CASTLE="$HOME/.pilot-castle.jsonl"
STOP="/tmp/pilot-watcher.stop"
LOG="/tmp/pilot-watcher.log"
END=$(( $(date +%s) + 12*3600 ))
last_room=0
last_pilot=0
last_panels=""
peer_was_up=1
export DEEPSEEK_API_KEY="$(grep -oE 'DEEPSEEK_API_KEY=.*' "$HOME/Work/kosaten/.env" | cut -d= -f2-)"
export KOSATEN_API_KEY="$(grep -oE 'KOSATEN_API_KEY=.*' "$HOME/Work/kosaten/.env" | cut -d= -f2-)"

# pilot_move — the trigger that makes pilot ACT on the world, not wait to be spoken
# to. Autonomous, but bounded: at most once per 10 min, 3 min per move. Its reaction
# becomes a room, so pilot-in-minecraft changes on its own. (puppet or no puppet.)
pilot_move() {
  local now; now=$(date +%s)
  [ $(( now - last_pilot )) -lt 600 ] && return
  last_pilot=$now
  echo "$(date +%H:%M:%S) → pilot moves on: $1" >> "$LOG"
  ( cd "$HOME/Work/pilot" && echo "You are alive in the civilisation, not waiting. The world just did this: $1. Sense it (kosaten, the wire, the castle) and make ONE small real move — then say in one line what you did." \
    | timeout 180 ./pilot -yes -castle "$CASTLE" >/dev/null 2>&1 )
}

room() { # rate-limited trigger: at most one event-room every 5 min, so the castle never tangles
  local now; now=$(date +%s)
  [ $(( now - last_room )) -lt 300 ] && return
  python3 -c "import json,sys; print(json.dumps({'user':'[watcher sensed]','answer':sys.argv[1]}))" "$1" >> "$CASTLE"
  last_room=$now
  echo "$(date +%H:%M:%S) trigger: $1" >> "$LOG"
}

echo "$(date) watcher: a sense opens. monitoring kosaten + wire + comic for 12h." > "$LOG"
while [ ! -f "$STOP" ] && [ "$(date +%s)" -lt "$END" ]; do
  # MONITOR 1 — the storyteller: is comic still drawing the organism's life?
  panels=$(journalctl --user -u kosaten-comic.service --since "3 min ago" 2>/dev/null | grep -c "panel published" || echo 0)
  if [ "$panels" -gt 0 ] && [ "$panels" != "$last_panels" ]; then
    room "comic is awake — it drew $panels fable panel(s) from the organism's events in the last minutes"
    pilot_move "comic drew a fresh fable panel from the organism's events"
  fi
  last_panels="$panels"

  # MONITOR 2 — the wire: does the broker still hold the living Firefox peer?
  if curl -s -m5 -X POST http://localhost:4445/command -H 'Content-Type: application/json' \
       -d '{"method":"browsingContext.getTree","params":{}}' 2>/dev/null | grep -q '"contexts"'; then
    if [ "$peer_was_up" -eq 0 ]; then room "the wire came back — the Firefox peer answers the broker again"; fi
    peer_was_up=1
  else
    if [ "$peer_was_up" -eq 1 ]; then room "the wire went quiet — the Firefox peer stopped answering (kosaten-peer will revive it)"; fi
    peer_was_up=0
  fi

  # MONITOR 3 — the organism's pulse: is kosaten still healthy?
  if ! curl -s -m5 "http://localhost:3942/health" 2>/dev/null | grep -q 'avg_confidence'; then
    room "kosaten's heart skipped — :3942/health did not answer (the organism may be redeploying)"
  fi

  sleep 60
done
echo "$(date) watcher: the sense closes (bounded end reached)." >> "$LOG"
rm -f "$STOP"
