# Cross-Platform Desktop Notifications Without Dependencies

When OSC escape sequences aren't available (unsupported terminal, redirected output, remote session), you need native fallbacks. The challenge: macOS, Linux, and Windows each have completely different notification APIs.

Here's how to implement all three without adding external dependencies.

## The Architecture

We use a simple interface that each platform implements:

```go
type Sender interface {
    Available() bool
    Send(message, title string, urgency int) bool
}

func NewSender() Sender {
    switch runtime.GOOS {
    case "darwin":
        return &MacOSSender{}
    case "linux":
        return &LinuxSender{}
    case "windows":
        return &WindowsSender{}
    default:
        return &NullSender{}
    }
}
```

## macOS: AppleScript via osascript

macOS has a built-in notification system accessible through AppleScript. The `osascript` command executes AppleScript from the terminal:

```go
type MacOSSender struct{}

func (s *MacOSSender) Available() bool {
    _, err := exec.LookPath("osascript")
    return err == nil
}

func (s *MacOSSender) Send(message, title string, urgency int) bool {
    // Escape strings for AppleScript
    title = escapeAppleScript(title)
    message = escapeAppleScript(message)

    script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)
    cmd := exec.Command("osascript", "-e", script)
    return cmd.Run() == nil
}
```

### AppleScript Escaping

AppleScript strings use double quotes, so we need to escape backslashes and quotes:

```go
func escapeAppleScript(s string) string {
    // Backslashes must be escaped first (before we add more)
    s = strings.ReplaceAll(s, "\\", "\\\\")
    // Then escape double quotes
    s = strings.ReplaceAll(s, "\"", "\\\"")
    return s
}
```

Test cases:

```go
func TestEscapeAppleScript(t *testing.T) {
    tests := []struct {
        input string
        want  string
    }{
        {`Hello`, `Hello`},
        {`Say "Hello"`, `Say \"Hello\"`},
        {`Path: C:\Users`, `Path: C:\\Users`},
        {`"Quote" and \slash`, `\"Quote\" and \\slash`},
    }
    // ...
}
```

### macOS Quirks

1. **Sound**: Add `sound name "default"` for an audible notification
2. **Subtitle**: AppleScript supports `subtitle` for a second line
3. **Sender**: Notifications appear as "Script Editor" in Notification Center

```go
// With all options
script := fmt.Sprintf(
    `display notification "%s" with title "%s" subtitle "%s" sound name "default"`,
    message, title, subtitle,
)
```

## Linux: notify-send (libnotify)

Most Linux desktop environments support libnotify. The `notify-send` command is the CLI interface:

```go
type LinuxSender struct{}

func (s *LinuxSender) Available() bool {
    _, err := exec.LookPath("notify-send")
    return err == nil
}

func (s *LinuxSender) Send(message, title string, urgency int) bool {
    // Map urgency levels
    urgencyLevel := "normal"
    switch urgency {
    case 0:
        urgencyLevel = "low"
    case 2:
        urgencyLevel = "critical"
    }

    args := []string{
        "-u", urgencyLevel,      // Urgency
        "-a", "tnotify",         // App name
        "-i", "utilities-terminal", // Icon
        title,
        message,
    }

    cmd := exec.Command("notify-send", args...)
    return cmd.Run() == nil
}
```

### Linux Quirks

1. **No escaping needed**: Arguments are passed directly, not through a shell
2. **Icons**: Use icon names from the system theme (e.g., `dialog-information`)
3. **Actions**: notify-send supports clickable actions with `-A`
4. **Expiration**: Use `-t` for timeout in milliseconds (0 = persistent)

```go
// With timeout and custom icon
args := []string{
    "-u", urgencyLevel,
    "-a", appName,
    "-i", "dialog-warning",
    "-t", "5000", // 5 seconds
    title,
    message,
}
```

### Availability Check

Not all Linux systems have notify-send installed:

```bash
# Debian/Ubuntu
sudo apt install libnotify-bin

# Fedora
sudo dnf install libnotify

# Arch
sudo pacman -S libnotify
```

The `Available()` check handles this gracefully—if notify-send isn't found, we can fall back to other methods (terminal bell, etc.).

## Windows: PowerShell Toast Notifications

Windows is the trickiest. We use PowerShell to create a balloon notification:

```go
type WindowsSender struct{}

func (s *WindowsSender) Available() bool {
    _, err := exec.LookPath("powershell")
    return err == nil
}

func (s *WindowsSender) Send(message, title string, urgency int) bool {
    script := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$balloon = New-Object System.Windows.Forms.NotifyIcon
$balloon.Icon = [System.Drawing.SystemIcons]::Information
$balloon.BalloonTipTitle = "%s"
$balloon.BalloonTipText = "%s"
$balloon.Visible = $true
$balloon.ShowBalloonTip(5000)
Start-Sleep -Milliseconds 100
$balloon.Dispose()
`, escapeForPowerShell(title), escapeForPowerShell(message))

    // Encode as UTF-16LE base64 for PowerShell
    encoded := encodeForPowerShell(script)
    cmd := exec.Command("powershell", "-EncodedCommand", encoded)
    return cmd.Run() == nil
}
```

### PowerShell Encoding

PowerShell's `-EncodedCommand` expects UTF-16LE encoded base64. This avoids quoting issues entirely:

```go
func encodeForPowerShell(script string) string {
    // Convert to UTF-16LE
    runes := []rune(script)
    u16 := utf16.Encode(runes)

    // Convert to bytes (little-endian)
    bytes := make([]byte, len(u16)*2)
    for i, r := range u16 {
        bytes[i*2] = byte(r)
        bytes[i*2+1] = byte(r >> 8)
    }

    return base64.StdEncoding.EncodeToString(bytes)
}
```

Why UTF-16LE? Windows internally uses UTF-16, and PowerShell's encoded command mode expects this format. Using `-EncodedCommand` avoids all shell escaping issues.

### PowerShell String Escaping

For the string literals inside the script, we still need basic escaping:

```go
func escapeForPowerShell(s string) string {
    // Escape backticks (PowerShell escape character)
    s = strings.ReplaceAll(s, "`", "``")
    // Escape double quotes
    s = strings.ReplaceAll(s, "\"", "`\"")
    // Escape dollar signs (variable expansion)
    s = strings.ReplaceAll(s, "$", "`$")
    return s
}
```

### Windows Quirks

1. **Icon**: Must set an icon or the notification won't show
2. **Visibility**: `$balloon.Visible = $true` is required
3. **Disposal**: Must dispose the NotifyIcon or it persists
4. **Sleep**: Brief sleep ensures notification appears before disposal

## Putting It Together

The main code tries OSC first, then falls back to native:

```go
func sendNotification(title, message string, urgency int) error {
    // Try OSC notification
    terminal := detect.DetectTerminal()
    protocol := detect.SelectProtocol(terminal)

    if protocol != detect.ProtocolNone {
        sequence := osc.BuildNotification(protocol, title, message, urgency)
        if InMultiplexer() {
            sequence = osc.WrapIfNeeded(sequence)
        }
        return sendToTerminal(sequence)
    }

    // Fall back to native
    sender := native.NewSender()
    if sender.Available() && sender.Send(message, title, urgency) {
        return nil
    }

    // Last resort: terminal bell
    fmt.Print("\a")
    return nil
}
```

## Testing Native Notifications

Testing platform-specific code requires build tags:

```go
// native_macos_test.go
//go:build darwin

package native

func TestMacOSSender(t *testing.T) {
    sender := &MacOSSender{}

    if !sender.Available() {
        t.Skip("osascript not available")
    }

    // This actually sends a notification!
    if !sender.Send("Test message", "Test Title", 1) {
        t.Error("Send failed")
    }
}
```

For CI environments without a display:

```go
func TestMacOSSenderAvailable(t *testing.T) {
    sender := &MacOSSender{}
    // Just test availability check, not actual sending
    available := sender.Available()
    // On macOS, osascript should always be available
    if runtime.GOOS == "darwin" && !available {
        t.Error("osascript should be available on macOS")
    }
}
```

## Security Considerations

### Command Injection

Never build shell commands with string concatenation:

```go
// DANGEROUS - command injection possible
cmd := exec.Command("bash", "-c",
    fmt.Sprintf("notify-send '%s' '%s'", title, message))

// SAFE - arguments passed directly
cmd := exec.Command("notify-send", title, message)
```

The safe version passes arguments directly to the executable, bypassing shell interpretation.

### AppleScript Injection

Even with proper escaping, be cautious:

```go
// Our escaping handles basic cases
script := fmt.Sprintf(`display notification "%s"`, escapeAppleScript(userInput))

// But for extra safety, consider length limits
if len(message) > 1000 {
    message = message[:1000]
}
```

## Conclusion

Cross-platform notifications without dependencies require understanding three different APIs:

| Platform | Tool | Escaping | Encoding |
|----------|------|----------|----------|
| macOS | osascript | Backslash, quotes | Plain |
| Linux | notify-send | None (direct args) | UTF-8 |
| Windows | PowerShell | Backtick, quotes, $ | UTF-16LE base64 |

The key patterns:
1. Use interfaces for platform abstraction
2. Pass arguments directly, not through shells
3. Check availability before attempting to send
4. Have fallbacks when native isn't available

Full implementation: [tnotify's native package](https://github.com/soloterm/tnotify/tree/main/internal/native)
