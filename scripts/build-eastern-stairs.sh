#!/bin/bash
# Build staircase connecting Eastern Gardens to upper levels
# Location: x=193-194, y=-15 to y=-29
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Eastern Garden Staircase..."

# Stone staircase going down from y=-15 to y=-29
for y in $(seq -15 -29); do
  place 193 $y 3
  place 194 $y 3
done

# Add wood railings on the sides
for y in $(seq -15 -29); do
  place 192 $y 4
  place 195 $y 4
done

# Add lanterns at key levels
place 192 -16 6
place 195 -16 6
place 192 -22 6
place 195 -22 6
place 192 -28 6
place 195 -28 6

echo "Eastern Garden Staircase complete!"
