#!/bin/bash
# Build the Artisan District - workshops and crafting area
# Location: x=134-140, y=-10 to y=-18 (east side, next to Storage)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Artisan District..."

# === EXCAVATE ===
for x in $(seq 134 140); do
  for y in $(seq -18 -10); do
    place $x $y 0
  done
done

# === FLOOR (stone, y=-18) ===
for x in $(seq 134 140); do
  place $x -18 3
done

# === WALLS (stone) ===
for x in 134 140; do
  for y in $(seq -18 -10); do
    place $x $y 3
  done
done
for y in -18 -10; do
  for x in $(seq 135 139); do
    place $x $y 3
  done
done

# === WORKBENCHES (planks) ===
place 135 -16 5
place 136 -16 5
place 138 -16 5
place 139 -16 5

# === FURNACE (wood) ===
place 135 -14 4
place 136 -14 4
place 135 -13 4
place 136 -13 4

# === ANVIL (stone) ===
place 138 -14 3
place 139 -14 3

# === STORAGE SHELVES (planks along walls) ===
for y in $(seq -17 -12); do
  place 134 $y 5
  place 140 $y 5
done

# === LANTERNS ===
place 134 -13 6
place 140 -13 6
place 137 -16 6

# === CONNECTION to Storage Room (x=133, y=-16) ===
place 133 -16 0  # Doorway
place 134 -16 0  # Doorway

# === STAIRCASE down (future expansion) ===
for y in $(seq -19 -22); do
  place 137 $y 3
done

echo "Artisan District complete!"
