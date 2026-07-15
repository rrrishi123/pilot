#!/bin/bash
# Build the Deep Forge - a grand forge complex deep underground
# Location: x=115-125, y=-30 to y=-35
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Deep Forge..."

# === FLOOR (stone, y=-35) ===
for x in $(seq 115 125); do
  place $x -35 3
done

# === WALLS (stone) ===
# Left wall
for y in $(seq -35 -30); do
  place 115 $y 3
done
# Right wall
for y in $(seq -35 -30); do
  place 125 $y 3
done
# Back wall (y=-30)
for x in $(seq 116 124); do
  place $x -30 3
done

# === ENTRANCE ARCH (front, y=-35) ===
place 120 -35 0  # Doorway center
place 121 -35 0  # Doorway

# === GRAND FORGE (center back) ===
# Main forge structure (stone)
for x in 118 119 120 121 122; do
  place $x -33 3
  place $x -32 3
done
# Forge fire glow (leaves)
place 119 -32 6
place 120 -32 6
place 121 -32 6

# === BELLOWS (wood) ===
place 117 -34 4
place 117 -33 4

# === ANVILS (stone) ===
place 123 -34 3
place 124 -34 3

# === WORKBENCHES (planks) ===
for x in 116 117; do
  place $x -34 5
done

# === TOOL RACKS (planks on walls) ===
for y in -34 -33 -32; do
  place 115 $y 5
  place 125 $y 5
done

# === WATER TROUGH (wood, left side) ===
for x in 116 117; do
  place $x -32 4
done

# === COAL STORAGE (dirt, right side) ===
for x in 123 124; do
  place $x -32 2
done

# === LANTERNS ===
place 115 -31 6
place 125 -31 6
place 118 -31 6
place 122 -31 6

# === CONNECTION to staircase (x=114, y=-35) ===
place 114 -35 0  # Open doorway to staircase

echo "Deep Forge complete!"
