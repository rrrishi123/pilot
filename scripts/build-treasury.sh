#!/bin/bash
# Build the Gopher Treasury - secure vault for valuable resources
# Location: x=126-133, y=-22 to y=-28 (right of Library, below Storage)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Treasury..."

# === EXCAVATE ===
for x in $(seq 126 133); do
  for y in $(seq -28 -21); do
    place $x $y 0
  done
done

# === DOUBLE-THICK STONE FLOOR (y=-28) ===
for x in $(seq 126 133); do
  place $x -28 3
  place $x -29 3
done

# === DOUBLE-THICK STONE WALLS ===
for x in 126 133; do
  for y in $(seq -28 -21); do
    place $x $y 3
    place $x $y 3  # double layer
  done
done
for y in -28 -21; do
  for x in $(seq 127 132); do
    place $x $y 3
    place $x $y 3
  done
done

# === CEILING (stone, y=-20) ===
for x in $(seq 126 133); do
  place $x -20 3
done

# === VAULT DOOR (iron - represented by stone, only one block wide) ===
place 126 -22 0
place 126 -23 0

# === TREASURE PILES (planks = chests) ===
place 127 -27 5
place 128 -27 5
place 129 -27 5
place 130 -27 5
place 131 -27 5

# === GOLD (leaves = gold piles) ===
place 127 -28 6
place 129 -28 6
place 131 -28 6

# === GUARD POST (wood) ===
place 126 -26 4
place 126 -27 4

# === TORCHES ===
place 126 -24 6
place 133 -24 6
place 133 -27 6

# === CONNECTION to Library ===
# Tunnel from Library (x=124, y=-22) to Treasury (x=126, y=-22)
place 124 -22 0
place 125 -22 0
place 125 -23 3
place 125 -24 3

echo "Treasury complete!"
