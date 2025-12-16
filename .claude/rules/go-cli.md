---
paths: "**/*.go"
---

# Go CLI Application Rules

## Project Structure

```
myapp/
├── main.go              # Entry point, minimal logic
├── cmd/                 # Command definitions (for multi-command CLIs)
│   ├── root.go          # Root command setup
│   ├── serve.go         # Subcommand example
│   └── version.go
├── internal/            # Private application code
│   ├── config/          # Configuration handling
│   ├── service/         # Business logic
│   └── output/          # Output formatting (JSON, table, etc.)
├── pkg/                 # Public libraries (if any)
├── go.mod
├── go.sum
└── .goreleaser.yaml     # Release automation
```

## Cobra CLI Framework

### Root Command Setup

```go
package cmd

import (
    "os"

    "github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
    Use:   "myapp",
    Short: "A brief description of your application",
    Long:  `A longer description that spans multiple lines.`,
    // Uncomment if bare command should run something:
    // Run: func(cmd *cobra.Command, args []string) { },
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        os.Exit(1)
    }
}

func init() {
    rootCmd.PersistentFlags().StringP("config", "c", "", "config file path")
    rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
    rootCmd.PersistentFlags().StringP("output", "o", "text", "output format (text, json)")
}
```

### Subcommand Pattern

```go
package cmd

import (
    "fmt"

    "github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the server",
    Long:  `Start the HTTP server on the specified port.`,
    Args:  cobra.NoArgs,  // or cobra.ExactArgs(1), cobra.MinimumNArgs(1)
    RunE: func(cmd *cobra.Command, args []string) error {
        port, _ := cmd.Flags().GetInt("port")
        return runServer(port)
    },
}

func init() {
    rootCmd.AddCommand(serveCmd)
    serveCmd.Flags().IntP("port", "p", 8080, "port to listen on")
}
```

### Flag Best Practices

```go
// Persistent flags (inherited by subcommands)
rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")

// Local flags (only for this command)
serveCmd.Flags().IntP("port", "p", 8080, "port number")

// Required flags
serveCmd.MarkFlagRequired("port")

// Flag groups (mutually exclusive)
serveCmd.MarkFlagsMutuallyExclusive("json", "yaml")

// Hidden flags (for advanced users)
rootCmd.Flags().Bool("debug", false, "enable debug mode")
rootCmd.Flags().MarkHidden("debug")
```

## Error Handling

### Return Errors, Don't os.Exit

```go
// Good: Return errors from RunE
var myCmd = &cobra.Command{
    Use:  "mycommand",
    RunE: func(cmd *cobra.Command, args []string) error {
        result, err := doSomething()
        if err != nil {
            return fmt.Errorf("failed to do something: %w", err)
        }
        fmt.Println(result)
        return nil
    },
}

// Bad: Calling os.Exit in command logic
var myCmd = &cobra.Command{
    Run: func(cmd *cobra.Command, args []string) {
        if err := doSomething(); err != nil {
            fmt.Fprintln(os.Stderr, err)
            os.Exit(1)  // Don't do this
        }
    },
}
```

### Wrap Errors with Context

```go
// Good: Wrap errors with context
if err := loadConfig(path); err != nil {
    return fmt.Errorf("loading config from %s: %w", path, err)
}

// Bad: Bare error return
if err := loadConfig(path); err != nil {
    return err
}
```

## Output Patterns

### Stderr for Errors, Stdout for Data

```go
// Errors and logs go to stderr
fmt.Fprintln(os.Stderr, "Error: connection failed")

// Data output goes to stdout (pipeable)
fmt.Println(result)

// Use cmd's output streams in Cobra
cmd.PrintErr("Error: something went wrong\n")
cmd.PrintErrln("Error: something went wrong")
cmd.Println(result)
```

### Support Multiple Output Formats

```go
type Output struct {
    format string
    writer io.Writer
}

func (o *Output) Print(data interface{}) error {
    switch o.format {
    case "json":
        enc := json.NewEncoder(o.writer)
        enc.SetIndent("", "  ")
        return enc.Encode(data)
    case "yaml":
        return yaml.NewEncoder(o.writer).Encode(data)
    default:
        // Human-readable text format
        return printText(o.writer, data)
    }
}
```

### Progress and Verbose Output

```go
// Check verbose flag before printing
verbose, _ := cmd.Flags().GetBool("verbose")
if verbose {
    fmt.Fprintf(os.Stderr, "Processing file: %s\n", filename)
}

// For progress bars, use stderr
bar := progressbar.New(100)
bar.SetWriter(os.Stderr)
```

## Configuration

### Viper Integration

```go
import (
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        home, _ := os.UserHomeDir()
        viper.AddConfigPath(home)
        viper.AddConfigPath(".")
        viper.SetConfigType("yaml")
        viper.SetConfigName(".myapp")
    }

    viper.SetEnvPrefix("MYAPP")
    viper.AutomaticEnv()

    if err := viper.ReadInConfig(); err == nil {
        fmt.Fprintln(os.Stderr, "Using config:", viper.ConfigFileUsed())
    }
}

func init() {
    cobra.OnInitialize(initConfig)
}
```

### Environment Variable Binding

```go
// Bind flag to env var
viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
viper.BindEnv("port", "MYAPP_PORT")

// Priority: flag > env > config file > default
port := viper.GetInt("port")
```

## Testing CLI Commands

### Testing Cobra Commands

```go
func TestServeCommand(t *testing.T) {
    // Create a new root command for testing
    cmd := NewRootCmd()

    // Capture output
    buf := new(bytes.Buffer)
    cmd.SetOut(buf)
    cmd.SetErr(buf)

    // Set args and execute
    cmd.SetArgs([]string{"serve", "--port", "9000"})
    err := cmd.Execute()

    assert.NoError(t, err)
    assert.Contains(t, buf.String(), "listening on :9000")
}
```

### Table-Driven Command Tests

```go
func TestCommands(t *testing.T) {
    tests := []struct {
        name    string
        args    []string
        wantErr bool
        want    string
    }{
        {"help", []string{"--help"}, false, "Usage:"},
        {"version", []string{"version"}, false, "v1.0.0"},
        {"invalid flag", []string{"--invalid"}, true, ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            cmd := NewRootCmd()
            buf := new(bytes.Buffer)
            cmd.SetOut(buf)
            cmd.SetArgs(tt.args)

            err := cmd.Execute()

            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.Contains(t, buf.String(), tt.want)
            }
        })
    }
}
```

## Cross-Platform Considerations

### File Paths

```go
import "path/filepath"

// Always use filepath.Join for paths
configPath := filepath.Join(homeDir, ".config", "myapp", "config.yaml")

// Use os.UserHomeDir() for home directory
home, err := os.UserHomeDir()

// Use os.UserConfigDir() for config directory
configDir, err := os.UserConfigDir()
```

### Platform-Specific Code

```go
// file_unix.go
//go:build unix

package myapp

func platformSpecificSetup() {
    // Unix-specific code
}

// file_windows.go
//go:build windows

package myapp

func platformSpecificSetup() {
    // Windows-specific code
}
```

## Release with GoReleaser

### Basic .goreleaser.yaml

```yaml
version: 2

builds:
  - env:
      - CGO_ENABLED=0
    goos:
      - linux
      - darwin
      - windows
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip
    name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude:
      - "^docs:"
      - "^test:"
```

### Version Information

```go
// Set via ldflags at build time
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version information",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("%s version %s (commit: %s, built: %s)\n",
            rootCmd.Name(), version, commit, date)
    },
}
```

## Common Mistakes

### Don't Use Global State

```go
// Bad: Global variables for configuration
var globalConfig Config

// Good: Pass dependencies explicitly
type App struct {
    config Config
    logger *slog.Logger
}

func (a *App) Run() error { ... }
```

### Don't Ignore Context

```go
// Good: Support context cancellation
var serveCmd = &cobra.Command{
    RunE: func(cmd *cobra.Command, args []string) error {
        ctx, cancel := signal.NotifyContext(
            cmd.Context(),
            os.Interrupt,
            syscall.SIGTERM,
        )
        defer cancel()

        return runServer(ctx)
    },
}
```

### Don't Forget Cleanup

```go
// Good: Use defer for cleanup
func processFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()

    // Process file...
    return nil
}
```

### Don't Hardcode Terminal Assumptions

```go
// Good: Check if stdout is a terminal
import "golang.org/x/term"

if term.IsTerminal(int(os.Stdout.Fd())) {
    // Interactive mode - use colors, progress bars
} else {
    // Piped mode - plain output
}
```

## Quick Reference

| Pattern | Example |
|---------|---------|
| Add subcommand | `rootCmd.AddCommand(subCmd)` |
| Required flag | `cmd.MarkFlagRequired("name")` |
| Persistent flag | `rootCmd.PersistentFlags().String(...)` |
| Get flag value | `cmd.Flags().GetString("name")` |
| Error output | `fmt.Fprintln(os.Stderr, err)` |
| Wrap error | `fmt.Errorf("context: %w", err)` |
| Config path | `filepath.Join(home, ".config", "app")` |
| Check terminal | `term.IsTerminal(int(os.Stdout.Fd()))` |
