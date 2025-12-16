package osc

import "testing"

func TestBuildOSC133PromptStart(t *testing.T) {
	want := "\x1b]133;A\x07"
	got := BuildOSC133PromptStart()
	if got != want {
		t.Errorf("BuildOSC133PromptStart() = %q, want %q", got, want)
	}
}

func TestBuildOSC133CommandStart(t *testing.T) {
	want := "\x1b]133;B\x07"
	got := BuildOSC133CommandStart()
	if got != want {
		t.Errorf("BuildOSC133CommandStart() = %q, want %q", got, want)
	}
}

func TestBuildOSC133CommandExecuted(t *testing.T) {
	want := "\x1b]133;C\x07"
	got := BuildOSC133CommandExecuted()
	if got != want {
		t.Errorf("BuildOSC133CommandExecuted() = %q, want %q", got, want)
	}
}

func TestBuildOSC133CommandFinished(t *testing.T) {
	tests := []struct {
		exitCode int
		want     string
	}{
		{0, "\x1b]133;D;0\x07"},
		{1, "\x1b]133;D;1\x07"},
		{127, "\x1b]133;D;127\x07"},
		{255, "\x1b]133;D;255\x07"},
	}

	for _, tt := range tests {
		got := BuildOSC133CommandFinished(tt.exitCode)
		if got != tt.want {
			t.Errorf("BuildOSC133CommandFinished(%d) = %q, want %q", tt.exitCode, got, tt.want)
		}
	}
}

func TestItoa(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{1, "1"},
		{10, "10"},
		{127, "127"},
		{255, "255"},
		{-1, "-1"},
		{-127, "-127"},
	}

	for _, tt := range tests {
		got := itoa(tt.input)
		if got != tt.want {
			t.Errorf("itoa(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
