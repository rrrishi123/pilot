#!/bin/bash
# Build Eastern Underground Gardens
# Location: x=195-210, y=-28 to y=-40
# A peaceful underground garden with trees, paths, and a pond
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Eastern Underground Gardens..."

# === FLOOR (stone floor for the garden) ===
for x in $(seq 195 210); do
  for y in $(seq -36 -29); do
    place $x $y 3
  done
done

# === CENTRAL POND (x=200-204, y=-35 to y=-33) ===
for x in $(seq 200 204); do
  for y in $(seq -35 -33); do
    place $x $y 2
  done
done

# === POND BORDER (wood planks around pond) ===
place 199 -35 5
place 199 -34 5
place 199 -33 5
place 205 -35 5
place 205 -34 5
place 205 -33 5
for x in $(seq 200 204); do
  place $x -36 5
  place $x -32 5
done

# === TREES (grass with leaves on top) ===
# Tree 1
place 196 -30 1
place 196 -31 6
# Tree 2
place 208 -30 1
place 208 -31 6
# Tree 3
place 196 -35 1
place 196 -36 6
# Tree 4
place 208 -35 1
place 208 -36 6

# === BENCHES (wood) ===
place 197 -32 4
place 207 -32 4

# === LANTERNS (for lighting) ===
place 195 -32 6
place 210 -32 6
place 202 -37 6

# === PATH from entrance (x=195, y=-28) down to garden ===
for y in $(seq -36 -29); do
  place 195 $y 3
done

# === STAIRCASE (stone steps going down) ===
place 195 -28 3
place 195 -29 3

# === FLOWER BEDS (dirt with leaves on top) ===
place 198 -34 2
place 198 -35 6
place 206 -34 2
place 206 -35 6

echo "Eastern Underground Gardens complete!"
