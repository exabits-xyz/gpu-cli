package cmd

import (
	"github.com/exabits-xyz/gpu-cli/internal/api"
	"github.com/exabits-xyz/gpu-cli/internal/types"
	"github.com/spf13/cobra"
)

// ── region ────────────────────────────────────────────────────────────────────

var regionCmd = &cobra.Command{
	Use:   "region",
	Short: "Query available regions",
}

var regionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available datacenter regions",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		var regions []types.Region
		if err := client.Get("/regions", &regions); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(regions)
		return nil
	},
}

// ── flavor ────────────────────────────────────────────────────────────────────

var flavorCmd = &cobra.Command{
	Use:   "flavor",
	Short: "Query available GPU hardware flavors",
}

var flavorRegionID string

var flavorListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available GPU hardware flavors",
	Long: `Lists GPU VM hardware configurations (flavors), grouped by region.

Use --region-id to filter to a specific region. Obtain region IDs with:
  egpu region list`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		path := "/flavors"
		if flavorRegionID != "" {
			path += "?region_id=" + flavorRegionID
		}

		var flavors []types.FlavorGroup
		if err := client.Get(path, &flavors); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(flavors)
		return nil
	},
}

// ── image ─────────────────────────────────────────────────────────────────────

var imageCmd = &cobra.Command{
	Use:   "image",
	Short: "Query available OS images",
}

var imageRegionID string

var imageListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available OS images for VM deployment",
	Long: `Lists operating system images available for VM deployment.

Use --region-id to filter to a specific region. Obtain region IDs with:
  egpu region list

Note: image_id and flavor_id passed to 'egpu vm create' must belong to the
same region.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		path := "/images"
		if imageRegionID != "" {
			path += "?region_id=" + imageRegionID
		}

		var images []types.Image
		if err := client.Get(path, &images); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(images)
		return nil
	},
}

func init() {
	// flavor list flags
	flavorListCmd.Flags().StringVar(&flavorRegionID, "region-id", "", "Filter flavors by region ID")

	// image list flags
	imageListCmd.Flags().StringVar(&imageRegionID, "region-id", "", "Filter images by region ID")

	regionCmd.AddCommand(regionListCmd)
	flavorCmd.AddCommand(flavorListCmd)
	imageCmd.AddCommand(imageListCmd)

	rootCmd.AddCommand(regionCmd)
	rootCmd.AddCommand(flavorCmd)
	rootCmd.AddCommand(imageCmd)
}
