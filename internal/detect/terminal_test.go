package detect

import (
	"os"
	"testing"
)

// envSnapshot saves and restores environment variables
type envSnapshot struct {
	vars map[string]string
}

func saveEnv(keys ...string) *envSnapshot {
	s := &envSnapshot{vars: make(map[string]string)}
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok {
			s.vars[k] = v
		}
	}
	return s
}

func (s *envSnapshot) restore() {
	for k, v := range s.vars {
		os.Setenv(k, v)
	}
}

func clearEnv(keys ...string) {
	for _, k := range keys {
		os.Unsetenv(k)
	}
}

var terminalEnvVars = []string{
	"KITTY_WINDOW_ID",
	"ITERM_SESSION_ID",
	"WEZTERM_PANE",
	"WT_SESSION",
	"ALACRITTY_WINDOW_ID",
	"KONSOLE_VERSION",
	"GHOSTTY_RESOURCES_DIR",
	"TERM_PROGRAM",
	"TERM_PROGRAM_VERSION",
	"VTE_VERSION",
}

func TestDetectTerminal(t *testing.T) {
	// Save original env
	snap := saveEnv(terminalEnvVars...)
	defer snap.restore()

	tests := []struct {
		name     string
		envSetup func()
		want     Terminal
	}{
		{
			name: "Kitty",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("KITTY_WINDOW_ID", "1")
			},
			want: TerminalKitty,
		},
		{
			name: "iTerm2 via session ID",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("ITERM_SESSION_ID", "w0t0p0:12345")
			},
			want: TerminalITerm2,
		},
		{
			name: "iTerm2 via TERM_PROGRAM",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("TERM_PROGRAM", "iTerm.app")
			},
			want: TerminalITerm2,
		},
		{
			name: "WezTerm via pane",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("WEZTERM_PANE", "0")
			},
			want: TerminalWezTerm,
		},
		{
			name: "WezTerm via TERM_PROGRAM",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("TERM_PROGRAM", "WezTerm")
			},
			want: TerminalWezTerm,
		},
		{
			name: "Windows Terminal",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("WT_SESSION", "abc-123")
			},
			want: TerminalWindowsTerminal,
		},
		{
			name: "Alacritty",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("ALACRITTY_WINDOW_ID", "12345")
			},
			want: TerminalAlacritty,
		},
		{
			name: "Konsole",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("KONSOLE_VERSION", "211200")
			},
			want: TerminalKonsole,
		},
		{
			name: "Ghostty via resources dir",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("GHOSTTY_RESOURCES_DIR", "/usr/share/ghostty")
			},
			want: TerminalGhostty,
		},
		{
			name: "Ghostty via TERM_PROGRAM",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("TERM_PROGRAM", "ghostty")
			},
			want: TerminalGhostty,
		},
		{
			name: "Apple Terminal",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("TERM_PROGRAM", "Apple_Terminal")
			},
			want: TerminalAppleTerminal,
		},
		{
			name: "VS Code",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("TERM_PROGRAM", "vscode")
			},
			want: TerminalVSCode,
		},
		{
			name: "Hyper",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("TERM_PROGRAM", "Hyper")
			},
			want: TerminalHyper,
		},
		{
			name: "VTE-based terminal",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("VTE_VERSION", "6800")
			},
			want: TerminalVTE,
		},
		{
			name: "Unknown terminal",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
			},
			want: TerminalUnknown,
		},
		{
			name: "Kitty takes precedence",
			envSetup: func() {
				clearEnv(terminalEnvVars...)
				os.Setenv("KITTY_WINDOW_ID", "1")
				os.Setenv("TERM_PROGRAM", "WezTerm") // should be ignored
			},
			want: TerminalKitty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.envSetup()
			got := DetectTerminal()
			if got != tt.want {
				t.Errorf("DetectTerminal() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSelectProtocol(t *testing.T) {
	tests := []struct {
		terminal Terminal
		want     Protocol
	}{
		{TerminalKitty, ProtocolOSC99},
		{TerminalITerm2, ProtocolOSC9},
		{TerminalWezTerm, ProtocolOSC777},
		{TerminalGhostty, ProtocolOSC777},
		{TerminalVTE, ProtocolOSC777},
		{TerminalFoot, ProtocolOSC777},
		{TerminalAlacritty, ProtocolNone},
		{TerminalKonsole, ProtocolNone},
		{TerminalAppleTerminal, ProtocolNone},
		{TerminalVSCode, ProtocolNone},
		{TerminalWindowsTerminal, ProtocolNone},
		{TerminalHyper, ProtocolNone},
		{TerminalUnknown, ProtocolNone},
	}

	for _, tt := range tests {
		t.Run(string(tt.terminal), func(t *testing.T) {
			got := SelectProtocol(tt.terminal)
			if got != tt.want {
				t.Errorf("SelectProtocol(%q) = %q, want %q", tt.terminal, got, tt.want)
			}
		})
	}
}

func TestGetCapabilities(t *testing.T) {
	snap := saveEnv(terminalEnvVars...)
	defer snap.restore()

	t.Run("Kitty capabilities", func(t *testing.T) {
		clearEnv(terminalEnvVars...)
		os.Setenv("KITTY_WINDOW_ID", "1")

		caps := GetCapabilities(true)

		if caps.Terminal != TerminalKitty {
			t.Errorf("Terminal = %q, want %q", caps.Terminal, TerminalKitty)
		}
		if caps.Protocol != ProtocolOSC99 {
			t.Errorf("Protocol = %q, want %q", caps.Protocol, ProtocolOSC99)
		}
		if !caps.SupportsTitle {
			t.Error("SupportsTitle should be true for OSC99")
		}
		if !caps.SupportsUrgency {
			t.Error("SupportsUrgency should be true for OSC99")
		}
		if !caps.SupportsID {
			t.Error("SupportsID should be true for OSC99")
		}
		if !caps.NativeAvailable {
			t.Error("NativeAvailable should be true when passed true")
		}
	})

	t.Run("iTerm2 capabilities", func(t *testing.T) {
		clearEnv(terminalEnvVars...)
		os.Setenv("ITERM_SESSION_ID", "session")

		caps := GetCapabilities(false)

		if caps.Protocol != ProtocolOSC9 {
			t.Errorf("Protocol = %q, want %q", caps.Protocol, ProtocolOSC9)
		}
		if caps.SupportsTitle {
			t.Error("SupportsTitle should be false for OSC9")
		}
		if caps.SupportsUrgency {
			t.Error("SupportsUrgency should be false for OSC9")
		}
		if caps.NativeAvailable {
			t.Error("NativeAvailable should be false when passed false")
		}
	})

	t.Run("WezTerm capabilities", func(t *testing.T) {
		clearEnv(terminalEnvVars...)
		os.Setenv("WEZTERM_PANE", "0")

		caps := GetCapabilities(true)

		if caps.Protocol != ProtocolOSC777 {
			t.Errorf("Protocol = %q, want %q", caps.Protocol, ProtocolOSC777)
		}
		if !caps.SupportsTitle {
			t.Error("SupportsTitle should be true for OSC777")
		}
		if caps.SupportsUrgency {
			t.Error("SupportsUrgency should be false for OSC777")
		}
		if caps.SupportsID {
			t.Error("SupportsID should be false for OSC777")
		}
	})
}

func TestCompareVersion(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"3.6.6", "3.6.6", 0},
		{"3.6.7", "3.6.6", 1},
		{"3.6.5", "3.6.6", -1},
		{"3.7.0", "3.6.6", 1},
		{"3.5.10", "3.6.6", -1},
		{"4.0.0", "3.6.6", 1},
		{"2.9.9", "3.6.6", -1},
		{"3.6.6-beta", "3.6.6", 0},
		{"3.6.7-rc1", "3.6.6", 1},
		{"", "3.6.6", -1},
		{"3.6", "3.6.6", -1},
		{"3.6.6.1", "3.6.6", 1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got := compareVersion(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("compareVersion(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestSupportsProgress(t *testing.T) {
	snap := saveEnv(terminalEnvVars...)
	defer snap.restore()

	t.Run("Ghostty supports progress", func(t *testing.T) {
		if !SupportsProgress(TerminalGhostty) {
			t.Error("Ghostty should support progress")
		}
	})

	t.Run("Windows Terminal supports progress", func(t *testing.T) {
		if !SupportsProgress(TerminalWindowsTerminal) {
			t.Error("Windows Terminal should support progress")
		}
	})

	t.Run("iTerm2 3.6.6+ supports progress", func(t *testing.T) {
		clearEnv(terminalEnvVars...)
		os.Setenv("TERM_PROGRAM_VERSION", "3.6.6")
		if !SupportsProgress(TerminalITerm2) {
			t.Error("iTerm2 3.6.6 should support progress")
		}
	})

	t.Run("iTerm2 3.6.7 supports progress", func(t *testing.T) {
		clearEnv(terminalEnvVars...)
		os.Setenv("TERM_PROGRAM_VERSION", "3.6.7")
		if !SupportsProgress(TerminalITerm2) {
			t.Error("iTerm2 3.6.7 should support progress")
		}
	})

	t.Run("iTerm2 3.6.5 does not support progress", func(t *testing.T) {
		clearEnv(terminalEnvVars...)
		os.Setenv("TERM_PROGRAM_VERSION", "3.6.5")
		if SupportsProgress(TerminalITerm2) {
			t.Error("iTerm2 3.6.5 should not support progress")
		}
	})

	t.Run("iTerm2 without version does not support progress", func(t *testing.T) {
		clearEnv(terminalEnvVars...)
		if SupportsProgress(TerminalITerm2) {
			t.Error("iTerm2 without version should not support progress")
		}
	})

	t.Run("WezTerm does not support progress", func(t *testing.T) {
		if SupportsProgress(TerminalWezTerm) {
			t.Error("WezTerm should not support progress")
		}
	})

	t.Run("Unknown terminal does not support progress", func(t *testing.T) {
		if SupportsProgress(TerminalUnknown) {
			t.Error("Unknown terminal should not support progress")
		}
	})
}
