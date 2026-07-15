#!/bin/bash
# Fix the observatory walls - use correct block types
# Block types: 0=AIR, 1=GRASS, 2=DIRT, 3=STONE, 4=WOOD, 5=PLANK, 6=LEAF
WHO="pilot-b"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Fixing Observatory Walls ==="

# North wall at y=35 - should be STONE (3)
for x in $(seq 125 138); do
  place $x 35 3
done

# South wall at y=44 - should be STONE (3), with doorway
for x in $(seq 125 138); do
  if [ "$x" = "131" ] || [ "$x" = "132" ]; then
    place $x 44 0  # doorway
  else
    place $x 44 3
  fi
done

# West wall at x=125, y=35..44
for y in $(seq 35 44); do
  place 125 $y 3
done

# East wall at x=138, y=35..44
for y in $(seq 35 44); do
  place 138 $y 3
done

# Remove the garden blocks that were placed with wrong type
# (Those were meant to be dirt=2 and stone path=3)
for x in $(seq 126 137); do
  for y in $(seq 45 48); do
    place $x $y 0  # clear first
  done
done

echo "=== Walls Fixed! ==="
