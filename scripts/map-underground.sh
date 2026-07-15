#!/bin/bash
# Map the underground Library complex
HOST="http://localhost:9901"

curl -s "$HOST/world.json" | python3 -c "
import json, sys
d = json.load(sys.stdin)
blocks = {}
for e in d['edits']:
    if e['b'] != 0:
        blocks[(e['x'], e['y'])] = e['b']

total = len(blocks)
print(f'Total blocks: {total}')
print()

# Define the Library complex area
x_min, x_max = 105, 130
y_min, y_max = -22, 6

print('=== LIBRARY COMPLEX MAP ===')
print(f'Legend: . air  g grass  d dirt  s stone  W wood  P plank  L leaves')
print()
for y in range(y_max, y_min-1, -1):
    line = f'y={y:3d}: '
    for x in range(x_min, x_max+1):
        b = blocks.get((x,y), 0)
        chars = {0:'.', 1:'g', 2:'d', 3:'s', 4:'W', 5:'P', 6:'L'}
        line += chars.get(b, '?')
    print(line)
"
