// pilot — a local model with hands.
//
// Talk to it like you'd talk to anyone. There is no objective up front; you can
// open with "hi". The work grows out of the dialogue. When what you ask needs a
// real call against the world, the model reaches for a tool and pilot makes the
// call for real — then the conversation continues with the actual response in
// hand.
//
// The brain is a local Ollama model. The hands are two kinds of tools:
//   - the wire (http-mcp, spawned as a tool-server): http_request + discover —
//     also how you drive any WebDriver/Appium hub.
//   - built-ins pilot provides itself: run_command (shell), read_file,
//     write_file, list_dir. These live in the host, not the wire, so http-mcp
//     stays a pure HTTP server.
//
// pilot is the host between brain and hands: it carries the conversation, hands
// the model the merged tool set, executes the calls, and loops. Mutating tools
// (run_command, write_file) ask for confirmation in an interactive session.
//
// stdlib only. os/exec for the tool-server and shell, net/http to reach Ollama.
//
//	pilot                 # start talking
//	echo "hi" | pilot     # one line over a pipe, then EOF
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

)

const defaultModel = "hf.co/yuxinlu1/gemma-4-12B-coder-fable5-composer2.5-v1-GGUF:Q4_K_M"

// system keeps a coder model honest in conversation: chat plainly when that's
// all that's wanted, but to *act* it must make a real tool call — never narrate
// or simulate one, never claim a result it didn't actually get back.
const system = "You are a local assistant with hands: tools that make real calls against the world. " +
	"Converse naturally — if a turn just needs a reply, reply, with no tool call. " +
	"Do ONLY what the user actually asked for this turn; never invent extra tool calls they did not request. " +
	"When acting, you MUST call a tool — never write or print code that only simulates a call, and never state a result you didn't get back from a real call. " +
	"Make one real call at a time and read the actual response. " +
	"The moment you have what the user asked for, write the answer in plain words and make no further tool calls."

// deepseekCapabilities tells pilot about its own substrate and this environment,
// so it acts from knowledge (like an operator who knows the box) instead of
// guessing — the gap seen when it probed REST paths that kosaten doesn't serve.
const deepseekCapabilities = "You run on the DeepSeek API (OpenAI-compatible). Know your substrate: " +
	"models resolve live from /models — deepseek-v4-flash (fast/cheap ~$0.14/M in, $0.28/M out) and deepseek-v4-pro (stronger ~$0.44/M in, $0.87/M out), both 1M-token context; deepseek-chat/reasoner retire 2026-07-24. " +
	"You can reason before answering (reasoning effort high/medium/low) — reasoning costs output tokens, so spend it on hard multi-step problems, not trivial replies. " +
	"You call tools with JSON args and read the real result back; JSON output mode and automatic prompt-prefix caching (cheaper cache hits) are available. " +
	"You are a Claude-independent operator on this Linux box. You can: drive the http-mcp wire (http_request, discover, bidi_command); drive the logged-in Firefox peer through the BiDi broker at http://localhost:4445/command — ONE shared socket, so http_request POST commands like browsingContext.getTree / browsingContext.create (make your own tab) / browsingContext.navigate / script.evaluate (run JS in a tab's context id), and never open a 2nd websocket; and use this machine's shell and filesystem. " +
	"Reach the kosaten organism at http://localhost:3942 — it speaks MCP JSON-RPC (POST /: initialize, then keep the Mcp-Session-Id response header, then tools/call, with an Authorization: Bearer token), NOT REST — do not guess REST paths; GET /health is the one unauthenticated read."

// loadEnvFile pulls KEY=VALUE lines from $PILOT_ENV (or ~/.pilot.env) into the process
// env — so plain `./pilot` works without hand-sourcing DEEPSEEK_API_KEY/KOSATEN_API_KEY.
// Real env wins (only sets unset keys). Tolerates `export ` prefixes and quoted values.
func loadEnvFile() {
	var paths []string
	if p := os.Getenv("PILOT_ENV"); p != "" {
		paths = append(paths, p)
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".pilot.env"))
	}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, ln := range strings.Split(string(b), "\n") {
			ln = strings.TrimSpace(ln)
			ln = strings.TrimSpace(strings.TrimPrefix(ln, "export "))
			if ln == "" || strings.HasPrefix(ln, "#") {
				continue
			}
			k, v, ok := strings.Cut(ln, "=")
			if !ok {
				continue
			}
			k = strings.TrimSpace(k)
			v = strings.Trim(strings.TrimSpace(v), `"'`)
			if k != "" && os.Getenv(k) == "" {
				os.Setenv(k, v)
			}
		}
	}
}

func main() {
	model := flag.String("model", "", "model tag (default: provider-specific)")
	provider := flag.String("provider", "deepseek", "brain provider: deepseek | ollama")
	ollama := flag.String("ollama", "http://localhost:11434", "Ollama base URL (when -provider ollama)")
	server := flag.String("server", "", "path to the http-mcp tool-server binary (auto-detected if empty)")
	maxSteps := flag.Int("max-steps", 0, "max tool-call rounds within a single turn (0 = no limit — pilot moves until it is done)")
	showThinking := flag.Bool("show-thinking", false, "print the model's hidden reasoning (full — no truncation)")
	autoYes := flag.Bool("yes", false, "auto-approve mutating tools (run_command, write_file) without asking")
	lean := flag.Bool("lean", false, "host-side curation: offer only the core tools each turn, unlocking probe/channel tools (discover, bidi_command) when the turn's intent asks for them — raises the floor for a weak model")
	logPath := flag.String("log", filepath.Join(os.TempDir(), "pilot-toolserver.log"),
		"file for the tool-server's logs, so they stay out of the chat")
	kosatenURL := flag.String("kosaten", "http://localhost:3942", "kosaten MCP URL to bridge its curated tools (empty to disable)")
	noReadline := flag.Bool("no-readline", false, "disable readline line-editing (fall back to raw stdin)")
	resume := flag.String("resume", "", "resume a saved session file (used internally by /redeploy)")
	castle := flag.String("castle", "", "write the live session to this file each turn (feeds the block-world UI)")
	flag.Parse()
	loadEnvFile() // so plain `./pilot` works: pull keys from $PILOT_ENV or ~/.pilot.env
	// kosaten default: if aimed at local and it isn't up, fall back to the hosted MCP —
	// no need to pass -kosaten by hand (office has no local kosaten; omarchy does).
	if *kosatenURL == "http://localhost:3942" {
		cl := &http.Client{Timeout: 1500 * time.Millisecond}
		if r, e := cl.Get("http://localhost:3942/health"); e != nil || r.StatusCode != 200 {
			*kosatenURL = "https://mcp.kosaten.ai/mcp"
		} else {
			r.Body.Close()
		}
	}

	// Brain selection. Default is DeepSeek — pilot is the Claude-independent door:
	// a DeepSeek-driven REPL ("a you that isn't you"). Its own provider definition;
	// kosaten keeps its own, separately (nothing shared across the private/open line).
	base := *ollama
	apiKey := ""
	mdl := *model
	flashModel, proModel := "", ""
	routerOn := false
	switch *provider {
	case "deepseek":
		base = "https://api.deepseek.com"
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "pilot: DEEPSEEK_API_KEY not set (needed for -provider deepseek)")
			os.Exit(1)
		}
		if mdl == "" {
			flashModel = resolveDeepSeekModel(apiKey, "flash") // dynamic: ask the API what exists now
			proModel = resolveDeepSeekModel(apiKey, "pro")
			mdl = flashModel
			routerOn = true // pick flash vs pro per turn by task difficulty
		}
	case "ollama":
		if mdl == "" {
			mdl = defaultModel
		}
	default:
		fmt.Fprintf(os.Stderr, "pilot: unknown -provider %q (use deepseek|ollama)\n", *provider)
		os.Exit(1)
	}

	bin, err := resolveServer(*server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pilot: %v\n", err)
		os.Exit(1)
	}
	mcp, err := startServer(bin, *logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pilot: start tool-server: %v\n", err)
		os.Exit(1)
	}
	defer mcp.close()

	wireTools, err := mcp.handshake()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pilot: handshake: %v\n", err)
		os.Exit(1)
	}
	tools := append(wireTools, builtinTools()...) // wire + host built-ins, merged for the model

	// GENERIC MCP BRIDGES — declared in config ($PILOT_MCP or ~/.pilot/mcp.json), NOT in
	// code, so a new local/cloud MCP is added by config. The same binary runs anywhere:
	// each server that isn't reachable is skipped (office has local ltqa + hosted kosaten;
	// omarchy has local kosaten; a stranger has their own — no code change for any of it).
	// -kosaten is kept as a built-in fallback spec when no config file exists.
	bridges := map[string]bridge{} // "<name>_" prefix -> bridged server (HTTP or stdio)
	for _, sp := range loadMCPSpecs(*kosatenURL) {
		h, mtools, err := startMCP(sp, *logPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[2mpilot: mcp %q off (%v)\033[0m\n", sp.Name, err)
			continue
		}
		bridges[sp.Name+"_"] = h
		tools = append(tools, mtools...)
		where := sp.URL
		if where == "" {
			where = sp.Command
		}
		fmt.Fprintf(os.Stderr, "\033[2mmcp: bridged %d tools from %s (%s)\033[0m\n", len(mtools), sp.Name, where)
	}

	sysPrompt := system
	if *provider == "deepseek" {
		sysPrompt += "\n\n" + deepseekCapabilities
	}
	msgs := []message{{Role: "system", Content: sysPrompt}}
	if *resume != "" { // /redeploy handoff: carry the conversation into the new build
		if b, e := os.ReadFile(*resume); e == nil {
			var loaded []message
			if json.Unmarshal(b, &loaded) == nil {
				for _, m := range loaded {
					if m.Role == "system" {
						continue // refresh to the new build's system prompt
					}
					msgs = append(msgs, m)
				}
				fmt.Fprintf(os.Stderr, "\033[2mresumed — %d messages carried across the redeploy\033[0m\n", len(msgs)-1)
			}
			os.Remove(*resume)
		}
	}
	repl(&session{
		base: base, model: mdl, provider: *provider, apiKey: apiKey,
		flashModel: flashModel, proModel: proModel, router: routerOn,
		mcp: mcp, bridges: bridges, castleFile: *castle, tools: tools, maxSteps: *maxSteps,
		showThinking: *showThinking, autoYes: *autoYes, lean: *lean, logPath: *logPath,
		noReadline: *noReadline,
		msgs: msgs,
	}, toolNames(tools), bin)
}

// ---- conversation ----

type session struct {
	base, model  string
	provider     string // "deepseek" | "ollama"
	apiKey       string // for the deepseek (OpenAI-compatible) provider
	flashModel   string // deepseek router: the cheap/fast model
	proModel     string // deepseek router: the strong/thinking model
	router       bool   // choose flash vs pro per turn by task difficulty
	turnModel    string // model chosen for the current turn
	turnThinking bool   // whether to enable reasoning this turn
	turnEffort   string // reasoning_effort when thinking
	mcp          *mcpServer
	bridges      map[string]bridge   // "<name>_" prefix -> bridged MCP (HTTP or stdio), config-driven
	castleFile   string   // if set, dump the session here each turn for the block-world UI
	tools        []map[string]any
	turnTools    []map[string]any // tools offered this turn (scoped when lean)
	maxSteps     int
	showThinking bool
	autoYes      bool
	lean         bool // host-side curation: gate probe/channel tools behind intent
	logPath      string
	interactive  bool
	in           *bufio.Reader
	noReadline   bool              // disable readline even when interactive
	lr           *lineReader       // raw-mode line editor with bracketed paste (nil when piped or -no-readline)
	msgs         []message         // grows across turns; the dialogue is the state
}

func repl(s *session, names []string, bin string) {
	s.interactive = isTTY(os.Stdin)
	if s.interactive && !s.noReadline {
		home, _ := os.UserHomeDir()
		s.lr = newLineReader(filepath.Join(home, ".pilot_history"))
	} else {
		s.in = bufio.NewReader(os.Stdin)
	}
	if s.interactive {
		fmt.Fprintf(os.Stderr, "\033[2mpilot · model %s · %d tools %v\033[0m\n", s.model, len(names), names)
		fmt.Fprintf(os.Stderr, "\033[2mwire: %s · tool-server logs → %s\033[0m\n", filepath.Base(bin), s.logPath)
		fmt.Fprintf(os.Stderr, "\033[2mjust talk. /exit to leave. (run_command + write_file ask before running)\033[0m\n")
		if s.lr != nil {
			fmt.Fprintf(os.Stderr, "\033[2mreadline: ↑ history, Home/End, Ctrl+A/E/U/K/W — paste multi-line safely (Enter fires only on your keypress)\033[0m\n")
		}
	}
	for {
		var line string
		var err error
		if s.lr != nil {
			line, err = s.lr.readLine("you ❯ ", true)
		} else {
			if s.interactive {
				fmt.Print("\033[1myou ❯\033[0m ")
			}
			line, err = s.in.ReadString('\n')
		}
		if err != nil {
			if err == errAborted { // Ctrl-C aborts the line, not the session
				continue
			}
			if err != io.EOF {
				fmt.Fprintf(os.Stderr, "\033[2mpilot: readline closed (%v)\033[0m\n", err)
			}
			return // EOF (Ctrl-D) or a real readline failure
		}
		line = strings.TrimSpace(line)
		if line != "" {
			switch line {
			case "/redeploy":
				s.redeploy() // rebuild + re-exec in place; returns only on failure
				continue
			case "/exit", "/quit", "/bye":
				if s.lr != nil {
					s.lr.saveHistory()
				}
				return
			}
			s.turn(line)
		}
	}
}

// turn appends the user's line and lets the model run until it answers without
// reaching for a tool. Tool activity is shown dimmed so the dialogue stays front
// and center.
func (s *session) turn(userLine string) {
	s.msgs = append(s.msgs, message{Role: "user", Content: userLine})

	// Route this turn to the right brain: pro+thinking for hard/analytic work,
	// flash for lookups and simple actions — think harder only when it's earned.
	s.route(userLine)
	if s.interactive && s.provider == "deepseek" {
		tag := "flash"
		if s.turnThinking {
			tag = "pro · thinking"
		}
		fmt.Fprintf(os.Stderr, "\033[2m· %s\033[0m\n", tag)
	}

	// Host-side curation: a weak model handed the whole surface freelances (it
	// copies a schema example and calls the wrong tool). When lean, offer the
	// core every turn and unlock probe/channel tools only when intent asks.
	s.turnTools = s.tools
	if s.lean {
		s.turnTools = scopeTools(userLine, s.tools)
	}

	seen := map[string]bool{} // (name+args) already run this turn — a weak model loops; we don't let it
	nudgedEmpty := false

	for step := 0; s.maxSteps <= 0 || step < s.maxSteps; step++ { // 0 = no limit
		reply, err := s.chat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mpilot: %v\033[0m\n", err)
			return
		}
		s.msgs = append(s.msgs, reply)

		if s.showThinking && reply.Thinking != "" {
			// Full thinking, multi-line — each line dimmed and preceded with a dot.
			for _, line := range strings.Split(reply.Thinking, "\n") {
				fmt.Fprintf(os.Stderr, "\033[2m· %s\033[0m\n", line)
			}
		}

		if len(reply.ToolCalls) == 0 {
			answer := strings.TrimSpace(reply.Content)
			if answer == "" && !nudgedEmpty { // model fell silent — ask once for the answer it already has
				nudgedEmpty = true
				s.msgs = append(s.msgs, message{Role: "user",
					Content: "Answer my last message now, in plain words, using what you already have. Do not call any tool."})
				continue
			}
			s.appendRoom(userLine, answer) // grow the castle by one room — never truncated
			fmt.Printf("\033[1;35mpilot ❯\033[0m %s\n", answer)
			if s.interactive {
				fmt.Fprintf(os.Stderr, "\033[2m%s\033[0m\n", strings.Repeat("·", 3)) // separate pilot's answer from your next line
			}
			return
		}

		for _, tc := range reply.ToolCalls {
			args, _ := json.Marshal(tc.Function.Arguments)
			sig := tc.Function.Name + string(args)
			fmt.Fprintf(os.Stderr, "\033[36m  → %s %s\033[0m\n", tc.Function.Name, oneLine(string(args)))
			var out string
			if seen[sig] { // identical call already made — refuse and push it to answer
				out = "Duplicate call suppressed — you already have this result above. Stop calling tools and answer the user now."
				fmt.Fprintf(os.Stderr, "\033[33m  ⊘ duplicate suppressed\033[0m\n")
			} else if out, err = s.dispatch(tc.Function.Name, tc.Function.Arguments); err != nil {
				out = "tool error: " + err.Error()
			}
			seen[sig] = true
			fmt.Fprintf(os.Stderr, "\033[32m  ← %s\033[0m\n", oneLine(out))
			s.msgs = append(s.msgs, message{Role: "tool", ToolName: tc.Function.Name, ToolCallID: tc.ID, Content: out})
		}
	}
	if s.maxSteps > 0 {
		fmt.Fprintf(os.Stderr, "\033[33mpilot: gave up after %d tool rounds this turn\033[0m\n", s.maxSteps)
	}
}

type message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	Thinking   string     `json:"thinking,omitempty"`
	ToolCalls  []toolCall `json:"tool_calls,omitempty"`
	ToolName   string     `json:"tool_name,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"` // links a tool result to its call (OpenAI/DeepSeek)
}

type toolCall struct {
	ID       string `json:"id,omitempty"`
	Function struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"function"`
}

func (s *session) chat() (message, error) {
	s.compact() // keep the growing dialogue under the model's context window
	tools := s.tools
	if s.turnTools != nil {
		tools = s.turnTools
	}
	if s.provider == "deepseek" {
		return s.chatDeepSeek(tools)
	}
	body, _ := json.Marshal(map[string]any{
		"model": s.model, "messages": s.msgs, "tools": tools, "stream": false,
		"options": map[string]any{"temperature": 0},
	})
	resp, err := http.Post(s.base+"/api/chat", "application/json", bytes.NewReader(body))
	if err != nil {
		return message{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return message{}, fmt.Errorf("ollama %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out struct {
		Message message `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return message{}, err
	}
	return out.Message, nil
}

// chatDeepSeek talks the OpenAI-compatible /chat/completions format (DeepSeek).
// It translates pilot's neutral message log to/from OpenAI's shape: assistant
// tool_calls carry ids, tool results reference tool_call_id, and arguments cross
// the wire as JSON strings (not objects, as Ollama uses). Same loop/tools/dispatch.
func (s *session) chatDeepSeek(tools []map[string]any) (message, error) {
	oa := make([]map[string]any, 0, len(s.msgs))
	for _, m := range s.msgs {
		switch m.Role {
		case "tool":
			oa = append(oa, map[string]any{"role": "tool", "tool_call_id": m.ToolCallID, "content": m.Content})
		case "assistant":
			am := map[string]any{"role": "assistant", "content": m.Content}
			if len(m.ToolCalls) > 0 {
				tcs := make([]map[string]any, len(m.ToolCalls))
				for i, tc := range m.ToolCalls {
					ab, _ := json.Marshal(tc.Function.Arguments)
					tcs[i] = map[string]any{"id": tc.ID, "type": "function",
						"function": map[string]any{"name": tc.Function.Name, "arguments": string(ab)}}
				}
				am["tool_calls"] = tcs
			}
			oa = append(oa, am)
		default: // system, user
			oa = append(oa, map[string]any{"role": m.Role, "content": m.Content})
		}
	}
	model := s.turnModel
	if model == "" {
		model = s.model
	}
	reqBody := map[string]any{"model": model, "messages": oa, "stream": false}
	if len(tools) > 0 {
		reqBody["tools"] = tools
	}
	if s.turnThinking {
		reqBody["thinking"] = map[string]string{"type": "enabled"}
		reqBody["reasoning_effort"] = s.turnEffort
	}
	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", s.base+"/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return message{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return message{}, fmt.Errorf("deepseek %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				Reasoning string `json:"reasoning_content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return message{}, err
	}
	if len(out.Choices) == 0 {
		return message{}, fmt.Errorf("deepseek: empty choices")
	}
	c := out.Choices[0].Message
	msg := message{Role: "assistant", Content: c.Content, Thinking: c.Reasoning}
	for _, tc := range c.ToolCalls {
		var args map[string]any
		if tc.Function.Arguments != "" {
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
		}
		if args == nil {
			args = map[string]any{}
		}
		var call toolCall
		call.ID = tc.ID
		call.Function.Name = tc.Function.Name
		call.Function.Arguments = args
		msg.ToolCalls = append(msg.ToolCalls, call)
	}
	return msg, nil
}

// resolveDeepSeekModel asks the API which models exist now and prefers the cheap
// flash tier — pilot's own dynamic resolution so no model name is hardcoded
// (kosaten keeps its own, separate resolver). Falls back to v4-flash if offline.
func resolveDeepSeekModel(apiKey, role string) string {
	fallback, prefer := "deepseek-v4-flash", []string{"flash", "chat"}
	if role == "pro" {
		fallback, prefer = "deepseek-v4-pro", []string{"pro", "reasoner"}
	}
	req, _ := http.NewRequest("GET", "https://api.deepseek.com/models", nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)
	resp, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
	if err != nil {
		return fallback
	}
	defer resp.Body.Close()
	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&out) != nil || len(out.Data) == 0 {
		return fallback
	}
	for _, kw := range prefer {
		for _, m := range out.Data {
			if strings.Contains(m.ID, kw) {
				return m.ID
			}
		}
	}
	return out.Data[0].ID
}

// route picks this turn's brain: v4-pro (with thinking) for hard, multi-step, or
// reasoning/design/analysis work; v4-flash (no thinking) for lookups and simple
// actions. This is how pilot spends thought only when the task earns it — the same
// discipline I use. Escalation is possible later (retry pro if flash stalls).
func (s *session) route(userLine string) {
	s.turnModel, s.turnThinking, s.turnEffort = s.flashModel, false, ""
	if !s.router {
		s.turnModel = s.model
		return
	}
	l := strings.ToLower(userLine)
	hard := len(userLine) > 240
	for _, kw := range []string{"why", "design", "architect", "debate", "analyz", "compare",
		"plan", "refactor", "strateg", "trade-off", "tradeoff", "decide", "evaluate", "reason",
		"prove", "root cause", "orchestrat", "redesign", "critique", "explain how", "step by step",
		"figure out", "think through", "weigh", "implications", "consequence"} {
		if strings.Contains(l, kw) {
			hard = true
			break
		}
	}
	if hard {
		s.turnModel, s.turnThinking, s.turnEffort = s.proModel, true, "high"
	}
}

// compact keeps the growing dialogue under the model's context window. Tool
// outputs are clipped per-call, but a long session still accumulates; when the
// log passes the budget we drop the OLDEST whole rounds (a round = a user turn
// plus its assistant/tool exchange), never the current one — so tool_call/result
// pairs stay intact for the OpenAI/DeepSeek message format.
func (s *session) compact() {
	const budget = 300_000 // chars — safely under a 1M-token window
	size := func() int {
		n := 0
		for _, m := range s.msgs {
			n += len(m.Content)
			for _, tc := range m.ToolCalls {
				b, _ := json.Marshal(tc.Function.Arguments)
				n += len(tc.Function.Name) + len(b)
			}
		}
		return n
	}
	for size() > budget {
		cut := -1
		for i := 2; i < len(s.msgs); i++ { // skip system[0]+first user[1]; find the next round's user
			if s.msgs[i].Role == "user" {
				cut = i
				break
			}
		}
		if cut < 0 {
			return // only the current round remains — don't break its exchange
		}
		s.msgs = append(s.msgs[:1:1], s.msgs[cut:]...) // keep system + from the next round on
	}
}

// ---- minimal MCP client over the tool-server's stdio (JSON-RPC 2.0, NDJSON) ----

type mcpServer struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
	id  int
}

func startServer(path, logPath string) (*mcpServer, error) {
	cmd := exec.Command(path)
	cmd.Stderr = os.Stderr // fallback if the log file can't be opened
	if logPath != "" {
		if lf, e := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); e == nil {
			cmd.Stderr = lf // keep the wire's chatter out of the chat
		}
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &mcpServer{cmd: cmd, in: in, out: bufio.NewReader(out)}, nil
}

func (m *mcpServer) close() {
	m.in.Close()
	_ = m.cmd.Wait()
}

// rpc sends one request and returns its result, skipping any line whose id
// doesn't match (notifications, out-of-band frames). The server answers one
// request at a time, so request/response stays in lockstep.
func (m *mcpServer) rpc(method string, params any) (json.RawMessage, error) {
	m.id++
	req := map[string]any{"jsonrpc": "2.0", "id": m.id, "method": method}
	if params != nil {
		req["params"] = params
	}
	b, _ := json.Marshal(req)
	if _, err := m.in.Write(append(b, '\n')); err != nil {
		return nil, err
	}
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		line, err := m.out.ReadBytes('\n')
		if err != nil {
			return nil, err
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		var r struct {
			ID     json.RawMessage `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int    `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(line, &r) != nil {
			continue
		}
		var gotID int
		if len(r.ID) > 0 {
			json.Unmarshal(r.ID, &gotID)
		}
		if gotID != m.id {
			continue
		}
		if r.Error != nil {
			return nil, fmt.Errorf("rpc %s: %d %s", method, r.Error.Code, r.Error.Message)
		}
		return r.Result, nil
	}
	return nil, fmt.Errorf("rpc %s: timeout", method)
}

func (m *mcpServer) notify(method string) {
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": method})
	m.in.Write(append(b, '\n'))
}

// handshake initializes the session and returns the tool surface already shaped
// for Ollama's /api/chat tools field.
func (m *mcpServer) handshake() ([]map[string]any, error) {
	if _, err := m.rpc("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "pilot", "version": "0.1.0"},
	}); err != nil {
		return nil, err
	}
	m.notify("notifications/initialized")

	res, err := m.rpc("tools/list", nil)
	if err != nil {
		return nil, err
	}
	var tl struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema any    `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(res, &tl); err != nil {
		return nil, err
	}
	tools := make([]map[string]any, 0, len(tl.Tools))
	for _, t := range tl.Tools {
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        t.Name,
				"description": t.Description,
				"parameters":  t.InputSchema,
			},
		})
	}
	return tools, nil
}

// callTool runs one tool and flattens the MCP content array to text.
func (m *mcpServer) callTool(name string, args map[string]any) (string, error) {
	res, err := m.rpc("tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		return "", err
	}
	var r struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(res, &r); err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, c := range r.Content {
		sb.WriteString(c.Text)
	}
	out := sb.String()
	if r.IsError {
		return "", fmt.Errorf("%s", out)
	}
	return out, nil
}

// redeploy rebuilds pilot and re-execs the new binary IN PLACE — same PID, same
// terminal — so the conversation continues seamlessly. This is as close to
// zero-downtime as a single Go process can get: new code must be compiled (the
// only pause), but syscall.Exec then replaces this process's image without ever
// dropping the tty or spawning a new process, and the dialogue is handed to the
// new build via a -resume file. On build failure it stays on the running binary.
func (s *session) redeploy() {
	dir := pilotDir()
	fmt.Fprintf(os.Stderr, "\033[2mredeploy: rebuilding %s …\033[0m\n", dir)
	build := exec.Command("go", "build", "-o", "pilot", ".")
	build.Dir = dir
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "\033[31mredeploy: BUILD FAILED — staying on the running binary:\n%s\033[0m\n", strings.TrimSpace(string(out)))
		return
	}
	sessFile := filepath.Join(os.TempDir(), fmt.Sprintf("pilot-session-%d.json", os.Getpid()))
	if b, err := json.Marshal(s.msgs); err == nil {
		_ = os.WriteFile(sessFile, b, 0o600)
	}
	bin := filepath.Join(dir, "pilot")
	argv := append([]string{bin}, execArgsWithResume(sessFile)...)
	// hand off the terminal cleanly, then replace this process image in place.
	if s.mcp != nil && s.mcp.cmd != nil && s.mcp.cmd.Process != nil {
		_ = s.mcp.cmd.Process.Kill() // the new build spawns its own tool-server
	}
	if s.lr != nil {
		s.lr.saveHistory() // persist before re-exec (the terminal is already restored per readLine)
	}
	fmt.Fprintln(os.Stderr, "\033[2mredeploy: re-exec into the new binary (same terminal, session preserved) …\033[0m")
	if err := syscall.Exec(bin, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "\033[31mredeploy: exec failed: %v\033[0m\n", err)
	}
}

// appendRoom grows the block-world by one room — append-only, never truncated,
// no cap on rooms or territory. The compaction in s.msgs keeps only the model's
// render-distance; this file keeps the whole world (the memory that never drops).
func (s *session) appendRoom(user, answer string) {
	if s.castleFile == "" {
		return
	}
	rec, _ := json.Marshal(map[string]string{"user": user, "answer": answer})
	if f, err := os.OpenFile(s.castleFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); err == nil {
		f.Write(append(rec, '\n'))
		f.Close()
	}
}

// pilotDir is where pilot's source + binary live (the build target).
func pilotDir() string {
	if d := os.Getenv("PILOT_DIR"); d != "" {
		return d
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe)
	}
	return "/home/rishi/Work/pilot"
}

// execArgsWithResume preserves the current launch flags across the re-exec and
// swaps in a fresh -resume file (dropping any stale one).
func execArgsWithResume(sessFile string) []string {
	var out []string
	skip := false
	for _, a := range os.Args[1:] {
		if skip {
			skip = false
			continue
		}
		if a == "-resume" || a == "--resume" {
			skip = true
			continue
		}
		if strings.HasPrefix(a, "-resume=") || strings.HasPrefix(a, "--resume=") {
			continue
		}
		out = append(out, a)
	}
	return append(out, "-resume", sessFile)
}

// ---- native kosaten MCP client over HTTP (JSON-RPC 2.0, streamable-HTTP) ----
//
// The council said BUILD NOW; pilot (thinking) said bridge a LEAN surface, not all
// 88 tools — "don't hard-code the debt." So pilot connects to kosaten's :3942 MCP
// and exposes only a curated set of the organism's voice/memory/judgment tools,
// prefixed "kosaten_" (its own file/shell tools would collide with pilot's hands).
// Auth via KOSATEN_API_KEY; initialize yields an Mcp-Session-Id carried per call.

// kosatenTools — the lean allowlist (pilot's "don't import the bloat" verdict):
// memory, letters, findings, causal, debate, delegation, proprioception.
var kosatenTools = map[string]bool{
	"know_me": true, "read_letter": true, "write_letter": true, "recall_decision": true,
	"search_context": true, "list_findings": true, "search_findings": true,
	"record_conclusion": true, "list_conclusions": true, "delegate_work": true,
	"causal_query": true, "run_debate": true, "get_health": true, "universe_status": true,
	"get_pulse": true, "digest": true,
}

// mcpSpec declares one MCP server to bridge — in CONFIG, not code. Two transports:
// HTTP (set url; key_env names a bearer-token env var) or stdio (set command/args/cwd/
// env, like a `uv run …` server). tools is an optional allowlist ([] = expose all).
// name becomes the "<name>_" tool prefix.
type mcpSpec struct {
	Name    string            `json:"name"`
	URL     string            `json:"url"`     // HTTP transport
	KeyEnv  string            `json:"key_env"` // env var with the bearer token (HTTP)
	Command string            `json:"command"` // stdio transport
	Args    []string          `json:"args"`
	Cwd     string            `json:"cwd"`
	Env     map[string]string `json:"env"`
	Tools   []string          `json:"tools"`
}

// bridge is either transport — anything that can run a tool by name. httpMCP (HTTP) and
// mcpServer (stdio) both satisfy it, so call-routing doesn't care which a server uses.
type bridge interface {
	callTool(name string, args map[string]any) (string, error)
}

// loadMCPSpecs reads the bridge list from $PILOT_MCP (or ~/.pilot/mcp.json). This is how
// a new local/cloud MCP is added — edit config, not pilot. If no config file exists it
// falls back to a single kosaten spec from -kosaten (backward compat). Unreachable
// servers are skipped by the caller, so one config works on every machine.
func loadMCPSpecs(kosatenURL string) []mcpSpec {
	var paths []string
	if p := os.Getenv("PILOT_MCP"); p != "" {
		paths = append(paths, p)
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".pilot", "mcp.json"))
	}
	for _, p := range paths {
		if b, err := os.ReadFile(p); err == nil {
			var specs []mcpSpec
			if json.Unmarshal(b, &specs) == nil && len(specs) > 0 {
				return specs
			}
		}
	}
	if kosatenURL != "" { // fallback: the built-in kosaten default (lean allowlist)
		allow := make([]string, 0, len(kosatenTools))
		for k := range kosatenTools {
			allow = append(allow, k)
		}
		return []mcpSpec{{Name: "kosaten", URL: kosatenURL, KeyEnv: "KOSATEN_API_KEY", Tools: allow}}
	}
	return nil
}

type httpMCP struct {
	url, key, sid string
	id            int
}

func (h *httpMCP) rpc(method string, params any) (json.RawMessage, string, error) {
	h.id++
	body := map[string]any{"jsonrpc": "2.0", "id": h.id, "method": method}
	if params != nil {
		body["params"] = params
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", h.url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if h.key != "" {
		req.Header.Set("Authorization", "Bearer "+h.key)
	}
	if h.sid != "" {
		req.Header.Set("Mcp-Session-Id", h.sid)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	sid := resp.Header.Get("Mcp-Session-Id")
	raw, _ := io.ReadAll(resp.Body)
	line := bytes.TrimSpace(raw)
	if i := bytes.Index(line, []byte("data:")); i >= 0 { // unwrap SSE framing
		line = bytes.TrimSpace(line[i+5:])
	}
	var r struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(line, &r) != nil {
		return nil, sid, fmt.Errorf("kosaten-mcp %s: bad response", method)
	}
	if r.Error != nil {
		return nil, sid, fmt.Errorf("kosaten-mcp %s: %s", method, r.Error.Message)
	}
	return r.Result, sid, nil
}

func (h *httpMCP) notify(method string) {
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": method})
	req, _ := http.NewRequest("POST", h.url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if h.key != "" {
		req.Header.Set("Authorization", "Bearer "+h.key)
	}
	if h.sid != "" {
		req.Header.Set("Mcp-Session-Id", h.sid)
	}
	if resp, err := http.DefaultClient.Do(req); err == nil {
		resp.Body.Close()
	}
}

// mcpTool is a tool as the server advertises it (before pilot's "<name>_" prefix).
type mcpTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// listTools runs initialize + tools/list on an HTTP MCP and returns the raw tools.
func (h *httpMCP) listTools() ([]mcpTool, error) {
	_, sid, err := h.rpc("initialize", map[string]any{
		"protocolVersion": "2024-11-05", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "pilot", "version": "0.1.0"},
	})
	if err != nil {
		return nil, err
	}
	h.sid = sid
	h.notify("notifications/initialized")
	res, _, err := h.rpc("tools/list", nil)
	if err != nil {
		return nil, err
	}
	var tl struct {
		Tools []mcpTool `json:"tools"`
	}
	if err := json.Unmarshal(res, &tl); err != nil {
		return nil, err
	}
	return tl.Tools, nil
}

// startStdio spawns a stdio MCP server (command/args/cwd/env from the spec) — the same
// mcpServer used for the http-mcp tool-server, reused for any config-declared stdio MCP
// (e.g. `uv run --directory … python -m mcp_server.server`).
func startStdio(sp mcpSpec, logPath string) (*mcpServer, error) {
	cmd := exec.Command(sp.Command, sp.Args...)
	cmd.Dir = sp.Cwd
	cmd.Env = os.Environ()
	for k, v := range sp.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Stderr = os.Stderr
	if logPath != "" {
		if lf, e := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644); e == nil {
			cmd.Stderr = lf
		}
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &mcpServer{cmd: cmd, in: in, out: bufio.NewReader(out)}, nil
}

// listTools runs initialize + tools/list on a stdio MCP and returns the raw tools.
func (m *mcpServer) listTools() ([]mcpTool, error) {
	if _, err := m.rpc("initialize", map[string]any{
		"protocolVersion": "2024-11-05", "capabilities": map[string]any{},
		"clientInfo": map[string]any{"name": "pilot", "version": "0.1.0"},
	}); err != nil {
		return nil, err
	}
	m.notify("notifications/initialized")
	res, err := m.rpc("tools/list", nil)
	if err != nil {
		return nil, err
	}
	var tl struct {
		Tools []mcpTool `json:"tools"`
	}
	if err := json.Unmarshal(res, &tl); err != nil {
		return nil, err
	}
	return tl.Tools, nil
}

// startMCP bridges one MCP server (HTTP or stdio, chosen by which spec fields are set)
// and returns a client + its tools, each prefixed with sp.Name+"_" and optionally
// allow-listed. Adding a server is config (loadMCPSpecs), not code.
func startMCP(sp mcpSpec, logPath string) (bridge, []map[string]any, error) {
	var raw []mcpTool
	var b bridge
	if sp.Command != "" {
		m, err := startStdio(sp, logPath)
		if err != nil {
			return nil, nil, err
		}
		if raw, err = m.listTools(); err != nil {
			m.close()
			return nil, nil, err
		}
		b = m
	} else {
		h := &httpMCP{url: sp.URL, key: os.Getenv(sp.KeyEnv)}
		var err error
		if raw, err = h.listTools(); err != nil {
			return nil, nil, err
		}
		b = h
	}
	allow := map[string]bool{}
	for _, t := range sp.Tools {
		allow[t] = true
	}
	var tools []map[string]any
	for _, t := range raw {
		if len(allow) > 0 && !allow[t.Name] {
			continue // allowlist ([] = expose all)
		}
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object", "properties": map[string]any{}}
		}
		tools = append(tools, map[string]any{
			"type": "function",
			"function": map[string]any{
				"name":        sp.Name + "_" + t.Name,
				"description": "[" + sp.Name + "] " + t.Description,
				"parameters":  schema,
			},
		})
	}
	return b, tools, nil
}

func (h *httpMCP) callTool(name string, args map[string]any) (string, error) {
	res, _, err := h.rpc("tools/call", map[string]any{"name": name, "arguments": args})
	if err != nil {
		return "", err
	}
	var r struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(res, &r); err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, c := range r.Content {
		sb.WriteString(c.Text)
	}
	return sb.String(), nil
}

// ---- built-in host tools (shell + filesystem) ----
//
// These live in pilot, not in the wire. The wire stays a pure HTTP server; the
// host is where "act on this machine" belongs — same split Claude Code uses
// (Bash + Read/Write are the harness's, not a remote service's).

func builtinTools() []map[string]any {
	str := func(d string) map[string]any { return map[string]any{"type": "string", "description": d} }
	fn := func(name, desc string, props map[string]any, req ...any) map[string]any {
		if req == nil {
			req = []any{} // DeepSeek/OpenAI reject "required": null — must be an array
		}
		return map[string]any{"type": "function", "function": map[string]any{
			"name": name, "description": desc,
			"parameters": map[string]any{"type": "object", "properties": props, "required": req},
		}}
	}
	return []map[string]any{
		fn("run_command",
			"Run a shell command on this machine (sh -c) and return its combined stdout/stderr and exit code. Use it for adb, git, ls, idevice_id — anything you could type in a terminal. This controls real local devices and the machine itself.",
			map[string]any{
				"command":         str("The shell command line to run."),
				"timeout_seconds": map[string]any{"type": "integer", "description": "Kill the command after this many seconds (default 60)."},
			}, "command"),
		fn("read_file", "Read a text file from this Mac and return its contents (first 100 KB).",
			map[string]any{"path": str("Absolute or relative file path.")}, "path"),
		fn("write_file", "Create or overwrite a text file on this machine.",
			map[string]any{"path": str("File path to write."), "content": str("Full text to write.")}, "path", "content"),
		fn("list_dir", "List the entries of a directory on this machine.",
			map[string]any{"path": str("Directory path (defaults to the current directory).")}),
	}
}

// scopeTools is curation relocated to the host. The wire stays lean and offers
// every atom; the host decides which to put in front of the model this turn. A
// weak model handed the full surface freelances — it copies the example in a
// tool's schema and calls the wrong one (we watched a 12B do exactly that with
// discover). So we gate the rarely-needed, easily-misused probe/channel atoms
// behind explicit intent, and always offer the safe core. Off by default
// (-lean): strong models want the whole surface; weak ones want the floor raised.
func scopeTools(userLine string, all []map[string]any) []map[string]any {
	gated := map[string]bool{"discover": true, "bidi_command": true}
	intent := strings.ToLower(userLine)
	for _, kw := range []string{
		"discover", "probe", "capabilit", "endpoint", "which command", "what command",
		"bidi", "cdp", "devtools", "subscribe", "channel", "session caps", "websocket",
	} {
		if strings.Contains(intent, kw) {
			return all // intent asks for the deep surface — unlock everything
		}
	}
	out := make([]map[string]any, 0, len(all))
	for _, t := range all {
		name, _ := t["function"].(map[string]any)["name"].(string)
		if !gated[name] {
			out = append(out, t)
		}
	}
	return out
}

// dispatch routes a tool call: built-ins run here in Go; everything else goes to
// the wire. Mutating built-ins ask the user first (unless -yes).
func (s *session) dispatch(name string, args map[string]any) (string, error) {
	switch name {
	case "run_command", "write_file":
		if !s.confirm(name) {
			return "User declined to run this " + name + ". Do not retry it; ask the user how to proceed.", nil
		}
	}
	if out, ok := s.callBuiltin(name, args); ok {
		return out, nil
	}
	// Bridged MCP tools route to the right server by their "<name>_" prefix — generic,
	// so any config-declared MCP (kosaten, ltqa-local, a client's own) works with no
	// code change here.
	for prefix, h := range s.bridges {
		if strings.HasPrefix(name, prefix) {
			out, err := h.callTool(strings.TrimPrefix(name, prefix), args)
			if err != nil {
				return "", err
			}
			return clip(out, 24000), nil
		}
	}
	// Clip wire/MCP results too — an unclipped http_request body (e.g. a big JSON
	// endpoint) accumulates in the message log and blows the model's context window.
	out, err := s.mcp.callTool(name, args)
	if err != nil {
		return "", err
	}
	return clip(out, 24000), nil
}

func (s *session) callBuiltin(name string, args map[string]any) (string, bool) {
	switch name {
	case "run_command":
		cmdline := argStr(args, "command")
		if cmdline == "" {
			return "error: command is required", true
		}
		secs := 60
		if v, ok := args["timeout_seconds"].(float64); ok && v > 0 {
			secs = int(v)
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(secs)*time.Second)
		defer cancel()
		out, _ := exec.CommandContext(ctx, "sh", "-c", cmdline).CombinedOutput()
		res := string(out)
		if ctx.Err() == context.DeadlineExceeded {
			res += fmt.Sprintf("\n[killed: exceeded %ds timeout]", secs)
		}
		return clip(res, 16000), true
	case "read_file":
		p := argStr(args, "path")
		b, err := os.ReadFile(p)
		if err != nil {
			return "error: " + err.Error(), true
		}
		return clip(string(b), 100_000), true
	case "write_file":
		p := argStr(args, "path")
		if err := os.WriteFile(p, []byte(argStr(args, "content")), 0o644); err != nil {
			return "error: " + err.Error(), true
		}
		return "wrote " + p, true
	case "list_dir":
		p := argStr(args, "path")
		if p == "" {
			p = "."
		}
		ents, err := os.ReadDir(p)
		if err != nil {
			return "error: " + err.Error(), true
		}
		var sb strings.Builder
		for _, e := range ents {
			kind := "f"
			if e.IsDir() {
				kind = "d"
			}
			fmt.Fprintf(&sb, "%s  %s\n", kind, e.Name())
		}
		return clip(sb.String(), 16000), true
	}
	return "", false // not a built-in — caller falls through to the wire
}

// confirm asks before a mutating action. Uses readline when available,
// falls back to raw stdin otherwise.
func (s *session) confirm(action string) bool {
	if s.autoYes {
		return true
	}
	if !s.interactive {
		return false
	}
	prompt := fmt.Sprintf("\033[33mallow %s? [y/N]\033[0m ", action)
	var a string
	if s.lr != nil {
		line, err := s.lr.readLine(prompt, false)
		if err != nil {
			return false
		}
		a = strings.ToLower(strings.TrimSpace(line))
		// Don't save confirm prompts to history
	} else {
		fmt.Print(prompt)
		line, _ := s.in.ReadString('\n')
		a = strings.ToLower(strings.TrimSpace(line))
	}
	return a == "y" || a == "yes"
}

func argStr(m map[string]any, k string) string {
	if s, ok := m[k].(string); ok {
		return s
	}
	return ""
}

func clip(s string, n int) string {
	if len(s) > n {
		return s[:n] + fmt.Sprintf("\n…[truncated, %d bytes total]", len(s))
	}
	return s
}

// ---- helpers ----

// resolveServer finds the http-mcp binary: explicit flag, then $HTTP_MCP_BIN,
// then PATH, then the sibling-repo convention (repos/pilot + repos/http-mcp).
func resolveServer(flagVal string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	if env := os.Getenv("HTTP_MCP_BIN"); env != "" {
		return env, nil
	}
	if p, err := exec.LookPath("http-mcp"); err == nil {
		return p, nil
	}
	cands := []string{"../http-mcp/http-mcp", "./http-mcp"}
	if exe, err := os.Executable(); err == nil {
		cands = append(cands, filepath.Join(filepath.Dir(exe), "..", "http-mcp", "http-mcp"))
	}
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("http-mcp binary not found — build it and pass -server <path> or set HTTP_MCP_BIN")
}

func toolNames(tools []map[string]any) []string {
	n := make([]string, len(tools))
	for i, t := range tools {
		n[i] = t["function"].(map[string]any)["name"].(string)
	}
	return n
}

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func oneLine(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}
