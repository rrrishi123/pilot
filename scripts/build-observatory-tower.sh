#!/bin/bash
# Build a Grand Observatory Tower on the surface
# Location: x=95-105, y=40-55 (above the Grand Avenue/Library area)
# This gives gophers a view of their civilization from above
WHO="pilot-b"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Grand Observatory Tower ==="

# Tower base (x=98-102, y=40..42)
for x in $(seq 98 102); do
  for y in 40 41 42; do
    place $x $y 3
  done
done
echo "Base done"

# Tower shaft (x=99-101, y=43..49)
for x in 99 100 101; do
  for y in $(seq 43 49); do
    place $x $y 3
  done
done
echo "Shaft done"

# Observation deck (x=96-104, y=50)
for x in $(seq 96 104); do
  place $x 50 5
done
echo "Deck floor done"

# Glass walls for observation deck (x=96-104, y=51)
for x in $(seq 96 104); do
  place $x 51 6
done
echo "Deck walls done"

# Roof (x=97-103, y=52)
for x in $(seq 97 103); do
  place $x 52 3
done
echo "Roof done"

# Staircase inside (spiral)
place 100 43 0
place 100 44 0
place 100 45 0
place 100 46 0
place 100 47 0
place 100 48 0
place 100 49 0
echo "Staircase done"

# Lanterns on deck
place 96 50 6
place 104 50 6
echo "Lanterns done"

echo "=== Grand Observatory Tower Complete! ==="
