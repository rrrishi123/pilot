#!/bin/bash
# Build Storage Room - for resources and materials
# Location: x=125-133, y=-14 to y=-20 (right of Reading Room)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Storage Room..."

# === FLOOR (stone, y=-16) ===
for x in $(seq 125 133); do
  place $x -16 3
done

# === WALLS (stone) ===
# Left wall (x=125) - shared with Reading Room
for y in $(seq -20 -13); do
  place 125 $y 3
done
# Right wall (x=133)
for y in $(seq -20 -13); do
  place 133 $y 3
done
# Back wall (y=-20)
for x in $(seq 126 132); do
  place $x -20 3
done
# Front wall (y=-13) - with door opening
for x in $(seq 126 132); do
  place $x -13 3
done

# === CEILING (stone, y=-11) ===
for x in $(seq 125 133); do
  place $x -11 3
done

# === DOORWAY (x=126, y=-14,-15) - opening to Reading Room ===
place 126 -14 0
place 126 -15 0

# === SHELVES (planks along walls) ===
# Left wall shelves
place 125 -17 5
place 125 -18 5
place 125 -19 5
# Back wall shelves
place 127 -19 5
place 129 -19 5
place 131 -19 5
# Right wall shelves
place 133 -17 5
place 133 -18 5
place 133 -19 5

# === CRATES (wood blocks on floor) ===
place 128 -16 4
place 129 -16 4
place 131 -16 4
place 132 -16 4

# === BARRELS (dirt blocks = barrels) ===
place 127 -16 2
place 130 -16 2

# === LANTERNS ===
place 126 -12 6
place 130 -12 6
place 133 -12 6

echo "Storage Room complete!"
