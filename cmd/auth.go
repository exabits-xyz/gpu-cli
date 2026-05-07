package cmd

import (
	"crypto/md5"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/exabits-xyz/gpu-cli/internal/api"
	"github.com/exabits-xyz/gpu-cli/internal/securestore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.yaml.in/yaml/v3"
)

const (
	deviceAuthPollInterval = 3 * time.Second
)

var authNoBrowser bool

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with the Exabits API",
	Long: `Starts browser-based authentication.

The CLI requests an authorization state from the Exabits API, opens the login
URL in your default browser, then polls until browser authorization succeeds.
The returned API token is encrypted locally and saved in ~/.exabits/config.yaml.

For the legacy username/password flow, use:
  egpu auth login --username <user> --password <password>`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		baseURL := api.ResolveBaseURL(viper.GetString("api_url"))
		start, err := api.RequestDeviceAuth(baseURL)
		if err != nil {
			exitAPIError(err)
			return nil
		}
		if start.State == "" {
			exitAPIError(fmt.Errorf("authorization response did not include state"))
			return nil
		}

		loginURL := buildAuthURL(baseURL, start.State)
		if !authNoBrowser {
			if err := openBrowser(loginURL); err != nil {
				fmt.Fprintf(os.Stderr, "Could not open browser automatically: %s\n", err)
			}
		}

		fmt.Fprintf(os.Stderr, "Open this URL to authorize egpu:\n%s\n\n", loginURL)
		fmt.Fprintf(os.Stderr, "Waiting for browser authorization")

		token, err := pollDeviceAuth(baseURL, start.State, time.Duration(start.ExpiresIn)*time.Second)
		if err != nil {
			fmt.Fprintln(os.Stderr)
			exitAPIError(err)
			return nil
		}
		fmt.Fprintln(os.Stderr)

		if err := saveEncryptedAPIToken(token); err != nil {
			exitAPIError(fmt.Errorf("authorization succeeded but could not save token: %w", err))
			return nil
		}

		printJSON(map[string]string{
			"status":  "authenticated",
			"message": "browser authorization successful — encrypted API token saved to ~/.exabits/config.yaml",
		})
		return nil
	},
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

func saveEncryptedAPIToken(token string) error {
	encrypted, err := securestore.EncryptToken(token)
	if err != nil {
		return err
	}
	return saveConfigKeys(map[string]any{
		"api_token":           "",
		"api_token_encrypted": encrypted,
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

func buildAuthURL(apiBaseURL, state string) string {
	base := viper.GetString("auth_url")
	if base == "" {
		base = authURLFromAPIBase(apiBaseURL)
	}
	sep := "?"
	if strings.Contains(base, "?") {
		sep = "&"
	}
	return base + sep + "state=" + url.QueryEscape(state)
}

func authURLFromAPIBase(apiBaseURL string) string {
	u, err := url.Parse(api.ResolveBaseURL(apiBaseURL))
	if err != nil || u.Scheme == "" || u.Host == "" {
		u, _ = url.Parse(api.DefaultBaseURL())
	}
	u.Host = strings.Replace(u.Host, "gpu-api.", "gpu.", 1)
	u.Path = "/login"
	u.RawPath = ""
	u.RawQuery = ""
	u.Fragment = ""
	return u.String()
}

func pollDeviceAuth(baseURL, state string, expiresIn time.Duration) (string, error) {
	if expiresIn <= 0 {
		expiresIn = 15 * time.Minute
	}
	deadline := time.Now().Add(expiresIn)
	for time.Now().Before(deadline) {
		time.Sleep(deviceAuthPollInterval)
		fmt.Fprint(os.Stderr, ".")

		data, ok, err := api.ValidateDeviceAuth(baseURL, state)
		if err != nil {
			return "", err
		}
		if !ok {
			continue
		}
		if data == nil || data.Token == "" {
			return "", fmt.Errorf("authorization response did not include token")
		}
		return data.Token, nil
	}
	return "", fmt.Errorf("authorization timed out after %s", expiresIn.Round(time.Second))
}

func openBrowser(rawURL string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", rawURL)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL)
	default:
		cmd = exec.Command("xdg-open", rawURL)
	}
	return cmd.Start()
}

func init() {
	authCmd.Flags().BoolVar(&authNoBrowser, "no-browser", false, "Print the browser authorization URL without opening it")

	authLoginCmd.Flags().StringVar(&loginUsername, "username", "", "Exabits account username (required)")
	authLoginCmd.Flags().StringVar(&loginPassword, "password", "", "Exabits account password, plain text — hashed with MD5 before sending (required)")

	authCmd.AddCommand(authLoginCmd)
	rootCmd.AddCommand(authCmd)
}
