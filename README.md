# genie

Translate plain English into shell commands.

```
$ genie
🧞 What do you need? what's in the desktop folder?

  $ ls ~/Desktop

  Copy and paste to run, or use genie -r to auto-execute.
```

### Prerequisites

- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) installed and logged in

## Install

### Homebrew

```bash
brew install saadjs/tap/genie
```

### Script

```bash
curl -sL https://github.com/saadjs/genie-cli/releases/latest/download/install.sh | bash
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
```

## Flags

| Flag              | Description                              |
| ----------------- | ---------------------------------------- |
| `-r`, `--run`     | Show the command, then ask to execute it |
| `--model`         | Override Claude model (default: `haiku`) |
| `-v`, `--version` | Print version                            |

## Config

Optional config file at `~/.genie.yaml`:

```yaml
model: haiku # Claude model (default: haiku for speed)
auto_run: false # Always prompt to run (default: false)
```

## Build from source

```bash
git clone https://github.com/saadjs/genie-cli.git
cd genie-cli
make build
./bin/genie
```

## How it works

Genie sends your plain English request to Claude Code CLI in non-interactive mode (`claude -p`), with a prompt that instructs it to return only the shell command. The response is parsed from JSON and displayed. Nothing is executed unless you use the `-r` flag and confirm.
