package cmd

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/exabits-xyz/gpu-cli/internal/api"
	"github.com/exabits-xyz/gpu-cli/internal/types"
	"github.com/spf13/cobra"
)

var billingCmd = &cobra.Command{
	Use:   "billing",
	Short: "View account billing information",
}

// ── billing balance ───────────────────────────────────────────────────────────

var billingBalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Retrieve current credit balance",
	Long: `Retrieves the current credit balance for your account.

A positive balance is required to create new resources.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		var balance types.CreditBalance
		if err := client.Get("/billing/balance", &balance); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(balance)
		return nil
	},
}

// ── billing usage ─────────────────────────────────────────────────────────────

var (
	usageLimit     int
	usageOffset    int
	usageSortField string
	usageSortOrder string
	usageFilter    string
)

var billingUsageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Retrieve resource usage cost history",
	Long: `Returns a paginated history of usage costs for all past and present
resources, including per-minute fee, total uptime, and total cost.

status values: active, terminated
type values:   vm, volume`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if usageSortOrder != "" && usageSortOrder != "asc" && usageSortOrder != "desc" {
			exitInvalidArgs(fmt.Errorf("--sort-order must be \"asc\" or \"desc\", got %q", usageSortOrder))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		params := url.Values{}
		if cmd.Flags().Changed("limit") {
			params.Set("limit", strconv.Itoa(usageLimit))
		}
		if cmd.Flags().Changed("offset") {
			params.Set("offset", strconv.Itoa(usageOffset))
		}
		if usageSortField != "" {
			params.Set("sortField", usageSortField)
		}
		if usageSortOrder != "" {
			params.Set("sortOrder", usageSortOrder)
		}
		if usageFilter != "" {
			params.Set("filters", usageFilter)
		}

		path := "/billing/resources/usage"
		if len(params) > 0 {
			path += "?" + params.Encode()
		}

		var records []types.UsageRecord
		var total int
		if err := client.GetPaged(path, &records, &total); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(types.UsageListResult{Total: total, Data: records})
		return nil
	},
}

// ── billing statement ─────────────────────────────────────────────────────────

var (
	stmtLimit     int
	stmtOffset    int
	stmtSortField string
	stmtSortOrder string
	stmtFilter    string
)

var billingStatementCmd = &cobra.Command{
	Use:   "statement",
	Short: "Retrieve billing statements",
	Long: `Returns a paginated list of all billing statements including amount,
payment status, resource type, and time window.

status values: paid, pending, ...
type values:   lease_fee, ...`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if stmtSortOrder != "" && stmtSortOrder != "asc" && stmtSortOrder != "desc" {
			exitInvalidArgs(fmt.Errorf("--sort-order must be \"asc\" or \"desc\", got %q", stmtSortOrder))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		params := url.Values{}
		if cmd.Flags().Changed("limit") {
			params.Set("limit", strconv.Itoa(stmtLimit))
		}
		if cmd.Flags().Changed("offset") {
			params.Set("offset", strconv.Itoa(stmtOffset))
		}
		if stmtSortField != "" {
			params.Set("sortField", stmtSortField)
		}
		if stmtSortOrder != "" {
			params.Set("sortOrder", stmtSortOrder)
		}
		if stmtFilter != "" {
			params.Set("filters", stmtFilter)
		}

		path := "/billing/resources/statements"
		if len(params) > 0 {
			path += "?" + params.Encode()
		}

		var statements []types.Statement
		var total int
		if err := client.GetPaged(path, &statements, &total); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(types.StatementListResult{Total: total, Data: statements})
		return nil
	},
}

func init() {
	// billing usage flags
	billingUsageCmd.Flags().IntVar(&usageLimit, "limit", 0, "Maximum number of records to return")
	billingUsageCmd.Flags().IntVar(&usageOffset, "offset", 0, "Number of records to skip (pagination)")
	billingUsageCmd.Flags().StringVar(&usageSortField, "sort-field", "", "Field to sort by (e.g. created_time, total_fee, status)")
	billingUsageCmd.Flags().StringVar(&usageSortOrder, "sort-order", "", "Sort direction: asc or desc")
	billingUsageCmd.Flags().StringVar(&usageFilter, "filter", "", `JSON filter array, e.g. '[{"key":"status","op":"eq","val":"active"}]'`)

	// billing statement flags
	billingStatementCmd.Flags().IntVar(&stmtLimit, "limit", 0, "Maximum number of statements to return")
	billingStatementCmd.Flags().IntVar(&stmtOffset, "offset", 0, "Number of statements to skip (pagination)")
	billingStatementCmd.Flags().StringVar(&stmtSortField, "sort-field", "", "Field to sort by (e.g. started_time, amount, status)")
	billingStatementCmd.Flags().StringVar(&stmtSortOrder, "sort-order", "", "Sort direction: asc or desc")
	billingStatementCmd.Flags().StringVar(&stmtFilter, "filter", "", `JSON filter array, e.g. '[{"key":"resource_type","op":"eq","val":"vm"}]'`)

	billingCmd.AddCommand(billingBalanceCmd)
	billingCmd.AddCommand(billingUsageCmd)
	billingCmd.AddCommand(billingStatementCmd)

	rootCmd.AddCommand(billingCmd)
}
