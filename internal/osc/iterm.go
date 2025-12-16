package osc

// BuildITermRequestAttention builds an iTerm2 OSC 1337 RequestAttention sequence.
// This bounces the dock icon on macOS to get the user's attention.
// fireworks: if true, shows a brief fireworks animation.
func BuildITermRequestAttention(fireworks bool) string {
	value := "yes"
	if fireworks {
		value = "fireworks"
	}
	return "\x1b]1337;RequestAttention=" + value + "\x07"
}

// BuildITermStealFocus builds an iTerm2 OSC 1337 StealFocus sequence.
// This brings the iTerm2 window to the front.
func BuildITermStealFocus() string {
	return "\x1b]1337;StealFocus\x07"
}
