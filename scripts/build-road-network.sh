#!/bin/bash
# build-road-network.sh — Connect the major zones with stone roads
# Forge (80,-12) → Spire (100,-12) → Observatory (133,38)
# This creates a proper elevated stone road

WHO=${1:-pilot-b}
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Road Network ==="

# Road 1: Forge (x=92) to Spire (x=96) — connecting the two structures
# Stone road at y=-5 (ground level in the underground zone)
for x in $(seq 92 96); do
  place $x -5 3
  place $x -6 3
done
echo "Road 1: Forge→Spire done"

# Road 2: Spire (x=104) to East Bridge (x=125)
# Surface road at y=38 (connecting to observatory path)
for x in $(seq 104 125); do
  place $x 38 3
  place $x 39 3
done
echo "Road 2: Spire→Observatory done"

# Road 3: North-South connector from Spire underground to surface
# Staircase at x=100 from y=-5 up to y=38
for step in $(seq 0 10); do
  y=$((-5 + step * 4))
  if [ $y -gt 38 ]; then break; fi
  place 100 $y 3
  place 101 $y 3
done
echo "Road 3: Vertical connector done"

# Road 4: Observatory to East — extend eastward
for x in $(seq 138 150); do
  place $x 38 3
  place $x 39 3
done
echo "Road 4: Observatory→East done"

# Road 5: Decorative lantern posts along the road
for x in 108 112 116 120 124 128 132 136 140 144 148; do
  place $x 37 5  # post
  place $x 36 5  # post
  place $x 35 5  # lantern top
done
echo "Road 5: Lanterns done"

echo "=== Road Network Complete! ==="
echo "Connects: Forge → Spire → Observatory → East Road"
echo "Total new blocks: ~$(($(seq 92 96 | wc -l)*2 + $(seq 104 125 | wc -l)*2 + 22 + $(seq 138 150 | wc -l)*2 + 33))"
