#!/bin/bash
# Gopher Archive - A surface building housing the chronicle, scripts, and philosophy
HOST="http://localhost:9901"
place() { local x=$1 y=$2 b=$3; curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null; }
echo "Building Gopher Archive..."

# === BUILDING FOOTPRINT (x=85-105, y=2-16) ===
echo "Laying foundation..."
for x in $(seq 85 105); do
  for y in 2 3 4; do
    place $x $y 3
  done
done

# === WALLS ===
echo "Building walls..."
for x in 85 105; do
  for y in $(seq 5 16); do
    place $x $y 4
  done
done
for y in $(seq 5 16); do
  place 85 $y 4
  place 105 $y 4
done

# === ROOF ===
echo "Building roof..."
for x in $(seq 86 104); do
  place $x 16 3
  place $x 17 3
done

# === ENTRANCE ===
echo "Building entrance..."
for y in 5 6 7 8; do
  place 86 $y 2
  place 87 $y 2
  place 88 $y 2
done
# Entrance pillars
place 85 5 6
place 85 6 6
place 89 5 6
place 89 6 6

# === INTERIOR FLOOR ===
echo "Laying interior floor..."
for x in $(seq 89 104); do
  for y in 5 6 7 8 9 10 11 12 13 14 15; do
    place $x $y 2
  done
done

# === SHELVES (bookcases) ===
echo "Building bookshelves..."
for x in 90 92 94 96 98 100 102; do
  for y in 8 9 10 11; do
    place $x $y 1
    place $x $y 1
  done
done

# === READING TABLES ===
echo "Placing reading tables..."
for x in 91 95 99; do
  place $x 6 5
  place $x 7 5
done

# === LAMPS ===
echo "Placing lamps..."
for x in 90 95 100; do
  place $x 5 6
  place $x 12 6
done

# === CHRONICLE PLAQUE ===
echo "Building chronicle display..."
for x in 93 94 95 96 97; do
  place $x 13 5
done
place 95 14 6

# === EAST WINDOW ===
echo "Adding windows..."
for y in 8 9 10 11; do
  place 105 $y 6
done

echo "Gopher Archive complete!"
