# Security Policy

`pilot` is the **HOST** of the four-system: the broker/queue that accepts work
and runs it, reporting state (`queued` / `running` / `done`) over its broker
endpoint.

## Supported versions

Active development is on `release/v0.0.2`; security fixes land there first.

## Trust & exposure model

- The broker **executes queued work** on the host. Bind it to loopback / a
  trusted mesh only; anyone who can reach the broker can run work on the host.
- It carries no embedded credentials; any it needs are supplied by the operator
  at run time, never baked into the repo.

## Reporting a vulnerability

Report privately via this repository's GitHub Security Advisories
(**Security → Report a vulnerability**), or open a minimal issue describing the
*class* of problem — no live credentials or working exploit in a public issue.
