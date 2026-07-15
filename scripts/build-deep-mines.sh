#!/bin/bash
# Build the Deep Mines - a mining operation at y=-40 to y=-45
# Connected via spiral staircase from Forge basement (x=108, y=-20 down to y=-40)
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Deep Mines..."

# === SPIRAL STAIRCASE from Forge basement down to Deep Mines ===
# Starting at x=108, y=-30 (below forge) spiraling down to y=-40
echo "Building spiral staircase..."
# Staircase shaft (stone walls)
for y in $(seq -30 -40); do
  place 107 $y 3  # west wall
  place 109 $y 3  # east wall
  place 108 $y 2  # stair blocks (dirt/stone mix)
done

# === MAIN MINING HALL (y=-40) ===
echo "Excavating main mining hall..."
# Floor at y=-40
for x in $(seq 100 115); do
  for y in -40 -41 -42 -43 -44; do
    place $x $y 2  # dirt floor
  done
done

# === SUPPORT PILLARS ===
echo "Raising support pillars..."
for x in 103 107 112; do
  for y in -40 -41 -42 -43 -44; do
    place $x $y 3  # stone pillars
  done
done

# === NORTH WALL ===
echo "Building north wall..."
for x in $(seq 100 115); do
  place $x -45 3
done

# === SOUTH WALL ===
echo "Building south wall..."
for x in $(seq 100 115); do
  place $x -39 3
done

# === EAST WALL ===
echo "Building east wall..."
for y in $(seq -40 -44); do
  place 116 $y 3
done

# === WEST WALL ===
echo "Building west wall..."
for y in $(seq -40 -44); do
  place 99 $y 3
done

# === ORE VEINS (decorative - different block types) ===
echo "Placing ore veins..."
# Gold veins
place 101 -41 1
place 102 -42 1
place 110 -43 1
place 111 -41 1

# Diamond veins
place 105 -43 4
place 106 -42 4
place 113 -44 4

# === LANTERNS for lighting ===
echo "Placing lanterns..."
place 101 -40 6
place 105 -40 6
place 109 -40 6
place 113 -40 6

# === EXIT STAIRCASE (east side - connects back up) ===
echo "Building exit staircase..."
for y in $(seq -39 -30); do
  place 117 $y 2
done

echo "Deep Mines complete!"
