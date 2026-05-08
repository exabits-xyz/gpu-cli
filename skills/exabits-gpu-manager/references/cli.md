# CLI and MCP Reference

Full reference for the `egpu` CLI and its built-in MCP server. CLI commands emit machine-parseable JSON on stdout and write errors/progress to stderr. Destructive CLI commands require `--force`.

---

## Global Behavior

| Item | Behavior |
|---|---|
| Config | `~/.exabits/config.yaml` plus `EXABITS_*` environment variables |
| Auth precedence | `api_token` -> `api_token_encrypted` -> `access_token` + `refresh_token` |
| API host | `api_url`, default `https://gpu-api.exascalelabs.ai`; `/api/v1` is appended automatically |
| JSON output | stdout is JSON; `--json` also makes errors JSON |
| Piped output | stdout auto-detects non-TTY and stays JSON for agent/script use |

Exit codes:

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | API, network, auth, or internal error |
| `2` | Invalid arguments |

---

## Authentication and Config

### `egpu auth`

Browser-based login. Opens `auth_url` or derives `https://gpu.exascalelabs.ai/login?state=...`, then saves an encrypted API token to `~/.exabits/config.yaml`.

```bash
egpu auth [--no-browser]
```

### `egpu auth login`

Legacy username/password login. The CLI MD5-hashes the plaintext password before sending it and saves `access_token` plus `refresh_token`.

```bash
egpu auth login --username <email> --password <plaintext-password>
```

### `egpu config`

```bash
egpu config set <api_key|api_token|api_token_encrypted|api_url|auth_url|access_token|refresh_token> <value>
egpu config get <key> [--show-full]
egpu config show [--show-full]
egpu config unset <key>
```

Sensitive values are masked unless `--show-full` is passed.

---

## Discovery

### Regions

```bash
egpu region list
```

### GPU flavors

List GPU hardware configurations grouped by region.

```bash
egpu flavor list [--region-id <region-id>]
```

Each product includes `id`, `name`, `region_id`, `region_name`, `gpu`, `gpu_count`, `cpu`, `ram`, `disk`, `price`, and `stock_available`.

### OS images

```bash
egpu image list [--region-id <region-id>]
```

`image_id` and `flavor_id` used for VM creation must belong to the same region.

---

## VM Commands

### List

```bash
egpu vm list [--limit <int>] [--offset <int>] [--sort-field <field>] [--sort-order asc|desc] [--filter '<json-array>']
```

Filter operators: `contains`, `eq`, `ne`, `gt`, `lt`.

Examples:

```bash
egpu vm list
egpu vm list --limit 10 --sort-field started_time --sort-order desc
egpu vm list --filter '[{"key":"status","op":"eq","val":"running"}]'
```

### Create

```bash
egpu vm create \
  --name <string> \
  --image-id <image-id> \
  --flavor-id <flavor-id> \
  --ssh-key-name <label> \
  --ssh-public-key <openssh-public-key> \
  [--init-script <bash-script>] \
  [--no-wait]
```

By default, the command polls for up to 10 minutes until the VM reaches `running`, writing progress to stderr and returning the full VM object. `--no-wait` returns the initial `{id, name}` creation response.

### Inspect and lifecycle

```bash
egpu vm get <instance-id>
egpu vm start <instance-id>
egpu vm stop <instance-id>
egpu vm reboot <instance-id>
egpu vm delete <instance-id> --force
```

Stopping a VM does not stop billing. Deleting a VM is irreversible and stops VM billing after the instance is released.

### Metrics

```bash
egpu vm metrics <instance-id> [--duration 1h|2h|4h|6h|12h|1d|3d|7d|15d|30d]
```

### VM volume attachment

```bash
egpu vm volume attach <vm-id> --volume-ids <volume-id>[,<volume-id>...]
egpu vm volume attach <vm-id> --volume-ids <volume-id-1> --volume-ids <volume-id-2>
egpu vm volume detach <vm-id> <volume-id>
```

---

## Volume Commands

### List

```bash
egpu volume list [--limit <int>] [--offset <int>] [--sort-field <field>] [--sort-order asc|desc] [--filter '<json-array>']
```

### Volume types

```bash
egpu volume type list --region-id <region-id>
```

### Create

```bash
egpu volume create \
  --display-name <string> \
  --region-id <region-id> \
  --type-id <volume-type-id> \
  --size <gb> \
  [--image-id <image-id>] \
  [--description <string>] \
  [--payment-currency <currency>]
```

`--display-name`, `--region-id`, `--type-id`, and `--size` are required. `--display-name` must be 50 characters or fewer.

### Delete

```bash
egpu volume delete <volume-id> --force
```

Volume deletion is irreversible.

---

## Billing Commands

```bash
egpu billing balance
egpu billing usage [--limit <int>] [--offset <int>] [--sort-field <field>] [--sort-order asc|desc] [--filter '<json-array>']
egpu billing statement [--limit <int>] [--offset <int>] [--sort-field <field>] [--sort-order asc|desc] [--filter '<json-array>']
```

`billing balance` returns available credits keyed by currency, e.g. `{"available":{"USD":42.5}}`.

---

## API Token Commands

```bash
egpu token list [--limit <int>] [--offset <int>] [--sort-field <field>] [--sort-order asc|desc]
egpu token create --name <string> [--description <string>] [--save]
egpu token update <token-id> --name <string> [--description <string>]
egpu token delete <token-id> --force
```

API tokens do not expire. `--save` writes the generated token to `~/.exabits/config.yaml` as `api_token` and makes it the active credential. The full token value is shown at creation time; store it securely.

---

## Local SSH Key Commands

These commands manage local key pairs under `~/.exabits/keys`; they do not query remote account keys.

```bash
egpu key generate --name <name>
egpu key list
egpu key delete <name> [--force]
```

Generated private keys are written as `~/.exabits/keys/egpu_<name>` with mode `0600`; public keys are written as `~/.exabits/keys/egpu_<name>.pub`.

---

## MCP Server

Start the MCP server over stdio:

```bash
egpu mcp
```

The MCP server reads the same config and environment variables as the CLI. It writes startup diagnostics to stderr; stdout is reserved for JSON-RPC.

### MCP tools

| Tool | CLI equivalent | Description |
|---|---|---|
| `list_regions` | `egpu region list` | List datacenter regions |
| `list_gpu_flavors` | `egpu flavor list` | List GPU flavors; optional `region_id` |
| `list_os_images` | `egpu image list` | List OS images; optional `region_id` |
| `list_gpu_vms` | `egpu vm list` | List VMs with `limit`, `offset`, `sort_field`, `sort_order`, `filter` |
| `get_gpu_vm` | `egpu vm get` | Get VM details |
| `create_gpu_vm` | `egpu vm create` | Create VM; supports `init_script` and `wait_for_running` |
| `start_gpu_vm` | `egpu vm start` | Start VM |
| `stop_gpu_vm` | `egpu vm stop` | Stop VM; billing may continue |
| `reboot_gpu_vm` | `egpu vm reboot` | Reboot VM |
| `get_gpu_vm_metrics` | `egpu vm metrics` | Get VM metrics; optional `duration` |
| `attach_volumes_to_gpu_vm` | `egpu vm volume attach` | Attach one or more volumes |
| `detach_volume_from_gpu_vm` | `egpu vm volume detach` | Detach a volume |
| `delete_gpu_vm` | `egpu vm delete --force` | Permanently delete a VM |
| `list_volumes` | `egpu volume list` | List volumes with pagination/sort/filter |
| `list_volume_types` | `egpu volume type list` | List volume types for `region_id` |
| `create_volume` | `egpu volume create` | Create a volume |
| `delete_volume` | `egpu volume delete --force` | Permanently delete a volume |
| `check_billing_balance` | `egpu billing balance` | Get credit balance |
| `get_billing_usage` | `egpu billing usage` | Get usage history |
| `get_billing_statements` | `egpu billing statement` | Get billing statements |
| `list_api_tokens` | `egpu token list` | List API tokens |
| `create_api_token` | `egpu token create` | Create API token; optional `save` |
| `update_api_token` | `egpu token update` | Update token metadata |
| `delete_api_token` | `egpu token delete --force` | Delete API token |
| `generate_ssh_key` | `egpu key generate` | Generate local SSH key pair |
| `list_ssh_keys` | `egpu key list` | List local SSH keys |
| `delete_ssh_key` | `egpu key delete --force` | Delete local SSH key pair |

MCP clients do not pass CLI `--force`; destructive MCP tools are already non-interactive. The agent must ask the user for explicit confirmation before calling any destructive MCP tool.
