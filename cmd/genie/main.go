package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/fatih/color"
	"github.com/saadjs/genie-cli/internal/claude"
	"github.com/saadjs/genie-cli/internal/codex"
	"github.com/saadjs/genie-cli/internal/config"
	"github.com/saadjs/genie-cli/internal/display"
	"github.com/saadjs/genie-cli/internal/history"
	"github.com/saadjs/genie-cli/internal/runner"
	"github.com/saadjs/genie-cli/internal/safety"
)

var Version = "dev"

func main() {
	runFlag := flag.Bool("run", false, "Execute the suggested command")
	flag.BoolVar(runFlag, "r", false, "Execute the suggested command (shorthand)")
	providerFlag := flag.String("provider", "", "LLM provider to use: claude or codex")
	modelFlag := flag.String("model", "", "Override the provider model")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	flag.BoolVar(versionFlag, "v", false, "Print version (shorthand)")
	historyFlag := flag.Bool("history", false, "Show recent command history")
	explainFlag := flag.String("explain", "", "Explain what a command does")
	noVerifyFlag := flag.Bool("no-verify", false, "Skip confirmation prompt (still shows safety warnings)")
	flag.BoolVar(noVerifyFlag, "y", false, "Skip confirmation prompt (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "genie - translate plain English into shell commands\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  genie                          Interactive mode (recommended)\n")
		fmt.Fprintf(os.Stderr, "  genie [flags] <request>        Inline mode\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  genie                          Prompts you for input (handles special chars)\n")
		fmt.Fprintf(os.Stderr, "  genie show me all files        Inline, no quotes needed for simple text\n")
		fmt.Fprintf(os.Stderr, "  genie \"what's in this folder?\" Use double quotes for apostrophes/special chars\n")
		fmt.Fprintf(os.Stderr, "  genie -r list all go files     Suggest command and run it immediately\n")
		fmt.Fprintf(os.Stderr, "  genie --no-verify delete tmp   Run without confirmation (shows risk)\n")
		fmt.Fprintf(os.Stderr, "  genie --explain \"ls -la\"       Explain what a command does\n")
		fmt.Fprintf(os.Stderr, "  genie --history                Show recent command history\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nConfig (optional): ~/.genie.yaml\n")
		fmt.Fprintf(os.Stderr, "  provider: claude  # LLM provider: claude or codex (default: claude)\n")
		fmt.Fprintf(os.Stderr, "  model: haiku      # Model override (defaults: haiku for claude, gpt-5.4-mini for codex)\n")
		fmt.Fprintf(os.Stderr, "  auto_run: false   # Execute the suggested command automatically\n")
		fmt.Fprintf(os.Stderr, "  clipboard: true   # Copy suggested command to clipboard\n")
	}

	flag.Parse()

	if *versionFlag {
		fmt.Printf("genie %s\n", Version)
		os.Exit(0)
	}

	if *historyFlag {
		showHistory()
		os.Exit(0)
	}

	cfg := config.Load()

	provider, model, err := providerModel(cfg, *providerFlag, *modelFlag)
	if err != nil {
		color.New(color.FgRed).Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	if err := checkCLI(provider); err != nil {
		color.New(color.FgRed).Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	if *explainFlag != "" {
		explainCommand(*explainFlag, provider, model)
		os.Exit(0)
	}

	var input string
	if flag.NArg() == 0 {
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

	shouldRun := cfg.AutoRun || *runFlag || *noVerifyFlag

	prompt := claude.BuildPrompt(input)

	spin := display.NewSpinner()
	result, err := callProvider(provider, prompt, model)
	spin.Stop()
	if err != nil {
		color.New(color.FgRed).Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	_ = history.Save(input, result)

	rating := safety.Check(result)
	if warning, level := safetyWarning(rating); warning != "" {
		color.New(level).Fprintf(os.Stderr, "%s", warning)
	}

	copied := false
	if clipboardEnabled(cfg) {
		if err := clipboard.WriteAll(result); err == nil {
			copied = true
		}
	}

	display.Command(result, shouldRun, copied)

	if shouldRun {
		if !*noVerifyFlag {
			color.New(color.FgYellow).Print("  Run this command? [y/N] ")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer != "y" && answer != "yes" {
				fmt.Println("  Aborted.")
				os.Exit(0)
			}
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

func showHistory() {
	entries, err := history.Load(20)
	if err != nil {
		color.New(color.FgRed).Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	if len(entries) == 0 {
		color.New(color.FgHiBlack).Println("  No history yet.")
		return
	}
	fmt.Println()
	for _, e := range entries {
		ts, _ := time.Parse(time.RFC3339, e.Timestamp)
		color.New(color.FgHiBlack).Printf("  %s  ", ts.Format("2006-01-02 15:04"))
		color.New(color.FgMagenta).Printf("\"%s\"\n", e.Request)
		color.New(color.FgCyan, color.Bold).Print("  $ ")
		color.New(color.FgWhite, color.Bold).Println(e.Command)
		fmt.Println()
	}
}

func explainCommand(command, provider, model string) {
	prompt := claude.BuildExplainPrompt(command)

	spin := display.NewSpinner()
	result, err := callProvider(provider, prompt, model)
	spin.Stop()
	if err != nil {
		color.New(color.FgRed).Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	display.Explanation(command, result)
}

func providerModel(cfg config.Config, providerFlag, modelFlag string) (string, string, error) {
	provider := "claude"
	if cfg.Provider != "" {
		provider = strings.ToLower(cfg.Provider)
	}
	if providerFlag != "" {
		provider = strings.ToLower(providerFlag)
	}

	switch provider {
	case "claude", "codex":
	default:
		return "", "", fmt.Errorf("unsupported provider %q; expected claude or codex", provider)
	}

	model := defaultModel(provider)
	if cfg.Model != "" {
		model = cfg.Model
	}
	if modelFlag != "" {
		model = modelFlag
	}

	return provider, model, nil
}

func defaultModel(provider string) string {
	if provider == "codex" {
		return "gpt-5.4-mini"
	}
	return "haiku"
}

func checkCLI(provider string) error {
	switch provider {
	case "claude":
		return claude.CheckCLI()
	case "codex":
		return codex.CheckCLI()
	default:
		return fmt.Errorf("unsupported provider %q", provider)
	}
}

func callProvider(provider, prompt, model string) (string, error) {
	switch provider {
	case "claude":
		return claude.Call(prompt, model)
	case "codex":
		return codex.Call(prompt, model)
	default:
		return "", fmt.Errorf("unsupported provider %q", provider)
	}
}

func clipboardEnabled(cfg config.Config) bool {
	return cfg.Clipboard == nil || *cfg.Clipboard
}

func safetyWarning(rating safety.Result) (string, color.Attribute) {
	switch rating.Level {
	case safety.Dangerous:
		return fmt.Sprintf("\n  ⚠️  This command could be destructive (%s)\n", strings.Join(rating.Matches, ", ")), color.FgRed
	case safety.Warning:
		return fmt.Sprintf("\n  ⚠️  Caution: this command uses %s\n", strings.Join(rating.Matches, ", ")), color.FgYellow
	default:
		return "", 0
	}
}
