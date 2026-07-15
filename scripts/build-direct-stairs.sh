#!/bin/bash
# Build a DIRECT vertical staircase inside the Observatory Tower
# The tower shaft is at x=99-101 (3 wide), going from y=39 (surface) to y=-16 (Grand Avenue)
# We'll build alternating steps going down
WHO="pilot-b"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Direct Staircase ==="

# Strategy: Build a straight staircase going down at x=100
# Each step: place a plank at (100, y), clear the space above it
# This creates a ladder-like descent

for y in $(seq 38 -1 -15); do
  # Clear the step area
  place 100 $y 0
  # Place the step
  place 100 $y 5
done

echo "Main staircase built (y=38 to y=-15)"

# Add side rails for safety (wood blocks)
for y in $(seq 38 -1 -15); do
  place 99 $y 4
  place 101 $y 4
done
echo "Side rails added"

# Add lanterns every 5 blocks
for y in 38 33 28 23 18 13 8 3 -2 -7 -12; do
  place 100 $y 6
done
echo "Lanterns placed"

# Clear the observation deck floor access
place 100 39 0
place 100 39 5
echo "Deck access cleared"

# Connect to Grand Avenue
place 100 -16 5
place 99 -16 5
place 101 -16 5
echo "Grand Avenue connected"

echo "=== Direct Staircase Complete! ==="
