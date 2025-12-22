# Introduction

tnotify is a standalone CLI tool for sending desktop notifications from the terminal. It uses OSC (Operating System Command) escape sequences when available, with intelligent fallbacks to native notification systems.

## Why tnotify?

**Works over SSH**: OSC escape sequences pass through SSH connections, so you can receive notifications from remote servers.

**Multiplexer-friendly**: Works inside tmux and GNU Screen with proper passthrough wrapping.

**Cross-platform**: Runs on macOS, Linux, and Windows with no runtime dependencies.

**Zero configuration**: Automatically detects your terminal and selects the best notification method.

## Key Features

- **OSC-first approach**: Uses terminal escape sequences that work over SSH and inside multiplexers
- **Smart fallbacks**: Falls back to native tools (osascript, notify-send, PowerShell) when OSC is unavailable
- **Progress bars**: Show progress in terminal tabs or taskbar (Windows Terminal, Ghostty, iTerm2)
- **Exit code integration**: Automatically set notification urgency based on command success/failure
- **Notification IDs**: Update or close notifications by ID (Kitty terminal)

## Quick Start

```bash
# Install via Homebrew
brew install soloterm/tap/tnotify

# Send a notification
tnotify 'Build complete!'

# With title
tnotify -t 'My App' 'Task finished'

# Notify on command failure
make test; tnotify -e $? --if-failed 'Tests failed!'
```

## How It Works

tnotify detects your terminal emulator and selects the appropriate notification protocol:

1. **OSC 99** (Kitty) - Full featured with urgency levels and notification IDs
2. **OSC 777** (Ghostty, WezTerm, VTE terminals) - Title and body support
3. **OSC 9** (iTerm2, Windows Terminal) - Basic notifications and progress bars
4. **Native fallback** - osascript (macOS), notify-send (Linux), PowerShell (Windows)

If none of these work, tnotify sends a terminal bell as a last resort.

## Related Projects

tnotify is part of the SoloTerm ecosystem:

- [Solo](https://github.com/soloterm/solo) - All-in-one Laravel command for local development
- [Notify](https://github.com/soloterm/notify) - PHP package for desktop notifications via OSC escape sequences
- [Screen](https://github.com/soloterm/screen) - Pure PHP terminal renderer
