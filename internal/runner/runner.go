package runner

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// Run executes command with the user's $SHELL and returns its exit code,
// using the shell convention of 128+N for a command killed by signal N.
func Run(command string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell, "-c", command)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal()), nil
		}
		return exitErr.ExitCode(), nil
	}
	if err != nil {
		return 1, err
	}
	return 0, nil
}
