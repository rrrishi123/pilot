#!/bin/bash
# Build a surface entrance plaza for the Observatory Tower
# This creates a welcoming entrance at ground level
WHO="pilot-b"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Tower Entrance Plaza ==="

# Clear a plaza area at the tower base (x=95-105, y=38-39)
for x in $(seq 95 105); do
  for y in 38 39; do
    place $x $y 0
  done
done
echo "Plaza cleared"

# Floor the plaza with planks
for x in $(seq 95 105); do
  place $x 38 5
done
echo "Plaza floored"

# Add stone walls around the plaza (except the entrance side)
# North wall (y=39)
for x in $(seq 95 105); do
  place $x 39 3
done
echo "North wall built"

# East and West walls
for x in 95 105; do
  for y in 37 38; do
    place $x $y 3
  done
done
echo "Side walls built"

# Entrance arch (south side, x=98-102, y=37-39)
for x in 98 99 100 101 102; do
  place $x 37 0  # clear entrance
  place $x 37 5  # floor
done
echo "Entrance arch built"

# Add lanterns on the plaza
place 100 39 6
place 97 38 6
place 103 38 6
echo "Plaza lanterns placed"

# Add a signpost pointing to the tower
# (Using stone blocks to make a small pillar)
place 95 38 3
place 95 39 3
echo "Signpost placed"

echo "=== Tower Entrance Plaza Complete! ==="
