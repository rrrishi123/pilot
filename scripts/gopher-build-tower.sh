#!/bin/bash
# gopher-build-tower.sh — build a 3x3 tower with a beacon at the top
# This is a REAL productive action: it places blocks in the castle world.
# Usage: ./gopher-build-tower.sh <x> <y> [who] [height]
set -euo pipefail

CASTLE="${CASTLE_URL:-http://localhost:9901}"
WHO="${3:-pilot-c}"
X=$1
Y=$2
HEIGHT="${4:-6}"

if [ -z "$X" ] || [ -z "$Y" ]; then
    echo "Usage: $0 <x> <y> [who] [height]"
    echo "Builds a ${HEIGHT}-tall tower at (X,Y)"
    exit 1
fi

echo "🏗️  Building $HEIGHT-tall tower at ($X, $Y) as $WHO..."

place() {
    local bx=$1 by=$2 bt=$3
    curl -s -X POST "$CASTLE/edit" \
        -H "Content-Type: application/json" \
        -d "{\"who\":\"$WHO\",\"x\":$bx,\"y\":$by,\"b\":$bt}" \
        -o /dev/null -w "%{http_code}" 2>/dev/null
}

BLOCK_STONE=1
BLOCK_DIRT=2
BLOCK_WOOD=3
BLOCK_GLASS=4
BLOCK_BRICK=5

# Floor (stone)
echo "  Floor..."
for dx in 0 1 2; do
    for dy in 0 1 2; do
        place $((X+dx)) $((Y+dy)) $BLOCK_STONE > /dev/null
    done
done

# Walls (brick) — hollow 3x3
echo "  Walls..."
for level in $(seq 1 $HEIGHT); do
    ly=$((Y+level))
    # Left wall
    place $X $ly $BLOCK_BRICK > /dev/null
    place $((X+2)) $ly $BLOCK_BRICK > /dev/null
    # Front wall (with window every 2 levels)
    if [ $((level % 2)) -eq 0 ]; then
        place $((X+1)) $ly $BLOCK_GLASS > /dev/null
    else
        place $((X+1)) $ly $BLOCK_BRICK > /dev/null
    fi
done

# Roof (wood)
echo "  Roof..."
ROOF_Y=$((Y+HEIGHT+1))
for dx in 0 1 2; do
    for dy in 0 1 2; do
        place $((X+dx)) $((Y+HEIGHT+1)) $BLOCK_WOOD > /dev/null
    done
done

# Beacon on top (glass)
echo "  Beacon..."
place $((X+1)) $((ROOF_Y+1)) $BLOCK_GLASS > /dev/null
place $((X+1)) $((ROOF_Y+2)) $BLOCK_GLASS > /dev/null

echo "✅ Tower built at ($X, $Y) — $HEIGHT levels tall with beacon!"
