package osc

import "testing"

func TestBuildOSC8Hyperlink(t *testing.T) {
	tests := []struct {
		name string
		url  string
		text string
		want string
	}{
		{
			"basic hyperlink",
			"https://example.com",
			"Click here",
			"\x1b]8;;https://example.com\x07Click here\x1b]8;;\x07",
		},
		{
			"empty text uses URL",
			"https://example.com",
			"",
			"\x1b]8;;https://example.com\x07https://example.com\x1b]8;;\x07",
		},
		{
			"empty URL returns text only",
			"",
			"Plain text",
			"Plain text",
		},
		{
			"both empty",
			"",
			"",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildOSC8Hyperlink(tt.url, tt.text)
			if got != tt.want {
				t.Errorf("BuildOSC8Hyperlink(%q, %q) = %q, want %q", tt.url, tt.text, got, tt.want)
			}
		})
	}
}

func TestBuildOSC8HyperlinkWithID(t *testing.T) {
	tests := []struct {
		name string
		url  string
		text string
		id   string
		want string
	}{
		{
			"hyperlink with ID",
			"https://example.com",
			"Click here",
			"link1",
			"\x1b]8;id=link1;https://example.com\x07Click here\x1b]8;;\x07",
		},
		{
			"empty ID falls back to basic",
			"https://example.com",
			"Click here",
			"",
			"\x1b]8;;https://example.com\x07Click here\x1b]8;;\x07",
		},
		{
			"empty URL returns text",
			"",
			"Plain text",
			"link1",
			"Plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildOSC8HyperlinkWithID(tt.url, tt.text, tt.id)
			if got != tt.want {
				t.Errorf("BuildOSC8HyperlinkWithID(%q, %q, %q) = %q, want %q", tt.url, tt.text, tt.id, got, tt.want)
			}
		})
	}
}
