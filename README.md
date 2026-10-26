# 🧞 genie

Translate plain English into shell commands.

```bash
$ genie
🧞 What do you need? what's in the desktop folder?

  $ ls ~/Desktop

  ✓ Copied to clipboard
```

Genie supports both Claude and Codex as non-interactive backends. Claude is the default provider.

## Prerequisites

- One supported provider CLI installed and logged in:
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) for the default `claude` provider
- [OpenAI Codex CLI](https://developers.openai.com/codex/cli) for the optional `codex` provider

## Install

Homebrew is the recommended install method:

### Homebrew

```bash
brew install saadjs/tap/genie
```

## Usage

```bash
# Interactive mode (recommended — handles apostrophes and special chars)
genie

# Inline mode
genie show me all files in this folder
genie "what's in the desktop folder?"

# Ask before running the suggested command
genie -r list all go files

# Run without asking (safety warnings still show)
genie -y list all go files

# Explain an existing command
genie -e "ls -la | grep .go"

# Show recent history
genie --history

# Use Codex instead of Claude
genie --provider codex show me all files in this folder
```

## Flags

| Flag                | Description                               |
| ------------------- | ----------------------------------------- |
| `-r`, `--run`       | Show the command, then ask to run it      |
| `-y`, `--no-verify` | Run the command without asking            |
| `-e`, `--explain`   | Explain a command instead of writing one  |
| `--history`         | Show the last 20 requests and commands    |
| `--provider`        | LLM provider: `claude` or `codex`         |
| `--model`           | Override the provider model               |
| `-v`, `--version`   | Print version                             |

## Config

Optional config file at `~/.genie.yaml`:

```yaml
provider: claude # or codex (default: claude)
model: claude-haiku-5-5 # default is claude-haiku-5-5 for claude, gpt-6-luna for codex
auto_run: false # Always ask to run the suggested command (default: false)
clipboard: true # Copy the suggested command to clipboard (default: true)
```

`model` applies to the configured `provider` only, so `--provider codex` falls back to the Codex default.

## Scripts and agents

Genie keeps stdout clean so it composes with pipes and is safe to call from scripts and coding agents:

- stdout carries only the result. Prompts, the spinner, warnings, and errors go to stderr.
- When stdout isn't a terminal, the command is printed bare (no color, no `$`) and the clipboard is left alone.
- `--history` prints JSON lines when piped.
- `-r` refuses to prompt without a terminal; pass `-y` to run non-interactively.
- Exit status: `0` success, `1` error or declined, `2` usage error, `130` interrupted. With `-r`/`-y`, genie exits with the command's own status.

```bash
genie list all go files | xargs wc -l
```

## Features

- Translate plain English into shell commands tailored to your OS and shell
- Explain existing shell commands with `--explain`
- Show recent history with `--history`
- Copy suggested commands to the clipboard by default
- Warn before risky commands such as `sudo`, recursive `rm`, `mkfs`, and device writes
- Optionally run the suggested command after confirmation

## Build from source

```bash
git clone https://github.com/saadjs/genie-cli.git
cd genie-cli
make build
./bin/genie
```

## Testing

```bash
make test
```

## How it works

Genie sends your request to the selected provider CLI in non-interactive mode: `claude -p` or `codex exec`. It strips everything a one-shot answer doesn't need (tools, MCP servers, session files, the agent system prompt), which roughly halves Claude's startup latency. The reply is cleaned up and displayed as a shell command. Nothing is executed unless you use `-r` and confirm, or pass `-y`.
