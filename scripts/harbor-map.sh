#!/bin/bash
# harbor-map.sh — Render the harbor district from the live castle world
# Usage: bash scripts/harbor-map.sh [x_min] [x_max] [y_min] [y_max]
# Default: harbor district (x=155 to x=215, y=-20 to y=12)

HOST="http://localhost:9901"
XMIN=${1:-155}
XMAX=${2:-215}
YMIN=${3:--20}
YMAX=${4:-12}

echo "Fetching world.json from $HOST..."
DATA=$(curl -s "$HOST/world.json")
echo "$DATA" > /tmp/harbor-map-data.json

python3 << PYEOF
import json, sys

with open('/tmp/harbor-map-data.json') as f:
    world = json.load(f)

blocks = {}
for e in world.get('edits', []):
    if e.get('b', 0) != 0:
        blocks[(e['x'], e['y'])] = e['b']

xmin, xmax = $XMIN, $XMAX
ymin, ymax = $YMIN, $YMAX

total = len(blocks)
print(f"Total blocks in world: {total}")
print(f"Map: x={xmin}..{xmax}, y={ymin}..{ymax}")
print()

CHARS = {0: '.', 1: 'g', 2: 'd', 3: 's', 4: 'W', 5: 'P', 6: 'L'}
LEGEND = {'.': 'air', 'g': 'grass', 'd': 'dirt', 's': 'stone', 'W': 'water', 'P': 'planks', 'L': 'lantern'}

for y in range(ymax, ymin - 1, -1):
    line = f"y={y:3d}: "
    for x in range(xmin, xmax + 1):
        b = blocks.get((x, y), 0)
        line += CHARS.get(b, '?')
    print(line)

print()
print("Legend:")
for c, name in LEGEND.items():
    print(f"  {c} = {name}")
PYEOF
