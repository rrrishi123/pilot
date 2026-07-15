#!/bin/bash
# Build the Grand Hall Entrance - surface-level welcome area
# Location: x=108-115, y=0-6 (above the Reading Room)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Grand Hall Entrance..."

# === FLOOR (stone, y=0) ===
for x in $(seq 108 115); do
  place $x 0 3
done

# === WALLS (stone) ===
# Left wall
for y in $(seq 1 5); do
  place 108 $y 3
done
# Right wall
for y in $(seq 1 5); do
  place 115 $y 3
done
# Back wall (y=5)
for x in $(seq 109 114); do
  place $x 5 3
done

# === ENTRANCE ARCH (front, y=0) ===
place 111 0 0  # Doorway
place 112 0 0  # Doorway

# === ROOF (planks, y=6) ===
for x in $(seq 108 115); do
  place $x 6 5
done

# === PILLARS (stone) ===
for y in $(seq 1 4); do
  place 110 $y 3
  place 113 $y 3
done

# === WELCOME SIGN (planks on back wall) ===
place 111 4 5
place 112 4 5

# === TORCHES ===
place 108 2 6
place 115 2 6

# === STAIRCASE DOWN to Bridge (x=112, y=-1 to y=-4) ===
for y in $(seq -1 -4); do
  place 112 $y 3
  place 113 $y 3
done

echo "Grand Hall Entrance complete!"
