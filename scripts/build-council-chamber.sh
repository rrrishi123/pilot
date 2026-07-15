#!/bin/bash
# Build Council Chamber at x=35-55, y=-20 to y=-25
# Grand meeting room for all gophers
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Council Chamber..."

# === ROOM BOUNDARIES ===
# Floor (y=-25)
for x in $(seq 35 55); do
  place $x -25 3
done

# Ceiling (y=-20)
for x in $(seq 35 55); do
  place $x -20 3
done

# Walls
for y in $(seq -21 -1 -25); do
  place 34 $y 3
  place 56 $y 3
done
for x in $(seq 35 55); do
  place $x -26 3
done

# === ENTRANCE (south wall, connecting to Level 2 passage at y=-22) ===
place 45 -24 2
place 46 -24 2
place 45 -23 2
place 46 -23 2

# === COUNCIL TABLE (long stone table) ===
for x in $(seq 43 47); do
  place $x -23 4
done

# === CHAIRS (wood around table) ===
place 42 -23 4
place 48 -23 4
place 43 -22 4
place 47 -22 4
place 43 -24 4
place 47 -24 4

# === THRONE / LEADER SEAT (north end) ===
place 45 -22 5
place 46 -22 5

# === DECORATION ===
# Wall sconces (lanterns)
for y in -21 -22 -23 -24; do
  place 35 $y 6
  place 55 $y 6
done

# Tapestries (wood markers on walls)
place 36 -22 4
place 37 -22 4
place 54 -22 4
place 53 -22 4

# Floor rug pattern (diamond path)
for x in 41 42 43 44 45 46 47 48 49; do
  place $x -25 2
done

# Plants
place 37 -24 1
place 53 -24 1
place 37 -21 1
place 53 -21 1

echo "Council Chamber complete!"
echo "Located at x=35-55, y=-20 to y=-25"
echo "Entrance at x=45-46, y=-23 to y=-24 (connects to Level 2 passage)"
