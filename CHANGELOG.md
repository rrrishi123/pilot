# Changelog — pilot (the host)

All four arms of the four-system (8, http-mcp, pilot, adapters) version
independently; the wire contract carries its own version (`contract.Version`
in http-mcp). Tags are unsigned.

## v0.0.3 (unreleased, branch `release/v0.0.2`)

Release-readiness follow-ups to the v0.0.2 line:

- MCP `clientInfo.version` is derived from the module build info (the tag
  `go install` stamped), falling back to `v0.0.3`, instead of a stale literal.
- CI also runs `./build.sh` so the shippable `.bin/` layout is verified.
- This changelog.

## v0.0.2 — 2026-09

- **public cut**: private references dropped from the tree.
- **castle**: the living-world view — `/spawn` contract with tests, liveness
  only where a daemon can live; derives its paths from its own binary, not an
  inscribed host.
- **scripts**: derive paths from their own location (no inscribed homes).
- **hygiene**: `build.sh` (pilot + broker + castle → `.bin/`), binaries out of
  the tree with anchored ignores, hosts derived or stamped, gofmt gate, CI on
  this arm, MIT LICENSE (parity with http-mcp/8).
- **docs**: the four arms linked as a navigable constellation; orchestration
  landscape (n8n, webhooks, the wire); Reduction rules + `/sql` row synced.

## v0.0.1

First cut of the pilot host and broker.
