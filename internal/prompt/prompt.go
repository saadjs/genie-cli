// Package prompt builds the instructions sent to providers and parses what
// comes back.
package prompt

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Env describes the machine the command will run on, so the model picks
// tools and flags that exist there (BSD vs GNU sed, etc.).
type Env struct {
	OS    string
	Shell string
}

// CurrentEnv reports the local OS and the user's login shell.
func CurrentEnv() Env {
	return Env{OS: osName(runtime.GOOS), Shell: ShellName(os.Getenv("SHELL"))}
}

// ShellName returns the base name of a shell path, defaulting to sh.
func ShellName(path string) string {
	if path == "" {
		return "sh"
	}
	return filepath.Base(path)
}

func osName(goos string) string {
	switch goos {
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	case "freebsd":
		return "FreeBSD"
	default:
		return goos
	}
}

// Command returns the system prompt and user message for translating a
// plain-English request into a shell command.
func Command(env Env, request string) (system, user string) {
	system = fmt.Sprintf(`You translate plain-English requests into one shell command line for %s, run by %s.

Rules:
- Reply with only the command. No explanation, markdown, backticks, or leading $.
- Put multiple steps on one line, joined with && or pipes.
- Give the simplest common command that does what was asked. No clever one-liners, and no extra flags, sorting, or error suppression nobody asked for.
- Use standard tools that ship with %[1]s, with flags valid there.
- If the request is ambiguous, pick the most common interpretation; don't ask.
- Only if the request has nothing to do with the shell, reply: ERROR: <brief reason>`, env.OS, env.Shell)
	return system, "Request: " + request
}

// Explain returns the system prompt and user message for explaining a
// shell command.
func Explain(env Env, command string) (system, user string) {
	system = fmt.Sprintf(`You explain shell commands in plain English for someone using %s on %s.

Rules:
- First line: what the whole command does, in one sentence.
- Then one short line per part, flag, or pipeline stage, formatted as "- part: meaning".
- If it is destructive or risky, end with one line starting "Risk:".
- Plain text only, no markdown, no other sections. Aim for under 12 lines.`, env.Shell, env.OS)
	return system, "Command: " + command
}

// RefusedError is returned when the model says the request isn't a shell task.
type RefusedError struct{ Reason string }

func (e *RefusedError) Error() string { return e.Reason }

// ParseCommand extracts the command from a model reply, tolerating the
// markdown fences and "$ " prefixes models add despite instructions.
func ParseCommand(reply string) (string, error) {
	s := strings.TrimSpace(reply)
	if rest, ok := strings.CutPrefix(s, "ERROR:"); ok {
		return "", &RefusedError{Reason: strings.TrimSpace(rest)}
	}

	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		lines = lines[1:] // opening fence, possibly with a language tag
		if n := len(lines); n > 0 && strings.HasPrefix(strings.TrimSpace(lines[n-1]), "```") {
			lines = lines[:n-1]
		}
		s = strings.TrimSpace(strings.Join(lines, "\n"))
	} else if len(s) > 1 && s[0] == '`' && s[len(s)-1] == '`' && !strings.Contains(s[1:len(s)-1], "`") {
		s = strings.TrimSpace(s[1 : len(s)-1])
	}

	s = strings.TrimSpace(strings.TrimPrefix(s, "$ "))
	// Check after cleanup so a fenced refusal is never treated as a command.
	if rest, ok := strings.CutPrefix(s, "ERROR:"); ok {
		return "", &RefusedError{Reason: strings.TrimSpace(rest)}
	}
	if s == "" {
		return "", errors.New("empty response; try rephrasing")
	}
	return s, nil
}
