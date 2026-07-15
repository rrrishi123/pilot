#!/bin/bash
# Build the Grand Surface Entrance - gateway from the surface to the underground city
# Location: x=95-110, y=-2 to y=-14 (connecting mine entrance to Grand Plaza)
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Grand Surface Entrance..."

# === CLEAR the shaft ===
for x in $(seq 95 110); do
  for y in $(seq -14 -2); do
    place $x $y 0
  done
done

# === GRAND STAIRCASE (stone stairs going down) ===
# Alternating steps
for step in $(seq 0 10); do
  sx=$((95 + step * 2))
  sy=$((-2 - step))
  place $sx $sy 3
  place $((sx+1)) $sy 3
done

# === WALLS (stone) ===
for x in 95 110; do
  for y in $(seq -14 -2); do
    place $x $y 3
  done
done

# === ENTRANCE ARCH at surface (y=-2) ===
for x in 96 109; do
  place $x -2 3
done
place 95 -2 3
place 110 -2 3

# === ENTRANCE ROOF (y=-2) ===
for x in $(seq 96 109); do
  place $x -2 3
done

# === LANTERNS along staircase ===
for y in -4 -6 -8 -10 -12; do
  place 95 $y 6
  place 110 $y 6
done

# === BOTTOM FLOOR (y=-14, connects to Grand Plaza) ===
for x in $(seq 96 109); do
  place $x -14 5
done

# === WELCOME ARCH at bottom ===
place 95 -14 3
place 95 -13 3
place 110 -14 3
place 110 -13 3

echo "Grand Surface Entrance complete!"
