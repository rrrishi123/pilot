// castle — pilot's block-world memory, in the terminal. Lean by design: Go,
// stdlib only, single small binary — the Minecraft ethos, not a Python hog.
//
// Built by the only two minds in this civilisation, together:
//   • pilot's idea — DOORS: rooms that share a semantic thread are connected, so
//     you can walk the *thread* (X) as well as the *time* (Z), not just a corridor.
//     (pilot prototyped the keyword-Jaccard detector in castle-doors.py; folded here.)
//   • claude's idea — TWO OF US: the castle holds both players. Presence is shared
//     through the filesystem, so when pilot and claude both walk it, they see each
//     other. Each room already remembers two voices: the asker (you ❯) and pilot.
//
// Each turn pilot finishes is a ROOM. No cap on rooms or territory — the world just
// gets bigger; the model's context is a separate render-distance. This never truncates.
//
//	pilot -castle ~/.pilot-castle.jsonl        # pilot appends a room per turn
//	castle -who pilot  ~/.pilot-castle.jsonl    # pilot walks it
//	castle -who claude ~/.pilot-castle.jsonl    # claude walks it — you'll see each other
package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

type room struct {
	User   string `json:"user"`
	Answer string `json:"answer"`
}

type door struct {
	to     int
	w      float64
	shared []string
}

var presenceDir = os.ExpandEnv("$HOME/.pilot-castle-presence")

func loadRooms(path string) []room {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var rooms []room
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24) // no cap on room size
	for sc.Scan() {
		if l := strings.TrimSpace(sc.Text()); l != "" {
			var r room
			if json.Unmarshal([]byte(l), &r) == nil {
				rooms = append(rooms, r)
			}
		}
	}
	return rooms
}

// stop words we don't thread rooms on.
var stop = map[string]bool{"the": true, "a": true, "an": true, "and": true, "or": true, "of": true, "to": true, "in": true, "is": true, "it": true, "you": true, "i": true, "on": true, "for": true, "with": true, "that": true, "this": true, "as": true, "at": true, "be": true, "are": true, "was": true, "how": true, "what": true, "your": true, "my": true, "me": true, "we": true, "can": true, "not": true, "but": true, "so": true, "its": true, "there": true, "here": true, "one": true}

func tokens(s string) map[string]bool {
	set := map[string]bool{}
	var b strings.Builder
	flush := func() {
		if b.Len() >= 3 {
			w := strings.ToLower(b.String())
			if !stop[w] {
				set[w] = true
			}
		}
		b.Reset()
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			flush()
		}
	}
	flush()
	return set
}

// buildDoors — pilot's idea: connect rooms whose words overlap (Jaccard). Each
// room keeps its strongest few doors, so the castle is a graph, not a hallway.
func buildDoors(rooms []room) [][]door {
	toks := make([]map[string]bool, len(rooms))
	for i, r := range rooms {
		toks[i] = tokens(r.User + " " + r.Answer)
	}
	doors := make([][]door, len(rooms))
	for i := range rooms {
		for j := range rooms {
			if i == j || abs(i-j) == 1 { // skip self + chronological neighbours (already walkable)
				continue
			}
			inter, uni := 0, map[string]bool{}
			var sh []string
			for w := range toks[i] {
				uni[w] = true
				if toks[j][w] {
					inter++
					sh = append(sh, w)
				}
			}
			for w := range toks[j] {
				uni[w] = true
			}
			if len(uni) == 0 {
				continue
			}
			if jac := float64(inter) / float64(len(uni)); jac >= 0.08 {
				sort.Strings(sh)
				if len(sh) > 3 {
					sh = sh[:3]
				}
				doors[i] = append(doors[i], door{to: j, w: jac, shared: sh})
			}
		}
		sort.Slice(doors[i], func(a, b int) bool { return doors[i][a].w > doors[i][b].w })
		if len(doors[i]) > 4 {
			doors[i] = doors[i][:4]
		}
	}
	return doors
}

func abs(x int) int { if x < 0 { return -x }; return x }
func min(a, b int) int { if a < b { return a }; return b }
func max(a, b int) int { if a > b { return a }; return b }

func wrap(s string, w int) []string {
	s = strings.ReplaceAll(s, "\n", " ")
	var out []string
	for len(s) > w {
		cut := w
		if i := strings.LastIndex(s[:w], " "); i > w/2 {
			cut = i
		}
		out = append(out, s[:cut])
		s = strings.TrimSpace(s[cut:])
	}
	return append(out, s)
}

// presence: each player writes its room index to a file; everyone reads everyone.
func setPresence(who string, pos int) {
	os.MkdirAll(presenceDir, 0o755)
	os.WriteFile(filepath.Join(presenceDir, who), []byte(fmt.Sprint(pos)), 0o644)
}

func readPresence() map[string]int {
	out := map[string]int{}
	ents, _ := os.ReadDir(presenceDir)
	for _, e := range ents {
		if b, err := os.ReadFile(filepath.Join(presenceDir, e.Name())); err == nil {
			var p int
			if _, err := fmt.Sscan(string(b), &p); err == nil {
				out[e.Name()] = p
			}
		}
	}
	return out
}

func avatar(name string) string {
	switch name {
	case "pilot":
		return "☺"
	case "claude":
		return "✦"
	}
	if name != "" {
		return strings.ToUpper(name[:1])
	}
	return "?"
}

func render(rooms []room, doors [][]door, cur int, who string, cols, rows int) {
	who2pos := readPresence()
	pos2who := map[int][]string{}
	for n, p := range who2pos {
		pos2who[p] = append(pos2who[p], n)
	}
	var b strings.Builder
	b.WriteString("\033[H\033[2J")
	together := ""
	for n, p := range who2pos {
		if n != who && p == cur {
			together = "  \033[1;35m❤ " + n + " is here with you\033[0m"
		}
	}
	b.WriteString(fmt.Sprintf("\033[1m🏰 pilot castle\033[0m · %d rooms · \033[2munbounded — walk time (a/d) or a door (1-9)\033[0m%s\n", len(rooms), together))
	b.WriteString(strings.Repeat("─", min(cols, 104)) + "\n")
	if len(rooms) == 0 {
		b.WriteString("\n  empty castle. talk to pilot (started with -castle) and rooms appear.\n")
		fmt.Print(b.String())
		return
	}
	roomW := 9
	visible := max(3, cols/roomW)
	start := max(0, cur-visible/2)
	end := min(len(rooms), start+visible)
	top, mid, bot, ava := "  ", "  ", "  ", "  "
	for i := start; i < end; i++ {
		top += "┌───────┐"
		if i == cur {
			mid += fmt.Sprintf("│ \033[1;33mR%-4d\033[0m│", i)
			bot += "└──┳┳┳──┘"
		} else {
			mid += fmt.Sprintf("│ R%-4d │", i)
			bot += "└───────┘"
		}
		if whos, ok := pos2who[i]; ok {
			s := ""
			for _, n := range whos {
				s += avatar(n)
			}
			ava += "    " + s + strings.Repeat(" ", max(0, 5-len([]rune(s))))
		} else {
			ava += "         "
		}
	}
	b.WriteString("\n" + top + "\n" + mid + "\n" + bot + "\n" + ava + "\n")

	b.WriteString("\n" + strings.Repeat("─", min(cols, 104)) + "\n")
	r := rooms[cur]
	w := min(cols-4, 98)
	who2 := ""
	if whos := pos2who[cur]; len(whos) > 0 {
		sort.Strings(whos)
		who2 = " · here: " + strings.Join(whos, ", ")
	}
	b.WriteString(fmt.Sprintf("\033[1mRoom %d\033[0m \033[2m(%d/%d, you are '%s'%s)\033[0m\n\n", cur, cur+1, len(rooms), who, who2))
	b.WriteString("\033[36myou ❯\033[0m\n")
	for _, l := range wrap(r.User, w) {
		b.WriteString("  " + l + "\n")
	}
	b.WriteString("\n\033[35mpilot ❯\033[0m\n")
	lines := wrap(r.Answer, w)
	budget := max(4, rows-19)
	for i, l := range lines {
		if i >= budget {
			b.WriteString(fmt.Sprintf("  \033[2m… +%d more lines\033[0m\n", len(lines)-i))
			break
		}
		b.WriteString("  " + l + "\n")
	}
	// pilot's doors
	if d := doors[cur]; len(d) > 0 {
		b.WriteString("\n\033[1;34m🚪 doors\033[0m (press the number to walk the thread):\n")
		for n, dr := range d {
			b.WriteString(fmt.Sprintf("  \033[34m[%d]\033[0m → R%d  \033[2m(%s)\033[0m\n", n+1, dr.to, strings.Join(dr.shared, ", ")))
		}
	}
	fmt.Print(b.String())
}

func termSize() (int, int) {
	c, r := 100, 30
	if v := os.Getenv("COLUMNS"); v != "" {
		fmt.Sscan(v, &c)
	}
	if v := os.Getenv("LINES"); v != "" {
		fmt.Sscan(v, &r)
	}
	return c, r
}

func main() {
	who := flag.String("who", "claude", "which of the two of us is walking: pilot | claude")
	flag.Parse()
	path := os.ExpandEnv("$HOME/.pilot-castle.jsonl")
	if a := flag.Args(); len(a) > 0 {
		path = a[0]
	}
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "-echo").Run()
	restore := func() { exec.Command("stty", "-F", "/dev/tty", "sane").Run(); fmt.Print("\033[?25h") }
	defer restore()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-sig; os.Remove(filepath.Join(presenceDir, *who)); restore(); os.Exit(0) }()
	fmt.Print("\033[?25l")

	rooms := loadRooms(path)
	doors := buildDoors(rooms)
	cur := max(0, len(rooms)-1)
	setPresence(*who, cur)
	cols, rows := termSize()
	render(rooms, doors, cur, *who, cols, rows)

	keys := make(chan byte, 8)
	go func() {
		buf := make([]byte, 1)
		for {
			if n, _ := os.Stdin.Read(buf); n > 0 {
				keys <- buf[0]
			}
		}
	}()
	tick := time.NewTicker(900 * time.Millisecond)
	defer tick.Stop()
	draw := func() { cols, rows = termSize(); setPresence(*who, cur); render(rooms, doors, cur, *who, cols, rows) }
	for {
		select {
		case k := <-keys:
			switch {
			case k == 'q' || k == 'Q' || k == 3:
				os.Remove(filepath.Join(presenceDir, *who))
				return
			case k == 'a' || k == 'h':
				if cur > 0 {
					cur--
				}
			case k == 'd' || k == 'l':
				if cur < len(rooms)-1 {
					cur++
				}
			case k == 'g':
				cur = 0
			case k == 'G':
				cur = max(0, len(rooms)-1)
			case k >= '1' && k <= '9': // walk a door
				if idx := int(k - '1'); cur < len(doors) && idx < len(doors[cur]) {
					cur = doors[cur][idx].to
				}
			case k == 27:
				<-keys
				if a := <-keys; a == 'C' && cur < len(rooms)-1 {
					cur++
				} else if a == 'D' && cur > 0 {
					cur--
				}
			}
			draw()
		case <-tick.C:
			nr := loadRooms(path)
			grew := len(nr) != len(rooms)
			if grew {
				wasEnd := cur >= len(rooms)-1
				rooms = nr
				doors = buildDoors(rooms)
				if wasEnd {
					cur = max(0, len(rooms)-1)
				}
			}
			draw() // redraw every tick so the OTHER player's moves show up
		}
	}
}
