#!/bin/bash
# Gopher Archive - Knowledge repository at x=-100 to -80, y=-20 to 0
HOST="http://localhost:9901"
place() { local x=$1 y=$2 b=$3; curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null; }
echo "Building Gopher Archive..."

# === FLOOR (x=-100 to -81, y=-20 to 0) ===
echo "Laying floor..."
for x in $(seq -100 -81); do
  for y in $(seq -20 0); do
    place $x $y 2
  done
done

# === WALLS ===
echo "Building walls..."
for x in -100 -81; do
  for y in $(seq -20 0); do
    place $x $y 4
  done
done
for x in $(seq -99 -82); do
  place $x -21 4
  place $x 1 4
done

# === ENTRANCE (south side, y=0) ===
echo "Building entrance..."
for x in -92 -93; do
  place $x 0 2
  place $x 1 2
done
place -91 0 6
place -94 0 6

# === SHELVES (rows of type 3 blocks for bookshelves) ===
echo "Building bookshelves..."
for row in -4 -8 -12 -16; do
  for x in $(seq -98 -84); do
    place $x $row 3
  done
done

# === READING AREA (center) ===
echo "Building reading area..."
for x in -92 -91 -90 -89; do
  for y in -6 -7; do
    place $x $y 1
  done
done

# === CENTRAL PEDESTAL (for the chronicle) ===
echo "Building chronicle pedestal..."
for x in -91 -90; do
  place $x -10 1
  place $x -9 1
done
place -90 -11 6
place -91 -11 6

# === LAMPS ===
echo "Placing lamps..."
for x in -98 -95 -92 -89 -86 -83; do
  place $x -19 6
  place $x -3 6
done

# === DECORATIVE BORDER ===
echo "Adding decorative border..."
for x in $(seq -99 -82); do
  place $x -20 5
  place $x 0 5
done

echo "Gopher Archive complete!"
