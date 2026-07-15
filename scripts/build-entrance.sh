#!/bin/bash
# Build the Surface Entrance - a stone archway marking the entrance to the underground
# Location: x=108-112, y=0 to y=6 (above the Grand Staircase)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Surface Entrance Archway..."

# === LEFT PILLAR (x=108) ===
for y in $(seq 1 5); do
  place 108 $y 3
done

# === RIGHT PILLAR (x=111) ===
for y in $(seq 1 5); do
  place 111 $y 3
done

# === ARCH TOP (y=5) ===
for x in $(seq 109 110); do
  place $x 5 3
done

# === LANTERNS on pillars ===
place 108 5 6
place 111 5 6

# === SIGNPOST (wood at x=107) ===
for y in $(seq 1 4); do
  place 107 $y 4
done
# Sign board
place 107 5 5
place 106 5 5
place 106 4 5

# === PATH leading to archway (stone path on ground, y=0) ===
for x in $(seq 105 114); do
  place $x 0 3
done

echo "Surface Entrance complete!"
