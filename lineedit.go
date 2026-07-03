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
// (Linux), kitty (Linux), and Windows Terminal all use:
//
//	Control char     e.g. Ctrl+W = 0x17, plain Backspace = 0x7f
//	ESC+letter       emacs meta: Alt+b = ESC+b = word-left (iTerm2 default)
//	                 Alt+f = ESC+f = word-right, Alt+d = delete-word-forward
//	CSI sequence     ESC [ ... (cursor keys, Home/End, modified variants)
//
// iTerm2 ships with "Left/Right Option key acts as +Esc" by default — so Mac
// users get Alt+b/f for word-move out of the box. If they switch to "Send CSI
// sequence", the CSI variants work too.
type lineReader struct {
	in       *bufio.Reader
	fd       int
	histFile string
	hist     []string
	curRow   int
	preload  string // set by interrupt recovery
}

func newLineReader(histFile string) *lineReader {
	lr := &lineReader{in: bufio.NewReader(os.Stdin), fd: int(os.Stdin.Fd()), histFile: histFile}
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

// render draws the prompt + buffer, parks the cursor.
func (lr *lineReader) render(prompt string, buf []rune, cursor int) {
	if lr.curRow > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dA", lr.curRow)
	}
	fmt.Fprint(os.Stdout, "\r\033[J")
	fmt.Fprint(os.Stdout, prompt)
	for _, ch := range buf {
		if ch == '\n' {
			fmt.Fprint(os.Stdout, "\r\n")
		} else {
			fmt.Fprint(os.Stdout, string(ch))
		}
	}
	row, col := 0, 0
	for i := 0; i < cursor && i < len(buf); i++ {
		if buf[i] == '\n' {
			row++
			col = 0
		} else {
			col++
		}
	}
	if up := strings.Count(string(buf), "\n") - row; up > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dA", up)
	}
	fmt.Fprint(os.Stdout, "\r")
	tcol := col
	if row == 0 {
		tcol += len([]rune(prompt))
	}
	if tcol > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dC", tcol)
	}
	lr.curRow = row
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

func wordLeftPos(buf []rune, pos int) int {
	for pos > 0 && buf[pos-1] == ' ' {
		pos--
	}
	for pos > 0 && buf[pos-1] != ' ' {
		pos--
	}
	return pos
}

func wordRightPos(buf []rune, pos int) int {
	for pos < len(buf) && buf[pos] != ' ' {
		pos++
	}
	for pos < len(buf) && buf[pos] == ' ' {
		pos++
	}
	return pos
}

func killWordBack(buf []rune, cursor int) ([]rune, int) {
	i := cursor
	for i > 0 && buf[i-1] == ' ' {
		i--
	}
	for i > 0 && buf[i-1] != ' ' {
		i--
	}
	return append(buf[:i], buf[cursor:]...), i
}

func killWordForward(buf []rune, cursor int) ([]rune, int) {
	i := cursor
	for i < len(buf) && buf[i] == ' ' {
		i++
	}
	for i < len(buf) && buf[i] != ' ' {
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

	buf := []rune(lr.preload)
	lr.preload = ""
	cursor := len(buf)
	lr.curRow = 0
	histIdx := len(lr.hist)
	var saved []rune

	lr.render(prompt, buf, cursor)

	for {
		r, _, err := lr.in.ReadRune()
		if err != nil {
			return "", err
		}

		switch r {

		case '\r': // Enter — submit, or line-continuation via trailing \
			s := strings.TrimRight(string(buf), " ")
			if strings.HasSuffix(s, "\\") && len(s) > 0 {
				trail := len(buf) - len([]rune(s)) + 1
				buf = buf[:len(buf)-trail]
				buf = append(buf, '\n')
				cursor = len(buf)
				lr.render(prompt, buf, cursor)
				continue
			}
			if down := strings.Count(string(buf), "\n") - lr.curRow; down > 0 {
				fmt.Fprintf(os.Stdout, "\033[%dB", down)
			}
			fmt.Fprint(os.Stdout, "\r\n")
			lr.curRow = 0
			line := string(buf)
			if record && strings.TrimSpace(line) != "" {
				lr.hist = append(lr.hist, line)
			}
			return line, nil

		case 3: // Ctrl+C — abort
			fmt.Fprint(os.Stdout, "\r\n")
			return "", errAborted

		case 4: // Ctrl+D — EOF on empty, ignored on non-empty
			if len(buf) == 0 {
				fmt.Fprint(os.Stdout, "\r\n")
				return "", io.EOF
			}
			// non-empty: fall through — no-op, still re-render

		case '\n': // Ctrl+J — newline
			buf = append(buf[:cursor], append([]rune{'\n'}, buf[cursor:]...)...)
			cursor++

		case 127, 8: // Backspace (DEL, BS)
			if cursor > 0 {
				buf = append(buf[:cursor-1], buf[cursor:]...)
				cursor--
			}

		case 1: // Ctrl+A — start of line
			cursor = 0
		case 5: // Ctrl+E — end of line
			cursor = len(buf)
		case 21: // Ctrl+U — kill to start
			buf = append([]rune{}, buf[cursor:]...)
			cursor = 0
		case 11: // Ctrl+K — kill to end
			buf = buf[:cursor]
		case 23: // Ctrl+W — delete word back
			buf, cursor = killWordBack(buf, cursor)

		case 27: // ESC — read the rest
			seq := lr.readEscSeq()

			switch seq {
			// arrows
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

			// Home / End
			case "[H", "[1~", "OH":
				cursor = 0
			case "[F", "[4~", "OF":
				cursor = len(buf)

			// Delete
			case "[3~":
				if cursor < len(buf) {
					buf = append(buf[:cursor], buf[cursor+1:]...)
				}

			// word-left: ESC+b (iTerm2 default), CSI, kitty
			case "b":
				cursor = wordLeftPos(buf, cursor)
			case "[1;3D", "[1;5D", "[1;4D", "[1;6D", "[5D":
				cursor = wordLeftPos(buf, cursor)
			case "[27;5;113~", "[27;3;113~":
				cursor = wordLeftPos(buf, cursor)

			// word-right: ESC+f, CSI, kitty
			case "f":
				cursor = wordRightPos(buf, cursor)
			case "[1;3C", "[1;5C", "[1;4C", "[1;6C", "[5C":
				cursor = wordRightPos(buf, cursor)
			case "[27;5;115~", "[27;3;115~":
				cursor = wordRightPos(buf, cursor)

			// delete-word-back: Ctrl+Backspace / Alt+Backspace / Opt+Backspace
			case "\x7f", "\x08": // ESC+DEL / ESC+BS (xterm, iTerm2)
				buf, cursor = killWordBack(buf, cursor)
			case "[8;127u", "[127;8u": // kitty CSI u
				buf, cursor = killWordBack(buf, cursor)
			case "[27;8;127~", "[27;5;127~", "[27;3;127~": // kitty modifyOtherKeys
				buf, cursor = killWordBack(buf, cursor)

			// delete-word-forward: Alt+d / Opt+⌦
			case "d":
				buf, cursor = killWordForward(buf, cursor)

			// newline: Shift+Enter, Alt+Enter
			case "[27;2;13~", "[13;2u", "\r":
				buf = append(buf[:cursor], append([]rune{'\n'}, buf[cursor:]...)...)
				cursor++
			}

		default:
			if r >= 32 {
				buf = append(buf[:cursor], append([]rune{r}, buf[cursor:]...)...)
				cursor++
			}
		}

		// Every path that didn't return renders once here.
		lr.render(prompt, buf, cursor)
	}
}
