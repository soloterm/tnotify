# GoReleaser: Multi-Platform CLI Distribution in One Config File

Shipping a Go CLI to macOS, Linux, and Windows—for both Intel and ARM—used to require complex Makefiles and CI scripts. GoReleaser reduces it to a single YAML file and one command.

Here's how tnotify ships to 6 platforms with Homebrew integration, automatic changelogs, and macOS code signing considerations.

## Basic Setup

Install GoReleaser:

```bash
brew install goreleaser
```

Create `.goreleaser.yaml`:

```yaml
version: 2

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
```

That's 6 binaries (3 OS × 2 architectures) in one configuration.

## Version Injection with ldflags

Go binaries can embed version info at build time:

```go
// main.go
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

func main() {
    if showVersion {
        fmt.Printf("%s version %s (commit: %s, built: %s)\n",
            appName, version, commit, date)
    }
}
```

GoReleaser sets these via ldflags:

```yaml
builds:
  - ldflags:
      - -s -w  # Strip debug info for smaller binary
      - -X main.version={{.Version}}
      - -X main.commit={{.ShortCommit}}
      - -X main.date={{.Date}}
```

When you run `goreleaser release --clean`, it builds with:

```bash
go build -ldflags "-s -w -X main.version=0.1.5 -X main.commit=abc123 -X main.date=2025-12-16"
```

## Archive Naming

Control the archive filenames:

```yaml
archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
```

This produces:
- `tnotify_0.1.5_darwin_amd64.tar.gz`
- `tnotify_0.1.5_darwin_arm64.tar.gz`
- `tnotify_0.1.5_linux_amd64.tar.gz`
- `tnotify_0.1.5_linux_arm64.tar.gz`
- `tnotify_0.1.5_windows_amd64.zip`
- `tnotify_0.1.5_windows_arm64.zip`

## Checksums

Always include checksums for verification:

```yaml
checksum:
  name_template: "checksums.txt"
  algorithm: sha256
```

Users can verify downloads:

```bash
sha256sum -c checksums.txt
```

## Homebrew Integration

GoReleaser can automatically update your Homebrew tap:

```yaml
brews:
  - name: tnotify
    repository:
      owner: soloterm
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    directory: Formula
    homepage: "https://github.com/soloterm/tnotify"
    description: "Send desktop notifications from the terminal"
    license: "MIT"
    install: |
      bin.install "tnotify"
    test: |
      system "#{bin}/tnotify", "--version"
```

When you release, GoReleaser:
1. Builds all binaries
2. Creates GitHub release with assets
3. Updates the Homebrew formula in your tap repo
4. Users can now `brew install soloterm/tap/tnotify`

### macOS Quarantine Removal

Downloaded binaries on macOS get quarantined. Users see "cannot be opened because the developer cannot be verified."

Add a post-install hook to remove quarantine:

```yaml
brews:
  - name: tnotify
    # ... other config ...
    post_install: |
      system_command "/usr/bin/xattr",
        args: ["-dr", "com.apple.quarantine", "#{bin}/tnotify"]
```

This runs `xattr -dr com.apple.quarantine /usr/local/bin/tnotify` after installation.

## Changelog Generation

GoReleaser can generate changelogs from git commits:

```yaml
changelog:
  sort: asc
  use: github
  filters:
    exclude:
      - "^docs:"
      - "^test:"
      - "^ci:"
      - "^chore:"
  groups:
    - title: Features
      regexp: "^feat"
    - title: Bug Fixes
      regexp: "^fix"
    - title: Others
      order: 999
```

Or use a pre-existing CHANGELOG.md (recommended for curated release notes):

```yaml
changelog:
  disable: true
```

Then use GitHub Actions to extract the relevant section:

```yaml
# .github/workflows/release.yaml
- name: Extract changelog
  id: changelog
  run: |
    VERSION=${GITHUB_REF#refs/tags/v}
    CONTENT=$(awk "/## \[${VERSION}\]/{flag=1; next} /## \[/{flag=0} flag" CHANGELOG.md)
    echo "content<<EOF" >> $GITHUB_OUTPUT
    echo "$CONTENT" >> $GITHUB_OUTPUT
    echo "EOF" >> $GITHUB_OUTPUT

- name: Run GoReleaser
  uses: goreleaser/goreleaser-action@v5
  with:
    args: release --clean --release-notes <(echo "${{ steps.changelog.outputs.content }}")
```

## GitHub Actions Integration

Full release workflow:

```yaml
# .github/workflows/release.yaml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

The `HOMEBREW_TAP_TOKEN` is a personal access token with repo access to your tap repository.

## Local Testing

Test your configuration without releasing:

```bash
# Check syntax
goreleaser check

# Build locally without publishing
goreleaser build --snapshot --clean

# Full release dry-run
goreleaser release --snapshot --clean
```

## Complete Configuration

Here's tnotify's full `.goreleaser.yaml`:

```yaml
version: 2

before:
  hooks:
    - go mod tidy
    - go test ./...

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.ShortCommit}}
      - -X main.date={{.Date}}

archives:
  - format: tar.gz
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"
  algorithm: sha256

brews:
  - name: tnotify
    repository:
      owner: soloterm
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    directory: Formula
    homepage: "https://github.com/soloterm/tnotify"
    description: "Send desktop notifications from the terminal"
    license: "MIT"
    install: |
      bin.install "tnotify"
    test: |
      system "#{bin}/tnotify", "--version"
    post_install: |
      system_command "/usr/bin/xattr",
        args: ["-dr", "com.apple.quarantine", "#{bin}/tnotify"]

changelog:
  disable: true

release:
  github:
    owner: soloterm
    name: tnotify
```

## Release Workflow

1. Update `CHANGELOG.md` with new version section
2. Commit: `git commit -am "Release v0.1.5"`
3. Tag: `git tag v0.1.5`
4. Push: `git push origin main --tags`
5. GitHub Actions runs GoReleaser
6. Binaries uploaded to GitHub Releases
7. Homebrew formula updated automatically

Users install with:

```bash
brew install soloterm/tap/tnotify
```

Or download binaries directly from GitHub Releases.

## Common Issues

### Token Permissions

The `GITHUB_TOKEN` needs `contents: write` permission:

```yaml
permissions:
  contents: write
```

### Homebrew Tap Updates Failing

Ensure `HOMEBREW_TAP_TOKEN` has repo access to the tap repository. Use a Fine-grained PAT with:
- Repository access: Only select repositories → your tap repo
- Permissions: Contents → Read and write

### Version Mismatch

GoReleaser extracts version from the git tag. If your code has a hardcoded version, it won't match:

```go
// Don't do this
var version = "0.1.4"

// Do this
var version = "dev"  // Overwritten by ldflags at build time
```

## Conclusion

GoReleaser handles the tedious parts of releasing:
- Cross-platform builds
- Version injection
- Archive creation
- GitHub release creation
- Homebrew formula updates
- Changelog generation

One config file, one command, six platforms.

Full configuration: [tnotify's .goreleaser.yaml](https://github.com/soloterm/tnotify/blob/main/.goreleaser.yaml)

## References

- [GoReleaser Documentation](https://goreleaser.com/)
- [GoReleaser GitHub Action](https://github.com/goreleaser/goreleaser-action)
- [Homebrew Formula Cookbook](https://docs.brew.sh/Formula-Cookbook)
