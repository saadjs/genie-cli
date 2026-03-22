package codex

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func CheckCLI() error {
	_, err := exec.LookPath("codex")
	if err != nil {
		return fmt.Errorf("codex CLI not found in PATH. Install it from https://developers.openai.com/codex/cli")
	}
	return nil
}

func Call(prompt string, model string) (string, error) {
	outputFile, err := os.CreateTemp("", "genie-codex-output-*")
	if err != nil {
		return "", fmt.Errorf("failed to create codex output file: %w", err)
	}
	outputPath := outputFile.Name()
	outputFile.Close()
	defer os.Remove(outputPath)

	args := []string{"exec", "--skip-git-repo-check", "--color", "never", "-o", outputPath}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, prompt)

	cmd := exec.Command("codex", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	runErr := cmd.Run()

	output, readErr := os.ReadFile(outputPath)
	result := strings.TrimSpace(string(output))
	if readErr != nil && runErr == nil {
		return "", fmt.Errorf("failed to read codex response: %w", readErr)
	}

	if runErr != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = strings.TrimSpace(stdout.String())
		}
		if message == "" {
			message = runErr.Error()
		}
		return "", fmt.Errorf("codex: %s", message)
	}

	if result == "" {
		return "", fmt.Errorf("codex returned an empty response")
	}

	if strings.HasPrefix(result, "ERROR:") {
		return "", fmt.Errorf("%s", result)
	}

	return result, nil
}
