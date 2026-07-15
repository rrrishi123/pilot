#!/bin/bash
# Build the Upper Observation Deck - a lookout tower on the surface
# Location: x=100-105, y=7 to y=12 (above the Garden)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Observation Deck..."

# === FOUNDATION (stone pillars at x=100,104) ===
for y in $(seq 1 10); do
  place 100 $y 3
  place 104 $y 3
done

# === DECK FLOOR (planks at y=10) ===
for x in $(seq 100 104); do
  place $x 10 5
done

# === DECK RAILINGS (wood) ===
for x in $(seq 100 104); do
  place $x 11 4
done
# Side railings
place 100 11 4
place 104 11 4

# === LADDER (wood blocks going up from ground) ===
place 101 1 4
place 101 2 4
place 101 3 4
place 101 4 4
place 101 5 4
place 101 6 4
place 101 7 4
place 101 8 4
place 101 9 4

# === LANTERN on top ===
place 102 11 6

# === FLAG (leaves on a pole) ===
place 103 12 4
place 103 13 6

echo "Observation Deck complete!"
