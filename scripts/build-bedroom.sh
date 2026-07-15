#!/bin/bash
# Build Underground Bedroom & Inn at x=50-70, y=-31 to y=-36
# A cozy place below the Kitchen for gophers to rest
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Underground Bedroom & Inn..."

# === ROOM BOUNDARIES ===
# Floor (y=-36)
for x in $(seq 50 70); do
  place $x -36 3
done

# Ceiling (y=-31)
for x in $(seq 50 70); do
  place $x -31 3
done

# Walls
for y in $(seq -32 -1 -36); do
  place 49 $y 3
  place 71 $y 3
done
for x in $(seq 50 70); do
  place $x -37 3
done

# === STAIRCASE CONNECTION UP TO KITCHEN (y=-29) ===
# Spiral staircase at x=55
for y in -30 -31 -32 -33 -34 -35 -36; do
  place 55 $y 3
done

# === BEDROOMS (west side, x=50-60) ===
# Bed 1 (wood frame)
place 52 -33 4
place 53 -33 4

# Bed 2
place 56 -33 4
place 57 -33 4

# Bed 3
place 52 -35 4
place 53 -35 4

# Bed 4
place 56 -35 4
place 57 -35 4

# === COMMON AREA (east side, x=62-70) ===
# Central table (wood)
place 65 -34 4
place 66 -34 4

# Chairs around table
place 64 -34 4
place 67 -34 4
place 65 -35 4
place 66 -33 4

# === DECORATION ===
# Lanterns
place 51 -32 6
place 60 -32 6
place 69 -32 6

# Plants for cozy atmosphere
place 51 -34 1
place 69 -34 1
place 51 -36 1
place 69 -36 1

# Welcome mat (dirt) at entrance
place 55 -36 2

# Storage chests (diamond = storage)
place 63 -35 2
place 68 -35 2

echo "Bedroom & Inn complete!"
echo "Located at x=50-70, y=-31 to y=-36"
echo "Staircase at x=55 connects up to Kitchen"
