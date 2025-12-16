package detect

import (
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// InTmux returns true if running inside tmux.
func InTmux() bool {
	return os.Getenv("TMUX") != ""
}

// InScreen returns true if running inside GNU Screen.
func InScreen() bool {
	return os.Getenv("STY") != ""
}

// InMultiplexer returns true if running inside any terminal multiplexer.
func InMultiplexer() bool {
	return InTmux() || InScreen()
}

// TmuxVersion returns the tmux version string, or empty if not available.
func TmuxVersion() string {
	if !InTmux() {
		return ""
	}

	cmd := exec.Command("tmux", "-V")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	// Output is like "tmux 3.3a" or "tmux 3.4"
	return strings.TrimSpace(string(out))
}

// TmuxAllowPassthrough checks if tmux allow-passthrough is enabled.
// Returns: "on", "off", "all", or empty string if unknown.
func TmuxAllowPassthrough() string {
	if !InTmux() {
		return ""
	}

	cmd := exec.Command("tmux", "show-options", "-gv", "allow-passthrough")
	out, err := cmd.Output()
	if err != nil {
		// Option might not exist in older tmux versions
		return ""
	}

	return strings.TrimSpace(string(out))
}

// TmuxSupportsPassthrough returns true if tmux version is 3.2 or higher.
func TmuxSupportsPassthrough() bool {
	version := TmuxVersion()
	if version == "" {
		return false
	}

	// Extract version number from "tmux X.Y" or "tmux X.Ya"
	re := regexp.MustCompile(`tmux\s+(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) < 3 {
		return false
	}

	major := 0
	minor := 0
	_, _ = parseSimpleInt(matches[1], &major)
	_, _ = parseSimpleInt(matches[2], &minor)

	// 3.2+ supports allow-passthrough
	return major > 3 || (major == 3 && minor >= 2)
}

// parseSimpleInt is a simple integer parser to avoid importing strconv.
func parseSimpleInt(s string, out *int) (bool, error) {
	*out = 0
	for _, c := range s {
		if c < '0' || c > '9' {
			break
		}
		*out = *out*10 + int(c-'0')
	}
	return true, nil
}
