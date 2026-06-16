# pilot

A local model with hands.

Talk to it like you'd talk to anyone. There is no objective up front — you can
open with `hi`. The work grows out of the dialogue. When what you ask needs a
real call against the world, the model reaches for a tool and pilot makes the
call for real; the conversation then continues with the actual response in hand.

```
brain   a local Ollama model
hands   MCP tool-servers — today http-mcp, whose one tool is an HTTP request
        (which is also how you drive any WebDriver / Appium hub)
pilot   the host between them: carries the conversation, hands the model its
        tools, executes the calls, loops
```

No SDK, no framework. stdlib only: `os/exec` runs the tool-server, `net/http`
reaches Ollama.

## Roles, kept separate

pilot is a **host** (model-aware: it drives the model and executes tools).
[http-mcp](https://github.com/rrrishi123/http-mcp) is a **server** (the wire —
model-agnostic, pure protocol calls). MCP draws that line on purpose; this repo
keeps it. The wire never grows model code; pilot grows toward more tools and
more models without polluting the wire.

## Run

```
go build -o pilot .
./pilot
```

It auto-detects the http-mcp binary: `$HTTP_MCP_BIN`, then `PATH`, then the
sibling-repo convention (`repos/pilot` next to `repos/http-mcp`). Override with
`-server /path/to/http-mcp`.

```
you ❯ hi
pilot ❯ Hey — what are we doing?
you ❯ what does github's zen endpoint say right now
  → http_request {"method":"GET","url":"https://api.github.com/zen"}
  ← {"status":200, ...}
pilot ❯ It says: "Non-blocking is better than blocking."
```

`/exit` to leave. Pipe a single line with `echo "hi" | ./pilot`.

## Flags

| flag | default | |
|---|---|---|
| `-model` | the local Gemma-4 coder GGUF | Ollama model tag |
| `-ollama` | `http://localhost:11434` | Ollama base URL |
| `-server` | auto-detect | http-mcp binary path |
| `-max-steps` | `12` | max tool rounds within one turn |

## Note on the model

A 12B local coder model calls single tools well but is weaker at long multi-step
chains than a frontier model — it sometimes narrates instead of acting. pilot
pins `temperature: 0` and a system prompt that forbids simulated calls to keep it
on the wire. Bigger quants behave better but need more memory.
