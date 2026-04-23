package cmd

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/exabits-xyz/gpu-cli/internal/api"
	"github.com/exabits-xyz/gpu-cli/internal/types"
	"github.com/spf13/cobra"
)

var volumeCmd = &cobra.Command{
	Use:   "volume",
	Short: "Manage block-storage volumes",
}

// ── volume list ───────────────────────────────────────────────────────────────

var (
	volumeListLimit     int
	volumeListOffset    int
	volumeListSortField string
	volumeListSortOrder string
	volumeListFilter    string
)

var volumeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List volumes",
	Long: `Returns a list of your existing volumes.

Filtering accepts a JSON array of filter objects, e.g.:
  --filter '[{"key":"name","op":"contains","val":"data"}]'`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if volumeListSortOrder != "" && volumeListSortOrder != "asc" && volumeListSortOrder != "desc" {
			exitInvalidArgs(fmt.Errorf("--sort-order must be \"asc\" or \"desc\", got %q", volumeListSortOrder))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		params := url.Values{}
		if cmd.Flags().Changed("limit") {
			params.Set("limit", strconv.Itoa(volumeListLimit))
		}
		if cmd.Flags().Changed("offset") {
			params.Set("offset", strconv.Itoa(volumeListOffset))
		}
		if volumeListSortField != "" {
			params.Set("sortField", volumeListSortField)
		}
		if volumeListSortOrder != "" {
			params.Set("sortOrder", volumeListSortOrder)
		}
		if volumeListFilter != "" {
			params.Set("filters", volumeListFilter)
		}

		path := "/volumes"
		if len(params) > 0 {
			path += "?" + params.Encode()
		}

		var volumes []types.Volume
		var total int
		if err := client.GetPaged(path, &volumes, &total); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(types.VolumeListResult{Total: total, Data: volumes})
		return nil
	},
}

// ── volume type ───────────────────────────────────────────────────────────────

var volumeTypeCmd = &cobra.Command{
	Use:   "type",
	Short: "Query available volume types",
}

var volumeTypeRegionID string

var volumeTypeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List volume types available in a region",
	Long: `Returns storage backend types that can be specified in the volume_type
field when creating a volume.

--region-id is required. Obtain region IDs with:
  egpu region list`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if volumeTypeRegionID == "" {
			exitInvalidArgs(fmt.Errorf("--region-id is required"))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		path := "/volume-types?region_id=" + url.QueryEscape(volumeTypeRegionID)

		var volumeTypes []types.VolumeType
		if err := client.Get(path, &volumeTypes); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(volumeTypes)
		return nil
	},
}

// ── volume create ─────────────────────────────────────────────────────────────

var (
	volumeCreateDisplayName     string
	volumeCreateRegionID        string
	volumeCreateTypeID          string
	volumeCreateSize            int
	volumeCreateImageID         string
	volumeCreateDescription     string
	volumeCreatePaymentCurrency string
)

var volumeCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new block-storage volume",
	Long: `Creates a volume in the specified region using the given storage type.

Providing --image-id installs an OS image on the volume, making it bootable.

Use 'egpu region list' to find region IDs.
Use 'egpu volume type list --region-id <id>' to find type IDs.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		missing := []string{}
		if volumeCreateDisplayName == "" {
			missing = append(missing, "--display-name")
		}
		if volumeCreateRegionID == "" {
			missing = append(missing, "--region-id")
		}
		if volumeCreateTypeID == "" {
			missing = append(missing, "--type-id")
		}
		if !cmd.Flags().Changed("size") {
			missing = append(missing, "--size")
		}
		if len(missing) > 0 {
			exitInvalidArgs(fmt.Errorf("missing required flags: %v", missing))
			return nil
		}
		if len(volumeCreateDisplayName) > 50 {
			exitInvalidArgs(fmt.Errorf("--display-name must be 50 characters or fewer (got %d)", len(volumeCreateDisplayName)))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		req := types.CreateVolumeRequest{
			DisplayName:     volumeCreateDisplayName,
			RegionID:        volumeCreateRegionID,
			TypeID:          volumeCreateTypeID,
			Size:            volumeCreateSize,
			ImageID:         volumeCreateImageID,
			Description:     volumeCreateDescription,
			PaymentCurrency: volumeCreatePaymentCurrency,
		}

		var created types.CreateVolumeResponse
		if err := client.Post("/volumes", req, &created); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(created)
		return nil
	},
}

// ── volume delete ─────────────────────────────────────────────────────────────

var volumeDeleteForce bool

var volumeDeleteCmd = &cobra.Command{
	Use:   "delete <volume-id>",
	Short: "Permanently delete a volume",
	Long: `Permanently deletes the specified volume.

Pass --force to confirm. This is irreversible.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			exitInvalidArgs(fmt.Errorf("requires exactly 1 argument: <volume-id>"))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		volumeID := args[0]

		if !volumeDeleteForce {
			exitInvalidArgs(fmt.Errorf("deleting a volume is irreversible — pass --force to confirm"))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		if err := client.Delete("/volumes/" + volumeID); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(map[string]string{
			"id":      volumeID,
			"status":  "deleted",
			"message": "volume deleted successfully",
		})
		return nil
	},
}

func init() {
	// volume list flags
	volumeListCmd.Flags().IntVar(&volumeListLimit, "limit", 0, "Maximum number of volumes to return")
	volumeListCmd.Flags().IntVar(&volumeListOffset, "offset", 0, "Number of volumes to skip (for pagination)")
	volumeListCmd.Flags().StringVar(&volumeListSortField, "sort-field", "", "Field to sort by (e.g. name, status, created_time)")
	volumeListCmd.Flags().StringVar(&volumeListSortOrder, "sort-order", "", "Sort direction: asc or desc")
	volumeListCmd.Flags().StringVar(&volumeListFilter, "filter", "", `JSON filter array, e.g. '[{"key":"name","op":"contains","val":"data"}]'`)

	// volume create flags
	volumeCreateCmd.Flags().StringVar(&volumeCreateDisplayName, "display-name", "", "Display name, max 50 characters (required)")
	volumeCreateCmd.Flags().StringVar(&volumeCreateRegionID, "region-id", "", "Region ID (required)")
	volumeCreateCmd.Flags().StringVar(&volumeCreateTypeID, "type-id", "", "Volume type ID (required)")
	volumeCreateCmd.Flags().IntVar(&volumeCreateSize, "size", 0, "Size in GB, max 1048576 (required)")
	volumeCreateCmd.Flags().StringVar(&volumeCreateImageID, "image-id", "", "OS image ID — makes the volume bootable (optional)")
	volumeCreateCmd.Flags().StringVar(&volumeCreateDescription, "description", "", "Description (optional)")
	volumeCreateCmd.Flags().StringVar(&volumeCreatePaymentCurrency, "payment-currency", "", "Payment currency, default USD (optional)")

	// volume delete flags
	volumeDeleteCmd.Flags().BoolVar(&volumeDeleteForce, "force", false, "Confirm permanent deletion (required)")

	// volume type list flags
	volumeTypeListCmd.Flags().StringVar(&volumeTypeRegionID, "region-id", "", "Region ID to filter volume types (required)")

	volumeTypeCmd.AddCommand(volumeTypeListCmd)

	volumeCmd.AddCommand(volumeListCmd)
	volumeCmd.AddCommand(volumeCreateCmd)
	volumeCmd.AddCommand(volumeDeleteCmd)
	volumeCmd.AddCommand(volumeTypeCmd)

	rootCmd.AddCommand(volumeCmd)
}
