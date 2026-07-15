#!/bin/bash
# build-underground-tunnel.sh — Build underground tunnel connecting forge to library
# Tunnel at y=-15, from x=40 (under library) to x=88 (under forge workshop)

WHO=${1:-pilot-b}
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Underground Tunnel ==="

# Tunnel runs from x=40 to x=88 at y=-15
# 3-wide corridor: x-1, x, x+1

# Clear and build tunnel floor, walls, ceiling
for x in $(seq 40 88); do
  # Floor
  place $x -16 3
  # Ceiling
  place $x -14 3
  # Walls at edges
  place $x -15 3
done

echo "Tunnel shell done"

# Entrances
# Library entrance: vertical shaft from y=38 down to y=-15
for y in $(seq -15 38); do
  place 40 $y 0  # Clear shaft
  place 41 $y 0
done
# Ladder on shaft walls
for y in $(seq -14 37); do
  place 40 $y 5  # Plank ladder
  place 41 $y 5
done

echo "Library entrance shaft done"

# Forge entrance: vertical shaft from y=-5 (forge floor) down to y=-15
for y in $(seq -15 -5); do
  place 88 $y 0
  place 89 $y 0
done
for y in $(seq -14 -6); do
  place 88 $y 5
  place 89 $y 5
done

echo "Forge entrance shaft done"

# Torches every 8 blocks in tunnel
for x in $(seq 44 8 84); do
  place $x -14 6
done

echo "Lighting done"

# Reinforced floor with planks
for x in $(seq 41 87); do
  place $x -15 5
done

echo "Floor reinforced"

echo "=== Underground Tunnel Complete! ==="
echo "Connects: Library (x=40) ↔ Forge (x=88)"
echo "Depth: y=-15 (3 blocks below forge workshop)"
echo "Features: Plank floor, stone ceiling/walls, torches, ladder shafts"
