# Workflow: Provision a GPU VM

Provision a new GPU virtual machine on Exabits GPU Cloud — from hardware selection through to a running instance with SSH access.

## Prerequisites

Load [../references/cli.md](../references/cli.md) before running any commands.
If credentials are missing, load [../references/installation.md](../references/installation.md) first.

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
egpu resource list
```

- If the user did not specify a GPU, apply the routing rules in the Hardware Context table above.
- Filter for `"stock_available": true`. If the preferred tier is out of stock, see **Error Handling** below.
- Note the `region_name` and `id` of the chosen product — you will need them in step 3.

### 3. Discover available OS images in the same region

```bash
egpu resource list | jq '[.[] | select(.region == "<region_name>") | .products[0]]'
```

Or use the MCP tool `list_os_images` and filter by `region_name`.

- Default to the latest **Ubuntu LTS** image unless the user specifies otherwise.
- Pre-configured AI/ML images (e.g. `PyTorch`, `CUDA`) are available — prefer them for training/inference workloads.
- Confirm the `region_name` on the image matches the flavor's region.

### 4. Obtain or confirm the SSH public key

Ask the user for their SSH public key if not already provided. The expected format is:

```
ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... user@host
```

You may also check `egpu key list` if the user has keys stored on the account.

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
4. Re-run `egpu resource list`, filter `stock_available: true`, present alternatives, and wait for user confirmation before proceeding.

### Billing error

If creation fails with a billing error:
1. Run `egpu billing balance` and show the result.
2. Ask the user to top up their account, then retry.

### Region mismatch

If the API returns a region mismatch error:
1. Re-run `egpu resource list` and confirm `region_name` on both the flavor and the image.
2. Re-run the create command with a corrected `image_id` that shares the same region as the `flavor_id`.
