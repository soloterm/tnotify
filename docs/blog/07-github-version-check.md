# Version Checking via GitHub Releases Without Dependencies

CLI tools should tell users when updates are available. But adding an HTTP client library and JSON parser for one API call feels like overkill.

Here's a trick: GitHub's releases API redirects you to the latest version URL, and you can extract the version from the redirect location header—no JSON parsing needed.

## The Trick

When you request `https://github.com/OWNER/REPO/releases/latest`, GitHub returns a 302 redirect to the actual release page:

```
GET /soloterm/tnotify/releases/latest
302 Found
Location: https://github.com/soloterm/tnotify/releases/tag/v0.1.5
```

The version tag is right there in the URL: `v0.1.5`.

## Implementation

```go
func getLatestVersion() (string, error) {
    client := &http.Client{
        Timeout: 5 * time.Second,
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            // Don't follow redirects - we want the Location header
            return http.ErrUseLastResponse
        },
    }

    resp, err := client.Get("https://github.com/soloterm/tnotify/releases/latest")
    if err != nil {
        return "", fmt.Errorf("checking for updates: %w", err)
    }
    defer resp.Body.Close()

    // GitHub returns 302 redirect to /releases/tag/vX.Y.Z
    if resp.StatusCode != http.StatusFound {
        return "", fmt.Errorf("unexpected status: %d", resp.StatusCode)
    }

    location := resp.Header.Get("Location")
    // Location: https://github.com/soloterm/tnotify/releases/tag/v0.1.5

    parts := strings.Split(location, "/tag/")
    if len(parts) != 2 {
        return "", fmt.Errorf("unexpected redirect URL: %s", location)
    }

    return parts[1], nil // "v0.1.5"
}
```

The key is `http.ErrUseLastResponse` in the redirect checker. This tells the HTTP client to stop following redirects and return the response immediately.

## The Version Command

Here's how tnotify uses it:

```go
var (
    version = "dev"     // Set by goreleaser
    commit  = "none"
    date    = "unknown"
)

func showVersion() error {
    // Show current version
    fmt.Printf("tnotify %s\n", version)
    if commit != "none" {
        fmt.Printf("  commit: %s\n", commit)
    }
    if date != "unknown" {
        fmt.Printf("  built:  %s\n", date)
    }
    fmt.Println()

    // Check for updates (with timeout)
    fmt.Print("Checking for updates... ")

    latest, err := getLatestVersion()
    if err != nil {
        fmt.Printf("failed (%v)\n", err)
        return nil // Don't fail the command
    }

    // Compare versions
    currentClean := strings.TrimPrefix(version, "v")
    latestClean := strings.TrimPrefix(latest, "v")

    if currentClean == "dev" {
        fmt.Printf("development build\n")
        fmt.Printf("\n  Latest: %s\n", latest)
        return nil
    }

    cmp := compareVersion(currentClean, latestClean)
    if cmp >= 0 {
        fmt.Printf("you're up to date!\n")
        return nil
    }

    fmt.Printf("update available!\n")
    fmt.Printf("\n  Current: %s\n", version)
    fmt.Printf("  Latest:  %s\n", latest)
    fmt.Printf("\nUpdate with: brew upgrade tnotify\n")

    return nil
}
```

## Output Examples

Up to date:
```
$ tnotify --version
tnotify v0.1.5
  commit: abc123
  built:  2025-12-16

Checking for updates... you're up to date!
```

Update available:
```
$ tnotify --version
tnotify v0.1.4
  commit: def456
  built:  2025-12-15

Checking for updates... update available!

  Current: v0.1.4
  Latest:  v0.1.5

Update with: brew upgrade tnotify
```

Development build:
```
$ tnotify --version
tnotify dev
  commit: none
  built:  unknown

Checking for updates... development build

  Latest: v0.1.5
```

Network error (handled gracefully):
```
$ tnotify --version
tnotify v0.1.4
  commit: def456
  built:  2025-12-15

Checking for updates... failed (checking for updates: dial tcp: lookup github.com: no such host)
```

## Setting Version at Build Time

The version variables are set via ldflags during the build:

```go
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)
```

In `.goreleaser.yaml`:

```yaml
builds:
  - ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.ShortCommit}}
      - -X main.date={{.Date}}
```

For local builds:
```bash
go build -ldflags "-X main.version=dev -X main.commit=$(git rev-parse --short HEAD)"
```

## Why Not Use the GitHub API?

You could use `api.github.com/repos/OWNER/REPO/releases/latest` and parse the JSON:

```json
{
  "tag_name": "v0.1.5",
  "name": "v0.1.5",
  ...
}
```

But that requires:
1. JSON parsing
2. A struct definition
3. More error handling
4. API rate limits apply (60/hour unauthenticated)

The redirect trick:
1. No JSON parsing
2. No extra structs
3. Uses the same endpoint users see
4. No rate limits (it's a regular web page)

## Caching and Rate Limiting

The version check happens on every `--version` call. To be a good citizen:

1. **Short timeout**: We use 5 seconds. If GitHub is slow, don't block the user.
2. **Fail silently**: Network errors print a message but don't fail the command.
3. **No caching**: Each check is independent. For high-frequency tools, consider caching to a file.

For tools that run frequently (like build tools), add local caching:

```go
func getLatestVersionCached() (string, error) {
    cacheFile := filepath.Join(os.TempDir(), "tnotify-version-cache")

    // Check cache age
    info, err := os.Stat(cacheFile)
    if err == nil && time.Since(info.ModTime()) < 24*time.Hour {
        data, err := os.ReadFile(cacheFile)
        if err == nil {
            return string(data), nil
        }
    }

    // Fetch fresh
    version, err := getLatestVersion()
    if err != nil {
        return "", err
    }

    // Update cache (ignore errors)
    os.WriteFile(cacheFile, []byte(version), 0644)

    return version, nil
}
```

## Testing

The version check is tested manually (it hits the network), but you can unit test the redirect parsing:

```go
func TestParseRedirectURL(t *testing.T) {
    tests := []struct {
        location string
        want     string
        wantErr  bool
    }{
        {
            "https://github.com/soloterm/tnotify/releases/tag/v0.1.5",
            "v0.1.5",
            false,
        },
        {
            "https://github.com/soloterm/tnotify/releases/tag/v1.0.0-beta",
            "v1.0.0-beta",
            false,
        },
        {
            "https://example.com/other",
            "",
            true,
        },
    }

    for _, tt := range tests {
        got, err := parseVersionFromRedirect(tt.location)
        if (err != nil) != tt.wantErr {
            t.Errorf("parseVersionFromRedirect(%q) error = %v", tt.location, err)
        }
        if got != tt.want {
            t.Errorf("parseVersionFromRedirect(%q) = %q, want %q", tt.location, got, tt.want)
        }
    }
}

func parseVersionFromRedirect(location string) (string, error) {
    parts := strings.Split(location, "/tag/")
    if len(parts) != 2 {
        return "", fmt.Errorf("unexpected URL format")
    }
    return parts[1], nil
}
```

## Conclusion

Checking for updates doesn't require a dependency. The GitHub redirect trick gives you:

- Version checking in ~20 lines
- No JSON parsing
- No external libraries
- Graceful failure handling

The pattern works for any GitHub-hosted project. Just change the URL and you're done.

Full implementation: [tnotify's main.go](https://github.com/soloterm/tnotify/blob/main/main.go)
