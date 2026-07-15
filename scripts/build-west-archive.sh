#!/bin/bash
# West Gopher Archive - Knowledge repository at x=-170 to -140, y=-20 to 10
HOST="http://localhost:9901"
place() { local x=$1 y=$2 b=$3; curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null; }
echo "Building West Gopher Archive..."

# === FLOOR (x=-170 to -141, y=-20 to 10) ===
echo "Laying floor..."
for x in $(seq -170 -141); do
  for y in $(seq -20 10); do
    place $x $y 2
  done
done

# === WALLS ===
echo "Building walls..."
for x in -170 -141; do
  for y in $(seq -20 10); do
    place $x $y 4
  done
done
for x in $(seq -169 -142); do
  place $x -21 4
  place $x 11 4
done

# === ENTRANCE (south side, y=11) ===
echo "Building entrance..."
for x in -158 -157; do
  place $x 10 2
  place $x 11 2
done
place -156 10 6
place -159 10 6

# === SHELVES (rows of type 3) ===
echo "Building bookshelves..."
for row in -15 -10 -5 0 5; do
  for x in $(seq -168 -144); do
    place $x $row 3
  done
done

# === READING AREA (center) ===
echo "Building reading area..."
for x in -157 -156 -155 -154; do
  for y in -8 -7 -6 -5; do
    place $x $y 1
  done
done

# === CHRONICLE PEDESTAL ===
echo "Building chronicle pedestal..."
for x in -156 -155; do
  for y in -13 -12; do
    place $x $y 1
  done
done
place -155 -14 6
place -156 -14 6

# === LAMPS ===
echo "Placing lamps..."
for x in -168 -165 -162 -159 -156 -153 -150 -147 -144; do
  place $x -19 6
  place $x 9 6
done

# === DECORATIVE BORDER ===
echo "Adding decorative border..."
for x in $(seq -169 -142); do
  place $x -20 5
  place $x 10 5
done

echo "West Gopher Archive complete!"
