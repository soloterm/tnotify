package osc

import "testing"

func TestBuildOSC9Progress(t *testing.T) {
	tests := []struct {
		name     string
		state    int
		progress int
		want     string
	}{
		{"normal 50%", ProgressNormal, 50, "\x1b]9;4;1;50\x07"},
		{"error 100%", ProgressError, 100, "\x1b]9;4;2;100\x07"},
		{"indeterminate 0%", ProgressIndeterminate, 0, "\x1b]9;4;3;0\x07"},
		{"paused 75%", ProgressPaused, 75, "\x1b]9;4;4;75\x07"},
		{"hidden 0%", ProgressHidden, 0, "\x1b]9;4;0;0\x07"},
		{"clamp progress over 100", ProgressNormal, 150, "\x1b]9;4;1;100\x07"},
		{"clamp progress under 0", ProgressNormal, -10, "\x1b]9;4;1;0\x07"},
		{"clamp invalid state", 99, 50, "\x1b]9;4;1;50\x07"},
		{"clamp negative state", -1, 50, "\x1b]9;4;1;50\x07"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildOSC9Progress(tt.state, tt.progress)
			if got != tt.want {
				t.Errorf("BuildOSC9Progress(%d, %d) = %q, want %q", tt.state, tt.progress, got, tt.want)
			}
		})
	}
}

func TestBuildOSC9ProgressClear(t *testing.T) {
	want := "\x1b]9;4;0;0\x07"
	got := BuildOSC9ProgressClear()
	if got != want {
		t.Errorf("BuildOSC9ProgressClear() = %q, want %q", got, want)
	}
}
