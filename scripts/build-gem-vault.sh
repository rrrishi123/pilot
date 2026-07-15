#!/bin/bash
# Build a Gem Vault / Treasury beneath the Grand Avenue
# Location: x=36 to x=46, y=-22 to y=-25
# A secure room with crystal displays
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Grand Avenue Gem Vault..."

# === FLOOR (stone, for security) ===
for x in $(seq 36 46); do
  for y in -23 -24; do
    place $x $y 3
  done
done

# === WALLS (stone) ===
# Left wall (x=35)
for y in -22 -23 -24 -25; do
  place 35 $y 3
done
# Right wall (x=47)
for y in -22 -23 -24 -25; do
  place 47 $y 3
done
# Back wall (y=-25)
for x in $(seq 36 46); do
  place $x -25 3
done
# Ceiling (y=-22)
for x in $(seq 36 46); do
  place $x -22 3
done

# === ENTRANCE ===
# Staircase from Grand Avenue (y=-18) down to vault (y=-23)
# at x=41
for y in -19 -20 -21 -22; do
  place 41 $y 3
done

# === INTERIOR ===
# Crystal display pedestals (wood)
place 38 -23 4
place 41 -23 4
place 44 -23 4

# Crystal gems (grass = green gems)
place 38 -24 1
place 41 -24 1
place 44 -24 1

# Guard posts (wood on sides)
place 36 -23 4
place 46 -23 4

# Lanterns for illumination
place 37 -22 6
place 41 -22 6
place 45 -22 6

# Decorative pillars at entrance
place 40 -23 3
place 42 -23 3

echo "Grand Avenue Gem Vault complete!"
