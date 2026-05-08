# Workflow: Manage Block-Storage Volumes

Create, list, attach, detach, and delete persistent block-storage volumes on Exabits GPU Cloud.

## Prerequisites

Load [../references/cli.md](../references/cli.md) before running any commands.

If MCP is available, prefer `list_volumes`, `list_regions`, `list_volume_types`, `create_volume`, `attach_volumes_to_gpu_vm`, `detach_volume_from_gpu_vm`, and `delete_volume`. Ask for explicit confirmation before calling `delete_volume`.

---

## List Volumes

```bash
egpu volume list
```

Returns all volumes on the account including their IDs, sizes, regions, and attachment status.

---

## Create a Volume

```bash
egpu region list
egpu volume type list --region-id <region-id>
egpu volume create \
  --display-name <string> \
  --region-id <region-id> \
  --type-id <volume-type-id> \
  --size <int-GB>
```

- `--region-id` must match the region of the VM you intend to attach it to. Cross-region attachment is not supported.
- `--type-id` comes from `egpu volume type list --region-id <region-id>`.
- Volumes are billed from creation, not from when they are attached.

Example:

```bash
egpu volume create \
  --display-name dataset-storage \
  --region-id <region-id> \
  --type-id <volume-type-id> \
  --size 500
```

Optional fields: `--image-id` for a bootable volume, `--description`, and `--payment-currency`.

---

## Attach a Volume to a VM

The VM must be in `running` or `stopped` state. The volume and VM must be in the same region.

```bash
egpu vm volume attach <vm-id> --volume-ids <volume-id>
```

Attach multiple volumes at once:

```bash
egpu vm volume attach <vm-id> --volume-ids vol-id-1,vol-id-2
```

After attaching, the volume will appear as a block device inside the VM (e.g. `/dev/vdb`). It may need to be formatted and mounted on first use:

```bash
# Inside the VM — first-time setup only
sudo mkfs.ext4 /dev/vdb
sudo mkdir -p /mnt/data
sudo mount /dev/vdb /mnt/data
```

---

## Detach a Volume from a VM

Unmount the volume inside the VM before detaching to avoid data corruption:

```bash
# Inside the VM first
sudo umount /mnt/data
```

Then detach via CLI:

```bash
egpu vm volume detach <vm-id> <volume-id>
```

---

## Delete a Volume

> **Warning:** Volume deletion is irreversible — all data is permanently erased.

Before deleting, confirm the volume is detached:

```bash
egpu volume list --filter '[{"key":"id","op":"eq","val":"<volume-id>"}]'
```

Delete with `--force`:

```bash
egpu volume delete <volume-id> --force
```

---

## Common Patterns

Attach a pre-existing dataset volume to a newly provisioned VM:

```bash
# After provisioning completes
VM_ID=$(egpu vm list | jq -r '.data[] | select(.name == "training-run-01") | .id')
egpu vm volume attach "$VM_ID" --volume-ids vol-abc123
```

Move a volume to a new VM (detach → attach):

```bash
egpu vm volume detach <old-vm-id> <volume-id>
egpu vm volume attach <new-vm-id> --volume-ids <volume-id>
```
