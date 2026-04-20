package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"time"

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

const (
	pollInterval   = 5 * time.Second
	pollTimeout    = 10 * time.Minute
	spinnerClearFmt = "\r%-60s\r" // overwrite + rewind without leaving stale chars
)

var spinnerFrames = []string{"|", "/", "-", "\\"}

var (
	createName         string
	createImageID      string
	createFlavorID     string
	createSSHKeyName   string
	createSSHPublicKey string
	createInitScript   string
	createNoWait       bool
)

var vmCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new VM instance",
	Long: `Create a GPU VM with the specified image, flavor, and SSH key.

image_id and flavor_id must belong to the same region; otherwise the API will
reject the request.

The --ssh-public-key value is the full public key string, e.g.:
  ssh-ed25519 AAAAC3Nz... user@host

Use --init-script to pass a bash script that runs at first boot (cloud-init).
Use --no-wait to return immediately after the create request without polling.`,
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

		// --no-wait: return the minimal create response and exit immediately.
		if createNoWait {
			printJSON(created)
			return nil
		}

		// Poll until the VM reaches "running", streaming progress to stderr.
		vm, err := pollVMUntilRunning(client, created.ID)
		if err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(vm)
		return nil
	},
}

// pollVMUntilRunning polls GET /virtual-machines/{id} until status == "running"
// or the 10-minute timeout is reached.
//
// Progress output goes to stderr so stdout remains pure JSON:
//   • TTY stderr    — animated spinner with elapsed time (overwritten each tick)
//   • Piped stderr  — one JSON object per poll for agent consumption
func pollVMUntilRunning(client *api.Client, instanceID string) (*types.VM, error) {
	start := time.Now()
	deadline := start.Add(pollTimeout)
	tty := stderrIsTTY()

	for attempt := 0; time.Now().Before(deadline); attempt++ {
		var vm types.VM
		if err := client.Get("/virtual-machines/"+instanceID, &vm); err != nil {
			// Tolerate the first few polls — the VM may not be queryable yet.
			if attempt < 3 {
				writePollProgress(tty, attempt, instanceID, "provisioning", time.Since(start))
				time.Sleep(pollInterval)
				continue
			}
			if tty {
				fmt.Fprintf(os.Stderr, spinnerClearFmt, "")
			}
			return nil, fmt.Errorf("polling failed: %w", err)
		}

		elapsed := time.Since(start).Round(time.Second)

		if vm.Status == "running" {
			if tty {
				fmt.Fprintf(os.Stderr, spinnerClearFmt, "")
				fmt.Fprintf(os.Stderr, "✓ VM %s is running (%s)\n", instanceID, elapsed)
			} else {
				writeProgressJSON(instanceID, "running", elapsed)
			}
			return &vm, nil
		}

		writePollProgress(tty, attempt, instanceID, vm.Status, elapsed)
		time.Sleep(pollInterval)
	}

	if tty {
		fmt.Fprintf(os.Stderr, spinnerClearFmt, "")
	}
	return nil, fmt.Errorf(
		"timeout: VM %s did not reach 'running' within %v", instanceID, pollTimeout,
	)
}

// writePollProgress emits one progress tick to stderr.
func writePollProgress(tty bool, attempt int, id, status string, elapsed time.Duration) {
	if tty {
		frame := spinnerFrames[attempt%len(spinnerFrames)]
		fmt.Fprintf(os.Stderr, "\r%s  VM %s — %s (%s)...",
			frame, id, status, elapsed)
	} else {
		writeProgressJSON(id, status, elapsed)
	}
}

// writeProgressJSON emits a single-line JSON progress event to stderr.
func writeProgressJSON(id, status string, elapsed time.Duration) {
	enc := json.NewEncoder(os.Stderr)
	_ = enc.Encode(map[string]string{
		"id":      id,
		"status":  status,
		"elapsed": elapsed.Round(time.Second).String(),
	})
}

// stderrIsTTY reports whether stderr is an interactive terminal.
func stderrIsTTY() bool {
	stat, err := os.Stderr.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// ── vm get ────────────────────────────────────────────────────────────────────

var vmGetCmd = &cobra.Command{
	Use:   "get <instance-id>",
	Short: "Retrieve details of a VM instance",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceID := args[0]

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		var vm types.VM
		if err := client.Get("/virtual-machines/"+instanceID, &vm); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(vm)
		return nil
	},
}

// ── vm metrics ────────────────────────────────────────────────────────────────

var metricsDuration string

// validDurations is the full set accepted by the API.
var validDurations = map[string]bool{
	"1h": true, "2h": true, "4h": true, "6h": true, "12h": true,
	"1d": true, "3d": true, "7d": true, "15d": true, "30d": true,
}

var vmMetricsCmd = &cobra.Command{
	Use:   "metrics <instance-id>",
	Short: "Retrieve CPU, memory, disk, and network metrics for a VM",
	Long: `Fetches time-series performance metrics for the specified virtual machine.

Use --duration to limit the window. Valid values:
  1h  2h  4h  6h  12h  1d  3d  7d  15d  30d

Omitting --duration returns all recorded data.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		instanceID := args[0]

		if metricsDuration != "" && !validDurations[metricsDuration] {
			exitInvalidArgs(fmt.Errorf(
				"invalid --duration %q — valid values: 1h, 2h, 4h, 6h, 12h, 1d, 3d, 7d, 15d, 30d",
				metricsDuration,
			))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		path := "/virtual-machines/" + instanceID + "/metrics"
		if metricsDuration != "" {
			path += "?duration=" + metricsDuration
		}

		var metrics types.VMMetrics
		if err := client.Get(path, &metrics); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(metrics)
		return nil
	},
}

// ── vm start / stop / reboot ──────────────────────────────────────────────────

// vmAction builds a command that calls GET /virtual-machines/{id}/<action>
// and emits a simple {id, status, message} JSON object on success.
func vmAction(action, short, long, statusWord string) *cobra.Command {
	return &cobra.Command{
		Use:   action + " <instance-id>",
		Short: short,
		Long:  long,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			instanceID := args[0]

			client, err := api.NewClient()
			if err != nil {
				exitAPIError(err)
				return nil
			}

			if err := client.Get("/virtual-machines/"+instanceID+"/"+action, nil); err != nil {
				exitAPIError(err)
				return nil
			}

			printJSON(map[string]string{
				"id":      instanceID,
				"status":  statusWord,
				"message": "virtual machine " + statusWord + " successfully",
			})
			return nil
		},
	}
}

var vmStartCmd = vmAction(
	"start",
	"Start a stopped VM instance",
	"Initiates startup of the specified virtual machine.",
	"starting",
)

var vmStopCmd = vmAction(
	"stop",
	"Stop a running VM instance",
	`Shuts down the specified virtual machine.

NOTE: The VM remains intact and continues to incur charges while stopped.
      It can be restarted at any time. To release the instance entirely, use 'egpu vm delete'.`,
	"stopping",
)

var vmRebootCmd = vmAction(
	"reboot",
	"Hard-reboot a VM instance",
	"Performs a hard reboot of the specified virtual machine (equivalent to a physical power cycle).",
	"rebooting",
)

// ── vm volume (attach / detach) ───────────────────────────────────────────────

var vmVolumeCmd = &cobra.Command{
	Use:   "volume",
	Short: "Attach or detach volumes on a VM instance",
}

var attachVolumeIDs []string

var vmVolumeAttachCmd = &cobra.Command{
	Use:   "attach <vm-id>",
	Short: "Attach one or more volumes to a VM instance",
	Long: `Attaches non-bootable volumes to the specified virtual machine.

--volume-ids accepts a comma-separated list or can be repeated:
  --volume-ids id1,id2
  --volume-ids id1 --volume-ids id2`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmID := args[0]

		if len(attachVolumeIDs) == 0 {
			exitInvalidArgs(fmt.Errorf("--volume-ids is required"))
			return nil
		}

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		req := types.AttachVolumesRequest{VolumeIDs: attachVolumeIDs}
		if err := client.Post("/virtual-machines/"+vmID+"/volumes", req, nil); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(map[string]any{
			"vm_id":      vmID,
			"volume_ids": attachVolumeIDs,
			"message":    "volumes attached successfully",
		})
		return nil
	},
}

var vmVolumeDetachCmd = &cobra.Command{
	Use:   "detach <vm-id> <volume-id>",
	Short: "Detach a volume from a VM instance",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			exitInvalidArgs(fmt.Errorf("requires exactly 2 arguments: <vm-id> <volume-id>"))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		vmID, volumeID := args[0], args[1]

		client, err := api.NewClient()
		if err != nil {
			exitAPIError(err)
			return nil
		}

		var result types.DetachVolumeResponse
		if err := client.DeleteParsed("/virtual-machines/"+vmID+"/volumes/"+volumeID, &result); err != nil {
			exitAPIError(err)
			return nil
		}

		printJSON(result)
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
	vmCreateCmd.Flags().BoolVar(&createNoWait, "no-wait", false, "Return immediately after the create request without polling for 'running' status")

	// vm delete flags
	vmDeleteCmd.Flags().BoolVar(&deleteForce, "force", false, "Confirm permanent deletion (required — all data will be erased)")

	// vm metrics flags
	vmMetricsCmd.Flags().StringVar(&metricsDuration, "duration", "", "Time window: 1h, 2h, 4h, 6h, 12h, 1d, 3d, 7d, 15d, 30d (default: all)")

	// vm volume flags
	vmVolumeAttachCmd.Flags().StringSliceVar(&attachVolumeIDs, "volume-ids", nil, "Comma-separated volume IDs to attach (required)")

	vmVolumeCmd.AddCommand(vmVolumeAttachCmd)
	vmVolumeCmd.AddCommand(vmVolumeDetachCmd)

	vmCmd.AddCommand(vmListCmd)
	vmCmd.AddCommand(vmGetCmd)
	vmCmd.AddCommand(vmCreateCmd)
	vmCmd.AddCommand(vmStartCmd)
	vmCmd.AddCommand(vmStopCmd)
	vmCmd.AddCommand(vmRebootCmd)
	vmCmd.AddCommand(vmMetricsCmd)
	vmCmd.AddCommand(vmVolumeCmd)
	vmCmd.AddCommand(vmDeleteCmd)

	rootCmd.AddCommand(vmCmd)
}
