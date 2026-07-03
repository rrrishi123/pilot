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
		// final byte: >= 0x40 (but '[' and 'O' are CSI/SS3 introducers)
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

		// ── control characters ────────────────────────────────────────────
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

		case '\n': // Ctrl+J → newline
			buf = append(buf[:cursor], append([]rune{'\n'}, buf[cursor:]...)...)
			cursor++

		case 3: // Ctrl+C
			fmt.Fprint(os.Stdout, "\r\n")
			return "", errAborted

		case 4: // Ctrl+D
			if len(buf) == 0 {
				fmt.Fprint(os.Stdout, "\r\n")
				return "", io.EOF
			}

		case 127, 8: // Backspace
			if cursor > 0 {
				buf = append(buf[:cursor-1], buf[cursor:]...)
				cursor--
			}

		case 1: // Ctrl+A
			cursor = 0
		case 5: // Ctrl+E
			cursor = len(buf)
		case 21: // Ctrl+U
			buf = append([]rune{}, buf[cursor:]...)
			cursor = 0
		case 11: // Ctrl+K
			buf = buf[:cursor]
		case 23: // Ctrl+W → delete word back
			buf, cursor = killWordBack(buf, cursor)

		// ── escape sequences ──────────────────────────────────────────────
		case 27:
			seq := lr.readEscSeq()
			seqHandled := false

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
				seqHandled = true
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
				seqHandled = true
			case "[C": // right
				if cursor < len(buf) {
					cursor++
				}
				seqHandled = true
			case "[D": // left
				if cursor > 0 {
					cursor--
				}
				seqHandled = true

			// Home / End (CSI, vt100, SS3)
			case "[H", "[1~", "OH":
				cursor = 0
				seqHandled = true
			case "[F", "[4~", "OF":
				cursor = len(buf)
				seqHandled = true

			// Delete
			case "[3~":
				if cursor < len(buf) {
					buf = append(buf[:cursor], buf[cursor+1:]...)
				}
				seqHandled = true

			// ── word-left ───────────────────────────────────────────────
			// iTerm2 default:       ESC+b
			// Terminal.app:         ESC+b
			// xterm modifyOtherKeys: [1;5D (Ctrl) / [1;3D (Alt)
			// Windows Terminal:     [1;5D / [1;3D
			// kitty:                [27;5;113~ (Ctrl) / [27;3;113~ (Alt)
			case "b":
				cursor = wordLeftPos(buf, cursor)
				seqHandled = true
			case "[1;3D", "[1;5D", "[1;4D", "[1;6D":
				cursor = wordLeftPos(buf, cursor)
				seqHandled = true
			case "[5D", "[5C":
				cursor = wordLeftPos(buf, cursor)
				seqHandled = true
			case "[27;5;113~", "[27;3;113~":
				cursor = wordLeftPos(buf, cursor)
				seqHandled = true

			// ── word-right ──────────────────────────────────────────────
			case "f":
				cursor = wordRightPos(buf, cursor)
				seqHandled = true
			case "[1;3C", "[1;5C", "[1;4C", "[1;6C":
				cursor = wordRightPos(buf, cursor)
				seqHandled = true
			case "[27;5;115~", "[27;3;115~":
				cursor = wordRightPos(buf, cursor)
				seqHandled = true

			// ── delete-word-back (Ctrl+Backspace / Alt+Backspace / Opt+⌫)
			// xterm:          ESC+DEL (\x1b\x7f) or ESC+BS (\x1b\x08)
			// iTerm2 default: ESC+DEL (\x1b\x7f)
			// Terminal.app:   ESC+DEL
			// kitty:          [27;5;127~ or [8;127u or [27;3;127~
			case "\x7f", "\x08":
				buf, cursor = killWordBack(buf, cursor)
				seqHandled = true
			case "[8;127u", "[127;8u":
				buf, cursor = killWordBack(buf, cursor)
				seqHandled = true
			case "[27;8;127~", "[27;5;127~", "[27;3;127~":
				buf, cursor = killWordBack(buf, cursor)
				seqHandled = true

			// ── delete-word-forward (Alt+d / Meta+d / Opt+⌦) ──────────
			case "d":
				buf, cursor = killWordForward(buf, cursor)
				seqHandled = true

			// ── newline via Shift+Enter / Alt+Enter ────────────────────
			case "[27;2;13~", "[13;2u":
				buf = append(buf[:cursor], append([]rune{'\n'}, buf[cursor:]...)...)
				cursor++
				seqHandled = true
			case "\r": // Alt+Enter = ESC+Enter
				buf = append(buf[:cursor], append([]rune{'\n'}, buf[cursor:]...)...)
				cursor++
				seqHandled = true
			}

			if seqHandled {
				lr.render(prompt, buf, cursor)
			}

		// ── printable characters ──────────────────────────────────────────
		default:
			if r >= 32 {
				buf = append(buf[:cursor], append([]rune{r}, buf[cursor:]...)...)
				cursor++
			}
			lr.render(prompt, buf, cursor)
		}
	}
}
