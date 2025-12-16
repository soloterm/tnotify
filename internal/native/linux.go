package native

import (
	"os/exec"
	"strconv"
)

// LinuxSender sends notifications via notify-send (libnotify).
type LinuxSender struct {
	available *bool
	appName   string
	icon      string
	timeout   int // milliseconds, 0 = default, -1 = never expire
}

// IsAvailable checks if notify-send is available.
func (s *LinuxSender) IsAvailable() bool {
	if s.available != nil {
		return *s.available
	}

	avail := commandExists("notify-send")
	s.available = &avail
	return avail
}

// SetAppName sets the application name for notifications.
func (s *LinuxSender) SetAppName(name string) {
	s.appName = name
}

// SetIcon sets the icon for notifications.
// Can be a stock icon name (e.g., "dialog-information") or a file path.
func (s *LinuxSender) SetIcon(icon string) {
	s.icon = icon
}

// SetTimeout sets the notification timeout in milliseconds.
// Use 0 for default, -1 for never expire.
func (s *LinuxSender) SetTimeout(ms int) {
	s.timeout = ms
}

// Send sends a notification via notify-send.
func (s *LinuxSender) Send(message, title string, urgency int) bool {
	if !s.IsAvailable() {
		return false
	}

	if title == "" {
		title = "Notification"
	}

	// Map urgency to notify-send levels: low, normal, critical
	urgencyLevel := "normal"
	switch urgency {
	case 0:
		urgencyLevel = "low"
	case 2:
		urgencyLevel = "critical"
	}

	args := []string{"-u", urgencyLevel}

	// Add app name if set
	if s.appName != "" {
		args = append(args, "-a", s.appName)
	} else {
		args = append(args, "-a", "tnotify")
	}

	// Add icon if set
	if s.icon != "" {
		args = append(args, "-i", s.icon)
	} else {
		args = append(args, "-i", "utilities-terminal")
	}

	// Add timeout if set
	if s.timeout != 0 {
		args = append(args, "-t", strconv.Itoa(s.timeout))
	}

	args = append(args, title, message)

	cmd := exec.Command("notify-send", args...)
	err := cmd.Run()
	return err == nil
}
