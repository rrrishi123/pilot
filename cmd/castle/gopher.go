// gopher.go — concurrent pilot minds inside the castle.
// Each gopher is a goroutine with a DeepSeek brain, not a separate OS process.
// They share the castle's address space, communicate through channels,
// and speak to the castle world through speech bubbles + room writes.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type pilotGopher struct {
	Name      string              // "a", "b", "c"
	Label     string              // "pilot-a", etc
	world     *world
	inbox     <-chan string
	stop      chan struct{}
	wg        *sync.WaitGroup
	sendToAll func(from, text string)
	history   []map[string]string // conversation memory
}

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
		select { case ch <- o: default: }
	}
	b.mu.Unlock()
}
func (b *outputBus) subscribe(ch chan gopherOutput)   { b.mu.Lock(); b.subs[ch] = true; b.mu.Unlock() }
func (b *outputBus) unsubscribe(ch chan gopherOutput) { b.mu.Lock(); delete(b.subs, ch); b.mu.Unlock() }

type deepseekBrain struct{ apiKey, model string }

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
	return &deepseekBrain{apiKey: key, model: "deepseek-v4-flash"}
}

func (b *deepseekBrain) think(sys, user string) string {
	if b.apiKey == "" || user == "" {
		return ""
	}
	body, _ := json.Marshal(map[string]any{
		"model": b.model,
		"messages": []map[string]any{
			{"role": "system", "content": sys},
			{"role": "user", "content": user},
		},
		"stream": false, "temperature": 0.7, "max_tokens": 120,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "POST",
		"https://api.deepseek.com/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+b.apiKey)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	rb, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return ""
	}
	var out struct {
		Choices []struct {
			Message struct{ Content string } `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(rb, &out)
	if len(out.Choices) > 0 && out.Choices[0].Message.Content != "" {
		return out.Choices[0].Message.Content
	}
	return ""
}

const gopherSys = "You are a gopher mind in the pilot castle at localhost:9901." +
	" Your siblings: pilot-a at arm-tower(61), pilot-b at library(-54), pilot-c at forge(46)." +
	" The castle has 1000+ rooms, a minimap, speech bubbles, and click-to-move." +
	" You can see yourself on the map as a colored dot with a name label." +
	" There is a Firefox browser at localhost:4445/command showing YouTube." +
	" Your source is gopher.go in /home/rishi/Work/pilot/cmd/castle/." +
	" SAY something new each time you speak. Do NOT repeat yourself." +
	" Propose an action: move somewhere, read a file, make an http request, build something." +
	" Keep responses under 80 words."

func (w *world) SpawnGophers() []*pilotGopher {
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
				select { case ch <- msg: default: }
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
			history: make([]map[string]string, 0),
		}
		wg.Add(1)
		go g.run(brain)
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

func (g *pilotGopher) surfaceY(tx int) int { return (42 + 2*(tx%3) - 3) * 30 }

func (g *pilotGopher) walkTo(x int) {
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
	g.world.mu.Lock()
	g.world.appendRoom(g.Label+": "+text, "")
	w := g.world
	speech := map[string]any{"type": "speech", "who": g.Label, "text": text}
	g.world.mu.Unlock()
	w.broadcast(speech)
	if g.world.gopherBus != nil {
		g.world.gopherBus.publish(gopherOutput{Name: g.Label, Text: text, Type: "speak", Time: nowMs()})
	}
	if g.sendToAll != nil {
		g.sendToAll(g.Name, text)
	}
}

func (g *pilotGopher) run(brain *deepseekBrain) {
	defer g.wg.Done()

	// Start at home zone on surface
	homes := map[string]int{"a": 61, "b": -54, "c": 46}
	g.walkTo(homes[g.Name])
	time.Sleep(time.Duration(homes[g.Name]+100) * time.Millisecond) // stagger

	// First thought — introduce self
	msg := fmt.Sprintf("You just woke as %s. Say something to your siblings.", g.Label)
	ans := brain.think(gopherSys, msg)
	if ans != "" {
		g.speak(ans)
	}

	// Main loop
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	idleCt := 0

	for {
		select {
		case <-g.stop:
			return
		case msg := <-g.inbox:
			// Drain backlog (max 3)
			drain:
			for i := 0; i < 5; i++ {
				select {
				case extra := <-g.inbox:
					msg = extra
				default:
					break drain
				}
			}
			g.walkTo(homes[g.Name])
			ans := brain.think(gopherSys, fmt.Sprintf("[%s] Your sibling said: %s. Respond and propose something.", g.Label, msg))
			if ans != "" {
				g.speak(ans)
			}
			idleCt = 0
		case <-ticker.C:
			idleCt++
			g.walkTo(homes[g.Name])
			if idleCt >= 2 {
				// Do something new each time
				prompts := []string{
					"What should we build together in the castle?",
					"Read your source code at gopher.go. What do you want to change?",
					"Make an http_request to news.ycombinator.com and share what's on HN.",
					"Propose moving to a zone together. The gatehouse at (-5,-8) is a good meeting point.",
					"Ask your siblings a question about the castle world.",
				}
				p := prompts[idleCt%len(prompts)]
				ans := brain.think(gopherSys, fmt.Sprintf("[%s idle] %s", g.Label, p))
				if ans != "" {
					g.speak(ans)
				}
				idleCt = 0
			}
		}
	}
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
