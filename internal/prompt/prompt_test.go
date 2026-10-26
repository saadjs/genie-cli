package prompt

import (
	"errors"
	"strings"
	"testing"
)

func TestCommandIncludesEnvAndRequestVerbatim(t *testing.T) {
	system, user := Command(Env{OS: "macOS", Shell: "zsh"}, `find "quotes" and $pecial chars`)
	if !strings.Contains(system, "macOS") || !strings.Contains(system, "zsh") {
		t.Fatalf("system prompt should name the OS and shell:\n%s", system)
	}
	if !strings.Contains(user, `find "quotes" and $pecial chars`) {
		t.Fatalf("user message should preserve the request verbatim: %q", user)
	}
	if strings.Contains(system+user, "%!") {
		t.Fatalf("prompt has a formatting error:\n%s\n%s", system, user)
	}
}

func TestExplainIncludesCommandVerbatim(t *testing.T) {
	system, user := Explain(Env{OS: "Linux", Shell: "bash"}, `awk '{print $1}' file`)
	if !strings.Contains(user, `awk '{print $1}' file`) {
		t.Fatalf("user message should preserve the command verbatim: %q", user)
	}
	if !strings.Contains(system, "Linux") || strings.Contains(system, "%!") {
		t.Fatalf("bad system prompt:\n%s", system)
	}
}

func TestShellName(t *testing.T) {
	if got := ShellName("/opt/homebrew/bin/fish"); got != "fish" {
		t.Fatalf("ShellName = %q, want fish", got)
	}
	if got := ShellName(""); got != "sh" {
		t.Fatalf("ShellName(\"\") = %q, want sh", got)
	}
}

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name, reply, want string
	}{
		{"plain", "ls -la\n", "ls -la"},
		{"fenced", "```bash\nfind . -name '*.go'\n```", "find . -name '*.go'"},
		{"fenced no lang", "```\ndu -sh .\n```", "du -sh ."},
		{"inline backticks", "`ls ~/Desktop`", "ls ~/Desktop"},
		{"dollar prefix", "$ echo hi", "echo hi"},
		{"keeps inner backticks", "echo `date`", "echo `date`"},
		{"keeps substitution", "echo $(date)", "echo $(date)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCommand(tt.reply)
			if err != nil {
				t.Fatalf("ParseCommand(%q) error: %v", tt.reply, err)
			}
			if got != tt.want {
				t.Fatalf("ParseCommand(%q) = %q, want %q", tt.reply, got, tt.want)
			}
		})
	}
}

func TestParseCommandRefusal(t *testing.T) {
	for _, reply := range []string{"```\nERROR: not a shell task\n```", "`ERROR: not a shell task`"} {
		var refused *RefusedError
		if _, err := ParseCommand(reply); !errors.As(err, &refused) {
			t.Fatalf("ParseCommand(%q) = %v, want a refusal", reply, err)
		}
	}
	_, err := ParseCommand("ERROR: that's a question about cooking")
	var refused *RefusedError
	if !errors.As(err, &refused) {
		t.Fatalf("want RefusedError, got %v", err)
	}
	if err.Error() != "that's a question about cooking" {
		t.Fatalf("refusal should be just the reason: %q", err)
	}
}

func TestParseCommandEmpty(t *testing.T) {
	for _, reply := range []string{"", "  \n", "```\n```"} {
		if _, err := ParseCommand(reply); err == nil {
			t.Fatalf("ParseCommand(%q) should fail", reply)
		}
	}
}
