# Workflow: Manage API Tokens

Create, list, update, and delete long-lived API tokens for the Exabits GPU Cloud account.

## Prerequisites

Load [../references/cli.md](../references/cli.md) before running any commands.
You must be authenticated with a valid session (JWT or existing API token) to manage tokens.

If MCP is available, prefer `list_api_tokens`, `create_api_token`, `update_api_token`, and `delete_api_token`. Ask for explicit confirmation before calling `delete_api_token`.

---

## When to Use API Tokens

API tokens never expire, making them the preferred credential for:

- CI/CD pipelines
- Automated agents (Claude Code, Cursor, scripts)
- Long-running background jobs
- Any non-interactive environment where re-login is impractical

---

## List Tokens

```bash
egpu token list
egpu token list --sort-field created_at --sort-order desc
```

Returns all tokens with their IDs, names, descriptions, creation time, and last-used time. `last_used: 0` means the token has never been used.

---

## Create a Token

```bash
egpu token create --name <string> [--description <string>] [--save]
```

| Flag | Description |
|---|---|
| `--name` | Token label, max 50 characters |
| `--description` | Optional description |
| `--save` | Write the token to `~/.exabits/config.yaml` as `api_token` and activate it immediately |

```bash
egpu token create --name ci-agent --description "GitHub Actions pipeline" --save
```

> **Important:** The full token value is only shown once at creation. Copy it immediately — it cannot be retrieved again.

---

## Update a Token

Rename or update the description. The token value itself cannot be changed.

```bash
egpu token update <token-id> --name <new-name> [--description <new-description>]
```

---

## Delete a Token

Permanently revokes the token. Any system using it will immediately lose API access.

```bash
egpu token delete <token-id> --force
```

`--force` is required. Confirm the correct token ID before proceeding — show the user the token name and description from `egpu token list` first, then ask for explicit confirmation.

---

## Set Up a Token for Agent Use

The recommended setup for AI agents and automation:

```bash
# 1. Create and activate
egpu token create --name my-agent --description "Claude Code / Cursor automation" --save

# 2. Verify it works
egpu billing balance

# 3. For CI environments — export as env var instead of using the config file
export EXABITS_API_TOKEN="<token-value>"
```

Once `EXABITS_API_TOKEN` is set, the CLI uses it automatically on every invocation with no refresh cycle needed.
