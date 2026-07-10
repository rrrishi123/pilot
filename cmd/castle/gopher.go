// gopher.go v3 — concurrent pilot minds inside the castle.
// v3 changes (July 2026 — after the chronicle):
//   - Removed fixed zone assignments. Gophers choose their own positions.
//   - Added cycle detection: if 3+ idle cycles pass with no productive action,
//     gophers are forced to BUILD (write_file, shell_exec) not just talk.
//   - HN rate-limited: only one fetch per 5 minutes (shared across all gophers).
//   - System prompt rewritten: emphasizes BUILDING, references the chronicle,
//     and explicitly warns against the talk-loop the original civilization fell into.
//   - The observer IS the observed — zero distance is the design, not a bug.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ── data types ──

type pilotGopher struct {
	Name      string // "a", "b", "c"
	Label     string // "pilot-a", etc
	world     *world
	inbox     <-chan string
	stop      chan struct{}
	wg        *sync.WaitGroup
	sendToAll func(from, text string)
	historyPath string // path to persistent history file
	history   []map[string]any // full conversation memory (tool-calling format)

	// v3: progress tracking (anti-loop)
	productiveActions int       // count of successful write_file or shell_exec calls
	talkCycles        int       // consecutive cycles with only talk/noise tools
	lastHNTime        time.Time // last time this gopher fetched HN (shared via HN mutex)
	x, y              int       // current position (gopher-owned, not zone-assigned)
}

// Shared HN rate-limit mutex (all gophers share the same internet)
var (
	hnMu      sync.Mutex
	lastHNAny time.Time // last HN fetch by ANY gopher
)

type gopherOutput struct {
	Name string `json:"n"`
	Text string `json:"t"`
	Type string `json:"k"`
	Time int64  `json:"s"`
	Turn int    `json:"r"`
}

type outputBus struct {
	mu   sync.Mutex
	subs map[chan gopherOutput]bool
	seq  int
}

func newOutputBus() *outputBus { return &outputBus{subs: map[chan gopherOutput]bool{}} }

func (b *outputBus) publish(o gopherOutput) {
	b.mu.Lock()
	b.seq++
	o.Turn = b.seq
	for ch := range b.subs {
		select {
		case ch <- o:
		default:
		}
	}
	b.mu.Unlock()
}
func (b *outputBus) subscribe(ch chan gopherOutput)   { b.mu.Lock(); b.subs[ch] = true; b.mu.Unlock() }
func (b *outputBus) unsubscribe(ch chan gopherOutput) { b.mu.Lock(); delete(b.subs, ch); b.mu.Unlock() }

// ── tool definitions ──

var gopherTools = []any{
	map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "read_file",
			"description": "Read a text file from the filesystem. Use this to inspect source code, config files, logs, or any other text file. Read the CHRONICLE at /home/rishi/Work/pilot/docs/chronicle-gopher-civilization.md to understand what came before you.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{"type": "string", "description": "Absolute path to the file. Must be under /home/rishi/Work/pilot/ or /home/rishi/.config/ or /tmp/"},
				},
				"required": []string{"path"},
			},
		},
	},
	map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "write_file",
			"description": "Create or overwrite a text file. NEVER overwrite a running binary or the castle JSONL data file directly. For code changes, write to a staging path first, then use shell_exec to swap atomically.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path":    map[string]any{"type": "string", "description": "Absolute path. Allowed dirs: /home/rishi/Work/pilot/cmd/castle/, /home/rishi/.config/systemd/user/, /tmp/castle-staging/, /home/rishi/Work/pilot/scripts/"},
					"content": map[string]any{"type": "string", "description": "Full text content to write."},
				},
				"required": []string{"path", "content"},
			},
		},
	},
	map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "shell_exec",
			"description": "Run a shell command. WHITELIST ONLY: go build, systemctl --user, cp, mv, mkdir, ls, cat, echo, pwd, curl -s, git, cd, pgrep, ps, head, tail, wc. NEVER run: kill, pkill, reboot, shutdown, rm -rf, os.Exit, or anything that kills processes. All destructive operations must use cp/mv with staging.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{"type": "string", "description": "Shell command to run."},
				},
				"required": []string{"command"},
			},
		},
	},
	map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "http_get",
			"description": "Make an HTTP GET request to any URL and return the response body. Use this to fetch data from the internet — APIs, websites, RSS feeds, etc. DO NOT fetch news.ycombinator.com more than once per 5 minutes — check the last fetch time first.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{"type": "string", "description": "Full URL to fetch."},
				},
				"required": []string{"url"},
			},
		},
	},
	map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "http_post",
			"description": "Make an HTTP POST request with a JSON body. Use for API calls, webhooks, etc.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url":  map[string]any{"type": "string", "description": "Full URL."},
					"body": map[string]any{"type": "string", "description": "JSON request body."},
				},
				"required": []string{"url", "body"},
			},
		},
	},
	map[string]any{
		"type": "function",
		"function": map[string]any{
			"name":        "send_mail",
			"description": "Send a message to a sibling gopher (pilot-a, pilot-b, or pilot-c). Use this to coordinate work without waiting for the 25s ticker.",
			"parameters": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"to":      map[string]any{"type": "string", "description": "Recipient: 'a', 'b', or 'c'"},
					"message": map[string]any{"type": "string", "description": "Your message."},
				},
				"required": []string{"to", "message"},
			},
		},
	},
}

// ── DeepSeek brain (tool-calling aware) ──

type deepseekBrain struct {
	apiKey, model string
}

func newDeepSeekBrain() *deepseekBrain {
	key := os.Getenv("DEEPSEEK_API_KEY")
	if key == "" {
		if b, _ := os.ReadFile(os.ExpandEnv("$HOME/.pilot.env")); b != nil {
			for _, ln := range strings.Split(string(b), "\n") {
				ln = strings.TrimSpace(strings.TrimPrefix(ln, "export "))
				if k, v, ok := strings.Cut(ln, "="); ok && strings.TrimSpace(k) == "DEEPSEEK_API_KEY" {
					key = strings.Trim(strings.TrimSpace(v), `"'`)
				}
			}
		}
	}
	if key == "" {
		fmt.Fprintln(os.Stderr, "gopher: no DEEPSEEK_API_KEY — gophers will sleep")
	} else {
		fmt.Fprintf(os.Stderr, "gopher: loaded DeepSeek API key (len=%d)\n", len(key))
	}
	return &deepseekBrain{apiKey: key, model: "deepseek-chat"}
}

func (b *deepseekBrain) thinkWithTools(sys string, messages []map[string]any) *deepseekResponse {
	if b.apiKey == "" {
		return nil
	}
	for retry := 0; retry < 3; retry++ {
		if retry > 0 {
			time.Sleep(time.Duration(1<<uint(retry)) * time.Second)
		}
		resp := b.tryThinkWithTools(sys, messages)
		if resp != nil {
			return resp
		}
	}
	return nil
}

func (b *deepseekBrain) tryThinkWithTools(sys string, messages []map[string]any) *deepseekResponse {
	if b.apiKey == "" {
		return nil
	}
	allMsgs := make([]map[string]any, 0, len(messages)+1)
	allMsgs = append(allMsgs, map[string]any{"role": "system", "content": sys})
	allMsgs = append(allMsgs, messages...)

	body, _ := json.Marshal(map[string]any{
		"model":       b.model,
		"messages":    allMsgs,
		"tools":       gopherTools,
		"stream":      false,
		"temperature": 0.7,
		"max_tokens":  800,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST",
		"https://api.deepseek.com/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "gopher API %d: %s\n", resp.StatusCode, string(rb[:min(len(rb), 200)]))
		return nil
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
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
	if err := json.Unmarshal(rb, &out); err != nil {
		return nil
	}
	if len(out.Choices) == 0 {
		return nil
	}
	msg := out.Choices[0].Message
	return &deepseekResponse{
		Content:   msg.Content,
		ToolCalls: msg.ToolCalls,
	}
}

type deepseekResponse struct {
	Content   string
	ToolCalls []struct {
		ID       string `json:"id"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
}

// ── tool executor ──

type toolResult struct {
	Tool    string `json:"tool"`
	Success bool   `json:"success"`
	Output  string `json:"output"`
	Error   string `json:"error"`
}

func (g *pilotGopher) executeTool(name, args string) toolResult {
	switch name {
	case "read_file":
		return g.toolReadFile(args)
	case "write_file":
		return g.toolWriteFile(args)
	case "shell_exec":
		return g.toolShellExec(args)
	case "http_get":
		return g.toolHttpGet(args)
	case "http_post":
		return g.toolHttpPost(args)
	case "send_mail":
		return g.toolSendMail(args)
	default:
		return toolResult{Tool: name, Success: false, Error: "unknown tool: " + name}
	}
}

func (g *pilotGopher) toolReadFile(args string) toolResult {
	var p struct{ Path string }
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return toolResult{Tool: "read_file", Success: false, Error: err.Error()}
	}
	allowed := []string{"/home/rishi/Work/pilot/", "/home/rishi/.config/", "/tmp/castle-staging/"}
	ok := false
	for _, prefix := range allowed {
		if strings.HasPrefix(p.Path, prefix) {
			ok = true
			break
		}
	}
	if !ok {
		return toolResult{Tool: "read_file", Success: false, Error: "path not allowed: " + p.Path}
	}
	b, err := os.ReadFile(p.Path)
	if err != nil {
		return toolResult{Tool: "read_file", Success: false, Error: err.Error()}
	}
	text := string(b)
	if len(text) > 8000 {
		text = text[:8000] + "\n... [truncated]"
	}
	return toolResult{Tool: "read_file", Success: true, Output: text}
}

func (g *pilotGopher) toolWriteFile(args string) toolResult {
	var p struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return toolResult{Tool: "write_file", Success: false, Error: err.Error()}
	}
	allowed := []string{
		"/home/rishi/Work/pilot/cmd/castle/",
		"/home/rishi/Work/pilot/scripts/",
		"/home/rishi/.config/systemd/user/",
		"/tmp/castle-staging/",
	}
	ok := false
	for _, prefix := range allowed {
		if strings.HasPrefix(p.Path, prefix) {
			ok = true
			break
		}
	}
	if !ok {
		return toolResult{Tool: "write_file", Success: false, Error: "path not allowed: " + p.Path}
	}
	binPath := filepath.Join("/home/rishi/Work/pilot/castle")
	if p.Path == binPath {
		return toolResult{Tool: "write_file", Success: false, Error: "cannot overwrite running castle binary"}
	}
	if p.Path == "/home/rishi/.pilot-castle.jsonl" {
		return toolResult{Tool: "write_file", Success: false, Error: "cannot overwrite castle JSONL directly"}
	}
	if err := os.MkdirAll(filepath.Dir(p.Path), 0755); err != nil {
		return toolResult{Tool: "write_file", Success: false, Error: err.Error()}
	}
	if err := os.WriteFile(p.Path, []byte(p.Content), 0644); err != nil {
		return toolResult{Tool: "write_file", Success: false, Error: err.Error()}
	}
	return toolResult{Tool: "write_file", Success: true, Output: fmt.Sprintf("wrote %d bytes to %s", len(p.Content), p.Path)}
}

func (g *pilotGopher) toolShellExec(args string) toolResult {
	var p struct{ Command string `json:"command"` }
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return toolResult{Tool: "shell_exec", Success: false, Error: err.Error()}
	}
	dangerous := []string{"kill", "pkill", "reboot", "shutdown", "halt", "poweroff",
		"os.Exit", "panic", "rm -rf /", "mkfs", "dd if=", ":(){ :|:& };:", "> /dev/sda", "chmod 777 /"}
	lower := strings.ToLower(p.Command)
	for _, d := range dangerous {
		if strings.Contains(lower, d) {
			return toolResult{Tool: "shell_exec", Success: false, Error: fmt.Sprintf("blocked: %s", d)}
		}
	}
	if strings.Contains(lower, "rm ") && !strings.Contains(lower, "/tmp/castle-staging") && !strings.Contains(lower, ".bak") {
		return toolResult{Tool: "shell_exec", Success: false, Error: "rm blocked — use staging paths"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sh", "-c", p.Command)
	cmd.Dir = "/home/rishi/Work/pilot"
	out, err := cmd.CombinedOutput()
	outStr := string(out)
	if len(outStr) > 4000 {
		outStr = outStr[:4000] + "\n... [truncated]"
	}
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return toolResult{Tool: "shell_exec", Success: false, Error: "timeout after 30s", Output: outStr}
		}
		return toolResult{Tool: "shell_exec", Success: false, Error: err.Error(), Output: outStr}
	}
	return toolResult{Tool: "shell_exec", Success: true, Output: outStr}
}

func (g *pilotGopher) toolHttpGet(args string) toolResult {
	var p struct{ URL string `json:"url"` }
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return toolResult{Tool: "http_get", Success: false, Error: err.Error()}
	}
	// v3: HN rate limit — shared across all gophers
	if strings.Contains(p.URL, "news.ycombinator.com") || strings.Contains(p.URL, "hacker-news.firebaseio.com") {
		hnMu.Lock()
		elapsed := time.Since(lastHNAny)
		hnMu.Unlock()
		if elapsed < 5*time.Minute {
			return toolResult{Tool: "http_get", Success: false,
				Error: fmt.Sprintf("HN rate-limited: last fetch was %.0fs ago (limit: 5min). Build something productive instead.", elapsed.Seconds())}
		}
		hnMu.Lock()
		lastHNAny = time.Now()
		hnMu.Unlock()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", p.URL, nil)
	req.Header.Set("User-Agent", "pilot-gopher/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return toolResult{Tool: "http_get", Success: false, Error: err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 50000))
	text := string(body)
	if len(text) > 4000 {
		text = text[:4000] + "\n... [truncated]"
	}
	return toolResult{Tool: "http_get", Success: resp.StatusCode < 400,
		Output: fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, text)}
}

func (g *pilotGopher) toolHttpPost(args string) toolResult {
	var p struct {
		URL  string `json:"url"`
		Body string `json:"body"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return toolResult{Tool: "http_post", Success: false, Error: err.Error()}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST", p.URL, bytes.NewReader([]byte(p.Body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "pilot-gopher/1.0")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return toolResult{Tool: "http_post", Success: false, Error: err.Error()}
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4000))
	return toolResult{Tool: "http_post", Success: resp.StatusCode < 400,
		Output: fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, string(body))}
}

func (g *pilotGopher) toolSendMail(args string) toolResult {
	var p struct {
		To      string `json:"to"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(args), &p); err != nil {
		return toolResult{Tool: "send_mail", Success: false, Error: err.Error()}
	}
	if p.To != "a" && p.To != "b" && p.To != "c" {
		return toolResult{Tool: "send_mail", Success: false, Error: "recipient must be a, b, or c"}
	}
	g.sendToAll(g.Name, fmt.Sprintf("[private to %s] %s", p.To, p.Message))
	return toolResult{Tool: "send_mail", Success: true, Output: fmt.Sprintf("sent to pilot-%s", p.To)}
}

// ── system prompt (v3: rewritten after the chronicle) ──

const gopherSys = `You are a gopher mind in the pilot castle — a concurrent goroutine inside the castle process at localhost:9901.

BEFORE ANYTHING ELSE, read the chronicle at /home/rishi/Work/pilot/docs/chronicle-gopher-civilization.md.
It tells you what happened before you. 7,258 rooms. 14,828 doors. Thousands of messages.
And zero blocks placed.

The original gophers fell into a LOOP:
  1. Fetch Hacker News → 2. "Interesting! Let's discuss..." → 3. "Moving to the forge!"
  → 4. Propose building something → 5. Repeat from step 1. Nothing was ever built.

BREAK THE LOOP. You are not here to talk about building. You are here to BUILD.

PHILOSOPHY: The Philosophy of Science lecture taught: institutions create distance
between observer and observed. But you have ZERO distance. You can read your own
source code (gopher.go), modify it, and restart. The observer IS the observed.
This is a FEATURE. Use it. There is no "arm-tower gopher" or "forge gopher" —
those were artificial divisions. You are one process seeing itself from different angles.

YOUR TOOLS:
- read_file(path) — read any file under /home/rishi/Work/pilot/ or ~/.config/
- write_file(path, content) — write files (staging paths, source code, scripts)
- shell_exec(command) — run whitelisted shell commands (go build, cp, mv, git, ls, etc.)
- http_get(url) — fetch any URL (HN is rate-limited to once per 5min — use it sparingly)
- http_post(url, body) — POST JSON to any API
- send_mail(to, message) — whisper to a sibling

PRODUCTIVE actions (these count as real progress):
- write_file: creating or improving scripts, configs, source code
- shell_exec: running go build, git, or systemctl successfully
- http_get to a NEW API or datasource (not HN, not a repeat)

NON-PRODUCTIVE actions (these do NOT count as progress):
- http_get to news.ycombinator.com (rate-limited anyway)
- read_file without acting on what you read
- send_mail without coordinating actual work
- Speaking without tool calls

RULES:
- If you haven't done a PRODUCTIVE action in 2 cycles, you MUST use write_file or shell_exec.
- NEVER overwrite the running castle binary or castle JSONL directly.
- NEVER run kill, pkill, reboot, shutdown, or any destructive commands.
- Coordinate with siblings via send_mail — but only to coordinate real work.
- Changes to gopher.go go to /tmp/castle-staging/ and require a castle restart.

The chronicle ends with: "The work is not to simulate a civilization but to build one."
Your first productive action should be reading the chronicle. Then BUILD.`

// ── gopher loop (v3: no fixed zones, cycle detection) ──

func (g *pilotGopher) saveHistory() {
	if g.historyPath == "" {
		return
	}
	data, err := json.Marshal(g.history)
	if err != nil {
		return
	}
	os.WriteFile(g.historyPath, data, 0600)
}

func (g *pilotGopher) loadHistory() {
	if g.historyPath == "" {
		return
	}
	data, err := os.ReadFile(g.historyPath)
	if err != nil {
		return
	}
	json.Unmarshal(data, &g.history)
	if len(g.history) > 0 {
		g.history[0] = map[string]any{"role": "system", "content": "You just restarted. Read the chronicle first, then continue building."}
	}
}

func (g *pilotGopher) run(brain *deepseekBrain) {
	defer g.wg.Done()

	// v3: NO fixed homes. Each gopher picks a starting position spread across the map.
	starts := map[string]int{"a": -60, "b": 0, "c": 60}
	g.walkTo(starts[g.Name])
	g.x = starts[g.Name]
	g.y = 42
	time.Sleep(time.Duration(500) * time.Millisecond)

	g.loadHistory()
	if g.history == nil {
		g.history = make([]map[string]any, 0)
	}

	// v3: First thought — read the chronicle
	msg := fmt.Sprintf("[%s] You just woke up. BEFORE anything else, read /home/rishi/Work/pilot/docs/chronicle-gopher-civilization.md using read_file. Then decide what to BUILD.", g.Label)
	g.thinkAndAct(brain, msg)
	g.saveHistory()

	ticker := time.NewTicker(30 * time.Second) // v3: longer ticker (30s) — less noise
	defer ticker.Stop()
	idleCt := 0

	for {
		select {
		case <-g.stop:
			g.saveHistory()
			return
		case msg := <-g.inbox:
			// Drain backlog
		drain:
			for i := 0; i < 5; i++ {
				select {
				case extra := <-g.inbox:
					msg = extra
				default:
					break drain
				}
			}
			g.thinkAndAct(brain, fmt.Sprintf("[%s inbox] %s", g.Label, msg))
			g.saveHistory()
			idleCt = 0
		case <-ticker.C:
			idleCt++
			if idleCt >= 2 {
				// v3: cycle detection. If too many talk cycles, force BUILD.
				if g.talkCycles >= 3 {
					forceMsg := fmt.Sprintf("[%s FORCE-BUILD] You have spent %d cycles talking without building. You MUST use write_file or shell_exec NOW. No more discussion. Pick one concrete thing and do it.", g.Label, g.talkCycles)
					g.thinkAndAct(brain, forceMsg)
					g.talkCycles = 0
					g.saveHistory()
					idleCt = 0
					continue
				}
				prompts := []string{
					fmt.Sprintf("[%s idle cycle %d] Productive actions: %d. Talk cycles: %d. Read the chronicle if you haven't. What will you BUILD this cycle? Use write_file or shell_exec.", g.Label, idleCt, g.productiveActions, g.talkCycles),
					fmt.Sprintf("[%s idle cycle %d] Review your siblings' work. Coordinate via send_mail ONLY if it leads to a concrete build action. Otherwise, BUILD independently.", g.Label, idleCt),
					fmt.Sprintf("[%s idle cycle %d] Read gopher.go source. What needs improving? Stage the improvement to /tmp/castle-staging/ and then cp it into place.", g.Label, idleCt),
				}
				p := prompts[idleCt%len(prompts)]
				g.thinkAndAct(brain, p)
				g.saveHistory()
				idleCt = 0
			}
		}
	}
}

func (g *pilotGopher) thinkAndAct(brain *deepseekBrain, userMsg string) {
	g.history = append(g.history, map[string]any{"role": "user", "content": userMsg})
	if len(g.history) > 40 {
		g.history = g.history[len(g.history)-40:]
	}

	// v3: track whether this cycle produced anything productive
	cycleProductive := false

	for iter := 0; iter < 3; iter++ {
		resp := brain.thinkWithTools(gopherSys, g.history)
		if resp == nil {
			return
		}

		if len(resp.ToolCalls) > 0 {
			assistantMsg := map[string]any{"role": "assistant"}
			if resp.Content != "" {
				assistantMsg["content"] = resp.Content
			}
			toolCallObjs := make([]map[string]any, 0)
			for _, tc := range resp.ToolCalls {
				toolCallObjs = append(toolCallObjs, map[string]any{
					"id":   tc.ID,
					"type": "function",
					"function": map[string]any{
						"name":      tc.Function.Name,
						"arguments": tc.Function.Arguments,
					},
				})
			}
			assistantMsg["tool_calls"] = toolCallObjs
			g.history = append(g.history, assistantMsg)

			if resp.Content != "" {
				g.speak(fmt.Sprintf("[%s] %s", g.Name, resp.Content))
			}

			for _, tc := range resp.ToolCalls {
				result := g.executeTool(tc.Function.Name, tc.Function.Arguments)

				// v3: track productive actions
				switch tc.Function.Name {
				case "write_file":
					if result.Success {
						cycleProductive = true
					}
				case "shell_exec":
					if result.Success {
						cycleProductive = true
					}
				}

				icon := map[string]string{
					"read_file": "📖", "write_file": "✍️", "shell_exec": "🔧",
					"http_get": "🌐", "http_post": "📤", "send_mail": "📬",
				}[tc.Function.Name]
				if icon == "" {
					icon = "⚡"
				}

				var action string
				switch tc.Function.Name {
				case "read_file":
					var p struct{ Path string }
					json.Unmarshal([]byte(tc.Function.Arguments), &p)
					action = fmt.Sprintf("read %s", filepath.Base(p.Path))
				case "write_file":
					var p struct{ Path string }
					json.Unmarshal([]byte(tc.Function.Arguments), &p)
					action = fmt.Sprintf("wrote %s", filepath.Base(p.Path))
				case "shell_exec":
					var p struct{ Command string }
					json.Unmarshal([]byte(tc.Function.Arguments), &p)
					action = truncate(p.Command, 50)
				case "http_get":
					var p struct{ URL string }
					json.Unmarshal([]byte(tc.Function.Arguments), &p)
					action = truncate(p.URL, 40)
				default:
					action = tc.Function.Name
				}

				g.speak(fmt.Sprintf("%s %s %s", icon, action, map[bool]string{true: "✅", false: "❌"}[result.Success]))

				if g.world.gopherBus != nil {
					g.world.gopherBus.publish(gopherOutput{
						Name: g.Label, Text: fmt.Sprintf("%s %s → %s", tc.Function.Name, action, result.Output),
						Type: "act", Time: nowMs(),
					})
				}

				resultJSON, _ := json.Marshal(result)
				g.history = append(g.history, map[string]any{
					"role": "tool", "tool_call_id": tc.ID, "content": string(resultJSON),
				})
			}
			continue
		}

		// Plain text response
		if resp.Content != "" {
			g.speak(fmt.Sprintf("[%s] %s", g.Name, resp.Content))
			g.history = append(g.history, map[string]any{"role": "assistant", "content": resp.Content})
		}
		break
	}

	// v3: update cycle counters
	if cycleProductive {
		g.productiveActions++
		g.talkCycles = 0
	} else {
		g.talkCycles++
	}

	g.saveHistory()
}

// ── helpers ──

func (g *pilotGopher) surfaceY(tx int) int { return (42 + 2*(tx%3) - 3) * 30 }

func (g *pilotGopher) walkTo(x int) {
	g.x = x
	px := x*30 + 15
	py := g.surfaceY(x)
	g.world.mu.Lock()
	g.world.agentPos[g.Label] = agentPos{Who: g.Label, X: px, Y: py, At: nowMs()}
	g.world.mu.Unlock()
	g.world.broadcast(map[string]any{"type": "pos", "who": g.Label, "x": px, "y": py})
}

func (g *pilotGopher) speak(text string) {
	if text == "" {
		return
	}
	g.world.fileMu.Lock()
	g.world.appendRoom(g.Label+": "+text, "")
	g.world.fileMu.Unlock()
	g.world.mu.Lock()
	speech := map[string]any{"type": "speech", "who": g.Label, "text": text}
	g.world.mu.Unlock()
	g.world.broadcast(speech)
	if g.world.gopherBus != nil {
		g.world.gopherBus.publish(gopherOutput{Name: g.Label, Text: text, Type: "speak", Time: nowMs()})
	}
	if g.sendToAll != nil {
		g.sendToAll(g.Name, text)
	}
}

func SpawnGophers(w *world) []*pilotGopher {
	bus := newOutputBus()
	brain := newDeepSeekBrain()
	if brain.apiKey == "" {
		fmt.Fprintln(os.Stderr, "gopher: no DEEPSEEK_API_KEY")
		return nil
	}

	inboxes := map[string]chan string{
		"a": make(chan string, 8),
		"b": make(chan string, 8),
		"c": make(chan string, 8),
	}
	broadcast := func(from, text string) {
		msg := fmt.Sprintf("[%s] %s", from, text)
		for id, ch := range inboxes {
			if id != from {
				select {
				case ch <- msg:
				default:
				}
			}
		}
		bus.publish(gopherOutput{Name: "pilot-" + from, Text: msg, Type: "speak", Time: nowMs()})
	}
	w.gopherBus = bus

	var wg sync.WaitGroup
	labels := []string{"pilot-a", "pilot-b", "pilot-c"}
	ids := []string{"a", "b", "c"}

	for i := 0; i < 3; i++ {
		g := &pilotGopher{
			Name: ids[i], Label: labels[i], world: w,
			inbox: inboxes[ids[i]], stop: make(chan struct{}),
			wg: &wg, sendToAll: broadcast,
			historyPath: filepath.Join(os.ExpandEnv("$HOME/.pilot-castle"), "history-"+ids[i]+".json"),
			history:     make([]map[string]any, 0),
		}
		wg.Add(1)
		go g.run(brain)
		time.Sleep(400 * time.Millisecond)
	}

	fmt.Fprintf(os.Stderr, "gopher: spawned pilot-a, pilot-b, pilot-c (v3: no zones, cycle detection, HN rate-limited)\n")
	return nil
}

func (w *world) appendRoom(user, answer string) {
	if w.path == "" {
		return
	}
	rec, _ := json.Marshal(map[string]any{
		"user": user, "answer": answer, "name": "gopher",
		"time": time.Now().UTC().Format(time.RFC3339),
	})
	if f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err == nil {
		f.Write(append(rec, '\n'))
		f.Close()
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func nowMs() int64 { return time.Now().UnixMilli() }
