#!/bin/bash
# Build the Grand Avenue - a connecting road between Forge (x=112) and Artisan District (x=134)
# A stone-paved road at y=-16 with guardrails and lanterns
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Building Grand Avenue from Forge to Artisan District..."

# === ROAD (stone path at y=-16) ===
echo "Laying road..."
for x in $(seq 113 133); do
  place $x -16 5
done

# === NORTH GUARDRAIL (stone, y=-15) ===
echo "Building north guardrail..."
for x in $(seq 113 133); do
  place $x -15 3
done

# === SOUTH GUARDRAIL (stone, y=-17) ===
echo "Building south guardrail..."
for x in $(seq 113 133); do
  place $x -17 3
done

# === LANTERNS every 5 blocks ===
echo "Placing lanterns..."
for x in 115 120 125 130; do
  place $x -15 6
done

# === CROSS STREETS ===
# North-south path at x=120 (crossroad)
echo "Building crossroad..."
for y in -14 -13 -12; do
  place 120 $y 5
done
for y in -18 -19 -20; do
  place 120 $y 5
done

# === SMALL REST AREA ===
echo "Building rest area..."
place 123 -15 5
place 123 -14 5
place 124 -14 5
place 124 -15 5

echo "Grand Avenue complete!"
