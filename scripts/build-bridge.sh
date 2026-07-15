#!/bin/bash
# Build a Bridge connecting Garden to Entrance, and extend the underground
# Location: x=100-112, y=-4 to y=-10 (between surface and Reading Room)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Underground Bridge and Garden Extension..."

# === BRIDGE from Garden (x=100-107, y=-4) to Reading Room (x=108-112, y=-4) ===
# Bridge floor (planks)
for x in $(seq 104 112); do
  place $x -4 5
done
# Bridge railings (wood)
for x in $(seq 104 112); do
  place $x -5 4
  place $x -3 4
done

# === STAIRCASE from Bridge down to Reading Room floor (y=-11) ===
# Staircase at x=112 going down
for y in $(seq -4 -1 -11); do
  place 112 $y 3
done

# === GARDEN EXTENSION (underground greenhouse at x=100-107, y=-6 to y=-10) ===
# Clear space
for x in $(seq 100 107); do
  for y in $(seq -6 -10); do
    place $x $y 0
  done
done
# Floor (grass)
for x in $(seq 100 107); do
  place $x -10 1
done
# Walls (wood)
for x in 100 107; do
  for y in $(seq -6 -9); do
    place $x $y 4
  done
done
# Ceiling (planks)
for x in $(seq 100 107); do
  place $x -6 5
done

# === GLOWSHROOMS (leaves = glowing mushrooms) ===
place 102 -9 6
place 105 -9 6
place 103 -8 6
place 106 -8 6

# === PATH from underground garden to staircase ===
place 108 -10 3
place 109 -10 3
place 110 -10 3
place 111 -10 3

echo "Underground Bridge and Garden Extension complete!"
