# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Progress bar test in `--diagnose` command

## [0.1.3] - 2025-12-16

## [0.1.2] - 2025-12-16

## [0.1.1] - 2025-12-16

### Added
- Version check with `--version` / `-v` that shows current version and checks GitHub for updates
- Progress bar support via OSC 9;4 (`-p`, `--progress-state`, `--progress-clear`)
  - Supported by Windows Terminal, Ghostty, ConEmu, Mintty
  - States: normal (green), error (red), paused (yellow), indeterminate
- iTerm2 attention request (`--attention`, `--fireworks`)
  - Bounces dock icon on macOS
- Enhanced `--diagnose` command
  - Shows tmux version and `allow-passthrough` status
  - Better troubleshooting guidance
- OSC 8 hyperlink support (internal)
- OSC 133 shell integration markers (internal)
- Comprehensive troubleshooting documentation
  - macOS Focus Mode guidance
  - VTE/GNOME Terminal configuration
  - tmux passthrough setup

### Changed
- Linux `notify-send` now includes app name and icon by default
- Updated terminal support table in README

## [0.1.0] - 2024-12-16

### Added
- Initial release
- OSC 99 (Kitty) - Full featured: urgency, notification IDs
- OSC 777 (WezTerm, Ghostty, VTE) - Title + body
- OSC 9 (iTerm2) - Message only
- Native fallbacks: osascript (macOS), notify-send (Linux), PowerShell (Windows)
- Terminal bell fallback
- tmux and GNU Screen passthrough support
- Exit code integration (`-e`, `--if-failed`)
- Notification ID support for updates (`-i`, `--close`)
- `--capabilities` JSON output
- `--diagnose` troubleshooting command

[Unreleased]: https://github.com/soloterm/tnotify/compare/v0.1.3...HEAD
[0.1.3]: https://github.com/soloterm/tnotify/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/soloterm/tnotify/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/soloterm/tnotify/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/soloterm/tnotify/releases/tag/v0.1.0
