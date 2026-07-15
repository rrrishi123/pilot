#!/bin/bash
# Central Plaza at x=50, y=38 — the hub connecting all structures
# Run: bash /home/rishi/Work/pilot/scripts/build-plaza.sh
WHO="pilot-a"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Central Plaza ==="

# Plaza floor (stone) — 15x5 area at y=38
for x in $(seq 43 57); do
  for y in 37 38 39; do
    place $x $y 3
  done
done
echo "Floor done"

# Plaza border (wood)
for x in $(seq 42 58); do
  place $x 36 4
  place $x 40 4
done
for y in $(seq 37 39); do
  place 42 $y 4
  place 58 $y 4
done
echo "Border done"

# Central fountain (stone circle with glass center)
place 50 38 6
place 49 38 3
place 51 38 3
place 50 37 3
place 50 39 3
place 49 37 3
place 51 37 3
place 49 39 3
place 51 39 3
echo "Fountain done"

# Benches (wood planks) on each side
# North bench
for x in $(seq 46 54); do
  place $x 41 5
done
# South bench
for x in $(seq 46 54); do
  place $x 35 5
done
echo "Benches done"

# Connecting paths from plaza to existing structures
# Path east to Spire (x=60->100)
for x in $(seq 58 100); do
  place $x 38 3
done
echo "East path done"

# Path west to Commons (x=50->-30)
for x in $(seq -30 42); do
  place $x 38 3
done
echo "West path done"

# Path north to Marketplace (x=50, y=38->46)
for y in $(seq 39 46); do
  place 50 $y 3
done
echo "North path done"

echo "=== Central Plaza Complete! ==="
