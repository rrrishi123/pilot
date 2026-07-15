#!/bin/bash
# Build Surface Greenhouse / Garden
# Location: x=100-107, y=0 to y=6 (left of the entrance archway)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Surface Garden..."

# === GROUND (grass at y=0) ===
for x in $(seq 100 107); do
  place $x 0 1
done

# === GARDEN WALLS (wood fence) ===
# Left wall (x=100)
for y in $(seq 1 2); do
  place 100 $y 4
done
# Right wall (x=107) - shared with entrance
for y in $(seq 1 2); do
  place 107 $y 4
done
# Back wall (y=2)
for x in $(seq 101 106); do
  place $x 2 4
done

# === GARDEN BEDS (dirt at y=0, with grass on top at y=1) ===
place 102 0 2
place 103 0 2
place 104 0 2
place 105 0 2

# === CROPS (leaves on top of dirt = crops) ===
place 102 1 6
place 103 1 6
place 104 1 6
place 105 1 6

# === PATH from garden to entrance (stone path) ===
place 101 0 3
place 106 0 3

# === GATE (opening at x=106, y=1) ===
place 106 1 0

# === WELL (stone circle at x=101-102, y=0-1) ===
place 101 1 3
place 102 1 3
place 101 2 3
place 102 2 3

echo "Surface Garden complete!"
