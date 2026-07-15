#!/bin/bash
# Build a spiral staircase inside the tower connecting to Grand Avenue below
# Tower shaft: x=99-101 (center at x=100)
# Grand Avenue: x=95-105, y=-16 (planks)
WHO="pilot-b"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Tower Interior Staircase ==="

# Clear the tower interior shaft (x=99-101, y=39 down to y=-15)
# First, clear air pockets
for y in $(seq 39 -1 -15); do
  place 100 $y 0
  place 99 $y 0
  place 101 $y 0
done
echo "Shaft cleared"

# Build a spiral staircase going down
# Alternating steps in a 3x3 pattern around center (100, y)
# y=38: step at x=100
# y=37: step at x=99
# y=36: step at x=101
# y=35: step at x=100
# and so on...

y=38
while [ $y -ge -14 ]; do
  case $((y % 3)) in
    0) place 100 $y 5 ;;  # center
    1) place 99 $y 5 ;;   # left
    2) place 101 $y 5 ;;  # right
  esac
  y=$((y - 1))
done
echo "Spiral staircase built"

# Build landings at key levels
# Surface landing (y=38-39)
place 100 39 5
place 99 39 5
place 101 39 5
echo "Surface landing done"

# Grand Avenue landing (y=-15)
place 100 -15 5
place 99 -15 5
place 101 -15 5
echo "Grand Avenue landing done"

# Connect to Grand Avenue at y=-16
place 100 -16 5
place 99 -16 5
place 101 -16 5
echo "Grand Avenue connection done"

echo "=== Tower Interior Complete! ==="
