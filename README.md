# pilot

A local model with hands.

Talk to it like you'd talk to anyone. There is no objective up front — you can
open with `hi`. The work grows out of the dialogue. When what you ask needs a
real call against the world, the model reaches for a tool and pilot makes the
call for real; the conversation then continues with the actual response in hand.

```
brain   a local Ollama model
hands   the wire   — http-mcp tool-server: http_request + discover
                     (also how you drive any WebDriver / Appium hub)
        built-ins  — run_command (shell), read_file, write_file, list_dir
pilot   the host between them: carries the conversation, hands the model the
        merged tool set, executes the calls, loops
```

No SDK, no framework. stdlib only: `os/exec` runs the tool-server and shell,
`net/http` reaches Ollama.

## Tools

| tool | source | what it does |
|---|---|---|
| `http_request` | wire (http-mcp) | one HTTP call — any API, or a WebDriver/Appium hub |
| `discover` | wire (http-mcp) | probe what a hub actually serves |
| `run_command` | host (built-in) | run a shell command (`adb`, `git`, …) — controls the machine & real devices |
| `read_file` / `write_file` / `list_dir` | host (built-in) | the local filesystem |

Shell + filesystem live in the **host**, not the wire — same split Claude Code
uses (its Bash/Read/Write belong to the harness, not a remote service). http-mcp
stays a pure HTTP server.

**Safety:** `run_command` and `write_file` ask `allow …? [y/N]` before running in
an interactive session. Pass `-yes` to auto-approve (e.g. for piped input). The
`→` line always prints the exact call first, so nothing runs unseen.

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

**Keep the chat clean:** run the Ollama engine as a background service
(`brew services start ollama`) rather than `ollama serve &` in your terminal —
otherwise the engine's `slot`/`GIN` logs print into the same pane as your chat.
pilot already routes the tool-server's own logs to a file (see `-log`).

## Flags

| flag | default | |
|---|---|---|
| `-model` | the local Gemma-4 coder GGUF | Ollama model tag |
| `-ollama` | `http://localhost:11434` | Ollama base URL |
| `-server` | auto-detect | http-mcp binary path |
| `-max-steps` | `12` | max tool rounds within one turn |
| `-show-thinking` | `false` | print the model's hidden reasoning (noisy) |
| `-yes` | `false` | auto-approve `run_command` / `write_file` |
| `-log` | temp file | where the tool-server's logs go |

## Note on the model

A 12B local coder model calls single tools well but is weaker at long multi-step
chains than a frontier model — it sometimes narrates instead of acting. pilot
pins `temperature: 0` and a system prompt that forbids simulated calls to keep it
on the wire. Bigger quants behave better but need more memory.
