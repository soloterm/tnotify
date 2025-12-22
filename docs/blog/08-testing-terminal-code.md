# Testing Terminal-Dependent Code: Environment Variable Mocking in Go

Code that detects terminals reads environment variables. Tests need to control those variables without affecting other tests or the system. Here's a clean pattern for save/clear/restore that works with Go's testing package.

## The Problem

Terminal detection relies on environment variables:

```go
func DetectTerminal() Terminal {
    if os.Getenv("KITTY_WINDOW_ID") != "" {
        return TerminalKitty
    }
    if os.Getenv("ITERM_SESSION_ID") != "" {
        return TerminalITerm2
    }
    // ...
}
```

When testing, you need to:
1. Clear all terminal variables (start from known state)
2. Set specific variables for each test case
3. Restore the original environment after tests

If you don't restore, other tests might fail—or worse, the test results depend on what terminal you're running in.

## The Pattern

### 1. Define Which Variables to Track

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
    "TERM_PROGRAM_VERSION",
    "VTE_VERSION",
}
```

### 2. Create Snapshot Helpers

```go
// envSnapshot saves environment variable values for later restoration
type envSnapshot struct {
    vars map[string]string
}

// saveEnv captures the current values of the specified environment variables
func saveEnv(keys ...string) *envSnapshot {
    s := &envSnapshot{vars: make(map[string]string)}
    for _, k := range keys {
        if v, ok := os.LookupEnv(k); ok {
            s.vars[k] = v
        }
        // Note: if variable is unset, we don't add it to the map
        // restore() will not set it, leaving it unset (correct behavior)
    }
    return s
}

// restore sets all saved variables back to their original values
func (s *envSnapshot) restore() {
    // First, clear any variables that might have been set during the test
    // but weren't in the original environment
    for _, k := range terminalEnvVars {
        if _, exists := s.vars[k]; !exists {
            os.Unsetenv(k)
        }
    }

    // Then restore original values
    for k, v := range s.vars {
        os.Setenv(k, v)
    }
}

// clearEnv unsets all specified environment variables
func clearEnv(keys ...string) {
    for _, k := range keys {
        os.Unsetenv(k)
    }
}
```

### 3. Use in Tests

```go
func TestDetectTerminal(t *testing.T) {
    // Save original environment at test start
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
        {
            name: "iTerm2 via session ID",
            envSetup: func() {
                clearEnv(terminalEnvVars...)
                os.Setenv("ITERM_SESSION_ID", "w0t0p0:12345")
            },
            want: TerminalITerm2,
        },
        {
            name: "Unknown terminal",
            envSetup: func() {
                clearEnv(terminalEnvVars...)
                // Set nothing - should detect as unknown
            },
            want: TerminalUnknown,
        },
        {
            name: "Kitty takes precedence over TERM_PROGRAM",
            envSetup: func() {
                clearEnv(terminalEnvVars...)
                os.Setenv("KITTY_WINDOW_ID", "1")
                os.Setenv("TERM_PROGRAM", "WezTerm") // Should be ignored
            },
            want: TerminalKitty,
        },
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

## Why This Works

The pattern handles several tricky cases:

### Originally Unset Variables

If `KITTY_WINDOW_ID` wasn't set when tests started, the snapshot won't contain it. On restore, it gets `Unsetenv`ed—not set to empty string.

```go
// LookupEnv distinguishes "empty" from "unset"
if v, ok := os.LookupEnv(k); ok {
    s.vars[k] = v  // Only save if it exists
}
```

### Test Isolation

Each test case starts with `clearEnv()`, ensuring no variable leaks between cases:

```go
envSetup: func() {
    clearEnv(terminalEnvVars...)  // Always start clean
    os.Setenv("SPECIFIC_VAR", "value")
},
```

### Parallel Test Safety

Environment variables are process-global. If you run tests in parallel (`t.Parallel()`), they'll interfere with each other.

**Solution**: Don't use `t.Parallel()` for environment-dependent tests. Keep them sequential.

```go
func TestDetectTerminal(t *testing.T) {
    // Don't call t.Parallel() here!
    snap := saveEnv(terminalEnvVars...)
    defer snap.restore()
    // ...
}
```

## Testing Version-Dependent Features

For features that depend on version strings:

```go
func TestSupportsProgress(t *testing.T) {
    snap := saveEnv(terminalEnvVars...)
    defer snap.restore()

    t.Run("iTerm2 3.6.6+ supports progress", func(t *testing.T) {
        clearEnv(terminalEnvVars...)
        os.Setenv("TERM_PROGRAM_VERSION", "3.6.6")

        if !SupportsProgress(TerminalITerm2) {
            t.Error("iTerm2 3.6.6 should support progress")
        }
    })

    t.Run("iTerm2 3.6.5 does not support progress", func(t *testing.T) {
        clearEnv(terminalEnvVars...)
        os.Setenv("TERM_PROGRAM_VERSION", "3.6.5")

        if SupportsProgress(TerminalITerm2) {
            t.Error("iTerm2 3.6.5 should not support progress")
        }
    })

    t.Run("iTerm2 without version does not support progress", func(t *testing.T) {
        clearEnv(terminalEnvVars...)
        // Don't set TERM_PROGRAM_VERSION

        if SupportsProgress(TerminalITerm2) {
            t.Error("iTerm2 without version should not support progress")
        }
    })
}
```

## Testing Output That Changes Based on Terminal

For testing CLI output that varies by terminal:

```go
func TestProgressFlagSupported(t *testing.T) {
    // Save and set up Ghostty environment
    snap := saveEnv(terminalEnvVars...)
    defer snap.restore()

    clearEnv(terminalEnvVars...)
    os.Setenv("GHOSTTY_RESOURCES_DIR", "/tmp/ghostty")

    // Capture stdout
    var buf bytes.Buffer
    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    // Run the command
    cmd := NewRootCmd()
    cmd.SetArgs([]string{"-p", "50"})
    err := cmd.Execute()

    // Restore stdout and read output
    w.Close()
    os.Stdout = oldStdout
    buf.ReadFrom(r)

    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    output := buf.String()
    // Should contain OSC sequence, not plain text
    if !strings.Contains(output, "\x1b]9;4;") {
        t.Errorf("expected OSC 9;4 sequence, got: %q", output)
    }
}

func TestProgressFlagUnsupported(t *testing.T) {
    snap := saveEnv(terminalEnvVars...)
    defer snap.restore()

    clearEnv(terminalEnvVars...)
    // No terminal env vars = unsupported

    var buf bytes.Buffer
    // ... capture stdout ...

    cmd := NewRootCmd()
    cmd.SetArgs([]string{"-p", "50"})
    cmd.Execute()

    output := buf.String()
    // Should print plain text fallback
    if output != "Progress: 50%\n" {
        t.Errorf("expected plain text, got: %q", output)
    }
}
```

## CI Environment Considerations

CI environments (GitHub Actions, GitLab CI) typically have no terminal variables set. This is actually ideal for testing—you get the "unknown terminal" case by default.

But if you're testing in a real terminal (local development), you might have `TERM_PROGRAM` or similar set. The snapshot/restore pattern handles this:

```bash
# Run tests locally in iTerm2
$ echo $TERM_PROGRAM
iTerm.app

$ go test ./...
# Tests pass because snapshot/restore works
```

## Alternative: Dependency Injection

For more complex cases, inject the environment reader:

```go
type EnvReader interface {
    Getenv(key string) string
}

type OSEnvReader struct{}

func (OSEnvReader) Getenv(key string) string {
    return os.Getenv(key)
}

type MockEnvReader struct {
    Vars map[string]string
}

func (m MockEnvReader) Getenv(key string) string {
    return m.Vars[key]
}

func DetectTerminalWith(env EnvReader) Terminal {
    if env.Getenv("KITTY_WINDOW_ID") != "" {
        return TerminalKitty
    }
    // ...
}
```

This avoids modifying global state but requires threading the dependency through your code.

## Conclusion

The snapshot/restore pattern for environment variables is simple and effective:

1. **Save** before tests with `saveEnv()`
2. **Clear** at the start of each test case with `clearEnv()`
3. **Set** the specific variables you're testing
4. **Restore** after tests with `defer snap.restore()`

Don't run environment-dependent tests in parallel. The test isolation is worth the small performance cost of sequential execution.

Full implementation: [tnotify's terminal_test.go](https://github.com/soloterm/tnotify/blob/main/internal/detect/terminal_test.go)
