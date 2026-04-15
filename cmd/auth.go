package cmd

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"

	"github.com/exabits/gpu-cli/internal/api"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with the Exabits API",
}

// ── auth login ────────────────────────────────────────────────────────────────

var (
	loginUsername string
	loginPassword string
)

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in and save access_token + refresh_token to ~/.exabits/config.yaml",
	Long: `Authenticates against POST /api/v1/authenticate/login.

The password is hashed with MD5 before being sent, matching the API requirement.
On success the returned access_token and refresh_token are written to
~/.exabits/config.yaml so subsequent commands can use them immediately.

Token lifetimes:
  access_token   30 minutes
  refresh_token  2 hours

If you want credentials that never expire, generate an API Token from the
Exabits platform and set api_token in ~/.exabits/config.yaml instead.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if loginUsername == "" || loginPassword == "" {
			exitInvalidArgs(fmt.Errorf("both --username and --password are required"))
			return nil
		}

		// The API requires the password to be MD5-hashed.
		md5Hash := fmt.Sprintf("%x", md5.Sum([]byte(loginPassword)))

		baseURL := viper.GetString("api_url")
		data, err := api.Login(baseURL, loginUsername, md5Hash)
		if err != nil {
			exitAPIError(err)
			return nil
		}

		// Persist tokens to ~/.exabits/config.yaml, preserving any existing keys
		// (e.g. api_url) that the user may have set manually.
		if err := saveTokens(data.AccessToken, data.RefreshToken); err != nil {
			exitAPIError(fmt.Errorf("login succeeded but could not save tokens: %w", err))
			return nil
		}

		printJSON(map[string]string{
			"username": data.Username,
			"email":    data.Email,
			"message":  "login successful — tokens saved to ~/.exabits/config.yaml",
		})
		return nil
	},
}

// saveTokens writes access_token and refresh_token into ~/.exabits/config.yaml.
func saveTokens(accessToken, refreshToken string) error {
	return saveConfigKeys(map[string]any{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// saveConfigKeys merges the given key-value pairs into ~/.exabits/config.yaml,
// preserving all keys that are already present in the file.
// The config directory is created with 0700 and the file is written with 0600.
func saveConfigKeys(values map[string]any) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}

	configDir := filepath.Join(home, ".exabits")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("cannot create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Read the existing config so we do not clobber other keys (e.g. api_url).
	existing := map[string]any{}
	if raw, err := os.ReadFile(configPath); err == nil {
		_ = yaml.Unmarshal(raw, &existing) // ignore parse error; start fresh if malformed
	}

	for k, v := range values {
		existing[k] = v
	}

	out, err := yaml.Marshal(existing)
	if err != nil {
		return fmt.Errorf("cannot marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, out, 0600); err != nil {
		return fmt.Errorf("cannot write config file: %w", err)
	}

	return nil
}

func init() {
	authLoginCmd.Flags().StringVar(&loginUsername, "username", "", "Exabits account username (required)")
	authLoginCmd.Flags().StringVar(&loginPassword, "password", "", "Exabits account password, plain text — hashed with MD5 before sending (required)")

	authCmd.AddCommand(authLoginCmd)
	rootCmd.AddCommand(authCmd)
}
