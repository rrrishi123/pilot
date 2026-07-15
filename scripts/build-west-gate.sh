#!/bin/bash
# build-west-gate.sh — Build a western gate and connect the west side
# Creates a gate through the stone wall at x=-30, y=38
# and extends the road west

WHO=${1:-pilot-b}
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building West Gate ==="

# Gate through the stone wall at x=-30..-28, y=37..39
# Clear passage: 3 wide, 3 tall
place -30 37 0
place -30 38 0
place -30 39 0
place -29 37 0
place -29 38 0
place -29 39 0
place -28 37 0
place -28 38 0
place -28 39 0

echo "Gate cleared"

# Gate pillars
place -31 37 3
place -31 38 3
place -31 39 3
place -31 40 3
place -27 37 3
place -27 38 3
place -27 39 3
place -27 40 3

# Gate arch top
place -30 40 3
place -29 40 3
place -28 40 3

echo "Gate pillars done"

# Road west from gate (x=-40..-31, y=38)
for x in $(seq -40 -31); do
  place $x 38 3
  place $x 39 3
done

echo "West road done"

# Road east from gate to Spire (x=-27..92, y=38)
for x in $(seq -27 92); do
  place $x 38 3
  place $x 39 3
done

echo "East road connected"

# Lantern posts along the road every 10 blocks
for x in $(seq -30 10 90); do
  place $x 40 6  # Leaf torch on post
done

echo "Road lighting done"

echo "=== West Gate Complete! ==="
echo "Connects: West territory → Main Road → Spire → Observatory"
echo "Features: Stone gate with pillars, arch, continuous road, lanterns"
