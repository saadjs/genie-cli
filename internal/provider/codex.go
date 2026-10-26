package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

// Codex runs prompts through `codex exec`.
type Codex struct{}

func (Codex) Name() string         { return "codex" }
func (Codex) DefaultModel() string { return "gpt-6-luna" }

func (Codex) Check() error {
	return lookPath("codex", "https://developers.openai.com/codex/cli")
}

// codexArgs keeps codex exec as light as possible: no session files, no MCP
// servers, no exec-policy rules, and low reasoning effort, which is plenty
// for a one-line answer.
func codexArgs(req Request, outputPath string) []string {
	args := []string{
		"exec",
		"--skip-git-repo-check",
		"--ephemeral",
		"--ignore-rules",
		"--sandbox", "read-only",
		"--color", "never",
		"-c", `model_reasoning_effort="low"`,
		"-c", "mcp_servers={}",
		"-o", outputPath,
	}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	return append(args, combinePrompt(req))
}

func combinePrompt(req Request) string {
	if req.System == "" {
		return req.Prompt
	}
	return req.System + "\n\n" + req.Prompt
}

func (Codex) Complete(ctx context.Context, req Request) (string, error) {
	outputFile, err := os.CreateTemp("", "genie-codex-*")
	if err != nil {
		return "", fmt.Errorf("codex: %w", err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputPath)

	cmd := command(ctx, "codex", codexArgs(req, outputPath)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	if runErr != nil {
		message := extractError(stderr.String() + "\n" + stdout.String())
		if message == "" {
			message = strings.TrimSpace(stderr.String())
		}
		if message == "" {
			message = runErr.Error()
		}
		return "", fmt.Errorf("codex: %s", message)
	}

	output, err := os.ReadFile(outputPath)
	if err != nil {
		return "", fmt.Errorf("codex: reading response: %w", err)
	}
	result := strings.TrimSpace(string(output))
	if result == "" {
		return "", errors.New("codex returned an empty response")
	}
	return result, nil
}

// extractError pulls unique ERROR: lines out of codex CLI output, which
// otherwise includes a session banner, the echoed prompt, and other noise.
func extractError(output string) string {
	seen := make(map[string]bool)
	var messages []string
	for line := range strings.Lines(output) {
		line = strings.TrimSpace(line)
		rest, ok := strings.CutPrefix(line, "ERROR:")
		if !ok {
			continue
		}
		rest = apiErrorMessage(strings.TrimSpace(rest))
		if rest != "" && !seen[rest] {
			seen[rest] = true
			messages = append(messages, rest)
		}
	}
	return strings.Join(messages, "; ")
}

// apiErrorMessage unwraps a raw API error body like
// {"error":{"message":"..."}} to just its message.
func apiErrorMessage(s string) string {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if strings.HasPrefix(s, "{") && json.Unmarshal([]byte(s), &body) == nil && body.Error.Message != "" {
		return body.Error.Message
	}
	return s
}
