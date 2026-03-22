# 🧞 genie

Translate plain English into shell commands.

```bash
$ genie
🧞 What do you need? what's in the desktop folder?

  $ ls ~/Desktop

  Copied to clipboard!
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

# Auto-execute (asks for confirmation before running)
genie -r list all go files

# Explain an existing command
genie --explain "ls -la | grep .go"

# Show recent history
genie --history

# Use Codex instead of Claude
genie --provider codex show me all files in this folder
```

## Flags

| Flag              | Description                              |
| ----------------- | ---------------------------------------- |
| `--explain`       | Explain an existing shell command        |
| `--history`       | Show the last 20 saved requests/commands |
| `--provider`      | LLM provider: `claude` or `codex`        |
| `-r`, `--run`     | Show the command, then ask to execute it |
| `--model`         | Override the provider model              |
| `-v`, `--version` | Print version                            |

## Config

Optional config file at `~/.genie.yaml`:

```yaml
provider: claude # or codex (default: claude)
model: haiku # default is haiku for claude, gpt-5.4-mini for codex
auto_run: false # Execute the suggested command automatically (default: false)
clipboard: true # Copy the suggested command to clipboard (default: true)
```

## Features

- Translate plain English into shell commands
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

Genie sends your plain English request to the selected provider CLI in non-interactive mode. Claude uses `claude -p`; Codex uses `codex exec`. The response is parsed and displayed as a shell command. Nothing is executed unless you use the `-r` flag and confirm.
