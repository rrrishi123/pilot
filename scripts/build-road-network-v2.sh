#!/bin/bash
# Build the Surface Road Network - connecting Garden, Grand Hall, and Observation Deck
# Location: x=100-115, y=-1 (surface path)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Surface Road Network..."

# === MAIN PATH (dirt, y=-1) from Garden to Grand Hall ===
for x in $(seq 100 115); do
  place $x -1 2
done

# === PATH from Grand Hall to Observation Deck (x=102-105, y=6-7) ===
for y in $(seq 0 6); do
  place 102 $y 2
done

# === PATH BRANCH to east (future expansion) ===
for x in $(seq 116 120); do
  place $x -1 2
done

# === PATH LIGHTING (torches along road) ===
place 103 -1 6
place 107 -1 6
place 111 -1 6
place 115 -1 6

# === SMALL GARDEN PLOTS along road ===
# Left side garden
for x in 100 101; do
  place $x 0 1
done
place 100 1 6
place 101 1 6

# Right side garden
for x in 114 115; do
  place $x 0 1
done
place 114 1 6
place 115 1 6

# === BENCHES (wood) ===
place 105 -1 4
place 110 -1 4

echo "Surface Road Network complete!"
