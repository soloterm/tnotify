# Designing a Capabilities API for Feature Detection

Terminal emulators are a fragmented ecosystem. Kitty supports OSC 99 for notifications. iTerm2 uses OSC 9. WezTerm implements OSC 777. Some terminals support none of these. When you're building a CLI tool that needs to work across this landscape, you need a way to detect what features are available and adapt accordingly.

I built tnotify to send desktop notifications from the terminal using OSC escape sequences. The core challenge was making it work reliably across different terminals without requiring users to manually configure anything. The solution was a capabilities API that handles detection, protocol selection, and feature availability in one place.

## The Problem with Feature Detection

You could check environment variables and select a protocol inline every time you need to send a notification:

```go
// Don't do this
func sendNotification(message string) {
    if os.Getenv("KITTY_WINDOW_ID") != "" {
        fmt.Print(buildOSC99(message))
    } else if os.Getenv("ITERM_SESSION_ID") != "" {
        fmt.Print(buildOSC9(message))
    } else if os.Getenv("WEZTERM_PANE") != "" {
        fmt.Print(buildOSC777(message))
    }
    // ... repeat this everywhere
}
```

This gets messy fast. You duplicate detection logic across your codebase. Adding support for a new terminal means hunting down every location that does feature detection. Testing becomes difficult because the logic is scattered.

## A Structured Capabilities API

The tnotify capabilities API consolidates all feature detection into a single struct that describes what the current environment can do. Here's the implementation from `internal/detect/terminal.go`:

```go
// Capabilities describes terminal notification capabilities.
type Capabilities struct {
    Terminal         Terminal `json:"terminal"`
    Protocol         Protocol `json:"protocol"`
    SupportsTitle    bool     `json:"supports_title"`
    SupportsUrgency  bool     `json:"supports_urgency"`
    SupportsID       bool     `json:"supports_id"`
    SupportsProgress bool     `json:"supports_progress"`
    InMultiplexer    bool     `json:"in_multiplexer"`
    NativeAvailable  bool     `json:"native_available"`
}

// GetCapabilities returns the notification capabilities for the current environment.
func GetCapabilities(nativeAvailable bool) Capabilities {
    terminal := DetectTerminal()
    protocol := SelectProtocol(terminal)

    return Capabilities{
        Terminal:         terminal,
        Protocol:         protocol,
        SupportsTitle:    protocol == ProtocolOSC777 || protocol == ProtocolOSC99,
        SupportsUrgency:  protocol == ProtocolOSC99,
        SupportsID:       protocol == ProtocolOSC99,
        SupportsProgress: SupportsProgress(terminal),
        InMultiplexer:    InTmux() || InScreen(),
        NativeAvailable:  nativeAvailable,
    }
}
```

This design separates three concerns:

1. **What terminal are we running in?** (Kitty, iTerm2, WezTerm, etc.)
2. **What protocol should we use?** (OSC 9, OSC 777, OSC 99)
3. **What features does this combination support?** (title, urgency, IDs, progress bars)

The struct is JSON-serializable because tnotify exposes it as a debugging tool. Users can run `tnotify --capabilities` to see exactly what their terminal supports:

```json
{
  "terminal": "kitty",
  "protocol": "osc99",
  "supports_title": true,
  "supports_urgency": true,
  "supports_id": true,
  "supports_progress": false,
  "in_multiplexer": false,
  "native_available": true
}
```

## Protocol Selection as a Separate Step

Terminal detection and protocol selection are decoupled. `DetectTerminal()` returns which terminal is running. `SelectProtocol()` takes that terminal and returns the best protocol to use:

```go
// SelectProtocol returns the best OSC protocol for a terminal.
func SelectProtocol(terminal Terminal) Protocol {
    switch terminal {
    case TerminalKitty:
        return ProtocolOSC99
    case TerminalITerm2:
        return ProtocolOSC9
    case TerminalWezTerm, TerminalGhostty, TerminalVTE, TerminalFoot:
        return ProtocolOSC777
    case TerminalAlacritty, TerminalKonsole, TerminalAppleTerminal,
        TerminalVSCode, TerminalWindowsTerminal:
        return ProtocolNone
    default:
        return ProtocolNone
    }
}
```

This separation matters because the mapping between terminals and protocols isn't one-to-one. Multiple terminals can use the same protocol (WezTerm, Ghostty, and VTE all use OSC 777). A terminal might support multiple protocols but one is preferred. You might want to override protocol selection for testing without changing terminal detection.

## Feature Flags Derived from Protocol

Once you know the protocol, you can derive feature availability:

```go
SupportsTitle:    protocol == ProtocolOSC777 || protocol == ProtocolOSC99,
SupportsUrgency:  protocol == ProtocolOSC99,
SupportsID:       protocol == ProtocolOSC99,
```

OSC 9 (iTerm2) only supports a message field. OSC 777 adds title and body. OSC 99 (Kitty) adds urgency levels and notification IDs for updates. These are protocol characteristics, not terminal characteristics.

Progress bars are different. They use OSC 9;4, which is separate from the notification protocols. Terminal support varies:

```go
// SupportsProgress returns true if the terminal supports OSC 9;4 progress bars.
func SupportsProgress(terminal Terminal) bool {
    switch terminal {
    case TerminalWindowsTerminal, TerminalGhostty:
        return true
    case TerminalITerm2:
        // iTerm2 added OSC 9;4 support in version 3.6.6
        return compareVersion(os.Getenv("TERM_PROGRAM_VERSION"), "3.6.6") >= 0
    default:
        return false
    }
}
```

This requires version checking for iTerm2, where the feature was added in a specific release. Windows Terminal and Ghostty support it unconditionally.

## Using Capabilities in Application Code

In `main.go`, the `showCapabilities` function demonstrates how simple it is to use this API:

```go
func showCapabilities() error {
    caps := detect.GetCapabilities(native.IsAvailable())

    data, err := json.MarshalIndent(caps, "", "  ")
    if err != nil {
        return fmt.Errorf("marshaling capabilities: %w", err)
    }

    fmt.Println(string(data))
    return nil
}
```

One function call gives you complete information about the runtime environment. No scattered checks. No duplicated logic.

The main notification sending logic uses capabilities to decide what to do:

```go
func sendOSC(message, title string, urgency int, id string) error {
    terminal := detect.DetectTerminal()
    protocol := detect.SelectProtocol(terminal)

    if protocol == detect.ProtocolNone {
        return fmt.Errorf("no OSC notification support detected for terminal: %s", terminal)
    }

    var sequence string
    switch protocol {
    case detect.ProtocolOSC9:
        sequence = osc.BuildOSC9(message)
    case detect.ProtocolOSC777:
        sequence = osc.BuildOSC777(message, title)
    case detect.ProtocolOSC99:
        sequence = osc.BuildOSC99(message, title, urgency, id)
    }

    sequence = osc.WrapForMultiplexer(sequence)
    fmt.Print(sequence)
    return nil
}
```

The code checks capabilities once, makes a decision, and executes. No nested conditionals checking environment variables.

## Why Not Just Use Interfaces?

You might think "this should be an interface with different implementations per terminal." That works for simple cases but breaks down when features don't align neatly with terminal boundaries.

Multiple terminals use identical protocols. An interface would duplicate code across WezTerm, Ghostty, and VTE implementations even though they all use OSC 777 identically.

Features need to be queryable before you act. With interfaces, you'd need methods like `SupportsTitle()` on each implementation. With a capabilities struct, you get all the information upfront in a form you can pass around, serialize, or display to users.

Capabilities are environment properties, not behavior. You're not polymorphically calling different implementations. You're adapting single implementations based on available features.

## Testing Benefits

The capabilities struct makes testing straightforward. You can construct any capability combination without mocking environment variables:

```go
func TestNotificationWithLimitedCapabilities(t *testing.T) {
    caps := Capabilities{
        Terminal:        TerminalITerm2,
        Protocol:        ProtocolOSC9,
        SupportsTitle:   false,
        SupportsUrgency: false,
        SupportsID:      false,
    }

    // Test how code handles OSC 9 limitations
}
```

This is cleaner than setting up environment variables and hoping detection works correctly in tests.

## Where This Pattern Applies

This pattern works well when:

- You have multiple backends with overlapping but not identical features
- Feature availability depends on runtime detection, not compile-time configuration
- You need to explain to users what's available (diagnostics, debugging)
- Features are independent toggles rather than deeply coupled behaviors

I've used variations of this pattern for:

- Database feature detection (does this PostgreSQL version support `ON CONFLICT`?)
- Browser capability detection in web apps
- Cloud provider API feature flags

The core idea is the same: detect once, decide once, then execute with that information.

## Making It Discoverable

The `--capabilities` flag turns this from internal plumbing into a user-facing debugging tool. When someone reports "notifications aren't working," you can ask them to run `tnotify --capabilities` and paste the output. You immediately know their terminal, protocol, and available features.

This kind of introspection is rare in CLI tools but costs almost nothing to implement. You're already computing capabilities internally. Just add a flag that prints them.

The diagnostic mode in tnotify goes further, actually testing each capability and reporting results. But the capabilities struct makes this possible by giving you a list of what to test.

## Extensions and Variations

As tnotify adds features, the capabilities struct grows. When Ghostty 1.2 added progress bar support, I added version detection to `SupportsProgress()`. The change was isolated to one function. Call sites didn't change.

You could extend this pattern with capability negotiation. Query the terminal with escape sequences to discover features instead of relying on environment variables. The struct shape stays the same; only `GetCapabilities()` changes.

You could add confidence levels: "detected Kitty with 99% confidence" vs "unknown terminal, assuming no features." The struct would gain a `Confidence` field.

You could cache capabilities globally after first detection if environment can't change during execution. This turns repeated capability checks into struct field accesses.

## The Core Insight

Feature detection is data, not behavior. Model it as data. Compute it once. Use it everywhere.

The capabilities struct in tnotify is 27 lines of code. It eliminates scattered conditionals across the codebase, enables comprehensive testing, and provides users with debugging information. That's a good return on investment for such a simple pattern.
