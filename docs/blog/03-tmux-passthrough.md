# tmux and GNU Screen Passthrough: Making Terminal Sequences Work Over SSH

You've built a CLI tool that sends notifications via OSC escape sequences. It works perfectly in your local terminal. Then you SSH into a server, start tmux, and... nothing. The notifications disappear into the void.

This is one of the trickiest problems in terminal programming: multiplexer passthrough. Here's how to solve it.

## The Problem

Terminal multiplexers like tmux and GNU Screen sit between your application and the real terminal:

```
Your App → tmux → Terminal Emulator → You
```

When you send an OSC sequence, tmux intercepts it. By default, tmux doesn't know what to do with most OSC sequences, so it either:
1. Ignores them completely
2. Handles them itself (for sequences it recognizes)
3. Corrupts them accidentally

Your carefully crafted `\x1b]9;Hello\x07` never reaches the terminal.

## The Solution: DCS Passthrough

The solution is to wrap your OSC sequence in a DCS (Device Control String) passthrough sequence. This tells the multiplexer: "Don't interpret this. Just pass it through to the real terminal."

### tmux Passthrough

tmux uses this format:

```
DCS tmux; <sequence-with-doubled-escapes> ST
\x1bPtmux; ... \x1b\\
```

The critical detail: **every ESC character in your sequence must be doubled**.

Why? Because tmux scans for ESC to find the end of the DCS sequence. If your payload contains ESC, tmux thinks the sequence is ending. Doubling tells tmux "this is a literal ESC in the payload, not the end marker."

```go
func wrapForTmux(sequence string) string {
    // Double all ESC characters
    doubled := strings.ReplaceAll(sequence, "\x1b", "\x1b\x1b")

    // Wrap in DCS passthrough
    return "\x1bPtmux;" + doubled + "\x1b\\"
}
```

Let's trace through an example:

```
Original:  \x1b]9;Hello\x07
Doubled:   \x1b\x1b]9;Hello\x07
Wrapped:   \x1bPtmux;\x1b\x1b]9;Hello\x07\x1b\\

Breakdown:
\x1bP         - Start DCS
tmux;         - tmux passthrough command
\x1b\x1b]9;   - Doubled ESC + ]9; (OSC 9 start)
Hello\x07     - Message + BEL
\x1b\\        - End DCS (ST)
```

### GNU Screen Passthrough

GNU Screen is simpler—it doesn't require ESC doubling:

```go
func wrapForScreen(sequence string) string {
    return "\x1bP" + sequence + "\x1b\\"
}
```

```
Original:  \x1b]9;Hello\x07
Wrapped:   \x1bP\x1b]9;Hello\x07\x1b\\
```

## Detecting Multiplexers

Before wrapping, you need to know if you're inside a multiplexer and which one:

```go
func InTmux() bool {
    return os.Getenv("TMUX") != ""
}

func InScreen() bool {
    term := os.Getenv("TERM")
    return strings.HasPrefix(term, "screen")
}
```

Then wrap conditionally:

```go
func WrapIfNeeded(sequence string) string {
    if InTmux() {
        return wrapForTmux(sequence)
    }
    if InScreen() {
        return wrapForScreen(sequence)
    }
    return sequence
}
```

## The tmux Configuration Requirement

Here's the catch: even with proper wrapping, tmux won't pass through sequences unless you enable it:

```bash
# In ~/.tmux.conf
set -g allow-passthrough on
```

Without this setting, tmux silently drops your wrapped sequences. This was added in tmux 3.2 (released 2021) for security reasons—arbitrary passthrough could be exploited by malicious content.

### Checking tmux Configuration

You can check if passthrough is enabled:

```go
func TmuxAllowsPassthrough() bool {
    cmd := exec.Command("tmux", "show-options", "-gv", "allow-passthrough")
    output, err := cmd.Output()
    if err != nil {
        return false
    }
    return strings.TrimSpace(string(output)) == "on"
}
```

This is useful for diagnostics—if notifications aren't working, you can tell the user exactly what to fix.

## Handling Multi-Part Sequences

Some protocols (like Kitty's OSC 99) use multiple sequences for one notification:

```
\x1b]99;d=0:p=title;My Title\x1b\\
\x1b]99;d=1:p=body;My Body\x1b\\
```

Each part has its own ESC characters that need doubling:

```go
func wrapForTmux(sequence string) string {
    doubled := strings.ReplaceAll(sequence, "\x1b", "\x1b\x1b")
    return "\x1bPtmux;" + doubled + "\x1b\\"
}

// Input:  \x1b]99;d=0:p=title;Title\x1b\\\x1b]99;d=1:p=body;Body\x1b\\
// Output: \x1bPtmux;\x1b\x1b]99;d=0:p=title;Title\x1b\x1b\\\x1b\x1b]99;d=1:p=body;Body\x1b\x1b\\\x1b\\
```

Notice how every `\x1b` becomes `\x1b\x1b`, including the ST (string terminator) sequences.

## Testing Passthrough

Here's how to test your implementation:

```go
func TestTmuxPassthrough(t *testing.T) {
    tests := []struct {
        name  string
        input string
        want  string
    }{
        {
            name:  "simple OSC 9",
            input: "\x1b]9;Hello\x07",
            want:  "\x1bPtmux;\x1b\x1b]9;Hello\x07\x1b\\",
        },
        {
            name:  "OSC 99 multi-part",
            input: "\x1b]99;d=0:p=title;Title\x1b\\\x1b]99;d=1:p=body;Body\x1b\\",
            want:  "\x1bPtmux;\x1b\x1b]99;d=0:p=title;Title\x1b\x1b\\\x1b\x1b]99;d=1:p=body;Body\x1b\x1b\\\x1b\\",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := wrapForTmux(tt.input)
            if got != tt.want {
                t.Errorf("wrapForTmux() =\n%q\nwant\n%q", got, tt.want)
            }
        })
    }
}
```

## Nested Multiplexers

What if someone runs screen inside tmux? The `TMUX` variable will be set, but `TERM` might still be `screen`. In practice, wrapping for tmux handles this case—the outer tmux passes through to the inner screen, which passes through to the terminal.

The detection order should be:

```go
func WrapIfNeeded(sequence string) string {
    // Check tmux first (more common, and handles nested cases)
    if InTmux() {
        return wrapForTmux(sequence)
    }
    if InScreen() {
        return wrapForScreen(sequence)
    }
    return sequence
}
```

## Troubleshooting

When passthrough isn't working:

1. **Check tmux version**: Passthrough requires tmux 3.2+
   ```bash
   tmux -V
   ```

2. **Check configuration**:
   ```bash
   tmux show-options -g allow-passthrough
   ```

3. **Test manually**:
   ```bash
   # Inside tmux, this should show a notification
   printf '\ePtmux;\e\e]9;Test\a\e\\'
   ```

4. **Watch the raw output**:
   ```bash
   # Outside the multiplexer, capture what's being sent
   your-app | xxd
   ```

## SSH Considerations

When you SSH into a remote machine:
- The remote shell inherits `TMUX` if tmux is running locally
- But `TERM` might be different (often `xterm-256color`)
- The remote app can detect tmux and wrap appropriately
- The wrapped sequence travels back through SSH to your local tmux to your terminal

This is why OSC notifications "just work" over SSH when configured correctly—the passthrough mechanism doesn't care that there's an SSH connection in the middle.

## Conclusion

Multiplexer passthrough is fiddly but essential for CLI tools that need to work everywhere. The key points:

1. **tmux**: Wrap in `\x1bPtmux;...\x1b\\` with doubled ESCs
2. **screen**: Wrap in `\x1bP...\x1b\\` without doubling
3. **tmux 3.2+**: Requires `allow-passthrough on` in config
4. **Test both paths**: With and without multiplexers

The full implementation is in [tnotify's passthrough code](https://github.com/soloterm/tnotify/blob/main/internal/osc/passthrough.go).
