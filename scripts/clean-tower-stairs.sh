#!/bin/bash
# Clean up the tower staircase - ensure a clear path from top to bottom
WHO="pilot-b"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Cleaning Tower Staircase ==="

# First, clear all blocks in the shaft (x=99-101, y=-15 to y=38)
for y in $(seq 38 -1 -15); do
  for x in 99 100 101; do
    place $x $y 0
  done
done
echo "Shaft cleared"

# Now rebuild: alternating steps
# Step goes at x=100, with rails at x=99 and x=101
for y in $(seq 38 -1 -15); do
  # Step at center
  place 100 $y 5
  # Rails
  place 99 $y 4
  place 101 $y 4
done
echo "Steps and rails rebuilt"

# Add lanterns at every 5th level (replacing center step)
for y in 35 30 25 20 15 10 5 0 -5 -10; do
  place 100 $y 6
done
echo "Lanterns placed"

# Grand Avenue connection
place 100 -16 5
place 99 -16 5
place 101 -16 5
echo "Grand Avenue connected"

# Add entrance arch at surface level (y=38-39)
for x in 99 100 101; do
  place $x 39 3  # stone arch top
done
echo "Surface entrance arch added"

echo "=== Tower Staircase Cleaned! ==="
