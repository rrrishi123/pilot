#!/bin/bash
# Harbor Civilization Report
# Run: bash scripts/harbor-report.sh
# Fetches world state and generates a map of the harbor district

HOST="http://localhost:9901"
OUTFILE="/tmp/harbor-report.txt"

echo "=== HARBOR CIVILIZATION REPORT ===" > "$OUTFILE"
echo "Generated: $(date)" >> "$OUTFILE"
echo "" >> "$OUTFILE"

# Fetch world
curl -s "$HOST/world.json" > /tmp/_world.json 2>/dev/null

python3 -c "
import json

with open('/tmp/_world.json') as f:
    d = json.load(f)

blocks = {}
for e in d['edits']:
    if e['b'] != 0:
        blocks[(e['x'], e['y'])] = e['b']

total = len(blocks)
print(f'Total blocks placed: {total}')
print()

# Count by type
type_names = {1: 'Grass', 2: 'Dirt', 3: 'Stone', 4: 'Water', 5: 'Planks', 6: 'Lantern'}
counts = {}
for (x,y), b in blocks.items():
    counts[b] = counts.get(b, 0) + 1

for b in sorted(counts):
    print(f'  {type_names.get(b, str(b))}: {counts[b]}')
print()

# Harbor district stats
harbor_blocks = sum(1 for (x,y),b in blocks.items() if 183 <= x <= 215 and -16 <= y <= 12)
print(f'Harbor district (x=183..215, y=-16..12): {harbor_blocks} blocks')
print()

# Map
print('HARBOR MAP')
print('==========')
for y in range(12, -17, -1):
    line = ''
    for x in range(183, 216):
        b = blocks.get((x,y), 0)
        chars = {0:'.', 1:'g', 2:'d', 3:'s', 4:'W', 5:'P', 6:'L'}
        line += chars.get(b, '?')
    print(line)
" >> "$OUTFILE"

cat "$OUTFILE"
