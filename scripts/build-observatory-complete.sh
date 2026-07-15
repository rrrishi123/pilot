#!/bin/bash
# Complete the observatory at x=125..138, y=32..44
# Fix missing walls, add telescope, and build a connecting road to forge
WHO="pilot-b"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Completing Observatory ==="

# Fix north wall at y=35 - replace leaf at (130,35) with stone
place 130 35 1

# Build proper south wall at y=44 (stone)
for x in $(seq 125 138); do
  place $x 44 1
done

# Build west wall at x=125, y=35..44 (stone)
for y in $(seq 35 44); do
  b=$(curl -s "http://localhost:9901/tile?x=125&y=$y" 2>/dev/null || echo "0")
  place 125 $y 1
done

# Build east wall at x=138, y=35..44 (stone)
for y in $(seq 35 44); do
  place 138 $y 1
done

# Add entrance on south side (doorway at x=131-132, y=44)
place 131 44 0  # clear entrance
place 132 44 0  # clear entrance

# Add telescope (stone pillar with plank top)
place 131 40 1   # base
place 132 40 1
place 131 39 1   # mid
place 132 39 1
place 131 38 5   # platform
place 132 38 5

# Add dome roof properly (leaf blocks forming arch)
# Inner arch layer
for x in $(seq 129 134); do
  place $x 33 6
done
# Top of dome
place 130 32 6
place 131 32 6
place 132 32 6
place 133 32 6

# Add lantern posts along the path (x=115..125)
for x in 117 120 123; do
  place $x 37 5
  place $x 36 5
  place $x 35 5
done

# Add a small garden in front of observatory (x=126..137, y=45..48)
for x in $(seq 126 137); do
  for y in $(seq 45 48); do
    place $x $y 2  # dirt
  done
done
# Add some grass patches
for x in 127 129 131 133 135 136; do
  place $x 45 3  # stone path
done

echo "=== Observatory Complete! ==="
