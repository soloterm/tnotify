# Terminal Detection: Identifying 12+ Terminal Emulators with Environment Variables

When building CLI tools that need terminal-specific features, the first challenge is: *which terminal is the user running?* There's no standard API for this. Instead, you have to detective your way through environment variables.

In building [tnotify](https://github.com/soloterm/tnotify), a tool that sends desktop notifications from the terminal, I needed to detect 12+ different terminal emulators to select the right notification protocol. Here's how.

## The Problem

Different terminals support different notification protocols:
- **Kitty** uses OSC 99 (the most feature-rich)
- **iTerm2** uses OSC 9 (simple but effective)
- **WezTerm/Ghostty** use OSC 777 (title + body support)
- **Apple Terminal/VS Code** have no OSC notification support at all

Send the wrong escape sequence to the wrong terminal and you'll get garbage output, or nothing at all.

## The Detection Strategy

The key insight is that terminals set unique environment variables. But not all variables are created equal—some are more reliable than others.

### Priority 1: Terminal-Specific Variables

These are set exclusively by one terminal and are the most reliable:

```go
func DetectTerminal() Terminal {
    // Kitty sets KITTY_WINDOW_ID
    if os.Getenv("KITTY_WINDOW_ID") != "" {
        return TerminalKitty
    }

    // iTerm2 sets ITERM_SESSION_ID
    if os.Getenv("ITERM_SESSION_ID") != "" {
        return TerminalITerm2
    }

    // WezTerm sets WEZTERM_PANE
    if os.Getenv("WEZTERM_PANE") != "" {
        return TerminalWezTerm
    }

    // Windows Terminal sets WT_SESSION
    if os.Getenv("WT_SESSION") != "" {
        return TerminalWindowsTerminal
    }

    // Alacritty sets ALACRITTY_WINDOW_ID
    if os.Getenv("ALACRITTY_WINDOW_ID") != "" {
        return TerminalAlacritty
    }

    // Konsole sets KONSOLE_VERSION
    if os.Getenv("KONSOLE_VERSION") != "" {
        return TerminalKonsole
    }

    // Ghostty sets GHOSTTY_RESOURCES_DIR
    if os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
        return TerminalGhostty
    }

    // ... continue to fallbacks
}
```

### Priority 2: TERM_PROGRAM Fallback

If no terminal-specific variable is found, check `TERM_PROGRAM`. This is less reliable because it can be overwritten, but it's widely set:

```go
    termProgram := os.Getenv("TERM_PROGRAM")
    switch termProgram {
    case "iTerm.app":
        return TerminalITerm2
    case "WezTerm":
        return TerminalWezTerm
    case "Apple_Terminal":
        return TerminalAppleTerminal
    case "vscode":
        return TerminalVSCode
    case "Hyper":
        return TerminalHyper
    case "ghostty":
        return TerminalGhostty
    }
```

### Priority 3: VTE-Based Terminals

GNOME Terminal, Tilix, and other GTK-based terminals use VTE (Virtual Terminal Emulator) as their backend. They set `VTE_VERSION`:

```go
    // VTE-based terminals (GNOME Terminal, Tilix, etc.)
    if os.Getenv("VTE_VERSION") != "" {
        return TerminalVTE
    }

    return TerminalUnknown
```

## Why Order Matters

The detection order is critical. Consider this scenario:

1. User runs Kitty
2. Inside Kitty, they start tmux
3. Inside tmux, they SSH to a remote server
4. On the remote server, `TERM_PROGRAM` might be set to something generic

If we checked `TERM_PROGRAM` first, we'd get the wrong answer. But `KITTY_WINDOW_ID` persists through tmux and SSH (if the environment is forwarded), giving us the correct detection.

Here's a test that verifies this priority:

```go
{
    name: "Kitty takes precedence",
    envSetup: func() {
        clearEnv(terminalEnvVars...)
        os.Setenv("KITTY_WINDOW_ID", "1")
        os.Setenv("TERM_PROGRAM", "WezTerm") // should be ignored
    },
    want: TerminalKitty,
},
```

## The Complete Environment Variable Reference

| Terminal | Primary Variable | TERM_PROGRAM Value |
|----------|-----------------|-------------------|
| Kitty | `KITTY_WINDOW_ID` | - |
| iTerm2 | `ITERM_SESSION_ID` | `iTerm.app` |
| WezTerm | `WEZTERM_PANE` | `WezTerm` |
| Ghostty | `GHOSTTY_RESOURCES_DIR` | `ghostty` |
| Windows Terminal | `WT_SESSION` | - |
| Alacritty | `ALACRITTY_WINDOW_ID` | - |
| Konsole | `KONSOLE_VERSION` | - |
| VS Code | - | `vscode` |
| Apple Terminal | - | `Apple_Terminal` |
| Hyper | - | `Hyper` |
| VTE-based | `VTE_VERSION` | varies |

## Mapping Terminals to Capabilities

Once you know the terminal, you can select the right protocol:

```go
func SelectProtocol(terminal Terminal) Protocol {
    switch terminal {
    case TerminalKitty:
        return ProtocolOSC99  // Full featured
    case TerminalITerm2:
        return ProtocolOSC9   // Message only
    case TerminalWezTerm, TerminalGhostty, TerminalVTE, TerminalFoot:
        return ProtocolOSC777 // Title + body
    default:
        return ProtocolNone   // No OSC support
    }
}
```

## Testing Terminal Detection

Testing code that depends on environment variables requires careful setup and teardown:

```go
var terminalEnvVars = []string{
    "KITTY_WINDOW_ID",
    "ITERM_SESSION_ID",
    "WEZTERM_PANE",
    "WT_SESSION",
    "ALACRITTY_WINDOW_ID",
    "KONSOLE_VERSION",
    "GHOSTTY_RESOURCES_DIR",
    "TERM_PROGRAM",
    "VTE_VERSION",
}

func TestDetectTerminal(t *testing.T) {
    // Save original environment
    snap := saveEnv(terminalEnvVars...)
    defer snap.restore()

    tests := []struct {
        name     string
        envSetup func()
        want     Terminal
    }{
        {
            name: "Kitty",
            envSetup: func() {
                clearEnv(terminalEnvVars...)
                os.Setenv("KITTY_WINDOW_ID", "1")
            },
            want: TerminalKitty,
        },
        // ... more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.envSetup()
            got := DetectTerminal()
            if got != tt.want {
                t.Errorf("DetectTerminal() = %q, want %q", got, tt.want)
            }
        })
    }
}
```

## Lessons Learned

1. **Specific beats generic**: Always check terminal-specific variables before falling back to `TERM_PROGRAM`.

2. **Environment variables persist**: Through tmux, SSH, and subshells, these variables usually survive. That's both a feature (detection works) and a bug (stale values after switching terminals).

3. **Unknown is okay**: When you can't detect the terminal, fall back gracefully. In tnotify, we try native notifications (osascript, notify-send) when OSC isn't available.

4. **Test with real environments**: Your CI environment has no terminal variables set. Mock them appropriately in tests, but also test manually in real terminals.

## Conclusion

Terminal detection isn't glamorous, but it's essential for building tools that work everywhere. The strategy is simple: check specific variables first, fall back to generic ones, and always have a graceful default.

The full implementation is available in [tnotify's detect package](https://github.com/soloterm/tnotify/tree/main/internal/detect).
