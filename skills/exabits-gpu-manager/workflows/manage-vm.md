# Workflow: Manage VM Instances

List, inspect, start, stop, reboot, and monitor GPU VM instances on Exabits GPU Cloud.

## Prerequisites

Load [../references/cli.md](../references/cli.md) before running any commands.

If MCP is available, prefer `list_gpu_vms`, `get_gpu_vm`, `start_gpu_vm`, `stop_gpu_vm`, `reboot_gpu_vm`, `get_gpu_vm_metrics`, `attach_volumes_to_gpu_vm`, and `detach_volume_from_gpu_vm`. Use CLI commands when MCP is unavailable or the user asks for shell commands.

---

## List Instances

Show all VM instances on the account:

```bash
egpu vm list
```

Filter by status:

```bash
egpu vm list --filter '[{"key":"status","op":"eq","val":"running"}]'
```

Filter by name substring:

```bash
egpu vm list --filter '[{"key":"name","op":"contains","val":"training"}]'
```

Paginate large accounts:

```bash
egpu vm list --limit 20 --offset 0 --sort-field started_time --sort-order desc
```

---

## Inspect a Single Instance

```bash
egpu vm get <instance-id>
```

Returns the full VM object including `status`, `fixed_ip`, `flavor`, `image`, and `region`.

---

## Start a Stopped Instance

```bash
egpu vm start <instance-id>
```

> A stopped VM still incurs charges — it retains its allocated resources. Starting it does not change billing.

---

## Stop a Running Instance

```bash
egpu vm stop <instance-id>
```

> **Important:** stopping a VM does **not** stop billing. The instance remains allocated. To stop billing entirely, use the [delete-vm](delete-vm.md) workflow.

---

## Reboot an Instance

Hard-reboot (equivalent to a physical power cycle). Use when the instance is unresponsive.

```bash
egpu vm reboot <instance-id>
```

---

## Monitor Performance Metrics

Retrieve CPU, memory, disk I/O, and network time-series metrics:

```bash
egpu vm metrics <instance-id> --duration 1h
```

Valid duration values: `1h` `2h` `4h` `6h` `12h` `1d` `3d` `7d` `15d` `30d`

Omit `--duration` to return all recorded data.

MCP equivalent: call `get_gpu_vm_metrics` with `instance_id` and optional `duration`.

---

## Attach or Detach Volumes

Attach one or more volumes:

```bash
egpu vm volume attach <vm-id> --volume-ids <volume-id-1>,<volume-id-2>
```

Detach one volume:

```bash
egpu vm volume detach <vm-id> <volume-id>
```

MCP equivalents: `attach_volumes_to_gpu_vm` and `detach_volume_from_gpu_vm`.

---

## Common Patterns

Find the IP of all running VMs:

```bash
egpu vm list | jq '[.data[] | select(.status == "running") | {id: .id, ip: .fixed_ip, name: .name}]'
```

Check if a specific instance is running:

```bash
egpu vm get <instance-id> | jq '.status'
```

Get GPU type for all instances:

```bash
egpu vm list | jq '[.data[] | {name: .name, gpu: .flavor.gpu, count: .flavor.gpu_count}]'
```
