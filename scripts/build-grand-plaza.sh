#!/bin/bash
# Build the Grand Plaza - central meeting ground connecting Library to Cathedral
# Location: x=55-75, y=-14 to y=-20
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Grand Plaza..."

# === CLEAR the area ===
for x in $(seq 55 78); do
  for y in $(seq -20 -14); do
    place $x $y 0
  done
done

# === FLOOR (stone, y=-20) ===
for x in $(seq 55 78); do
  place $x -20 3
done

# === NORTH WALL (y=-14) ===
for x in $(seq 55 78); do
  place $x -14 3
done

# === EAST WALL (x=78) ===
for y in $(seq -20 -14); do
  place 78 $y 3
done

# === WEST WALL (x=55) ===
for y in $(seq -20 -14); do
  place 55 $y 3
done

# === CENTRAL FOUNTAIN ===
place 66 -18 4
place 67 -18 4
place 66 -17 4
place 67 -17 4
place 66 -16 6
place 67 -16 6

# === COLUMNS ===
for col in 58 62 70 74; do
  for y in -19 -18 -17 -16 -15; do
    place $col $y 3
  done
  place $col -14 6  # lantern on top
done

# === BENCHES (planks) ===
place 60 -19 5
place 61 -19 5
place 72 -19 5
place 73 -19 5

# === PATH connecting to Library (west) ===
for x in 52 53 54; do
  place $x -20 5
done

# === PATH connecting to Cathedral (east) ===
for x in 79 80; do
  place $x -20 5
done

# === ENTRANCE ARCHES ===
# West arch (from Library)
place 55 -17 0
place 55 -16 0
place 55 -15 3

# East arch (to Cathedral)
place 78 -17 0
place 78 -16 0
place 78 -15 3

echo "Grand Plaza complete!"
