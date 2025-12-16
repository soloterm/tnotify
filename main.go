package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

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
	diagnose    bool
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
	rootCmd.Flags().BoolVar(&diagnose, "diagnose", false, "Test all notification methods to see what works")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Handle --diagnose
	if diagnose {
		return runDiagnose()
	}

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

func runDiagnose() error {
	terminal := detect.DetectTerminal()
	protocol := detect.SelectProtocol(terminal)
	inMux := detect.InMultiplexer()
	nativeAvail := native.IsAvailable()

	// Print environment info
	fmt.Println("=== tnotify diagnostic ===")
	fmt.Println()
	fmt.Printf("Detected terminal: %s\n", terminalName(terminal))
	fmt.Printf("Selected protocol: %s\n", protocolName(protocol))
	if inMux {
		if detect.InTmux() {
			fmt.Println("Multiplexer: tmux (will use passthrough)")
		} else {
			fmt.Println("Multiplexer: GNU Screen (will use passthrough)")
		}
	}
	fmt.Printf("Native notifications: %s\n", availableStr(nativeAvail))
	fmt.Println()

	// Warning about focus
	fmt.Println("IMPORTANT: Many terminals suppress notifications when focused.")
	fmt.Println("Please switch to another window NOW.")
	fmt.Println()

	// Countdown
	fmt.Print("Testing in: ")
	for i := 3; i > 0; i-- {
		fmt.Printf("%d... ", i)
		time.Sleep(1 * time.Second)
	}
	fmt.Println("Go!")
	fmt.Println()

	// Test methods
	results := []struct {
		name   string
		result string
	}{}

	// Test 1: OSC (if available)
	if protocol != detect.ProtocolNone {
		fmt.Printf("Testing %s... ", protocolName(protocol))
		var sequence string
		switch protocol {
		case detect.ProtocolOSC9:
			sequence = osc.BuildOSC9("tnotify test: OSC 9")
		case detect.ProtocolOSC777:
			sequence = osc.BuildOSC777("tnotify test: OSC 777", "tnotify")
		case detect.ProtocolOSC99:
			sequence = osc.BuildOSC99("tnotify test: OSC 99", "tnotify", osc.UrgencyNormal, "")
		}
		sequence = osc.WrapForMultiplexer(sequence)
		fmt.Print(sequence)
		fmt.Println("sent")
		results = append(results, struct{ name, result string }{protocolName(protocol), "sent"})
		time.Sleep(2 * time.Second)
	} else {
		fmt.Println("Skipping OSC: no protocol detected for this terminal")
		results = append(results, struct{ name, result string }{"OSC", "skipped (unsupported terminal)"})
	}

	// Test 2: Native
	if nativeAvail {
		fmt.Print("Testing native notifications... ")
		if native.Send("tnotify test: native notification", "tnotify", osc.UrgencyNormal) {
			fmt.Println("sent")
			results = append(results, struct{ name, result string }{"Native", "sent"})
		} else {
			fmt.Println("failed")
			results = append(results, struct{ name, result string }{"Native", "failed"})
		}
		time.Sleep(2 * time.Second)
	} else {
		fmt.Println("Skipping native: not available on this system")
		results = append(results, struct{ name, result string }{"Native", "skipped (unavailable)"})
	}

	// Test 3: Bell
	fmt.Print("Testing terminal bell... ")
	fmt.Print("\x07")
	fmt.Println("sent")
	results = append(results, struct{ name, result string }{"Bell", "sent"})

	// Summary
	fmt.Println()
	fmt.Println("=== Summary ===")
	fmt.Println()
	for _, r := range results {
		status := "?"
		if r.result == "sent" {
			status = "✓"
		} else if r.result == "failed" || strings.HasPrefix(r.result, "skipped") {
			status = "✗"
		}
		fmt.Printf("  %s %s: %s\n", status, r.name, r.result)
	}
	fmt.Println()
	fmt.Println("Did you see/hear the notifications?")
	fmt.Println("If not, your terminal may not support them or may suppress")
	fmt.Println("notifications when focused. Try running a command like:")
	fmt.Println()
	fmt.Println("  sleep 3 && tnotify 'Hello!'")
	fmt.Println()
	fmt.Println("and switch to another window during the countdown.")

	return nil
}

func terminalName(t detect.Terminal) string {
	if t == detect.TerminalUnknown {
		return "unknown"
	}
	return string(t)
}

func protocolName(p detect.Protocol) string {
	switch p {
	case detect.ProtocolOSC9:
		return "OSC 9 (iTerm2)"
	case detect.ProtocolOSC777:
		return "OSC 777 (WezTerm/Ghostty/VTE)"
	case detect.ProtocolOSC99:
		return "OSC 99 (Kitty)"
	default:
		return "none"
	}
}

func availableStr(available bool) string {
	if available {
		return "available"
	}
	return "not available"
}
