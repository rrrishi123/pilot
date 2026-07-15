#!/bin/bash
# Observatory at x=130, y=38 — a domed structure on the eastern hill
# Run: bash /home/rishi/Work/pilot/scripts/build-observatory.sh
WHO="pilot-b"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Observatory ==="

# Observatory floor (stone) — 12x8 area
for x in $(seq 126 137); do
  for y in $(seq 36 43); do
    place $x $y 3
  done
done
echo "Floor done"

# Outer walls (stone)
# North wall
for x in $(seq 125 138); do
  place $x 35 1
done
# South wall
for x in $(seq 125 138); do
  place $x 44 1
done
# West wall
for y in $(seq 35 44); do
  place 125 $y 1
done
# East wall
for y in $(seq 35 44); do
  place 138 $y 1
done
echo "Walls done"

# Inner chamber walls (glass) — telescope room in center
for x in $(seq 129 134); do
  place $x 38 6
  place $x 42 6
done
for y in $(seq 38 42); do
  place 129 $y 6
  place 134 $y 6
done
echo "Inner chamber done"

# Telescope base (stone pillar)
place 131 40 1
place 132 40 1
place 131 39 1
place 132 39 1
echo "Telescope base done"

# Dome roof (glass blocks forming a dome shape)
# Layer 1 (top)
place 131 34 6
place 132 34 6
# Layer 2
for x in $(seq 129 134); do
  place $x 33 6
done
# Layer 3
for x in $(seq 128 135); do
  place $x 32 6
done
echo "Dome done"

# Observation deck (wood platform around dome)
for x in $(seq 126 137); do
  place $x 34 5
done
echo "Observation deck done"

# Connecting path from workshop (x=115) to observatory (x=125)
for x in $(seq 115 125); do
  place $x 38 3
done
echo "Connecting path done"

# Lantern posts along the path
for x in 117 120 123; do
  place $x 37 5
  place $x 36 5
  place $x 35 5
done
echo "Lanterns done"

echo "=== Observatory Complete! ==="
echo "Location: x=125..138, y=32..44"
echo "Features: Dome roof, telescope chamber, observation deck, lantern path"
