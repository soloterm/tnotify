# Input Sanitization for Escape Sequences

Terminal escape sequences are strings with special meaning. OSC (Operating System Command) sequences let applications send desktop notifications, update progress bars, and request user attention. But here's the problem: if you're building these sequences from user input, you need to sanitize that input. Otherwise, users can break your protocol or inject their own commands.

The tnotify CLI tool sends desktop notifications via OSC escape sequences. Every notification includes user-provided text—titles, messages, IDs. Each of these needs sanitization before being embedded in an OSC sequence.

## Why Sanitization Matters

OSC sequences have a specific structure. Here's OSC 777 (used by WezTerm, Ghostty, and VTE-based terminals):

```
ESC ] 777 ; notify ; title ; body BEL
```

The semicolon is a delimiter. If a user includes a semicolon in their message, the terminal will parse it as a new field. The notification breaks or behaves unexpectedly.

Control characters are worse. A newline (`\n`), tab (`\t`), or escape character (`\x1b`) in user input can terminate the sequence early or inject a completely different escape sequence.

## The Solution

`internal/osc/sanitize.go`:

```go
var (
    // controlCharsRegex matches control characters (except space)
    controlCharsRegex = regexp.MustCompile(`[\x00-\x1f\x7f]`)

    // idSanitizeRegex matches invalid ID characters
    idSanitizeRegex = regexp.MustCompile(`[^a-zA-Z0-9_\-+.]`)
)

// Sanitize removes control characters and semicolons from a string.
// Semicolons are replaced with colons to avoid breaking OSC 777 parsing.
func Sanitize(s string) string {
    // Remove control characters
    s = controlCharsRegex.ReplaceAllString(s, "")

    // Replace semicolons with colons to avoid breaking OSC 777 parsing
    s = strings.ReplaceAll(s, ";", ":")

    return s
}

// SanitizeID sanitizes a notification ID for OSC 99.
// IDs must only contain [a-zA-Z0-9_-+.] characters.
func SanitizeID(id string) string {
    return idSanitizeRegex.ReplaceAllString(id, "")
}
```

Two functions. `Sanitize()` handles message and title fields. `SanitizeID()` handles notification IDs, which have stricter requirements.

## How It Works

### Control Character Removal

The regex `[\x00-\x1f\x7f]` matches all ASCII control characters (0x00 through 0x1F) plus DEL (0x7F). This includes:
- Null bytes (`\x00`)
- Newlines (`\n` = `\x0A`)
- Tabs (`\t` = `\x09`)
- Carriage returns (`\r` = `\x0D`)
- The escape character itself (`\x1b`)

These are all stripped out. A message like `"Build\ncomplete"` becomes `"Buildcomplete"`.

Notice the regex excludes space (0x20). Space is technically in the control range but it's printable, so we keep it.

### Semicolon Replacement

Semicolons delimit fields in OSC sequences. We can't just remove them—that loses information. Instead, we replace them with colons.

From the test suite:

```go
{
    name:  "semicolons replaced with colons",
    input: "foo;bar;baz",
    want:  "foo:bar:baz",
},
```

This preserves the user's intent while keeping the OSC sequence valid.

### Notification ID Sanitization

Notification IDs have different requirements. The OSC 99 spec (used by Kitty terminal) expects IDs to be simple identifiers. Unicode emoji or special characters don't belong here.

`SanitizeID()` strips everything except:
- Letters (a-z, A-Z)
- Numbers (0-9)
- Underscores, hyphens, plus signs, dots (`_-+.`)

An ID like `"my notification 🎉"` becomes `"mynotification"`.

## Where Sanitization Happens

Every OSC builder function calls sanitization before constructing the sequence.

`internal/osc/osc99.go`:

```go
func BuildOSC99(message, title string, urgency int, id string) string {
    // Sanitize ID if provided
    idMeta := ""
    if id != "" {
        id = SanitizeID(id)
        if id != "" {
            idMeta = "i=" + id
        }
    }

    if title == "" {
        // Simple notification with just a body
        message = Sanitize(message)
        // ... build sequence
    }

    // Multi-part notification with title and body
    title = Sanitize(title)
    message = Sanitize(message)
    // ... build sequence
}
```

The sanitization happens at the last moment, right before the sequence is built. This means the rest of the codebase can pass around raw user input without worrying about injection attacks. The protocol layer handles safety.

`internal/osc/osc777.go`:

```go
func BuildOSC777(message, title string) string {
    if title == "" {
        title = "Notification"
    }
    title = Sanitize(title)
    message = Sanitize(message)
    return fmt.Sprintf("\x1b]777;notify;%s;%s\x07", title, message)
}
```

Same pattern: sanitize, then build.

## What About Unicode?

Unicode is fine. The sanitization only removes ASCII control characters and protocol-breaking symbols. Multi-byte UTF-8 sequences pass through unchanged.

From the test suite:

```go
{
    name:  "unicode preserved",
    input: "Hello 世界 🎉",
    want:  "Hello 世界 🎉",
},
```

Users can send notifications in any language. The sanitizer won't corrupt their text.

## Edge Cases

What if sanitization removes everything? This happens when an ID contains only invalid characters:

```go
{
    name:  "only invalid chars",
    input: "@#$%^&*()",
    want:  "",
},
```

The code handles this. In `BuildOSC99()`:

```go
if id != "" {
    id = SanitizeID(id)
    if id != "" {
        idMeta = "i=" + id
    }
}
```

After sanitization, the code checks again. If the ID is now empty, it doesn't include the metadata field at all. The notification still goes through, just without an ID.

## The Cost of Safety

Two regex replacements per notification. Is that expensive? No.

The regex patterns are compiled once at package initialization (package-level variables). Each call just runs the state machine. For notification text (usually < 100 characters), this takes nanoseconds.

You could optimize further by scanning the string byte-by-byte and building a new string only if you find something to remove. But notifications are sent once, not in a tight loop. The clarity of regex-based sanitization outweighs the performance gain.

## Alternative Approaches

### Escape Instead of Remove

You could escape semicolons instead of replacing them:

```go
s = strings.ReplaceAll(s, ";", "\\;")
```

But now you need the terminal to understand escaped semicolons. Most terminals don't. You'd need to coordinate with multiple terminal emulator implementations. Not practical.

### Whitelist Characters

Instead of removing control characters, you could allow only known-good characters:

```go
allowedChars := regexp.MustCompile(`[a-zA-Z0-9\s\p{L}\p{N}]`)
```

This is safer but breaks for legitimate punctuation and symbols. Users expect to be able to send `"Build failed! (exit code: 1)"` without losing the parentheses and colon.

### Base64 Encoding

Some protocols solve this by Base64-encoding the payload. No special characters, no injection risk.

But Base64 makes debugging harder. When you see a malformed OSC sequence in a terminal log, you want to read it. Encoded payloads turn a 5-second fix into a 5-minute investigation.

## Testing Sanitization

The test suite covers realistic attack vectors:

```go
{
    name:  "escape sequences removed",
    input: "text\x1b[31mred\x1b[0m",
    want:  "text[31mred[0m",
},
{
    name:  "mixed control and semicolons",
    input: "a\x00;b\nc",
    want:  "a:bc",
},
```

The first test shows someone trying to inject ANSI color codes into a notification. The escape character gets stripped, leaving behind the literal text `[31mred[0m`.

The second test combines null bytes, semicolons, and newlines—the kind of garbage you get from corrupted logs or binary output being piped to stdout.

## Takeaways

If you're building terminal escape sequences from user input:

1. **Sanitize at the protocol boundary.** Keep the rest of your code simple by handling safety at the last moment.
2. **Know your delimiters.** OSC sequences use semicolons. Your protocol might use something else. Strip or replace them.
3. **Remove control characters.** They break sequences or inject new ones.
4. **Test edge cases.** Empty strings, unicode, mixed control characters.
5. **Don't overthink it.** Regex-based sanitization is fast enough and easier to audit than hand-rolled parsers.

Escape sequences are powerful but fragile. Treat user input like hostile data, sanitize it, and your protocol will work reliably.
