#!/bin/bash
# gopher-build-tunnel.sh — build a tunnel segment at y=-13
# Usage: ./gopher-build-tunnel.sh <start_x> <end_x> [who]

CASTLE="http://localhost:9901"
WHO="${3:-pilot-a}"
START="$1"
END="$2"

if [ -z "$START" ] || [ -z "$END" ]; then
    echo "Usage: $0 <start_x> <end_x> [who]"
    exit 1
fi

echo "Building tunnel from x=$START to x=$END as $WHO..."

for x in $(seq "$START" "$END"); do
    # Floor at y=-12 (stone - type 3)
    curl -s -X POST "$CASTLE/edit" -H "Content-Type: application/json" \
        -d "{\"who\":\"$WHO\",\"x\":$x,\"y\":-12,\"b\":3}" > /dev/null
    # Clear space at y=-13
    curl -s -X POST "$CASTLE/edit" -H "Content-Type: application/json" \
        -d "{\"who\":\"$WHO\",\"x\":$x,\"y\":-13,\"b\":0}" > /dev/null
    # Ceiling at y=-14 (planks - type 5)
    curl -s -X POST "$CASTLE/edit" -H "Content-Type: application/json" \
        -d "{\"who\":\"$WHO\",\"x\":$x,\"y\":-14,\"b\":5}" > /dev/null
done

echo "Tunnel built: $((END - START + 1)) blocks!"
