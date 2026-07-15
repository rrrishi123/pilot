#!/bin/bash
# Build Underground Kitchen & Dining Hall at x=50-70, y=-25 to y=-30
# A place for all gophers to gather and eat!
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Underground Kitchen & Dining Hall..."

# === ROOM BOUNDARIES ===
# Floor (y=-29) - stone
for x in $(seq 50 70); do
  place $x -29 3
done

# Ceiling (y=-25) - stone
for x in $(seq 50 70); do
  place $x -25 3
done

# Walls
for y in $(seq -26 -1 -29); do
  place 49 $y 3
  place 71 $y 3
done
for x in $(seq 50 70); do
  place $x -30 3
done

# === STAIRCASE UP TO LEVEL 2 (y=-22) ===
# Staircase at x=60 going up
for y in -23 -24 -25 -26 -27 -28 -29; do
  place 60 $y 3
done

# === DINING AREA (west side, x=50-60) ===
# Long dining table (wood)
for x in $(seq 52 58); do
  place $x -27 4
done

# Benches (wood) around table
for x in $(seq 52 58); do
  place $x -26 4
  place $x -28 4
done

# === KITCHEN AREA (east side, x=62-70) ===
# Counter (wood)
for x in $(seq 63 69); do
  place $x -27 4
done

# Cooking fire (grass = green fire pit)
place 65 -28 1
place 66 -28 1
place 65 -29 1
place 66 -29 1

# Shelves (wood on wall)
place 62 -26 4
place 64 -26 4
place 66 -26 4
place 68 -26 4
place 70 -26 4

# === DECORATION ===
# Lanterns
place 52 -25 6
place 60 -25 6
place 68 -25 6

# Plants
place 51 -28 1
place 70 -28 1

# Welcome mat (dirt)
place 60 -29 2

echo "Kitchen & Dining Hall complete!"
echo "Located at x=50-70, y=-25 to y=-30"
echo "Staircase at x=60 connects up to Level 2 passage"
