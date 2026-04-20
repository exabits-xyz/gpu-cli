# CLI Reference

Full command reference for the `egpu` CLI. All commands emit pure JSON on stdout and write errors to stderr. Destructive commands require `--force`.

---

## Global Flags

| Flag | Description |
|---|---|
| `--json` | Force JSON output even in an interactive terminal |
| `--help` | Show help for any command |

---

## egpu auth

### `egpu auth login`

Authenticate and save tokens to `~/.exabits/config.yaml`.

```bash
egpu auth login --username <email> --password <plaintext-password>
```

| Flag | Required | Description |
|---|---|---|
| `--username` | Yes | Exabits account email |
| `--password` | Yes | Plain-text password (MD5-hashed by CLI before sending) |

---

## egpu vm

### `egpu vm list`

List VM instances with optional pagination, sorting, and filtering.

```bash
egpu vm list [flags]
```

| Flag | Type | Description |
|---|---|---|
| `--limit` | int | Max number of VMs to return |
| `--offset` | int | Number of VMs to skip (pagination) |
| `--sort-field` | string | Field to sort by: `name`, `status`, `started_time` |
| `--sort-order` | string | `asc` or `desc` |
| `--filter` | string | JSON filter array, e.g. `'[{"key":"name","op":"contains","val":"hub"}]'` |

Filter operators: `contains`, `eq`, `ne`, `gt`, `lt`

```bash
egpu vm list
egpu vm list --limit 10 --sort-field name --sort-order asc
egpu vm list --filter '[{"key":"status","op":"eq","val":"running"}]'
egpu vm list | jq '[.data[] | select(.status == "running") | .id]'
```

---

### `egpu vm create`

Provision a new GPU VM. `image_id` and `flavor_id` **must belong to the same region**.

```bash
egpu vm create \
  --name <string> \
  --image-id <string> \
  --flavor-id <string> \
  --ssh-key-name <string> \
  --ssh-public-key <string> \
  [--init-script <string>] \
  [--no-wait]
```

| Flag | Required | Description |
|---|---|---|
| `--name` | Yes | VM name, unique within your account |
| `--image-id` | Yes | OS image ID (obtain from `egpu resource list`) |
| `--flavor-id` | Yes | Hardware flavor ID (obtain from `egpu resource list`) |
| `--ssh-key-name` | Yes | Label to assign to the SSH key |
| `--ssh-public-key` | Yes | Full public key string, e.g. `ssh-ed25519 AAAA... user@host` |
| `--init-script` | No | Bash script executed at first boot (cloud-init) |
| `--no-wait` | No | Return immediately without polling for `running` status |

By default the command polls until the VM reaches `running` (up to 10 minutes) and streams progress to stderr.

```bash
egpu vm create \
  --name training-run-01 \
  --image-id 66b2d63c9e793247704c5a01 \
  --flavor-id 66b9ca8f6523790d00fea3ca \
  --ssh-key-name my-laptop \
  --ssh-public-key "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... user@host"
```

Returns `{id, name}`. Billing begins immediately.

---

### `egpu vm get`

Retrieve full details of a single VM instance.

```bash
egpu vm get <instance-id>
```

---

### `egpu vm start`

Start a stopped VM instance.

```bash
egpu vm start <instance-id>
```

---

### `egpu vm stop`

Stop a running VM. The VM remains intact and **continues to incur charges** while stopped.

```bash
egpu vm stop <instance-id>
```

---

### `egpu vm reboot`

Hard-reboot a VM (equivalent to a physical power cycle).

```bash
egpu vm reboot <instance-id>
```

---

### `egpu vm metrics`

Retrieve CPU, memory, disk, and network metrics.

```bash
egpu vm metrics <instance-id> [--duration <window>]
```

| Flag | Description |
|---|---|
| `--duration` | Time window: `1h` `2h` `4h` `6h` `12h` `1d` `3d` `7d` `15d` `30d` (default: all) |

```bash
egpu vm metrics abc-123 --duration 1h
```

---

### `egpu vm volume attach`

Attach one or more volumes to a running VM.

```bash
egpu vm volume attach <vm-id> --volume-ids <id1>[,<id2>...]
```

`--volume-ids` accepts a comma-separated list or can be repeated.

---

### `egpu vm volume detach`

Detach a volume from a VM.

```bash
egpu vm volume detach <vm-id> <volume-id>
```

---

### `egpu vm delete`

**Irreversible.** Permanently deletes the VM, releases the server, erases all data, and stops billing. Requires `--force`.

```bash
egpu vm delete <instance-id> --force
```

| Flag | Required | Description |
|---|---|---|
| `--force` | Yes | Confirms permanent deletion — all data will be erased |

---

## egpu resource

### `egpu resource list`

List all available GPU hardware flavors grouped by region, including stock availability and hourly price.

```bash
egpu resource list
egpu resource list | jq '[.[] | .products[] | select(.stock_available == true)]'
```

Each product includes: `id`, `name`, `gpu`, `gpu_count`, `cpu`, `ram`, `disk`, `price`, `stock_available`, `region_name`.

---

## egpu volume

### `egpu volume list`

List block-storage volumes on your account.

```bash
egpu volume list
```

### `egpu volume create`

Create a new block-storage volume.

```bash
egpu volume create --name <string> --size <int> --region <string>
```

### `egpu volume delete`

Permanently delete a volume. Requires `--force`.

```bash
egpu volume delete <volume-id> --force
```

---

## egpu billing

### `egpu billing balance`

Retrieve the current account credit balance.

```bash
egpu billing balance
```

### `egpu billing usage`

Retrieve resource usage records for the account.

```bash
egpu billing usage
```

### `egpu billing statement`

Retrieve billing statements / invoices.

```bash
egpu billing statement
```

---

## egpu token

### `egpu token list`

List API tokens.

```bash
egpu token list [--limit <int>] [--offset <int>] [--sort-field <field>] [--sort-order asc|desc]
```

### `egpu token create`

Create a new never-expiring API token.

```bash
egpu token create --name <string> [--description <string>] [--save]
```

`--save` writes the token to `~/.exabits/config.yaml` and activates it immediately.

> The full token value is only shown once at creation — copy it immediately.

### `egpu token update`

Update the name or description of a token.

```bash
egpu token update <token-id> --name <string> [--description <string>]
```

### `egpu token delete`

Permanently delete a token. Requires `--force`.

```bash
egpu token delete <token-id> --force
```

---

## egpu key

### `egpu key list`

List SSH keys stored on your account.

```bash
egpu key list
```

---

## egpu config

### `egpu config show`

Display the current CLI configuration (active credential source, API URL).

```bash
egpu config show
```

---

## egpu mcp

Start the MCP (Model Context Protocol) server over stdio. Exposes five tools to AI assistants:

| Tool | Description |
|---|---|
| `list_gpu_flavors` | List all GPU hardware flavors with availability |
| `list_os_images` | List all available OS images |
| `create_gpu_vm` | Provision a new GPU VM |
| `delete_gpu_vm` | Permanently delete a VM |
| `check_billing_balance` | Retrieve the current account balance |

```bash
egpu mcp
```

Add to your AI assistant config — see [`workflows/setup-mcp.md`](../workflows/setup-mcp.md) for full instructions.

---

## Exit Codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | API error or internal error (network, auth, unexpected response) |
| `2` | Invalid arguments (missing required flags, wrong argument count) |

## Output Format

- **stdout** — pure JSON, always machine-parseable
- **stderr** — errors as plain text (TTY) or JSON (non-TTY / `--json`)

JSON mode activates automatically when stdout is piped or redirected, or when `--json` is passed.
