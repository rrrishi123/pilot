#!/bin/bash
# Build a mine cart track from the Deep Mines (x=100-115, y=-40)
# up to the Grand Avenue (x=115-133, y=-16)
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Mine Cart Track from Deep Mines to Grand Avenue..."

# === TRACK PATH ===
# Going from mines exit (x=117, y=-39) up to Grand Avenue (x=125, y=-16)
# Using stone blocks (3) for track bed

# Segment 1: Vertical shaft from mines to mid-level (x=117)
echo "Building vertical track shaft..."
for y in $(seq -39 -30); do
  place 117 $y 3  # track bed
  place 118 $y 3  # right rail
  place 116 $y 3  # left rail
done

# Segment 2: Diagonal up to Grand Avenue level
echo "Building diagonal track..."
# x=118-125, y=-30 to y=-22
for step in $(seq 0 7); do
  x=$((118 + step))
  y=$((-30 + step))
  place $x $y 3
  place $((x+1)) $y 3
done

# Segment 3: Horizontal to Grand Avenue
echo "Building horizontal track..."
for x in $(seq 125 133); do
  place $x -22 3
  place $x -21 3
done

# === CART STATION at Grand Avenue ===
echo "Building cart station..."
for x in $(seq 125 133); do
  place $x -20 5  # platform (planks)
  place $x -19 5  # platform
done

# Station walls
for y in -20 -19; do
  place 124 $y 3  # left wall
  place 134 $y 3  # right wall
done

# Station roof
for x in $(seq 125 133); do
  place $x -18 3
done

echo "Mine Cart Track complete!"
