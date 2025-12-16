package detect

import "os"

// InTmux returns true if running inside tmux.
func InTmux() bool {
	return os.Getenv("TMUX") != ""
}

// InScreen returns true if running inside GNU Screen.
func InScreen() bool {
	return os.Getenv("STY") != ""
}

// InMultiplexer returns true if running inside any terminal multiplexer.
func InMultiplexer() bool {
	return InTmux() || InScreen()
}
