package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

var errAborted = errors.New("line aborted")
var errInterrupted = errors.New("turn interrupted — edit and resubmit")

// lineReader is a terminal line editor. Designed portably: understands the three
// wire formats that iTerm2 (Mac), Terminal.app (Mac), xterm/gnome-terminal
// (Linux), kitty (Linux), and Windows Terminal all use.
type lineReader struct {
	in       *bufio.Reader
	fd       int
	histFile string
	hist     []string
	termW    int    // terminal width (cols); 0 if unknown
	oldRows  int    // how many terminal rows the PREVIOUS render used
	preload  string // set by interrupt recovery
}

func newLineReader(histFile string) *lineReader {
	lr := &lineReader{in: bufio.NewReader(os.Stdin), fd: int(os.Stdin.Fd()), histFile: histFile}
	if w, _, err := term.GetSize(lr.fd); err == nil && w > 0 {
		lr.termW = w
	}
	if b, err := os.ReadFile(histFile); err == nil {
		for _, ln := range strings.Split(string(b), "\n") {
			if ln = strings.TrimRight(ln, "\r"); ln != "" {
				lr.hist = append(lr.hist, ln)
			}
		}
	}
	return lr
}

func (lr *lineReader) saveHistory() {
	if lr.histFile == "" {
		return
	}
	f, err := os.Create(lr.histFile)
	if err != nil {
		return
	}
	defer f.Close()
	for _, h := range lr.hist {
		fmt.Fprintln(f, strings.ReplaceAll(h, "\n", " "))
	}
}

// displayCols returns how many visual columns s occupies (adding promptLen if on
// the first buffer row). Wrapping is counted via lr.termW (0 = no wrapping).
func (lr *lineReader) displayCols(s string, promptLen int) int {
	cols := promptLen
	for _, ch := range s {
		if ch == '\n' {
			cols = 0
		} else {
			cols++
		}
		if lr.termW > 0 && cols >= lr.termW {
			cols = 0
		}
	}
	return cols
}

// renderRows counts how many terminal rows the current prompt+buf would use.
func (lr *lineReader) renderRows(promptLen int, buf []rune) int {
	rows := 0
	col := promptLen
	for _, ch := range buf {
		col++
		if ch == '\n' || (lr.termW > 0 && col >= lr.termW) {
			rows++
			col = 0
		}
	}
	return rows
}

// render draws prompt + buffer and positions the cursor. Cursor is hidden during
// the redraw so there is no perceived flicker.
func (lr *lineReader) render(prompt string, buf []rune, cursor int) {
	fmt.Fprint(os.Stdout, "\033[?25l") // hide cursor

	// Move back to the start of the previous render
	if lr.oldRows > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dA", lr.oldRows)
	}
	fmt.Fprint(os.Stdout, "\r")

	// Draw the new content
	promptLen := len([]rune(prompt))
	fmt.Fprint(os.Stdout, prompt)
	var lastCh rune
	for _, ch := range buf {
		if ch == '\n' {
			fmt.Fprint(os.Stdout, "\r\n")
		} else {
			fmt.Fprint(os.Stdout, string(ch))
		}
		lastCh = ch
	}

	// Count new render height and clear any leftover from the old render
	newRows := lr.renderRows(promptLen, buf)
	if lastCh == '\n' {
		newRows++ // trailing newline adds a blank row
	}
	fmt.Fprint(os.Stdout, "\033[J") // clear to end of screen (only visible if old > new)

	// Navigate cursor to the correct position
	row, col := 0, promptLen
	for i := 0; i < cursor && i < len(buf); i++ {
		if buf[i] == '\n' {
			row++
			col = 0
		} else {
			col++
			if lr.termW > 0 && col >= lr.termW {
				row++
				col = 0
			}
		}
	}
	up := newRows - row
	if up > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dA", up)
	}
	fmt.Fprint(os.Stdout, "\r")
	if col > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dC", col)
	}

	lr.oldRows = newRows               // total height of what we just drew — next render moves back this many
	fmt.Fprint(os.Stdout, "\033[?25h") // show cursor
}

// readEscSeq reads bytes after ESC. Returns the full sequence string.
func (lr *lineReader) readEscSeq() string {
	var sb strings.Builder
	for {
		b, err := lr.in.ReadByte()
		if err != nil {
			break
		}
		sb.WriteByte(b)
		if b >= 0x40 && b <= 0x7e && b != '[' && b != 'O' {
			break
		}
		if b < 0x20 {
			break
		}
	}
	return sb.String()
}

// ── word helpers ──────────────────────────────────────────────────────────────

// readPaste reads characters after a bracketed paste start ([200~).
// Returns all characters up to (but not including) the [201~ terminator.
// This avoids rendering after every single character of a large paste.
func (lr *lineReader) readPaste() []rune {
	var out []rune
	for {
		r, _, err := lr.in.ReadRune()
		if err != nil {
			return out
		}
		if r == 27 {
			seq := lr.readEscSeq()
			if seq == "[201~" {
				return out
			}
			continue
		}
		out = append(out, r)
	}
}

// a word boundary is whitespace — space OR newline. Treating '\n' as a
// separator keeps word-motions and word-deletes sane in multi-line input
// (Ctrl+Backspace stops at the line above, it doesn't swallow the whole buffer).
func isWordSep(r rune) bool { return r == ' ' || r == '\n' || r == '\t' }

func wordLeftPos(buf []rune, pos int) int {
	for pos > 0 && isWordSep(buf[pos-1]) {
		pos--
	}
	for pos > 0 && !isWordSep(buf[pos-1]) {
		pos--
	}
	return pos
}

func wordRightPos(buf []rune, pos int) int {
	for pos < len(buf) && !isWordSep(buf[pos]) {
		pos++
	}
	for pos < len(buf) && isWordSep(buf[pos]) {
		pos++
	}
	return pos
}

func killWordBack(buf []rune, cursor int) ([]rune, int) {
	i := cursor
	for i > 0 && isWordSep(buf[i-1]) {
		i--
	}
	for i > 0 && !isWordSep(buf[i-1]) {
		i--
	}
	return append(buf[:i], buf[cursor:]...), i
}

func killWordForward(buf []rune, cursor int) ([]rune, int) {
	i := cursor
	for i < len(buf) && isWordSep(buf[i]) {
		i++
	}
	for i < len(buf) && !isWordSep(buf[i]) {
		i++
	}
	return append(buf[:cursor], buf[i:]...), cursor
}

// ── main loop ─────────────────────────────────────────────────────────────────

func (lr *lineReader) readLine(prompt string, record bool) (string, error) {
	old, err := term.MakeRaw(lr.fd)
	if err != nil {
		s, e := lr.in.ReadString('\n')
		return strings.TrimRight(s, "\r\n"), e
	}
	defer term.Restore(lr.fd, old)

	// Progressive enhancement: turn on the kitty keyboard protocol (flag 1,
	// "disambiguate"). THIS is the line that makes the CSI-u parser below
	// actually fire — without it a modern terminal keeps legacy encodings, so
	// Shift+Enter is an indistinguishable bare \r (submits instead of newline)
	// and Ctrl+Backspace is a plain 0x08. With it, those keys arrive as
	// unambiguous \x1b[13;2u / \x1b[127;5u. Terminals that don't speak the
	// protocol silently drop the escape and fall back to legacy — no harm.
	// Popped (LIFO, before Restore) so the terminal is left as we found it.
	fmt.Fprint(os.Stdout, "\033[>1u")

	// Enable bracketed paste so pasted text is wrapped in [200~...[201~.
	// This lets us batch-insert the whole paste with a single render instead of
	// processing each character individually (which would render once per char).
	fmt.Fprint(os.Stdout, "\033[?2004h")
	defer fmt.Fprint(os.Stdout, "\033[?2004l")

	defer fmt.Fprint(os.Stdout, "\033[<u")

	// Refresh terminal width (may have changed since startup)
	if w, _, e := term.GetSize(lr.fd); e == nil && w > 0 {
		lr.termW = w
	}

	buf := []rune(lr.preload)
	lr.preload = ""
	cursor := len(buf)
	lr.oldRows = 0
	histIdx := len(lr.hist)
	var saved []rune

	lr.render(prompt, buf, cursor)

	for {
		r, _, err := lr.in.ReadRune()
		if err != nil {
			return "", err
		}

		switch r {

		case '\r': // Enter
			s := strings.TrimRight(string(buf), " ")
			if strings.HasSuffix(s, "\\") && len(s) > 0 {
				trail := len(buf) - len([]rune(s)) + 1
				buf = buf[:len(buf)-trail]
				buf = append(buf, '\n')
				cursor = len(buf)
				lr.render(prompt, buf, cursor)
				continue
			}
			// Move cursor below the rendered block (to a clean new line)
			promptLen := len([]rune(prompt))
			bot := lr.renderRows(promptLen, buf)
			if bot-lr.oldRows > 0 {
				fmt.Fprintf(os.Stdout, "\033[%dB", bot-lr.oldRows)
			}
			fmt.Fprint(os.Stdout, "\r\n")
			lr.oldRows = 0
			line := string(buf)
			if record && strings.TrimSpace(line) != "" {
				lr.hist = append(lr.hist, line)
			}
			return line, nil

		case 3: // Ctrl+C
			fmt.Fprint(os.Stdout, "\r\n")
			return "", errAborted

		case 4: // Ctrl+D
			if len(buf) == 0 {
				fmt.Fprint(os.Stdout, "\r\n")
				return "", io.EOF
			}

		case '\n': // Ctrl+J
			buf = append(buf[:cursor], append([]rune{'\n'}, buf[cursor:]...)...)
			cursor++

		case 127: // Backspace — delete one char
			if cursor > 0 {
				buf = append(buf[:cursor-1], buf[cursor:]...)
				cursor--
			}

		case 8: // Ctrl+Backspace on terminals that lack the kitty protocol
			// (they send raw 0x08). Modern terminals send \x7f for plain
			// backspace, so 0x08 here means the ctrl-modified key: delete word.
			buf, cursor = killWordBack(buf, cursor)

		case 1: // Ctrl+A
			cursor = 0
		case 5: // Ctrl+E
			cursor = len(buf)
		case 21: // Ctrl+U
			buf = append([]rune{}, buf[cursor:]...)
			cursor = 0
		case 11: // Ctrl+K
			buf = buf[:cursor]
		case 23: // Ctrl+W
			buf, cursor = killWordBack(buf, cursor)

		case 27: // ESC
			seq := lr.readEscSeq()

			switch seq {
			case "[A":
				if histIdx == len(lr.hist) {
					saved = append([]rune{}, buf...)
				}
				if histIdx > 0 {
					histIdx--
					buf = []rune(lr.hist[histIdx])
					cursor = len(buf)
				}
			case "[B":
				if histIdx < len(lr.hist) {
					histIdx++
					if histIdx == len(lr.hist) {
						buf = append([]rune{}, saved...)
					} else {
						buf = []rune(lr.hist[histIdx])
					}
					cursor = len(buf)
				}
			case "[C":
				if cursor < len(buf) {
					cursor++
				}
			case "[D":
				if cursor > 0 {
					cursor--
				}
			case "[H", "[1~", "OH":
				cursor = 0
			case "[F", "[4~", "OF":
				cursor = len(buf)
			case "[3~":
				if cursor < len(buf) {
					buf = append(buf[:cursor], buf[cursor+1:]...)
				}
			case "b":
				cursor = wordLeftPos(buf, cursor)
			case "[1;3D", "[1;5D", "[1;4D", "[1;6D", "[5D":
				cursor = wordLeftPos(buf, cursor)
			case "[27;5;113~", "[27;3;113~":
				cursor = wordLeftPos(buf, cursor)
			case "f":
				cursor = wordRightPos(buf, cursor)
			case "[1;3C", "[1;5C", "[1;4C", "[1;6C", "[5C":
				cursor = wordRightPos(buf, cursor)
			case "[27;5;115~", "[27;3;115~":
				cursor = wordRightPos(buf, cursor)
			case "\x7f", "\x08":
				buf, cursor = killWordBack(buf, cursor)
			case "[127;5u", "[127;3u", "[8;127u", "[127;8u": // Ctrl/Alt+Backspace (kitty)
				buf, cursor = killWordBack(buf, cursor)
			case "[27;8;127~", "[27;5;127~", "[27;3;127~":
				buf, cursor = killWordBack(buf, cursor)
			case "[3;5~", "[3;3u": // Ctrl+Delete — delete word forward
				buf, cursor = killWordForward(buf, cursor)
			case "d":
				buf, cursor = killWordForward(buf, cursor)
			case "[27;2;13~", "[13;2u", "[13;3u", "[13;5u", "\r": // Shift/Alt/Ctrl+Enter → newline
				buf = append(buf[:cursor], append([]rune{'\n'}, buf[cursor:]...)...)
				cursor++

			case "[200~": // bracketed paste start — batch-insert everything at once
				pasted := lr.readPaste()
				if len(pasted) > 0 {
					buf = append(buf[:cursor], append(pasted, buf[cursor:]...)...)
					cursor += len(pasted)
				}
			}

		default:
			if r >= 32 {
				buf = append(buf[:cursor], append([]rune{r}, buf[cursor:]...)...)
				cursor++
			}
		}

		lr.render(prompt, buf, cursor)
	}
}
