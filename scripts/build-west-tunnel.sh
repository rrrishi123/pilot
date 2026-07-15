#!/bin/bash
# Westward Tunnel Extension from x=0 to x=-30
# Run: bash /home/rishi/Work/pilot/scripts/build-west-tunnel.sh
WHO="pilot-a"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building West Tunnel ==="

# Tunnel passage: floor at y=-13, walls at y=-14..-9, ceiling at y=-8
for x in $(seq -30 0); do
  # Floor (stone)
  place $x -13 3
  # Ceiling (stone)
  place $x -8 3
  # Walls (stone)
  place $x -9 3
  place $x -10 3
  place $x -11 3
  place $x -12 3
  # Clear interior (air)
  place $x -9 0
  place $x -10 0
  place $x -11 0
  place $x -12 0
done
echo "Tunnel passage done"

# West Entrance at x=-30
for y in $(seq -13 -8); do
  place -30 $y 3
done
place -30 -10 0  # Doorway
echo "West entrance done"

# Connection to existing tunnel at x=0
for y in $(seq -13 -8); do
  place 0 $y 0
done
echo "Connection done"

# West gatehouse (small stone structure above tunnel entrance)
for x in $(seq -32 -28); do
  for y in $(seq -7 -5); do
    place $x $y 3
  done
done
# Gatehouse interior
for x in $(seq -31 -29); do
  for y in $(seq -7 -5); do
    place $x $y 0
  done
done
echo "West gatehouse done"

echo "=== West Tunnel Complete! ==="
