package cmd

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/exabits/gpu-cli/internal/api"
	"github.com/exabits/gpu-cli/internal/types"
	"github.com/spf13/cobra"
)

// vmCmd is the parent command: exabits vm <subcommand>
var vmCmd = &cobra.Command{
	Use:   "vm",
	Short: "Manage GPU VM instances",
}

// ── vm list ──────────────────────────────────────────────────────────────────

var (
	listLimit     int
	listOffset    int
	listSortField string
	listSortOrder string
	listFilter    string
)

var vmListCmd = &cobra.Command{
	Use:   "list",
	Short: "List VM instances",
	Long: `Retrieve VM instances with optional pagination, sorting, and filtering.

Filtering accepts a JSON array of filter objects (stringified), e.g.:
  --filter '[{"key":"name","op":"contains","val":"hub"}]'

Supported filter operators: contains, eq, ne, gt, lt`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate sortOrder when provided.
		if listSortOrder != "" && listSortOrder != "asc" && listSortOrder != "desc" {
			exitInvalidArgs(fmt.Errorf("--sort-order must be \"asc\" or \"desc\", got %q", listSortOrder))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		// Build query string from whichever flags were set.
		params := url.Values{}
		if cmd.Flags().Changed("limit") {
			params.Set("limit", strconv.Itoa(listLimit))
		}
		if cmd.Flags().Changed("offset") {
			params.Set("offset", strconv.Itoa(listOffset))
		}
		if listSortField != "" {
			params.Set("sortField", listSortField)
		}
		if listSortOrder != "" {
			params.Set("sortOrder", listSortOrder)
		}
		if listFilter != "" {
			params.Set("filters", listFilter)
		}

		path := "/virtual-machines"
		if len(params) > 0 {
			path += "?" + params.Encode()
		}

		var vms []types.VM
		var total int
		if err := client.GetPaged(path, &vms, &total); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(types.VMListResult{Total: total, Data: vms})
		return nil
	},
}

// ── vm create ─────────────────────────────────────────────────────────────────

var (
	createName         string
	createImageID      string
	createFlavorID     string
	createSSHKeyName   string
	createSSHPublicKey string
	createInitScript   string
)

var vmCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new VM instance",
	Long: `Create a GPU VM with the specified image, flavor, and SSH key.

image_id and flavor_id must belong to the same region; otherwise the API will
reject the request.

The --ssh-public-key value is the full public key string, e.g.:
  ssh-ed25519 AAAAC3Nz... user@host

Use --init-script to pass a bash script that runs at first boot (cloud-init).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate all required flags up-front for a clear exit code 2.
		missing := []string{}
		if createName == "" {
			missing = append(missing, "--name")
		}
		if createImageID == "" {
			missing = append(missing, "--image-id")
		}
		if createFlavorID == "" {
			missing = append(missing, "--flavor-id")
		}
		if createSSHKeyName == "" {
			missing = append(missing, "--ssh-key-name")
		}
		if createSSHPublicKey == "" {
			missing = append(missing, "--ssh-public-key")
		}
		if len(missing) > 0 {
			exitInvalidArgs(fmt.Errorf("missing required flags: %v", missing))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		req := types.CreateVMRequest{
			Name:     createName,
			ImageID:  createImageID,
			FlavorID: createFlavorID,
			SSHKey: types.SSHKeyInput{
				Name:      createSSHKeyName,
				PublicKey: createSSHPublicKey,
			},
			InitScript: createInitScript,
		}

		var created types.CreateVMResponse
		if err := client.Post("/virtual-machines", req, &created); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(created)
		return nil
	},
}

// ── vm delete ─────────────────────────────────────────────────────────────────

var deleteForce bool

var vmDeleteCmd = &cobra.Command{
	Use:   "delete <instance-id>",
	Short: "Permanently delete a VM instance",
	Long: `Permanently deletes the specified virtual machine.

WARNING: This operation is irreversible. The server is immediately released,
becomes inaccessible, and all data is permanently erased. Billing ceases once
the server is deleted.

Pass --force to confirm. This flag is required to prevent accidental deletion
and to keep the command fully non-interactive for scripted/agent use.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceID := args[0]

		if !deleteForce {
			exitInvalidArgs(fmt.Errorf(
				"deleting a VM is irreversible and erases all data — pass --force to confirm",
			))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		if err := client.Delete("/virtual-machines/" + instanceID); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(map[string]string{
			"id":      instanceID,
			"status":  "deleted",
			"message": "virtual machine deleted successfully",
		})
		return nil
	},
}

func init() {
	// vm list flags
	vmListCmd.Flags().IntVar(&listLimit, "limit", 0, "Maximum number of VMs to return")
	vmListCmd.Flags().IntVar(&listOffset, "offset", 0, "Number of VMs to skip (for pagination)")
	vmListCmd.Flags().StringVar(&listSortField, "sort-field", "", "Field to sort by (e.g. name, status, started_time)")
	vmListCmd.Flags().StringVar(&listSortOrder, "sort-order", "", "Sort direction: asc or desc")
	vmListCmd.Flags().StringVar(&listFilter, "filter", "", `JSON filter array, e.g. '[{"key":"name","op":"contains","val":"hub"}]'`)

	// vm create flags
	vmCreateCmd.Flags().StringVar(&createName, "name", "", "Name for the new VM instance (required)")
	vmCreateCmd.Flags().StringVar(&createImageID, "image-id", "", "Image ID (must match flavor region) (required)")
	vmCreateCmd.Flags().StringVar(&createFlavorID, "flavor-id", "", "Flavor ID (hardware spec, must match image region) (required)")
	vmCreateCmd.Flags().StringVar(&createSSHKeyName, "ssh-key-name", "", "Name to assign to the SSH key (required)")
	vmCreateCmd.Flags().StringVar(&createSSHPublicKey, "ssh-public-key", "", "Public key string, e.g. 'ssh-ed25519 AAAA...' (required)")
	vmCreateCmd.Flags().StringVar(&createInitScript, "init-script", "", "Bash script to run at first boot (cloud-init, optional)")

	// vm delete flags
	vmDeleteCmd.Flags().BoolVar(&deleteForce, "force", false, "Confirm permanent deletion (required — all data will be erased)")

	vmCmd.AddCommand(vmListCmd)
	vmCmd.AddCommand(vmCreateCmd)
	vmCmd.AddCommand(vmDeleteCmd)

	rootCmd.AddCommand(vmCmd)
}
