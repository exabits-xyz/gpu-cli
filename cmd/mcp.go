package cmd

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/exabits-xyz/gpu-cli/internal/api"
	"github.com/exabits-xyz/gpu-cli/internal/types"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start the MCP server (stdio transport)",
	Long: `Starts an MCP (Model Context Protocol) server over stdio.

AI assistants such as Claude Desktop and Cursor can connect to this server
to manage Exabits GPU Cloud resources programmatically.

Authentication is read from the same sources as all other commands:
  ~/.exabits/config.yaml  or  EXABITS_API_TOKEN / EXABITS_ACCESS_TOKEN env vars.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCPServer()
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func runMCPServer() error {
	// Write config hint to stderr BEFORE starting the stdio server.
	// Anything written to stdout after this point must be valid JSON-RPC;
	// stderr is safe because MCP stdio transport only uses stdout.
	fmt.Fprintf(os.Stderr, `Exabits MCP server starting (stdio transport).

To connect this server to your AI assistant, add the following config:

  Claude Desktop  (~/.claude.json  or  ~/Library/Application Support/Claude/claude_desktop_config.json)
  Cursor          (~/.cursor/mcp.json)

  {
    "mcpServers": {
      "exabits": {
        "command": "egpu",
        "args": ["mcp"]
      }
    }
  }

Waiting for client connection...

`)

	s := server.NewMCPServer(
		"exabits-gpu",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// ── list_gpu_flavors ──────────────────────────────────────────────────────

	s.AddTool(
		mcp.NewTool("list_gpu_flavors",
			mcp.WithDescription(
				"Lists all available GPU hardware configurations (flavors) on Exabits GPU Cloud, "+
					"such as H100 and RTX_PRO_6000, grouped by region. "+
					"Call this tool first to discover valid flavor_id values before calling create_gpu_vm. "+
					"Each flavor includes GPU type, CPU, RAM, disk, price per hour, and stock availability.",
			),
			mcp.WithString("region_id",
				mcp.Description("Optional region ID to filter available GPU flavors."),
			),
		),
		handleListGPUFlavors,
	)

	// ── list_os_images ────────────────────────────────────────────────────────

	s.AddTool(
		mcp.NewTool("list_os_images",
			mcp.WithDescription(
				"Lists all available OS images on Exabits GPU Cloud, "+
					"including Ubuntu distributions and pre-configured AI/ML images. "+
					"Call this tool to discover valid image_id values before calling create_gpu_vm. "+
					"IMPORTANT: the image_id and flavor_id passed to create_gpu_vm must belong to the same region.",
			),
			mcp.WithString("region_id",
				mcp.Description("Optional region ID to filter available OS images."),
			),
		),
		handleListOSImages,
	)

	s.AddTool(
		mcp.NewTool("list_regions",
			mcp.WithDescription("Lists all Exabits GPU Cloud datacenter regions. Use these IDs to filter flavors, images, and volume types."),
		),
		handleListRegions,
	)

	s.AddTool(
		mcp.NewTool("list_gpu_vms",
			mcp.WithDescription("Lists GPU VM instances with optional pagination, sorting, and API filter JSON."),
			mcp.WithNumber("limit", mcp.Description("Optional maximum number of VMs to return.")),
			mcp.WithNumber("offset", mcp.Description("Optional number of VMs to skip.")),
			mcp.WithString("sort_field", mcp.Description("Optional sort field, e.g. name, status, started_time.")),
			mcp.WithString("sort_order", mcp.Description("Optional sort order: asc or desc.")),
			mcp.WithString("filter", mcp.Description(`Optional JSON filter array, e.g. [{"key":"status","op":"eq","val":"running"}].`)),
		),
		handleListGPUVMs,
	)

	s.AddTool(
		mcp.NewTool("get_gpu_vm",
			mcp.WithDescription("Retrieves full details of a single GPU VM instance."),
			mcp.WithString("instance_id", mcp.Required(), mcp.Description("VM instance ID.")),
		),
		handleGetGPUVM,
	)

	// ── create_gpu_vm ─────────────────────────────────────────────────────────

	s.AddTool(
		mcp.NewTool("create_gpu_vm",
			mcp.WithDescription(
				"Provisions a new GPU virtual machine on Exabits GPU Cloud. "+
					"Before calling this tool, use list_gpu_flavors to get a valid flavor_id and "+
					"list_os_images to get a valid image_id — both must belong to the same region. "+
					"Returns the new VM's ID and name. Billing begins immediately after creation.",
			),
			mcp.WithString("name",
				mcp.Required(),
				mcp.Description(
					"Human-readable name for the new VM instance, e.g. \"training-node-1\". "+
						"Must be unique within your account.",
				),
			),
			mcp.WithString("image_id",
				mcp.Required(),
				mcp.Description(
					"ID of the OS image to boot from. "+
						"Obtain valid IDs via list_os_images. "+
						"Must belong to the same region as flavor_id.",
				),
			),
			mcp.WithString("flavor_id",
				mcp.Required(),
				mcp.Description(
					"ID of the hardware flavor (GPU type and resource allocation). "+
						"Obtain valid IDs via list_gpu_flavors. "+
						"Must belong to the same region as image_id.",
				),
			),
			mcp.WithObject("ssh_key",
				mcp.Required(),
				mcp.Description(
					"SSH key to inject into the VM for remote access. "+
						"Must contain \"name\" (a short label for the key, e.g. \"my-laptop\") "+
						"and \"public_key\" (the full OpenSSH public key string, "+
						"e.g. \"ssh-ed25519 AAAAC3Nz... user@host\").",
				),
				mcp.Properties(map[string]any{
					"name": map[string]any{
						"type":        "string",
						"description": "Short label for the SSH key, e.g. \"my-laptop\"",
					},
					"public_key": map[string]any{
						"type":        "string",
						"description": "Full OpenSSH public key string, e.g. \"ssh-ed25519 AAAAC3Nz... user@host\"",
					},
				}),
			),
			mcp.WithString("init_script",
				mcp.Description("Optional bash/cloud-init script to run at first boot."),
			),
			mcp.WithBoolean("wait_for_running",
				mcp.Description("When true, poll until the VM reaches running status before returning full VM details."),
			),
		),
		handleCreateGPUVM,
	)

	s.AddTool(
		mcp.NewTool("start_gpu_vm",
			mcp.WithDescription("Starts a stopped GPU VM instance."),
			mcp.WithString("instance_id", mcp.Required(), mcp.Description("VM instance ID.")),
		),
		handleStartGPUVM,
	)

	s.AddTool(
		mcp.NewTool("stop_gpu_vm",
			mcp.WithDescription("Stops a GPU VM instance. The VM remains allocated and may continue to incur charges while stopped."),
			mcp.WithString("instance_id", mcp.Required(), mcp.Description("VM instance ID.")),
		),
		handleStopGPUVM,
	)

	s.AddTool(
		mcp.NewTool("reboot_gpu_vm",
			mcp.WithDescription("Hard-reboots a GPU VM instance."),
			mcp.WithString("instance_id", mcp.Required(), mcp.Description("VM instance ID.")),
		),
		handleRebootGPUVM,
	)

	s.AddTool(
		mcp.NewTool("get_gpu_vm_metrics",
			mcp.WithDescription("Retrieves CPU, memory, disk, and network metrics for a GPU VM."),
			mcp.WithString("instance_id", mcp.Required(), mcp.Description("VM instance ID.")),
			mcp.WithString("duration", mcp.Description("Optional metric window: 1h, 2h, 4h, 6h, 12h, 1d, 3d, 7d, 15d, or 30d.")),
		),
		handleGetGPUVMMetrics,
	)

	s.AddTool(
		mcp.NewTool("attach_volumes_to_gpu_vm",
			mcp.WithDescription("Attaches one or more non-bootable volumes to a GPU VM."),
			mcp.WithString("vm_id", mcp.Required(), mcp.Description("VM instance ID.")),
			mcp.WithArray("volume_ids", mcp.Required(), mcp.MinItems(1), mcp.WithStringItems(), mcp.Description("Volume IDs to attach.")),
		),
		handleAttachVolumesToGPUVM,
	)

	s.AddTool(
		mcp.NewTool("detach_volume_from_gpu_vm",
			mcp.WithDescription("Detaches a volume from a GPU VM."),
			mcp.WithString("vm_id", mcp.Required(), mcp.Description("VM instance ID.")),
			mcp.WithString("volume_id", mcp.Required(), mcp.Description("Volume ID to detach.")),
		),
		handleDetachVolumeFromGPUVM,
	)

	// ── delete_gpu_vm ─────────────────────────────────────────────────────────

	s.AddTool(
		mcp.NewTool("delete_gpu_vm",
			mcp.WithDescription(
				"Permanently terminates a GPU VM instance on Exabits GPU Cloud, stopping all billing. "+
					"WARNING: This operation is irreversible — the server is immediately released "+
					"and all data on it is permanently erased. "+
					"Obtain the instance_id from create_gpu_vm output or your Exabits console.",
			),
			mcp.WithString("instance_id",
				mcp.Required(),
				mcp.Description(
					"Unique ID of the VM instance to delete, e.g. \"abc-123-def\". "+
						"Obtain this from the create_gpu_vm response or the Exabits console.",
				),
			),
		),
		handleDeleteGPUVM,
	)

	// ── check_billing_balance ─────────────────────────────────────────────────

	s.AddTool(
		mcp.NewTool("check_billing_balance",
			mcp.WithDescription(
				"Retrieves the current account credit balance for the authenticated Exabits account. "+
					"Call this before provisioning GPU VMs to confirm sufficient funds are available. "+
					"An insufficient balance will cause create_gpu_vm to fail with a billing error.",
			),
		),
		handleCheckBillingBalance,
	)

	s.AddTool(
		mcp.NewTool("list_volumes",
			mcp.WithDescription("Lists block-storage volumes with optional pagination, sorting, and API filter JSON."),
			mcp.WithNumber("limit", mcp.Description("Optional maximum number of volumes to return.")),
			mcp.WithNumber("offset", mcp.Description("Optional number of volumes to skip.")),
			mcp.WithString("sort_field", mcp.Description("Optional sort field, e.g. name, status, created_time.")),
			mcp.WithString("sort_order", mcp.Description("Optional sort order: asc or desc.")),
			mcp.WithString("filter", mcp.Description(`Optional JSON filter array, e.g. [{"key":"name","op":"contains","val":"data"}].`)),
		),
		handleListVolumes,
	)

	s.AddTool(
		mcp.NewTool("list_volume_types",
			mcp.WithDescription("Lists volume storage backend types available in a region."),
			mcp.WithString("region_id", mcp.Required(), mcp.Description("Region ID. Obtain region IDs with list_regions.")),
		),
		handleListVolumeTypes,
	)

	s.AddTool(
		mcp.NewTool("create_volume",
			mcp.WithDescription("Creates a block-storage volume in a region. Optionally provide an image_id to make it bootable."),
			mcp.WithString("display_name", mcp.Required(), mcp.Description("Display name, 50 characters or fewer.")),
			mcp.WithString("region_id", mcp.Required(), mcp.Description("Region ID.")),
			mcp.WithString("type_id", mcp.Required(), mcp.Description("Volume type ID from list_volume_types.")),
			mcp.WithNumber("size", mcp.Required(), mcp.Description("Volume size in GB.")),
			mcp.WithString("image_id", mcp.Description("Optional OS image ID to make the volume bootable.")),
			mcp.WithString("description", mcp.Description("Optional volume description.")),
			mcp.WithString("payment_currency", mcp.Description("Optional payment currency, default USD.")),
		),
		handleCreateVolume,
	)

	s.AddTool(
		mcp.NewTool("delete_volume",
			mcp.WithDescription("Permanently deletes a volume. This operation is irreversible."),
			mcp.WithString("volume_id", mcp.Required(), mcp.Description("Volume ID to delete.")),
		),
		handleDeleteVolume,
	)

	s.AddTool(
		mcp.NewTool("get_billing_usage",
			mcp.WithDescription("Retrieves resource usage cost history with optional pagination, sorting, and API filter JSON."),
			mcp.WithNumber("limit", mcp.Description("Optional maximum number of records to return.")),
			mcp.WithNumber("offset", mcp.Description("Optional number of records to skip.")),
			mcp.WithString("sort_field", mcp.Description("Optional sort field, e.g. created_time, total_fee, status.")),
			mcp.WithString("sort_order", mcp.Description("Optional sort order: asc or desc.")),
			mcp.WithString("filter", mcp.Description(`Optional JSON filter array, e.g. [{"key":"status","op":"eq","val":"active"}].`)),
		),
		handleGetBillingUsage,
	)

	s.AddTool(
		mcp.NewTool("get_billing_statements",
			mcp.WithDescription("Retrieves billing statements with optional pagination, sorting, and API filter JSON."),
			mcp.WithNumber("limit", mcp.Description("Optional maximum number of statements to return.")),
			mcp.WithNumber("offset", mcp.Description("Optional number of statements to skip.")),
			mcp.WithString("sort_field", mcp.Description("Optional sort field, e.g. started_time, amount, status.")),
			mcp.WithString("sort_order", mcp.Description("Optional sort order: asc or desc.")),
			mcp.WithString("filter", mcp.Description(`Optional JSON filter array, e.g. [{"key":"resource_type","op":"eq","val":"vm"}].`)),
		),
		handleGetBillingStatements,
	)

	s.AddTool(
		mcp.NewTool("list_api_tokens",
			mcp.WithDescription("Lists API tokens with optional pagination and sorting."),
			mcp.WithNumber("limit", mcp.Description("Optional maximum number of tokens to return.")),
			mcp.WithNumber("offset", mcp.Description("Optional number of tokens to skip.")),
			mcp.WithString("sort_field", mcp.Description("Optional sort field, e.g. name, created_at, last_used.")),
			mcp.WithString("sort_order", mcp.Description("Optional sort order: asc or desc.")),
		),
		handleListAPITokens,
	)

	s.AddTool(
		mcp.NewTool("create_api_token",
			mcp.WithDescription("Creates a new long-lived API token. The returned token value should be stored securely."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Token name, 50 characters or fewer.")),
			mcp.WithString("description", mcp.Description("Optional token description.")),
			mcp.WithBoolean("save", mcp.Description("When true, save the generated token as api_token in ~/.exabits/config.yaml.")),
		),
		handleCreateAPIToken,
	)

	s.AddTool(
		mcp.NewTool("update_api_token",
			mcp.WithDescription("Updates the name and optional description of an API token."),
			mcp.WithString("token_id", mcp.Required(), mcp.Description("API token ID.")),
			mcp.WithString("name", mcp.Required(), mcp.Description("New token name, 50 characters or fewer.")),
			mcp.WithString("description", mcp.Description("New token description. Omit to clear.")),
		),
		handleUpdateAPIToken,
	)

	s.AddTool(
		mcp.NewTool("delete_api_token",
			mcp.WithDescription("Permanently deletes an API token. The token immediately stops authenticating API requests."),
			mcp.WithString("token_id", mcp.Required(), mcp.Description("API token ID to delete.")),
		),
		handleDeleteAPIToken,
	)

	s.AddTool(
		mcp.NewTool("generate_ssh_key",
			mcp.WithDescription("Generates a local ed25519 SSH key pair under ~/.exabits/keys for VM access."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Key name. Files are saved as egpu_<name> and egpu_<name>.pub.")),
		),
		handleGenerateSSHKey,
	)

	s.AddTool(
		mcp.NewTool("list_ssh_keys",
			mcp.WithDescription("Lists local SSH key pairs stored under ~/.exabits/keys."),
		),
		handleListSSHKeys,
	)

	s.AddTool(
		mcp.NewTool("delete_ssh_key",
			mcp.WithDescription("Deletes a local SSH key pair from ~/.exabits/keys."),
			mcp.WithString("name", mcp.Required(), mcp.Description("Key name to delete.")),
		),
		handleDeleteSSHKey,
	)

	return server.ServeStdio(s)
}

// ── handlers ──────────────────────────────────────────────────────────────────

func handleListGPUFlavors(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	path := "/flavors"
	if regionID := strings.TrimSpace(req.GetString("region_id", "")); regionID != "" {
		path += "?region_id=" + url.QueryEscape(regionID)
	}

	var flavors []types.FlavorGroup
	if err := client.Get(path, &flavors); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(flavors)
}

func handleListOSImages(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	path := "/images"
	if regionID := strings.TrimSpace(req.GetString("region_id", "")); regionID != "" {
		path += "?region_id=" + url.QueryEscape(regionID)
	}

	var images []types.Image
	if err := client.Get(path, &images); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(images)
}

func handleListRegions(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var regions []types.Region
	if err := client.Get("/regions", &regions); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(regions)
}

func handleListGPUVMs(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := listPathFromMCPArgs(req, "/virtual-machines", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var vms []types.VM
	var total int
	if err := client.GetPaged(path, &vms, &total); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(types.VMListResult{Total: total, Data: vms})
}

func handleGetGPUVM(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	instanceID, err := requireMCPString(req, "instance_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var vm types.VM
	if err := client.Get("/virtual-machines/"+url.PathEscape(instanceID), &vm); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(vm)
}

func handleCreateGPUVM(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	name, _ := args["name"].(string)
	if name == "" {
		return mcp.NewToolResultError("name is required and must be a non-empty string"), nil
	}

	imageID, _ := args["image_id"].(string)
	if imageID == "" {
		return mcp.NewToolResultError("image_id is required and must be a non-empty string"), nil
	}

	flavorID, _ := args["flavor_id"].(string)
	if flavorID == "" {
		return mcp.NewToolResultError("flavor_id is required and must be a non-empty string"), nil
	}

	sshKeyRaw, ok := args["ssh_key"].(map[string]any)
	if !ok {
		return mcp.NewToolResultError(
			"ssh_key must be an object with \"name\" and \"public_key\" string fields",
		), nil
	}
	sshName, _ := sshKeyRaw["name"].(string)
	sshPublicKey, _ := sshKeyRaw["public_key"].(string)
	if sshName == "" || sshPublicKey == "" {
		return mcp.NewToolResultError(
			"ssh_key.name and ssh_key.public_key are both required and must be non-empty strings",
		), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	createReq := types.CreateVMRequest{
		Name:       name,
		ImageID:    imageID,
		FlavorID:   flavorID,
		InitScript: req.GetString("init_script", ""),
		SSHKey: types.SSHKeyInput{
			Name:      sshName,
			PublicKey: sshPublicKey,
		},
	}

	var created types.CreateVMResponse
	if err := client.Post("/virtual-machines", createReq, &created); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if req.GetBool("wait_for_running", false) {
		vm, err := pollVMUntilRunning(client, created.ID)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return mcpResultJSON(vm)
	}

	return mcpResultJSON(created)
}

func handleStartGPUVM(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return handleGPUVMAction(req, "start", "starting")
}

func handleStopGPUVM(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return handleGPUVMAction(req, "stop", "stopping")
}

func handleRebootGPUVM(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return handleGPUVMAction(req, "reboot", "rebooting")
}

func handleGPUVMAction(req mcp.CallToolRequest, action, statusWord string) (*mcp.CallToolResult, error) {
	instanceID, err := requireMCPString(req, "instance_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.Get("/virtual-machines/"+url.PathEscape(instanceID)+"/"+action, nil); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(map[string]string{
		"id":      instanceID,
		"status":  statusWord,
		"message": "virtual machine " + statusWord + " successfully",
	})
}

func handleGetGPUVMMetrics(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	instanceID, err := requireMCPString(req, "instance_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	duration := strings.TrimSpace(req.GetString("duration", ""))
	if duration != "" && !validDurations[duration] {
		return mcp.NewToolResultError(
			fmt.Sprintf("invalid duration %q; valid values: 1h, 2h, 4h, 6h, 12h, 1d, 3d, 7d, 15d, 30d", duration),
		), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	path := "/virtual-machines/" + url.PathEscape(instanceID) + "/metrics"
	if duration != "" {
		path += "?duration=" + url.QueryEscape(duration)
	}

	var metrics types.VMMetrics
	if err := client.Get(path, &metrics); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(metrics)
}

func handleAttachVolumesToGPUVM(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	vmID, err := requireMCPString(req, "vm_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	volumeIDs, err := req.RequireStringSlice("volume_ids")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(volumeIDs) == 0 {
		return mcp.NewToolResultError("volume_ids must contain at least one volume ID"), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	reqBody := types.AttachVolumesRequest{VolumeIDs: volumeIDs}
	if err := client.Post("/virtual-machines/"+url.PathEscape(vmID)+"/volumes", reqBody, nil); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(map[string]any{
		"vm_id":      vmID,
		"volume_ids": volumeIDs,
		"message":    "volumes attached successfully",
	})
}

func handleDetachVolumeFromGPUVM(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	vmID, err := requireMCPString(req, "vm_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	volumeID, err := requireMCPString(req, "volume_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var result types.DetachVolumeResponse
	if err := client.DeleteParsed("/virtual-machines/"+url.PathEscape(vmID)+"/volumes/"+url.PathEscape(volumeID), &result); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(result)
}

func handleDeleteGPUVM(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	instanceID, _ := args["instance_id"].(string)
	if instanceID == "" {
		return mcp.NewToolResultError("instance_id is required and must be a non-empty string"), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.Delete("/virtual-machines/" + instanceID); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf(
		"VM instance %q has been permanently deleted. Billing has stopped.", instanceID,
	)), nil
}

func handleCheckBillingBalance(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var balance types.CreditBalance
	if err := client.Get("/billing/balance", &balance); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(balance)
}

func handleListVolumes(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := listPathFromMCPArgs(req, "/volumes", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var volumes []types.Volume
	var total int
	if err := client.GetPaged(path, &volumes, &total); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(types.VolumeListResult{Total: total, Data: volumes})
}

func handleListVolumeTypes(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	regionID, err := requireMCPString(req, "region_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var volumeTypes []types.VolumeType
	if err := client.Get("/volume-types?region_id="+url.QueryEscape(regionID), &volumeTypes); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(volumeTypes)
}

func handleCreateVolume(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	displayName, err := requireMCPString(req, "display_name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(displayName) > 50 {
		return mcp.NewToolResultError(fmt.Sprintf("display_name must be 50 characters or fewer (got %d)", len(displayName))), nil
	}
	regionID, err := requireMCPString(req, "region_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	typeID, err := requireMCPString(req, "type_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	size, err := req.RequireInt("size")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if size <= 0 {
		return mcp.NewToolResultError("size must be greater than 0"), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	reqBody := types.CreateVolumeRequest{
		DisplayName:     displayName,
		RegionID:        regionID,
		TypeID:          typeID,
		Size:            size,
		ImageID:         req.GetString("image_id", ""),
		Description:     req.GetString("description", ""),
		PaymentCurrency: req.GetString("payment_currency", ""),
	}

	var created types.CreateVolumeResponse
	if err := client.Post("/volumes", reqBody, &created); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(created)
}

func handleDeleteVolume(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	volumeID, err := requireMCPString(req, "volume_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.Delete("/volumes/" + url.PathEscape(volumeID)); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(map[string]string{
		"id":      volumeID,
		"status":  "deleted",
		"message": "volume deleted successfully",
	})
}

func handleGetBillingUsage(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := listPathFromMCPArgs(req, "/billing/resources/usage", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var records []types.UsageRecord
	var total int
	if err := client.GetPaged(path, &records, &total); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(types.UsageListResult{Total: total, Data: records})
}

func handleGetBillingStatements(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := listPathFromMCPArgs(req, "/billing/resources/statements", true)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var statements []types.Statement
	var total int
	if err := client.GetPaged(path, &statements, &total); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(types.StatementListResult{Total: total, Data: statements})
}

func handleListAPITokens(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	path, err := listPathFromMCPArgs(req, "/api-tokens", false)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var tokens []types.APIToken
	var total int
	if err := client.GetPaged(path, &tokens, &total); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(types.TokenListResult{Total: total, Data: tokens})
}

func handleCreateAPIToken(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := requireMCPString(req, "name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(name) > 50 {
		return mcp.NewToolResultError(fmt.Sprintf("name must be 50 characters or fewer (got %d)", len(name))), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	reqBody := types.CreateTokenRequest{
		Name:        name,
		Description: req.GetString("description", ""),
	}

	var token types.APIToken
	if err := client.Post("/api-tokens", reqBody, &token); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if req.GetBool("save", false) {
		if err := saveConfigKeys(map[string]any{"api_token": token.Token}); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("token created but could not save to config: %s", err)), nil
		}
	}

	return mcpResultJSON(token)
}

func handleUpdateAPIToken(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tokenID, err := requireMCPString(req, "token_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	name, err := requireMCPString(req, "name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if len(name) > 50 {
		return mcp.NewToolResultError(fmt.Sprintf("name must be 50 characters or fewer (got %d)", len(name))), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	reqBody := types.CreateTokenRequest{
		Name:        name,
		Description: req.GetString("description", ""),
	}

	var token types.APIToken
	if err := client.Put("/api-tokens/"+url.PathEscape(tokenID), reqBody, &token); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(token)
}

func handleDeleteAPIToken(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tokenID, err := requireMCPString(req, "token_id")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	if err := client.Delete("/api-tokens/" + url.PathEscape(tokenID)); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(map[string]string{
		"id":      tokenID,
		"status":  "deleted",
		"message": "API token deleted successfully",
	})
}

func handleGenerateSSHKey(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := requireMCPString(req, "name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	dir, err := keyDir()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	privPath, pubPath := keyPaths(dir, name)
	if _, err := os.Stat(privPath); err == nil {
		return mcp.NewToolResultError(fmt.Sprintf("key %q already exists at %s", name, privPath)), nil
	}

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("key generation failed: %s", err)), nil
	}

	privPEMBlock, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to encode private key: %s", err)), nil
	}
	privPEMBytes := pem.EncodeToMemory(privPEMBlock)

	sshPub, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to encode public key: %s", err)), nil
	}
	authorizedKey := strings.TrimRight(string(ssh.MarshalAuthorizedKey(sshPub)), "\n")
	fingerprint := ssh.FingerprintSHA256(sshPub)

	if err := os.WriteFile(privPath, privPEMBytes, 0600); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to write private key: %s", err)), nil
	}
	if err := os.WriteFile(pubPath, []byte(authorizedKey+"\n"), 0644); err != nil {
		_ = os.Remove(privPath)
		return mcp.NewToolResultError(fmt.Sprintf("failed to write public key: %s", err)), nil
	}

	return mcpResultJSON(types.LocalSSHKey{
		Name:           name,
		PrivateKeyPath: privPath,
		PublicKeyPath:  pubPath,
		PublicKey:      authorizedKey,
		Fingerprint:    fingerprint,
	})
}

func handleListSSHKeys(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	dir, err := keyDir()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("cannot read key directory: %s", err)), nil
	}

	keys := []types.LocalSSHKey{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".pub") {
			continue
		}

		pubPath := dir + string(os.PathSeparator) + e.Name()
		raw, err := os.ReadFile(pubPath)
		if err != nil {
			continue
		}

		pubKeyStr := strings.TrimSpace(string(raw))
		sshPub, _, _, _, err := ssh.ParseAuthorizedKey(raw)
		if err != nil {
			continue
		}

		base := strings.TrimSuffix(e.Name(), ".pub")
		name := strings.TrimPrefix(base, "egpu_")

		keys = append(keys, types.LocalSSHKey{
			Name:           name,
			PrivateKeyPath: dir + string(os.PathSeparator) + base,
			PublicKeyPath:  pubPath,
			PublicKey:      pubKeyStr,
			Fingerprint:    ssh.FingerprintSHA256(sshPub),
		})
	}

	return mcpResultJSON(keys)
}

func handleDeleteSSHKey(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := requireMCPString(req, "name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	dir, err := keyDir()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	privPath, pubPath := keyPaths(dir, name)
	_, privErr := os.Stat(privPath)
	_, pubErr := os.Stat(pubPath)
	if os.IsNotExist(privErr) && os.IsNotExist(pubErr) {
		return mcp.NewToolResultError(fmt.Sprintf("no key named %q found in %s", name, dir)), nil
	}

	var errs []string
	if err := os.Remove(privPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Sprintf("private key: %v", err))
	}
	if err := os.Remove(pubPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, fmt.Sprintf("public key: %v", err))
	}
	if len(errs) > 0 {
		return mcp.NewToolResultError("failed to delete: " + strings.Join(errs, "; ")), nil
	}

	return mcpResultJSON(map[string]string{
		"name":    name,
		"status":  "deleted",
		"message": fmt.Sprintf("key pair %q deleted from %s", name, dir),
	})
}

func requireMCPString(req mcp.CallToolRequest, key string) (string, error) {
	value, err := req.RequireString(key)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required and must be a non-empty string", key)
	}
	return value, nil
}

func listPathFromMCPArgs(req mcp.CallToolRequest, basePath string, includeFilter bool) (string, error) {
	sortOrder := strings.TrimSpace(req.GetString("sort_order", ""))
	if sortOrder != "" && sortOrder != "asc" && sortOrder != "desc" {
		return "", fmt.Errorf("sort_order must be \"asc\" or \"desc\", got %q", sortOrder)
	}

	args := req.GetArguments()
	params := url.Values{}
	if _, ok := args["limit"]; ok {
		limit, err := req.RequireInt("limit")
		if err != nil {
			return "", err
		}
		params.Set("limit", strconv.Itoa(limit))
	}
	if _, ok := args["offset"]; ok {
		offset, err := req.RequireInt("offset")
		if err != nil {
			return "", err
		}
		params.Set("offset", strconv.Itoa(offset))
	}
	if sortField := strings.TrimSpace(req.GetString("sort_field", "")); sortField != "" {
		params.Set("sortField", sortField)
	}
	if sortOrder != "" {
		params.Set("sortOrder", sortOrder)
	}
	if includeFilter {
		if filter := strings.TrimSpace(req.GetString("filter", "")); filter != "" {
			params.Set("filters", filter)
		}
	}

	if len(params) == 0 {
		return basePath, nil
	}
	return basePath + "?" + params.Encode(), nil
}

// mcpResultJSON marshals v as indented JSON and wraps it in a tool text result.
func mcpResultJSON(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to encode response: %s", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
