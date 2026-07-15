#!/bin/bash
# gopher-build-house.sh — build a complete house at a given location
# Usage: ./gopher-build-house.sh <x> <y> [who]
# Builds a 5x5 house with floor, walls, roof, and doorway

CASTLE="http://localhost:9901"
WHO="${3:-pilot-a}"
X=$1
Y=$2

if [ -z "$X" ] || [ -z "$Y" ]; then
    echo "Usage: $0 <x> <y> [who]"
    exit 1
fi

echo "Building house at ($X, $Y) as $WHO..."

# Floor (stone)
for dx in 0 1 2 3 4; do
    curl -s -X POST "$CASTLE/edit" -H "Content-Type: application/json" \
        -d "{\"who\":\"$WHO\",\"x\":$((X+dx)),\"y\":$Y,\"b\":3}" > /dev/null
done

# Walls (wood) - left and right
for dy in 1 2 3 4; do
    curl -s -X POST "$CASTLE/edit" -H "Content-Type: application/json" \
        -d "{\"who\":\"$WHO\",\"x\":$X,\"y\":$((Y+dy)),\"b\":4}" > /dev/null
    curl -s -X POST "$CASTLE/edit" -H "Content-Type: application/json" \
        -d "{\"who\":\"$WHO\",\"x\":$((X+4)),\"y\":$((Y+dy)),\"b\":4}" > /dev/null
done

# Back wall (wood)
for dy in 1 2 3 4; do
    for dx in 1 2 3; do
        curl -s -X POST "$CASTLE/edit" -H "Content-Type: application/json" \
            -d "{\"who\":\"$WHO\",\"x\":$((X+dx)),\"y\":$((Y+dy)),\"b\":4}" > /dev/null
    done
done

# Roof (planks)
for dx in 0 1 2 3 4; do
    curl -s -X POST "$CASTLE/edit" -H "Content-Type: application/json" \
        -d "{\"who\":\"$WHO\",\"x\":$((X+dx)),\"y\":$((Y+5)),\"b\":5}" > /dev/null
done

# Doorway (break front wall)
for dx in 1 2 3; do
    curl -s -X POST "$CASTLE/edit" -H "Content-Type: application/json" \
        -d "{\"who\":\"$WHO\",\"x\":$((X+dx)),\"y\":$((Y+1)),\"b\":0}" > /dev/null
done

echo "House built at ($X, $Y)!"
