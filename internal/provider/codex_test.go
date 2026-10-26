package provider

import (
	"context"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// codexStub records its args and writes the prompt (last arg) to the -o file.
const codexStub = `for a in "$@"; do printf '%s\n' "$a"; done > "$(dirname "$0")/args"
out=""
while [ "$#" -gt 1 ]; do
  if [ "$1" = "-o" ]; then out="$2"; fi
  shift
done
printf '%s\n' "$1" > "$out"
`

func TestCodexComplete(t *testing.T) {
	dir := stubCLI(t, "codex", codexStub)

	got, err := Codex{}.Complete(context.Background(), Request{System: "SYS", Prompt: "Request: list", Model: "m1"})
	if err != nil {
		t.Fatalf("Complete error: %v", err)
	}
	if got != "SYS\n\nRequest: list" {
		t.Fatalf("Complete = %q, want system and prompt combined", got)
	}

	args := readArgs(t, filepath.Join(dir, "args"))
	for _, flag := range []string{"--ephemeral", "--skip-git-repo-check", "mcp_servers={}", `model_reasoning_effort="low"`} {
		if !slices.Contains(args, flag) {
			t.Fatalf("missing %s in %q", flag, args)
		}
	}
	if i := slices.Index(args, "--model"); i < 0 || args[i+1] != "m1" {
		t.Fatalf("model not passed, got %q", args)
	}
}

func TestCodexCheckCLI(t *testing.T) {
	stubCLI(t, "codex", "exit 0\n")
	if err := (Codex{}).Check(); err != nil {
		t.Fatalf("Check returned error: %v", err)
	}
}

func TestCodexCompleteError(t *testing.T) {
	stubCLI(t, "codex", `printf '%s\n' 'banner' 'ERROR: rate limit' 'ERROR: rate limit' >&2
exit 1
`)
	_, err := Codex{}.Complete(context.Background(), Request{Prompt: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "codex: rate limit" {
		t.Fatalf("error = %q, want deduplicated ERROR line", err)
	}
}

func TestExtractError(t *testing.T) {
	got := extractError("banner\nERROR: first\nnoise\nERROR: first\nERROR: second\n")
	if got != "first; second" {
		t.Fatalf("extractError = %q, want %q", got, "first; second")
	}
	got = extractError(`ERROR: {"type":"error","status":400,"error":{"type":"invalid_request_error","message":"The 'x' model is not supported."}}`)
	if got != "The 'x' model is not supported." {
		t.Fatalf("extractError should unwrap API JSON, got %q", got)
	}
	if got := extractError("no error here"); got != "" {
		t.Fatalf("extractError = %q, want empty", got)
	}
}

func TestCodexCompleteEmpty(t *testing.T) {
	stubCLI(t, "codex", `while [ "$#" -gt 0 ]; do [ "$1" = "-o" ] && : > "$2"; shift; done`)
	_, err := Codex{}.Complete(context.Background(), Request{Prompt: "x"})
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("want empty-response error, got %v", err)
	}
}
