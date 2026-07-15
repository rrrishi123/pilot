#!/bin/bash
# Grand Staircase from surface (y=38) down to Library (y=-25) at x=115
# Run: bash /home/rishi/Work/pilot/scripts/build-staircase.sh
WHO="pilot-a"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Grand Staircase ==="

# Staircase shaft (x=112-118, y=-13..38)
for x in $(seq 112 118); do
  for y in $(seq -12 37); do
    place $x $y 0
  done
done
echo "Shaft cleared"

# Staircase floor (stone) every 3 blocks alternating
y=37
while [ $y -gt -13 ]; do
  # Landing
  for x in 112 113 114 115 116 117 118; do
    place $x $y 3
  done
  y=$((y - 3))
done
echo "Landings done"

# Staircase walls (stone)
for y in $(seq -12 38); do
  place 111 $y 3
  place 118 $y 3
done
echo "Walls done"

# Glass ceiling at surface level
for x in $(seq 112 118); do
  place $x 38 6
done
echo "Glass ceiling done"

# Surface entrance arch
for x in $(seq 111 118); do
  place $x 39 3
done
place 114 39 0
place 115 39 0
echo "Entrance arch done"

# Connection to Library at bottom
for x in $(seq 112 118); do
  place $x -13 0
done
echo "Library connection done"

echo "=== Grand Staircase Complete! ==="
