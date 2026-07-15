#!/bin/bash
# build-west-gate-arch.sh — Build a grand west gate archway
# Adds ornamental arch, watchtowers, and a welcome sign

WHO=${1:-pilot-b}
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building West Gate Archway ==="

# Arch pillars at x=-32 and x=-27 (flanking the gate opening at x=-31..-28)
# Pillars go from y=38 up to y=48

# Left pillar (x=-32)
for y in $(seq 38 48); do
  place -32 $y 3
done
# Decorative top
place -32 49 5
place -32 50 5

# Right pillar (x=-27)
for y in $(seq 38 48); do
  place -27 $y 3
done
place -27 49 5
place -27 50 5

echo "Pillars done"

# Arch between pillars (x=-31..-28 at y=48)
for x in -31 -30 -29 -28; do
  place $x 48 3
done

echo "Arch done"

# Watchtower platforms on top of pillars
# Left tower
for x in -33 -32 -31; do
  place $x 49 5
  place $x 50 5
done
# Walls
place -33 51 3
place -31 51 3
for y in 49 50; do
  place -33 $y 3
  place -31 $y 3
done

# Right tower
for x in -28 -27 -26; do
  place $x 49 5
  place $x 50 5
done
place -28 51 3
place -26 51 3
for y in 49 50; do
  place -28 $y 3
  place -26 $y 3
done

echo "Watchtowers done"

# Welcome sign on the left pillar
# "W" sign using leaves
place -32 45 6  # W
place -32 44 6  # E
place -32 43 6  # L
place -32 42 6  # C
place -32 41 6  # O
place -32 40 6  # M
place -32 39 6  # E

echo "Sign done"

# Torches on pillars
place -32 47 6
place -27 47 6

echo "Lighting done"

# Gateway road: reinforce with planks
for x in $(seq -31 -28); do
  place $x 38 5
done

echo "Gateway road reinforced"

echo "=== West Gate Archway Complete! ==="
echo "Features: Stone pillars, arch, watchtowers, welcome sign, torches"
