package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/saadjs/genie-cli/internal/claude"
	"github.com/saadjs/genie-cli/internal/config"
	"github.com/saadjs/genie-cli/internal/display"
	"github.com/saadjs/genie-cli/internal/runner"
)

var Version = "dev"

func main() {
	runFlag := flag.Bool("run", false, "Execute the suggested command")
	flag.BoolVar(runFlag, "r", false, "Execute the suggested command (shorthand)")
	modelFlag := flag.String("model", "", "Override the Claude model (default: haiku)")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.BoolVar(versionFlag, "v", false, "Print version (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "genie - translate plain English into shell commands\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  genie                          Interactive mode (recommended)\n")
		fmt.Fprintf(os.Stderr, "  genie [flags] <request>        Inline mode\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  genie                          Prompts you for input (handles special chars)\n")
		fmt.Fprintf(os.Stderr, "  genie show me all files        Inline, no quotes needed for simple text\n")
		fmt.Fprintf(os.Stderr, "  genie \"what's in this folder?\" Use double quotes for apostrophes/special chars\n")
		fmt.Fprintf(os.Stderr, "  genie -r list all go files     Suggest command and run it immediately\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nConfig (optional): ~/.genie.yaml\n")
		fmt.Fprintf(os.Stderr, "  model: haiku      # Claude model (default: haiku)\n")
		fmt.Fprintf(os.Stderr, "  auto_run: false   # Execute the suggested command automatically\n")
	}

	flag.Parse()

	if *versionFlag {
		fmt.Printf("genie %s\n", Version)
		os.Exit(0)
	}

	if err := claude.CheckCLI(); err != nil {
		color.New(color.FgRed).Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	cfg := config.Load()

	var input string
	if flag.NArg() == 0 {
		// Interactive mode — avoids shell quoting issues
		color.New(color.FgMagenta, color.Bold).Print("🧞 What do you need? ")
		reader := bufio.NewReader(os.Stdin)
		line, err := readInput(reader)
		if err != nil {
			os.Exit(1)
		}
		input = line
		if input == "" {
			flag.Usage()
			os.Exit(1)
		}
	} else {
		input = strings.Join(flag.Args(), " ")
	}

	model := "haiku"
	if cfg.Model != "" {
		model = cfg.Model
	}
	if *modelFlag != "" {
		model = *modelFlag
	}

	shouldRun := cfg.AutoRun || *runFlag

	prompt := claude.BuildPrompt(input)

	spin := display.NewSpinner()
	result, err := claude.Call(prompt, model)
	spin.Stop()
	if err != nil {
		color.New(color.FgRed).Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	display.Command(result, shouldRun)

	if shouldRun {
		color.New(color.FgYellow).Print("  Run this command? [y/N] ")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("  Aborted.")
			os.Exit(0)
		}
		fmt.Println()
		if err := runner.Run(result); err != nil {
			os.Exit(1)
		}
	}
}

func readInput(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

