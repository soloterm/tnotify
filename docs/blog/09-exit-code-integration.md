# Exit Code Integration: Smart Notifications for Build Systems

When a long build finishes, you want to know immediately—and you want to know if it succeeded or failed. This pattern integrates exit codes with notifications, so failures get urgent alerts while successes are quiet.

## The Use Case

You're running a build, test suite, or deployment. You switch to another window to work while it runs. When it finishes, you want:

1. **Notification** that it's done
2. **Urgency** that reflects success/failure
3. **Optional silence** on success (only notify on failure)

```bash
# Notify with auto-urgency based on exit code
make build; tnotify -e $? 'Build complete'

# Only notify on failure
make test; tnotify -e $? --if-failed 'Tests failed!'

# With custom title
./deploy.sh; tnotify -e $? -t 'Deploy' 'Finished'
```

## Implementation

### Flag Definition

```go
var (
    exitCode    int
    useExitCode bool  // --if-failed flag
    urgency     string
)

func init() {
    rootCmd.Flags().IntVarP(&exitCode, "exit-code", "e", 0,
        "Previous command's exit code (sets urgency automatically)")
    rootCmd.Flags().BoolVar(&useExitCode, "if-failed", false,
        "Only notify if exit code is non-zero")
    rootCmd.Flags().StringVarP(&urgency, "urgency", "u", "normal",
        "Notification urgency: low, normal, critical")
}
```

### Exit Code to Urgency Mapping

```go
const (
    UrgencyLow      = 0
    UrgencyNormal   = 1
    UrgencyCritical = 2
)

func urgencyFromExitCode(code int) int {
    if code == 0 {
        return UrgencyNormal
    }
    return UrgencyCritical
}
```

Simple rule: exit code 0 means success (normal urgency), anything else means failure (critical urgency).

### Main Logic

```go
func run(cmd *cobra.Command, args []string) error {
    // Handle --if-failed: skip notification if exit code is 0
    if useExitCode && exitCode == 0 {
        return nil  // Silent success
    }

    message := args[0]  // Or from stdin

    // Determine urgency
    urgencyLevel := parseUrgency(urgency)  // From -u flag

    // Override with exit code if -e was provided
    if cmd.Flags().Changed("exit-code") {
        urgencyLevel = urgencyFromExitCode(exitCode)
    }

    // Send notification with appropriate urgency
    return sendNotification(title, message, urgencyLevel)
}

func parseUrgency(s string) int {
    switch strings.ToLower(s) {
    case "low":
        return UrgencyLow
    case "critical":
        return UrgencyCritical
    default:
        return UrgencyNormal
    }
}
```

The `cmd.Flags().Changed("exit-code")` check is important. If the user explicitly passes `-e 0`, we want to use it. But if they don't pass `-e` at all, we shouldn't override the `-u` flag.

## Usage Patterns

### Basic Build Notification

```bash
make build; tnotify -e $? 'Build finished'
```

If `make build` exits 0 → normal notification.
If `make build` exits non-zero → critical/urgent notification.

### Failure-Only Notification

```bash
make test; tnotify -e $? --if-failed 'Tests failed!'
```

If tests pass (exit 0) → no notification at all.
If tests fail → critical notification.

### Combined with Title

```bash
./deploy.sh; tnotify -e $? -t 'Production Deploy' 'Deploy completed'
```

The message says "completed" but the urgency tells you if it actually succeeded.

### In CI/CD Pipelines

```bash
#!/bin/bash
set -e

npm run build
npm run test
npm run deploy

# If we get here, everything succeeded
tnotify -t 'Pipeline' 'All stages passed'
```

Or with explicit handling:

```bash
#!/bin/bash

npm run build
BUILD_EXIT=$?

npm run test
TEST_EXIT=$?

if [ $BUILD_EXIT -ne 0 ] || [ $TEST_EXIT -ne 0 ]; then
    tnotify -u critical -t 'Pipeline' 'Build or tests failed'
    exit 1
fi

tnotify -t 'Pipeline' 'All stages passed'
```

### Watch Mode Integration

```bash
# Notify on each test run result
while true; do
    inotifywait -r -e modify src/
    npm test; tnotify -e $? --if-failed 'Tests broken!'
done
```

## Shell Function Wrapper

For frequent use, wrap it in a shell function:

```bash
# Add to ~/.bashrc or ~/.zshrc
notify-after() {
    "$@"
    local exit_code=$?
    tnotify -e $exit_code -t "$(basename $1)" "${*:2} finished"
    return $exit_code
}

# Usage
notify-after make build
notify-after npm test
notify-after ./deploy.sh production
```

Or for failure-only:

```bash
notify-on-fail() {
    "$@"
    local exit_code=$?
    if [ $exit_code -ne 0 ]; then
        tnotify -u critical -t "$(basename $1)" "${*:2} failed (exit $exit_code)"
    fi
    return $exit_code
}
```

## How OSC Protocols Handle Urgency

Different terminals handle urgency differently:

### Kitty (OSC 99)

Kitty's protocol has explicit urgency support:

```go
func BuildOSC99(title, body string, urgency int) string {
    // u=0 (low), u=1 (normal), u=2 (critical)
    return fmt.Sprintf("\x1b]99;d=0:p=title:u=%d;%s\x1b\\", urgency, title) +
           fmt.Sprintf("\x1b]99;d=1:p=body;%s\x1b\\", body)
}
```

Critical notifications may:
- Play a sound
- Persist longer
- Show more prominently

### OSC 9 and OSC 777

These protocols don't have urgency parameters. When using native fallback, we can pass urgency to the system notification:

```go
// Linux notify-send supports urgency
cmd := exec.Command("notify-send", "-u", "critical", title, message)
```

## Testing Exit Code Handling

```go
func TestExitCodeUrgency(t *testing.T) {
    tests := []struct {
        exitCode int
        want     int
    }{
        {0, UrgencyNormal},
        {1, UrgencyCritical},
        {127, UrgencyCritical},  // Command not found
        {255, UrgencyCritical},
    }

    for _, tt := range tests {
        got := urgencyFromExitCode(tt.exitCode)
        if got != tt.want {
            t.Errorf("urgencyFromExitCode(%d) = %d, want %d",
                tt.exitCode, got, tt.want)
        }
    }
}

func TestIfFailedFlag(t *testing.T) {
    // Test that --if-failed with exit code 0 produces no output
    var buf bytes.Buffer
    cmd := NewRootCmd()
    cmd.SetOut(&buf)
    cmd.SetArgs([]string{"-e", "0", "--if-failed", "Should not appear"})

    err := cmd.Execute()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if buf.Len() > 0 {
        t.Errorf("--if-failed with exit 0 should produce no output, got: %q", buf.String())
    }
}
```

## Edge Cases

### Exit Code 0 with Critical Urgency

The user might want success to be urgent:

```bash
# Explicit urgency overrides exit code behavior
tnotify -e 0 -u critical 'Deploy succeeded - verify in production!'
```

Wait, that won't work with our current logic—`-e` overrides `-u`. Let's reconsider:

```go
// Better logic: -u explicitly set takes precedence
urgencyLevel := UrgencyNormal

if cmd.Flags().Changed("urgency") {
    urgencyLevel = parseUrgency(urgency)
} else if cmd.Flags().Changed("exit-code") {
    urgencyLevel = urgencyFromExitCode(exitCode)
}
```

Now `-u critical` works even with `-e 0`.

### Missing Exit Code

If the user forgets `$?`:

```bash
make build
tnotify -e 'Build finished'  # Oops, forgot $?
```

Cobra will complain that "Build finished" isn't a valid integer. Clear error message.

### Exit Codes > 128

Exit codes 128+ often indicate the process was killed by a signal:
- 130 = SIGINT (Ctrl+C)
- 137 = SIGKILL
- 139 = SIGSEGV

These are still "failures" and get critical urgency, which is correct.

## Conclusion

Exit code integration is a small feature with outsized utility for build automation:

1. **Pass exit code**: `command; tnotify -e $?`
2. **Auto-urgency**: 0 = normal, non-zero = critical
3. **Failure-only**: `--if-failed` skips notification on success
4. **Explicit override**: `-u critical` takes precedence

The pattern turns any command into a notifying command with a simple suffix.

Full implementation: [tnotify's main.go](https://github.com/soloterm/tnotify/blob/main/main.go)
