package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestWrap(t *testing.T) {
	text := "Lists files.\n- -l uses the long listing format, showing permissions, owner, size, and modification time\nshort"
	got := Wrap(text, 30)
	want := []string{
		"Lists files.",
		"- -l uses the long listing",
		"  format, showing permissions,",
		"  owner, size, and",
		"  modification time",
		"short",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("Wrap =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for _, line := range got {
		if len(line) > 30 {
			t.Fatalf("line exceeds width: %q", line)
		}
	}
}

func TestWrapNumberedList(t *testing.T) {
	got := Wrap("1. grep searches the input for lines matching the pattern", 24)
	if len(got) < 2 || !strings.HasPrefix(got[1], "   ") || strings.HasPrefix(got[1], "    ") {
		t.Fatalf("continuation should hang under the text: %q", got)
	}
}

func TestPlainStreamsGetNoDecoration(t *testing.T) {
	var out, errOut bytes.Buffer
	term := New(&out, &errOut)

	term.Command(term.Out, "ls -la")
	term.Warn(true, "rm on root")
	term.Success("Copied")
	s := term.Spin("Thinking")
	s.Stop()

	if out.String() != "ls -la\n" {
		t.Fatalf("stdout = %q, want bare command", out.String())
	}
	if strings.Contains(errOut.String(), "\x1b") {
		t.Fatalf("stderr has escape codes: %q", errOut.String())
	}
	if errOut.String() != "genie: warning: rm on root\n" {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestSpinnerSkipsEscapesOnDumbTerminal(t *testing.T) {
	t.Setenv("TERM", "dumb")
	var errOut bytes.Buffer
	term := New(&bytes.Buffer{}, &errOut)
	term.ErrTTY = true // a real but dumb terminal

	term.Spin("Thinking").Stop()
	if strings.Contains(errOut.String(), "\x1b") {
		t.Fatalf("dumb terminal got escape codes: %q", errOut.String())
	}
}
