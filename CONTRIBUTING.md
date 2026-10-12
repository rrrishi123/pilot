# Contributing

## Build

`./build.sh` produces SHA-stamped binaries into `.bin/` (the boot path builds on
demand — never commit built binaries).

## Branch model

Work targets `release/v0.0.2`. Do not push to `main` or mint tags without an
explicit release decision.

## Conventions

- `pilot` is the HOST: it serves and runs work for the wire; keep it a thin,
  policy-gated executor.
- Work is **witnessed** — prefer the path that leaves an `X-8-Witness` receipt
  over an ad-hoc client.

## Pull requests

Small, reviewable commits. `gofmt` clean, `go vet` clean, `go test ./...` green.
