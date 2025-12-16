package detect

import (
	"os"
	"strconv"
	"strings"
)

// Terminal represents a detected terminal emulator.
type Terminal string

const (
	TerminalKitty           Terminal = "kitty"
	TerminalITerm2          Terminal = "iterm2"
	TerminalWezTerm         Terminal = "wezterm"
	TerminalGhostty         Terminal = "ghostty"
	TerminalVTE             Terminal = "vte"
	TerminalFoot            Terminal = "foot"
	TerminalAlacritty       Terminal = "alacritty"
	TerminalKonsole         Terminal = "konsole"
	TerminalAppleTerminal   Terminal = "apple-terminal"
	TerminalWindowsTerminal Terminal = "windows-terminal"
	TerminalVSCode          Terminal = "vscode"
	TerminalHyper           Terminal = "hyper"
	TerminalUnknown         Terminal = ""
)

// Protocol represents an OSC notification protocol.
type Protocol string

const (
	ProtocolOSC9   Protocol = "osc9"   // iTerm2 style - message only
	ProtocolOSC777 Protocol = "osc777" // rxvt/WezTerm/Ghostty - title + body
	ProtocolOSC99  Protocol = "osc99"  // Kitty - full featured (urgency, IDs)
	ProtocolNone   Protocol = ""
)

// DetectTerminal detects the current terminal emulator from environment variables.
func DetectTerminal() Terminal {
	// Fast path: terminal-specific environment variables

	// Kitty
	if os.Getenv("KITTY_WINDOW_ID") != "" {
		return TerminalKitty
	}

	// iTerm2
	if os.Getenv("ITERM_SESSION_ID") != "" {
		return TerminalITerm2
	}

	// WezTerm
	if os.Getenv("WEZTERM_PANE") != "" {
		return TerminalWezTerm
	}

	// Windows Terminal
	if os.Getenv("WT_SESSION") != "" {
		return TerminalWindowsTerminal
	}

	// Alacritty
	if os.Getenv("ALACRITTY_WINDOW_ID") != "" {
		return TerminalAlacritty
	}

	// Konsole
	if os.Getenv("KONSOLE_VERSION") != "" {
		return TerminalKonsole
	}

	// Ghostty
	if os.Getenv("GHOSTTY_RESOURCES_DIR") != "" {
		return TerminalGhostty
	}

	// Check TERM_PROGRAM
	termProgram := os.Getenv("TERM_PROGRAM")
	switch termProgram {
	case "iTerm.app":
		return TerminalITerm2
	case "WezTerm":
		return TerminalWezTerm
	case "Apple_Terminal":
		return TerminalAppleTerminal
	case "vscode":
		return TerminalVSCode
	case "Hyper":
		return TerminalHyper
	case "ghostty":
		return TerminalGhostty
	}

	// VTE-based terminals (GNOME Terminal, etc.)
	if os.Getenv("VTE_VERSION") != "" {
		return TerminalVTE
	}

	return TerminalUnknown
}

// SelectProtocol returns the best OSC protocol for a terminal.
func SelectProtocol(terminal Terminal) Protocol {
	switch terminal {
	// OSC 99 (Kitty protocol - most feature-rich)
	case TerminalKitty:
		return ProtocolOSC99

	// OSC 9 (iTerm2 style - simple but widely supported)
	case TerminalITerm2:
		return ProtocolOSC9

	// OSC 777 (supports title + body)
	case TerminalWezTerm, TerminalGhostty, TerminalVTE, TerminalFoot:
		return ProtocolOSC777

	// Terminals with no notification support
	case TerminalAlacritty, TerminalKonsole, TerminalAppleTerminal,
		TerminalVSCode, TerminalWindowsTerminal:
		return ProtocolNone

	default:
		return ProtocolNone
	}
}

// SupportsProgress returns true if the terminal supports OSC 9;4 progress bars.
// Supported by: Windows Terminal, Ghostty (1.2+), iTerm2 (3.6.6+), ConEmu, Mintty
func SupportsProgress(terminal Terminal) bool {
	switch terminal {
	case TerminalWindowsTerminal, TerminalGhostty:
		return true
	case TerminalITerm2:
		// iTerm2 added OSC 9;4 support in version 3.6.6
		return compareVersion(os.Getenv("TERM_PROGRAM_VERSION"), "3.6.6") >= 0
	default:
		return false
	}
}

// compareVersion compares two semantic version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Handles versions like "3.6.6" or "3.6.6-beta".
func compareVersion(a, b string) int {
	// Strip any suffix after hyphen (e.g., "3.6.6-beta" -> "3.6.6")
	a = strings.Split(a, "-")[0]
	b = strings.Split(b, "-")[0]

	partsA := strings.Split(a, ".")
	partsB := strings.Split(b, ".")

	// Compare each component
	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var numA, numB int
		if i < len(partsA) {
			numA, _ = strconv.Atoi(partsA[i])
		}
		if i < len(partsB) {
			numB, _ = strconv.Atoi(partsB[i])
		}

		if numA < numB {
			return -1
		}
		if numA > numB {
			return 1
		}
	}

	return 0
}

// Capabilities describes terminal notification capabilities.
type Capabilities struct {
	Terminal         Terminal `json:"terminal"`
	Protocol         Protocol `json:"protocol"`
	SupportsTitle    bool     `json:"supports_title"`
	SupportsUrgency  bool     `json:"supports_urgency"`
	SupportsID       bool     `json:"supports_id"`
	SupportsProgress bool     `json:"supports_progress"`
	InMultiplexer    bool     `json:"in_multiplexer"`
	NativeAvailable  bool     `json:"native_available"`
}

// GetCapabilities returns the notification capabilities for the current environment.
func GetCapabilities(nativeAvailable bool) Capabilities {
	terminal := DetectTerminal()
	protocol := SelectProtocol(terminal)

	return Capabilities{
		Terminal:         terminal,
		Protocol:         protocol,
		SupportsTitle:    protocol == ProtocolOSC777 || protocol == ProtocolOSC99,
		SupportsUrgency:  protocol == ProtocolOSC99,
		SupportsID:       protocol == ProtocolOSC99,
		SupportsProgress: SupportsProgress(terminal),
		InMultiplexer:    InTmux() || InScreen(),
		NativeAvailable:  nativeAvailable,
	}
}
