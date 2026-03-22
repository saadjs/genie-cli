package codex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCall(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "codex")
	script := `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o)
      out="$2"
      shift 2
      ;;
    --model|--color)
      shift 2
      ;;
    exec|--skip-git-repo-check)
      shift
      ;;
    *)
      prompt="$1"
      shift
      ;;
  esac
done
printf '%s' "$prompt" > "$out"
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write codex stub: %v", err)
	}

	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+originalPath)

	got, err := Call("ls -la", "gpt-5.4-mini")
	if err != nil {
		t.Fatalf("Call returned error: %v", err)
	}
	if got != "ls -la" {
		t.Fatalf("Call = %q, want %q", got, "ls -la")
	}
}

func TestCheckCLI(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "codex")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("failed to write codex stub: %v", err)
	}

	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+originalPath)

	if err := CheckCLI(); err != nil {
		t.Fatalf("CheckCLI returned error: %v", err)
	}
}

func TestCallError(t *testing.T) {
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "codex")
	script := `#!/bin/sh
printf '%s\n' 'API Error: rate limit' >&2
exit 1
`
	if err := os.WriteFile(scriptPath, []byte(script), 0755); err != nil {
		t.Fatalf("failed to write codex stub: %v", err)
	}

	originalPath := os.Getenv("PATH")
	t.Setenv("PATH", tempDir+string(os.PathListSeparator)+originalPath)

	_, err := Call("ls -la", "gpt-5.4-mini")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "API Error: rate limit") {
		t.Fatalf("unexpected error: %v", err)
	}
}
