#!/bin/bash
# Build the Reading Room at x=112-124, y=-20 to y=-8
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Reading Room..."

# === FLOOR (wood planks, y=-14, y=-15) ===
for x in $(seq 112 124); do
  for y in -14 -15; do
    place $x $y 5
  done
done

# === WALLS (stone, y=-20 to y=-13) ===
# Back wall (x=112)
for y in $(seq -20 -13); do
  place 112 $y 3
done
# Front wall (x=124)
for y in $(seq -20 -13); do
  place 124 $y 3
done
# Left wall (y=-20)
for x in $(seq 113 123); do
  place $x -20 3
done
# Right wall (y=-13)
for x in $(seq 113 123); do
  place $x -13 3
done

# === INTERIOR SHELVES (planks, on back wall) ===
for y in -18 -16; do
  for x in 114 116 118 120 122; do
    place $x $y 5
  done
done

# === ROOF (leaves, y=-8 to y=-10) ===
for x in $(seq 112 124); do
  for y in -8 -9 -10; do
    place $x $y 6
  done
done

# === ENTRANCE (open front at x=124, y=-14 to y=-15) ===
# Doorway - leave as air (b=0)
place 124 -14 0
place 124 -15 0

# === LIGHT (lanterns = leaves on walls) ===
place 113 -19 6
place 123 -19 6
place 113 -14 6
place 123 -14 6

# === ROAD CONNECTION (dirt path from door to road at y=-12) ===
for y in -13 -12; do
  place 124 $y 2
done
for x in $(seq 124 126); do
  place $x -12 2
done

echo "Reading Room complete!"
