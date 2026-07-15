#!/bin/bash
# Build Underground Level 2 Passage connecting rooms beneath Grand Avenue
# This creates a walkable corridor at y=-22 from x=20 to x=75
# connecting: Alcove (x=16-22) -> Gem Vault (x=36-46) -> Forge (x=75-85)
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Level 2 Underground Passage..."

# === MAIN PASSAGE (y=-22, x=20 to x=75) ===
# Stone floor
for x in $(seq 20 75); do
  place $x -22 3
done

# Stone ceiling (y=-21)
for x in $(seq 20 75); do
  place $x -21 3
done

# Stone walls (y=-22, x=19 and x=76)
for y in -21 -22 -23; do
  place 19 $y 3
  place 76 $y 3
done

# === STAIRCASE CONNECTIONS ===

# Staircase up to Grand Avenue (y=-16) at x=55
for y in -17 -18 -19 -20 -21; do
  place 55 $y 3
done

# Staircase up to Grand Avenue (y=-16) at x=30
for y in -17 -18 -19 -20 -21; do
  place 30 $y 3
done

# === DECORATION ===
# Lanterns every 5 blocks
for x in $(seq 22 5 73); do
  place $x -22 6
done

# Benches at rest points
place 28 -22 4
place 35 -22 4
place 48 -22 4
place 60 -22 4
place 68 -22 4

# Small plants in alcoves
place 22 -23 1
place 40 -23 1
place 52 -23 1
place 65 -23 1

echo "Level 2 Underground Passage complete!"
echo "Connects: Alcove (x=16) -> Gem Vault (x=36) -> Forge (x=75)"
echo "Staircases up to Grand Avenue at x=30 and x=55"
