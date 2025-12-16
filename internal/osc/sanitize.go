package osc

import (
	"regexp"
	"strings"
)

var (
	// controlCharsRegex matches control characters (except space)
	controlCharsRegex = regexp.MustCompile(`[\x00-\x1f\x7f]`)

	// idSanitizeRegex matches invalid ID characters
	idSanitizeRegex = regexp.MustCompile(`[^a-zA-Z0-9_\-+.]`)
)

// Sanitize removes control characters and semicolons from a string.
// Semicolons are replaced with colons to avoid breaking OSC 777 parsing.
func Sanitize(s string) string {
	// Remove control characters
	s = controlCharsRegex.ReplaceAllString(s, "")

	// Replace semicolons with colons to avoid breaking OSC 777 parsing
	s = strings.ReplaceAll(s, ";", ":")

	return s
}

// SanitizeID sanitizes a notification ID for OSC 99.
// IDs must only contain [a-zA-Z0-9_-+.] characters.
func SanitizeID(id string) string {
	return idSanitizeRegex.ReplaceAllString(id, "")
}
