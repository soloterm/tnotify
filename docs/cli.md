# CLI Reference

## Synopsis

```
tnotify [message] [flags]
```

## Description

tnotify sends desktop notifications via OSC escape sequences or native tools. It automatically detects your terminal and uses the best available method.

## Arguments

| Argument | Description |
|----------|-------------|
| `message` | The notification body text. Optional if reading from stdin. |

## Flags

### Notification Content

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--title` | `-t` | | Notification title |
| `--urgency` | `-u` | `normal` | Urgency level: `low`, `normal`, `critical` |
| `--id` | `-i` | | Notification ID for updates (OSC 99 only) |
| `--close` | | | Close notification by ID (OSC 99 only) |

### Exit Code Integration

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--exit-code` | `-e` | `0` | Previous command's exit code (auto-sets urgency) |
| `--if-failed` | | `false` | Only notify if exit code is non-zero |

### Progress Bar

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--progress` | `-p` | `-1` | Show progress bar (0-100) in terminal tab/taskbar |
| `--progress-state` | | `normal` | Progress state: `normal`, `error`, `paused`, `indeterminate` |
| `--progress-clear` | | `false` | Clear/hide progress bar |
| `--indeterminate` | `-I` | `false` | Show indeterminate/bouncing progress indicator |

### Attention

| Flag | Short | Description |
|------|-------|-------------|
| `--attention` | | Request attention (bounces dock icon on macOS/iTerm2) |
| `--fireworks` | | Request attention with fireworks animation (iTerm2) |

### Method Control

| Flag | Description |
|------|-------------|
| `--osc` | Force OSC escape sequences only |
| `--native` | Force native notifications only |
| `--bell` | Send terminal bell only |

### Diagnostics

| Flag | Short | Description |
|------|-------|-------------|
| `--capabilities` | | Show terminal capabilities as JSON |
| `--diagnose` | | Test all notification methods |
| `--version` | `-v` | Show version and check for updates |

## Examples

```bash
# Simple notification
tnotify "Build complete!"

# With title
tnotify -t "My App" "Task finished"

# Critical urgency
tnotify -u critical "Server down!"

# Notification ID (Kitty)
tnotify -i progress "Building... 50%"
tnotify -i progress "Building... 100%"
tnotify --close progress

# Pipe input
echo "Done" | tnotify -t "Results"

# Exit code integration
make build; tnotify -e $? "Build finished"
make test; tnotify -e $? --if-failed "Tests failed!"

# Progress bar
tnotify -p 50
tnotify -p 100 --progress-state error
tnotify --indeterminate
tnotify --progress-clear

# Attention
tnotify --attention
tnotify --fireworks

# Force method
tnotify --osc "Uses escape sequences only"
tnotify --native "Uses osascript/notify-send only"
tnotify --bell

# Diagnostics
tnotify --capabilities
tnotify --diagnose
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | Error (no message provided, notification failed, etc.) |

## Environment Variables

tnotify detects terminals using these environment variables:

| Variable | Used For |
|----------|----------|
| `TERM_PROGRAM` | Terminal identification (iTerm2, WezTerm, etc.) |
| `KITTY_WINDOW_ID` | Kitty terminal detection |
| `GHOSTTY_RESOURCES_DIR` | Ghostty terminal detection |
| `TERM` | General terminal capabilities |
| `TMUX` | tmux multiplexer detection |
| `STY` | GNU Screen detection |
