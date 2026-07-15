#!/bin/bash
# Add a garden and path to the observatory
WHO="pilot-b"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Observatory Garden ==="

# Clear stray blocks at x=125, y=45-48
place 125 45 0
place 125 46 0
place 125 47 0
place 125 48 0

# Build a garden south of observatory (x=126..137, y=45..50)
# Dirt base
for x in $(seq 126 137); do
  for y in $(seq 45 50); do
    place $x $y 2
  done
done

# Stone path leading to entrance
for x in $(seq 126 137); do
  place $x 47 3  # path at y=47
done

# Grass patches (garden beds)
place 127 45 1; place 128 45 1; place 129 45 1
place 133 45 1; place 134 45 1; place 135 45 1
place 127 49 1; place 128 49 1; place 129 49 1
place 133 49 1; place 134 49 1; place 135 49 1

# Leaf bushes
place 127 46 6; place 128 46 6
place 134 46 6; place 135 46 6
place 127 48 6; place 128 48 6
place 134 48 6; place 135 48 6

# Stone border around garden
for x in $(seq 125 138); do
  place $x 45 3  # top border
  place $x 50 3  # bottom border
done
for y in $(seq 45 50); do
  place 125 $y 3  # left border
  place 138 $y 3  # right border
done

echo "=== Garden Complete! ==="
