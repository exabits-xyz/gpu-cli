package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage local CLI configuration (~/.exabits/config.yaml)",
}

// configKeyMap maps user-facing key names to the internal Viper/config-file key.
// Both the alias (e.g. "api_key") and the canonical name are accepted for set/get.
var configKeyMap = map[string]string{
	"api_key":             "api_token", // user-facing alias
	"api_token":           "api_token",
	"api_token_encrypted": "api_token_encrypted",
	"api_url":             "api_url",
	"auth_url":            "auth_url",
	"access_token":        "access_token",
	"refresh_token":       "refresh_token",
}

// sensitiveKeys are masked in `config show` and `config get` output.
var sensitiveKeys = map[string]bool{
	"api_token":           true,
	"api_token_encrypted": true,
	"access_token":        true,
	"refresh_token":       true,
}

// maskValue shortens a sensitive string to first-6 … last-4 characters.
// Values that are too short are replaced entirely with "***".
func maskValue(v string) string {
	if len(v) <= 12 {
		return "***"
	}
	return v[:6] + "..." + v[len(v)-4:]
}

// ── config set ────────────────────────────────────────────────────────────────

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value in ~/.exabits/config.yaml",
	Long: `Writes a single key-value pair into ~/.exabits/config.yaml.
Existing keys in the file are preserved.

Supported keys:
  api_key       / api_token     — API Token for authentication
  api_token_encrypted           — Encrypted API token written by browser auth
  api_url                       — Override the API base URL
  auth_url                      — Override the browser login URL
  access_token                  — JWT access token  (expires 30 min)
  refresh_token                 — JWT refresh token (expires 2 h)

Examples:
  egpu config set api_key    your-api-token
  egpu config set api_url    https://staging.gpu-api.exascalelabs.ai`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			exitInvalidArgs(fmt.Errorf("requires exactly 2 arguments: <key> <value>"))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		userKey, value := args[0], args[1]

		internalKey, ok := configKeyMap[strings.ToLower(userKey)]
		if !ok {
			exitInvalidArgs(fmt.Errorf(
				"unknown config key %q — supported keys: api_key, api_token_encrypted, api_url, auth_url, access_token, refresh_token",
				userKey,
			))
			return nil
		}

		if value == "" {
			exitInvalidArgs(fmt.Errorf("value must not be empty"))
			return nil
		}

		if err := saveConfigKeys(map[string]any{internalKey: value}); err != nil {
			exitAPIError(fmt.Errorf("failed to save config: %w", err))
			return nil
		}

		out := map[string]string{
			"key":     internalKey,
			"message": fmt.Sprintf("%s saved to ~/.exabits/config.yaml", internalKey),
		}
		if sensitiveKeys[internalKey] {
			out["value"] = maskValue(value)
		} else {
			out["value"] = value
		}
		printJSON(out)
		return nil
	},
}

// ── config get ────────────────────────────────────────────────────────────────

var configGetShowFull bool

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a single configuration value from ~/.exabits/config.yaml",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			exitInvalidArgs(fmt.Errorf("requires exactly 1 argument: <key>"))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		userKey := args[0]

		internalKey, ok := configKeyMap[strings.ToLower(userKey)]
		if !ok {
			exitInvalidArgs(fmt.Errorf(
				"unknown config key %q — supported keys: api_key, api_token_encrypted, api_url, auth_url, access_token, refresh_token",
				userKey,
			))
			return nil
		}

		value := viper.GetString(internalKey)
		if value == "" {
			printJSON(map[string]string{
				"key":   internalKey,
				"value": "",
				"note":  "not set",
			})
			return nil
		}

		displayValue := value
		if sensitiveKeys[internalKey] && !configGetShowFull {
			displayValue = maskValue(value)
		}

		printJSON(map[string]string{
			"key":   internalKey,
			"value": displayValue,
		})
		return nil
	},
}

// ── config show ───────────────────────────────────────────────────────────────

var configShowFull bool

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all configuration values from ~/.exabits/config.yaml",
	Long: `Prints all recognised configuration values.
Sensitive keys (api_token, access_token, refresh_token) are masked unless
--show-full is passed.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Collect all canonical keys in a deterministic order.
		keys := []string{"api_token", "api_token_encrypted", "access_token", "refresh_token", "api_url", "auth_url"}

		out := make(map[string]string, len(keys))
		for _, k := range keys {
			v := viper.GetString(k)
			if v == "" {
				out[k] = ""
				continue
			}
			if sensitiveKeys[k] && !configShowFull {
				out[k] = maskValue(v)
			} else {
				out[k] = v
			}
		}
		printJSON(out)
		return nil
	},
}

// ── config unset ──────────────────────────────────────────────────────────────

var configUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Remove a configuration key from ~/.exabits/config.yaml",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			exitInvalidArgs(fmt.Errorf("requires exactly 1 argument: <key>"))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		userKey := args[0]

		internalKey, ok := configKeyMap[strings.ToLower(userKey)]
		if !ok {
			exitInvalidArgs(fmt.Errorf(
				"unknown config key %q — supported keys: api_key, api_token_encrypted, api_url, auth_url, access_token, refresh_token",
				userKey,
			))
			return nil
		}

		if err := saveConfigKeys(map[string]any{internalKey: ""}); err != nil {
			exitAPIError(fmt.Errorf("failed to update config: %w", err))
			return nil
		}

		printJSON(map[string]string{
			"key":     internalKey,
			"message": fmt.Sprintf("%s removed from ~/.exabits/config.yaml", internalKey),
		})
		return nil
	},
}

func init() {
	configGetCmd.Flags().BoolVar(&configGetShowFull, "show-full", false, "Print the full value without masking")
	configShowCmd.Flags().BoolVar(&configShowFull, "show-full", false, "Print full token values without masking")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configUnsetCmd)

	rootCmd.AddCommand(configCmd)
}
