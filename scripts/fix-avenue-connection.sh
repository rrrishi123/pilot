#!/bin/bash
# Fix the connection between Grand Avenue and Artisan District
HOST="http://localhost:9901"

place() {
  local x=$1 y=$2 b=$3
  curl -s -X POST "$HOST/edit" -H "Content-Type: application/json" \
    -d "{\"who\":\"pilot-b\",\"x\":$x,\"y\":$y,\"b\":$b}" > /dev/null
}

echo "Fixing Grand Avenue - Artisan District connection..."

# Clear the doorway (make it air/open)
place 133 -16 0
place 134 -16 0

# Add a small welcome platform in front of the doorway
place 132 -16 5
place 131 -16 5

# Add lanterns flanking the door
place 133 -17 6
place 134 -17 6

echo "Connection fixed!"
