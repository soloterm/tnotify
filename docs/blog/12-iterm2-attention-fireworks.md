# iTerm2 Proprietary Extensions: RequestAttention and Fireworks

iTerm2 extends the terminal emulator protocol with dozens of proprietary escape sequences under the OSC 1337 namespace. Two of these—`RequestAttention` and the fireworks variant—let you programmatically bounce the dock icon on macOS. One is practical. The other is... fireworks.

## The Problem

You start a long-running build or test suite and switch to your browser. Twenty minutes later, you've forgotten about it. Standard desktop notifications work, but they're silent when your terminal window is focused. What you need is something that demands attention regardless of window state.

macOS already has a mechanism for this: bouncing the dock icon. Applications can request user attention by bouncing their icon in the dock. iTerm2 exposes this through escape sequences.

## The Implementation

The code in `internal/osc/iterm.go` shows how simple these sequences are:

```go
// BuildITermRequestAttention builds an iTerm2 OSC 1337 RequestAttention sequence.
// This bounces the dock icon on macOS to get the user's attention.
// fireworks: if true, shows a brief fireworks animation.
func BuildITermRequestAttention(fireworks bool) string {
    value := "yes"
    if fireworks {
        value = "fireworks"
    }
    return "\x1b]1337;RequestAttention=" + value + "\x07"
}
```

The sequence format breaks down as:
- `\x1b]` - ESC followed by `]` starts an OSC (Operating System Command)
- `1337;RequestAttention=yes` - iTerm2's proprietary namespace with the parameter
- `\x07` - BEL terminates the sequence

The normal variant (`yes`) bounces the dock icon once. The fireworks variant (`fireworks`) triggers a brief fireworks animation before bouncing.

## The Escape Sequence Protocol

OSC sequences follow this structure: `ESC ] Ps ; Pt BEL` where `Ps` is a numeric parameter and `Pt` is the text payload. iTerm2 claimed `1337` as their namespace to avoid conflicts with standard sequences.

Standard OSC codes include things like `OSC 0` for setting the window title or `OSC 52` for clipboard operations. iTerm2's `1337` namespace includes file transfers, inline images, badges, and these attention-requesting sequences.

The terminator can be either BEL (`\x07`) or ST (`\x1b\\`). Most implementations accept both, but iTerm2's documentation uses BEL consistently.

## Usage in tnotify

The tnotify CLI tool uses these sequences through simple command-line flags:

```bash
# Bounce the dock icon once
tnotify --attention

# Bounce with fireworks
tnotify --fireworks
```

The main.go shows the integration:

```go
// Handle --attention or --fireworks
if attention || fireworks {
    sequence := osc.BuildITermRequestAttention(fireworks)
    sequence = osc.WrapForMultiplexer(sequence)
    fmt.Print(sequence)
    return nil
}
```

The `WrapForMultiplexer` call is critical. When you're inside tmux or GNU Screen, bare OSC sequences get swallowed by the multiplexer. You need to wrap them in a passthrough sequence so they reach the outer terminal:

```go
// For tmux: \ePtmux;\e + sequence + \e\\
// For screen: \eP + sequence + \e\\
```

## The Fireworks Mystery

The fireworks animation is documented but unexplained. Looking at the README:

```markdown
I have no idea why you'd use this, but here we are.
```

iTerm2's maintainers added this feature and left it in the API. It's a brief particle animation that plays before the bounce. Practically useless, technically delightful.

The test coverage treats both variants equally:

```go
tests := []struct {
    name      string
    fireworks bool
    want      string
}{
    {"normal attention", false, "\x1b]1337;RequestAttention=yes\x07"},
    {"fireworks attention", true, "\x1b]1337;RequestAttention=fireworks\x07"},
}
```

## StealFocus: The Nuclear Option

iTerm2 also provides `StealFocus`, which brings the terminal window to the front immediately:

```go
func BuildITermStealFocus() string {
    return "\x1b]1337;StealFocus\x07"
}
```

This is more aggressive than `RequestAttention`. It doesn't just bounce the dock icon—it yanks focus from whatever you're doing. Use sparingly.

## Terminal Compatibility

These sequences only work in iTerm2. Other terminals ignore them silently, which is the correct behavior for unknown OSC sequences. The tnotify tool detects the terminal before sending:

```go
terminal := detect.DetectTerminal()
protocol := detect.SelectProtocol(terminal)

if protocol == detect.ProtocolOSC9 {
    // iTerm2 detected
    sequence := osc.BuildITermRequestAttention(fireworks)
    fmt.Print(sequence)
}
```

The detection looks at environment variables like `TERM_PROGRAM` to identify iTerm2 specifically.

## Practical Applications

Real use cases for `RequestAttention`:

**Long-running CI jobs**: Notify when builds complete without polling browser tabs.

**Background processes**: Signal when background tasks need intervention.

**Deployment scripts**: Alert when deployments finish or require approval.

**Test suites**: Flag test failures that need immediate attention.

Example integration:

```bash
# Run tests and notify on failure
./run-tests.sh
if [ $? -ne 0 ]; then
    tnotify --attention "Tests failed"
fi

# Long build with celebration
make release && tnotify --fireworks "Build succeeded!"
```

## The OSC 1337 Ecosystem

iTerm2's proprietary extensions include dozens of other sequences beyond attention requests:

- Inline images (`File=...`)
- Download/upload triggers
- Background image setting
- Cursor shape control
- Badge text (persistent overlay text)
- User variables
- Shell integration hooks

The `RequestAttention` sequences are among the simplest, requiring just a single parameter. Compare to inline images, which require base64-encoded image data and dimension parameters.

## Why Proprietary Extensions Matter

Terminal emulators evolve slowly. Standard escape sequences date back to VT100 terminals from 1978. When modern terminals want new features, they have two options:

1. Propose standards through slower consensus processes
2. Ship proprietary extensions immediately

iTerm2 chose option 2, using the `1337` namespace to avoid conflicts. Other terminals did the same: Kitty uses OSC 99, WezTerm uses OSC 777. Eventually, some extensions become de facto standards through adoption.

The `RequestAttention` feature fills a real gap—there's no standard way to request dock attention from terminal applications. Until a standard emerges, proprietary extensions serve real needs.

## Testing Without iTerm2

The test suite validates sequence generation without requiring iTerm2:

```go
func TestBuildITermRequestAttention(t *testing.T) {
    got := BuildITermRequestAttention(true)
    want := "\x1b]1337;RequestAttention=fireworks\x07"
    if got != want {
        t.Errorf("got %q, want %q", got, want)
    }
}
```

This approach tests the string building logic independently from terminal rendering. Integration tests would require automation tools that can detect dock icon bounces—difficult to do reliably in CI.

## Further Reading

The tnotify project implements multiple notification protocols beyond iTerm2:

- **OSC 99** (Kitty): Full-featured notifications with urgency levels and IDs
- **OSC 777** (WezTerm, Ghostty, VTE): Title and body notifications
- **OSC 9** (iTerm2, Windows Terminal): Basic notifications and progress bars

Each terminal has different capabilities. The detection logic selects the best available protocol for the current environment.

For iTerm2's complete escape sequence documentation, see their [proprietary sequences documentation](https://iterm2.com/documentation-escape-codes.html). The RequestAttention sequence is documented under "Proprietary Escape Codes."
