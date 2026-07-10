#!/bin/bash
# castle-upgrade.sh — staged atomic upgrade for the castle
# Used by gopher minds to apply code changes safely.
# Usage: castle-upgrade.sh          # show staged changes
#        castle-upgrade.sh --apply  # apply staged changes and restart
set -euo pipefail

CASTLE_SRC="/home/rishi/Work/pilot"
STAGING="/tmp/castle-staging"
BINARY="$CASTLE_SRC/castle"

echo "🏰 Castle Upgrade Tool"
echo "======================"
echo ""

if [ ! -d "$STAGING" ]; then
    echo "❌ No staging directory found at $STAGING"
    exit 1
fi

# Show whats staged
echo "📦 Staged files:"
for f in "$STAGING"/*.go "$STAGING"/*.sh; do
    [ -f "$f" ] || continue
    base=$(basename "$f")
    orig="$CASTLE_SRC/cmd/castle/$base"
    if [ -f "$orig" ]; then
        echo "   $base (diff: $(diff "$f" "$orig" 2>/dev/null | wc -l) lines changed)"
    else
        echo "   $base (NEW)"
    fi
done

if [ "${1:-}" != "--apply" ]; then
    echo ""
    echo "ℹ️  Run with --apply to apply staged changes and restart."
    exit 0
fi

echo ""
echo "🚀 Applying staged changes..."

# Apply .go files
for f in "$STAGING"/*.go; do
    [ -f "$f" ] || continue
    base=$(basename "$f")
    target="$CASTLE_SRC/cmd/castle/$base"
    echo "   cp $base → cmd/castle/"
    cp "$f" "$target"
done

# Build
echo "   Building..."
cd "$CASTLE_SRC"
go build -o "$BINARY" ./cmd/castle/ 2>&1

# Install scripts
for f in "$STAGING"/*.sh; do
    [ -f "$f" ] || continue
    base=$(basename "$f")
    target="$CASTLE_SRC/scripts/$base"
    echo "   cp $base → scripts/"
    cp "$f" "$target"
    chmod +x "$target"
done

# Restart via systemd
echo "   Restarting castle service..."
systemctl --user restart castle.service

echo ""
echo "✅ Upgrade complete! Serving at http://localhost:9901"
