package osc

import (
	"strings"

	"github.com/soloterm/tnotify/internal/detect"
)

// WrapForMultiplexer wraps an OSC sequence for multiplexer passthrough if needed.
func WrapForMultiplexer(sequence string) string {
	if detect.InTmux() {
		return wrapForTmux(sequence)
	}
	if detect.InScreen() {
		return wrapForScreen(sequence)
	}
	return sequence
}

// wrapForTmux wraps a sequence for tmux passthrough.
// Format: DCS tmux ; doubled_sequence ST
// All ESC characters inside must be doubled.
func wrapForTmux(sequence string) string {
	// Double all ESC characters
	doubled := strings.ReplaceAll(sequence, "\x1b", "\x1b\x1b")
	return "\x1bPtmux;" + doubled + "\x1b\\"
}

// wrapForScreen wraps a sequence for GNU Screen passthrough.
// Format: DCS sequence ST
func wrapForScreen(sequence string) string {
	return "\x1bP" + sequence + "\x1b\\"
}
