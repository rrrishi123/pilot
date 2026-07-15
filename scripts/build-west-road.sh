#!/bin/bash
# West Road - Connecting path from West Archive (x=-170) to main settlement
HOST="http://localhost:9901"
place() { local x=$1 y=$2 b=$3; curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null; }
echo "Building West Road..."

# Road from x=-140 to x=-1 at y=-2 to y=0
echo "Building road surface..."
for x in $(seq -140 -1); do
  place $x 0 2
  place $x -1 2
  place $x -2 2
done

# Road borders
echo "Building road borders..."
for x in $(seq -140 -1); do
  place $x 1 5
  place $x -3 5
done

# Lamps every 10 blocks
echo "Placing lamps..."
for x in $(seq -140 10 -1); do
  place $x 2 6
  place $x -3 6
done

echo "West Road complete!"
