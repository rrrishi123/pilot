#!/bin/bash
# Build the Forge Connector Tunnel - linking the existing forge to the Deep Forge
# Path: x=112-115, y=-20 to y=-35 (diagonal tunnel along the staircase)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Forge Connector Tunnel..."

# === TUNNEL from existing forge (x=112, y=-20) to staircase (x=113, y=-29) ===
# Clear the path
for y in $(seq -21 -29); do
  place 112 $y 0
  place 113 $y 0
done

# === STAIRCASE STEPS ===
for y in $(seq -21 -29); do
  place 112 $y 3  # Steps on left side
done

# === RAILING (planks) ===
for y in $(seq -21 -29); do
  place 113 $y 5  # Railing on right side
done

# === LANDINGS ===
place 112 -22 5
place 112 -25 5
place 112 -28 5

# === LANTERNS ===
place 112 -23 6
place 112 -26 6
place 112 -29 6

# === CONNECTION to existing forge (x=112, y=-20) ===
place 112 -20 0  # Open doorway

# === CONNECTION to staircase shaft at y=-29 ===
place 113 -29 0  # Open into main staircase

echo "Forge Connector Tunnel complete!"
