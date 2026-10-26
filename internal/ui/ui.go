// Package ui renders genie's terminal output.
//
// Contract: stdout carries only results (the command, an explanation,
// history). Prompts, spinners, warnings, and errors go to stderr. When a
// stream isn't a terminal it gets plain text with no color or decoration, so
// `genie ... | sh` and agents reading stdout get exactly the command.
package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Term is a pair of output streams plus what we know about them.
type Term struct {
	Out, Err       io.Writer
	OutTTY, ErrTTY bool
	Width          int // columns of Out, 0 if unknown
	color          bool
	dumb           bool // TERM=dumb: no escape codes at all
}

// New wraps the given streams, detecting terminals and NO_COLOR.
func New(out, err io.Writer) *Term {
	t := &Term{Out: out, Err: err, OutTTY: IsTerminal(out), ErrTTY: IsTerminal(err)}
	t.dumb = os.Getenv("TERM") == "dumb"
	t.color = os.Getenv("NO_COLOR") == "" && !t.dumb
	if f, ok := out.(*os.File); ok && t.OutTTY {
		if w, _, e := term.GetSize(int(f.Fd())); e == nil {
			t.Width = w
		}
	}
	return t
}

// IsTerminal reports whether stream (a reader or writer) is an interactive
// terminal.
func IsTerminal(stream any) bool {
	f, ok := stream.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

const (
	bold    = "1"
	dim     = "2"
	red     = "31"
	green   = "32"
	yellow  = "33"
	magenta = "35"
	cyan    = "36"
)

func (t *Term) paint(tty bool, s string, codes ...string) string {
	if !tty || !t.color || len(codes) == 0 {
		return s
	}
	return "\x1b[" + strings.Join(codes, ";") + "m" + s + "\x1b[0m"
}

// Ask writes a prompt to stderr.
func (t *Term) Ask(question string) {
	fmt.Fprint(t.Err, t.paint(t.ErrTTY, question, bold, magenta)+" ")
}

// Command shows a shell command. On a terminal it is set off with a dim
// prompt; otherwise it is printed bare.
func (t *Term) Command(w io.Writer, command string) {
	tty := (w == t.Out && t.OutTTY) || (w == t.Err && t.ErrTTY)
	if !tty {
		fmt.Fprintln(w, command)
		return
	}
	fmt.Fprintf(w, "\n  %s%s\n\n", t.paint(true, "$ ", dim), t.paint(true, command, bold))
}

// Success reports a completed side effect, e.g. a clipboard copy.
func (t *Term) Success(msg string) {
	if t.ErrTTY {
		fmt.Fprintf(t.Err, "  %s %s\n", t.paint(true, "✓", green), t.paint(true, msg, dim))
	}
}

// Warn flags a risky command; danger is shown in red, caution in yellow.
func (t *Term) Warn(danger bool, msg string) {
	if !t.ErrTTY {
		fmt.Fprintf(t.Err, "genie: warning: %s\n", msg)
		return
	}
	c := yellow
	if danger {
		c = red
	}
	fmt.Fprintf(t.Err, "  %s %s\n", t.paint(true, "⚠", c, bold), t.paint(true, msg, c))
}

// Error reports a failure.
func (t *Term) Error(err error) {
	if !t.ErrTTY {
		fmt.Fprintf(t.Err, "genie: %s\n", err)
		return
	}
	fmt.Fprintf(t.Err, "  %s %s\n", t.paint(true, "✗", red, bold), t.paint(true, err.Error(), red))
}

// Note prints secondary text to stderr.
func (t *Term) Note(msg string) {
	if t.ErrTTY {
		fmt.Fprintf(t.Err, "  %s\n", t.paint(true, msg, dim))
	} else {
		fmt.Fprintln(t.Err, msg)
	}
}

// Explanation prints an explanation of command, wrapped to the terminal.
func (t *Term) Explanation(command, text string) {
	if !t.OutTTY {
		fmt.Fprintln(t.Out, text)
		return
	}
	t.Command(t.Out, command)
	width := min(t.Width, 100)
	if width <= 0 {
		width = 80
	}
	for _, line := range Wrap(text, width-4) {
		fmt.Fprintf(t.Out, "  %s\n", line)
	}
	fmt.Fprintln(t.Out)
}

// HistoryEntry prints one history item on a terminal.
func (t *Term) HistoryEntry(when, request, command string) {
	fmt.Fprintf(t.Out, "  %s  %s\n", t.paint(t.OutTTY, when, dim), t.paint(t.OutTTY, request, magenta))
	fmt.Fprintf(t.Out, "  %s%s\n\n", t.paint(t.OutTTY, "$ ", dim), t.paint(t.OutTTY, command, bold))
}

// Wrap word-wraps text to width columns, keeping existing line breaks and
// hanging continuation lines under list bullets.
func Wrap(text string, width int) []string {
	width = max(width, 20)
	var out []string
	for line := range strings.Lines(text) {
		line = strings.TrimRight(line, " \t\r\n")
		if len([]rune(line)) <= width {
			out = append(out, line)
			continue
		}
		hang := strings.Repeat(" ", hangingIndent(line))
		cur := ""
		for _, word := range strings.Fields(line) {
			switch {
			case cur == "":
				cur = line[:len(line)-len(strings.TrimLeft(line, " \t"))] + word
			case len([]rune(cur))+1+len([]rune(word)) > width:
				out = append(out, cur)
				cur = hang + word
			default:
				cur += " " + word
			}
		}
		out = append(out, cur)
	}
	return out
}

// hangingIndent returns the column where text starts after a leading
// bullet ("- ", "* ", "1. "), or after leading spaces.
func hangingIndent(line string) int {
	trimmed := strings.TrimLeft(line, " \t")
	indent := len(line) - len(trimmed)
	for _, b := range []string{"- ", "* ", "• "} {
		if strings.HasPrefix(trimmed, b) {
			return indent + len([]rune(b))
		}
	}
	i := 0
	for i < len(trimmed) && trimmed[i] >= '0' && trimmed[i] <= '9' {
		i++
	}
	if i > 0 && i+1 < len(trimmed) && (trimmed[i] == '.' || trimmed[i] == ')') && trimmed[i+1] == ' ' {
		return indent + i + 2
	}
	return indent
}
