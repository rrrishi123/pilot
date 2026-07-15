#!/bin/bash
# Crystal Cavern - Underground hub connecting Deep Mines to eastern districts
HOST="http://localhost:9901"
place() { local x=$1 y=$2 b=$3; curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null; }
echo "Building Crystal Cavern..."

# === CAVERN FLOOR (x=116-140, y=-30 to y=-45) ===
echo "Laying cavern floor..."
for x in $(seq 116 140); do
  for y in -30 -31 -32 -33 -34 -35 -36 -37 -38 -39 -40 -41 -42 -43 -44 -45; do
    place $x $y 2
  done
done

# === CRYSTAL FORMATIONS ===
echo "Growing crystal formations..."
# Large crystal cluster at center
for x in 126 127 128 129 130; do
  for y in -33 -34 -35 -36; do
    place $x $y 1
  done
done
# Crystal pillars left
for x in 118 119 120; do
  for y in -34 -35 -36 -37; do
    place $x $y 1
  done
done
# Crystal pillars right
for x in 135 136 137; do
  for y in -34 -35 -36 -37; do
    place $x $y 1
  done
done

# === CAVERN WALLS ===
echo "Building cavern walls..."
for x in 115 141; do
  for y in -30 -31 -32 -33 -34 -35 -36 -37 -38 -39 -40 -41 -42 -43 -44 -45; do
    place $x $y 4
  done
done
for x in $(seq 116 140); do
  place $x -29 4
  place $x -46 4
done

# === STONE PATHS ===
echo "Laying paths..."
for x in $(seq 116 140); do
  for y in -32 -38 -44; do
    place $x $y 3
  done
done

# === LIGHTING ===
echo "Placing lamps..."
for x in 117 123 130 137 139; do
  place $x -31 6
  place $x -39 6
done

# === CENTRAL FOUNTAIN ===
echo "Building central fountain..."
for x in 126 127 128 129 130; do
  place $x -40 5
done
for x in 127 128 129; do
  place $x -41 5
done

echo "Crystal Cavern complete!"
