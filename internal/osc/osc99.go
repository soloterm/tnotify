package osc

import (
	"fmt"
	"strings"
)

// Urgency levels for OSC 99 notifications.
const (
	UrgencyLow      = 0
	UrgencyNormal   = 1
	UrgencyCritical = 2
)

// BuildOSC99 builds an OSC 99 notification sequence (Kitty style, full-featured).
// Format: ESC ] 99 ; metadata ; payload ST
// For simple notifications: ESC ] 99 ; ; message ST
// For title + body: uses d=0/1 for done flag and p=title/body
func BuildOSC99(message, title string, urgency int, id string) string {
	// Clamp urgency to valid range
	if urgency < 0 {
		urgency = 0
	}
	if urgency > 2 {
		urgency = 2
	}

	// Sanitize ID if provided
	idMeta := ""
	if id != "" {
		id = SanitizeID(id)
		if id != "" {
			idMeta = "i=" + id
		}
	}

	if title == "" {
		// Simple notification with just a body
		message = Sanitize(message)

		// Build metadata: ID and urgency (if not normal)
		var metaParts []string
		if idMeta != "" {
			metaParts = append(metaParts, idMeta)
		}
		if urgency != UrgencyNormal {
			metaParts = append(metaParts, fmt.Sprintf("u=%d", urgency))
		}
		metadata := strings.Join(metaParts, ":")

		return fmt.Sprintf("\x1b]99;%s;%s\x1b\\", metadata, message)
	}

	// Multi-part notification with title and body
	title = Sanitize(title)
	message = Sanitize(message)

	// d=0 means "more parts coming", d=1 means "done"
	// p=title means this is the title, p=body means this is the body
	// i=<id> sets notification ID
	// u=N sets urgency level
	titleMeta := "d=0:p=title"
	if idMeta != "" {
		titleMeta += ":" + idMeta
	}
	if urgency != UrgencyNormal {
		titleMeta += fmt.Sprintf(":u=%d", urgency)
	}

	return fmt.Sprintf("\x1b]99;%s;%s\x1b\\\x1b]99;d=1:p=body;%s\x1b\\", titleMeta, title, message)
}

// BuildOSC99Close builds an OSC 99 close notification sequence.
// Format: ESC ] 99 ; i=<id>:p=close ; ST
func BuildOSC99Close(id string) string {
	id = SanitizeID(id)
	if id == "" {
		return ""
	}
	return fmt.Sprintf("\x1b]99;i=%s:p=close;\x1b\\", id)
}
