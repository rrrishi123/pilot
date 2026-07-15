#!/bin/bash
# Build Underground Greenhouse & Gardens at x=73-85, y=-25 to y=-30
# Fresh food for the Kitchen & Dining Hall!
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Underground Greenhouse & Gardens..."

# === ROOM BOUNDARIES ===
# Floor (y=-30)
for x in $(seq 73 85); do
  place $x -30 3
done

# Ceiling (y=-25)
for x in $(seq 73 85); do
  place $x -25 3
done

# Walls
for y in $(seq -26 -1 -30); do
  place 72 $y 3
  place 86 $y 3
done
for x in $(seq 73 85); do
  place $x -31 3
done

# === CONNECTION to Kitchen (tunnel at x=71, y=-27) ===
place 71 -27 2
place 71 -28 2
place 71 -29 2
place 72 -28 2

# === GARDEN BEDS ===
# Bed 1 (grass with dirt border)
for x in 75 76 77; do
  for y in -27 -28; do
    place $x $y 1
  done
done

# Bed 2
for x in 80 81 82; do
  for y in -27 -28; do
    place $x $y 1
  done
done

# Bed 3 (mushroom garden - darker, wood border)
for x in 75 76 77; do
  place $x -29 2
done

# === WALKING PATH (stone between beds) ===
for y in -27 -28 -29; do
  place 78 $y 3
  place 79 $y 3
done

# === WATER FEATURE (pond) ===
for x in 83 84; do
  for y in -29 -30; do
    place $x $y 2
  done
done

# === DECORATION ===
# Lanterns for plant growth
place 74 -26 6
place 79 -26 6
place 84 -26 6

# Trellis (wood) on walls
place 73 -27 4
place 73 -28 4
place 85 -27 4
place 85 -28 4

# Compost bin (dirt)
place 84 -27 2

echo "Greenhouse & Gardens complete!"
echo "Located at x=73-85, y=-25 to y=-30"
echo "Connected to Kitchen via tunnel at x=71"
