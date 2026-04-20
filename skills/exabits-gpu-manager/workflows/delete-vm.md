# Workflow: Delete a VM Instance

Permanently terminate a GPU VM, release its resources, erase all data, and stop billing.

## Prerequisites

Load [../references/cli.md](../references/cli.md) before running any commands.

---

## DESTRUCTIVE ACTION LOCK

> **CRITICAL:** Before calling `egpu vm delete` or the MCP tool `delete_gpu_vm`, you **MUST** halt execution and ask the user for explicit confirmation.

Use this exact prompt:

> "Are you sure you want to permanently terminate instance `<instance_id>`? This operation is **irreversible** — the server will be immediately released and **all data will be permanently erased**. Billing will stop. Please confirm by typing 'yes, delete it'."

**Only proceed if the user gives unambiguous affirmative confirmation.** A vague "ok", "sure", or "yes" alone is not sufficient — require the specific phrase or equivalent explicit consent. If in doubt, abort and ask again.

---

## Steps

### 1. Identify the instance

If the user has not provided an instance ID, list instances first:

```bash
egpu vm list
```

Show the user the list and ask them to confirm which instance to delete.

### 2. Show instance details before deletion

```bash
egpu vm get <instance-id>
```

Show the user the instance name, GPU type, IP, and current status before proceeding. This gives them a final chance to verify they have the right instance.

### 3. Request explicit confirmation

Use the prompt in the DESTRUCTIVE ACTION LOCK section above. Do not proceed without it.

### 4. Delete the instance

```bash
egpu vm delete <instance-id> --force
```

`--force` is required by the CLI. Do not pass it until the user has confirmed.

### 5. Confirm deletion

```bash
egpu vm list | jq '[.data[] | .id]'
```

Verify the deleted instance ID is no longer present. Report to the user that the instance has been terminated and billing has stopped.

---

## Volumes

Deleting a VM does **not** automatically delete attached volumes. After deletion:

```bash
egpu volume list
```

Ask the user whether they want to keep or delete the detached volumes. If deletion is needed, follow the [manage-volumes](manage-volumes.md) workflow.

---

## If the User Just Wants to Pause

If the user wants to stop spending without destroying data, recommend stopping instead of deleting:

```bash
egpu vm stop <instance-id>
```

Note that a stopped VM still incurs charges. If they want to eliminate all costs, deletion is the only option.
