---
name: exabits-gpu-manager
description: "GPU cloud management powered by the Exabits CLI: provision H100/H200/GB200/RTX GPU virtual machines, manage instance lifecycle (start/stop/reboot/delete), attach block-storage volumes, monitor billing, manage API tokens, and connect AI assistants via MCP. Use when the user wants to deploy compute, run AI workloads, manage GPU servers, or configure cloud infrastructure on Exabits."
metadata:
  openclaw:
    requires:
      bins:
        - egpu
---

# Exabits GPU Manager

Skills for managing GPU cloud infrastructure on [Exabits](https://gpu.exabits.ai) — from provisioning H100/H200 instances to monitoring billing and connecting AI assistants via MCP.

## References

Load these files when needed — do not load all of them upfront:

- [references/cli.md](references/cli.md) — full `egpu` command reference with all flags and examples. Load before running any `egpu` command or looking up syntax.
- [references/installation.md](references/installation.md) — installation and authentication setup. Load if `egpu` is not installed, not on `$PATH`, or credentials are missing.

## Environment Check

Before running any workflow, verify the CLI is available and authenticated:

```bash
egpu --help
egpu billing balance
```

If `egpu` is not installed or credentials are not configured, load [references/installation.md](references/installation.md) and stop until setup is complete.

## CLI Reference

Before running any `egpu` command, load [references/cli.md](references/cli.md) as the source of truth for exact flags, subcommands, and examples. Do not guess command syntax from memory.

## MCP Usage

When an Exabits MCP server is available, prefer MCP tools for normal cloud operations because they use the same API client without shell parsing. Still load [references/cli.md](references/cli.md) first to confirm exact tool names and argument shapes. Use CLI commands when the user asks for terminal commands, when MCP is not connected, or when debugging the local installation.

Destructive MCP tools are non-interactive. Before calling `delete_gpu_vm`, `delete_volume`, `delete_api_token`, or `delete_ssh_key`, ask the user for explicit confirmation just as you would before passing `--force` to the CLI.

## Workflow Routing

---

**`workflows/provision-vm.md`**
- "Spin up a GPU server" / "Provision a VM" / "Create a new instance"
- "Deploy [model name] to a GPU" / "I need compute for training"
- "Give me an H100 / H200 / RTX" / "I need a GPU"
- "Launch a machine with [N] GPUs"
- "Start a cloud instance for inference / fine-tuning / experimentation"

---

**`workflows/manage-vm.md`**
- "List my VMs" / "What instances do I have running?"
- "Start / stop / reboot my instance"
- "Check the status of my VM"
- "Show metrics for my server" / "How much CPU/memory is my VM using?"
- "Get details about instance [id]"
- "I want to pause my VM without deleting it"
- "Attach / detach storage to a VM" if the focus is VM lifecycle

---

**`workflows/delete-vm.md`**
- "Delete my VM" / "Terminate this instance" / "Shut it down permanently"
- "I'm done with this server, clean it up"
- "Remove instance [id]" / "Destroy the VM and stop billing"

---

**`workflows/manage-volumes.md`**
- "Create a storage volume" / "Add more disk space to my VM"
- "Attach / detach a volume"
- "List my volumes" / "What storage do I have?"
- "Delete a volume" / "Clean up unused storage"

---

**`workflows/check-billing.md`**
- "What's my account balance?" / "Do I have enough credits?"
- "Show my usage" / "How much have I spent?"
- "Get a billing statement" / "Show my invoice"
- "Check if I have enough funds before provisioning"

---

**`workflows/manage-tokens.md`**
- "Create an API token" / "Generate a long-lived credential"
- "List my tokens" / "What API keys do I have?"
- "Rename / update a token" / "Delete an expired token"
- "Set up a token for CI" / "I need a non-expiring credential for automation"

---

**`workflows/setup-mcp.md`**
- "Connect Claude / Cursor to Exabits" / "Set up the MCP server"
- "I want my AI assistant to manage GPU instances"
- "Configure MCP for Claude Desktop / Cursor"
- "Start the MCP server" / "How do I wire up the AI tools?"

---

Once you identify the right workflow, load that file and follow its instructions exactly.

If the user's intent matches more than one workflow, ask one clarifying question before routing. If it matches none, ask what they are trying to accomplish. Do not guess.

Some requests can be handled directly with the CLI without loading a workflow. Load [references/cli.md](references/cli.md) and execute directly when the user's intent is a single, self-contained operation:

- List available GPU hardware — `egpu flavor list`
- List local SSH keys — `egpu key list`
- Generate a local SSH key — `egpu key generate --name <name>`
- Show the current CLI configuration — `egpu config show`
- Search for a specific instance by name — `egpu vm list --filter '...'`
