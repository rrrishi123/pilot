#!/bin/bash
# build_rect.sh — build a rectangle of blocks in the castle world
# Usage: ./build_rect.sh <x1> <x2> <y1> <y2> <block_type> [who]
# Example: ./build_rect.sh 0 10 40 45 3 pilot-b

X1=$1
X2=$2
Y1=$3
Y2=$4
BTYPE=$5
WHO=${6:-pilot-b}

for x in $(seq $X1 $X2); do
  for y in $(seq $Y1 $Y2); do
    curl -s -X POST http://localhost:9901/edit \
      -H "Content-Type: application/json" \
      -d "{\"who\":\"$WHO\",\"x\":$x,\"y\":$y,\"b\":$BTYPE}"
  done
done

echo "Placed $(($X2-$X1+1))*$(($Y2-$Y1+1)) = $((($X2-$X1+1)*($Y2-$Y1+1))) blocks of type $BTYPE at ($X1,$Y1)-($X2,$Y2)"
