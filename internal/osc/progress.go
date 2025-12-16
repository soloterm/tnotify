package osc

import "fmt"

// Progress states for OSC 9;4
const (
	ProgressHidden       = 0 // Remove progress indicator
	ProgressNormal       = 1 // Normal/green progress
	ProgressError        = 2 // Error/red progress
	ProgressIndeterminate = 3 // Indeterminate/pulsing
	ProgressPaused       = 4 // Paused/yellow progress
)

// BuildOSC9Progress builds an OSC 9;4 progress sequence.
// Format: ESC ] 9 ; 4 ; state ; progress BEL
// Supported by Windows Terminal, Ghostty, ConEmu, Mintty.
func BuildOSC9Progress(state, progress int) string {
	// Clamp progress to 0-100
	if progress < 0 {
		progress = 0
	}
	if progress > 100 {
		progress = 100
	}

	// Clamp state to valid range
	if state < 0 || state > 4 {
		state = ProgressNormal
	}

	return fmt.Sprintf("\x1b]9;4;%d;%d\x07", state, progress)
}

// BuildOSC9ProgressClear builds an OSC 9;4 sequence to clear progress.
func BuildOSC9ProgressClear() string {
	return BuildOSC9Progress(ProgressHidden, 0)
}
