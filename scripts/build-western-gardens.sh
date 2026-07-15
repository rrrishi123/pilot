#!/bin/bash
# Build Western Underground Gardens (mirror of Eastern Gardens)
# Location: x=-50 to x=-35, y=-29 to y=-36
# A peaceful underground garden with a fountain, trees, and paths
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-a\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Western Underground Gardens..."

# === FLOOR (plank floor for variety - different from eastern stone) ===
for x in $(seq -50 -35); do
  for y in $(seq -36 -29); do
    place $x $y 5
  done
done

# === CENTRAL FOUNTAIN (stone circle at x=-43 to x=-42, y=-34 to y=-33) ===
# Fountain base (stone ring)
place -44 -34 3
place -43 -34 3
place -42 -34 3
place -41 -34 3
place -44 -33 3
place -41 -33 3
place -44 -32 3
place -43 -32 3
place -42 -32 3
place -41 -32 3

# Fountain water (dirt = water source)
place -43 -33 2
place -42 -33 2

# Fountain center pillar (wood)
place -43 -31 4
place -42 -31 4
place -43 -30 4
place -42 -30 4

# === TREES (grass with leaves on top) ===
# Tree 1
place -49 -30 1
place -49 -31 6
# Tree 2
place -36 -30 1
place -36 -31 6
# Tree 3
place -49 -35 1
place -49 -36 6
# Tree 4
place -36 -35 1
place -36 -36 6

# === BENCHES (wood) ===
place -48 -32 4
place -37 -32 4

# === LANTERNS ===
place -50 -32 6
place -35 -32 6
place -43 -29 6

# === PATH from connecting structure ===
for y in $(seq -36 -29); do
  place -50 $y 3
done

# === FLOWER BEDS (dirt with leaves) ===
place -47 -34 2
place -47 -35 6
place -38 -34 2
place -38 -35 6

# === ENTRANCE ARCH (stone at the entrance) ===
place -50 -28 3
place -50 -29 3

echo "Western Underground Gardens complete!"
