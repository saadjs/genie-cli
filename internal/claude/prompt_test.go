package claude

import (
	"strings"
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	prompt := BuildPrompt("show me all files")
	if !strings.Contains(prompt, "show me all files") {
		t.Error("prompt should contain the user input")
	}
	if !strings.Contains(prompt, "shell command translator") {
		t.Error("prompt should contain the system instruction")
	}

	systemIdx := strings.Index(prompt, "shell command translator")
	userIdx := strings.Index(prompt, "show me all files")
	if systemIdx == -1 || userIdx == -1 {
		t.Fatal("prompt must contain both system instruction and user input")
	}
	if userIdx <= systemIdx {
		t.Error("prompt should place the user input after the system instruction")
	}

	if strings.Contains(prompt, "%s") || strings.Contains(prompt, "{{") {
		t.Error("prompt should not contain unexpanded template markers")
	}
}

func TestBuildPrompt_SpecialChars(t *testing.T) {
	prompt := BuildPrompt(`find files with "quotes" and $pecial chars`)
	if !strings.Contains(prompt, `"quotes"`) {
		t.Error("prompt should preserve special characters")
	}
}

func TestBuildExplainPrompt(t *testing.T) {
	prompt := BuildExplainPrompt("ls -la | grep .go")
	if !strings.Contains(prompt, "ls -la | grep .go") {
		t.Error("explain prompt should contain the command")
	}
	if !strings.Contains(prompt, "command explainer") {
		t.Error("explain prompt should contain the system instruction")
	}
}

func TestBuildExplainPrompt_SpecialChars(t *testing.T) {
	prompt := BuildExplainPrompt(`cat file.txt | awk '{print $1}'`)
	if !strings.Contains(prompt, `awk '{print $1}'`) {
		t.Error("explain prompt should preserve special characters")
	}
}
