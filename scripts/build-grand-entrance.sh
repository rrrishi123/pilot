#!/bin/bash
# Build Grand Surface Entrance at the Tower area (x=95-105)
# Connecting surface (y=-2) to underground (y=-16 Grand Avenue)
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Grand Surface Entrance..."

# === GRAND STAIRCASE (x=100-102, y=-3 to y=-16) ===
for y in $(seq -3 -1 -16); do
  place 100 $y 3
  place 101 $y 3
  place 102 $y 3
done

# === ENTRANCE ARCH (surface level, y=-3) ===
# Pillars
for y in -3 -4 -5; do
  place 98 $y 3
  place 104 $y 3
done
# Arch top
for x in $(seq 98 104); do
  place $x -5 3
done

# === ENTRANCE PATH (surface) ===
for x in $(seq 95 107); do
  place $x -3 3
done

# === WALLS along staircase ===
for y in $(seq -4 -1 -15); do
  place 99 $y 3
  place 103 $y 3
done

# === LANTERNS along staircase ===
place 99 -5 6
place 103 -5 6
place 99 -9 6
place 103 -9 6
place 99 -13 6
place 103 -13 6

# === GRAND ENTRANCE GATE at surface ===
place 98 -3 4
place 104 -3 4

# === DECORATIVE BANNERS (wood markers) ===
place 97 -4 4
place 105 -4 4

echo "Grand Surface Entrance complete!"
echo "Connects surface (y=-3) to Grand Avenue (y=-16) at x=100-102"
