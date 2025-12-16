package native

import (
	"fmt"
	"os/exec"
	"strings"
)

// MacOSSender sends notifications via osascript (AppleScript).
type MacOSSender struct {
	available *bool
}

// IsAvailable checks if osascript is available.
func (s *MacOSSender) IsAvailable() bool {
	if s.available != nil {
		return *s.available
	}

	avail := commandExists("osascript")
	s.available = &avail
	return avail
}

// Send sends a notification via osascript.
func (s *MacOSSender) Send(message, title string, urgency int) bool {
	if !s.IsAvailable() {
		return false
	}

	if title == "" {
		title = "Notification"
	}

	// Escape for AppleScript
	title = escapeAppleScript(title)
	message = escapeAppleScript(message)

	script := fmt.Sprintf(`display notification "%s" with title "%s"`, message, title)

	cmd := exec.Command("osascript", "-e", script)
	err := cmd.Run()
	return err == nil
}

// escapeAppleScript escapes a string for use in AppleScript.
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}
