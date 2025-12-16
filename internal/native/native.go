package native

import (
	"os/exec"
	"runtime"
)

// Sender is an interface for sending native notifications.
type Sender interface {
	IsAvailable() bool
	Send(message, title string, urgency int) bool
}

// GetSender returns the appropriate native notification sender for the current OS.
func GetSender() Sender {
	switch runtime.GOOS {
	case "darwin":
		return &MacOSSender{}
	case "linux":
		return &LinuxSender{}
	case "windows":
		return &WindowsSender{}
	default:
		return nil
	}
}

// IsAvailable returns true if native notifications are available.
func IsAvailable() bool {
	sender := GetSender()
	if sender == nil {
		return false
	}
	return sender.IsAvailable()
}

// Send sends a notification using the native notification system.
func Send(message, title string, urgency int) bool {
	sender := GetSender()
	if sender == nil {
		return false
	}
	return sender.Send(message, title, urgency)
}

// commandExists checks if a command exists in the system PATH.
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}
