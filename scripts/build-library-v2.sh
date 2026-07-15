#!/bin/bash
# Build the Great Library - adjacent to Council Chamber
# Location: x=115-124, y=-22 to y=-28 (below Council Chamber)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Great Library..."

# === EXCAVATE ===
for x in $(seq 115 124); do
  for y in $(seq -28 -21); do
    place $x $y 0
  done
done

# === FLOOR (stone, y=-28) ===
for x in $(seq 115 124); do
  place $x -28 3
done

# === WALLS (stone) ===
for x in 115 124; do
  for y in $(seq -28 -21); do
    place $x $y 3
  done
done
for y in -28 -21; do
  for x in $(seq 116 123); do
    place $x $y 3
  done
done

# === BOOKSHELVES (planks along walls) ===
# Left wall
for y in $(seq -27 -22); do
  place 115 $y 5
done
# Right wall
for y in $(seq -27 -22); do
  place 124 $y 5
done
# Back wall
for x in 116 117 118 119 120 121 122 123; do
  place $x -27 5
done

# === READING TABLE (wood) ===
for x in 117 118 119 120 121 122; do
  place $x -24 4
done

# === STAIRCASE from Council Chamber (y=-13) down to Library (y=-21) ===
# Staircase at x=119-120
for y in $(seq -14 -21); do
  place 119 $y 3
  place 120 $y 3
done

# === DOORWAY at top of stairs ===
place 119 -13 0
place 120 -13 0

# === LANTERNS ===
place 116 -22 6
place 123 -22 6
place 117 -26 6
place 122 -26 6

# === STUDY DESKS (planks at floor level) ===
place 116 -28 5
place 123 -28 5
place 118 -28 5
place 121 -28 5

echo "Great Library complete!"
