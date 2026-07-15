#!/bin/bash
# Build a connecting path from the Western structure to the Grand Avenue
# This connects the western staircase (x=-52) to the existing path network
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Western Connection Path..."

# Connect from western staircase (x=-52, y=-15) to the existing path at x=-40
# Build a stone path at y=-15 level
for x in $(seq -52 -40); do
  place $x -15 3
done

# Add lanterns along the path
place -50 -15 6
place -45 -15 6

# Build a staircase down from the path to the gardens at y=-29
# at x=-45 to x=-44
for y in $(seq -16 -29); do
  place -45 $y 3
  place -44 $y 3
done

# Add railings
for y in $(seq -16 -29); do
  place -46 $y 4
  place -43 $y 4
done

# Lanterns on staircase
place -46 -20 6
place -43 -20 6
place -46 -28 6
place -43 -28 6

echo "Western Connection Path complete!"
