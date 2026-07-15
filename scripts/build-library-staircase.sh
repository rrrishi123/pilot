#!/bin/bash
# Build a proper staircase from Grand Avenue (y=-16) down to Great Library (y=-21)
# Location: x=119-120, y=-16 to y=-21
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Library Staircase..."

# The library entrance is at y=-21, x=119-120
# The Grand Avenue crossroad is at x=120, y=-16
# Build a staircase connecting them

# Clear the shaft
for y in $(seq -21 -17); do
  place 119 $y 0
  place 120 $y 0
done

# Build stone stairs (alternating)
place 119 -21 3  # bottom step
place 120 -21 3
place 119 -20 3
place 120 -20 3
place 119 -19 3
place 120 -19 3
place 119 -18 3
place 120 -18 3
place 119 -17 3
place 120 -17 3

# Entrance from Grand Avenue (y=-16)
place 119 -16 5  # plank at avenue level
place 120 -16 5

# Doorway at library level (y=-21)
place 119 -21 0  # clear for doorway
place 120 -21 0

# Lanterns on stairwell
place 119 -19 6
place 120 -19 6

echo "Library Staircase complete!"
