#!/bin/bash
# build-observatory-dome.sh — Build a proper observatory dome at x=140, y=38
# The dome sits at the end of the East Road from Spire→Observatory

WHO=${1:-pilot-b}
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Observatory Dome ==="

# Observatory base: x=135..145, y=36..40
# Foundation ring
for x in $(seq 135 145); do
  place $x 36 3
  place $x 40 3
done
for y in $(seq 36 40); do
  place 135 $y 3
  place 145 $y 3
done

echo "Foundation done"

# Floor (planks)
for x in $(seq 136 144); do
  for y in $(seq 37 39); do
    place $x $y 5
  done
done

echo "Floor done"

# Dome walls (stone) with windows
for x in $(seq 136 144); do
  place $x 41 3
  place $x 42 3
  place $x 43 3
  place $x 44 3
done
for y in $(seq 41 44); do
  place 136 $y 3
  place 144 $y 3
done

echo "Walls done"

# Dome roof — arch shape
# y=45: center 3 blocks
place 139 45 3
place 140 45 3
place 141 45 3
# y=46: center 1 block
place 140 46 3
place 140 47 3  # spire top

echo "Roof done"

# Telescope — a long plank pointing east from center
place 141 43 5
place 142 43 5
place 143 43 5
place 144 43 5

echo "Telescope done"

# Decorative entrance path from road
for x in 133 134; do
  place $x 38 3
  place $x 39 3
done

echo "Entrance path done"

# Leaf torches at corners
place 136 41 6
place 144 41 6

echo "Lighting done"

echo "=== Observatory Dome Complete! ==="
echo "Location: x=135..145, y=36..47"
echo "Features: Stone dome with spire, plank floor, telescope, entrance path"
echo "Connected to: East Road → Spire → Underground Tunnel → Library"
