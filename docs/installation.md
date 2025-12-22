# Installation

tnotify is distributed as a single binary with no runtime dependencies.

## Homebrew (macOS/Linux)

The recommended installation method for macOS and Linux:

```bash
brew install soloterm/tap/tnotify
```

To upgrade:

```bash
brew upgrade tnotify
```

## Go Install

If you have Go 1.22+ installed:

```bash
go install github.com/soloterm/tnotify@latest
```

This installs the binary to your `$GOPATH/bin` directory.

## Binary Download

Download pre-built binaries from [GitHub Releases](https://github.com/soloterm/tnotify/releases).

Available platforms:
- macOS (amd64, arm64)
- Linux (amd64, arm64)
- Windows (amd64, arm64)

### Manual Installation

1. Download the appropriate archive for your platform
2. Extract the binary
3. Move it to a directory in your PATH

Example for macOS/Linux:

```bash
# Download (replace VERSION and PLATFORM)
curl -L https://github.com/soloterm/tnotify/releases/download/VERSION/tnotify_VERSION_PLATFORM.tar.gz | tar xz

# Move to PATH
sudo mv tnotify /usr/local/bin/
```

## Verify Installation

Check that tnotify is installed correctly:

```bash
tnotify --version
```

Test that notifications work:

```bash
tnotify --diagnose
```

## Requirements

tnotify has no runtime dependencies. However, for full functionality:

- **OSC notifications**: Requires a terminal that supports OSC 9, OSC 99, or OSC 777
- **Native fallback on Linux**: Requires `notify-send` (libnotify) for native notifications
- **tmux passthrough**: Requires tmux 3.2+ with `allow-passthrough on`

See [Terminal Support](./terminal-support.md) for details on which terminals support which features.
