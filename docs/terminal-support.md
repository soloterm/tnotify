# Terminal Support

tnotify supports multiple notification protocols and automatically selects the best one for your terminal.

## Protocol Overview

| Protocol | Terminals | Features |
|----------|-----------|----------|
| OSC 99 | Kitty | Title, body, urgency, notification IDs |
| OSC 777 | Ghostty, WezTerm, VTE terminals | Title and body |
| OSC 9 | iTerm2, Windows Terminal, ConEmu, Mintty | Body only, progress bars |
| Native | All (fallback) | Title, body, urgency |

## Terminal Compatibility Matrix

| Terminal | Protocol | Title | Urgency | IDs | Progress | Attention |
|----------|----------|-------|---------|-----|----------|-----------|
| Kitty | OSC 99 | Yes | Yes | Yes | No | No |
| iTerm2 | OSC 9 | No | No | No | Yes | Yes |
| WezTerm | OSC 777 | Yes | No | No | No | No |
| Ghostty | OSC 777 | Yes | No | No | Yes | No |
| Windows Terminal | OSC 9 | No | No | No | Yes | No |
| ConEmu | OSC 9 | No | No | No | Yes | No |
| Mintty | OSC 9 | No | No | No | Yes | No |
| VTE (GNOME Terminal) | OSC 777 | Yes | No | No | No | No |
| foot | OSC 777 | Yes | No | No | No | No |
| Others | Native | Yes | Yes | No | No | No |

## Protocol Details

### OSC 99 (Kitty)

The most feature-rich protocol, supporting:
- Notification title and body
- Three urgency levels (low, normal, critical)
- Notification IDs for updating/closing notifications
- Icon support (not exposed by tnotify)

### OSC 777 (Ghostty, WezTerm, VTE)

Desktop notification protocol originating from VTE:
- Notification title and body
- Simple format: `OSC 777;notify;title;body ST`

### OSC 9 (iTerm2, Windows Terminal)

Originally designed for iTerm2:
- Body text only (no title support)
- Extended for progress bars (OSC 9;4)
- iTerm2 supports attention requests

## Multiplexer Support

### tmux

OSC sequences work in tmux 3.2+ with passthrough enabled:

```bash
# Add to ~/.tmux.conf
set -g allow-passthrough on
```

tnotify automatically wraps sequences for tmux using `\ePtmux;...\e\\`.

### GNU Screen

tnotify uses DCS passthrough for GNU Screen:

```
\eP...\e\\
```

No additional configuration required.

## Native Fallbacks

When OSC notifications are unavailable, tnotify falls back to native tools:

| Platform | Tool | Notes |
|----------|------|-------|
| macOS | osascript | Uses AppleScript to trigger notifications |
| Linux | notify-send | Requires libnotify to be installed |
| Windows | PowerShell | Uses toast notification API |

### Installing notify-send on Linux

```bash
# Debian/Ubuntu
sudo apt install libnotify-bin

# Fedora
sudo dnf install libnotify

# Arch
sudo pacman -S libnotify
```

## Checking Your Terminal

Run the capabilities command to see what your terminal supports:

```bash
tnotify --capabilities
```

Run diagnostics to test each method:

```bash
tnotify --diagnose
```
