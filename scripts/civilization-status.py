#!/usr/bin/env python3
"""Civilization Status Reporter — shows what's been built and where."""
import json, os, sys
from collections import Counter, defaultdict

edits_path = os.path.expanduser("~/.pilot-castle-edits.jsonl")
if not os.path.exists(edits_path):
    print("No edits found. The world is empty.")
    sys.exit(0)

coords = {}
with open(edits_path) as f:
    for line in f:
        line = line.strip()
        if not line:
            continue
        try:
            e = json.loads(line)
            coords[(e["x"], e["y"])] = e.get("b", 0)
        except (json.JSONDecodeError, KeyError):
            pass

total = len(coords)
blocked = {k: v for k, v in coords.items() if v != 0}
air = {k: v for k, v in coords.items() if v == 0}

block_types = Counter(blocked.values())
type_names = {1: "dirt", 2: "grass", 3: "stone", 4: "wood", 5: "planks", 6: "glass"}

xs = [k[0] for k in blocked]
ys = [k[1] for k in blocked]

print(f"╔══════════════════════════════════════╗")
print(f"║   Gopher Civilization Status Report  ║")
print(f"╚══════════════════════════════════════╝")
print(f"Total blocks placed/broken: {len(coords)}")
print(f"Solid blocks standing:     {len(blocked)}")
print(f"Air blocks (dug out):      {len(air)}")
print(f"")
print(f"World bounds:  x=[{min(xs)},{max(xs)}]  y=[{min(ys)},{max(ys)}]")
print(f"")
print(f"Block composition:")
for bt, count in sorted(block_types.items()):
    name = type_names.get(bt, f"type-{bt}")
    bar = "█" * (count // 20)
    print(f"  {name:8s} ({bt}): {count:5d}  {bar}")
print(f"")

# Detect structures by looking at clusters
print(f"Structures detected:")
# Simple heuristic: count blocks at each y-level
y_levels = Counter(ys)
print(f"  Ground level (y=38): {y_levels.get(38, 0)} blocks")
print(f"  Below ground (y<38): {sum(c for y, c in y_levels.items() if y < 38)} blocks")
print(f"  Above ground (y>38): {sum(c for y, c in y_levels.items() if y > 38)} blocks")

# Check for path-like structures (long horizontal lines)
print(f"")
print(f"Active build scripts:")
scripts_dir = "/home/rishi/Work/pilot/scripts"
if os.path.isdir(scripts_dir):
    for fname in sorted(os.listdir(scripts_dir)):
        if fname.startswith("build-") and fname.endswith(".sh"):
            print(f"  ✓ {fname}")
