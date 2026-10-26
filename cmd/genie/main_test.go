package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/saadjs/genie-cli/internal/config"
	"github.com/saadjs/genie-cli/internal/history"
	"github.com/saadjs/genie-cli/internal/provider"
)

type fakeProvider struct {
	reply string
	err   error
	got   *provider.Request
}

func (fakeProvider) Name() string         { return "claude" }
func (fakeProvider) DefaultModel() string { return "fake-model" }
func (fakeProvider) Check() error         { return nil }
func (f fakeProvider) Complete(_ context.Context, req provider.Request) (string, error) {
	if f.got != nil {
		*f.got = req
	}
	return f.reply, f.err
}

type result struct {
	code           int
	stdout, stderr string
	copied         []string
	req            provider.Request
}

// genie runs the CLI against a fake provider with piped (non-terminal)
// streams and an isolated HOME.
func genie(t *testing.T, reply string, stdin string, args ...string) result {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SHELL", "/bin/sh")

	var r result
	origProvider, origCopy := getProvider, copyToClipboard
	t.Cleanup(func() { getProvider, copyToClipboard = origProvider, origCopy })
	getProvider = func(name string) (provider.Provider, error) {
		if _, err := provider.Get(name); err != nil {
			return nil, err
		}
		return fakeProvider{reply: reply, got: &r.req}, nil
	}
	copyToClipboard = func(s string) error { r.copied = append(r.copied, s); return nil }

	var stdout, stderr bytes.Buffer
	r.code = run(args, strings.NewReader(stdin), &stdout, &stderr)
	r.stdout, r.stderr = stdout.String(), stderr.String()
	return r
}

func TestPipedStdoutIsBareCommand(t *testing.T) {
	r := genie(t, "find . -name '*.go'", "", "list", "go", "files")
	if r.code != 0 {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	if r.stdout != "find . -name '*.go'\n" {
		t.Fatalf("stdout = %q, want only the command", r.stdout)
	}
	if r.stderr != "" {
		t.Fatalf("stderr = %q, want nothing for a safe command", r.stderr)
	}
	if len(r.copied) != 0 {
		t.Fatal("must not touch the clipboard when stdout isn't a terminal")
	}
	if !strings.Contains(r.req.Prompt, "list go files") || r.req.Model != "fake-model" {
		t.Fatalf("provider got %+v", r.req)
	}

	entries, _ := history.Load(1)
	if len(entries) != 1 || entries[0].Command != "find . -name '*.go'" {
		t.Fatalf("history = %+v", entries)
	}
}

func TestRequestFromStdin(t *testing.T) {
	r := genie(t, "ls ~/Desktop", "what's on my desktop?\n")
	if r.code != 0 || r.stdout != "ls ~/Desktop\n" {
		t.Fatalf("exit %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
	if !strings.Contains(r.req.Prompt, "what's on my desktop?") {
		t.Fatalf("prompt = %q", r.req.Prompt)
	}
}

func TestEmptyStdinIsUsageError(t *testing.T) {
	r := genie(t, "ls", "")
	if r.code != exitUsage || !strings.Contains(r.stderr, "no request") {
		t.Fatalf("exit %d stderr %q", r.code, r.stderr)
	}
}

func TestFencedReplyIsCleaned(t *testing.T) {
	r := genie(t, "```bash\ndu -sh .\n```", "", "disk", "usage")
	if r.stdout != "du -sh .\n" {
		t.Fatalf("stdout = %q", r.stdout)
	}
}

func TestRefusalIsAnError(t *testing.T) {
	r := genie(t, "ERROR: that's a recipe, not a command", "", "bake", "a", "cake")
	if r.code != exitError || r.stdout != "" {
		t.Fatalf("exit %d stdout %q", r.code, r.stdout)
	}
	if !strings.Contains(r.stderr, "recipe") {
		t.Fatalf("stderr should carry the reason: %q", r.stderr)
	}
}

func TestDangerousCommandWarnsOnStderr(t *testing.T) {
	r := genie(t, "sudo rm -rf /", "", "nuke", "it")
	if r.stdout != "sudo rm -rf /\n" {
		t.Fatalf("stdout = %q", r.stdout)
	}
	if !strings.Contains(r.stderr, "Destructive") || !strings.Contains(r.stderr, "rm on root") {
		t.Fatalf("stderr = %q", r.stderr)
	}
}

func TestNoVerifyRunsAndPropagatesExitCode(t *testing.T) {
	r := genie(t, "echo ran; exit 3", "", "-y", "do", "it")
	if r.code != 3 {
		t.Fatalf("exit %d, want the command's status 3; stderr %q", r.code, r.stderr)
	}
	if r.stdout != "ran\n" {
		t.Fatalf("stdout = %q, want only the command's output", r.stdout)
	}
	if !strings.Contains(r.stderr, "echo ran; exit 3") {
		t.Fatalf("command should be shown on stderr: %q", r.stderr)
	}
}

func TestRunRefusesToPromptWithoutTerminal(t *testing.T) {
	r := genie(t, "echo ran", "y\n", "-r", "do", "it")
	if r.code != exitError || strings.Contains(r.stdout, "ran") {
		t.Fatalf("exit %d stdout %q", r.code, r.stdout)
	}
	if !strings.Contains(r.stderr, "-y") {
		t.Fatalf("stderr should point to -y: %q", r.stderr)
	}
}

func TestExplain(t *testing.T) {
	r := genie(t, "Lists all files, including hidden ones.", "", "-e", "ls", "-la")
	if r.code != 0 || r.stdout != "Lists all files, including hidden ones.\n" {
		t.Fatalf("exit %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
	if !strings.Contains(r.req.Prompt, "ls -la") || !strings.Contains(r.req.System, "explain") {
		t.Fatalf("explain request = %+v", r.req)
	}
	if entries, _ := history.Load(1); len(entries) != 0 {
		t.Fatal("explanations should not be saved to history")
	}
}

func TestProviderErrorExits1(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	orig := getProvider
	t.Cleanup(func() { getProvider = orig })
	getProvider = func(string) (provider.Provider, error) {
		return fakeProvider{err: errors.New("claude: rate limited")}, nil
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"list", "files"}, strings.NewReader(""), &stdout, &stderr)
	if code != exitError || !strings.Contains(stderr.String(), "rate limited") || stdout.Len() != 0 {
		t.Fatalf("exit %d stdout %q stderr %q", code, stdout.String(), stderr.String())
	}
}

func TestUsageErrors(t *testing.T) {
	if r := genie(t, "ls", "", "--bogus"); r.code != exitUsage {
		t.Fatalf("unknown flag: exit %d", r.code)
	}
	if r := genie(t, "ls", "", "--provider", "gemini", "x"); r.code != exitUsage {
		t.Fatalf("unknown provider: exit %d", r.code)
	}
	r := genie(t, "ls", "", "--help")
	if r.code != 0 || !strings.Contains(r.stdout, "Usage:") {
		t.Fatalf("--help: exit %d stdout %q", r.code, r.stdout)
	}
}

func TestMalformedConfigWarnsAndContinues(t *testing.T) {
	home := t.TempDir()
	orig := getProvider
	t.Cleanup(func() { getProvider = orig })
	getProvider = func(string) (provider.Provider, error) { return fakeProvider{reply: "ls"}, nil }
	t.Setenv("HOME", home)
	os.WriteFile(filepath.Join(home, ".genie.yaml"), []byte("provider: [oops\n"), 0o600)

	var stdout, stderr bytes.Buffer
	code := run([]string{"list"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || stdout.String() != "ls\n" || !strings.Contains(stderr.String(), "ignoring config") {
		t.Fatalf("exit %d stdout %q stderr %q", code, stdout.String(), stderr.String())
	}
}

func TestProviderModelDefaults(t *testing.T) {
	p, model, err := providerModel(config.Config{}, "", "")
	if err != nil || p.Name() != "claude" || model != "claude-haiku-5-5" {
		t.Fatalf("got %v %q %v", p, model, err)
	}
	p, model, err = providerModel(config.Config{}, "codex", "")
	if err != nil || p.Name() != "codex" || model != "gpt-6-luna" {
		t.Fatalf("got %v %q %v", p, model, err)
	}
}

func TestProviderModelUsesConfig(t *testing.T) {
	p, model, _ := providerModel(config.Config{Provider: "codex", Model: "gpt-5.4"}, "", "")
	if p.Name() != "codex" || model != "gpt-5.4" {
		t.Fatalf("got %s %q", p.Name(), model)
	}
}

func TestProviderModelFlagsOverrideConfig(t *testing.T) {
	p, model, _ := providerModel(config.Config{Provider: "claude", Model: "haiku"}, "codex", "gpt-5.4-mini")
	if p.Name() != "codex" || model != "gpt-5.4-mini" {
		t.Fatalf("got %s %q", p.Name(), model)
	}
}

func TestProviderFlagDoesNotInheritOtherProvidersModel(t *testing.T) {
	p, model, _ := providerModel(config.Config{Model: "haiku"}, "codex", "")
	if p.Name() != "codex" || model != "gpt-6-luna" {
		t.Fatalf("codex got model %q; the config model belongs to claude", model)
	}
	_, model, _ = providerModel(config.Config{Model: "sonnet"}, "claude", "")
	if model != "sonnet" {
		t.Fatalf("claude should use config model, got %q", model)
	}
}

func TestPipedHistoryIsJSONLines(t *testing.T) {
	genie(t, "ls 2>/dev/null", "", "list")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--history"}, strings.NewReader(""), &stdout, &stderr); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	var e history.Entry
	if err := json.Unmarshal(stdout.Bytes(), &e); err != nil || e.Command != "ls 2>/dev/null" {
		t.Fatalf("history output %q: %v", stdout.String(), err)
	}
	if strings.Contains(stdout.String(), "\\u003e") {
		t.Fatalf("history should not HTML-escape commands: %q", stdout.String())
	}
}

func TestAutoRunConfigPrintsCommandWhenPiped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	os.WriteFile(filepath.Join(home, ".genie.yaml"), []byte("auto_run: true\n"), 0o600)
	orig := getProvider
	t.Cleanup(func() { getProvider = orig })
	getProvider = func(string) (provider.Provider, error) { return fakeProvider{reply: "ls"}, nil }

	var stdout, stderr bytes.Buffer
	code := run([]string{"list"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 || stdout.String() != "ls\n" {
		t.Fatalf("exit %d stdout %q stderr %q", code, stdout.String(), stderr.String())
	}
}

func TestNoVerifyPassesRemainingStdinToCommand(t *testing.T) {
	r := genie(t, "wc -l | tr -d ' '", "count lines\na\nb\n", "-y")
	if r.code != 0 || r.stdout != "2\n" {
		t.Fatalf("exit %d stdout %q stderr %q", r.code, r.stdout, r.stderr)
	}
}
