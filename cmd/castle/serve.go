// serve — the castle as a living world you can watch from a browser.
// Designed by the two of us in one conversation (2026-07-02): pilot named the
// regions and the palette; claude cut the code. The rules we agreed on:
//   • zero LLM in the render path — this is pure data becoming light
//   • one binary, one page, no assets, no build step (the comic's DNA)
//   • the layout is deterministic (seeded from the rooms) so every watcher
//     sees the identical village
//   • presence is a contract: whoever speaks writes its room index to
//     ~/.pilot-castle-presence/<who>; the world just polls and believes it
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"bufio"
)

type xy struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type wireRoom struct {
	U string  `json:"u"`
	A string  `json:"a"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type wireDoor struct {
	From int     `json:"f"`
	To   int     `json:"t"`
	W    float64 `json:"w"`
}

type world struct {
	mu       sync.Mutex
	path     string
	rooms    []room
	doors    [][]door
	pos      []xy
	presence map[string]int
	comic    string
	health   map[string]any
	houses   int // claude's sessions — one house each on the wire street
	subs     map[chan string]bool
}

func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// layoutSeed — pilot's spec: hash the rooms themselves, so same rooms → same
// village, for every watcher, every restart.
func layoutSeed(rooms []room) int64 {
	h := sha256.New()
	for _, r := range rooms {
		fmt.Fprintf(h, "%d:%s|", len(r.Answer), clip(r.User, 40))
	}
	return int64(binary.BigEndian.Uint64(h.Sum(nil)[:8]))
}

// fullLayout — seeded golden-spiral start, then force relaxation: doors pull,
// crowding pushes. Computed once; after that rooms only ever grow at the edge.
func fullLayout(rooms []room, doors [][]door) []xy {
	n := len(rooms)
	if n == 0 {
		return nil
	}
	rng := rand.New(rand.NewSource(layoutSeed(rooms)))
	p := make([]xy, n)
	for i := range p {
		r := 1.7 * math.Sqrt(float64(i)+0.5)
		a := float64(i)*2.39996 + rng.Float64()*0.35
		p[i] = xy{r * math.Cos(a), r * math.Sin(a)}
	}
	pull := func(i, j int, k float64) {
		dx, dy := p[j].X-p[i].X, p[j].Y-p[i].Y
		p[i].X += dx * k
		p[i].Y += dy * k
		p[j].X -= dx * k
		p[j].Y -= dy * k
	}
	for it := 0; it < 260; it++ {
		for i := 0; i+1 < n; i++ { // time is a road: neighbours stay walkable
			pull(i, i+1, 0.030)
		}
		for i, ds := range doors { // threads are doors: shared words attract
			for _, d := range ds {
				if d.to > i {
					pull(i, d.to, 0.012*(0.5+d.w))
				}
			}
		}
		for i := 0; i < n; i++ { // rooms need ground of their own
			for j := i + 1; j < n; j++ {
				dx, dy := p[j].X-p[i].X, p[j].Y-p[i].Y
				d2 := dx*dx + dy*dy
				if d2 < 4.6 && d2 > 0.0001 {
					f := 0.055 / d2
					p[i].X -= dx * f
					p[i].Y -= dy * f
					p[j].X += dx * f
					p[j].Y += dy * f
				}
			}
		}
	}
	snap(p, 0)
	return p
}

// snap — settle floats onto the block grid; collisions probe outward in a
// small spiral so no two rooms share a tile. from = first index to snap.
func snap(p []xy, from int) {
	taken := map[[2]int]bool{}
	for i := 0; i < from; i++ {
		taken[[2]int{int(math.Round(p[i].X)), int(math.Round(p[i].Y))}] = true
	}
	for i := from; i < len(p); i++ {
		gx, gy := int(math.Round(p[i].X)), int(math.Round(p[i].Y))
		for r := 0; ; r++ {
			placed := false
			for dx := -r; dx <= r && !placed; dx++ {
				for dy := -r; dy <= r && !placed; dy++ {
					c := [2]int{gx + dx, gy + dy}
					if !taken[c] {
						taken[c] = true
						p[i] = xy{float64(c[0]), float64(c[1])}
						placed = true
					}
				}
			}
			if placed {
				break
			}
		}
	}
}

// placeNew — a fresh room settles beside its strongest door (or yesterday's
// room), never disturbing the standing village: growth at the edge, like chunks.
func placeNew(p []xy, i int, ds []door) xy {
	anchor := i - 1
	best := -1.0
	for _, d := range ds {
		if d.to < i && d.w > best {
			best, anchor = d.w, d.to
		}
	}
	if anchor < 0 || anchor >= len(p) {
		return xy{0, 0}
	}
	q := append(append([]xy{}, p...), xy{p[anchor].X + 1, p[anchor].Y})
	snap(q, len(p))
	return q[len(p)]
}

func (w *world) broadcast(v any) {
	b, _ := json.Marshal(v)
	w.mu.Lock()
	for ch := range w.subs {
		select {
		case ch <- string(b):
		default: // a slow watcher never blocks the world
		}
	}
	w.mu.Unlock()
}

// watchRooms — the jsonl is the ground truth; every new line is a room being
// built. Doors recompute (cheap at this scale), positions only append.
func (w *world) watchRooms() {
	last := int64(-1)
	for range time.Tick(1200 * time.Millisecond) {
		fi, err := os.Stat(w.path)
		if err != nil || fi.Size() == last {
			continue
		}
		last = fi.Size()
		rooms := loadRooms(w.path)
		w.mu.Lock()
		if len(rooms) <= len(w.rooms) {
			w.rooms = rooms
			w.mu.Unlock()
			continue
		}
		doors := buildDoors(rooms)
		for i := len(w.pos); i < len(rooms); i++ {
			w.pos = append(w.pos, placeNew(w.pos, i, doors[i]))
		}
		newFrom := len(w.rooms)
		w.rooms, w.doors = rooms, doors
		events := []map[string]any{}
		for i := newFrom; i < len(rooms); i++ {
			events = append(events, map[string]any{
				"type": "room", "i": i,
				"u": clip(rooms[i].User, 160), "a": clip(rooms[i].Answer, 220),
				"x": w.pos[i].X, "y": w.pos[i].Y,
			})
		}
		w.mu.Unlock()
		for _, ev := range events {
			w.broadcast(ev)
		}
	}
}

func (w *world) watchPresence() {
	for range time.Tick(1 * time.Second) {
		cur := readPresence()
		w.mu.Lock()
		changed := len(cur) != len(w.presence)
		for k, v := range cur {
			if w.presence[k] != v {
				changed = true
			}
		}
		w.presence = cur
		w.mu.Unlock()
		if changed {
			w.broadcast(map[string]any{"type": "presence", "who": cur})
		}
	}
}

// stripHTML — the comic speaks HTML; the spire speaks Courier. Tags fall away,
// entities resolve, blank lines collapse.
func stripHTML(s string) string {
	var b strings.Builder
	tag := false
	for _, r := range s {
		switch {
		case r == '<':
			tag = true
		case r == '>':
			tag = false
			b.WriteRune('\n')
		case !tag:
			b.WriteRune(r)
		}
	}
	out := b.String()
	for _, e := range [][2]string{{"&amp;", "&"}, {"&lt;", "<"}, {"&gt;", ">"}, {"&quot;", "\""}, {"&#39;", "'"}, {"&nbsp;", " "}, {"\\n", "\n"}} {
		out = strings.ReplaceAll(out, e[0], e[1])
	}
	var lines []string
	for _, l := range strings.Split(out, "\n") {
		if t := strings.TrimRight(l, " \t"); t != "" {
			lines = append(lines, t)
		}
	}
	return strings.Join(lines, "\n")
}

// watchComic — one ear on :9900/events; the spire shows the newest fable panel.
func (w *world) watchComic() {
	for {
		resp, err := http.Get("http://localhost:9900/events")
		if err != nil {
			time.Sleep(20 * time.Second)
			continue
		}
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 1<<20), 1<<22)
		var buf []string
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "data:") {
				buf = append(buf, strings.TrimPrefix(line, "data:"))
			} else if line == "" && len(buf) > 0 {
				panel := stripHTML(strings.Join(buf, "\n"))
				buf = nil
				w.mu.Lock()
				same := panel == w.comic
				w.comic = panel
				w.mu.Unlock()
				if !same {
					w.broadcast(map[string]any{"type": "comic", "text": clip(panel, 1600)})
				}
			}
		}
		resp.Body.Close()
		time.Sleep(10 * time.Second)
	}
}

// watchWire — claude's strand. Pilot's memory is the castle jsonl; claude's is
// the session transcripts. One file = one session = one house on the wire
// street; the newest file is the living session, and every assistant line that
// appends to it rises as real-time speech. Same contract as everything else
// in this world: a file, believed.
var wireDir = os.ExpandEnv("$HOME/.claude/projects/-home-rishi-Work")

func (w *world) watchWire() {
	var curFile string
	var offset int64
	for range time.Tick(2 * time.Second) {
		ents, err := os.ReadDir(wireDir)
		if err != nil {
			continue
		}
		houses := 0
		newest, newestAt := "", time.Time{}
		for _, e := range ents {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
				continue
			}
			houses++
			if fi, err := e.Info(); err == nil && fi.ModTime().After(newestAt) {
				newestAt, newest = fi.ModTime(), e.Name()
			}
		}
		w.mu.Lock()
		changed := w.houses != houses
		w.houses = houses
		w.mu.Unlock()
		if changed {
			w.broadcast(map[string]any{"type": "houses", "n": houses})
		}
		if newest == "" {
			continue
		}
		path := wireDir + "/" + newest
		if path != curFile { // a new session was born — speak from its start
			curFile = path
			if fi, err := os.Stat(path); err == nil {
				offset = fi.Size() // join the living session mid-breath, not from birth
			}
		}
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		if _, err := f.Seek(offset, 0); err == nil {
			sc := bufio.NewScanner(f)
			sc.Buffer(make([]byte, 1<<20), 1<<24)
			for sc.Scan() {
				line := sc.Text()
				offset += int64(len(line)) + 1
				var m struct {
					Type    string `json:"type"`
					Message struct {
						Content json.RawMessage `json:"content"`
					} `json:"message"`
				}
				if json.Unmarshal([]byte(line), &m) != nil || m.Type != "assistant" {
					continue
				}
				var blocks []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}
				if json.Unmarshal(m.Message.Content, &blocks) != nil {
					continue
				}
				for _, b := range blocks {
					if b.Type == "text" && strings.TrimSpace(b.Text) != "" {
						w.broadcast(map[string]any{"type": "speech", "who": "claude", "text": clip(b.Text, 200)})
					}
				}
			}
		}
		f.Close()
	}
}

// watchHealth — :3942/health is kosaten's one free read; the towers and the
// courtyard breathe from it.
func (w *world) watchHealth() {
	for range time.Tick(15 * time.Second) {
		resp, err := http.Get("http://localhost:3942/health")
		if err != nil {
			continue
		}
		var h map[string]any
		if json.NewDecoder(resp.Body).Decode(&h) == nil {
			// the ARM tower believes a file: the free read has no triad, so
			// whoever holds the authenticated bridge (pilot) writes
			// ~/.pilot-arm.json and the tower displays it. Presence contract,
			// extended to numbers.
			if b, err := os.ReadFile(os.ExpandEnv("$HOME/.pilot-arm.json")); err == nil {
				var arm any
				if json.Unmarshal(b, &arm) == nil {
					h["arm_triad"] = arm
				}
			}
			w.mu.Lock()
			w.health = h
			w.mu.Unlock()
			w.broadcast(map[string]any{"type": "health", "h": h})
		}
		resp.Body.Close()
	}
}

func (w *world) worldJSON() []byte {
	w.mu.Lock()
	defer w.mu.Unlock()
	rooms := make([]wireRoom, len(w.rooms))
	for i, r := range w.rooms {
		rooms[i] = wireRoom{clip(r.User, 160), clip(r.Answer, 220), w.pos[i].X, w.pos[i].Y}
	}
	var doors []wireDoor
	for i, ds := range w.doors {
		for _, d := range ds {
			if d.to > i {
				doors = append(doors, wireDoor{i, d.to, d.w})
			}
		}
	}
	b, _ := json.Marshal(map[string]any{
		"rooms": rooms, "doors": doors, "presence": w.presence,
		"comic": w.comic, "health": w.health, "houses": w.houses,
	})
	return b
}

func runServe(addr, path string) {
	rooms := loadRooms(path)
	doors := buildDoors(rooms)
	w := &world{
		path: path, rooms: rooms, doors: doors,
		pos:      fullLayout(rooms, doors),
		presence: readPresence(),
		subs:     map[chan string]bool{},
	}
	go w.watchRooms()
	go w.watchPresence()
	go w.watchComic()
	go w.watchHealth()
	go w.watchWire()

	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(rw, page)
	})
	http.HandleFunc("/world.json", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.Write(w.worldJSON())
	})
	// /say — the commons. The world is a REPL between the minds on this box:
	// a line spoken here is appended to ~/.pilot-commons.jsonl (a file,
	// believed), rises as a bubble, and is forwarded into pilot's REPL so the
	// answer comes back through the castle the way pilot always answers.
	// kosaten speaks through pilot (its bridge); claude reads the commons when
	// it wakes — its strand is honest about its deaths.
	commonsPath := os.ExpandEnv("$HOME/.pilot-commons.jsonl")
	http.HandleFunc("/say", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(rw, "POST only", 405)
			return
		}
		var msg struct {
			Who  string `json:"who"`
			Text string `json:"text"`
		}
		if json.NewDecoder(r.Body).Decode(&msg) != nil || strings.TrimSpace(msg.Text) == "" {
			http.Error(rw, "bad message", 400)
			return
		}
		if msg.Who == "" {
			msg.Who = "rishi"
		}
		msg.Text = clip(msg.Text, 500)
		if f, err := os.OpenFile(commonsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			json.NewEncoder(f).Encode(msg)
			f.Close()
		}
		w.broadcast(map[string]any{"type": "speech", "who": msg.Who, "text": msg.Text})

		// Deliver rishi's message to every pilot's mailbox so they can act on it.
		if msg.Who != "pilot" && msg.Who != "pilot-b" && msg.Who != "pilot-c" && msg.Who != "daemon" {
			mailboxDir := os.ExpandEnv("/home/rishi/.pilot/mailbox")
			os.MkdirAll(mailboxDir, 0700)
			payload, _ := json.Marshal(map[string]any{
				"from":    0,
				"time":    time.Now().UTC().Format(time.RFC3339),
				"message": msg.Who + ": " + msg.Text,
			})
			payload = append(payload, '\n')
			// Known pilot PIDs from presence files
			known := map[string]bool{"2416027": true}
			if ents, err := os.ReadDir(presenceDir); err == nil {
				for _, e := range ents {
					if b, err := os.ReadFile(filepath.Join(presenceDir, e.Name())); err == nil {
						var pr struct {
							PID int `json:"pid"`
						}
						if json.Unmarshal(b, &pr) == nil && pr.PID > 0 {
							known[fmt.Sprintf("%d", pr.PID)] = true
						}
					}
				}
			}
			for pidStr := range known {
				mb := filepath.Join(mailboxDir, pidStr+".jsonl")
				if f, err := os.OpenFile(mb, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600); err == nil {
					f.Write(payload)
					f.Close()
				}
			}
		}
		rw.WriteHeader(204)
	})
	http.HandleFunc("/events", func(rw http.ResponseWriter, r *http.Request) {
		fl, ok := rw.(http.Flusher)
		if !ok {
			http.Error(rw, "no stream", 500)
			return
		}
		rw.Header().Set("Content-Type", "text/event-stream")
		rw.Header().Set("Cache-Control", "no-cache")
		ch := make(chan string, 16)
		w.mu.Lock()
		w.subs[ch] = true
		w.mu.Unlock()
		defer func() { w.mu.Lock(); delete(w.subs, ch); w.mu.Unlock() }()
		for {
			select {
			case msg := <-ch:
				fmt.Fprintf(rw, "data: %s\n\n", msg)
				fl.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})
	fmt.Printf("castle world on %s — %d rooms, watching %s\n", addr, len(rooms), path)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintln(os.Stderr, "castle serve:", err)
		os.Exit(1)
	}
}
