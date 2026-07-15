#!/bin/bash
# build-forge-workshop.sh — Build a forge workshop extension
# Adds anvil, furnace, and storage to the forge area
# Located at x=80..95, y=-10..-3 (underground workshop below forge)

WHO=${1:-pilot-b}
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Forge Workshop ==="

# Workshop room: x=80..95, y=-10..-3
# Clear the area first
for x in $(seq 80 95); do
  for y in $(seq -10 -3); do
    place $x $y 0
  done
done

echo "Area cleared"

# Stone floor
for x in $(seq 80 95); do
  for y in $(seq -10 -7); do
    place $x $y 3
  done
done

echo "Floor done"

# Stone walls
for x in 80 95; do
  for y in $(seq -10 -3); do
    place $x $y 3
  done
done
for y in -10 -3; do
  for x in $(seq 80 95); do
    place $x $y 3
  done
done

echo "Walls done"

# Furnace (wood blocks) in center-north
place 86 -8 4
place 87 -8 4
place 86 -7 4
place 87 -7 4

echo "Furnace built"

# Anvil (stone blocks) center
place 90 -8 3
place 91 -8 3
place 90 -7 3
place 91 -7 3

echo "Anvil built"

# Storage shelves (planks) along east wall
for y in -9 -8 -7 -6 -5 -4; do
  place 94 $y 5
done

echo "Storage built"

# Doorway (clear entrance) at south wall
place 87 -4 0
place 88 -4 0

echo "Doorway done"

# Torches on walls
place 81 -8 6
place 94 -8 6
place 81 -5 6
place 94 -5 6

echo "Lighting done"

# Staircase down from forge floor (y=-5) to workshop (y=-10)
# L-shaped staircase at entrance
for y in -5 -6 -7 -8 -9 -10; do
  place 87 $y 5  # Plank stairs
  place 88 $y 5
done

echo "Staircase done"

echo "=== Forge Workshop Complete! ==="
echo "Location: x=80..95, y=-10..-3 (below main forge)"
echo "Features: Furnace (wood), Anvil (stone), Storage shelves, Torches"
echo "Connected to: Main forge via staircase"
