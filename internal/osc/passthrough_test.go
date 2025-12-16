package osc

import (
	"os"
	"testing"
)

func TestWrapForMultiplexer(t *testing.T) {
	// Save original env vars
	origTmux := os.Getenv("TMUX")
	origSty := os.Getenv("STY")

	// Clean up after test
	defer func() {
		if origTmux != "" {
			os.Setenv("TMUX", origTmux)
		} else {
			os.Unsetenv("TMUX")
		}
		if origSty != "" {
			os.Setenv("STY", origSty)
		} else {
			os.Unsetenv("STY")
		}
	}()

	t.Run("no multiplexer", func(t *testing.T) {
		os.Unsetenv("TMUX")
		os.Unsetenv("STY")

		input := "\x1b]9;Hello\x07"
		got := WrapForMultiplexer(input)
		if got != input {
			t.Errorf("WrapForMultiplexer() = %q, want %q (unchanged)", got, input)
		}
	})

	t.Run("inside tmux", func(t *testing.T) {
		os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
		os.Unsetenv("STY")

		input := "\x1b]9;Hello\x07"
		// Tmux wraps with DCS tmux; and doubles ESC
		want := "\x1bPtmux;\x1b\x1b]9;Hello\x07\x1b\\"
		got := WrapForMultiplexer(input)
		if got != want {
			t.Errorf("WrapForMultiplexer() in tmux = %q, want %q", got, want)
		}
	})

	t.Run("inside screen", func(t *testing.T) {
		os.Unsetenv("TMUX")
		os.Setenv("STY", "12345.pts-0.hostname")

		input := "\x1b]9;Hello\x07"
		// Screen wraps with DCS
		want := "\x1bP\x1b]9;Hello\x07\x1b\\"
		got := WrapForMultiplexer(input)
		if got != want {
			t.Errorf("WrapForMultiplexer() in screen = %q, want %q", got, want)
		}
	})

	t.Run("tmux takes precedence over screen", func(t *testing.T) {
		os.Setenv("TMUX", "/tmp/tmux")
		os.Setenv("STY", "screen")

		input := "\x1b]9;Hi\x07"
		// Should use tmux wrapping
		want := "\x1bPtmux;\x1b\x1b]9;Hi\x07\x1b\\"
		got := WrapForMultiplexer(input)
		if got != want {
			t.Errorf("WrapForMultiplexer() = %q, want %q", got, want)
		}
	})
}

func TestWrapForTmux(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "OSC 9 sequence",
			input: "\x1b]9;Hello\x07",
			want:  "\x1bPtmux;\x1b\x1b]9;Hello\x07\x1b\\",
		},
		{
			name:  "OSC 99 sequence with ST terminator",
			input: "\x1b]99;;Hello\x1b\\",
			want:  "\x1bPtmux;\x1b\x1b]99;;Hello\x1b\x1b\\\x1b\\",
		},
		{
			name:  "multiple ESC in sequence",
			input: "\x1b]99;d=0:p=title;Title\x1b\\\x1b]99;d=1:p=body;Body\x1b\\",
			want:  "\x1bPtmux;\x1b\x1b]99;d=0:p=title;Title\x1b\x1b\\\x1b\x1b]99;d=1:p=body;Body\x1b\x1b\\\x1b\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapForTmux(tt.input)
			if got != tt.want {
				t.Errorf("wrapForTmux() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWrapForScreen(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "OSC 9 sequence",
			input: "\x1b]9;Hello\x07",
			want:  "\x1bP\x1b]9;Hello\x07\x1b\\",
		},
		{
			name:  "OSC 99 sequence",
			input: "\x1b]99;;Hello\x1b\\",
			want:  "\x1bP\x1b]99;;Hello\x1b\\\x1b\\",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapForScreen(tt.input)
			if got != tt.want {
				t.Errorf("wrapForScreen() = %q, want %q", got, tt.want)
			}
		})
	}
}
