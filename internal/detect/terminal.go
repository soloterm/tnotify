package detect

import "os"

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

// Capabilities describes terminal notification capabilities.
type Capabilities struct {
	Terminal         Terminal `json:"terminal"`
	Protocol         Protocol `json:"protocol"`
	SupportsTitle    bool     `json:"supports_title"`
	SupportsUrgency  bool     `json:"supports_urgency"`
	SupportsID       bool     `json:"supports_id"`
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
		InMultiplexer:    InTmux() || InScreen(),
		NativeAvailable:  nativeAvailable,
	}
}
