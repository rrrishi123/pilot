#!/bin/bash
# Build the Mine Entrance - surface entrance to the Deep Forge
# Location: x=95-110, y=0 to y=8
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Mine Entrance..."

# === MINE SHAFT ENTRANCE (stone arch) ===
# Left pillar
for y in $(seq 0 5); do
  place 98 $y 3
done
# Right pillar
for y in $(seq 0 5); do
  place 102 $y 3
done
# Arch top (3 wide)
place 98 6 3
place 99 6 3
place 100 6 3
place 101 6 3
place 102 6 3

# === MINE CART TRACKS (wood leading to entrance) ===
for x in $(seq 95 97); do
  place $x 0 4
done
place 98 0 0  # Open doorway
place 99 0 0
place 100 0 0
place 101 0 0
place 102 0 0

# === SUPPORT BEAMS (wood, going down) ===
for y in $(seq -1 -10); do
  place 98 $y 4
  place 102 $y 4
done

# === LADDER (plank) ===
for y in $(seq -1 -10); do
  place 100 $y 5
done

# === LANTERNS ===
place 99 -3 6
place 101 -3 6
place 99 -7 6
place 101 -7 6

# === CONNECTION to underground bridge level ===
# Clear path at y=-5 to connect to bridge
for x in 98 99 100 101 102; do
  place $x -5 0
done

# === ORE DEPOSITS (stone with different blocks) ===
place 97 -3 2  # Dirt vein
place 103 -3 2
place 97 -6 2
place 103 -6 2
place 97 -9 2
place 103 -9 2

# === MINE CART (wood + planks) ===
place 95 0 4
place 96 0 5
place 95 -1 4
place 96 -1 5

echo "Mine Entrance complete!"
