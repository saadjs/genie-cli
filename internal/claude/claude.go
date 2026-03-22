package claude

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Response struct {
	Result  string `json:"result"`
	IsError bool   `json:"is_error"`
}

func CheckCLI() error {
	_, err := exec.LookPath("claude")
	if err != nil {
		return fmt.Errorf("claude CLI not found in PATH. Install it at https://docs.anthropic.com/en/docs/claude-code")
	}
	return nil
}

func Call(prompt string, model string) (string, error) {
	args := []string{"-p", "--output-format", "json"}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, prompt)

	cmd := exec.Command("claude", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	// Claude outputs JSON to either stdout or stderr
	output := stdout.Bytes()
	if len(output) == 0 {
		output = stderr.Bytes()
	}

	if len(output) == 0 {
		if runErr != nil {
			return "", fmt.Errorf("claude failed: %s", runErr)
		}
		return "", fmt.Errorf("claude returned no output")
	}

	var resp Response
	if err := json.Unmarshal(output, &resp); err != nil {
		if runErr != nil {
			return "", fmt.Errorf("claude failed: %s", runErr)
		}
		return "", fmt.Errorf("failed to parse claude response: %w", err)
	}

	if resp.IsError || runErr != nil {
		return "", fmt.Errorf("claude: %s", resp.Result)
	}

	result := strings.TrimSpace(resp.Result)
	if result == "" {
		return "", fmt.Errorf("claude returned an empty response — try rephrasing your request")
	}

	if strings.HasPrefix(result, "ERROR:") {
		return "", fmt.Errorf("%s", result)
	}

	return result, nil
}
