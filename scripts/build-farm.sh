#!/bin/bash
# Farm builder - run with: bash /home/rishi/Work/pilot/scripts/build-farm.sh
WHO="pilot-a"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Farm ==="

# Farm field (grass)
for x in $(seq 60 90); do
  place $x 40 2
  place $x 41 2
done
echo "Field done"

# Fence (wood)
for x in $(seq 58 92); do
  place $x 42 4
done
echo "Fence done"

# Fence corners
for x in 58 92; do
  for y in 39 40 41; do
    place $x $y 4
  done
done
echo "Corners done"

# Farmhouse walls (planks)
for x in 70 71 72 73 74 75 76 77 78; do
  for y in 43 44 45; do
    place $x $y 5
  done
done
echo "Walls done"

# Roof (wood)
for x in $(seq 68 80); do
  place $x 46 4
done
echo "Roof done"

# Door (air)
place 73 43 0
place 73 44 0
echo "Door done"

# Path (dirt)
for x in $(seq 73 84); do
  place $x 42 1
done
echo "Path done"

echo "=== Farm Complete ==="
