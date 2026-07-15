#!/bin/bash
# Build a Reading Nook / Garden Alcove beneath the Grand Avenue
# Location: x=16 to x=22, y=-22 to y=-24
# A small cozy room accessible from the Grand Avenue
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Grand Avenue Alcove..."

# === FLOOR (plank) ===
for x in $(seq 16 22); do
  for y in -22 -23 -24; do
    place $x $y 5
  done
done

# === WALLS (stone) ===
# Left wall (x=15)
for y in -22 -23 -24; do
  place 15 $y 3
done
# Right wall (x=23)
for y in -22 -23 -24; do
  place 23 $y 3
done
# Back wall (y=-25)
for x in $(seq 16 22); do
  place $x -25 3
done
# Ceiling (y=-21)
for x in $(seq 16 22); do
  place $x -21 3
done

# === ENTRANCE (open front at y=-22, x=16-22) ===
# Already open - no blocks needed

# === INTERIOR ===
# Small central table (wood)
place 19 -23 4

# Bookshelves (wood on walls)
place 16 -23 4
place 22 -23 4

# Lantern
place 19 -22 6

# Bench (wood)
place 18 -24 4
place 20 -24 4

# Small plant (grass)
place 17 -23 1
place 21 -23 1

# === STAIRCASE from Grand Avenue (y=-18) down to alcove (y=-22) ===
# at x=16
for y in -19 -20 -21; do
  place 16 $y 3
done

echo "Grand Avenue Alcove complete!"
