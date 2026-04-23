package cmd

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/exabits-xyz/gpu-cli/internal/api"
	"github.com/exabits-xyz/gpu-cli/internal/types"
	"github.com/spf13/cobra"
)

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "Manage API tokens",
}

// ── token create ──────────────────────────────────────────────────────────────

var (
	tokenCreateName        string
	tokenCreateDescription string
	tokenCreateSave        bool
)

var tokenCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new API token for authentication",
	Long: `Creates an API token via POST /api/v1/api-tokens.

API tokens do not expire and require only a single Authorization header,
unlike JWT access_token / refresh_token pairs which expire after 30 min / 2 h.

Use --save to write the generated token directly to ~/.exabits/config.yaml as
api_token, making it the active credential for all subsequent commands.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if tokenCreateName == "" {
			exitInvalidArgs(fmt.Errorf("--name is required"))
			return nil
		}
		if len(tokenCreateName) > 50 {
			exitInvalidArgs(fmt.Errorf("--name must be 50 characters or fewer (got %d)", len(tokenCreateName)))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		req := types.CreateTokenRequest{
			Name:        tokenCreateName,
			Description: tokenCreateDescription,
		}

		var token types.APIToken
		if err := client.Post("/api-tokens", req, &token); err != nil {
			exitAPIError(err)
			return nil
		}

		if tokenCreateSave {
			if err := saveConfigKeys(map[string]any{"api_token": token.Token}); err != nil {
				exitAPIError(fmt.Errorf("token created but could not save to config: %w", err))
				return nil
			}
		}

		printJSON(token)
		return nil
	},
}

// ── token update ──────────────────────────────────────────────────────────────

var (
	tokenUpdateName        string
	tokenUpdateDescription string
)

var tokenUpdateCmd = &cobra.Command{
	Use:   "update <token-id>",
	Short: "Update the name or description of an API token",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			exitInvalidArgs(fmt.Errorf("requires exactly 1 argument: <token-id>"))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		tokenID := args[0]

		if tokenUpdateName == "" {
			exitInvalidArgs(fmt.Errorf("--name is required"))
			return nil
		}
		if len(tokenUpdateName) > 50 {
			exitInvalidArgs(fmt.Errorf("--name must be 50 characters or fewer (got %d)", len(tokenUpdateName)))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		// UpdateTokenRequest has the same body shape as CreateTokenRequest.
		req := types.CreateTokenRequest{
			Name:        tokenUpdateName,
			Description: tokenUpdateDescription,
		}

		var token types.APIToken
		if err := client.Put("/api-tokens/"+tokenID, req, &token); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(token)
		return nil
	},
}

// ── token list ────────────────────────────────────────────────────────────────

var (
	tokenListLimit     int
	tokenListOffset    int
	tokenListSortField string
	tokenListSortOrder string
)

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List API tokens",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if tokenListSortOrder != "" && tokenListSortOrder != "asc" && tokenListSortOrder != "desc" {
			exitInvalidArgs(fmt.Errorf("--sort-order must be \"asc\" or \"desc\", got %q", tokenListSortOrder))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		params := url.Values{}
		if cmd.Flags().Changed("limit") {
			params.Set("limit", strconv.Itoa(tokenListLimit))
		}
		if cmd.Flags().Changed("offset") {
			params.Set("offset", strconv.Itoa(tokenListOffset))
		}
		if tokenListSortField != "" {
			params.Set("sortField", tokenListSortField)
		}
		if tokenListSortOrder != "" {
			params.Set("sortOrder", tokenListSortOrder)
		}

		path := "/api-tokens"
		if len(params) > 0 {
			path += "?" + params.Encode()
		}

		var tokens []types.APIToken
		var total int
		if err := client.GetPaged(path, &tokens, &total); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(types.TokenListResult{Total: total, Data: tokens})
		return nil
	},
}

// ── token delete ──────────────────────────────────────────────────────────────

var tokenDeleteForce bool

var tokenDeleteCmd = &cobra.Command{
	Use:   "delete <token-id>",
	Short: "Permanently delete an API token",
	Long: `Permanently deletes the specified API token.

The token will immediately stop authenticating API requests.
Pass --force to confirm. This flag is required to keep the command
fully non-interactive for scripted and agent use.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			exitInvalidArgs(fmt.Errorf("requires exactly 1 argument: <token-id>"))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		tokenID := args[0]

		if !tokenDeleteForce {
			exitInvalidArgs(fmt.Errorf(
				"deleting an API token is irreversible — pass --force to confirm",
			))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		if err := client.Delete("/api-tokens/" + tokenID); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(map[string]string{
			"id":      tokenID,
			"status":  "deleted",
			"message": "API token deleted successfully",
		})
		return nil
	},
}

func init() {
	// token list flags
	tokenListCmd.Flags().IntVar(&tokenListLimit, "limit", 0, "Maximum number of tokens to return")
	tokenListCmd.Flags().IntVar(&tokenListOffset, "offset", 0, "Number of tokens to skip (for pagination)")
	tokenListCmd.Flags().StringVar(&tokenListSortField, "sort-field", "", "Field to sort by (e.g. name, created_at, last_used)")
	tokenListCmd.Flags().StringVar(&tokenListSortOrder, "sort-order", "", "Sort direction: asc or desc")

	// token create flags
	tokenCreateCmd.Flags().StringVar(&tokenCreateName, "name", "", "Token name, max 50 characters (required)")
	tokenCreateCmd.Flags().StringVar(&tokenCreateDescription, "description", "", "Optional description for the token")
	tokenCreateCmd.Flags().BoolVar(&tokenCreateSave, "save", false, "Save the generated token as api_token in ~/.exabits/config.yaml")

	// token update flags
	tokenUpdateCmd.Flags().StringVar(&tokenUpdateName, "name", "", "New token name, max 50 characters (required)")
	tokenUpdateCmd.Flags().StringVar(&tokenUpdateDescription, "description", "", "New description (optional; omit to clear)")

	// token delete flags
	tokenDeleteCmd.Flags().BoolVar(&tokenDeleteForce, "force", false, "Confirm deletion (required — token will stop authenticating immediately)")

	tokenCmd.AddCommand(tokenListCmd)
	tokenCmd.AddCommand(tokenCreateCmd)
	tokenCmd.AddCommand(tokenUpdateCmd)
	tokenCmd.AddCommand(tokenDeleteCmd)
	rootCmd.AddCommand(tokenCmd)
}
