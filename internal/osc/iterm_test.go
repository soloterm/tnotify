package osc

import "testing"

func TestBuildITermRequestAttention(t *testing.T) {
	tests := []struct {
		name      string
		fireworks bool
		want      string
	}{
		{"normal attention", false, "\x1b]1337;RequestAttention=yes\x07"},
		{"fireworks attention", true, "\x1b]1337;RequestAttention=fireworks\x07"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildITermRequestAttention(tt.fireworks)
			if got != tt.want {
				t.Errorf("BuildITermRequestAttention(%v) = %q, want %q", tt.fireworks, got, tt.want)
			}
		})
	}
}

func TestBuildITermStealFocus(t *testing.T) {
	want := "\x1b]1337;StealFocus\x07"
	got := BuildITermStealFocus()
	if got != want {
		t.Errorf("BuildITermStealFocus() = %q, want %q", got, want)
	}
}
