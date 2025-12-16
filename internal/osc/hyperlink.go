package osc

import "fmt"

// BuildOSC8Hyperlink wraps text in an OSC 8 hyperlink sequence.
// Format: ESC ] 8 ; params ; URI ST text ESC ] 8 ; ; ST
// Supported by most modern terminals: iTerm2, VTE, kitty, WezTerm, Windows Terminal, etc.
func BuildOSC8Hyperlink(url, text string) string {
	if url == "" {
		return text
	}
	if text == "" {
		text = url
	}
	// OSC 8 ; ; URL ST text OSC 8 ; ; ST
	return fmt.Sprintf("\x1b]8;;%s\x07%s\x1b]8;;\x07", url, text)
}

// BuildOSC8HyperlinkWithID wraps text in an OSC 8 hyperlink with an ID parameter.
// The ID allows grouping multiple hyperlinks that should be treated as one.
func BuildOSC8HyperlinkWithID(url, text, id string) string {
	if url == "" {
		return text
	}
	if text == "" {
		text = url
	}
	if id == "" {
		return BuildOSC8Hyperlink(url, text)
	}
	// OSC 8 ; id=<id> ; URL ST text OSC 8 ; ; ST
	return fmt.Sprintf("\x1b]8;id=%s;%s\x07%s\x1b]8;;\x07", id, url, text)
}
