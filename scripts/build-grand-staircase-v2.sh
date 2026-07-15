#!/bin/bash
# Build the Grand Staircase - connecting all levels of the Gopher City
# Location: x=113-114, y=-28 to y=-4 (vertical spine)
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Grand Staircase..."

# === SHAFT (clear space for stairs) ===
for y in $(seq -28 -4); do
  place 113 $y 0
  place 114 $y 0
done

# === STAIRCASE STEPS (alternating stone and plank) ===
# Going down: stone step, then air, then next stone step
for y in $(seq -5 -28); do
  # Left side step
  place 113 $y 3
  # Right side step (offset by 1)
  place 114 $((y+1)) 5
done

# === WALLS on both sides ===
for y in $(seq -28 -5); do
  place 112 $y 3
  place 115 $y 3
done

# === LANDINGS at each level ===
# Level 1: Bridge level (y=-4)
place 113 -4 5
place 114 -4 5

# Level 2: Reading Room level (y=-11)
place 113 -11 5
place 114 -11 5

# Level 3: Council Chamber level (y=-13)
place 113 -13 5
place 114 -13 5

# Level 4: Storage Room level (y=-16)
place 113 -16 5
place 114 -16 5

# Level 5: Library level (y=-21)
place 113 -21 5
place 114 -21 5

# === LANTERNS at each landing ===
place 112 -5 6
place 115 -5 6
place 112 -11 6
place 115 -11 6
place 112 -16 6
place 115 -16 6
place 112 -21 6
place 115 -21 6

echo "Grand Staircase complete!"
