#!/bin/bash
# Build the Deep Forge - a subterranean forge chamber beneath the existing forge
# Located at x=105-112, y=-22 to y=-30, accessible via the staircase at x=113-114
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Deep Forge Chamber..."

# === DEEP FORGE FLOOR (y=-30) ===
for x in $(seq 105 112); do
  place $x -30 5
done

# === DEEP FORGE WALLS ===
# Left wall (x=105)
for y in $(seq -22 -29); do
  place 105 $y 3
done

# Right wall (x=112)
for y in $(seq -22 -29); do
  place 112 $y 3
done

# Back wall (y=-22)
for x in $(seq 106 111); do
  place $x -22 3
done

# === STAIRCASE from existing forge (y=-20) down to deep forge (y=-30) ===
# The staircase is at x=113-114
for step in $(seq 0 9); do
  sy=$((-20 - step))
  sx=$((113 + step / 3))
  place $sx $sy 5
  place $((sx + 1)) $sy 5
done

# === CENTRAL FORGE PIT ===
place 108 -29 0
place 109 -29 0
place 108 -28 0
place 109 -28 0

# Forge fire glow
place 108 -29 6
place 109 -29 6

# === ANVIL AREA ===
place 107 -27 5
place 110 -27 5

# === STORAGE LEDGES ===
for x in 106 111; do
  place $x -24 5
  place $x -25 5
  place $x -26 5
done

# === LIGHTING ===
for x in 107 110; do
  place $x -23 6
done

echo "Deep Forge Chamber complete!"
