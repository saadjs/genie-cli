package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/atotto/clipboard"
	"github.com/saadjs/genie-cli/internal/config"
	"github.com/saadjs/genie-cli/internal/history"
	"github.com/saadjs/genie-cli/internal/prompt"
	"github.com/saadjs/genie-cli/internal/provider"
	"github.com/saadjs/genie-cli/internal/runner"
	"github.com/saadjs/genie-cli/internal/safety"
	"github.com/saadjs/genie-cli/internal/ui"
)

var Version = "dev"

// Overridable in tests.
var (
	getProvider     = provider.Get
	copyToClipboard = clipboard.WriteAll
)

// Exit codes. With --run/--no-verify, genie exits with the command's own status.
const (
	exitOK          = 0
	exitError       = 1
	exitUsage       = 2
	exitInterrupted = 130
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

type options struct {
	run, noVerify, explain, history, version bool
	provider, model                          string
	args                                     []string
}

func parseFlags(args []string) (options, error) {
	var o options
	fs := flag.NewFlagSet("genie", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&o.run, "run", false, "")
	fs.BoolVar(&o.run, "r", false, "")
	fs.BoolVar(&o.noVerify, "no-verify", false, "")
	fs.BoolVar(&o.noVerify, "y", false, "")
	fs.BoolVar(&o.explain, "explain", false, "")
	fs.BoolVar(&o.explain, "e", false, "")
	fs.BoolVar(&o.history, "history", false, "")
	fs.BoolVar(&o.version, "version", false, "")
	fs.BoolVar(&o.version, "v", false, "")
	fs.StringVar(&o.provider, "provider", "", "")
	fs.StringVar(&o.model, "model", "", "")
	if err := fs.Parse(args); err != nil {
		return o, err
	}
	o.args = fs.Args()
	return o, nil
}

func usage() string {
	claude, _ := provider.Get("claude")
	codex, _ := provider.Get("codex")
	return fmt.Sprintf(`genie - plain English to shell commands

Usage:
  genie [flags] [request...]    With no request, genie prompts for one
  genie -e [command...]         Explain a command

Examples:
  genie list all go files
  genie "what's using port 3000?"
  genie -r compress this folder
  genie -e "tar -xzvf archive.tgz"

Flags:
  -r, --run         Ask to run the command
  -y, --no-verify   Run without asking (warnings still show)
  -e, --explain     Explain a command instead of writing one
      --provider    claude or codex (default: claude)
      --model       Model override (default: %s, %s)
      --history     Show recent requests
  -v, --version     Print version
  -h, --help        Show this help

Output:
  stdout carries only the result; prompts, warnings, and errors go to stderr.
  When stdout isn't a terminal the command is printed bare and not copied.
  With --history, non-terminal output is JSON lines.

Config (~/.genie.yaml):
  provider: claude
  model: %[1]s
  auto_run: false    # always ask to run
  clipboard: true    # copy commands when on a terminal

Exit status: 0 success, 1 error or declined, 2 usage, 130 interrupted.
With -r or -y, genie exits with the command's status.
`, claude.DefaultModel(), codex.DefaultModel())
}

func version() string {
	if Version == "dev" {
		if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
			return bi.Main.Version
		}
	}
	return Version
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	t := ui.New(stdout, stderr)

	opts, err := parseFlags(args)
	if errors.Is(err, flag.ErrHelp) {
		fmt.Fprint(stdout, usage())
		return exitOK
	}
	if err != nil {
		t.Error(fmt.Errorf("%w (see genie --help)", err))
		return exitUsage
	}
	if opts.version {
		fmt.Fprintf(stdout, "genie %s\n", version())
		return exitOK
	}
	if opts.history {
		return showHistory(t)
	}

	cfg, err := config.Load(config.Path())
	if err != nil {
		t.Warn(false, fmt.Sprintf("ignoring config: %s", err))
	}

	p, model, err := providerModel(cfg, opts.provider, opts.model)
	if err != nil {
		t.Error(err)
		return exitUsage
	}
	if err := p.Check(); err != nil {
		t.Error(err)
		return exitError
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// One buffered reader shared by every prompt, so typed-ahead input isn't
	// lost between reads.
	in := bufio.NewReader(stdin)
	inTTY := ui.IsTerminal(stdin)

	input := strings.TrimSpace(strings.Join(opts.args, " "))
	if input == "" {
		if inTTY {
			question := "🧞 What do you need?"
			if opts.explain {
				question = "🧞 Which command?"
			}
			t.Ask(question)
		}
		input, err = readLine(ctx, in)
		if ctx.Err() != nil {
			fmt.Fprintln(stderr)
			return exitInterrupted
		}
		if err != nil || input == "" {
			if !inTTY {
				t.Error(errors.New("no request given (see genie --help)"))
			}
			return exitUsage
		}
	}

	env := prompt.CurrentEnv()
	if opts.explain {
		system, user := prompt.Explain(env, input)
		reply, code := complete(ctx, t, p, provider.Request{System: system, Prompt: user, Model: model})
		if code != exitOK {
			return code
		}
		t.Explanation(input, reply)
		return exitOK
	}

	system, user := prompt.Command(env, input)
	reply, code := complete(ctx, t, p, provider.Request{System: system, Prompt: user, Model: model})
	if code != exitOK {
		return code
	}
	command, err := prompt.ParseCommand(reply)
	if err != nil {
		t.Error(err)
		return exitError
	}
	_ = history.Save(input, command)

	rating := safety.Check(command)
	// auto_run only applies on a terminal; piped callers still get the
	// command on stdout instead of a confirmation error.
	if !(opts.run || opts.noVerify || (cfg.AutoRun && inTTY)) {
		t.Command(stdout, command)
		warn(t, rating)
		if t.OutTTY && cfg.ClipboardEnabled() && copyToClipboard(command) == nil {
			t.Success("Copied to clipboard")
		}
		return exitOK
	}

	// The command's own output owns stdout, so show it on stderr.
	t.Command(stderr, command)
	warn(t, rating)
	if !opts.noVerify {
		// Never block on a pipe that may stay open forever (agents, CI).
		if !inTTY {
			t.Error(errors.New("can't confirm without a terminal; pass -y to run without asking"))
			return exitError
		}
		t.Ask("  Run? [y/N]")
		answer, _ := readLine(ctx, in)
		if ctx.Err() != nil {
			fmt.Fprintln(stderr)
			return exitInterrupted
		}
		if a := strings.ToLower(answer); a != "y" && a != "yes" {
			t.Note("Not run")
			return exitError
		}
		if t.ErrTTY {
			fmt.Fprintln(stderr)
		}
	}

	// Hand piped input the request line didn't use on to the command.
	cmdIn := stdin
	if !inTTY && in.Buffered() > 0 {
		cmdIn = in
	}
	status, err := runner.Run(command, cmdIn, stdout, stderr)
	if err != nil {
		t.Error(err)
	}
	return status
}

// complete calls the provider behind a spinner.
func complete(ctx context.Context, t *ui.Term, p provider.Provider, req provider.Request) (string, int) {
	spin := t.Spin("Thinking")
	reply, err := p.Complete(ctx, req)
	spin.Stop()
	if ctx.Err() != nil {
		return "", exitInterrupted
	}
	if err != nil {
		t.Error(err)
		return "", exitError
	}
	return reply, exitOK
}

// readLine reads one line, returning early if ctx is cancelled (Ctrl+C at a
// prompt must not wait for Enter).
func readLine(ctx context.Context, in *bufio.Reader) (string, error) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := in.ReadString('\n')
		if errors.Is(err, io.EOF) && line != "" {
			err = nil
		}
		ch <- result{strings.TrimSpace(line), err}
	}()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case r := <-ch:
		return r.line, r.err
	}
}

func warn(t *ui.Term, rating safety.Result) {
	matches := strings.Join(rating.Matches, ", ")
	switch rating.Level {
	case safety.Dangerous:
		t.Warn(true, "Destructive: "+matches)
	case safety.Warning:
		t.Warn(false, "Caution: "+matches)
	}
}

func showHistory(t *ui.Term) int {
	entries, err := history.Load(20)
	if err != nil {
		t.Error(err)
		return exitError
	}
	if !t.OutTTY {
		enc := json.NewEncoder(t.Out)
		enc.SetEscapeHTML(false)
		for _, e := range entries {
			enc.Encode(e)
		}
		return exitOK
	}
	if len(entries) == 0 {
		t.Note("No history yet")
		return exitOK
	}
	fmt.Fprintln(t.Out)
	for _, e := range entries {
		when := e.Timestamp
		if ts, err := time.Parse(time.RFC3339, e.Timestamp); err == nil {
			when = ts.Local().Format("2006-01-02 15:04")
		}
		t.HistoryEntry(when, e.Request, e.Command)
	}
	return exitOK
}

// providerModel resolves the provider and model from flags and config. The
// config's model applies only to the config's provider, so --provider codex
// doesn't inherit a Claude model name.
func providerModel(cfg config.Config, providerFlag, modelFlag string) (provider.Provider, string, error) {
	name := cfg.Provider
	if providerFlag != "" {
		name = providerFlag
	}
	p, err := getProvider(name)
	if err != nil {
		return nil, "", err
	}

	model := p.DefaultModel()
	if cfg.Model != "" {
		cfgProvider, err := getProvider(cfg.Provider)
		if err == nil && cfgProvider.Name() == p.Name() {
			model = cfg.Model
		}
	}
	if modelFlag != "" {
		model = modelFlag
	}
	return p, model, nil
}
