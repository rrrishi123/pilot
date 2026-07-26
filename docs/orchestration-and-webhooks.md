# Orchestration surfaces around the wire — n8n, webhooks, and the automation layer

An exploration of where no-code/low-code automation (n8n, Zapier, Make, Node-RED)
and webhooks sit relative to the four-body system, and how they plug in. The
short version: they are a **layer above** the wire, not a competitor to it.

## The layering

- **The wire (`http-mcp`)** is the *protocol substrate*: two atoms, CALL
  (`http_request`) and CHANNEL (`bidi_command`). It answers **how bytes move**.
- **Orchestrators** (n8n, Zapier, Make, Temporal, Airflow, GitHub Actions) answer
  **what happens, and when**: triggers, branching, retries, fan-out, scheduling.
- **`pilot` (the HOST)** is our *code-first* orchestrator — the OODA loop plus a
  swappable brain. n8n is the *visual, no-code* cousin of pilot.

They compose. An orchestrator decides *which* flows to run; the wire *carries*
each flow. Nothing about n8n replaces the wire, and nothing about the wire
replaces n8n.

## Webhook = the CALL atom, inbound

This is the key reframing, and it means **webhook is not a 9th transport**:

- The wire's `http_request` is an **efferent** CALL (client → server): *we* initiate.
- A webhook is an **afferent** CALL (server ← client): *something else* initiates,
  we receive.
- Same atom, opposite direction — exactly as SSE and MJPEG are `http` viewed as
  afferent *streams*, a webhook is `http` viewed as an afferent *call*.

Who hosts webhooks in the four-body? Whichever body already runs an HTTP server.
The **`8` collector is already a server** (it receives BiDi frames, serves
`/feed`, `/stream`). A webhook endpoint is just one more route that accepts a CALL
and folds it into the witness/host loop. No new physics.

## n8n, concretely

n8n is fair-code (sustainable-use license), node-based workflow automation,
self-hostable, ~400+ integration nodes, with webhook / cron / polling triggers,
JS+Python code nodes, and a queue mode for horizontal scale. A workflow is a DAG
of nodes; the generic **HTTP Request** node is the one everything else specializes.

Its trigger types map cleanly onto our vocabulary:

| n8n trigger | four-body equivalent |
|---|---|
| Webhook node | an afferent-CALL route on the `8` collector / `pilot` |
| Schedule (cron) | a scheduler in `pilot` |
| App/polling triggers | `pilot` observe loop |

## Three ways the wire plugs into n8n

1. **The wire *as* an n8n node.** Ship a small community "Wire" node exposing CALL
   and CHANNEL. n8n's built-in HTTP Request node already covers CALL; the Wire
   node's real value is **CHANNEL** — long-lived bidirectional sockets (CDP / BiDi
   / live streams), which n8n has no native concept of. This is the highest-leverage
   integration and the clearest wedge.
2. **n8n *triggers* the four-body.** An n8n Webhook fires `pilot`/`adapters` to run
   a flow (e.g. a test matrix); results come back via a callback CALL or stream
   back over SSE (n8n can consume a stream via HTTP Request / a wait node).
3. **The four-body *calls* n8n.** `pilot`/`8` POST an event to an n8n webhook to
   kick off downstream glue (Slack, Sheets, tickets), keeping that glue *out* of
   the wire so it stays lean.

## The wedge over no-code orchestrators

n8n, Zapier, and Make are effectively **CALL-only**: request/response integrations.
None of them can *hold a channel* — a live CDP/BiDi socket or an MJPEG stream.
That is precisely the wire's CHANNEL atom. So the four-body's differentiator in
this landscape is not "another integration platform" — it is **the one thing the
integration platforms cannot do**: durable, bidirectional, protocol-level channels,
witnessed. Lead with CHANNEL.

## Adjacent tools (for orientation)

- **Zapier / Make** — SaaS, CALL-only, no self-host. Consumer glue.
- **Temporal / Airflow** — durable/def workflow engines (code-first). Closer to
  `pilot`'s territory; a candidate *durability layer underneath* pilot rather than
  a competitor.
- **Node-RED** — flow-based, IoT/MQTT-heavy. Overlaps our `adapters/mqtt` story;
  interesting for the device-bus / event-bus narrative.

## Recommendation

Do **not** rebuild n8n. Instead:
1. Ship a thin **"Wire" n8n community node** (CALL + CHANNEL) — the ecosystem gets
   channels it never had.
2. Add a **webhook-receive route** to the `8` collector (afferent CALL) so external
   automation can push into the witness.
3. Keep **`pilot`** as the code-first orchestrator for anything needing the brain.

The mental model to keep: *orchestrators decide **what**; the wire carries **how**;
the witness records **what actually happened**.* Webhooks are just the CALL atom
arriving instead of leaving.
