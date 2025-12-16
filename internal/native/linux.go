package native

import (
	"os/exec"
)

// LinuxSender sends notifications via notify-send (libnotify).
type LinuxSender struct {
	available *bool
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

	cmd := exec.Command("notify-send", "-u", urgencyLevel, title, message)
	err := cmd.Run()
	return err == nil
}
