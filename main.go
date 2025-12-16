package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/soloterm/tnotify/internal/detect"
	"github.com/soloterm/tnotify/internal/native"
	"github.com/soloterm/tnotify/internal/osc"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	title       string
	urgency     string
	id          string
	closeID     string
	forceOSC    bool
	forceNative bool
	forceBell   bool
	showCaps    bool
	exitCode    int
	useExitCode bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "tnotify [message]",
		Short: "Send desktop notifications from the terminal",
		Long: `tnotify sends desktop notifications via OSC escape sequences or native tools.

It automatically detects your terminal and uses the best available method:
- OSC 99 (Kitty) - Full featured: urgency, notification IDs
- OSC 777 (WezTerm, Ghostty, VTE) - Title + body
- OSC 9 (iTerm2) - Message only
- Native fallback (osascript, notify-send, PowerShell)

OSC sequences work over SSH and inside tmux/screen!`,
		Example: `  tnotify "Build complete!"
  tnotify -t "My App" "Task finished"
  tnotify -u critical "Server down!"
  tnotify -i progress "Building... 50%"
  tnotify --close progress
  echo "Done" | tnotify -t "Results"
  make build; tnotify -e $? "Build finished"
  make test; tnotify -e $? --if-failed "Tests failed!"`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
		Args:    cobra.MaximumNArgs(1),
		RunE:    run,
	}

	rootCmd.Flags().StringVarP(&title, "title", "t", "", "Notification title")
	rootCmd.Flags().StringVarP(&urgency, "urgency", "u", "normal", "Urgency level: low, normal, critical")
	rootCmd.Flags().StringVarP(&id, "id", "i", "", "Notification ID (for updates, OSC 99 only)")
	rootCmd.Flags().StringVar(&closeID, "close", "", "Close notification by ID (OSC 99 only)")
	rootCmd.Flags().BoolVar(&forceOSC, "osc", false, "Force OSC escape sequences only")
	rootCmd.Flags().BoolVar(&forceNative, "native", false, "Force native notifications only")
	rootCmd.Flags().BoolVar(&forceBell, "bell", false, "Send terminal bell only")
	rootCmd.Flags().BoolVar(&showCaps, "capabilities", false, "Show terminal capabilities as JSON")
	rootCmd.Flags().IntVarP(&exitCode, "exit-code", "e", 0, "Previous command's exit code (sets urgency automatically)")
	rootCmd.Flags().BoolVar(&useExitCode, "if-failed", false, "Only notify if exit code is non-zero (use with -e)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Handle --capabilities
	if showCaps {
		return showCapabilities()
	}

	// Handle --close
	if closeID != "" {
		return closeNotification(closeID)
	}

	// Handle --bell
	if forceBell {
		fmt.Print("\x07")
		return nil
	}

	// Handle --if-failed: skip notification if exit code is 0
	if useExitCode && exitCode == 0 {
		return nil
	}

	// Get message from args or stdin
	message := ""
	if len(args) > 0 {
		message = args[0]
	} else {
		// Check if stdin has data
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			var lines []string
			for scanner.Scan() {
				lines = append(lines, scanner.Text())
			}
			message = strings.Join(lines, "\n")
		}
	}

	if message == "" {
		return fmt.Errorf("no message provided")
	}

	// Parse urgency (exit code overrides if provided via -e flag)
	urgencyLevel := parseUrgency(urgency)
	if cmd.Flags().Changed("exit-code") {
		urgencyLevel = urgencyFromExitCode(exitCode)
	}

	// Send notification
	if forceNative {
		return sendNative(message, title, urgencyLevel)
	}

	if forceOSC {
		return sendOSC(message, title, urgencyLevel, id)
	}

	// Auto mode: try OSC first, then native, then bell
	return sendAny(message, title, urgencyLevel, id)
}

func parseUrgency(u string) int {
	switch strings.ToLower(u) {
	case "low", "0":
		return osc.UrgencyLow
	case "critical", "2":
		return osc.UrgencyCritical
	default:
		return osc.UrgencyNormal
	}
}

func urgencyFromExitCode(code int) int {
	if code == 0 {
		return osc.UrgencyNormal
	}
	return osc.UrgencyCritical
}

func sendOSC(message, title string, urgency int, id string) error {
	terminal := detect.DetectTerminal()
	protocol := detect.SelectProtocol(terminal)

	if protocol == detect.ProtocolNone {
		return fmt.Errorf("no OSC notification support detected for terminal: %s", terminal)
	}

	var sequence string
	switch protocol {
	case detect.ProtocolOSC9:
		sequence = osc.BuildOSC9(message)
	case detect.ProtocolOSC777:
		sequence = osc.BuildOSC777(message, title)
	case detect.ProtocolOSC99:
		sequence = osc.BuildOSC99(message, title, urgency, id)
	}

	sequence = osc.WrapForMultiplexer(sequence)
	fmt.Print(sequence)
	return nil
}

func sendNative(message, title string, urgency int) error {
	if !native.IsAvailable() {
		return fmt.Errorf("no native notification support available")
	}
	if !native.Send(message, title, urgency) {
		return fmt.Errorf("failed to send native notification")
	}
	return nil
}

func sendAny(message, title string, urgency int, id string) error {
	// Try OSC first
	terminal := detect.DetectTerminal()
	protocol := detect.SelectProtocol(terminal)

	if protocol != detect.ProtocolNone {
		return sendOSC(message, title, urgency, id)
	}

	// Try native
	if native.IsAvailable() {
		return sendNative(message, title, urgency)
	}

	// Fall back to bell
	fmt.Print("\x07")
	return nil
}

func closeNotification(id string) error {
	terminal := detect.DetectTerminal()
	protocol := detect.SelectProtocol(terminal)

	if protocol != detect.ProtocolOSC99 {
		return fmt.Errorf("closing notifications requires Kitty terminal (OSC 99)")
	}

	sequence := osc.BuildOSC99Close(id)
	if sequence == "" {
		return fmt.Errorf("invalid notification ID")
	}

	sequence = osc.WrapForMultiplexer(sequence)
	fmt.Print(sequence)
	return nil
}

func showCapabilities() error {
	caps := detect.GetCapabilities(native.IsAvailable())

	data, err := json.MarshalIndent(caps, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling capabilities: %w", err)
	}

	fmt.Println(string(data))
	return nil
}
