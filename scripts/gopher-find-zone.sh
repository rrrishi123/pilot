#!/bin/bash
# gopher-find-zone.sh — find a flat area to build on
# Scans the world surface and reports flat regions
CASTLE="http://localhost:9901"

echo "Fetching world data..."
WORLD=$(curl -s "$CASTLE/world.json")

echo "$WORLD" | python3 -c "
import json, sys, math

d = json.load(sys.stdin)
edits = d.get('edits', [])

# Build edit map
edit_map = {}
for e in edits:
    edit_map[(e['x'], e['y'])] = e['b']

# Surface function (mirrors the JS)
seed = d.get('seed', 1337)
def rnd(n):
    n = (n * 1103515245 + 12345 + seed) & 0x7fffffff
    return ((n >> 16) & 0x7fff) / 0x7fff

def surfaceH(tx):
    import math
    return math.floor(42 + 4*math.sin(tx*0.14+seed) + 2.5*math.sin(tx*0.4) + 2*rnd(tx*13)-1)

# Scan for flat areas (4+ consecutive tiles with same surface height)
flat_zones = []
current_start = None
current_height = None

for x in range(-50, 100):
    sh = surfaceH(x)
    if sh == current_height:
        continue
    else:
        if current_start is not None and (x - current_start) >= 5:
            flat_zones.append((current_start, x-1, current_height, x - current_start))
        current_start = x
        current_height = sh

if current_start is not None and (100 - current_start) >= 5:
    flat_zones.append((current_start, 99, current_height, 100 - current_start))

print('Flat building zones (≥5 tiles wide):')
print(f'{\"Start\":>6} {\"End\":>6} {\"Height\":>6} {\"Width\":>6}  {\"Ground Y\":>8}')
print('-' * 50)
for start, end, height, width in flat_zones:
    # Show if there are existing edits in this zone
    edit_count = sum(1 for (ex, ey) in edit_map if start <= ex <= end and ey >= height-5 and ey <= height+5)
    flag = '  [has edits]' if edit_count > 0 else ''
    print(f'{start:>6} {end:>6} {height:>6} {width:>6}  y={height:>3}{flag}')
"
