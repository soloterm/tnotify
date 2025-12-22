# Troubleshooting

## Diagnostic Command

Start troubleshooting by running the diagnostic command:

```bash
tnotify --diagnose
```

This tests each notification method and reports what works.

## Common Issues

### Notification Doesn't Appear

**Terminal is focused**: Many terminals suppress notifications when the terminal window is focused. Test by switching to another window:

```bash
sleep 3 && tnotify 'Hello!'
# Switch windows during the sleep
```

**Focus Mode (macOS)**: If you're using Focus/Do Not Disturb, your terminal app must be in the allowed apps list:

1. Open System Settings → Focus → [Your Focus Mode]
2. Go to Allowed Apps
3. Add your terminal (Ghostty, iTerm2, etc.)

**Notification permissions (macOS)**: Ensure your terminal has notifications enabled:

1. Open System Settings → Notifications
2. Find your terminal app
3. Set notification style to "Banners" or "Alerts"

### Native Fallback Not Working

**macOS - Script Editor**: Native notifications use `osascript`, which sends notifications as "Script Editor". Enable notifications for Script Editor in System Settings → Notifications.

**Linux - notify-send not found**: Install libnotify:

```bash
# Debian/Ubuntu
sudo apt install libnotify-bin

# Fedora
sudo dnf install libnotify

# Arch
sudo pacman -S libnotify
```

**Windows - Focus Assist**: Windows Focus Assist blocks notifications. Check Settings → System → Focus Assist, or click the notification icon in the system tray.

### tmux Issues

**OSC sequences not working**: tmux requires passthrough mode for OSC sequences. Enable it:

```bash
# Add to ~/.tmux.conf
set -g allow-passthrough on

# Or set for current session
tmux set -g allow-passthrough on
```

**tmux version too old**: Passthrough requires tmux 3.2 or later. Check your version:

```bash
tmux -V
```

Upgrade tmux if needed:

```bash
# macOS
brew upgrade tmux

# Debian/Ubuntu
sudo apt install tmux
```

### VTE Terminals (GNOME Terminal, Tilix)

**OSC 777 not working**: VTE's OSC 777 support requires Fedora's VTE patches or manual configuration. On Ubuntu/Arch, add to your shell config:

```bash
# ~/.bashrc or ~/.zshrc
if [ "$VTE_VERSION" ]; then
    source /etc/profile.d/vte.sh 2>/dev/null
fi
```

## Debugging

### Check Terminal Detection

```bash
tnotify --capabilities
```

Look at the `terminal` and `protocol` fields to see what tnotify detected.

### Test Each Method

Force a specific method to isolate issues:

```bash
# Test OSC only
tnotify --osc 'Test OSC'

# Test native only
tnotify --native 'Test native'

# Test bell
tnotify --bell
```

### Environment Variables

Check what environment variables tnotify sees:

```bash
echo "TERM_PROGRAM: $TERM_PROGRAM"
echo "KITTY_WINDOW_ID: $KITTY_WINDOW_ID"
echo "GHOSTTY_RESOURCES_DIR: $GHOSTTY_RESOURCES_DIR"
echo "TERM: $TERM"
echo "TMUX: $TMUX"
```

## Platform-Specific Notes

### macOS

- Native notifications require Script Editor permissions
- OSC notifications work best with iTerm2, Kitty, or Ghostty
- Terminal.app has limited notification support

### Linux

- Ensure a notification daemon is running (dunst, mako, etc.)
- Wayland compositors may handle notifications differently
- X11 typically uses libnotify via D-Bus

### Windows

- Windows Terminal supports OSC 9 for progress bars
- PowerShell toast notifications require Windows 10+
- WSL can use native Windows notifications
