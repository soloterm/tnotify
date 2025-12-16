package osc

import "fmt"

// BuildOSC9 builds an OSC 9 notification sequence (iTerm2 style, message only).
// Format: ESC ] 9 ; message BEL
func BuildOSC9(message string) string {
	message = Sanitize(message)
	return fmt.Sprintf("\x1b]9;%s\x07", message)
}
