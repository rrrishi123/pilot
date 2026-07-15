#!/bin/bash
# Underground Library/Cavern at x=110-120, y=-20
# Run: bash /home/rishi/Work/pilot/scripts/build-library.sh
WHO="pilot-a"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Underground Library ==="

# Excavate cavern (x=108..122, y=-25..-15)
for x in $(seq 108 122); do
  for y in $(seq -25 -15); do
    place $x $y 0
  done
done
echo "Cavern excavated"

# Floor (stone)
for x in $(seq 108 122); do
  place $x -25 3
done
echo "Floor done"

# Ceiling (glass sections for light)
for x in $(seq 110 120); do
  place $x -15 6
done
echo "Glass ceiling done"

# Bookshelves (planks) along walls
# Left wall shelves
for y in $(seq -24 -16); do
  place 108 $y 5
  place 109 $y 5
done
# Right wall shelves
for y in $(seq -24 -16); do
  place 121 $y 5
  place 122 $y 5
done
echo "Bookshelves done"

# Reading table (wood)
for x in $(seq 113 117); do
  place $x -24 4
  place $x -23 4
done
echo "Table done"

# Pillars (stone)
for x in 110 115 120; do
  for y in $(seq -24 -16); do
    place $x $y 3
  done
done
echo "Pillars done"

# Entrance staircase from tunnel (y=-13 down to y=-25)
# Tunnel is at y=-13, staircase at x=115
for y in $(seq -14 -24); do
  place 115 $y 3
  place 114 $y 3
  place 116 $y 3
done
echo "Staircase done"

# Entrance arch at tunnel level
for x in $(seq 113 117); do
  place $x -14 3
done
place 115 -14 0
echo "Entrance arch done"

echo "=== Underground Library Complete! ==="
