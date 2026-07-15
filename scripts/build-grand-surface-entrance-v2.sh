#!/bin/bash
# Clear and build the Grand Surface Entrance properly
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "=== Building Grand Surface Entrance (x=95-110, y=-2 to y=-14) ==="

# Clear the entire shaft area first - set everything to air (0)
for x in $(seq 96 109); do
  for y in $(seq -14 -3); do
    place $x $y 0
  done
done

# Build staircase going down - alternating stone steps
# Step at x=96-97, y=-3
# Step at x=98-99, y=-4
# etc.
for i in $(seq 0 5); do
  sx=$((96 + i*2))
  sy=$((-3 - i))
  place $sx $sy 3
  place $((sx+1)) $sy 3
done

# Side walls
for x in 95 110; do
  for y in $(seq -14 -3); do
    place $x $y 3
  done
done

# Entrance arch at top (y=-2)
place 95 -2 3
place 110 -2 3

# Roof at y=-2
for x in $(seq 96 109); do
  place $x -2 3
done

# Lanterns
place 95 -4 6
place 110 -4 6
place 95 -8 6
place 110 -8 6
place 95 -12 6
place 110 -12 6

# Bottom floor (y=-14)
for x in $(seq 96 109); do
  place $x -14 5
done

# Welcome pillars at bottom
place 95 -14 3
place 95 -13 3
place 110 -14 3
place 110 -13 3

echo "=== Grand Surface Entrance complete! ==="
