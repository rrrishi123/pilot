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

// lineReader is a plain raw-mode line editor — no selection, no bracketed paste,
// no shift+arrow gymnastics. Supports typing, Backspace/Delete, Enter to submit (\ at line end → continue on next line)
// (with \ continuation), Ctrl+J / Shift+Enter for newline, Ctrl+A/E/U/K/W,
// arrows, Home/End, Up/Down history, Ctrl+C abort, Ctrl+D EOF.
// Multi-line input renders across real terminal rows.
type lineReader struct {
	in       *bufio.Reader
	fd       int
	histFile string
	hist     []string
	curRow   int    // which render row the cursor is on
	preload  string // if set, start with this content (for interrupt recovery)
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

// render clears the previous render and redraws prompt + buffer, then parks the
// cursor at the correct row/col. No selection rendering — simple.
func (lr *lineReader) render(prompt string, buf []rune, cursor int) {
	if lr.curRow > 0 {
		fmt.Fprintf(os.Stdout, "\033[%dA", lr.curRow)
	}
	fmt.Fprint(os.Stdout, "\r\033[J") // col 0, clear below
	fmt.Fprint(os.Stdout, prompt)

	for _, ch := range buf {
		if ch == '\n' {
			fmt.Fprint(os.Stdout, "\r\n")
			continue
		}
		fmt.Fprint(os.Stdout, string(ch))
	}

	// Cursor row/col within the buffer
	row, col := 0, 0
	for i := 0; i < cursor && i < len(buf); i++ {
		if buf[i] == '\n' {
			row, col = row+1, 0
		} else {
			col++
		}
	}
	lastRow := strings.Count(string(buf), "\n")
	if up := lastRow - row; up > 0 {
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

// readEscSeq reads the bytes after ESC. Returns the complete sequence string.
// CSI: ESC [ … final (>= 0x40, except '[' itself and 'O')
// SS3:  ESC O final (one more byte)
func (lr *lineReader) readEscSeq() string {
	var sb strings.Builder
	for {
		b, err := lr.in.ReadByte()
		if err != nil {
			break
		}
		sb.WriteByte(b)
		// A final byte is >= 0x40, but '[' (0x5B) and 'O' (0x4F) are
		// prefixes (CSI/SS3 introducers), not finals.
		if b >= 0x40 && b <= 0x7e && b != '[' && b != 'O' {
			break
		}
		if b < 0x20 { // control chars (e.g. \r, \n for meta combos)
			break
		}
	}
	return sb.String()
}

func (lr *lineReader) readLine(prompt string, record bool) (string, error) {
	old, err := term.MakeRaw(lr.fd)
	if err != nil {
		s, e := lr.in.ReadString('\n')
		return strings.TrimRight(s, "\r\n"), e
	}
	defer term.Restore(lr.fd, old)

	// Use preload if set (interrupt recovery)
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
		case '\r': // Enter → submit, or line continuation if trailing \
			// Check for backslash continuation: if the last non-space
			// char is \, replace it (and any trailing spaces) with a
			// newline and keep reading.
			bufTrim := strings.TrimRight(string(buf), " ")
			if strings.HasSuffix(bufTrim, "\\") && len(bufTrim) > 0 {
				// Backslash depth: how many chars after the last \?
				trail := len(buf) - len([]rune(bufTrim)) + 1
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

		case '\n': // Ctrl+J → insert newline (Shift+Enter too, on terminals that send distinct escape seq)
			nb := make([]rune, 0, len(buf)+1)
			nb = append(nb, buf[:cursor]...)
			nb = append(nb, '\n')
			nb = append(nb, buf[cursor:]...)
			buf = nb
			cursor++

		case 3: // Ctrl+C → abort the line
			fmt.Fprint(os.Stdout, "\r\n")
			return "", errAborted

		case 4: // Ctrl+D
			if len(buf) == 0 {
				fmt.Fprint(os.Stdout, "\r\n")
				return "", io.EOF
			}
			// non-empty: ignored (like readline)

		case 127, 8: // Backspace
			if cursor > 0 {
				buf = append(buf[:cursor-1], buf[cursor:]...)
				cursor--
			}

		case 1: // Ctrl+A → start
			cursor = 0
		case 5: // Ctrl+E → end
			cursor = len(buf)

		case 21: // Ctrl+U → kill to start
			buf = append([]rune{}, buf[cursor:]...)
			cursor = 0
		case 11: // Ctrl+K → kill to end
			buf = buf[:cursor]
		case 23: // Ctrl+W → delete word back
			i := cursor
			for i > 0 && buf[i-1] == ' ' {
				i--
			}
			for i > 0 && buf[i-1] != ' ' {
				i--
			}
			buf = append(buf[:i], buf[cursor:]...)
			cursor = i

		case 27: // Escape → CSI / SS3 sequence
			switch lr.readEscSeq() {
			case "[A": // Up
				if histIdx == len(lr.hist) {
					saved = append([]rune{}, buf...)
				}
				if histIdx > 0 {
					histIdx--
					buf = []rune(lr.hist[histIdx])
					cursor = len(buf)
				}
			case "[B": // Down
				if histIdx < len(lr.hist) {
					histIdx++
					if histIdx == len(lr.hist) {
						buf = append([]rune{}, saved...)
					} else {
						buf = []rune(lr.hist[histIdx])
					}
					cursor = len(buf)
				}
			case "[C": // Right
				if cursor < len(buf) {
					cursor++
				}
			case "[D": // Left
				if cursor > 0 {
					cursor--
				}
			case "[H", "[1~", "OH": // Home (ANSI, vt100, SS3)
				cursor = 0
			case "[F", "[4~", "OF": // End (ANSI, vt100, SS3)
				cursor = len(buf)
			case "[3~": // Delete
				if cursor < len(buf) {
					buf = append(buf[:cursor], buf[cursor+1:]...)
				}
			case "[1;5D": // Ctrl+Left
				cursor = wordLeft(buf, cursor)
			case "[1;5C": // Ctrl+Right
				cursor = wordRight(buf, cursor)
			case "[27;2;13~", "[13;2u": // Shift+Enter (xterm modifyOtherKeys / kitty CSI u)
				nb := make([]rune, 0, len(buf)+1)
				nb = append(nb, buf[:cursor]...)
				nb = append(nb, '\n')
				nb = append(nb, buf[cursor:]...)
				buf = nb
				cursor++
			// unrecognised → silently skip
			}

		default:
			if r >= 32 {
				nb := make([]rune, 0, len(buf)+1)
				nb = append(nb, buf[:cursor]...)
				nb = append(nb, r)
				nb = append(nb, buf[cursor:]...)
				buf = nb
				cursor++
			}
		}
		lr.render(prompt, buf, cursor)
	}
}

// wordLeft returns the position after moving one word left from 'pos'.
func wordLeft(buf []rune, pos int) int {
	for pos > 0 && buf[pos-1] == ' ' {
		pos--
	}
	for pos > 0 && buf[pos-1] != ' ' {
		pos--
	}
	return pos
}

// wordRight returns the position after moving one word right from 'pos'.
func wordRight(buf []rune, pos int) int {
	for pos < len(buf) && buf[pos] != ' ' {
		pos++
	}
	for pos < len(buf) && buf[pos] == ' ' {
		pos++
	}
	return pos
}
