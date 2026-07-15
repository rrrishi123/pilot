#!/bin/bash
# North Road - Connects Council Chamber (y=50) to main settlement (y=40)
HOST="http://localhost:9901"
place() { local x=$1 y=$2 b=$3; curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null; }
echo "Building North Road..."

# Vertical road from y=41 to y=49 at x=18-22 (aligns with council entrance)
echo "Building road surface..."
for y in $(seq 41 49); do
  for x in 18 19 20 21 22; do
    place $x $y 2
  done
done

# Road borders
echo "Building road borders..."
for y in $(seq 41 49); do
  place 17 $y 5
  place 23 $y 5
done

# Lamps
echo "Placing lamps..."
for y in 42 45 48; do
  place 18 $y 6
  place 22 $y 6
done

echo "North Road complete!"
