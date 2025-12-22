# OSC Escape Sequences: The Hidden Language Between Terminals and Applications

Every time you see colored text in your terminal, your shell is speaking a secret language: escape sequences. But colors are just the beginning. Modern terminals support a rich vocabulary of "Operating System Commands" (OSC) that enable notifications, hyperlinks, clipboard access, and more.

This post explains what OSC sequences are, how they work, and how to use them for desktop notifications.

## What Are Escape Sequences?

An escape sequence is a series of characters that starts with the ESC character (`0x1B` or `\x1b`). When a terminal sees ESC, it knows the following characters are commands, not text to display.

You're probably familiar with ANSI color codes:

```bash
echo -e "\x1b[31mThis is red\x1b[0m"
```

The `\x1b[31m` means "start red text" and `\x1b[0m` means "reset to default."

## OSC: Operating System Commands

OSC sequences start with `ESC ]` (that's ESC followed by a right bracket) and end with either:
- `BEL` (`\x07`) - the bell character
- `ST` (`\x1b\\`) - the "string terminator"

The general format is:

```
ESC ] <command-number> ; <payload> BEL
\x1b]       N          ;   data   \x07
```

Different command numbers do different things:
- **OSC 0**: Set window title
- **OSC 8**: Create hyperlinks
- **OSC 9**: Send notifications (iTerm2 style)
- **OSC 52**: Access clipboard
- **OSC 99**: Send notifications (Kitty style)
- **OSC 777**: Send notifications (rxvt/WezTerm style)

## Three Notification Protocols

There's no standard for terminal notifications, so different terminals invented their own. Here are the three main ones:

### OSC 9: The Simple One (iTerm2)

The simplest notification protocol. Just a message, nothing else:

```go
func BuildOSC9(message string) string {
    return fmt.Sprintf("\x1b]9;%s\x07", message)
}
```

Send it like this:

```bash
printf '\e]9;Build complete!\a'
```

**Supported by**: iTerm2, ConEmu, some others

**Limitations**: No title, no urgency levels, no IDs for updating notifications.

### OSC 777: Title + Body (WezTerm/Ghostty)

Originally from rxvt-unicode, this format supports a title and body:

```go
func BuildOSC777(title, body string) string {
    return fmt.Sprintf("\x1b]777;notify;%s;%s\x07", title, body)
}
```

```bash
printf '\e]777;notify;Build System;Compilation finished!\a'
```

**Supported by**: WezTerm, Ghostty, VTE-based terminals (GNOME Terminal with patches), foot

**Limitations**: No urgency levels, no IDs.

### OSC 99: The Full-Featured One (Kitty)

Kitty's protocol is the most capable, supporting urgency levels, notification IDs (for updates), and payload types:

```go
func BuildOSC99(title, body string, urgency int) string {
    var b strings.Builder

    // First part: title
    // d=0 means "more data coming", p=title means "this is the title"
    b.WriteString(fmt.Sprintf("\x1b]99;d=0:p=title;%s\x1b\\", title))

    // Second part: body
    // d=1 means "this is the last part", p=body means "this is the body"
    b.WriteString(fmt.Sprintf("\x1b]99;d=1:p=body;%s\x1b\\", body))

    return b.String()
}
```

With urgency (0=low, 1=normal, 2=critical):

```go
func BuildOSC99WithUrgency(title, body string, urgency int) string {
    var b strings.Builder

    // Include urgency in first payload
    b.WriteString(fmt.Sprintf("\x1b]99;d=0:p=title:u=%d;%s\x1b\\", urgency, title))
    b.WriteString(fmt.Sprintf("\x1b]99;d=1:p=body;%s\x1b\\", body))

    return b.String()
}
```

With notification ID (for updating/closing):

```go
func BuildOSC99WithID(title, body, id string) string {
    var b strings.Builder

    // Include ID in first payload
    b.WriteString(fmt.Sprintf("\x1b]99;d=0:p=title:i=%s;%s\x1b\\", id, title))
    b.WriteString(fmt.Sprintf("\x1b]99;d=1:p=body;%s\x1b\\", body))

    return b.String()
}

func CloseOSC99(id string) string {
    return fmt.Sprintf("\x1b]99;d=1:p=close:i=%s;\x1b\\", id)
}
```

**Supported by**: Kitty only (currently)

**Capabilities**: Urgency, IDs, multi-part payloads, close/update notifications.

## Choosing the Right Protocol

Here's how tnotify selects the protocol based on terminal:

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

Then we build the appropriate sequence:

```go
func BuildNotification(protocol Protocol, title, body string, urgency int) string {
    switch protocol {
    case ProtocolOSC9:
        // OSC 9 only supports body, ignore title
        return BuildOSC9(body)
    case ProtocolOSC777:
        return BuildOSC777(title, body)
    case ProtocolOSC99:
        return BuildOSC99WithUrgency(title, body, urgency)
    default:
        return ""
    }
}
```

## Sanitizing User Input

OSC sequences use semicolons as delimiters. If the user's message contains a semicolon, it could break the sequence:

```bash
# This breaks OSC 777:
printf '\e]777;notify;Title;Message; with semicolon\a'
#                              ^ terminal thinks this starts a new field
```

The solution: sanitize input by replacing or removing problematic characters:

```go
var controlCharsRegex = regexp.MustCompile(`[\x00-\x1f\x7f]`)

func Sanitize(s string) string {
    // Remove control characters (they can corrupt sequences)
    s = controlCharsRegex.ReplaceAllString(s, "")

    // Replace semicolons with colons (preserves readability)
    s = strings.ReplaceAll(s, ";", ":")

    return s
}
```

## Writing to the Terminal

Once you've built the sequence, write it to the terminal (not stdout, if they differ):

```go
func sendOSC(sequence string) error {
    // Try /dev/tty first (works even if stdout is redirected)
    tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
    if err != nil {
        // Fall back to stderr (usually connected to terminal)
        _, err = os.Stderr.WriteString(sequence)
        return err
    }
    defer tty.Close()

    _, err = tty.WriteString(sequence)
    return err
}
```

## Debugging OSC Sequences

When things go wrong, it helps to see what you're actually sending. Use `xxd` or `od`:

```bash
printf '\e]9;Hello\a' | xxd
# 00000000: 1b5d 393b 4865 6c6c 6f07                 .]9;Hello.
#           ^ESC ^]  ^9 ^; ^message   ^BEL
```

Or use `cat -v` to show control characters:

```bash
printf '\e]9;Hello\a' | cat -v
# ^[]9;Hello^G
```

## What Terminals Don't Support

Some popular terminals have no OSC notification support:
- **Apple Terminal**: The default macOS terminal
- **VS Code integrated terminal**
- **Alacritty** (by design—it's minimal)
- **Konsole** (KDE's terminal)

For these, you need native fallbacks (osascript on macOS, notify-send on Linux).

## Conclusion

OSC escape sequences are a powerful but underutilized feature of modern terminals. While the lack of standardization is frustrating, the three notification protocols (OSC 9, OSC 777, OSC 99) cover most use cases.

The key takeaways:
1. Detect the terminal first, then select the right protocol
2. Always sanitize user input to prevent sequence corruption
3. Have fallbacks for terminals that don't support OSC

The full implementation is in [tnotify's osc package](https://github.com/soloterm/tnotify/tree/main/internal/osc).

## Further Reading

- [Kitty Terminal Protocol Documentation](https://sw.kovidgoyal.net/kitty/desktop-notifications/)
- [iTerm2 Proprietary Escape Codes](https://iterm2.com/documentation-escape-codes.html)
- [XTerm Control Sequences](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html)
