package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	PathOverride = filepath.Join(t.TempDir(), "history")
	defer func() { PathOverride = "" }()

	_ = Save("list files", "ls -la")
	_ = Save("disk usage", "du -sh .")
	_ = Save("find go files", "find . -name '*.go'")

	entries, err := Load(2)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Request != "find go files" {
		t.Errorf("expected newest first, got %s", entries[0].Request)
	}
	if entries[1].Request != "disk usage" {
		t.Errorf("expected second newest, got %s", entries[1].Request)
	}
}

func TestLoadEmpty(t *testing.T) {
	PathOverride = filepath.Join(t.TempDir(), "nonexistent")
	defer func() { PathOverride = "" }()

	entries, err := Load(10)
	if err != nil {
		t.Fatalf("should not error on missing file: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestLoadMalformedLine(t *testing.T) {
	p := filepath.Join(t.TempDir(), "history")
	PathOverride = p
	defer func() { PathOverride = "" }()

	_ = os.WriteFile(p, []byte("not json\n{\"timestamp\":\"t\",\"request\":\"r\",\"command\":\"c\"}\n"), 0644)

	entries, err := Load(10)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 valid entry, got %d", len(entries))
	}
}
