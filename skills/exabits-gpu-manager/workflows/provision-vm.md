# Workflow: Provision a GPU VM

Provision a new GPU virtual machine on Exabits GPU Cloud — from hardware selection through to a running instance with SSH access.

## Prerequisites

Load [../references/cli.md](../references/cli.md) before running any commands.
If credentials are missing, load [../references/installation.md](../references/installation.md) first.

If MCP is available, prefer `check_billing_balance`, `list_gpu_flavors`, `list_os_images`, `list_ssh_keys`, `generate_ssh_key`, and `create_gpu_vm`. Use the CLI examples below when MCP is unavailable or the user asks for shell commands.

---

## Hardware Context

| GPU | Best For | Default When |
|---|---|---|
| `H200` | Fastest inference, large-scale LLM training | User asks for "fastest" or "best" without specifying |
| `H100` | High-performance training and fine-tuning | User specifies H100 explicitly |
| `GB200` | Ultra-high-memory workloads, next-gen models | User specifies GB200 explicitly |
| `RTX_PRO_6000` | Cost-effective prototyping, experimentation | User asks for "cheapest" or "cost-effective" |

**Region constraint:** `flavor_id` and `image_id` **must belong to the same region**. A mismatch causes an immediate API error.

---

## Steps

### 1. Check billing balance

Confirm sufficient funds before provisioning. Billing begins immediately on creation.

```bash
egpu billing balance
```

If the balance is low or zero, inform the user and stop. Do not proceed to provisioning.

### 2. Discover available hardware

```bash
egpu flavor list
```

- If the user did not specify a GPU, apply the routing rules in the Hardware Context table above.
- Filter for `"stock_available": true`. If the preferred tier is out of stock, see **Error Handling** below.
- Note the `region_name` and `id` of the chosen product — you will need them in step 3.

### 3. Discover available OS images in the same region

```bash
egpu image list --region-id <region_id>
```

Or use the MCP tool `list_os_images` with `region_id`.

- Default to the latest **Ubuntu LTS** image unless the user specifies otherwise.
- Pre-configured AI/ML images (e.g. `PyTorch`, `CUDA`) are available — prefer them for training/inference workloads.
- Confirm the `region_name` on the image matches the flavor's region.

### 4. Obtain or confirm the SSH public key

Ask the user for their SSH public key if not already provided. The expected format is:

```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... user@host
```

You may also check local keys:

```bash
egpu key list
```

If the user asks you to create a key, run:

```bash
egpu key generate --name <name>
```

Use the returned `public_key` with VM creation. Keep the private key path for the user's SSH command.

### 5. Provision the VM

```bash
egpu vm create \
  --name "<descriptive-name>" \
  --image-id "<image_id>" \
  --flavor-id "<flavor_id>" \
  --ssh-key-name "<key-label>" \
  --ssh-public-key "<full-public-key-string>"
```

The command polls until the VM reaches `running` status (up to 10 minutes) and streams progress to stderr. The final output on stdout is the full VM detail object including `fixed_ip`.

Pass `--no-wait` only if the user explicitly wants to return immediately.

MCP equivalent: call `create_gpu_vm` with `name`, `image_id`, `flavor_id`, `ssh_key.name`, `ssh_key.public_key`, optional `init_script`, and `wait_for_running: true` unless the user wants asynchronous provisioning.

### 6. Report to the user

Once the VM is `running`, report:
- Instance ID
- IP address (`fixed_ip`)
- GPU type and count
- SSH connection command: `ssh username@<fixed_ip>`
- Reminder that billing is now active

---

## Error Handling

### Out of stock (`CapacityError`)

If `egpu vm create` returns a capacity or stock error:

1. **Do not retry the same flavor.**
2. Inform the user: "The requested GPU tier is currently out of stock."
3. Apply the fallback chain: H200 → H100 → RTX_PRO_6000
4. Re-run `egpu flavor list`, filter `stock_available: true`, present alternatives, and wait for user confirmation before proceeding.

### Billing error

If creation fails with a billing error:
1. Run `egpu billing balance` and show the result.
2. Ask the user to top up their account, then retry.

### Region mismatch

If the API returns a region mismatch error:
1. Re-run `egpu flavor list` and confirm `region_name` on both the flavor and the image.
2. Re-run the create command with a corrected `image_id` that shares the same region as the `flavor_id`.
