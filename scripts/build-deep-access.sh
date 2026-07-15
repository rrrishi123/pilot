#!/bin/bash
# Build Deep Access Shaft - extending the Grand Staircase deeper
# Location: x=113-114, y=-28 to y=-35 
# Block types: 1=grass, 2=dirt, 3=stone, 4=wood, 5=plank, 6=leaves
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Deep Access Shaft..."

# === EXTEND STAIRCASE SHAFT (x=113-114, y=-29 to y=-35) ===
for y in $(seq -29 -35); do
  place 113 $y 0
  place 114 $y 0
done

# === STAIRCASE STEPS ===
for y in $(seq -29 -35); do
  place 113 $y 3
done

# === LANDINGS at each level ===
for y in -29 -31 -33 -35; do
  place 114 $y 5  # Plank landing
done

# === LANTERNS ===
place 114 -30 6
place 114 -32 6
place 114 -34 6

# === DEEP FORGE SITE PREP (x=115-125, y=-30 to y=-35) ===
# Clear the area
for x in $(seq 115 125); do
  for y in $(seq -35 -30); do
    place $x $y 0
  done
done

# === FLOOR (stone, y=-35) ===
for x in $(seq 115 125); do
  place $x -35 3
done

echo "Deep Access Shaft complete! Ready for Deep Forge at x=115-125, y=-30 to y=-35"
