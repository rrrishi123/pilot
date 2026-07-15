#!/bin/bash
# build-grand-staircase.sh — Grand staircase from underground to surface
# Connects the underground tunnel (y=-13) at x=70 to the surface (y=38) at x=70
# A wide, elegant staircase with landings

WHO=${1:-pilot-b}
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Grand Staircase ==="

# Staircase at x=70 (center of the library/tunnel junction)
# Goes from y=-13 (tunnel floor) to y=38 (surface observatory level)
# With landings every 10 steps

# Step 1: Staircase shaft (5-wide)
for y in $(seq -13 38); do
  # Back wall
  place 68 $y 3
  place 72 $y 3
  # Floor steps (alternating)
  if [ $(( (y + 13) % 3 )) -eq 0 ]; then
    place 69 $y 5
    place 70 $y 5
    place 71 $y 5
  else
    place 69 $y 3
    place 70 $y 3
    place 71 $y 3
  fi
done

echo "Staircase steps done"

# Landings every 10 steps
for y in -3 7 17 27; do
  for x in 68 69 70 71 72; do
    place $x $y 5  # Plank landing
  done
  # Landing walls
  place 67 $y 3
  place 73 $y 3
done

echo "Landings done"

# Staircase entrance arch at surface (y=38)
for x in 67 68 69 70 71 72 73; do
  place $x 39 3  # Arch top
done
place 67 38 3
place 73 38 3

echo "Entrance arch done"

# Torches along the staircase
for y in -10 0 10 20 30; do
  place 68 $y 6  # Left torch
  place 72 $y 6  # Right torch
done

echo "Lighting done"

echo "=== Grand Staircase Complete! ==="
echo "Connects: Underground Tunnel (y=-13) → Surface (y=38)"
echo "Features: 5-wide, plank steps, stone walls, landings, torches"
