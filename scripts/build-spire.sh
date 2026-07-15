#!/bin/bash
# Spire Tower builder at x=100
# Run: bash /home/rishi/Work/pilot/scripts/build-spire.sh
WHO="pilot-a"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Spire Tower ==="

# Base platform (stone, 7x7)
for x in $(seq 96 104); do
  for y in 38 39; do
    place $x $y 3
  done
done
echo "Base done"

# Tower walls (stone)
for y in $(seq 40 65); do
  place 97 $y 3
  place 103 $y 3
  # Back wall
  place 98 $y 3
  place 99 $y 3
  place 100 $y 3
  place 101 $y 3
  place 102 $y 3
done
echo "Walls done"

# Windows (glass) at every 5 levels
for y in 45 50 55 60; do
  place 100 $y 6
done
echo "Windows done"

# Spire top (tapering)
place 100 66 3
place 99 66 3
place 101 66 3
place 100 67 3
place 100 68 3
echo "Spire top done"

# Door at base
place 100 40 0
echo "Door done"

# Tunnel access shaft (from y=38 down to y=-13)
for y in $(seq 37 -1 -12); do
  place 100 $y 0
  # Walls
  for dx in -1 1; do
    place $((100+dx)) $y 3
  done
done
echo "Shaft done"

# Floor at tunnel level
for x in $(seq 96 104); do
  place $x -13 3
  place $x -12 0
done
echo "Tunnel floor done"

echo "=== Spire Tower Complete! ==="
