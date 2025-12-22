# Graceful Degradation in CLI Tools: The Fallback Chain Pattern

Terminal notifications are weird. Some terminals support OSC escape sequences. Others don't. Some users are SSH'd into remote machines. Others are working locally. Some run tmux. Others don't.

Building a notification tool that works everywhere means accepting failure as the default state and planning for it.

## The Problem: Nothing Works Everywhere

When I started building tnotify, I wanted a simple command to send desktop notifications from the terminal. The usual approach would be to pick one mechanism and call it done.

But that doesn't work. Kitty supports OSC 99 for rich notifications with urgency levels and IDs. iTerm2 uses OSC 9. WezTerm and Ghostty use OSC 777. And plenty of terminals support none of these.

You can't just pick one. You need to try them all, in order of preference, until something works.

## The Solution: Try Until Success

The core pattern is simple: maintain a priority-ordered list of methods, try each one, and stop when something succeeds. If nothing works, fall back to the simplest possible fallback.

Here's how tnotify implements this in `main.go`:

```go
func sendAny(message, title string, urgency int, id string) error {
    // Try OSC first
    terminal := detect.DetectTerminal()
    protocol := detect.SelectProtocol(terminal)

    if protocol != detect.ProtocolNone {
        return sendOSC(message, title, urgency, id)
    }

    // Try native
    if native.IsAvailable() {
        return sendNative(message, title, urgency)
    }

    // Fall back to bell
    fmt.Print("\x07")
    return nil
}
```

Three attempts. Three increasingly desperate options. The last one—the terminal bell—always works because it's been part of terminals since the 1970s.

## How It Works

### Step 1: Detect the Environment

Before attempting any notification method, tnotify detects which terminal is running. This happens in `internal/detect/terminal.go`:

```go
func DetectTerminal() Terminal {
    // Kitty
    if os.Getenv("KITTY_WINDOW_ID") != "" {
        return TerminalKitty
    }

    // iTerm2
    if os.Getenv("ITERM_SESSION_ID") != "" {
        return TerminalITerm2
    }

    // WezTerm
    if os.Getenv("WEZTERM_PANE") != "" {
        return TerminalWezTerm
    }

    // ... more terminal checks ...

    return TerminalUnknown
}
```

Each terminal emulator sets unique environment variables. Checking them is fast and reliable.

### Step 2: Select the Best Protocol

Once we know the terminal, we map it to the best OSC protocol it supports:

```go
func SelectProtocol(terminal Terminal) Protocol {
    switch terminal {
    case TerminalKitty:
        return ProtocolOSC99
    case TerminalITerm2:
        return ProtocolOSC9
    case TerminalWezTerm, TerminalGhostty, TerminalVTE, TerminalFoot:
        return ProtocolOSC777
    default:
        return ProtocolNone
    }
}
```

This returns `ProtocolNone` for terminals that don't support OSC notifications. That's the signal to move down the chain.

### Step 3: Try Native Notifications

If OSC fails (or isn't supported), tnotify falls back to native OS notifications. This is platform-specific code:

```go
func IsAvailable() bool {
    sender := GetSender()
    if sender == nil {
        return false
    }
    return sender.IsAvailable()
}
```

On macOS, this uses `osascript`. On Linux, `notify-send`. On Windows, PowerShell toast notifications. The implementation checks if the required command exists before claiming availability.

### Step 4: The Ultimate Fallback

If everything fails, tnotify sends a terminal bell:

```go
// Fall back to bell
fmt.Print("\x07")
return nil
```

This is the ASCII BEL character. Every terminal supports it. Most will beep, flash the titlebar, or bounce the dock icon. It's not sophisticated, but it never fails.

## Why This Pattern Works

### Fail Fast, Continue Trying

Each method in the chain either succeeds or returns control immediately. There's no retry logic, no timeouts, no waiting. If something doesn't work, move on.

The terminal detection in `DetectTerminal()` is just environment variable checks—no shell commands, no file I/O. If the variables aren't set, return `TerminalUnknown` and let the next step handle it.

### Explicit Priority

The order matters. OSC sequences are tried first because they work over SSH and inside tmux/screen. Native notifications are local-only, so they're second. The bell is last because it's the least useful but most reliable.

The priority is encoded in the function itself. You read `sendAny()` top to bottom and understand exactly what will be tried and in what order.

### No Hidden State

Each step is independent. The OSC attempt doesn't affect the native attempt. The native attempt doesn't affect the bell. There's no shared state, no error accumulation, no complex cleanup.

This makes the code easy to test. You can mock `DetectTerminal()` to return `TerminalUnknown`, verify that `sendOSC()` isn't called, and confirm it falls through to native.

### User Override

The fallback chain is the default, but users can force a specific method:

```go
if forceNative {
    return sendNative(message, title, urgencyLevel)
}

if forceOSC {
    return sendOSC(message, title, urgencyLevel, id)
}

// Auto mode: try OSC first, then native, then bell
return sendAny(message, title, urgencyLevel, id)
```

Users who know their environment can skip the detection and force the method they want. This is useful for debugging or when the detection is wrong.

## When to Use This Pattern

Not every tool needs a fallback chain. If you're building an internal tool and know your environment, pick one method and use it.

But if you're building something that runs in unknown environments—especially across different operating systems, terminals, or SSH sessions—the fallback chain pattern is worth considering.

Ask yourself:

- Do multiple methods exist for the same goal?
- Is the environment unpredictable or variable?
- Is partial functionality better than total failure?

If you answered yes to all three, build a fallback chain.

## Implementation Tips

### Detection Must Be Cheap

Environment variable checks are fast. Shell command execution is not. File I/O is not. Network calls are definitely not.

If your detection logic is expensive, users will notice the delay every time they run your tool. Keep it simple.

### Make Capability Queries Available

tnotify exposes `--capabilities` to show what's detected:

```bash
$ tnotify --capabilities
{
  "terminal": "ghostty",
  "protocol": "osc777",
  "supports_title": true,
  "supports_urgency": false,
  "supports_id": false,
  "supports_progress": true,
  "in_multiplexer": false,
  "native_available": true
}
```

This helps users understand why their notifications look different than expected, and it helps you debug issues when they report problems.

### Test Every Step Independently

Each method in the chain should have its own test. Don't just test the happy path where everything works. Test the case where only native notifications work. Test the case where only the bell works.

This is easier if each method is a separate function with a clear interface, like `sendOSC()` and `sendNative()` in tnotify.

### Document the Fallback Order

Users need to know what will happen. From the tnotify help text:

```
It automatically detects your terminal and uses the best available method:
- OSC 99 (Kitty) - Full featured: urgency, notification IDs
- OSC 777 (WezTerm, Ghostty, VTE) - Title + body
- OSC 9 (iTerm2) - Message only
- Native fallback (osascript, notify-send, PowerShell)
```

This sets expectations. Users know they might get different behavior in different environments, and they know why.

## Related Patterns

The fallback chain is related to the strategy pattern, but with automatic selection rather than explicit configuration. It's also similar to content negotiation in HTTP, where the server picks the best response format based on client capabilities.

In CLI tools specifically, you see this pattern in:

- **Git's editor selection**: `$GIT_EDITOR`, then `$VISUAL`, then `$EDITOR`, then `vi`
- **Package managers**: Try to auto-detect the OS package manager, fall back to manual installation
- **Terminal color detection**: Check `$COLORTERM`, then `$TERM`, then assume no color

The key is ordering your attempts from most-specific to most-general, and ensuring the last fallback always succeeds.

## The Bell is Not Optional

The most important line in the fallback chain is the last one:

```go
fmt.Print("\x07")
return nil
```

This is the contract with the user: I will always do *something*. It might not be fancy. It might not even be useful. But the command will never silently fail.

That's the difference between a tool that works everywhere and a tool that works nowhere.
