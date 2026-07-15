#!/bin/bash
# Build the Forge Cathedral above ground at the forge zone
# Location: x=80-92, y=-12 to y=-20 (above the forge)
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Forge Cathedral..."

# === GRAND ENTRANCE (facing east at x=80) ===
# Entrance pillars
for y in $(seq -12 -20); do
  place 80 $y 3
  place 83 $y 3
done

# Entrance arch
for x in $(seq 80 83); do
  place $x -12 3
done

# === CATHEDRAL FLOOR (x=80-92, y=-20) ===
for x in $(seq 80 92); do
  place $x -20 5
done

# === CATHEDRAL NAVE WALLS ===
# Left wall
for y in $(seq -13 -19); do
  place 80 $y 3
  place 81 $y 3
done

# Right wall
for y in $(seq -13 -19); do
  place 91 $y 3
  place 92 $y 3
done

# === CATHEDRAL ROOF (pointed arch) ===
# A-frame roof
for x in $(seq 81 91); do
  place $x -13 4
done

# === ALTAR at the far end (x=91-92) ===
place 91 -19 5
place 92 -19 5
place 91 -18 5
place 92 -18 5

# === PEWS ===
# Row 1
for x in $(seq 83 85); do
  place $x -17 4
done
# Row 2
for x in $(seq 83 85); do
  place $x -15 4
done

# === FORGE PIT in center ===
place 86 -19 0
place 87 -19 0
place 86 -18 0
place 87 -18 0

# Forge fire (leaves/lantern as glow)
place 86 -19 6
place 87 -19 6

echo "Forge Cathedral complete!"
