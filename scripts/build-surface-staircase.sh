#!/bin/bash
# Build a staircase from the surface down to the Council Chamber
# Location: x=40-54, y=0 to y=-6
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "=== Building Surface Entrance Staircase (x=40-54, y=0 to y=-6) ==="

# Clear a vertical shaft
for x in $(seq 41 53); do
  for y in $(seq -5 -1); do
    place $x $y 0
  done
done

# Staircase going down - alternating steps
# y=0 (surface entrance) -> y=-1 -> y=-2 -> ... -> y=-6 (Council Chamber roof)
# Steps at x=44-50 for the staircase

# Entrance opening at surface (y=0)
for x in $(seq 44 50); do
  place $x 0 0
done

# Staircase: zigzag pattern
# Step 1: x=44-45, y=-1
place 44 -1 5
place 45 -1 5
# Step 2: x=46-47, y=-2
place 46 -2 5
place 47 -2 5
# Step 3: x=48-49, y=-3
place 48 -3 5
place 49 -3 5
# Step 4: x=44-45, y=-4
place 44 -4 5
place 45 -4 5
# Step 5: x=46-47, y=-5
place 46 -5 5
place 47 -5 5

# Side walls
for x in 41 42 43 51 52 53; do
  for y in $(seq -5 -1); do
    place $x $y 3
  done
done

# Entrance arch (stone blocks around the opening at y=0)
place 43 0 3
place 51 0 3
place 44 0 5
place 45 0 5
place 46 0 5
place 47 0 5
place 48 0 5
place 49 0 5
place 50 0 5

# Lanterns in the staircase
place 43 -2 6
place 51 -2 6
place 43 -4 6
place 51 -4 6

echo "=== Surface Entrance Staircase complete! ==="
