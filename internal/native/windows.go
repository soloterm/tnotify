package native

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"unicode/utf16"
)

// WindowsSender sends notifications via PowerShell toast notifications.
type WindowsSender struct {
	available *bool
}

// IsAvailable checks if PowerShell is available on Windows.
func (s *WindowsSender) IsAvailable() bool {
	if s.available != nil {
		return *s.available
	}

	// PowerShell is available by default on modern Windows
	avail := commandExists("powershell")
	s.available = &avail
	return avail
}

// Send sends a notification via PowerShell toast.
func (s *WindowsSender) Send(message, title string, urgency int) bool {
	if !s.IsAvailable() {
		return false
	}

	if title == "" {
		title = "Notification"
	}

	// Escape for PowerShell
	title = escapeForPowerShell(title)
	message = escapeForPowerShell(message)

	// Use Windows balloon notification (works without BurntToast module)
	script := fmt.Sprintf(`$ErrorActionPreference = 'SilentlyContinue'
Add-Type -AssemblyName System.Windows.Forms
$balloon = New-Object System.Windows.Forms.NotifyIcon
$balloon.Icon = [System.Drawing.SystemIcons]::Information
$balloon.BalloonTipTitle = "%s"
$balloon.BalloonTipText = "%s"
$balloon.Visible = $true
$balloon.ShowBalloonTip(5000)
Start-Sleep -Milliseconds 100
$balloon.Dispose()`, title, message)

	// Encode as base64 UTF-16LE for PowerShell
	encoded := encodeForPowerShell(script)

	cmd := exec.Command("powershell", "-EncodedCommand", encoded)
	err := cmd.Run()
	return err == nil
}

// escapeForPowerShell escapes a string for use in PowerShell.
func escapeForPowerShell(s string) string {
	s = strings.ReplaceAll(s, "`", "``")
	s = strings.ReplaceAll(s, "\"", "`\"")
	s = strings.ReplaceAll(s, "$", "`$")
	return s
}

// encodeForPowerShell encodes a script as base64 UTF-16LE for PowerShell -EncodedCommand.
func encodeForPowerShell(script string) string {
	// Convert to UTF-16LE
	runes := []rune(script)
	u16 := utf16.Encode(runes)

	// Convert to bytes (little-endian)
	bytes := make([]byte, len(u16)*2)
	for i, r := range u16 {
		bytes[i*2] = byte(r)
		bytes[i*2+1] = byte(r >> 8)
	}

	return base64.StdEncoding.EncodeToString(bytes)
}
