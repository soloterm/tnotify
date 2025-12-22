# Usage

## Basic Notifications

Send a simple notification:

```bash
tnotify 'Build complete!'
```

Add a title:

```bash
tnotify -t 'My App' 'Task finished'
```

Set urgency level:

```bash
tnotify -u critical 'Server down!'
```

## Piped Input

Read the message from stdin:

```bash
echo 'Done' | tnotify -t 'Results'

# Multi-line input
cat results.txt | tnotify -t 'Test Results'
```

## Exit Code Integration

Use `-e` to pass the previous command's exit code. tnotify automatically sets urgency to critical for non-zero exit codes:

```bash
make build; tnotify -e $? 'Build finished'
```

Only notify on failure with `--if-failed`:

```bash
make test; tnotify -e $? --if-failed 'Tests failed!'
```

| Exit Code | Urgency |
|-----------|---------|
| 0 | normal |
| non-zero | critical |

## Progress Bars

Show progress in terminal tabs or taskbar. Supported by Windows Terminal, Ghostty (1.2+), iTerm2 (3.6.6+), ConEmu, and Mintty.

```bash
# Set progress percentage (0-100)
tnotify -p 50

# Error state (red)
tnotify -p 100 --progress-state error

# Paused state (yellow)
tnotify -p 75 --progress-state paused

# Indeterminate/bouncing progress (for unknown duration tasks)
tnotify --indeterminate
tnotify -I  # Short form

# Clear progress bar
tnotify --progress-clear
```

Progress states:

| State | Description |
|-------|-------------|
| `normal` | Green progress bar (default) |
| `error` | Red progress bar |
| `paused` | Yellow progress bar |
| `indeterminate` | Pulsing/animated bar |

On unsupported terminals, `-p` prints plain text instead of sending OSC sequences.

## Notification IDs (Kitty Only)

Update or replace notifications using IDs:

```bash
# Send initial notification
tnotify -i progress 'Building... 50%'

# Update same notification
tnotify -i progress 'Building... 100%'

# Close notification
tnotify --close progress
```

## Request Attention

Bounce the dock icon or flash the taskbar (iTerm2):

```bash
tnotify --attention

# With fireworks animation
tnotify --fireworks
```

## Force Specific Methods

Override auto-detection:

```bash
# Use only OSC escape sequences
tnotify --osc 'Uses escape sequences'

# Use only native notifications
tnotify --native 'Uses osascript/notify-send'

# Send terminal bell only
tnotify --bell
```

## Check Capabilities

View what your terminal supports:

```bash
tnotify --capabilities
```

Output is JSON:

```json
{
  "terminal": "ghostty",
  "protocol": "osc777",
  "supports_title": true,
  "supports_urgency": false,
  "supports_id": false,
  "supports_progress": true,
  "native_available": true
}
```

## Diagnose Issues

Test all notification methods:

```bash
tnotify --diagnose
```

This runs through each available method and reports what works. See [Troubleshooting](./troubleshooting.md) for common issues.
