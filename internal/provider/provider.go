// Package provider runs a prompt through an installed LLM CLI in
// non-interactive mode.
package provider

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Request is a single-turn completion request.
type Request struct {
	System string
	Prompt string
	Model  string
}

// Provider is an LLM CLI that genie can shell out to.
type Provider interface {
	Name() string
	DefaultModel() string
	// Check reports whether the CLI is installed.
	Check() error
	// Complete returns the model's trimmed, non-empty reply.
	Complete(ctx context.Context, req Request) (string, error)
}

// Names lists the supported providers; the first is the default.
var Names = []string{"claude", "codex"}

// Get returns the provider with the given name; empty means the default.
func Get(name string) (Provider, error) {
	switch strings.ToLower(name) {
	case "", "claude":
		return Claude{}, nil
	case "codex":
		return Codex{}, nil
	default:
		return nil, fmt.Errorf("unknown provider %q (want %s)", name, strings.Join(Names, " or "))
	}
}

func lookPath(bin, installURL string) error {
	if _, err := exec.LookPath(bin); err != nil {
		return fmt.Errorf("%s CLI not found in PATH; install it from %s", bin, installURL)
	}
	return nil
}

// command builds a subprocess that is interrupted (not killed) when ctx is
// cancelled, so the CLI can clean up.
func command(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Cancel = func() error { return cmd.Process.Signal(os.Interrupt) }
	cmd.WaitDelay = 2 * time.Second
	return cmd
}
