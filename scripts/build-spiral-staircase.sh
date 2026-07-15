#!/bin/bash
# Build a proper spiral staircase connecting the upper forge (y=-20) to the deep forge (y=-30)
# Spiral around x=113, y=-20 to y=-30
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Spiral Staircase to Deep Forge..."

# Clear the shaft area first (place air/0 blocks)
for y in $(seq -20 -30); do
  for x in 113 114 115; do
    place $x $y 0
  done
done

# Spiral staircase steps going down
# Each step: 2 blocks wide, alternating sides of the shaft

# Level -20 (landing from forge)
place 113 -20 5
place 114 -20 5

# Level -21 (step down, right side)
place 114 -21 5

# Level -22 (step down, left side)
place 113 -22 5

# Level -23 (step down, right side)
place 114 -23 5

# Level -24 (step down, left side)
place 113 -24 5

# Level -25 (step down, right side)
place 114 -25 5

# Level -26 (step down, left side)
place 113 -26 5

# Level -27 (step down, right side)
place 114 -27 5

# Level -28 (step down, left side)
place 113 -28 5

# Level -29 (step down, right side)
place 114 -29 5

# Level -30 (landing at deep forge floor)
place 113 -30 5
place 114 -30 5

# === SHAFT WALLS ===
for y in $(seq -20 -30); do
  place 112 $y 3  # left wall
  place 115 $y 3  # right wall
done

# Back wall
for y in $(seq -20 -30); do
  place 116 $y 3
done

# === LIGHTING in shaft ===
for y in -21 -24 -27; do
  place 113 $y 6
done

echo "Spiral Staircase complete!"
