// pilot — a local model with hands.
//
// Talk to it like you'd talk to anyone. There is no objective up front; you can
// open with "hi". The work grows out of the dialogue. When what you ask needs a
// real call against the world, the model reaches for a tool and pilot makes the
// call for real — then the conversation continues with the actual response in
// hand.
//
// The brain is a local Ollama model. The hands are MCP tool-servers (today:
// http-mcp, whose one tool is an HTTP request — which is also how you drive any
// WebDriver/Appium hub). pilot is the host between them: it carries the
// conversation, hands the model its tools, executes the calls, and loops.
//
// stdlib only. os/exec to run the tool-server, net/http to reach Ollama.
//
//	pilot                 # start talking
//	echo "hi" | pilot     # one line over a pipe, then EOF
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func main() {
	model := flag.String("model", defaultModel, "Ollama model tag")
	ollama := flag.String("ollama", "http://localhost:11434", "Ollama base URL")
	server := flag.String("server", "", "path to the http-mcp tool-server binary (auto-detected if empty)")
	maxSteps := flag.Int("max-steps", 12, "max tool-call rounds within a single turn")
	flag.Parse()

	bin, err := resolveServer(*server)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pilot: %v\n", err)
		os.Exit(1)
	}
	mcp, err := startServer(bin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "pilot: start tool-server: %v\n", err)
		os.Exit(1)
	}
	defer mcp.close()

	tools, err := mcp.handshake()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pilot: handshake: %v\n", err)
		os.Exit(1)
	}

	repl(&session{
		base: *ollama, model: *model, mcp: mcp, tools: tools, maxSteps: *maxSteps,
		msgs: []message{{Role: "system", Content: system}},
	}, toolNames(tools), bin)
}

// ---- conversation ----

type session struct {
	base, model string
	mcp         *mcpServer
	tools       []map[string]any
	maxSteps    int
	msgs        []message // grows across turns; the dialogue is the state
}

func repl(s *session, names []string, bin string) {
	interactive := isTTY(os.Stdin)
	if interactive {
		fmt.Fprintf(os.Stderr, "\033[2mpilot · model %s · tools %v · %s\033[0m\n", s.model, names, filepath.Base(bin))
		fmt.Fprintf(os.Stderr, "\033[2mjust talk. /exit to leave.\033[0m\n")
	}
	in := bufio.NewReader(os.Stdin)
	for {
		if interactive {
			fmt.Print("\033[1myou ❯\033[0m ")
		}
		line, err := in.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			switch line {
			case "/exit", "/quit", "/bye":
				return
			}
			s.turn(line)
		}
		if err != nil { // EOF (Ctrl-D or end of pipe)
			return
		}
	}
}

// turn appends the user's line and lets the model run until it answers without
// reaching for a tool. Tool activity is shown dimmed so the dialogue stays front
// and center.
func (s *session) turn(userLine string) {
	s.msgs = append(s.msgs, message{Role: "user", Content: userLine})

	seen := map[string]bool{} // (name+args) already run this turn — a weak model loops; we don't let it
	nudgedEmpty := false

	for range s.maxSteps {
		reply, err := s.chat()
		if err != nil {
			fmt.Fprintf(os.Stderr, "\033[31mpilot: %v\033[0m\n", err)
			return
		}
		s.msgs = append(s.msgs, reply)

		if reply.Thinking != "" {
			fmt.Fprintf(os.Stderr, "\033[2m· %s\033[0m\n", oneLine(reply.Thinking))
		}

		if len(reply.ToolCalls) == 0 {
			answer := strings.TrimSpace(reply.Content)
			if answer == "" && !nudgedEmpty { // model fell silent — ask once for the answer it already has
				nudgedEmpty = true
				s.msgs = append(s.msgs, message{Role: "user",
					Content: "Answer my last message now, in plain words, using what you already have. Do not call any tool."})
				continue
			}
			fmt.Printf("\033[1mpilot ❯\033[0m %s\n", answer)
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
			} else if out, err = s.mcp.callTool(tc.Function.Name, tc.Function.Arguments); err != nil {
				out = "tool error: " + err.Error()
			}
			seen[sig] = true
			fmt.Fprintf(os.Stderr, "\033[32m  ← %s\033[0m\n", oneLine(out))
			s.msgs = append(s.msgs, message{Role: "tool", ToolName: tc.Function.Name, Content: out})
		}
	}
	fmt.Fprintf(os.Stderr, "\033[33mpilot: gave up after %d tool rounds this turn\033[0m\n", s.maxSteps)
}

type message struct {
	Role      string     `json:"role"`
	Content   string     `json:"content"`
	Thinking  string     `json:"thinking,omitempty"`
	ToolCalls []toolCall `json:"tool_calls,omitempty"`
	ToolName  string     `json:"tool_name,omitempty"`
}

type toolCall struct {
	Function struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"function"`
}

func (s *session) chat() (message, error) {
	body, _ := json.Marshal(map[string]any{
		"model": s.model, "messages": s.msgs, "tools": s.tools, "stream": false,
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

// ---- minimal MCP client over the tool-server's stdio (JSON-RPC 2.0, NDJSON) ----

type mcpServer struct {
	cmd *exec.Cmd
	in  io.WriteCloser
	out *bufio.Reader
	id  int
}

func startServer(path string) (*mcpServer, error) {
	cmd := exec.Command(path)
	cmd.Stderr = os.Stderr
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
