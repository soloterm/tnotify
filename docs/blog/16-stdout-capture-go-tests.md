# Stdout Capture Patterns in Go Tests

Testing CLI tools in Go presents a unique challenge: you need to verify what the program writes to stdout. The standard library doesn't make this easy. You can't just redirect `fmt.Print()` calls with a simple flag.

The tnotify project uses three different patterns for capturing stdout in tests. Each serves a different purpose.

## The Problem: fmt.Print() Writes to os.Stdout

When you call `fmt.Print("hello")`, it writes directly to `os.Stdout`, which points to file descriptor 1. Cobra's `cmd.SetOut()` only redirects output that explicitly uses the command's output writers. If your code calls `fmt.Print()` directly, `SetOut()` won't catch it.

Here's an example from `main_test.go`:

```go
func TestCommandHelp(t *testing.T) {
    cmd := newTestCommand()
    buf := new(bytes.Buffer)
    cmd.SetOut(buf)
    cmd.SetErr(buf)
    cmd.SetArgs([]string{"--help"})

    err := cmd.Execute()
    if err != nil {
        t.Fatalf("--help returned error: %v", err)
    }

    output := buf.String()
    if !strings.Contains(output, "tnotify") {
        t.Error("Help output should contain 'tnotify'")
    }
    if !strings.Contains(output, "--title") {
        t.Error("Help output should contain '--title' flag")
    }
}
```

This works because Cobra's help text uses `cmd.Out()` internally. The buffer captures everything. But what if your command writes directly to stdout?

## Pattern 1: OS Pipe Redirection

For commands that write directly to `os.Stdout`, you need to swap the file descriptor. The pattern uses `os.Pipe()` to create a read/write pair, then temporarily replaces `os.Stdout` with the write end.

From `main_test.go`:

```go
func TestCommandBell(t *testing.T) {
    cmd := newTestCommand()
    buf := new(bytes.Buffer)
    cmd.SetOut(buf)
    cmd.SetErr(buf)
    cmd.SetArgs([]string{"--bell"})

    // Capture stdout
    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    err := cmd.Execute()

    w.Close()
    os.Stdout = oldStdout

    if err != nil {
        t.Fatalf("--bell returned error: %v", err)
    }

    var stdout bytes.Buffer
    stdout.ReadFrom(r)

    if stdout.String() != "\x07" {
        t.Errorf("--bell should output bell character, got: %q", stdout.String())
    }
}
```

The mechanics:

1. Save the original `os.Stdout` pointer
2. Create a pipe with `os.Pipe()` - returns a reader and writer
3. Replace `os.Stdout` with the write end of the pipe
4. Run the command (it writes to the pipe)
5. Close the writer (signals EOF to the reader)
6. Restore the original `os.Stdout`
7. Read all data from the pipe into a buffer

The `--bell` flag triggers this code in `main.go`:

```go
// Handle --bell
if forceBell {
    fmt.Print("\x07")
    return nil
}
```

That `fmt.Print()` writes directly to `os.Stdout`. Without the pipe swap, there's no way to capture it in tests.

## Pattern 2: Parallel Setup

The same pattern appears in several tests with slight variations. Here's how it looks when testing version output:

```go
func TestCommandVersion(t *testing.T) {
    cmd := newTestCommand()
    buf := new(bytes.Buffer)
    cmd.SetOut(buf)
    cmd.SetErr(buf)
    cmd.SetArgs([]string{"--version"})

    // Capture stdout
    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    err := cmd.Execute()

    w.Close()
    os.Stdout = oldStdout

    if err != nil {
        t.Fatalf("--version returned error: %v", err)
    }

    var stdout bytes.Buffer
    stdout.ReadFrom(r)
    output := stdout.String()

    // Should contain version info (dev in tests)
    if !strings.Contains(output, "tnotify") {
        t.Errorf("Version output should contain 'tnotify', got: %s", output)
    }
    if !strings.Contains(output, "Checking for updates") {
        t.Errorf("Version output should check for updates, got: %s", output)
    }
}
```

The version command makes HTTP requests to check for updates. The entire version check sequence writes to stdout, so the test captures everything the user would see.

## Pattern 3: Environmental Isolation

Some tests need to control environment variables to simulate different terminal types. This pattern appears in `TestProgressFlagSupported`:

```go
func TestProgressFlagSupported(t *testing.T) {
    // Mock Ghostty terminal (supports progress)
    os.Setenv("GHOSTTY_RESOURCES_DIR", "/tmp/ghostty")
    defer os.Unsetenv("GHOSTTY_RESOURCES_DIR")

    cmd := newTestCommand()
    buf := new(bytes.Buffer)
    cmd.SetOut(buf)
    cmd.SetErr(buf)
    cmd.SetArgs([]string{"-p", "50"})

    // Capture stdout
    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    err := cmd.Execute()

    w.Close()
    os.Stdout = oldStdout

    if err != nil {
        t.Fatalf("-p 50 returned error: %v", err)
    }

    var stdout bytes.Buffer
    stdout.ReadFrom(r)
    output := stdout.String()

    if !strings.Contains(output, "\x1b]9;4;") {
        t.Errorf("-p 50 should output OSC 9;4 sequence on supported terminal, got: %q", output)
    }
}
```

The environment variable `GHOSTTY_RESOURCES_DIR` tells tnotify to behave as if it's running in the Ghostty terminal. The test verifies that the correct OSC escape sequence appears in the output.

The inverse test clears all relevant environment variables to ensure fallback behavior works:

```go
func TestProgressFlagUnsupported(t *testing.T) {
    // Save and clear all terminal env vars that could indicate progress support
    envVars := []string{"GHOSTTY_RESOURCES_DIR", "WT_SESSION", "TERM_PROGRAM"}
    saved := make(map[string]string)
    for _, v := range envVars {
        saved[v] = os.Getenv(v)
        os.Unsetenv(v)
    }
    defer func() {
        for k, v := range saved {
            if v != "" {
                os.Setenv(k, v)
            }
        }
    }()

    cmd := newTestCommand()
    buf := new(bytes.Buffer)
    cmd.SetOut(buf)
    cmd.SetErr(buf)
    cmd.SetArgs([]string{"-p", "50"})

    // Capture stdout
    oldStdout := os.Stdout
    r, w, _ := os.Pipe()
    os.Stdout = w

    err := cmd.Execute()

    w.Close()
    os.Stdout = oldStdout

    if err != nil {
        t.Fatalf("-p 50 returned error: %v", err)
    }

    var stdout bytes.Buffer
    stdout.ReadFrom(r)
    output := stdout.String()

    if output != "Progress: 50%\n" {
        t.Errorf("-p 50 should output plain text on unsupported terminal, got: %q", output)
    }
}
```

This verifies the fallback: when progress indicators aren't supported, tnotify prints plain text instead of escape sequences.

## When to Use Each Pattern

**Use Cobra's SetOut() when**:
- Your command only writes via `cmd.Print()`, `cmd.Println()`, or `cmd.Printf()`
- You're testing help text, usage messages, or Cobra-generated output
- You don't need to intercept arbitrary stdout writes

**Use OS pipe redirection when**:
- Your code calls `fmt.Print()`, `fmt.Println()`, or writes directly to `os.Stdout`
- You're testing output that needs to work in non-interactive contexts (pipes, redirects)
- You need to verify exact byte sequences (like ANSI escape codes)

**Use environment mocking with pipe redirection when**:
- Output behavior changes based on environment variables
- You need to test fallback paths for different terminal types
- You want to verify both supported and unsupported scenarios

## Common Mistakes

The pipe pattern has a critical ordering requirement. You must close the writer before reading from the pipe. If you try to read first, your test will deadlock:

```go
// Wrong: This will hang
r, w, _ := os.Pipe()
os.Stdout = w
cmd.Execute()
var buf bytes.Buffer
buf.ReadFrom(r)  // Blocks forever waiting for EOF
w.Close()
```

The reader blocks until it sees EOF. EOF only arrives when you close the writer. Always close before reading.

Another gotcha: error handling on `os.Pipe()`. The code examples elide the error (`r, w, _ := os.Pipe()`). In production code, check that error. In tests, it's acceptable to skip it because pipe creation only fails in extreme circumstances (out of file descriptors, kernel bugs).

## Testing Binary Output

The bell test shows how to verify exact binary sequences. The expected output is `\x07` (ASCII BEL). String comparison works:

```go
if stdout.String() != "\x07" {
    t.Errorf("--bell should output bell character, got: %q", stdout.String())
}
```

The `%q` verb in the error message prints non-printable characters as escape sequences. If the test fails, you see exactly what bytes appeared instead of invisible characters.

For more complex sequences like ANSI escape codes, use `strings.Contains()`:

```go
if !strings.Contains(output, "\x1b]9;4;") {
    t.Errorf("Expected OSC 9;4 sequence, got: %q", output)
}
```

This checks for the prefix without requiring an exact match. Useful when the exact sequence includes dynamic values (like progress percentages).

## Cleanup and Restoration

Every test that modifies `os.Stdout` must restore it. Missing this causes test pollution - later tests see the redirected stdout instead of the real one.

The safest pattern:

```go
oldStdout := os.Stdout
r, w, _ := os.Pipe()
os.Stdout = w
defer func() { os.Stdout = oldStdout }()
```

The `defer` ensures restoration even if the test panics. But in the tnotify tests, restoration happens manually after closing the writer:

```go
w.Close()
os.Stdout = oldStdout
```

This works because Go tests abort on panic anyway. The defer isn't strictly necessary, but it doesn't hurt either.

## Alternative Approaches

Some codebases pass `io.Writer` parameters to all functions that produce output. This makes testing trivial - just pass a `bytes.Buffer`. But it's invasive. Every function signature gains an extra parameter.

Another option: build your CLI around an interface that wraps stdout. Then mock the interface in tests. This works well for larger applications but adds complexity for simple tools.

The pipe swap pattern requires no changes to your actual CLI code. It works with any Go code that writes to stdout, including third-party libraries. The tradeoff is test complexity - you need the same boilerplate in every test.

## Related Techniques

The same pattern works for stdin. Create a pipe, write test input to it, and swap `os.Stdin`:

```go
r, w, _ := os.Pipe()
oldStdin := os.Stdin
os.Stdin = r

go func() {
    w.Write([]byte("test input\n"))
    w.Close()
}()

// Run command that reads from stdin
cmd.Execute()

os.Stdin = oldStdin
```

The goroutine prevents deadlock. If you write to the pipe in the same goroutine that reads from it, you'll block.

For stderr, the pattern is identical - swap `os.Stderr` instead of `os.Stdout`. The tnotify tests don't need this because they use Cobra's `SetErr()` for error output.
