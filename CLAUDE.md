# genie

Go CLI that turns plain English into a shell command by shelling out to the `claude` or `codex` CLI.

## Commands

- `make check`: gofmt, vet, and race tests. Run before finishing any change.
- `make build`: builds `bin/genie`.
- Manual check: `./bin/genie list go files </dev/null` (piped mode); drive the interactive UI in tmux.

## Layout

- `cmd/genie`: flags and the `run()` flow. `run` takes stdin/stdout/stderr so tests drive it end to end with a fake provider (`getProvider`, `copyToClipboard` are swappable vars).
- `internal/provider`: one file per CLI behind the `Provider` interface. Tests install stub executables on PATH (`stubCLI`).
- `internal/prompt`: system prompts (include OS and shell) and `ParseCommand`, which cleans model replies.
- `internal/ui`: all terminal rendering, spinner, word wrap.
- `internal/safety`, `internal/history`, `internal/config`, `internal/runner`: small, single-purpose.

## Rules

- Output contract: stdout carries only results; everything else goes to stderr. Non-terminal streams get plain text with no ANSI codes, and the clipboard is only touched when stdout is a terminal. Agents and pipes depend on this, and `cmd/genie` tests enforce it.
- Never block on non-terminal stdin for confirmation; `-r` without a TTY errors and points to `-y`.
- Provider speed flags are deliberate (see comments in `claude.go`, `codex.go`). Don't use `claude --bare`: it ignores OAuth logins. Benchmark against real CLIs before and after changing them.
- Check `claude --help` / `codex exec --help` before adding flags; both CLIs change often.
- Tests that hit `history` must set `HOME` to a temp dir so they never write to the real `~/.genie_history`.
