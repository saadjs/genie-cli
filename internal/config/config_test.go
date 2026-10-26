package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingIsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatalf("missing config should not error: %v", err)
	}
	if cfg.Provider != "" || !cfg.ClipboardEnabled() {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadParses(t *testing.T) {
	p := filepath.Join(t.TempDir(), "genie.yaml")
	os.WriteFile(p, []byte("provider: codex\nmodel: m\nauto_run: true\nclipboard: false\n"), 0o600)
	cfg, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "codex" || cfg.Model != "m" || !cfg.AutoRun || cfg.ClipboardEnabled() {
		t.Fatalf("parsed %+v", cfg)
	}
}

func TestLoadMalformedReportsPath(t *testing.T) {
	p := filepath.Join(t.TempDir(), "genie.yaml")
	os.WriteFile(p, []byte("provider: [codex\n"), 0o600)
	if _, err := Load(p); err == nil {
		t.Fatal("malformed config should error")
	}
}
