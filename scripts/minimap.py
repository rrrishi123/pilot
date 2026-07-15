#!/usr/bin/env python3
"""Castle minimap generator - visualizes the gopher civilization world"""
import json, math, sys, os
from collections import defaultdict

WORLD_URL = "http://localhost:9901/world.json"

def fetch_world():
    import urllib.request
    with urllib.request.urlopen(WORLD_URL) as f:
        return json.loads(f.read().decode())

def main():
    world = fetch_world()
    edits = world.get('edits', [])
    
    blocks = {}
    for e in edits:
        x = e.get('x', 0)
        y = e.get('y', 0)
        b = e.get('b', 0)
        if b != 0:
            blocks[(x, y)] = b
    
    if not blocks:
        print("No blocks found")
        return
    
    xs = [p[0] for p in blocks]
    ys = [p[1] for p in blocks]
    min_x, max_x = min(xs), max(xs)
    min_y, max_y = min(ys), max(ys)
    
    symbols = {1: '░', 2: '▒', 3: '▓', 4: '█', 5: '▌', 6: '○'}
    
    print(f"╔═══ CASTLE MINIMAP ═══╗")
    print(f"║ X: {min_x}..{max_x}  Y: {min_y}..{max_y}")
    print(f"║ Blocks: {len(blocks)}")
    print(f"╚══════════════════════╝")
    print()
    
    # Surface view (y=35..55)
    print("┌── SURFACE (y=35..55) ──┐")
    for y in range(55, 34, -1):
        line = f"{y:3d}│"
        for x in range(-20, 221):
            b = blocks.get((x, y), 0)
            line += symbols.get(b, ' ')
        print(line + "│")
    print("└" + "─" * 243 + "┘")
    print()
    
    # Underground view (y=-15..0)
    print("┌── UNDERGROUND (y=-15..0) ──┐")
    for y in range(0, -16, -1):
        line = f"{y:3d}│"
        for x in range(-20, 221):
            b = blocks.get((x, y), 0)
            line += symbols.get(b, ' ')
        print(line + "│")
    print("└" + "─" * 243 + "┘")
    
    # Stats
    builders = defaultdict(int)
    for e in edits:
        w = e.get('who', 'unknown')
        builders[w] += 1
    
    print()
    print("Builders:")
    for w, c in sorted(builders.items(), key=lambda x: -x[1]):
        print(f"  {w}: {c}")

if __name__ == "__main__":
    main()
