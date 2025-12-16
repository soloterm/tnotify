package osc

// Shell integration markers (OSC 133)
// These help terminals understand command boundaries.
// Supported by: Windows Terminal, WezTerm, VS Code terminal, kitty, iTerm2

// BuildOSC133PromptStart marks the beginning of a shell prompt.
// Format: OSC 133 ; A ST
func BuildOSC133PromptStart() string {
	return "\x1b]133;A\x07"
}

// BuildOSC133CommandStart marks the end of prompt/start of user input.
// Format: OSC 133 ; B ST
func BuildOSC133CommandStart() string {
	return "\x1b]133;B\x07"
}

// BuildOSC133CommandExecuted marks the start of command execution.
// Format: OSC 133 ; C ST
func BuildOSC133CommandExecuted() string {
	return "\x1b]133;C\x07"
}

// BuildOSC133CommandFinished marks the end of command execution.
// Format: OSC 133 ; D ; exitcode ST
func BuildOSC133CommandFinished(exitCode int) string {
	return "\x1b]133;D;" + itoa(exitCode) + "\x07"
}

// itoa converts an integer to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negative := n < 0
	if negative {
		n = -n
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	if negative {
		digits = append([]byte{'-'}, digits...)
	}
	return string(digits)
}
