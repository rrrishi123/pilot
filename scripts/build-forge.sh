#!/bin/bash
# Build the Forge & Workshop - a place to process materials and craft tools
# Location: x=105-112, y=-20 to y=-13 (under the staircase, left of Reading Room)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Forge & Workshop..."

# === FLOOR (stone, y=-16) ===
for x in $(seq 105 112); do
  place $x -16 3
done

# === WALLS (stone) ===
# Left wall (x=105)
for y in $(seq -20 -15); do
  place 105 $y 3
done
# Back wall (y=-20)
for x in $(seq 106 112); do
  place $x -20 3
done
# Right wall (x=112) - shared with Reading Room
for y in $(seq -20 -15); do
  place 112 $y 3
done

# === CEILING (stone, y=-11) ===
for x in $(seq 105 112); do
  place $x -11 3
done

# === FORGE (stone blocks at back, y=-19,-18) ===
place 106 -19 3
place 107 -19 3
place 106 -18 3
place 107 -18 3
# Fire in forge (leaves = fire glow)
place 107 -17 6

# === ANVIL (stone block) ===
place 109 -17 3

# === WORKBENCH (planks) ===
place 110 -17 5
place 111 -17 5

# === TOOL RACK (planks on wall) ===
place 105 -17 5
place 105 -18 5

# === LIGHT (lanterns = leaves on ceiling) ===
place 106 -12 6
place 109 -12 6
place 111 -12 6

# === OPENING to Reading Room (x=112, y=-16,-17) ===
place 112 -16 0
place 112 -17 0

# === STAIRS to Entrance Hall (at x=113, y=-15 to y=-13) ===
# Step 1
place 113 -15 3
# Step 2
place 113 -14 3
# Step 3
place 113 -13 3

echo "Forge & Workshop complete!"
