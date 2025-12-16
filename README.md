# tnotify

Send desktop notifications from the terminal via OSC escape sequences or native tools.

## Features

- **OSC-first**: Works over SSH, inside tmux/screen
- **Smart fallbacks**: Uses native tools when OSC unavailable
- **Cross-platform**: macOS, Linux, Windows
- **Zero dependencies**: Single binary, no runtime requirements

## Installation

### Homebrew (macOS/Linux)

```bash
brew install soloterm/tap/tnotify
```

### Go Install

```bash
go install github.com/soloterm/tnotify@latest
```

### Binary Download

Download from [GitHub Releases](https://github.com/soloterm/tnotify/releases).

## Usage

```bash
# Simple notification
tnotify 'Build complete!'

# With title
tnotify -t 'My App' 'Task finished'

# With urgency (low, normal, critical)
tnotify -u critical 'Server down!'

# Notification ID for updates (Kitty only)
tnotify -i progress 'Building... 50%'
tnotify -i progress 'Building... 100%'
tnotify --close progress

# Pipe input
echo 'Done' | tnotify -t 'Results'

# Force specific method
tnotify --osc 'Uses escape sequences only'
tnotify --native 'Uses osascript/notify-send only'
tnotify --bell  # Just beep

# Show capabilities
tnotify --capabilities

# Diagnose notification issues
tnotify --diagnose

# Exit code integration (auto-sets urgency)
make build; tnotify -e $? 'Build finished'      # critical if failed
make test; tnotify -e $? --if-failed 'Tests failed!'  # only notify on failure
```

## Terminal Support

| Terminal | Protocol | Title | Urgency | IDs |
|----------|----------|-------|---------|-----|
| Kitty | OSC 99 | ✓ | ✓ | ✓ |
| iTerm2 | OSC 9 | ✗ | ✗ | ✗ |
| WezTerm | OSC 777 | ✓ | ✗ | ✗ |
| Ghostty | OSC 777 | ✓ | ✗ | ✗ |
| VTE (GNOME Terminal) | OSC 777 | ✓ | ✗ | ✗ |
| foot | OSC 777 | ✓ | ✗ | ✗ |
| Others | Native fallback | ✓ | ✓ | ✗ |

## Exit Code Integration

Use `-e` / `--exit-code` to pass the previous command's exit code:

```bash
# Notify with auto-urgency (critical if non-zero)
long-running-task; tnotify -e $? 'Task complete'

# Only notify on failure
make test; tnotify -e $? --if-failed 'Tests failed!'

# Combine with title
./deploy.sh; tnotify -e $? -t 'Deploy' 'Finished'
```

| Exit Code | Urgency |
|-----------|---------|
| 0 | normal |
| non-zero | critical |

The `--if-failed` flag skips the notification entirely if the exit code is 0.

## Native Fallbacks

When OSC notifications aren't supported, tnotify falls back to:

- **macOS**: `osascript` (AppleScript)
- **Linux**: `notify-send` (libnotify)
- **Windows**: PowerShell toast notifications

## Troubleshooting

**Notification doesn't appear?**

Run the diagnostic command to test all notification methods:

```bash
tnotify --diagnose
```

This will show your detected terminal, test each method, and report what works.

**Common issues:**

1. **macOS Focus Mode** — If you're using Focus/Do Not Disturb, your terminal app must be added to the allowed apps list. Go to System Settings → Focus → [Your Focus Mode] → Allowed Apps and add Ghostty, iTerm2, or your terminal.

2. **Terminal is focused** — Many terminals suppress notifications when the terminal window is focused. This is intentional. Test by switching to another window:
   ```bash
   sleep 3 && tnotify 'Hello!'  # Switch windows during the sleep
   ```

3. **Notification permissions** — Check System Settings → Notifications and ensure your terminal app has notifications enabled with "Banners" or "Alerts" style.

4. **Native fallback on macOS** — Native notifications use `osascript`, which sends notifications as "Script Editor". Enable notifications for Script Editor in System Settings → Notifications.

5. **Linux: notify-send not installed** — The native fallback requires `libnotify`. Install it with:
   ```bash
   # Debian/Ubuntu
   sudo apt install libnotify-bin
   # Fedora
   sudo dnf install libnotify
   # Arch
   sudo pacman -S libnotify
   ```

6. **Windows: Focus Assist** — Windows Focus Assist (Do Not Disturb) blocks notifications. Check Settings → System → Focus Assist, or click the notification icon in the system tray.

## License

MIT
