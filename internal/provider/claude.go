package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Claude runs prompts through `claude -p`.
type Claude struct{}

func (Claude) Name() string         { return "claude" }
func (Claude) DefaultModel() string { return "claude-haiku-5-5" }

func (Claude) Check() error {
	return lookPath("claude", "https://docs.anthropic.com/en/docs/claude-code")
}

type claudeResponse struct {
	Result  string `json:"result"`
	IsError bool   `json:"is_error"`
}

// claudeArgs strips everything a one-shot text completion doesn't need.
// Replacing the default system prompt and dropping tools, MCP servers, and
// skills roughly halves latency and cuts input tokens by ~95%. --bare would
// go further but ignores OAuth logins, so subscription users couldn't use it.
func claudeArgs(req Request) []string {
	args := []string{
		"-p",
		"--output-format", "json",
		"--system-prompt", req.System,
		"--tools", "",
		"--strict-mcp-config",
		"--disable-slash-commands",
		"--no-session-persistence",
	}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	return append(args, req.Prompt)
}

func (Claude) Complete(ctx context.Context, req Request) (string, error) {
	cmd := command(ctx, "claude", claudeArgs(req)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	output := stdout.Bytes()
	if len(bytes.TrimSpace(output)) == 0 {
		output = stderr.Bytes()
	}

	var resp claudeResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		msg := strings.TrimSpace(string(output))
		if msg == "" && runErr != nil {
			msg = runErr.Error()
		}
		if msg == "" {
			msg = "no output"
		}
		return "", fmt.Errorf("claude: %s", msg)
	}
	result := strings.TrimSpace(resp.Result)
	if resp.IsError || runErr != nil {
		if result == "" && runErr != nil {
			result = runErr.Error()
		}
		return "", fmt.Errorf("claude: %s", result)
	}
	if result == "" {
		return "", errors.New("claude returned an empty response")
	}
	return result, nil
}
