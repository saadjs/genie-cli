package main

import (
	"bufio"
	"strings"
	"testing"

	"github.com/fatih/color"
	"github.com/saadjs/genie-cli/internal/config"
	"github.com/saadjs/genie-cli/internal/safety"
)

func TestReadInputAcceptsTrailingEOFWithoutNewline(t *testing.T) {
	input, err := readInput(bufio.NewReader(strings.NewReader("show me all files")))
	if err != nil {
		t.Fatalf("readInput returned error: %v", err)
	}
	if input != "show me all files" {
		t.Fatalf("readInput = %q, want %q", input, "show me all files")
	}
}

func TestReadInputTrimsNewline(t *testing.T) {
	input, err := readInput(bufio.NewReader(strings.NewReader("show me all files\n")))
	if err != nil {
		t.Fatalf("readInput returned error: %v", err)
	}
	if input != "show me all files" {
		t.Fatalf("readInput = %q, want %q", input, "show me all files")
	}
}

func TestClipboardEnabledDefaultsToTrue(t *testing.T) {
	if !clipboardEnabled(config.Config{}) {
		t.Fatal("clipboard should default to enabled")
	}
}

func TestProviderModelDefaultsToClaudeHaiku(t *testing.T) {
	provider, model, err := providerModel(config.Config{}, "", "")
	if err != nil {
		t.Fatalf("providerModel returned error: %v", err)
	}
	if provider != "claude" {
		t.Fatalf("provider = %q, want %q", provider, "claude")
	}
	if model != "haiku" {
		t.Fatalf("model = %q, want %q", model, "haiku")
	}
}

func TestProviderModelDefaultsCodexModel(t *testing.T) {
	provider, model, err := providerModel(config.Config{}, "codex", "")
	if err != nil {
		t.Fatalf("providerModel returned error: %v", err)
	}
	if provider != "codex" {
		t.Fatalf("provider = %q, want %q", provider, "codex")
	}
	if model != "gpt-5.4-mini" {
		t.Fatalf("model = %q, want %q", model, "gpt-5.4-mini")
	}
}

func TestProviderModelUsesConfig(t *testing.T) {
	cfg := config.Config{Provider: "codex", Model: "gpt-5.4"}
	provider, model, err := providerModel(cfg, "", "")
	if err != nil {
		t.Fatalf("providerModel returned error: %v", err)
	}
	if provider != "codex" || model != "gpt-5.4" {
		t.Fatalf("got provider=%q model=%q", provider, model)
	}
}

func TestProviderModelFlagOverridesConfig(t *testing.T) {
	cfg := config.Config{Provider: "claude", Model: "haiku"}
	provider, model, err := providerModel(cfg, "codex", "gpt-5.4-mini")
	if err != nil {
		t.Fatalf("providerModel returned error: %v", err)
	}
	if provider != "codex" || model != "gpt-5.4-mini" {
		t.Fatalf("got provider=%q model=%q", provider, model)
	}
}

func TestProviderModelRejectsUnknownProvider(t *testing.T) {
	_, _, err := providerModel(config.Config{}, "unknown", "")
	if err == nil {
		t.Fatal("expected error for unknown provider")
	}
}

func TestClipboardEnabledHonorsFalse(t *testing.T) {
	enabled := false
	if clipboardEnabled(config.Config{Clipboard: &enabled}) {
		t.Fatal("clipboard should be disabled when config is false")
	}
}

func TestClipboardEnabledHonorsTrue(t *testing.T) {
	enabled := true
	if !clipboardEnabled(config.Config{Clipboard: &enabled}) {
		t.Fatal("clipboard should be enabled when config is true")
	}
}

func TestSafetyWarningDangerous(t *testing.T) {
	msg, level := safetyWarning(safety.Result{
		Level:   safety.Dangerous,
		Matches: []string{"rm on root", "sudo"},
	})
	if level != color.FgRed {
		t.Fatalf("dangerous warning color = %v, want %v", level, color.FgRed)
	}
	if !strings.Contains(msg, "could be destructive") {
		t.Fatalf("dangerous warning = %q", msg)
	}
	if !strings.Contains(msg, "rm on root, sudo") {
		t.Fatalf("dangerous warning should include matches, got %q", msg)
	}
}

func TestSafetyWarningWarning(t *testing.T) {
	msg, level := safetyWarning(safety.Result{
		Level:   safety.Warning,
		Matches: []string{"sudo"},
	})
	if level != color.FgYellow {
		t.Fatalf("warning color = %v, want %v", level, color.FgYellow)
	}
	if !strings.Contains(msg, "Caution") {
		t.Fatalf("warning message = %q", msg)
	}
}

func TestSafetyWarningSafe(t *testing.T) {
	msg, level := safetyWarning(safety.Result{Level: safety.Safe})
	if msg != "" {
		t.Fatalf("safe warning message = %q, want empty", msg)
	}
	if level != 0 {
		t.Fatalf("safe warning color = %v, want 0", level)
	}
}
