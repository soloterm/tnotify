package osc

import "fmt"

// BuildOSC777 builds an OSC 777 notification sequence (rxvt-unicode style).
// Format: ESC ] 777 ; notify ; title ; body BEL
func BuildOSC777(message, title string) string {
	if title == "" {
		title = "Notification"
	}
	title = Sanitize(title)
	message = Sanitize(message)
	return fmt.Sprintf("\x1b]777;notify;%s;%s\x07", title, message)
}
