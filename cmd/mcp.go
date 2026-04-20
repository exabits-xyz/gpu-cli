package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/exabits/gpu-cli/internal/api"
	"github.com/exabits/gpu-cli/internal/types"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
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
		),
		handleListOSImages,
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
		),
		handleCreateGPUVM,
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

	return server.ServeStdio(s)
}

// ── handlers ──────────────────────────────────────────────────────────────────

func handleListGPUFlavors(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var flavors []types.FlavorGroup
	if err := client.Get("/flavors", &flavors); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(flavors)
}

func handleListOSImages(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	client, err := api.NewClient()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	var images []types.Image
	if err := client.Get("/images", &images); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(images)
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
		Name:     name,
		ImageID:  imageID,
		FlavorID: flavorID,
		SSHKey: types.SSHKeyInput{
			Name:      sshName,
			PublicKey: sshPublicKey,
		},
	}

	var created types.CreateVMResponse
	if err := client.Post("/virtual-machines", createReq, &created); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcpResultJSON(created)
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

// mcpResultJSON marshals v as indented JSON and wraps it in a tool text result.
func mcpResultJSON(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to encode response: %s", err)), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}
