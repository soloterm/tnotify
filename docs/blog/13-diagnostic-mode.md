# Building a Diagnostic Mode for CLI Tools

CLI tools that interact with platform-specific features often fail silently. A user types a command, nothing happens, and they're left guessing. Is the terminal incompatible? Is a dependency missing? Are permissions wrong?

tnotify sends desktop notifications through terminal escape sequences. Different terminals support different protocols (OSC 9, OSC 777, OSC 99), some systems have native notification tools, and multiplexers like tmux require special configuration. Instead of making users debug this themselves, I built a `--diagnose` flag that tests everything and shows what works.

## The Problem with Silent Failures

Here's what makes terminal notifications tricky:

- **Protocol fragmentation**: Kitty speaks OSC 99, iTerm2 speaks OSC 9, WezTerm speaks OSC 777
- **Focus suppression**: Most terminals don't show notifications when they're focused
- **Multiplexer wrapping**: tmux needs `allow-passthrough on` to forward escape sequences
- **Native fallbacks**: macOS has `osascript`, Linux has `notify-send`, but they might not be installed

When `tnotify "Build complete!"` does nothing, users need to know *why*.

## Anatomy of a Diagnostic Command

The `runDiagnose` function in `main.go` follows a simple pattern: detect capabilities, warn about common issues, test each method, report results.

### Step 1: Detect the Environment

```go
func runDiagnose() error {
    terminal := detect.DetectTerminal()
    protocol := detect.SelectProtocol(terminal)
    inMux := detect.InMultiplexer()
    nativeAvail := native.IsAvailable()
```

The function starts by checking what's available. `DetectTerminal()` looks at environment variables like `TERM_PROGRAM` and `TERM` to identify the terminal. `SelectProtocol()` maps that to the appropriate OSC protocol. `InMultiplexer()` checks for tmux or screen. `IsAvailable()` verifies native notification commands exist.

### Step 2: Print Environment Details with Warnings

```go
fmt.Println("=== tnotify diagnostic ===")
fmt.Println()
fmt.Printf("Detected terminal: %s\n", terminalName(terminal))
fmt.Printf("Selected protocol: %s\n", protocolName(protocol))
if inMux {
    if detect.InTmux() {
        version := detect.TmuxVersion()
        passthrough := detect.TmuxAllowPassthrough()
        supportsPassthrough := detect.TmuxSupportsPassthrough()

        fmt.Printf("Multiplexer: %s\n", version)
        if !supportsPassthrough {
            fmt.Println("  WARNING: tmux < 3.2 does not support allow-passthrough")
        } else if passthrough == "" || passthrough == "off" {
            fmt.Println("  WARNING: allow-passthrough is off")
            fmt.Println("  Enable with: tmux set -g allow-passthrough on")
        } else {
            fmt.Printf("  allow-passthrough: %s\n", passthrough)
        }
    }
}
```

This section doesn't just dump data—it provides actionable feedback. If tmux is detected but `allow-passthrough` is disabled, it tells the user exactly how to fix it. This saves them from searching documentation.

### Step 3: Warn About Focus Suppression

```go
fmt.Println("IMPORTANT: Many terminals suppress notifications when focused.")
fmt.Println("Please switch to another window NOW.")
fmt.Println()

fmt.Print("Testing in: ")
for i := 3; i > 0; i-- {
    fmt.Printf("%d... ", i)
    time.Sleep(1 * time.Second)
}
fmt.Println("Go!")
```

This countdown gives users time to switch windows. Without it, every test would fail because the terminal is focused. Small detail, huge impact.

### Step 4: Test Each Method

```go
results := []struct {
    name   string
    result string
}{}

// Test 1: OSC (if available)
if protocol != detect.ProtocolNone {
    fmt.Printf("Testing %s... ", protocolName(protocol))
    var sequence string
    switch protocol {
    case detect.ProtocolOSC9:
        sequence = osc.BuildOSC9("tnotify test: OSC 9")
    case detect.ProtocolOSC777:
        sequence = osc.BuildOSC777("tnotify test: OSC 777", "tnotify")
    case detect.ProtocolOSC99:
        sequence = osc.BuildOSC99("tnotify test: OSC 99", "tnotify", osc.UrgencyNormal, "")
    }
    sequence = osc.WrapForMultiplexer(sequence)
    fmt.Print(sequence)
    fmt.Println("sent")
    results = append(results, struct{ name, result string }{protocolName(protocol), "sent"})
    time.Sleep(3 * time.Second)
} else {
    fmt.Println("Skipping OSC: no protocol detected for this terminal")
    results = append(results, struct{ name, result string }{"OSC", "skipped (unsupported terminal)"})
}
```

Each test sends a real notification with a descriptive message. The three-second sleep prevents tests from overlapping. If OSC isn't supported, it skips the test and records why.

The native test works the same way:

```go
if nativeAvail {
    fmt.Print("Testing native notifications... ")
    if native.Send("tnotify test: native notification", "tnotify", osc.UrgencyNormal) {
        fmt.Println("sent")
        results = append(results, struct{ name, result string }{"Native", "sent"})
    } else {
        fmt.Println("failed")
        results = append(results, struct{ name, result string }{"Native", "failed"})
    }
    time.Sleep(3 * time.Second)
}
```

### Step 5: Test Interactive Features

Progress bars need visual feedback. The diagnostic animates a progress bar from 0% to 100%:

```go
if detect.SupportsProgress(terminal) {
    fmt.Println("Testing progress bar (OSC 9;4)... watch the top of the terminal window")
    for i := 0; i <= 100; i += 5 {
        sequence := osc.BuildOSC9Progress(osc.ProgressNormal, i)
        sequence = osc.WrapForMultiplexer(sequence)
        fmt.Print(sequence)
        fmt.Printf("\r  Progress: %3d%%", i)
        time.Sleep(200 * time.Millisecond)
    }
    fmt.Println(" done")
    time.Sleep(1 * time.Second)

    clearSeq := osc.BuildOSC9ProgressClear()
    clearSeq = osc.WrapForMultiplexer(clearSeq)
    fmt.Print(clearSeq)
    fmt.Println("  Progress bar cleared")
    results = append(results, struct{ name, result string }{"Progress", "sent"})
}
```

Users see a progress bar animate in the terminal window chrome, confirming that their terminal supports it. The 200ms delay makes the animation visible.

### Step 6: Summarize Results

```go
fmt.Println()
fmt.Println("=== Summary ===")
fmt.Println()
for _, r := range results {
    status := "?"
    if r.result == "sent" {
        status = "✓"
    } else if r.result == "failed" || strings.HasPrefix(r.result, "skipped") {
        status = "✗"
    }
    fmt.Printf("  %s %s: %s\n", status, r.name, r.result)
}
```

The summary uses checkmarks and X's for quick scanning. Sample output:

```
=== Summary ===

  ✓ OSC 777 (WezTerm/Ghostty/VTE): sent
  ✗ Native: skipped (unavailable)
  ✓ Progress: sent
  ✓ Bell: sent
```

### Step 7: Provide Next Steps

```go
fmt.Println()
fmt.Println("Did you see/hear the notifications?")
fmt.Println("If not, your terminal may not support them or may suppress")
fmt.Println("notifications when focused. Try running a command like:")
fmt.Println()
fmt.Println("  sleep 3 && tnotify 'Hello!'")
fmt.Println()
fmt.Println("and switch to another window during the countdown.")
```

Even if tests report success, the notification might not appear due to OS-level notification settings. The diagnostic suggests a manual test users can run to verify end-to-end functionality.

## Design Decisions

### Why Not Automate Everything?

The diagnostic can't detect notification permissions or Focus Mode settings. Those require the user to actually see the notification. The countdown and manual test suggestion acknowledge this limitation.

### Why Sleep Between Tests?

Notifications need time to appear and clear. Without delays, users see a flash of notifications or miss them entirely. Three seconds per test feels slow but ensures reliability.

### Why Include Skipped Tests in Results?

Showing skipped tests clarifies what the tool *can't* do. A user running this on Linux sees "Native: skipped (unavailable)" and knows to install `notify-send` if they want that fallback.

### Why Test the Bell?

The bell is the ultimate fallback. If every other method fails, the tool can still beep. Testing it confirms the terminal isn't completely broken.

## Patterns to Reuse

1. **Detect, don't assume**: Use detection functions to check capabilities before testing
2. **Warn about common issues**: If tmux passthrough is off, tell the user how to fix it
3. **Give users time**: Add countdowns before tests that require focus changes
4. **Test the actual feature**: Send real notifications, not mocked versions
5. **Separate "sent" from "worked"**: The diagnostic can't verify OS-level settings, so it suggests manual verification
6. **Summarize visually**: Checkmarks and X's communicate faster than text
7. **Provide next steps**: End with actionable advice, not just test results

## Adding Diagnostics to Your CLI

If your tool interacts with platform-specific features (notifications, clipboard, GPUs, audio), a diagnostic command helps users help themselves.

Start with detection:

```go
func runDiagnose() error {
    // What do we need to check?
    hasFFmpeg := checkFFmpeg()
    hasGPU := checkGPU()
    hasCodec := checkCodec()

    // Print what we found
    fmt.Printf("FFmpeg: %s\n", formatAvailability(hasFFmpeg))
    fmt.Printf("GPU acceleration: %s\n", formatAvailability(hasGPU))
    fmt.Printf("H.264 codec: %s\n", formatAvailability(hasCodec))
}
```

Add tests that exercise each component:

```go
// Test encoding a small video
if hasFFmpeg && hasCodec {
    fmt.Print("Testing video encoding... ")
    if err := testEncode(); err != nil {
        fmt.Printf("failed: %v\n", err)
    } else {
        fmt.Println("success")
    }
}
```

Provide specific error messages:

```go
if !hasFFmpeg {
    fmt.Println("FFmpeg not found in PATH")
    fmt.Println("Install with: brew install ffmpeg")
}
```

The diagnostic doesn't fix problems—it surfaces them clearly so users can fix them.

## When Diagnostics Aren't Enough

Some issues require interactive debugging. If detection fails, add verbose logging:

```go
var verbose bool
rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

func detectTerminal() string {
    term := os.Getenv("TERM_PROGRAM")
    if verbose {
        fmt.Fprintf(os.Stderr, "TERM_PROGRAM=%s\n", term)
    }
    return term
}
```

Users can run `myapp --verbose --diagnose` to see every detection step.

## Further Reading

For other examples of diagnostic commands:

- `gh auth status` (GitHub CLI) checks authentication state
- `docker info` shows Docker daemon configuration
- `kubectl cluster-info` diagnoses Kubernetes connectivity

Each follows the same pattern: detect, test, report, suggest.
