package cmd

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/exabits-xyz/gpu-cli/internal/api"
	"github.com/exabits-xyz/gpu-cli/internal/types"
	"github.com/spf13/cobra"
)

var modelCmd = &cobra.Command{
	Use:   "model",
	Short: "Query available AI models",
}

// ── model list ────────────────────────────────────────────────────────────────

var (
	modelListLimit     int
	modelListOffset    int
	modelListSortField string
	modelListSortOrder string
)

var modelListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available AI models",
	Long: `Lists AI models available for your applications, including pricing
(per million input/output tokens, keyed by currency), provider information,
context length, and maximum completion tokens.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if modelListSortOrder != "" && modelListSortOrder != "asc" && modelListSortOrder != "desc" {
			exitInvalidArgs(fmt.Errorf("--sort-order must be \"asc\" or \"desc\", got %q", modelListSortOrder))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		params := url.Values{}
		if cmd.Flags().Changed("limit") {
			params.Set("limit", strconv.Itoa(modelListLimit))
		}
		if cmd.Flags().Changed("offset") {
			params.Set("offset", strconv.Itoa(modelListOffset))
		}
		if modelListSortField != "" {
			params.Set("sortField", modelListSortField)
		}
		if modelListSortOrder != "" {
			params.Set("sortOrder", modelListSortOrder)
		}

		path := "/models"
		if len(params) > 0 {
			path += "?" + params.Encode()
		}

		var models []types.Model
		var total int
		if err := client.GetPaged(path, &models, &total); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(types.ModelListResult{Total: total, Data: models})
		return nil
	},
}

func init() {
	// model list flags
	modelListCmd.Flags().IntVar(&modelListLimit, "limit", 0, "Maximum number of models to return")
	modelListCmd.Flags().IntVar(&modelListOffset, "offset", 0, "Number of models to skip (for pagination)")
	modelListCmd.Flags().StringVar(&modelListSortField, "sort-field", "", "Field to sort by (e.g. model_name, context_length)")
	modelListCmd.Flags().StringVar(&modelListSortOrder, "sort-order", "", "Sort direction: asc or desc")

	modelCmd.AddCommand(modelListCmd)

	rootCmd.AddCommand(modelCmd)
}
