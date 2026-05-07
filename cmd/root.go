package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var jsonOutput bool

var rootCmd = &cobra.Command{
	Use:   "egpu",
	Short: "Exabits GPU Cloud CLI",
	Long: `egpu is a command-line interface for managing resources on the
Exabits GPU Cloud platform (gpu-api.exascalelabs.ai).

Configuration is read from ~/.exabits/config.yaml or via environment variables
prefixed with EXABITS_ (e.g. EXABITS_ACCESS_TOKEN).`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute is the entrypoint called by main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		printError(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output results as JSON (always true for piped use)")

	// Bind the flag so Viper can also read EXABITS_JSON=true
	viper.BindPFlag("json", rootCmd.PersistentFlags().Lookup("json")) //nolint:errcheck
}

// initConfig reads configuration from ~/.exabits/config.yaml and from
// environment variables prefixed with EXABITS_.
func initConfig() {
	home, err := os.UserHomeDir()
	if err != nil {
		printError(fmt.Errorf("cannot determine home directory: %w", err))
		os.Exit(1)
	}

	configDir := filepath.Join(home, ".exabits")
	viper.AddConfigPath(configDir)
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Allow overriding any config key via env: EXABITS_ACCESS_TOKEN, etc.
	viper.SetEnvPrefix("EXABITS")
	viper.AutomaticEnv()

	// It is not an error if the config file is absent — tokens can come from env.
	_ = viper.ReadInConfig()
}

// isJSONMode returns true when structured JSON output is requested,
// either via --json flag or when stdout is not a terminal (piped usage).
func isJSONMode() bool {
	if jsonOutput {
		return true
	}
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// printJSON encodes v as indented JSON to stdout.
// Exits with code 1 on encoding failure.
func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		printError(fmt.Errorf("failed to encode output: %w", err))
		os.Exit(1)
	}
}

// printError writes a structured error to stderr.
// It always produces JSON when --json is active; otherwise plain text.
func printError(err error) {
	if isJSONMode() {
		payload := map[string]string{"error": err.Error()}
		enc := json.NewEncoder(os.Stderr)
		enc.SetIndent("", "  ")
		_ = enc.Encode(payload)
	} else {
		fmt.Fprintf(os.Stderr, "error: %s\n", err.Error())
	}
}

// exitInvalidArgs prints an error and exits with code 2 (invalid argument).
func exitInvalidArgs(err error) {
	printError(err)
	os.Exit(2)
}

// exitAPIError prints an error and exits with code 1 (API / internal error).
func exitAPIError(err error) {
	printError(err)
	os.Exit(1)
}
