# Workflow: Set Up the MCP Server

Connect an AI assistant such as Claude Desktop, Claude Code, or Cursor to Exabits GPU Cloud through the `egpu mcp` stdio server.

## Prerequisites

Load [../references/cli.md](../references/cli.md) before running any commands.
`egpu` must be installed and authenticated. Load [../references/installation.md](../references/installation.md) if installation or credentials are missing.

---

## What the MCP Server Provides

`egpu mcp` exposes Exabits cloud operations directly as MCP tools. It uses the same config, auth precedence, API client, retries, response decoding, and local key helpers as the CLI.

Supported tool groups:

| Area | Tools |
|---|---|
| Discovery | `list_regions`, `list_gpu_flavors`, `list_os_images` |
| VMs | `list_gpu_vms`, `get_gpu_vm`, `create_gpu_vm`, `start_gpu_vm`, `stop_gpu_vm`, `reboot_gpu_vm`, `get_gpu_vm_metrics`, `delete_gpu_vm` |
| VM volumes | `attach_volumes_to_gpu_vm`, `detach_volume_from_gpu_vm` |
| Volumes | `list_volumes`, `list_volume_types`, `create_volume`, `delete_volume` |
| Billing | `check_billing_balance`, `get_billing_usage`, `get_billing_statements` |
| API tokens | `list_api_tokens`, `create_api_token`, `update_api_token`, `delete_api_token` |
| Local SSH keys | `generate_ssh_key`, `list_ssh_keys`, `delete_ssh_key` |

Before calling destructive tools (`delete_gpu_vm`, `delete_volume`, `delete_api_token`, `delete_ssh_key`), ask the user for explicit confirmation.

---

## Steps

### 1. Verify `egpu` is on PATH and authenticated

```bash
which egpu
egpu --help
egpu billing balance
```

The MCP config references the binary by name, so it must be findable by the assistant process.

### 2. Configure the AI assistant

#### Claude Desktop

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` on macOS or `~/.claude.json`:

```json
{
  "mcpServers": {
    "exabits": {
      "command": "egpu",
      "args": ["mcp"]
    }
  }
}
```

#### Cursor

Edit `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "exabits": {
      "command": "egpu",
      "args": ["mcp"]
    }
  }
}
```

#### Claude Code in a project

Add `.claude/mcp.json` in the project root:

```json
{
  "mcpServers": {
    "exabits": {
      "command": "egpu",
      "args": ["mcp"]
    }
  }
}
```

### 3. Pass credentials when needed

The MCP server reads credentials from `~/.exabits/config.yaml` and `EXABITS_*`. For agents, a long-lived API token is preferred:

```json
{
  "mcpServers": {
    "exabits": {
      "command": "egpu",
      "args": ["mcp"],
      "env": {
        "EXABITS_API_TOKEN": "<your-api-token>"
      }
    }
  }
}
```

If `~/.exabits/config.yaml` already contains `api_token` or `api_token_encrypted`, no env block is required.

### 4. Restart and verify

Restart the assistant after editing config. Then ask:

> "List available GPU flavors on Exabits."

The assistant should call `list_gpu_flavors` and return JSON hardware options. If it fails, check:

1. `egpu` is on PATH for the assistant process
2. Credentials work in a terminal with `egpu billing balance`
3. The MCP JSON config is valid and in the expected file

---

## Manual Debugging

```bash
egpu mcp
```

Startup diagnostics are written to stderr. Stdout is reserved for MCP JSON-RPC, so do not print anything else there.
