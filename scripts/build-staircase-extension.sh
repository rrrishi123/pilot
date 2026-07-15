#!/bin/bash
# Extend the Grand Staircase from y=-28 down to y=-30 to connect to the Deep Forge
# Also build a landing corridor at y=-29 connecting x=113 to x=115
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Staircase Extension to Deep Forge..."

# === EXTEND STAIRCASE STEPS (y=-28 to y=-30) ===
# y=-28 already has steps, add next ones going down
place 113 -29 3  # Stone step
place 114 -30 5  # Plank step

# === LANDING at y=-29 connecting to deep forge ===
# Clear the path
place 115 -29 0
place 116 -29 0

# Landing floor
place 113 -29 5  # Already set above as stone, make it plank landing
place 114 -29 5
place 115 -29 5
place 116 -29 5

# === ENTRANCE ARCH to Deep Forge at x=115-116 ===
# Left pillar
place 115 -30 3
place 115 -31 3
place 115 -32 3
# Right pillar
place 116 -30 3
place 116 -31 3
place 116 -32 3
# Arch top
place 115 -33 3
place 116 -33 3

# === LANTERN at landing ===
place 114 -30 6

# === SUPPORT WALLS around staircase ===
for y in $(seq -28 -31); do
  place 112 $y 3
  place 117 $y 3
done

echo "Staircase Extension complete!"
