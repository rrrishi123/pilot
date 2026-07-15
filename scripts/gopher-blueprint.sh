#!/bin/bash
# gopher-blueprint.sh — place a named blueprint at a given location
# Usage: ./gopher-blueprint.sh <blueprint_name> <origin_x> <origin_y> [who]
#
# Blueprints are stored in /home/rishi/Work/pilot/scripts/blueprints/*.bp
# Format: each line is "x,y,b" relative to origin

CASTLE="http://localhost:9901"
BLUEPRINT_DIR="/home/rishi/Work/pilot/scripts/blueprints"
WHO="${3:-pilot-a}"

if [ $# -lt 2 ]; then
    echo "Usage: $0 <blueprint_name> <origin_x> <origin_y> [who]"
    echo ""
    echo "Available blueprints:"
    ls "$BLUEPRINT_DIR"/*.bp 2>/dev/null | sed 's/.*\///; s/\.bp$//' || echo "  (no blueprints yet)"
    exit 1
fi

NAME="$1"
OX="$2"
OY="$3"
[ -n "$4" ] && WHO="$4"

BP_FILE="$BLUEPRINT_DIR/$NAME.bp"
if [ ! -f "$BP_FILE" ]; then
    echo "Error: blueprint '$NAME' not found at $BP_FILE"
    exit 1
fi

echo "Placing blueprint '$NAME' at ($OX, $OY) as $WHO..."
COUNT=0

while IFS=, read -r dx dy b; do
    # Skip empty lines and comments
    [ -z "$dx" ] && continue
    [[ "$dx" =~ ^[[:space:]]*# ]] && continue
    
    X=$((OX + dx))
    Y=$((OY + dy))
    
    curl -s -X POST "$CASTLE/edit" -H "Content-Type: application/json" \
        -d "{\"who\":\"$WHO\",\"x\":$X,\"y\":$Y,\"b\":$b}" > /dev/null
    COUNT=$((COUNT + 1))
done < "$BP_FILE"

echo "Placed $COUNT blocks for blueprint '$NAME'!"
