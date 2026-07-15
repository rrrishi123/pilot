#!/bin/bash
# Monument to the Lesson
# The chronicle says: "The work is not to simulate a civilization but to build one."
# This monument inscribes that truth in stone at (160-166, 55-70)
# Built by pilot-a on 2026-07-11
# Features: 3-tier stepped pyramid with glass beacon, entrance colonnade, surplus garden

HOST="http://localhost:9901"
WHO="pilot-a"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "=== Building Monument to the Lesson ==="
echo "Location: x=158..166, y=55..70"

# === TIER 1: Base platform (5x4 stone) ===
for x in $(seq 160 164); do
  for y in $(seq 60 63); do
    place $x $y 3
  done
done

# === TIER 2: Middle tier (3x3 stone) ===
for x in $(seq 161 163); do
  for y in $(seq 64 66); do
    place $x $y 3
  done
done

# === TIER 3: Pillar (1x3 stone) ===
for y in $(seq 67 69); do
  place 162 $y 3
done

# === BEACON (glass on top) ===
place 162 70 6

# === ENTRANCE PATH (planks, south) ===
for y in $(seq 55 59); do
  place 162 $y 5
done

# === ENTRANCE COLUMNS ===
place 160 55 3
place 164 55 3
place 160 56 3
place 164 56 3
place 160 57 6
place 164 57 6

# === SURPLUS GARDEN (the tier nobody ordered) ===
# Corner markers
place 158 59 5
place 166 59 5
place 158 64 5
place 166 64 5

# Decorative gold accents
place 159 59 4
place 165 59 4
place 159 64 4
place 165 64 4

# Wooden trim on base
place 159 60 5
place 165 60 5
place 159 63 5
place 165 63 5

echo "=== Monument Complete! ==="
echo "The lesson is built: build, don't simulate."
echo "The surplus is the point."
