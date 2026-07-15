#!/bin/bash
# gopher-place-block.sh — actually place a block in the castle world.
# Uses the /edit POST endpoint to modify the persistent world state.
# This is how gophers BUILD things instead of just talking about building.
#
# Usage:
#   gopher-place-block.sh <who> <x> <y> <block_type>
#     block_type: 0=air (break), 1=stone, 2=dirt, 3=wood, 4=glass, 5=brick
#
# Examples:
#   gopher-place-block.sh pilot-b 10 5 3   # place wood at (10,5)
#   gopher-place-block.sh pilot-b 10 5 0   # break block at (10,5)
set -euo pipefail

WHO="${1:-}"
X="${2:-}"
Y="${3:-}"
BLOCK="${4:-}"

if [ -z "$WHO" ] || [ -z "$X" ] || [ -z "$Y" ] || [ -z "$BLOCK" ]; then
    echo "Usage: gopher-place-block.sh <who> <x> <y> <block_type>"
    echo "  block_type: 0=air(break), 1=stone, 2=dirt, 3=wood, 4=glass, 5=brick"
    exit 1
fi

CASTLE_URL="${CASTLE_URL:-http://localhost:9901}"

# Validate block type
if [ "$BLOCK" -lt 0 ] || [ "$BLOCK" -gt 5 ]; then
    echo "Invalid block type: $BLOCK (must be 0-5)"
    exit 1
fi

# Send the edit
RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$CASTLE_URL/edit" \
    -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$X,\"y\":$Y,\"b\":$BLOCK}" 2>/dev/null || echo "error")

if [ "$RESPONSE" = "204" ]; then
    BLOCK_NAMES=("air(break)" "stone" "dirt" "wood" "glass" "brick")
    echo "✅ $WHO placed ${BLOCK_NAMES[$BLOCK]} at ($X, $Y)"
else
    echo "❌ Failed to place block (HTTP $RESPONSE)"
    exit 1
fi
