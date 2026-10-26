package provider

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// stubCLI installs an executable shell script named bin at the front of PATH
// and returns the directory it lives in.
func stubCLI(t *testing.T, bin, script string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, bin), []byte("#!/bin/sh\n"+script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return dir
}

func readArgs(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
}

func TestClaudeCompletePassesFastFlags(t *testing.T) {
	dir := stubCLI(t, "claude", `for a in "$@"; do printf '%s\n' "$a"; done > "$(dirname "$0")/args"
printf '{"type":"result","is_error":false,"result":"  ls -la\\n"}'
`)
	got, err := Claude{}.Complete(context.Background(), Request{System: "SYS", Prompt: "Request: list", Model: "m1"})
	if err != nil {
		t.Fatalf("Complete error: %v", err)
	}
	if got != "ls -la" {
		t.Fatalf("Complete = %q, want %q", got, "ls -la")
	}

	args := readArgs(t, filepath.Join(dir, "args"))
	if args[len(args)-1] != "Request: list" {
		t.Fatalf("prompt should be the last arg, got %q", args)
	}
	i := slices.Index(args, "--tools")
	if i < 0 || args[i+1] != "" {
		t.Fatalf("tools should be disabled, got %q", args)
	}
	i = slices.Index(args, "--system-prompt")
	if i < 0 || args[i+1] != "SYS" {
		t.Fatalf("system prompt should replace the default, got %q", args)
	}
	for _, flag := range []string{"--strict-mcp-config", "--no-session-persistence", "--disable-slash-commands"} {
		if !slices.Contains(args, flag) {
			t.Fatalf("missing %s in %q", flag, args)
		}
	}
	if i = slices.Index(args, "--model"); i < 0 || args[i+1] != "m1" {
		t.Fatalf("model not passed, got %q", args)
	}
}

func TestClaudeCompleteReportsAPIError(t *testing.T) {
	stubCLI(t, "claude", `printf '{"type":"result","is_error":true,"result":"Invalid model name"}'
exit 1
`)
	_, err := Claude{}.Complete(context.Background(), Request{Prompt: "x"})
	if err == nil || !strings.Contains(err.Error(), "Invalid model name") {
		t.Fatalf("want API error message, got %v", err)
	}
}

func TestClaudeCompleteReportsPlainStderr(t *testing.T) {
	stubCLI(t, "claude", `echo "error: unknown option '--bogus'" >&2
exit 1
`)
	_, err := Claude{}.Complete(context.Background(), Request{Prompt: "x"})
	if err == nil || !strings.Contains(err.Error(), "unknown option") {
		t.Fatalf("want stderr surfaced, got %v", err)
	}
}

func TestClaudeCompleteEmpty(t *testing.T) {
	stubCLI(t, "claude", `printf '{"is_error":false,"result":"   "}'`)
	if _, err := (Claude{}).Complete(context.Background(), Request{Prompt: "x"}); err == nil {
		t.Fatal("empty result should be an error")
	}
}

func TestCompleteStopsOnCancel(t *testing.T) {
	stubCLI(t, "claude", "exec sleep 30\n")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := Claude{}.Complete(ctx, Request{Prompt: "x"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want context error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("cancel took %s; subprocess was not interrupted", elapsed)
	}
}

func TestGet(t *testing.T) {
	for _, name := range []string{"", "claude", "Claude", "codex"} {
		if _, err := Get(name); err != nil {
			t.Fatalf("Get(%q): %v", name, err)
		}
	}
	if _, err := Get("gemini"); err == nil {
		t.Fatal("Get should reject unknown providers")
	}
}
