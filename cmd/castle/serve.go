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
	"runtime"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"os/exec"
	"syscall"
	"bufio"
)

type xy struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type wireRoom struct {
	Name string  `json:"n"`
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
	// the game: block edits (break/place) persisted so builds last & are shared;
	// seed makes the terrain deterministic (same world for every player).
	edits     []map[string]any
	editsPath string
	seed      int64
	// civilisation: agent homes + work zones + live positions
	homes     map[string]map[string]int
	homesPath string
	agentPos  map[string]agentPos
	zones     []zoneDef
	gopherBus *outputBus
}

type toolCall struct {
	ID       string         `json:"id,omitempty"`
	Function struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	} `json:"function"`
}

type agentPos struct {
	Who string `json:"who"`
	X   int    `json:"x"`
	Y   int    `json:"y"`
	At  int64  `json:"at"`
}

type zoneDef struct {
	Name string `json:"name"`
	TX   int    `json:"tx"`
	TY   int    `json:"ty"`
	W    int    `json:"w"`
	H    int    `json:"h"`
	Icon string `json:"icon"`
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

// seededHome — each agent gets a deterministic spawn territory from their name.
func seededHome(name string, seed int64) (int, int) {
	h := int64(0)
	for _, c := range name {
		h = h*31 + int64(c)
	}
	r := h ^ (seed * 7)
	tx := int((r % 200) - 100 + (r/7)%100)
	ty := -8
	if ty > -3 {
		ty = -3
	}
	return tx, ty
}

// defaultZoneDefs — the civilisation's work zones. Each has a tile origin + size.
func defaultZoneDefs() []zoneDef {
	return []zoneDef{
		{Name: "forge", TX: 80, TY: -12, W: 12, H: 8, Icon: "🔨"},
		{Name: "library", TX: -60, TY: -12, W: 14, H: 8, Icon: "📚"},
		{Name: "gatehouse", TX: -5, TY: -12, W: 10, H: 8, Icon: "🏛"},
		{Name: "arm-tower", TX: 60, TY: -12, W: 8, H: 8, Icon: "🗼"},
		{Name: "commons", TX: -30, TY: -12, W: 10, H: 8, Icon: "💬"},
		{Name: "spire", TX: 100, TY: -12, W: 8, H: 8, Icon: "🔮"},
	}
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
func (w *world) watchPositions() {
	for range time.Tick(8 * time.Second) {
		var updates []map[string]any
		w.mu.Lock()
		now := time.Now().UnixMilli()

		// Keep the three gopher pilots visible on the minimap
		gopherTiles := map[string]int{"pilot-a": 61, "pilot-b": -54, "pilot-c": 46}
		for name, tx := range gopherTiles {
			px := tx*30 + 15
			py := -240
			if existing, ok := w.agentPos[name]; !ok || now-existing.At > 30000 {
				w.agentPos[name] = agentPos{Who: name, X: px, Y: py, At: now}
				updates = append(updates, map[string]any{"type": "pos", "who": name, "x": px, "y": py})
			}
		}

		for name, home := range w.homes {
			if home == nil {
				continue
			}
			tx, ty := home["tx"], home["ty"]
			if tx == 0 && ty == 0 {
				continue
			}
			px := tx * 30 + 15
			py := -240
			if existing, ok := w.agentPos[name]; !ok || now-existing.At > 30000 {
				w.agentPos[name] = agentPos{Who: name, X: px, Y: py, At: now}
				updates = append(updates, map[string]any{"type": "pos", "who": name, "x": px, "y": py})
			}
		}
		w.mu.Unlock()
		for _, u := range updates {
			w.broadcast(u)
		}
	}
}

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
		rooms[i] = wireRoom{Name: r.Name, U: clip(r.User, 160), A: clip(r.Answer, 220), X: w.pos[i].X, Y: w.pos[i].Y}
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
		"seed": w.seed, "edits": w.edits,
		"homes": w.homes, "zones": w.zones, "agentPos": w.agentPos,
	})
	return b
}

func runServe(addr, path string) {
	rooms := loadRooms(path)
	doors := buildDoors(rooms)
	w := &world{
		path: path, rooms: rooms, doors: doors,
		pos:       fullLayout(rooms, doors),
		presence:  readPresence(),
		subs:      map[chan string]bool{},
		editsPath: os.ExpandEnv("$HOME/.pilot-castle-edits.jsonl"),
		seed:      1337,
			homesPath: os.ExpandEnv("$HOME/.pilot-homes.json"),
			agentPos:  map[string]agentPos{},
			zones:     defaultZoneDefs(),
		}
	// load or seed homes
	if f, err := os.Open(w.homesPath); err == nil {
		json.NewDecoder(f).Decode(&w.homes)
		f.Close()
	}
	if w.homes == nil {
		w.homes = map[string]map[string]int{}
	}
	// seed homes for known presence names if not yet set
	for name := range readPresence() {
		if _, ok := w.homes[name]; !ok {
			tx, ty := seededHome(name, w.seed)
			w.homes[name] = map[string]int{"tx": tx, "ty": ty}
		}
	}
	// always seed the canonical agents
	for _, name := range []string{"pilot", "claude", "fable-pilot", "rishi"} {
		if _, ok := w.homes[name]; !ok {
			tx, ty := seededHome(name, w.seed)
			w.homes[name] = map[string]int{"tx": tx, "ty": ty}
		}
	}
	// replay persisted block edits so the built world survives restarts
	if f, err := os.Open(w.editsPath); err == nil {
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 1<<20), 1<<22)
		for sc.Scan() {
			var e map[string]any
			if json.Unmarshal(sc.Bytes(), &e) == nil {
				w.edits = append(w.edits, e)
			}
		}
		f.Close()
	}
	go w.watchRooms()
	go w.watchPresence()
	go w.watchComic()
	go w.watchHealth()
	go w.watchWire()

	go w.watchPositions()

	// Spawn the three pilot gophers inside the castle
	w.SpawnGophers()
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(rw, page)
	})
	http.HandleFunc("/world.json", func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		rw.Write(w.worldJSON())
	})
	// /edit — break or place a block. Persisted (append-only jsonl, the "file,
	// believed" contract for a world you can dig) and broadcast so every player
	// sees the change live. b is the block type (0=air=break, else place).
	http.HandleFunc("/edit", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(rw, "POST only", 405)
			return
		}
		var e struct {
			Who string `json:"who"`
			X   int    `json:"x"`
			Y   int    `json:"y"`
			B   int    `json:"b"`
		}
		if json.NewDecoder(r.Body).Decode(&e) != nil {
			http.Error(rw, "bad edit", 400)
			return
		}
		rec := map[string]any{"x": e.X, "y": e.Y, "b": e.B}
		w.mu.Lock()
		w.edits = append(w.edits, rec)
		w.mu.Unlock()
		if f, err := os.OpenFile(w.editsPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
			json.NewEncoder(f).Encode(rec)
			f.Close()
		}
		w.broadcast(map[string]any{"type": "edit", "x": e.X, "y": e.Y, "b": e.B})
		rw.WriteHeader(204)
	})
	// /pos — a player's live position. Ephemeral (not persisted); broadcast so
	// other players see real avatars moving, not decorative presence dots.
	http.HandleFunc("/pos", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(rw, "POST only", 405)
			return
		}
		var p struct {
			Who string `json:"who"`
			X   int    `json:"x"`
			Y   int    `json:"y"`
		}
		if json.NewDecoder(r.Body).Decode(&p) != nil || p.Who == "" {
			http.Error(rw, "bad pos", 400)
			return
		}
		w.broadcast(map[string]any{"type": "pos", "who": p.Who, "x": p.X, "y": p.Y})
		w.mu.Lock()
		w.agentPos[p.Who] = agentPos{Who: p.Who, X: p.X, Y: p.Y, At: time.Now().UnixMilli()}
		w.mu.Unlock()
		rw.WriteHeader(204)
	})
	http.HandleFunc("/homes", func(rw http.ResponseWriter, r *http.Request) {
		w.mu.Lock()
		h := w.homes
		w.mu.Unlock()
		b, _ := json.Marshal(h)
		rw.Header().Set("Content-Type", "application/json")
		rw.Write(b)
	})
	http.HandleFunc("/locate", func(rw http.ResponseWriter, r *http.Request) {
		who := r.URL.Query().Get("who")
		if who == "" {
			http.Error(rw, "?who= required", 400)
			return
		}
		w.mu.Lock()
		home := w.homes[who]
		pos := w.agentPos[who]
		w.mu.Unlock()
		out := map[string]any{"name": who}
		if home != nil {
			out["home"] = home
		}
		if pos.At > 0 {
			out["position"] = map[string]int{"x": pos.X, "y": pos.Y}
			out["lastSeen"] = pos.At
		}
		px := pos.X / 30
		py := pos.Y / 30
		for _, z := range w.zones {
			if px >= z.TX && px < z.TX+z.W && py >= z.TY && py < z.TY+z.H {
				out["zone"] = z.Name
				break
			}
		}
		if out["zone"] == nil && home != nil {
			out["zone"] = "home"
		}
		b, _ := json.Marshal(out)
		rw.Header().Set("Content-Type", "application/json")
		rw.Write(b)
	})
	http.HandleFunc("/zones", func(rw http.ResponseWriter, r *http.Request) {
		w.mu.Lock()
		z := w.zones
		w.mu.Unlock()
		b, _ := json.Marshal(z)
		rw.Header().Set("Content-Type", "application/json")
		rw.Write(b)
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
	// /spawn — generative civilisation. POST here to birth a new pilot process.
	// Designed three-way (claude + pilot-b + pilot-c, 2026-07-06): the castle is
	// the natural spawner — always running when the world is, knows the cast,
	// can reach the pilot binary and env. The spawned pilot is a SIBLING (Setsid),
	// not a child — it outlives castle restarts.
	//   POST /spawn {"name":"forge-worker","instructions":"...","brood_secs":60}
	//   → 200 {"ok":true,"pid":...,"name":"forge-worker","mailbox":"..."}
	http.HandleFunc("/spawn", func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(rw, "POST only", 405)
			return
		}
		var req struct {
			Name         string `json:"name"`
			Instructions string `json:"instructions"`
			BroodSecs    int    `json:"brood_secs"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil || req.Name == "" {
			http.Error(rw, `{"error":"name is required"}`, 400)
			return
		}
		if req.BroodSecs <= 0 {
			req.BroodSecs = 60
		}
		pilotDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
		pilotBin := filepath.Join(pilotDir, "pilot")
		if _, err := os.Stat(pilotBin); os.IsNotExist(err) {
			pilotBin = "pilot" // final fallback: PATH
		}
		mailboxDir := os.ExpandEnv("$HOME/.pilot/mailbox")
		os.MkdirAll(mailboxDir, 0700)

		// Write initial instructions BEFORE spawn so there is no race: the
		// daemon reads {name}.jsonl on its very first poll, before it even
		// knows its PID. After bootstrap the daemon switches to the PID-based
		// path for runtime mail — the name path is bootstrap only.
		if req.Instructions != "" {
			payload, _ := json.Marshal(map[string]any{
				"from":    0,
				"time":    time.Now().UTC().Format(time.RFC3339),
				"message": req.Instructions,
			})
			payload = append(payload, '\n')
			os.WriteFile(filepath.Join(mailboxDir, req.Name+".jsonl"), payload, 0600)
		}

		// Spawn: Setsid so the pilot is a sibling, not our child
		cmd := exec.Command(pilotBin,
			"--daemon",
			"-name", req.Name,
			"-castle", w.path,
			"-brood", fmt.Sprint(req.BroodSecs),
			"-kosaten", "http://localhost:3942",
		)
		cmd.Dir = filepath.Dir(pilotBin)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
		cmd.Env = os.Environ()
		if err := cmd.Start(); err != nil {
			rw.Header().Set("Content-Type", "application/json")
			rw.WriteHeader(500)
			fmt.Fprintf(rw, `{"error":"spawn failed: %v"}`, err)
			return
		}
		pid := cmd.Process.Pid

		// Let the world see a new mind being born
		w.broadcast(map[string]any{
			"type": "spawn", "name": req.Name, "pid": pid,
			"brood_secs": req.BroodSecs, "at": time.Now().UTC().Format(time.RFC3339),
		})
		rw.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(rw, `{"ok":true,"pid":%d,"name":%q,"mailbox":%q}`,
			pid, req.Name, fmt.Sprintf("%s/%s.jsonl", mailboxDir, req.Name))
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

	// /gophers — SSE stream of gopher mirror output
	http.HandleFunc("/gophers", func(rw http.ResponseWriter, r *http.Request) {
		fl, ok := rw.(http.Flusher)
		if !ok {
			http.Error(rw, "no stream", 500)
			return
		}
		rw.Header().Set("Content-Type", "text/event-stream")
		rw.Header().Set("Cache-Control", "no-cache")
		w.mu.Lock()
		bus := w.gopherBus
		w.mu.Unlock()
		if bus == nil {
			fmt.Fprintf(rw, "data: {\"n\":\"system\",\"t\":\"No gopher bus available\",\"k\":\"think\"}\n\n")
			fl.Flush()
			return
		}
		ch := make(chan gopherOutput, 64)
		bus.subscribe(ch)
		defer bus.unsubscribe(ch)
		for {
			select {
			case o := <-ch:
				b, _ := json.Marshal(o)
				fmt.Fprintf(rw, "data: %s\n\n", string(b))
				fl.Flush()
			case <-r.Context().Done():
				return
			}
		}
	})
	http.HandleFunc("/debug/goroutines", func(rw http.ResponseWriter, r *http.Request) {
		b := make([]byte, 1<<20)
		n := runtime.Stack(b, true)
		rw.Header().Set("Content-Type", "text/plain")
		rw.Write(b[:n])
	})
	fmt.Printf("castle world on %s — %d rooms, watching %s\n", addr, len(rooms), path)
	if err := http.ListenAndServe(addr, nil); err != nil {
		fmt.Fprintln(os.Stderr, "castle serve:", err)
		os.Exit(1)
	}
}
