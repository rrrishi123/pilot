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

// errAborted is Ctrl-C on the current line (abort the line, not the session).
var errAborted = errors.New("line aborted")

// lineReader is a raw-mode line editor with BRACKETED PASTE. The point: a pasted blob
// (with embedded newlines) is inserted as ONE input and is NEVER auto-submitted — only
// a manual Enter submits. So pasting a jsonl path list or JSON no longer fires the turn
// on the first newline. Display stays single-line (a pasted '\n' shows as a dim ↵), but
// the returned string keeps the real '\n'. Supports typing, Backspace/Delete, Ctrl-C
// (abort), Ctrl-D (EOF on empty), Up/Down history, Left/Right, Home/End, Ctrl-A/E/U/K/W.
// History persists to histFile. Falls back to a plain line read when stdin isn't a TTY.
type lineReader struct {
	in       *bufio.Reader
	fd       int
	histFile string
	hist     []string
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
	for _, h := range lr.hist { // one entry per file line (newlines flattened to spaces)
		fmt.Fprintln(f, strings.ReplaceAll(h, "\n", " "))
	}
}

// render redraws the single visible line: prompt + buffer (with '\n' shown as ↵), then
// parks the cursor at its column.
func (lr *lineReader) render(prompt string, buf []rune, cursor int) {
	disp := make([]rune, 0, len(buf))
	for _, r := range buf {
		if r == '\n' {
			disp = append(disp, '↵')
		} else {
			disp = append(disp, r)
		}
	}
	fmt.Fprintf(os.Stdout, "\r\033[K%s%s", prompt, string(disp))
	if col := len([]rune(prompt)) + cursor; col > 0 {
		fmt.Fprintf(os.Stdout, "\r\033[%dC", col)
	} else {
		fmt.Fprint(os.Stdout, "\r")
	}
}

// readEscSeq reads the rest of a CSI/escape sequence after ESC, up to and including its
// final byte (a letter or '~'). Returns e.g. "[A", "[3~", "[200~".
func (lr *lineReader) readEscSeq() string {
	var sb strings.Builder
	for {
		b, err := lr.in.ReadByte()
		if err != nil {
			break
		}
		sb.WriteByte(b)
		if b != '[' && b >= 0x40 && b <= 0x7e { // final byte
			break
		}
	}
	return sb.String()
}

// readPaste consumes a bracketed paste body (already past ESC[200~) up to ESC[201~,
// returning its text with CRLF normalized to '\n'. Newlines here are CONTENT, not Enter.
func (lr *lineReader) readPaste() string {
	var sb strings.Builder
	for {
		r, _, err := lr.in.ReadRune()
		if err != nil {
			break
		}
		if r == 27 { // possible paste-end ESC[201~
			if seq := lr.readEscSeq(); seq == "[201~" {
				break
			} else {
				sb.WriteRune(27)
				sb.WriteString(seq)
			}
			continue
		}
		if r == '\r' {
			continue // fold CRLF -> LF
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

func (lr *lineReader) readLine(prompt string, record bool) (string, error) {
	old, err := term.MakeRaw(lr.fd)
	if err != nil { // not a TTY (piped): plain line read
		s, e := lr.in.ReadString('\n')
		return strings.TrimRight(s, "\r\n"), e
	}
	defer term.Restore(lr.fd, old)
	fmt.Fprint(os.Stdout, prompt+"\033[?2004h") // draw prompt + enable bracketed paste
	defer fmt.Fprint(os.Stdout, "\033[?2004l")

	var buf []rune
	cursor := 0
	histIdx := len(lr.hist)
	var saved []rune
	insert := func(rs []rune) {
		nb := make([]rune, 0, len(buf)+len(rs))
		nb = append(nb, buf[:cursor]...)
		nb = append(nb, rs...)
		nb = append(nb, buf[cursor:]...)
		buf = nb
		cursor += len(rs)
	}
	for {
		r, _, err := lr.in.ReadRune()
		if err != nil {
			return "", err
		}
		switch r {
		case '\r', '\n': // SUBMIT
			fmt.Fprint(os.Stdout, "\r\n")
			line := string(buf)
			if record && strings.TrimSpace(line) != "" {
				lr.hist = append(lr.hist, line)
			}
			return line, nil
		case 3: // Ctrl-C
			fmt.Fprint(os.Stdout, "\r\n")
			return "", errAborted
		case 4: // Ctrl-D
			if len(buf) == 0 {
				fmt.Fprint(os.Stdout, "\r\n")
				return "", io.EOF
			}
		case 127, 8: // Backspace
			if cursor > 0 {
				buf = append(buf[:cursor-1], buf[cursor:]...)
				cursor--
			}
		case 1: // Ctrl-A
			cursor = 0
		case 5: // Ctrl-E
			cursor = len(buf)
		case 21: // Ctrl-U: kill to line start
			buf = append([]rune{}, buf[cursor:]...)
			cursor = 0
		case 11: // Ctrl-K: kill to line end
			buf = buf[:cursor]
		case 23: // Ctrl-W: delete word back
			i := cursor
			for i > 0 && buf[i-1] == ' ' {
				i--
			}
			for i > 0 && buf[i-1] != ' ' {
				i--
			}
			buf = append(buf[:i], buf[cursor:]...)
			cursor = i
		case 27: // escape sequence
			switch lr.readEscSeq() {
			case "[A": // Up: previous history
				if histIdx == len(lr.hist) {
					saved = append([]rune{}, buf...)
				}
				if histIdx > 0 {
					histIdx--
					buf = []rune(lr.hist[histIdx])
					cursor = len(buf)
				}
			case "[B": // Down: next history
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
			case "[H", "[1~": // Home
				cursor = 0
			case "[F", "[4~": // End
				cursor = len(buf)
			case "[3~": // Delete
				if cursor < len(buf) {
					buf = append(buf[:cursor], buf[cursor+1:]...)
				}
			case "[200~": // BRACKETED PASTE — insert as content, don't submit
				insert([]rune(lr.readPaste()))
			}
		default:
			if r >= 32 {
				insert([]rune{r})
			}
		}
		lr.render(prompt, buf, cursor)
	}
}
