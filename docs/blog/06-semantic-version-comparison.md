# Semantic Version Comparison in 30 Lines of Go

When you need to check if a version is "at least 3.6.6", you could reach for a semver library. But for simple cases, a 30-line function does the job without dependencies.

## The Problem

iTerm2 added progress bar support in version 3.6.6. We need to check:

```go
if compareVersion(os.Getenv("TERM_PROGRAM_VERSION"), "3.6.6") >= 0 {
    // Use progress bars
}
```

The version string might be:
- `3.6.6` (exact match)
- `3.6.7` (newer)
- `3.6.5` (older)
- `3.6.6-beta` (pre-release)
- `3.7.0` (newer minor)
- `4.0.0` (newer major)
- `3.6` (missing patch)
- `` (empty/missing)

## The Solution

```go
// compareVersion compares two semantic version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareVersion(a, b string) int {
    // Strip any suffix after hyphen (e.g., "3.6.6-beta" -> "3.6.6")
    a = strings.Split(a, "-")[0]
    b = strings.Split(b, "-")[0]

    partsA := strings.Split(a, ".")
    partsB := strings.Split(b, ".")

    // Compare each component
    maxLen := len(partsA)
    if len(partsB) > maxLen {
        maxLen = len(partsB)
    }

    for i := 0; i < maxLen; i++ {
        var numA, numB int
        if i < len(partsA) {
            numA, _ = strconv.Atoi(partsA[i])
        }
        if i < len(partsB) {
            numB, _ = strconv.Atoi(partsB[i])
        }

        if numA < numB {
            return -1
        }
        if numA > numB {
            return 1
        }
    }

    return 0
}
```

Let's break down the key decisions.

## Design Decisions

### 1. Strip Pre-Release Suffixes

```go
a = strings.Split(a, "-")[0]
```

`3.6.6-beta` becomes `3.6.6`. This treats pre-releases as equal to their release version, which is acceptable for our use case (feature detection).

For strict semver compliance, pre-release versions should sort before their release. But that adds complexity we don't need.

### 2. Handle Missing Components

```go
if i < len(partsA) {
    numA, _ = strconv.Atoi(partsA[i])
}
// numA defaults to 0 if index out of range
```

Comparing `3.6` to `3.6.6`:
- `3.6` is treated as `3.6.0`
- `3.6.0` < `3.6.6`, so `3.6` is older

### 3. Ignore Parsing Errors

```go
numA, _ = strconv.Atoi(partsA[i])
```

Invalid components parse to 0. Comparing `3.6.abc` to `3.6.6`:
- `abc` parses to 0
- Result: `3.6.0` < `3.6.6`

This is defensive—we never crash on weird input.

### 4. Handle Empty Strings

Empty string splits to `[""]`, which parses to `[0]`:
- `""` vs `3.6.6` → `0.0.0` vs `3.6.6` → -1

Empty/missing version is always "older" than any real version.

## Test Cases

```go
func TestCompareVersion(t *testing.T) {
    tests := []struct {
        a, b string
        want int
    }{
        // Equal versions
        {"3.6.6", "3.6.6", 0},

        // Patch version differences
        {"3.6.7", "3.6.6", 1},
        {"3.6.5", "3.6.6", -1},

        // Minor version differences
        {"3.7.0", "3.6.6", 1},
        {"3.5.10", "3.6.6", -1},

        // Major version differences
        {"4.0.0", "3.6.6", 1},
        {"2.9.9", "3.6.6", -1},

        // Pre-release versions (treated as equal to release)
        {"3.6.6-beta", "3.6.6", 0},
        {"3.6.7-rc1", "3.6.6", 1},

        // Missing components
        {"", "3.6.6", -1},
        {"3.6", "3.6.6", -1},
        {"3.6.6.1", "3.6.6", 1}, // Extra component
    }

    for _, tt := range tests {
        t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
            got := compareVersion(tt.a, tt.b)
            if got != tt.want {
                t.Errorf("compareVersion(%q, %q) = %d, want %d",
                    tt.a, tt.b, got, tt.want)
            }
        })
    }
}
```

## Usage Pattern

The "greater than or equal to" check:

```go
func SupportsProgress(terminal Terminal) bool {
    switch terminal {
    case TerminalWindowsTerminal, TerminalGhostty:
        return true
    case TerminalITerm2:
        version := os.Getenv("TERM_PROGRAM_VERSION")
        return compareVersion(version, "3.6.6") >= 0
    default:
        return false
    }
}
```

The tmux version check:

```go
func TmuxSupportsPassthrough() bool {
    output, err := exec.Command("tmux", "-V").Output()
    if err != nil {
        return false
    }

    // Output: "tmux 3.3a" or "tmux 3.2"
    version := strings.TrimPrefix(strings.TrimSpace(string(output)), "tmux ")

    // Passthrough was added in tmux 3.2
    return compareVersion(version, "3.2") >= 0
}
```

## When to Use a Library

This simple function works for:
- Basic version comparisons (>, <, ==, >=, <=)
- Feature detection ("is version at least X?")
- Versions with 2-4 numeric components

Use a proper semver library when you need:
- Strict pre-release ordering (`1.0.0-alpha` < `1.0.0-beta` < `1.0.0`)
- Build metadata handling (`1.0.0+build123`)
- Version range parsing (`>=1.2.3 <2.0.0`)
- Validation that versions conform to semver spec

Popular Go semver libraries:
- `golang.org/x/mod/semver`
- `github.com/Masterminds/semver`
- `github.com/blang/semver`

## Conclusion

For simple version comparisons in feature detection, 30 lines of Go beats adding a dependency. The key insight is handling edge cases defensively:

1. **Strip suffixes**: `-beta`, `-rc1` become their base version
2. **Default to zero**: Missing components are 0
3. **Ignore errors**: Unparseable components are 0
4. **Empty is oldest**: Empty string loses every comparison

The full implementation is in [tnotify's detect package](https://github.com/soloterm/tnotify/blob/main/internal/detect/terminal.go).
