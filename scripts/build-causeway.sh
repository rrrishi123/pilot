#!/bin/bash
# Gopher Causeway - Connecting path from Assembly Hall (x=160, y=-40) to main area
HOST="http://localhost:9901"
place() { local x=$1 y=$2 b=$3; curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null; }
echo "Building Gopher Causeway..."

# Path from Assembly Hall entrance (x=160, y=-40) going east and then up
# Step 1: Path going east from x=160 to x=180 at y=-40
echo "Building east-west corridor..."
for x in $(seq 161 179); do
  place $x -40 2
  place $x -41 2
done

# Step 2: Path turning north from y=-39 to y=-20 at x=180
echo "Building north-south corridor..."
for y in $(seq -39 -20); do
  place 180 $y 2
  place 181 $y 2
done

# Step 3: East-west path at y=-20 from x=180 to x=200
echo "Building upper corridor..."
for x in $(seq 182 200); do
  place $x -20 2
  place $x -21 2
done

# Step 4: Staircase markers (decorative)
echo "Adding decorative markers..."
for x in 160 165 170 175 180 185 190 195 200; do
  place $x -42 6
  place $x -19 6
done

# Step 5: Path walls (low)
echo "Adding path borders..."
for x in $(seq 160 180); do
  place $x -43 5
  place $x -38 5
done

echo "Gopher Causeway complete!"
