# Progress Bars in Terminal Tabs: OSC 9;4 Implementation

Modern terminals can display progress bars in their tab bar or taskbar icon. A long-running build shows 50% complete right in the tab title. When it fails, the tab turns red.

This is OSC 9;4, a relatively new protocol supported by Windows Terminal, Ghostty, iTerm2 (3.6.6+), ConEmu, and Mintty. Here's how to implement it.

## What It Looks Like

Instead of cluttering your terminal output with ASCII progress bars, OSC 9;4 shows progress in the UI chrome:

- **Windows Terminal**: Progress bar in the taskbar icon
- **Ghostty**: Progress indicator in the tab
- **iTerm2**: Progress bar in the tab

The user can switch to another tab and still see build progress at a glance.

## The Protocol

OSC 9;4 is an extension of OSC 9 (the iTerm2 notification protocol). Format:

```
ESC ] 9 ; 4 ; state ; progress BEL
\x1b]9;4;1;50\x07
```

Where:
- `state` is 0-4 (see below)
- `progress` is 0-100 (percentage)

### Progress States

| State | Meaning | Visual |
|-------|---------|--------|
| 0 | Hidden/Remove | Clears the progress bar |
| 1 | Normal | Green progress bar |
| 2 | Error | Red progress bar |
| 3 | Indeterminate | Pulsing/animated (no percentage) |
| 4 | Paused/Warning | Yellow progress bar |

## Implementation

```go
const (
    ProgressHidden       = 0
    ProgressNormal       = 1
    ProgressError        = 2
    ProgressIndeterminate = 3
    ProgressPaused       = 4
)

func BuildOSC9Progress(state, progress int) string {
    // Clamp progress to valid range
    if progress < 0 {
        progress = 0
    }
    if progress > 100 {
        progress = 100
    }

    // Clamp state to valid range
    if state < 0 || state > 4 {
        state = ProgressNormal
    }

    return fmt.Sprintf("\x1b]9;4;%d;%d\x07", state, progress)
}

func ClearProgress() string {
    return "\x1b]9;4;0;0\x07"
}
```

## Feature Detection

Not all terminals support OSC 9;4. Sending it to an unsupported terminal might:
- Display garbage characters
- Show the raw escape sequence in a notification
- Do nothing (best case)

We need feature detection:

```go
func SupportsProgress(terminal Terminal) bool {
    switch terminal {
    case TerminalWindowsTerminal, TerminalGhostty:
        return true
    case TerminalITerm2:
        // iTerm2 added support in 3.6.6
        version := os.Getenv("TERM_PROGRAM_VERSION")
        return compareVersion(version, "3.6.6") >= 0
    default:
        return false
    }
}
```

### iTerm2 Version Check

iTerm2 only added OSC 9;4 support in version 3.6.6 (December 2024). We need to check the version:

```go
func compareVersion(a, b string) int {
    // Strip suffixes like "3.6.6-beta"
    a = strings.Split(a, "-")[0]
    b = strings.Split(b, "-")[0]

    partsA := strings.Split(a, ".")
    partsB := strings.Split(b, ".")

    maxLen := len(partsA)
    if len(partsB) > maxLen {
        maxLen = len(partsB)
    }

    for i := 0; i < maxLen; i++ {
        var numA, numB int
        if i < len(partsA) {
            numA, _ = strconv.Atoi(partsA[i])
        }
        if i < len(partsB) {
            numB, _ = strconv.Atoi(partsB[i])
        }

        if numA < numB {
            return -1
        }
        if numA > numB {
            return 1
        }
    }
    return 0
}
```

## Graceful Fallback

When the terminal doesn't support progress bars, provide a text fallback:

```go
func ShowProgress(progress int) error {
    terminal := detect.DetectTerminal()

    if !detect.SupportsProgress(terminal) {
        // Plain text fallback
        fmt.Printf("Progress: %d%%\n", progress)
        return nil
    }

    // Send OSC 9;4 sequence
    sequence := osc.BuildOSC9Progress(osc.ProgressNormal, progress)
    return sendToTerminal(sequence)
}
```

## Real-World Usage Patterns

### Build Progress

```go
func runBuild() error {
    steps := []string{"compile", "test", "package"}

    for i, step := range steps {
        progress := (i * 100) / len(steps)
        ShowProgress(progress)

        if err := runStep(step); err != nil {
            // Show error state
            ShowProgressState(ProgressError, 100)
            return err
        }
    }

    // Complete
    ShowProgress(100)
    time.Sleep(time.Second) // Let user see 100%
    ClearProgress()
    return nil
}
```

### Download Progress

```go
func downloadWithProgress(url string) error {
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    total := resp.ContentLength
    var downloaded int64

    buf := make([]byte, 32*1024)
    for {
        n, err := resp.Body.Read(buf)
        if n > 0 {
            downloaded += int64(n)
            progress := int((downloaded * 100) / total)
            ShowProgress(progress)
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            ShowProgressState(ProgressError, int((downloaded*100)/total))
            return err
        }
    }

    ClearProgress()
    return nil
}
```

### Indeterminate Progress

For operations with unknown duration:

```go
func runUnknownDuration(ctx context.Context) error {
    // Start indeterminate animation
    sequence := BuildOSC9Progress(ProgressIndeterminate, 0)
    sendToTerminal(sequence)

    defer ClearProgress()

    // Do work...
    return doWork(ctx)
}
```

## CLI Integration

In tnotify, we expose progress via flags:

```bash
# Set progress
tnotify -p 50                          # 50% normal (green)
tnotify -p 100 --progress-state error  # 100% error (red)
tnotify -p 75 --progress-state paused  # 75% paused (yellow)

# Indeterminate progress (bouncing/pulsing for unknown duration)
tnotify --indeterminate                # Shorthand flag
tnotify -I                             # Even shorter
tnotify -p 0 --progress-state indeterminate  # Alternative syntax

# Clear progress
tnotify --progress-clear
```

The implementation:

```go
var (
    progress      int
    progressState string
    progressClear bool
    indeterminate bool
)

rootCmd.Flags().IntVarP(&progress, "progress", "p", -1,
    "Set progress bar (0-100)")
rootCmd.Flags().StringVar(&progressState, "progress-state", "normal",
    "Progress state: normal, error, paused, indeterminate")
rootCmd.Flags().BoolVar(&progressClear, "progress-clear", false,
    "Clear progress bar")
rootCmd.Flags().BoolVarP(&indeterminate, "indeterminate", "I", false,
    "Show indeterminate/bouncing progress indicator")

// In the run function:
if progressClear {
    terminal := detect.DetectTerminal()
    if detect.SupportsProgress(terminal) {
        fmt.Print(osc.ClearProgress())
    }
    return nil
}

// Handle --indeterminate flag (shorthand for indeterminate state)
if indeterminate {
    terminal := detect.DetectTerminal()
    if !detect.SupportsProgress(terminal) {
        fmt.Println("Progress: indeterminate")
        return nil
    }
    fmt.Print(osc.BuildOSC9Progress(osc.ProgressIndeterminate, 0))
    return nil
}

if progress >= 0 {
    terminal := detect.DetectTerminal()
    if !detect.SupportsProgress(terminal) {
        fmt.Printf("Progress: %d%%\n", progress)
        return nil
    }

    state := parseProgressState(progressState)
    fmt.Print(osc.BuildOSC9Progress(state, progress))
    return nil
}
```

## Testing

Test both the sequence generation and feature detection:

```go
func TestBuildOSC9Progress(t *testing.T) {
    tests := []struct {
        state    int
        progress int
        want     string
    }{
        {ProgressNormal, 50, "\x1b]9;4;1;50\x07"},
        {ProgressError, 100, "\x1b]9;4;2;100\x07"},
        {ProgressHidden, 0, "\x1b]9;4;0;0\x07"},
        // Edge cases
        {ProgressNormal, -10, "\x1b]9;4;1;0\x07"},   // Clamped to 0
        {ProgressNormal, 200, "\x1b]9;4;1;100\x07"}, // Clamped to 100
    }

    for _, tt := range tests {
        got := BuildOSC9Progress(tt.state, tt.progress)
        if got != tt.want {
            t.Errorf("BuildOSC9Progress(%d, %d) = %q, want %q",
                tt.state, tt.progress, got, tt.want)
        }
    }
}

func TestSupportsProgress(t *testing.T) {
    // Save and restore environment
    snap := saveEnv("TERM_PROGRAM_VERSION")
    defer snap.restore()

    t.Run("iTerm2 3.6.6 supports progress", func(t *testing.T) {
        os.Setenv("TERM_PROGRAM_VERSION", "3.6.6")
        if !SupportsProgress(TerminalITerm2) {
            t.Error("iTerm2 3.6.6 should support progress")
        }
    })

    t.Run("iTerm2 3.6.5 does not support progress", func(t *testing.T) {
        os.Setenv("TERM_PROGRAM_VERSION", "3.6.5")
        if SupportsProgress(TerminalITerm2) {
            t.Error("iTerm2 3.6.5 should not support progress")
        }
    })
}
```

## Multiplexer Considerations

OSC 9;4 needs passthrough in tmux, just like other OSC sequences:

```go
func ShowProgress(progress int) error {
    terminal := detect.DetectTerminal()

    if !detect.SupportsProgress(terminal) {
        fmt.Printf("Progress: %d%%\n", progress)
        return nil
    }

    sequence := osc.BuildOSC9Progress(osc.ProgressNormal, progress)

    // Wrap for multiplexers
    if detect.InTmux() {
        sequence = osc.WrapForTmux(sequence)
    } else if detect.InScreen() {
        sequence = osc.WrapForScreen(sequence)
    }

    return sendToTerminal(sequence)
}
```

## Conclusion

OSC 9;4 is a small feature that significantly improves UX for long-running operations. The key points:

1. **Feature detect first**: Don't send to unsupported terminals
2. **Graceful fallback**: Print text when OSC isn't available
3. **Version check for iTerm2**: Only 3.6.6+ supports it
4. **Clear when done**: Don't leave stale progress bars

The protocol is simple (one escape sequence), but proper implementation requires terminal detection and graceful degradation.

Full implementation: [tnotify's progress code](https://github.com/soloterm/tnotify/blob/main/internal/osc/progress.go)

## References

- [Windows Terminal Progress Bar](https://docs.microsoft.com/en-us/windows/terminal/tutorials/progress-bar-sequences)
- [ConEmu Progress Documentation](https://conemu.github.io/en/AnsiEscapeCodes.html#ConEmu_specific_OSC)
- [iTerm2 3.6.6 Release Notes](https://iterm2.com/downloads.html)
