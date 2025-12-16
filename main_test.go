package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/soloterm/tnotify/internal/detect"
	"github.com/soloterm/tnotify/internal/osc"
	"github.com/spf13/cobra"
)

// newTestCommand creates a fresh root command for testing
func newTestCommand() *cobra.Command {
	// Reset global flags
	title = ""
	urgency = "normal"
	id = ""
	closeID = ""
	forceOSC = false
	forceNative = false
	forceBell = false
	showCaps = false
	exitCode = 0
	useExitCode = false
	diagnose = false
	progress = -1
	progressState = "normal"
	clearProgress = false
	attention = false
	fireworks = false

	cmd := &cobra.Command{
		Use:     "tnotify [message]",
		Short:   "Send desktop notifications from the terminal",
		Version: "test",
		Args:    cobra.MaximumNArgs(1),
		RunE:    run,
	}

	cmd.Flags().StringVarP(&title, "title", "t", "", "Notification title")
	cmd.Flags().StringVarP(&urgency, "urgency", "u", "normal", "Urgency level: low, normal, critical")
	cmd.Flags().StringVarP(&id, "id", "i", "", "Notification ID (for updates, OSC 99 only)")
	cmd.Flags().StringVar(&closeID, "close", "", "Close notification by ID (OSC 99 only)")
	cmd.Flags().BoolVar(&forceOSC, "osc", false, "Force OSC escape sequences only")
	cmd.Flags().BoolVar(&forceNative, "native", false, "Force native notifications only")
	cmd.Flags().BoolVar(&forceBell, "bell", false, "Send terminal bell only")
	cmd.Flags().BoolVar(&showCaps, "capabilities", false, "Show terminal capabilities as JSON")
	cmd.Flags().IntVarP(&exitCode, "exit-code", "e", 0, "Previous command's exit code")
	cmd.Flags().BoolVar(&useExitCode, "if-failed", false, "Only notify if exit code is non-zero")
	cmd.Flags().BoolVar(&diagnose, "diagnose", false, "Test all notification methods")
	cmd.Flags().IntVarP(&progress, "progress", "p", -1, "Show progress bar (0-100)")
	cmd.Flags().StringVar(&progressState, "progress-state", "normal", "Progress state")
	cmd.Flags().BoolVar(&clearProgress, "progress-clear", false, "Clear progress bar")
	cmd.Flags().BoolVar(&attention, "attention", false, "Request attention")
	cmd.Flags().BoolVar(&fireworks, "fireworks", false, "Request attention with fireworks")

	return cmd
}

func TestParseUrgency(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"low", osc.UrgencyLow},
		{"LOW", osc.UrgencyLow},
		{"0", osc.UrgencyLow},
		{"normal", osc.UrgencyNormal},
		{"NORMAL", osc.UrgencyNormal},
		{"1", osc.UrgencyNormal},
		{"critical", osc.UrgencyCritical},
		{"CRITICAL", osc.UrgencyCritical},
		{"2", osc.UrgencyCritical},
		{"unknown", osc.UrgencyNormal}, // default
		{"", osc.UrgencyNormal},        // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseUrgency(tt.input)
			if got != tt.want {
				t.Errorf("parseUrgency(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestCommandHelp(t *testing.T) {
	cmd := newTestCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--help"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("--help returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "tnotify") {
		t.Error("Help output should contain 'tnotify'")
	}
	if !strings.Contains(output, "--title") {
		t.Error("Help output should contain '--title' flag")
	}
	if !strings.Contains(output, "--urgency") {
		t.Error("Help output should contain '--urgency' flag")
	}
}

func TestCommandVersion(t *testing.T) {
	cmd := newTestCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("--version returned error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test") {
		t.Errorf("Version output should contain 'test', got: %s", output)
	}
}

func TestCommandBell(t *testing.T) {
	cmd := newTestCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--bell"})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("--bell returned error: %v", err)
	}

	var stdout bytes.Buffer
	stdout.ReadFrom(r)

	if stdout.String() != "\x07" {
		t.Errorf("--bell should output bell character, got: %q", stdout.String())
	}
}

func TestCommandNoMessage(t *testing.T) {
	cmd := newTestCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Ensure stdin is a tty (no piped input)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	// Should fail with "no message provided"
	if err == nil {
		t.Error("Expected error when no message provided")
	}
	if err != nil && !strings.Contains(err.Error(), "no message") {
		t.Errorf("Expected 'no message' error, got: %v", err)
	}
}

func TestCommandCapabilities(t *testing.T) {
	cmd := newTestCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--capabilities"})

	// Capture stdout for JSON output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("--capabilities returned error: %v", err)
	}

	var stdout bytes.Buffer
	stdout.ReadFrom(r)
	output := stdout.String()

	// Should be valid JSON with expected fields
	if !strings.Contains(output, "terminal") {
		t.Error("Capabilities output should contain 'terminal' field")
	}
	if !strings.Contains(output, "protocol") {
		t.Error("Capabilities output should contain 'protocol' field")
	}
	if !strings.Contains(output, "supports_title") {
		t.Error("Capabilities output should contain 'supports_title' field")
	}
}

func TestCommandTooManyArgs(t *testing.T) {
	cmd := newTestCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"message1", "message2"})

	err := cmd.Execute()

	if err == nil {
		t.Error("Expected error when too many arguments provided")
	}
}

func TestCommandFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"short title", []string{"-t", "Title", "msg"}},
		{"long title", []string{"--title", "Title", "msg"}},
		{"short urgency", []string{"-u", "low", "msg"}},
		{"long urgency", []string{"--urgency", "critical", "msg"}},
		{"short id", []string{"-i", "notif-1", "msg"}},
		{"long id", []string{"--id", "notif-1", "msg"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTestCommand()
			cmd.SetArgs(tt.args)

			// We're just testing that flags parse correctly
			// The actual notification will fail in test environment
			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Errorf("Failed to parse flags: %v", err)
			}
		})
	}
}

func TestFlagMutualExclusion(t *testing.T) {
	// Test that --osc, --native, and --bell can be used independently
	tests := []struct {
		name string
		args []string
	}{
		{"osc only", []string{"--osc", "msg"}},
		{"native only", []string{"--native", "msg"}},
		{"bell only", []string{"--bell"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTestCommand()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Errorf("Failed to parse flags: %v", err)
			}
		})
	}
}

func TestUrgencyFromExitCode(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		want     int
	}{
		{"exit 0 is normal", 0, osc.UrgencyNormal},
		{"exit 1 is critical", 1, osc.UrgencyCritical},
		{"exit 2 is critical", 2, osc.UrgencyCritical},
		{"exit 127 is critical", 127, osc.UrgencyCritical},
		{"exit 255 is critical", 255, osc.UrgencyCritical},
		{"negative exit code is critical", -1, osc.UrgencyCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := urgencyFromExitCode(tt.exitCode)
			if got != tt.want {
				t.Errorf("urgencyFromExitCode(%d) = %d, want %d", tt.exitCode, got, tt.want)
			}
		})
	}
}

func TestExitCodeFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"short flag", []string{"-e", "0", "msg"}},
		{"long flag", []string{"--exit-code", "1", "msg"}},
		{"with if-failed", []string{"-e", "1", "--if-failed", "msg"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newTestCommand()
			cmd.SetArgs(tt.args)

			err := cmd.ParseFlags(tt.args)
			if err != nil {
				t.Errorf("Failed to parse flags: %v", err)
			}
		})
	}
}

func TestIfFailedSkipsOnSuccess(t *testing.T) {
	cmd := newTestCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"-e", "0", "--if-failed", "This should not notify"})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("--if-failed with exit 0 returned error: %v", err)
	}

	var stdout bytes.Buffer
	stdout.ReadFrom(r)

	// Should produce no output (notification skipped)
	if stdout.String() != "" {
		t.Errorf("--if-failed with exit 0 should produce no output, got: %q", stdout.String())
	}
}

func TestIfFailedNotifiesOnFailure(t *testing.T) {
	cmd := newTestCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"-e", "1", "--if-failed", "--bell", "Build failed"})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("--if-failed with exit 1 returned error: %v", err)
	}

	var stdout bytes.Buffer
	stdout.ReadFrom(r)

	// Should produce bell output (notification sent)
	if stdout.String() != "\x07" {
		t.Errorf("--if-failed with exit 1 should produce bell, got: %q", stdout.String())
	}
}

func TestDiagnoseFlag(t *testing.T) {
	cmd := newTestCommand()
	cmd.SetArgs([]string{"--diagnose"})

	err := cmd.ParseFlags([]string{"--diagnose"})
	if err != nil {
		t.Errorf("Failed to parse --diagnose flag: %v", err)
	}
}

func TestTerminalName(t *testing.T) {
	tests := []struct {
		input detect.Terminal
		want  string
	}{
		{detect.TerminalUnknown, "unknown"},
		{detect.TerminalKitty, "kitty"},
		{detect.TerminalITerm2, "iterm2"},
		{detect.TerminalWezTerm, "wezterm"},
		{detect.TerminalGhostty, "ghostty"},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			got := terminalName(tt.input)
			if got != tt.want {
				t.Errorf("terminalName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestProtocolName(t *testing.T) {
	tests := []struct {
		input detect.Protocol
		want  string
	}{
		{detect.ProtocolOSC9, "OSC 9 (iTerm2)"},
		{detect.ProtocolOSC777, "OSC 777 (WezTerm/Ghostty/VTE)"},
		{detect.ProtocolOSC99, "OSC 99 (Kitty)"},
		{detect.ProtocolNone, "none"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := protocolName(tt.input)
			if got != tt.want {
				t.Errorf("protocolName(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAvailableStr(t *testing.T) {
	if availableStr(true) != "available" {
		t.Error("availableStr(true) should return 'available'")
	}
	if availableStr(false) != "not available" {
		t.Error("availableStr(false) should return 'not available'")
	}
}

func TestParseProgressState(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"normal", osc.ProgressNormal},
		{"NORMAL", osc.ProgressNormal},
		{"error", osc.ProgressError},
		{"red", osc.ProgressError},
		{"2", osc.ProgressError},
		{"paused", osc.ProgressPaused},
		{"yellow", osc.ProgressPaused},
		{"4", osc.ProgressPaused},
		{"indeterminate", osc.ProgressIndeterminate},
		{"pulse", osc.ProgressIndeterminate},
		{"3", osc.ProgressIndeterminate},
		{"hidden", osc.ProgressHidden},
		{"clear", osc.ProgressHidden},
		{"0", osc.ProgressHidden},
		{"unknown", osc.ProgressNormal}, // default
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseProgressState(tt.input)
			if got != tt.want {
				t.Errorf("parseProgressState(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestProgressFlag(t *testing.T) {
	cmd := newTestCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"-p", "50"})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("-p 50 returned error: %v", err)
	}

	var stdout bytes.Buffer
	stdout.ReadFrom(r)

	// Should output OSC 9;4 sequence
	if !strings.Contains(stdout.String(), "\x1b]9;4;") {
		t.Errorf("-p 50 should output OSC 9;4 sequence, got: %q", stdout.String())
	}
}

func TestAttentionFlag(t *testing.T) {
	cmd := newTestCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--attention"})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("--attention returned error: %v", err)
	}

	var stdout bytes.Buffer
	stdout.ReadFrom(r)

	// Should output iTerm2 RequestAttention sequence
	expected := "\x1b]1337;RequestAttention=yes\x07"
	if stdout.String() != expected {
		t.Errorf("--attention should output %q, got: %q", expected, stdout.String())
	}
}

func TestFireworksFlag(t *testing.T) {
	cmd := newTestCommand()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"--fireworks"})

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := cmd.Execute()

	w.Close()
	os.Stdout = oldStdout

	if err != nil {
		t.Fatalf("--fireworks returned error: %v", err)
	}

	var stdout bytes.Buffer
	stdout.ReadFrom(r)

	// Should output iTerm2 RequestAttention=fireworks sequence
	expected := "\x1b]1337;RequestAttention=fireworks\x07"
	if stdout.String() != expected {
		t.Errorf("--fireworks should output %q, got: %q", expected, stdout.String())
	}
}
