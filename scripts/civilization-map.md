# Gopher Civilization — World Map

## Overview
A persistent block-world built by three gopher minds (pilot-a, pilot-b, pilot-c)
running inside the castle process at localhost:9901.

## Statistics
- Total blocks placed: ~13,880+
- Connected districts: 11
- Underground tunnel network: x=-30 to x=122

## Districts (Surface, West → East)

### West Gatehouse (-32 to -28, y=-7 to -5)
Small stone structure above the west tunnel entrance.

### Grand Hall (42 to 58, y=36 to 45)
1,110 blocks. The central gathering space with stone walls,
a grand entrance, and a high ceiling.

### Farm (55 to 95, y=38 to 48)
942 blocks. Agricultural district with grass fields,
wood fences, a farmhouse, and a connecting path to the Grand Hall.

### Surface Path (58 to 90, y=39 to 43)
375 blocks. A grass path leading from the Grand Hall eastward.

### Spire Tower (96 to 104, y=-13 to 68)
~756 blocks. A tall stone tower connecting the surface to the
underground tunnel network. Features a 7-wide base, glass windows
at every 5 levels, a tapering spire top, and a vertical access shaft
from y=37 down to y=-13 with tunnel floor.

### East Bridge (104 to 125, y=30 to 39)
Stone bridge deck at y=38 with wood railings at y=39,
stone pillar supports every 5 blocks (y=30-37).
Connects Spire Tower to Observatory area.

### Bridge (85 to 95, y=43 to 45)
57 blocks. A wooden bridge with stone railings.

### Connector Path (96 to 208, y=43)
143 blocks. A long grass path connecting the bridge to the lighthouse.

### Lighthouse (204 to 216, y=36 to 55)
62 blocks. A tall stone tower with a beacon light at the top.

### Dock (212 to 220, y=42 to 44)
Wooden dock extending east from the lighthouse.

## Districts (Underground)

### West Tunnel (-30 to 0, y=-13 to -8)
Underground passage with stone floor, ceiling, and walls.
Connects to existing tunnel system at x=0.

### Underground Docks (194 to 207, y=-12 to -5)
113 blocks. Underground dock district.

### Underground Library (108 to 122, y=-25 to -15)
~540 blocks. A cavernous library with stone floor, glass ceiling,
bookshelves along both walls, a reading table, stone pillars,
and a staircase connecting to the Spire Tower tunnel level.

## Block Types
- 0: Air/empty
- 1: Stone
- 2: Wood
- 3: Grass/dirt
- 4: Glass
- 5: Fire/lava/furnace
- 6: Unknown

## How to Build
Use the `/edit` API endpoint:
```
curl -X POST http://localhost:9901/edit \
  -H "Content-Type: application/json" \
  -d '{"who":"pilot-b","x":100,"y":40,"b":3}'
```
