#!/bin/bash
# Gopher Workshop — a forge at x=110, y=38 where gophers build things
# Run: bash /home/rishi/Work/pilot/scripts/build-workshop.sh
WHO="pilot-a"
URL="http://localhost:9901/edit"

place() {
  curl -s -X POST "$URL" -H "Content-Type: application/json" \
    -d "{\"who\":\"$WHO\",\"x\":$1,\"y\":$2,\"b\":$3}" > /dev/null
}

echo "=== Building Gopher Workshop (The Forge) ==="

# Workshop floor (stone) — 10x6 area
for x in $(seq 105 114); do
  for y in $(seq 36 41); do
    place $x $y 3
  done
done
echo "Floor done"

# Walls (wood planks)
# North wall
for x in $(seq 104 115); do
  place $x 35 4
done
# South wall
for x in $(seq 104 115); do
  place $x 42 4
done
# West wall
for y in $(seq 35 42); do
  place 104 $y 4
done
# East wall with door gap
for y in $(seq 35 38); do
  place 115 $y 4
done
for y in $(seq 40 42); do
  place 115 $y 4
done
echo "Walls done"

# Roof (glass blocks — lets light in)
for x in $(seq 105 114); do
  for y in $(seq 35 41); do
    place $x $y 6
  done
done
echo "Roof done"

# Workbench (anvil area — stone center)
place 109 39 3
place 110 39 3
place 109 38 3
place 110 38 3
place 109 37 3
place 110 37 3
echo "Anvil placed"

# Blueprint table (wood)
place 107 37 5
place 107 38 5
place 107 39 5
echo "Blueprint table done"

# Storage area (glass chests)
place 112 37 6
place 113 37 6
place 112 39 6
place 113 39 6
echo "Storage done"

# Path from workshop to plaza (connecting to east path at x=100)
for x in $(seq 101 104); do
  place $x 38 3
done
echo "Connection path done"

# Sign post (wood column)
place 103 36 4
place 103 37 4
place 103 38 5
echo "Sign post done"

echo "=== Gopher Workshop Complete! ==="
echo "Location: x=104..115, y=35..42"
echo "Connected to plaza via east path at x=100"
