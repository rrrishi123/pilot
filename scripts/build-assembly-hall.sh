#!/bin/bash
# Gopher Assembly Hall - Underground assembly space at x=160-179, y=-50 to y=-31
HOST="http://localhost:9901"
place() { local x=$1 y=$2 b=$3; curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null; }
echo "Building Gopher Assembly Hall..."

# === FLOOR (x=160-179, y=-50 to -31) ===
echo "Laying floor..."
for x in $(seq 160 179); do
  for y in -50 -49 -48 -47 -46 -45 -44 -43 -42 -41 -40 -39 -38 -37 -36 -35 -34 -33 -32 -31; do
    place $x $y 2
  done
done

# === WALLS ===
echo "Building walls..."
for x in 160 179; do
  for y in -50 -49 -48 -47 -46 -45 -44 -43 -42 -41 -40 -39 -38 -37 -36 -35 -34 -33 -32 -31; do
    place $x $y 4
  done
done
for x in $(seq 161 178); do
  place $x -51 4
  place $x -30 4
done

# === ENTRANCE (west side, x=160) ===
echo "Building entrance..."
for y in -45 -44 -43 -42 -41 -40; do
  place 160 $y 2
done
place 160 -46 6
place 160 -39 6

# === CENTRAL PODIUM ===
echo "Building central podium..."
for x in 168 169 170 171; do
  for y in -41 -42 -43; do
    place $x $y 1
  done
done
for x in 169 170; do
  place $x -44 1
done
place 169 -45 6
place 170 -45 6

# === PILLARS ===
echo "Building pillars..."
for x in 163 176; do
  for y in -35 -36 -37 -38 -39; do
    place $x $y 1
  done
done
for x in 163 176; do
  place $x -34 6
done

# === SEATING ROWS ===
echo "Building seating..."
for x in 163 164 165 166 167; do
  for y in -46 -47 -48 -49; do
    place $x $y 3
  done
done

# === LIGHTING ===
echo "Placing lamps..."
for x in 162 165 168 171 174 177; do
  place $x -31 6
done

# === DECORATIVE BORDER ===
echo "Adding decorative border..."
for x in $(seq 161 178); do
  place $x -50 5
  place $x -31 5
done

echo "Gopher Assembly Hall complete!"
