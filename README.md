# tnotify

Send desktop notifications from the terminal via OSC escape sequences or native tools.

## Features

- **OSC-first**: Works over SSH, inside tmux/screen
- **Smart fallbacks**: Uses native tools when OSC unavailable
- **Cross-platform**: macOS, Linux, Windows
- **Zero dependencies**: Single binary, no runtime requirements

## Installation

### Homebrew (macOS/Linux)

```bash
brew install soloterm/tap/tnotify
```

### Go Install

```bash
go install github.com/soloterm/tnotify@latest
```

### Binary Download

Download from [GitHub Releases](https://github.com/soloterm/tnotify/releases).

## Usage

```bash
# Simple notification
tnotify "Build complete!"

# With title
tnotify -t "My App" "Task finished"

# With urgency (low, normal, critical)
tnotify -u critical "Server down!"

# Notification ID for updates (Kitty only)
tnotify -i progress "Building... 50%"
tnotify -i progress "Building... 100%"
tnotify --close progress

# Pipe input
echo "Done" | tnotify -t "Results"

# Force specific method
tnotify --osc "Uses escape sequences only"
tnotify --native "Uses osascript/notify-send only"
tnotify --bell  # Just beep

# Show capabilities
tnotify --capabilities
```

## Terminal Support

| Terminal | Protocol | Title | Urgency | IDs |
|----------|----------|-------|---------|-----|
| Kitty | OSC 99 | ✓ | ✓ | ✓ |
| iTerm2 | OSC 9 | ✗ | ✗ | ✗ |
| WezTerm | OSC 777 | ✓ | ✗ | ✗ |
| Ghostty | OSC 777 | ✓ | ✗ | ✗ |
| VTE (GNOME Terminal) | OSC 777 | ✓ | ✗ | ✗ |
| foot | OSC 777 | ✓ | ✗ | ✗ |
| Others | Native fallback | ✓ | ✓ | ✗ |

## Native Fallbacks

When OSC notifications aren't supported, tnotify falls back to:

- **macOS**: `osascript` (AppleScript)
- **Linux**: `notify-send` (libnotify)
- **Windows**: PowerShell toast notifications

## Releasing (Maintainers)

### First-time Homebrew Setup

1. **Create the tap repository**: Create `soloterm/homebrew-tap` on GitHub (public repo)

2. **Create a Personal Access Token**:
   - Go to GitHub → Settings → Developer settings → Personal access tokens → Fine-grained tokens
   - Create token with:
     - Repository access: `soloterm/homebrew-tap` only
     - Permissions: Contents (Read and write)
   - Copy the token

3. **Add the secret to tnotify repo**:
   - Go to `soloterm/tnotify` → Settings → Secrets and variables → Actions
   - Add new repository secret: `HOMEBREW_TAP_GITHUB_TOKEN` with the token value

### Creating a Release

1. Go to Actions → Release → Run workflow
2. Enter version in `X.Y.Z` format (e.g., `1.0.0`)
3. Click "Run workflow"

This will:
- Validate the version format
- Create and push a git tag (`v1.0.0`)
- Build binaries for all platforms
- Create a GitHub release with artifacts
- Update the Homebrew formula in `soloterm/homebrew-tap`

## License

MIT
