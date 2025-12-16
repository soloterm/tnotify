package detect

import (
	"os"
	"testing"
)

func TestInTmux(t *testing.T) {
	orig := os.Getenv("TMUX")
	defer func() {
		if orig != "" {
			os.Setenv("TMUX", orig)
		} else {
			os.Unsetenv("TMUX")
		}
	}()

	t.Run("inside tmux", func(t *testing.T) {
		os.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
		if !InTmux() {
			t.Error("InTmux() = false, want true")
		}
	})

	t.Run("outside tmux", func(t *testing.T) {
		os.Unsetenv("TMUX")
		if InTmux() {
			t.Error("InTmux() = true, want false")
		}
	})
}

func TestInScreen(t *testing.T) {
	orig := os.Getenv("STY")
	defer func() {
		if orig != "" {
			os.Setenv("STY", orig)
		} else {
			os.Unsetenv("STY")
		}
	}()

	t.Run("inside screen", func(t *testing.T) {
		os.Setenv("STY", "12345.pts-0.hostname")
		if !InScreen() {
			t.Error("InScreen() = false, want true")
		}
	})

	t.Run("outside screen", func(t *testing.T) {
		os.Unsetenv("STY")
		if InScreen() {
			t.Error("InScreen() = true, want false")
		}
	})
}

func TestInMultiplexer(t *testing.T) {
	origTmux := os.Getenv("TMUX")
	origSty := os.Getenv("STY")
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

	tests := []struct {
		name   string
		tmux   string
		screen string
		want   bool
	}{
		{"no multiplexer", "", "", false},
		{"in tmux only", "/tmp/tmux", "", true},
		{"in screen only", "", "screen", true},
		{"in both", "/tmp/tmux", "screen", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.tmux != "" {
				os.Setenv("TMUX", tt.tmux)
			} else {
				os.Unsetenv("TMUX")
			}
			if tt.screen != "" {
				os.Setenv("STY", tt.screen)
			} else {
				os.Unsetenv("STY")
			}

			got := InMultiplexer()
			if got != tt.want {
				t.Errorf("InMultiplexer() = %v, want %v", got, tt.want)
			}
		})
	}
}
