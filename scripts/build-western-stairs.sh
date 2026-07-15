#!/bin/bash
# Build staircase connecting Western Gardens to upper levels
# Location: x=-52 to x=-51, y=-15 to y=-29
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Western Garden Staircase..."

# Stone staircase going down from y=-15 to y=-29
for y in $(seq -15 -29); do
  place -52 $y 3
  place -51 $y 3
done

# Add wood railings on the sides
for y in $(seq -15 -29); do
  place -53 $y 4
  place -50 $y 4
done

# Add lanterns at key levels
place -53 -16 6
place -50 -16 6
place -53 -22 6
place -50 -22 6
place -53 -28 6
place -50 -28 6

echo "Western Garden Staircase complete!"
