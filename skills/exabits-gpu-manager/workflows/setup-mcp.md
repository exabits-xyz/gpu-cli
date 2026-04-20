# Workflow: Set Up the MCP Server

Connect an AI assistant (Claude Desktop, Claude Code, Cursor) to the Exabits GPU Cloud via the Model Context Protocol (MCP) server built into the `egpu` CLI.

## Prerequisites

Load [../references/cli.md](../references/cli.md) before running any commands.
`egpu` must be installed and authenticated — load [../references/installation.md](../references/installation.md) if not.

---

## What the MCP Server Provides

Running `egpu mcp` starts a stdio-based MCP server that exposes five tools to any connected AI assistant:

| Tool | Description |
|---|---|
| `list_gpu_flavors` | List all GPU hardware configurations grouped by region, including stock and pricing |
| `list_os_images` | List all available OS images per region |
| `create_gpu_vm` | Provision a new GPU VM (requires `name`, `image_id`, `flavor_id`, `ssh_key`) |
| `delete_gpu_vm` | Permanently delete a VM — irreversible, always confirm before calling |
| `check_billing_balance` | Retrieve the current account credit balance |

---

## Steps

### 1. Verify `egpu` is on your PATH

```bash
which egpu
egpu --version
```

The MCP config references the binary by name — it must be findable without a full path.

### 2. Configure your AI assistant

#### Claude Desktop

Edit `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS) or `~/.claude.json`:

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

#### Claude Code (in-project)

Add to `.claude/mcp.json` in your project root:

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

### 3. Pass credentials to the server process

The MCP server reads credentials from the same sources as all other `egpu` commands. The recommended approach for MCP use is an API token passed via environment variable — include it in the server config:

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

Alternatively, if `~/.exabits/config.yaml` already contains a valid `api_token`, no additional env config is needed.

### 4. Restart your AI assistant

Fully restart the application after editing the config file. The MCP server starts on demand when the assistant first calls one of its tools.

### 5. Verify the connection

Ask your AI assistant:

> "List available GPU flavors on Exabits."

The assistant should call `list_gpu_flavors` and return a JSON list of hardware options. If it does not, check:

1. The `egpu` binary is on `$PATH` for the process that runs the assistant
2. Credentials are valid — run `egpu billing balance` in a terminal to verify
3. The config JSON is valid (no trailing commas, correct file path)

---

## Start the Server Manually (debugging)

To test the MCP server in isolation:

```bash
egpu mcp
```

The server starts and waits on stdin for JSON-RPC messages. Startup diagnostics are written to stderr. Press `Ctrl+C` to stop.
