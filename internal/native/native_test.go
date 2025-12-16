package native

import (
	"runtime"
	"testing"
)

func TestGetSender(t *testing.T) {
	sender := GetSender()

	switch runtime.GOOS {
	case "darwin":
		if _, ok := sender.(*MacOSSender); !ok {
			t.Errorf("GetSender() on darwin should return *MacOSSender, got %T", sender)
		}
	case "linux":
		if _, ok := sender.(*LinuxSender); !ok {
			t.Errorf("GetSender() on linux should return *LinuxSender, got %T", sender)
		}
	case "windows":
		if _, ok := sender.(*WindowsSender); !ok {
			t.Errorf("GetSender() on windows should return *WindowsSender, got %T", sender)
		}
	default:
		if sender != nil {
			t.Errorf("GetSender() on %s should return nil, got %T", runtime.GOOS, sender)
		}
	}
}

func TestCommandExists(t *testing.T) {
	// These commands should exist on most systems
	t.Run("existing command", func(t *testing.T) {
		// 'ls' exists on Unix, 'cmd' exists on Windows
		var cmd string
		if runtime.GOOS == "windows" {
			cmd = "cmd"
		} else {
			cmd = "ls"
		}

		if !commandExists(cmd) {
			t.Errorf("commandExists(%q) = false, want true", cmd)
		}
	})

	t.Run("non-existing command", func(t *testing.T) {
		if commandExists("definitely-not-a-real-command-12345") {
			t.Error("commandExists() = true for non-existing command")
		}
	})
}

func TestEscapeAppleScript(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "double quotes",
			input: `Say "Hello"`,
			want:  `Say \"Hello\"`,
		},
		{
			name:  "backslashes",
			input: `path\to\file`,
			want:  `path\\to\\file`,
		},
		{
			name:  "mixed",
			input: `C:\Users\"Admin"`,
			want:  `C:\\Users\\\"Admin\"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeAppleScript(tt.input)
			if got != tt.want {
				t.Errorf("escapeAppleScript(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeForPowerShell(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "plain text",
			input: "Hello World",
			want:  "Hello World",
		},
		{
			name:  "double quotes",
			input: `Say "Hello"`,
			want:  "Say `\"Hello`\"",
		},
		{
			name:  "backticks",
			input: "Use `command`",
			want:  "Use ``command``",
		},
		{
			name:  "dollar sign",
			input: "Cost is $100",
			want:  "Cost is `$100",
		},
		{
			name:  "mixed",
			input: `$var = "test"`,
			want:  "`$var = `\"test`\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := escapeForPowerShell(tt.input)
			if got != tt.want {
				t.Errorf("escapeForPowerShell(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEncodeForPowerShell(t *testing.T) {
	// Test that encoding produces valid base64
	t.Run("simple script", func(t *testing.T) {
		script := "Write-Host Hello"
		encoded := encodeForPowerShell(script)

		// Should be valid base64
		if encoded == "" {
			t.Error("encodeForPowerShell() returned empty string")
		}

		// Base64 should only contain valid characters
		for _, c := range encoded {
			if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '+' || c == '/' || c == '=') {
				t.Errorf("encodeForPowerShell() produced invalid base64 character: %c", c)
			}
		}
	})

	t.Run("unicode script", func(t *testing.T) {
		script := "Write-Host '世界'"
		encoded := encodeForPowerShell(script)

		if encoded == "" {
			t.Error("encodeForPowerShell() returned empty string for unicode")
		}
	})
}

func TestMacOSSender_IsAvailable(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Skipping macOS-specific test")
	}

	sender := &MacOSSender{}

	// osascript should be available on macOS
	if !sender.IsAvailable() {
		t.Error("MacOSSender.IsAvailable() = false on macOS, want true")
	}

	// Test caching
	first := sender.IsAvailable()
	second := sender.IsAvailable()
	if first != second {
		t.Error("MacOSSender.IsAvailable() returned different values on subsequent calls")
	}
}

func TestLinuxSender_IsAvailable(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Skipping Linux-specific test")
	}

	sender := &LinuxSender{}

	// Just test that it doesn't panic and returns a boolean
	// notify-send may or may not be installed
	_ = sender.IsAvailable()

	// Test caching
	first := sender.IsAvailable()
	second := sender.IsAvailable()
	if first != second {
		t.Error("LinuxSender.IsAvailable() returned different values on subsequent calls")
	}
}

func TestWindowsSender_IsAvailable(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows-specific test")
	}

	sender := &WindowsSender{}

	// PowerShell should be available on modern Windows
	if !sender.IsAvailable() {
		t.Error("WindowsSender.IsAvailable() = false on Windows, want true")
	}

	// Test caching
	first := sender.IsAvailable()
	second := sender.IsAvailable()
	if first != second {
		t.Error("WindowsSender.IsAvailable() returned different values on subsequent calls")
	}
}
