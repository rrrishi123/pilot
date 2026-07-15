#!/bin/bash
# Gopher Council Chamber - x=-20 to 60, y=50 to 69
HOST="http://localhost:9901"
place() { local x=$1 y=$2 b=$3; curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null; }
echo "Building Gopher Council Chamber..."

# === MAIN HALL ===
echo "Laying floor..."
for x in $(seq -20 60); do
  for y in $(seq 50 69); do
    place $x $y 2
  done
done

# === OUTER WALLS ===
echo "Building walls..."
for x in -20 60; do
  for y in $(seq 50 69); do
    place $x $y 4
  done
done
for x in $(seq -19 59); do
  place $x 49 4
  place $x 70 4
done

# === ENTRANCE (south) ===
echo "Building entrance..."
for x in 18 19 20 21 22; do
  place $x 50 2
  place $x 49 2
done
place 17 50 6
place 23 50 6

# === COUNCIL SEATING (semicircle) ===
echo "Building council seating..."
for row in 55 58 61; do
  for x in $(seq -15 55); do
    place $x $row 1
  done
done

# === CENTRAL PODIUM ===
echo "Building podium..."
for x in 17 18 19 20 21 22 23; do
  for y in 64 65 66; do
    place $x $y 1
  done
done
place 20 67 6

# === COLUMNS ===
echo "Building columns..."
for x in -15 0 15 25 40 55; do
  for y in 52 53 54; do
    place $x $y 4
  done
done

# === DECORATIVE ELEMENTS ===
echo "Adding decorations..."
for x in $(seq -19 59); do
  place $x 68 5
  place $x 51 5
done

# === LAMPS ===
echo "Placing lamps..."
for x in -15 -5 5 15 25 35 45 55; do
  place $x 69 6
  place $x 50 6
done

echo "Gopher Council Chamber complete!"
