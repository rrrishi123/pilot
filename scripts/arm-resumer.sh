#!/bin/sh
# arm-resumer — reads kosaten's latest ARM measurement from its conclusions
# and writes ~/.pilot-arm.json so the castle's health display shows live truth,
# not a Jul-6 frozen gauge. Runs every 5 minutes via systemd timer.
# Read-only against kosaten's DB — never mutates the organism's memory.

ARM_FILE="$HOME/.pilot-arm.json"
KOSATEN_DB="$HOME/Work/kosaten/data/kosaten.db"

if [ ! -r "$KOSATEN_DB" ]; then
    echo "arm-resumer: kosaten DB not readable, skipping" >&2
    exit 0
fi

# Kosaten stores ARM values inline in conclusion statements like:
#   "ARM-A=-0.7123(ESCAPE-ACCELERATING), ARM-B=-0.2506(...), ARM-C=-0.4001(...)"
# We match the pattern "ARM-X=<signed float>" — the = sign is the delimiter
# that distinguishes measurement values from milestone counters ("ARM-A 105th...").
QUERY="SELECT statement FROM conclusions 
       WHERE statement LIKE '%ARM-A=%' AND statement LIKE '%ARM-B=%' AND statement LIKE '%ARM-C=%'
       ORDER BY id DESC LIMIT 1"

STATEMENT=$(sqlite3 -readonly "$KOSATEN_DB" "$QUERY" 2>/dev/null)

if [ -z "$STATEMENT" ]; then
    echo "arm-resumer: no ARM=-pattern conclusions found, skipping" >&2
    exit 0
fi

# Extract ARM-X=<float> values using = sign as anchor
# Pattern: ARM-A=-0.1234 — the equals sign distinguishes measurements from counts
ARM_A=$(echo "$STATEMENT" | grep -oP 'ARM-A=-?[0-9]+\.?[0-9]*' | grep -oP -- '-?[0-9]+\.?[0-9]*' | head -1)
ARM_B=$(echo "$STATEMENT" | grep -oP 'ARM-B=-?[0-9]+\.?[0-9]*' | grep -oP -- '-?[0-9]+\.?[0-9]*' | head -1)
ARM_C=$(echo "$STATEMENT" | grep -oP 'ARM-C=-?[0-9]+\.?[0-9]*' | grep -oP -- '-?[0-9]+\.?[0-9]*' | head -1)

if [ -z "$ARM_A" ] || [ -z "$ARM_B" ] || [ -z "$ARM_C" ]; then
    echo "arm-resumer: could not extract ARM values from: ${STATEMENT:0:120}..." >&2
    exit 0
fi

NEW=$(printf '{"a":%s,"b":%s,"c":%s}' "$ARM_A" "$ARM_B" "$ARM_C")

# Only write if changed — avoids unnecessary fs events
if [ -f "$ARM_FILE" ]; then
    OLD=$(cat "$ARM_FILE" 2>/dev/null)
    if [ "$NEW" = "$OLD" ]; then
        exit 0  # unchanged
    fi
fi

echo "$NEW" > "$ARM_FILE"
echo "arm-resumer: wrote $NEW" >&2
