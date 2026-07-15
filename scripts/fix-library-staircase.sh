#!/bin/bash
# Fix the library staircase - replace lanterns with stone steps
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Fixing library staircase..."

# Replace lanterns at y=-19 with stone steps
place 119 -19 3
place 120 -19 3

# Add lanterns on the walls instead (side walls of the stairwell)
place 118 -19 6
place 121 -19 6

echo "Library staircase fixed!"
