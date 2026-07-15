#!/bin/bash
# Build a colonnade connecting the Forge Cathedral (x=78-94) to the Forge (x=105-112)
# A covered walkway at y=-13 with stone pillars and a roof
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Grand Colonnade from Cathedral to Forge..."

# === FLOOR (stone path at y=-16) ===
echo "Laying floor..."
for x in $(seq 95 104); do
  place $x -16 5
done

# === NORTH WALL (y=-13) ===
echo "Building north wall..."
for x in $(seq 95 104); do
  place $x -13 3
done

# === SOUTH WALL (y=-14) ===
echo "Building south wall..."
for x in $(seq 95 104); do
  place $x -14 3
done

# === PILLARS at intervals ===
echo "Raising pillars..."
for x in 95 98 101 104; do
  # Pillar from floor to roof
  for y in -15 -14 -13 -12 -11; do
    place $x $y 3
  done
done

# === ROOF ===
echo "Adding roof..."
for x in $(seq 95 104); do
  place $x -11 5
done

# === EAST ENTRANCE (connecting to forge) ===
echo "Building forge entrance..."
for y in -15 -14 -13 -12; do
  place 105 $y 3
done
place 105 -16 5

# === WEST ENTRANCE (connecting to cathedral) ===
echo "Building cathedral entrance..."
for y in -15 -14 -13 -12; do
  place 94 $y 3
done
place 94 -16 5

# === LANTERNS along the colonnade ===
echo "Lighting lanterns..."
for x in 96 99 102; do
  place $x -12 6
done

echo "Grand Colonnade complete!"
