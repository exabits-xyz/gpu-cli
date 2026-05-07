# egpu — GPU Cloud CLI

A command-line interface for managing resources on the [Exascalelabs GPU Cloud](https://gpu.exascalelabs.ai) platform.

Built with [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper), and designed to be **Agent-Ready** — fully usable by AI coding agents (Claude Code, Cursor, etc.) without human intervention.

---

## Table of Contents

- [Requirements](#requirements)
- [Installation](#installation)
- [Authentication](#authentication)
  - [Option 1 — auth login](#option-1--egpu-auth-login-quickest-start)
  - [Option 2 — JWT tokens](#option-2--jwt-tokens-config-file--env-vars)
  - [Option 3 — API Token](#option-3--api-token-never-expires)
- [Configuration Reference](#configuration-reference)
- [Commands](#commands)
  - [auth login](#egpu-auth-login)
  - [vm list](#egpu-vm-list)
  - [vm create](#egpu-vm-create)
  - [vm delete](#egpu-vm-delete)
  - [token list](#egpu-token-list)
  - [token create](#egpu-token-create)
  - [token update](#egpu-token-update)
  - [token delete](#egpu-token-delete)
- [Agent-Ready Design](#agent-ready-design)
  - [Stream Separation](#stream-separation)
  - [Exit Codes](#exit-codes)
  - [Auto-JSON Mode](#auto-json-mode)
- [Agent Skills](#agent-skills)
  - [How It Works](#how-it-works)
  - [The Skill File](#the-skill-file)
  - [Publish to npm](#publish-to-npm)
  - [Install the Skill](#install-the-skill)
  - [Supported Agents](#supported-agents)
- [Development](#development)
  - [Dev Build](#dev-build)
  - [Debugging](#debugging)
  - [Testing](#testing)
  - [Release & Deploy](#release--deploy)
- [Project Structure](#project-structure)
- [Extending the CLI](#extending-the-cli)

---

## Requirements

- Go 1.21 or later
- An Exabits account with valid credentials (`access_token` + `refresh_token`, or an `api_token`)

---

## Installation

```bash
# Clone and build
git clone https://github.com/exabits-xyz/gpu-cli
cd gpu-cli
go build -o egpu .

# Move to PATH (optional)
mv egpu /usr/local/bin/egpu
```

Or install directly with `go install`:

```bash
go install github.com/exabits-xyz/gpu-cli@latest
```

---

## Authentication

The CLI supports browser authorization, API tokens, and legacy JWT login. Plain `api_token` takes precedence, then encrypted browser-auth tokens, then JWT tokens.

| Method | Headers sent | Expiry |
|---|---|---|
| **API Token** (`api_token` or `api_token_encrypted`) | `Authorization: Bearer <api_token>` | Never |
| **JWT** (`access_token` + `refresh_token`) | `Authorization: Bearer <access_token>` + `refresh-token: <refresh_token>` | 30 min / 2 h |

All headers are injected automatically by the HTTP client on every request.

### Option 1 — `egpu auth` browser login (recommended)

Run `egpu auth` without username/password. The CLI requests a one-time authorization state, opens your browser at `https://gpu.exascalelabs.ai/login?state=...`, waits while you log in and authorize on the web, then encrypts the returned API token locally.

```bash
egpu auth
```

Use `--no-browser` to print the URL without launching the system browser.

### Option 2 — `egpu auth login` with username/password

Run the login command with your Exabits account credentials. The password is MD5-hashed by the CLI before being sent — pass the plain-text value. On success, `access_token` and `refresh_token` are written to `~/.exabits/config.yaml`.

```bash
egpu auth login --username you@example.com --password yourpassword
```

> Tokens expire: `access_token` after **30 minutes**, `refresh_token` after **2 hours**. Re-run `auth login` to refresh them, or use browser auth / an API Token to avoid expiry.

### Option 3 — JWT tokens (config file / env vars)

Obtain tokens via the Exabits platform and write them to `~/.exabits/config.yaml`:

```yaml
access_token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
refresh_token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Optional — defaults to https://gpu-api.exascalelabs.ai
# api_url: "https://gpu-api.exascalelabs.ai"
```

```bash
mkdir -p ~/.exabits && chmod 700 ~/.exabits
```

Or via environment variables (useful in CI and for AI agents):

```bash
export EXABITS_ACCESS_TOKEN="eyJ..."
export EXABITS_REFRESH_TOKEN="eyJ..."
```

Environment variables take precedence over the config file.

### Option 4 — API Token (never expires)

Generate a long-lived API Token with `egpu token create` or from the Exabits platform. Only a single header is required — no refresh cycle needed.

```bash
# Create a token and save it as the active credential immediately
egpu auth login --username you@example.com --password yourpassword
egpu token create --name ci-agent --description "CI pipeline" --save
```

Or set it manually in `~/.exabits/config.yaml`:

```yaml
api_token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

Or via env var:

```bash
export EXABITS_API_TOKEN="eyJ..."
```

---

## Configuration Reference

| Key | Env var | Required | Default | Description |
|---|---|---|---|---|
| `api_token` | `EXABITS_API_TOKEN` | — | — | Long-lived API Token. When set, `access_token` and `refresh_token` are ignored. |
| `api_token_encrypted` | `EXABITS_API_TOKEN_ENCRYPTED` | — | — | Encrypted API Token written by `egpu auth`. |
| `access_token` | `EXABITS_ACCESS_TOKEN` | Yes (JWT mode) | — | Short-lived JWT. Expires after **30 minutes**. |
| `refresh_token` | `EXABITS_REFRESH_TOKEN` | Yes (JWT mode) | — | JWT refresh token. Expires after **2 hours**. |
| `api_url` | `EXABITS_API_URL` | No | `https://gpu-api.exascalelabs.ai` | Override the API host (e.g. for staging). The `/api/v1` base path is appended automatically. |
| `auth_url` | `EXABITS_AUTH_URL` | No | `https://gpu.exascalelabs.ai/login` | Override the browser login URL. |

Auth precedence: `api_token` → `api_token_encrypted` → `access_token` + `refresh_token`. Environment variables take precedence over the config file.

---

## Commands

### Global flags

| Flag | Description |
|---|---|
| `--json` | Force JSON output even in an interactive terminal |
| `--help` | Show help for any command |

---

### `egpu auth`

Start browser-based authentication and save the returned API token encrypted in `~/.exabits/config.yaml`.

```
egpu auth [--no-browser]
```

| Flag | Required | Description |
|---|---|---|
| `--no-browser` | No | Print the authorization URL without opening a browser |

---

### `egpu auth login`

Authenticate with your Exabits account and save `access_token` + `refresh_token` to `~/.exabits/config.yaml`. Any pre-existing keys in the file (e.g. `api_url`) are preserved.

```
egpu auth login --username <string> --password <string>
```

| Flag | Required | Description |
|---|---|---|
| `--username` | Yes | Exabits account username |
| `--password` | Yes | Plain-text password (MD5-hashed by the CLI before sending) |

**Example:**

```bash
egpu auth login --username you@example.com --password mysecret
```

**Output:**

```json
{
  "email": "you@example.com",
  "message": "login successful — tokens saved to ~/.exabits/config.yaml",
  "username": "you@example.com"
}
```

---

### `egpu vm list`

List VM instances with optional pagination, sorting, and filtering.

```
egpu vm list [flags]
```

| Flag | Type | Description |
|---|---|---|
| `--limit` | int | Maximum number of VMs to return |
| `--offset` | int | Number of VMs to skip (pagination) |
| `--sort-field` | string | Field to sort by (e.g. `name`, `status`, `started_time`) |
| `--sort-order` | string | `asc` or `desc` |
| `--filter` | string | JSON filter array, e.g. `'[{"key":"name","op":"contains","val":"hub"}]'` |

**Example:**

```bash
# All VMs
egpu vm list

# First 5, sorted by name
egpu vm list --limit 5 --sort-field name --sort-order asc

# Filter by name substring
egpu vm list --filter '[{"key":"name","op":"contains","val":"training"}]'
```

**Output:**

```json
{
  "total": 1,
  "data": [
    {
      "id": "66bd5a1299f01e419f5ad5bc",
      "name": "VM-dhSqf5qh",
      "status": "running",
      "login": {
        "_id": "66bd5a1299f01e419f5ad5bd",
        "ssh_key": { "id": "66bd5a07...", "name": "sshkey" },
        "password": "password",
        "username": "username"
      },
      "fixed_ip": "198.51.100.42",
      "started_time": 1723685402,
      "flavor": {
        "name": "1 x RTX4090",
        "cpu": 16,
        "ram": 32,
        "disk": 250,
        "gpu": "RTX4090",
        "gpu_count": 1
      },
      "image": { "name": "Ubuntu-22.04" },
      "region": { "name": "DALLAS" }
    }
  ]
}
```

**Pipe-friendly:**

```bash
# IDs of all running VMs
egpu vm list | jq '[.data[] | select(.status == "running") | .id]'

# Page through all VMs (100 at a time)
egpu vm list --limit 100 --offset 0 | jq '.total, (.data | length)'
```

---

### `egpu vm create`

Create a GPU VM instance. `image_id` and `flavor_id` must belong to the same region.

```
egpu vm create --name <string> --image-id <string> --flavor-id <string> \
                  --ssh-key-name <string> --ssh-public-key <string> \
                  [--init-script <string>]
```

| Flag | Required | Description |
|---|---|---|
| `--name` | Yes | Name for the VM |
| `--image-id` | Yes | OS image ID (must match flavor region) |
| `--flavor-id` | Yes | Hardware flavor ID (must match image region) |
| `--ssh-key-name` | Yes | Label to assign to the SSH key |
| `--ssh-public-key` | Yes | Full public key string, e.g. `ssh-ed25519 AAAA...` |
| `--init-script` | No | Bash script executed at first boot (cloud-init) |

All five required flags must be present; any missing flag produces a JSON error on stderr and exits with code `2`.

**Example:**

```bash
egpu vm create \
  --name           training-run-01 \
  --image-id       66b2d63c9e793247704c5a01 \
  --flavor-id      66b9ca8f6523790d00fea3ca \
  --ssh-key-name   my-workstation-key \
  --ssh-public-key "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAA... user@host"
```

**Output:**

```json
{
  "id": "66bd5a1299f01e419f5ad5bc",
  "name": "training-run-01"
}
```

> The API returns only `id` and `name` at creation time. Use `egpu vm list` to inspect the full VM detail.

---

### `egpu vm delete`

Permanently delete a VM. **This operation is irreversible** — the server is immediately released, all data is erased, and billing stops. Pass `--force` to confirm.

```
egpu vm delete <instance-id> --force
```

| Argument / Flag | Required | Description |
|---|---|---|
| `<instance-id>` | Yes | ID of the VM to delete |
| `--force` | Yes | Confirms permanent deletion |

**Example:**

```bash
egpu vm delete 66bd5a1299f01e419f5ad5bc --force
```

**Output:**

```json
{
  "id": "66bd5a1299f01e419f5ad5bc",
  "message": "virtual machine deleted successfully",
  "status": "deleted"
}
```

---

### `egpu token list`

List API tokens with optional pagination and sorting.

```
egpu token list [flags]
```

| Flag | Type | Description |
|---|---|---|
| `--limit` | int | Maximum number of tokens to return |
| `--offset` | int | Number of tokens to skip (pagination) |
| `--sort-field` | string | Field to sort by (e.g. `name`, `created_at`, `last_used`) |
| `--sort-order` | string | `asc` or `desc` |

**Example:**

```bash
egpu token list
egpu token list --sort-field created_at --sort-order desc
```

**Output:**

```json
{
  "total": 1,
  "data": [
    {
      "id": "6707f627b4ff4e6387c91132",
      "name": "ci-agent",
      "description": "CI pipeline",
      "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
      "created_at": 1728575015,
      "last_used": 1728610774
    }
  ]
}
```

> `last_used: 0` means the token has never been used.

---

### `egpu token create`

Create a new API token. API tokens do not expire.

```
egpu token create --name <string> [--description <string>] [--save]
```

| Flag | Required | Description |
|---|---|---|
| `--name` | Yes | Token label, max 50 characters |
| `--description` | No | Optional description |
| `--save` | No | Write the generated token as `api_token` in `~/.exabits/config.yaml`, activating it immediately |

**Example:**

```bash
# Create and activate in one step
egpu token create --name ci-agent --description "CI pipeline" --save
```

**Output:**

```json
{
  "id": "670881c41221aabc72ed946b",
  "name": "ci-agent",
  "description": "CI pipeline",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "created_at": 1728610756
}
```

> Copy the `token` value — it is shown here in full but is not retrievable again from the API.

---

### `egpu token update`

Update the name or description of an existing API token.

```
egpu token update <token-id> --name <string> [--description <string>]
```

| Argument / Flag | Required | Description |
|---|---|---|
| `<token-id>` | Yes | ID of the token to update |
| `--name` | Yes | New token name, max 50 characters |
| `--description` | No | New description (omit to clear) |

**Example:**

```bash
egpu token update 670881c41221aabc72ed946b \
  --name "ci-agent-v2" \
  --description "Updated CI pipeline token"
```

**Output:**

```json
{
  "id": "670881c41221aabc72ed946b",
  "name": "ci-agent-v2",
  "description": "Updated CI pipeline token",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "created_at": 1728610756
}
```

---

### `egpu token delete`

Permanently delete an API token. The token immediately stops authenticating API requests. Pass `--force` to confirm.

```
egpu token delete <token-id> --force
```

| Argument / Flag | Required | Description |
|---|---|---|
| `<token-id>` | Yes | ID of the token to delete |
| `--force` | Yes | Confirms permanent deletion |

**Example:**

```bash
egpu token delete 670881c41221aabc72ed946b --force
```

**Output:**

```json
{
  "id": "670881c41221aabc72ed946b",
  "message": "API token deleted successfully",
  "status": "deleted"
}
```

---

## Agent-Ready Design

This CLI follows strict conventions so that AI agents can drive it programmatically without needing to parse human-readable text.

### Stream Separation

| Stream | Content |
|---|---|
| **stdout** | Pure, valid JSON — always machine-parseable |
| **stderr** | Errors — plain text (TTY) or JSON (non-TTY / `--json`) |

Agents should read stdout for data and inspect stderr + exit code on failure.

### Exit Codes

| Code | Meaning |
|---|---|
| `0` | Success |
| `1` | API error or internal error (network, auth, unexpected response) |
| `2` | Invalid arguments (missing required flags, wrong number of arguments) |

Destructive commands (`vm delete`, `token delete`) require `--force` and exit `2` without it — no interactive prompt is ever shown.

**Agent workflow example:**

```bash
OUTPUT=$(egpu vm list 2>/tmp/err)
EXIT=$?

if [ $EXIT -ne 0 ]; then
  echo "Failed:" >&2
  cat /tmp/err >&2
  exit $EXIT
fi

# .data[] because list commands always return {total, data: [...]}
echo "$OUTPUT" | jq '.data[].id'
```

### Auto-JSON Mode

JSON mode activates automatically in two ways:

1. **`--json` flag** — `egpu --json vm list`
2. **Non-TTY stdout** — when stdout is piped or redirected, JSON is always used

When JSON mode is active, errors on stderr are also emitted as JSON:

```json
{
  "error": "no credentials found — set api_token, or both access_token and refresh_token ..."
}
```

This lets agents parse error messages with the same tooling they use for success responses.

---

## Agent Skills

The `skills/exabits-gpu-manager/` directory contains an [Agent Skills](https://agentskills.io) package — a standardised way to give AI coding agents (Claude Code, Cursor, OpenClaw) **procedural knowledge** about how to use this CLI and its MCP tools.

Rather than discovering the tool surface by trial and error, an agent that has loaded this skill already knows:

- Which MCP tools exist and in what order to call them
- Which GPU tier to default to for a given workload
- How to handle capacity errors without retrying blindly
- That `delete_gpu_vm` requires explicit user confirmation before proceeding

### How It Works

```
User prompt
    │
    ▼
Agent reads SKILL.md          ← procedural knowledge: when to trigger,
    │                            routing rules, error handling, guardrails
    ▼
Agent calls MCP tools          ← list_gpu_flavors → list_os_images → create_gpu_vm
(via egpu mcp stdio server)
    │
    ▼
Exabits GPU Cloud API
```

The skill file is loaded by the agent framework at session start (or on demand via `npx skills add`). It does not add any runtime dependency — it is pure markdown that the agent reads as context.

### The Skill File

**Location:** [`skills/exabits-gpu-manager/SKILL.md`](skills/exabits-gpu-manager/SKILL.md)

**YAML frontmatter** declares the skill identity and a binary prerequisite check:

```yaml
---
name: exabits-gpu-manager
description: "Automates the deployment, scheduling, and management of Exabits GPU instances..."
metadata:
  openclaw:
    requires:
      bins:
        - egpu
---
```

The `openclaw.requires.bins` field tells compatible agents to verify that the `exabits` (or `egpu`) binary is on `$PATH` before activating the skill, so the agent surfaces a helpful install prompt instead of silently failing.

**Markdown body** is divided into six sections that mirror how a senior engineer would onboard a new team member to this tool:

| Section | Purpose |
|---|---|
| When to Use | Intent triggers — which user phrases activate this skill |
| Hardware Context | GPU tier table so the agent can reason about tradeoffs |
| Expert Routing Rules | Named defaults (H200 for speed, RTX_PRO_6000 for cost) |
| Standard Workflow | Ordered 5-step sequence to avoid region-mismatch errors |
| Error Handling & Guardrails | CapacityError fallback chain; DESTRUCTIVE ACTION LOCK for deletions |
| Authentication | Priority-ordered credential sources for non-interactive agent use |

### Publish to npm

The `package.json` at the root of this repo configures the skill for npm distribution. The `files` field ensures only the `skills/` directory is included in the published package — no Go source code or git history is shipped.

```bash
# Log in to npm (one-time)
npm login

# Publish (or bump the version first)
npm version patch   # or minor / major
npm publish --access public
```

The package will be available at `https://npmjs.com/package/@exabits/gpu-manager-skill`.

To publish under a different scope or name, update the `name` field in [`package.json`](package.json) before publishing.

### Install the Skill

**Via the Agent Skills CLI** (recommended):

```bash
npx skills add @exabits/gpu-manager-skill
```

This downloads the package and copies `skills/exabits-gpu-manager/SKILL.md` into the agent's local skill store, making it available in all future sessions automatically.

**Directly from this repository** (no npm publish needed):

```bash
npx skills add github:exabits-xyz/gpu-cli
```

**Manual installation** (copy the file yourself):

```bash
# Claude Code
cp skills/exabits-gpu-manager/SKILL.md ~/.claude/skills/

# Cursor / OpenClaw — place in your project root skills directory
cp skills/exabits-gpu-manager/SKILL.md ./skills/
```

After installation, verify the skill is active:

```bash
npx skills list
# exabits-gpu-manager   Automates the deployment, scheduling, and management...
```

### Supported Agents

| Agent | Install method | Notes |
|---|---|---|
| **Claude Code** | `npx skills add` or copy to `~/.claude/skills/` | Skill is loaded at session start |
| **Cursor** | Copy to project `skills/` directory | Loaded per-project |
| **OpenClaw** | `npx skills add` | `requires.bins` check is enforced |

For Claude Code specifically, you can also reference the skill inline without installing it:

```bash
# From within the project directory, Claude Code will discover skills/ automatically
egpu mcp   # start the MCP server first, then open Claude Code
```

---

## Development

### Dev Build

#### Quick build (current OS/arch)

```bash
go build -o egpu .
```

#### Build with version metadata embedded

```bash
VERSION=$(git describe --tags --always --dirty)
COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build \
  -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}" \
  -o egpu .
```

> To expose these in the CLI, add a `version` command in `cmd/version.go` that prints `main.version`, `main.commit`, and `main.buildDate`.

#### Cross-compile for all platforms

```bash
GOOS=linux   GOARCH=amd64 go build -o dist/egpu-linux-amd64 .
GOOS=linux   GOARCH=arm64 go build -o dist/egpu-linux-arm64 .
GOOS=darwin  GOARCH=arm64 go build -o dist/egpu-darwin-arm64 .
GOOS=darwin  GOARCH=amd64 go build -o dist/egpu-darwin-amd64 .
GOOS=windows GOARCH=amd64 go build -o dist/egpu-windows-amd64.exe .
```

#### Build all platforms in one shot

```bash
mkdir -p dist
for target in "linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64"; do
  OS=${target%/*}; ARCH=${target#*/}
  EXT=$([[ $OS == windows ]] && echo ".exe" || echo "")
  GOOS=$OS GOARCH=$ARCH go build -o "dist/egpu-${OS}-${ARCH}${EXT}" .
  echo "built dist/egpu-${OS}-${ARCH}${EXT}"
done
```

#### Verify the build

```bash
ls -lh egpu
./egpu --help

# Confirm no CGO dependency (pure static binary)
file egpu
ldd egpu 2>/dev/null || echo "statically linked (expected)"
```

---

### Debugging

#### Print raw HTTP request/response

Gate it behind `EXABITS_DEBUG=1` in `internal/api/client.go` inside `do()`, after the request is built:

```go
if os.Getenv("EXABITS_DEBUG") == "1" {
    dump, _ := httputil.DumpRequestOut(req, true)
    fmt.Fprintf(os.Stderr, ">>> REQUEST\n%s\n", dump)
}
// ... after resp ...
if os.Getenv("EXABITS_DEBUG") == "1" {
    dump, _ := httputil.DumpResponse(resp, true)
    fmt.Fprintf(os.Stderr, "<<< RESPONSE\n%s\n", dump)
}
```

```bash
EXABITS_DEBUG=1 egpu vm list
```

#### Use a local mock server

```bash
EXABITS_API_URL=http://localhost:3000 \
EXABITS_ACCESS_TOKEN=test \
EXABITS_REFRESH_TOKEN=test \
  egpu vm list
```

#### Step-debug with Delve

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
dlv debug . -- vm list
```

Useful Delve commands:

```
b cmd/vm.go:77      # breakpoint at line
b api.NewClient     # breakpoint on function
c                   # continue
n                   # next line
s                   # step into
p vms               # print variable
bt                  # stack trace
q                   # quit
```

#### VS Code launch configuration

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "egpu vm list",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}",
      "args": ["vm", "list"],
      "env": {
        "EXABITS_ACCESS_TOKEN": "your-token-here",
        "EXABITS_REFRESH_TOKEN": "your-refresh-token-here",
        "EXABITS_API_URL": "http://localhost:3000"
      }
    },
    {
      "name": "egpu vm create",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}",
      "args": [
        "vm", "create",
        "--name", "debug-vm",
        "--image-id", "66b2d63c9e793247704c5a01",
        "--flavor-id", "66b9ca8f6523790d00fea3ca",
        "--ssh-key-name", "my-key",
        "--ssh-public-key", "ssh-ed25519 AAAA..."
      ],
      "env": {
        "EXABITS_ACCESS_TOKEN": "your-token-here",
        "EXABITS_REFRESH_TOKEN": "your-refresh-token-here",
        "EXABITS_API_URL": "http://localhost:3000"
      }
    }
  ]
}
```

---

### Testing

#### Run all tests

```bash
go test ./...
go test -v -race ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out   # open in browser
go tool cover -func=coverage.out   # per-function summary
```

#### Test the HTTP client with a fake server

The client unwraps the Exabits envelope internally. Test servers must return the full `{"status": bool, "message": string, "data": ...}` shape.

```go
package api_test

import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/exabits-xyz/gpu-cli/internal/api"
    "github.com/exabits-xyz/gpu-cli/internal/types"
    "github.com/spf13/viper"
)

func TestVMList(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("Authorization") == "" {
            t.Error("missing Authorization header")
        }
        if r.Header.Get("refresh-token") == "" {
            t.Error("missing refresh-token header")
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]any{
            "status":  true,
            "message": "ok",
            "total":   1,
            "data": []types.VM{
                {ID: "inst-1", Name: "test-vm", Status: "running"},
            },
        })
    }))
    defer srv.Close()

    viper.Set("api_url", srv.URL)
    viper.Set("access_token", "test-access")
    viper.Set("refresh_token", "test-refresh")

    client, err := api.NewClient()
    if err != nil {
        t.Fatalf("NewClient: %v", err)
    }

    var vms []types.VM
    var total int
    if err := client.GetPaged("/virtual-machines", &vms, &total); err != nil {
        t.Fatalf("GetPaged: %v", err)
    }

    if total != 1 || len(vms) != 1 || vms[0].ID != "inst-1" {
        t.Errorf("unexpected response: total=%d vms=%+v", total, vms)
    }
}

func TestNewClient_MissingToken(t *testing.T) {
    viper.Reset()
    _, err := api.NewClient()
    if err == nil {
        t.Fatal("expected error for missing credentials, got nil")
    }
}
```

#### Test exit codes from the compiled binary

```go
package cmd_test

import (
    "os/exec"
    "testing"
)

var cliBin = "/tmp/egpu-test"

func TestMain(m *testing.M) {
    cmd := exec.Command("go", "build", "-o", cliBin, ".")
    cmd.Dir = "../.."
    if out, err := cmd.CombinedOutput(); err != nil {
        panic(string(out))
    }
    m.Run()
}

func TestVMCreate_MissingFlags_ExitsCode2(t *testing.T) {
    cmd := exec.Command(cliBin, "vm", "create", "--name", "only-name")
    cmd.Env = []string{"EXABITS_ACCESS_TOKEN=tok", "EXABITS_REFRESH_TOKEN=ref"}
    err := cmd.Run()
    exitErr, ok := err.(*exec.ExitError)
    if !ok {
        t.Fatalf("expected ExitError, got %T: %v", err, err)
    }
    if exitErr.ExitCode() != 2 {
        t.Errorf("expected exit 2, got %d", exitErr.ExitCode())
    }
}

func TestVMDelete_NoForce_ExitsCode2(t *testing.T) {
    cmd := exec.Command(cliBin, "vm", "delete", "inst-abc123")
    cmd.Env = []string{"EXABITS_ACCESS_TOKEN=tok", "EXABITS_REFRESH_TOKEN=ref"}
    err := cmd.Run()
    exitErr, ok := err.(*exec.ExitError)
    if !ok {
        t.Fatalf("expected ExitError, got %T: %v", err, err)
    }
    if exitErr.ExitCode() != 2 {
        t.Errorf("expected exit 2, got %d", exitErr.ExitCode())
    }
}
```

---

### Release & Deploy

#### Prerequisites

Install [GoReleaser](https://goreleaser.com/install/):

```bash
brew install goreleaser
# or: go install github.com/goreleaser/goreleaser/v2@latest
```

`GITHUB_TOKEN` is auto-fetched from the active `gh` CLI session via the Makefile — no manual export needed as long as you are logged in with an account that has push access to this repo:

```bash
gh auth status   # verify the correct account is active
```

#### Step 1 — Publish the release

```bash
make release version=v1.2.3
```

This tags the commit, builds binaries for all 5 platforms, and publishes the GitHub release with archives and `checksums.txt`:

| Archive | Platform |
|---|---|
| `egpu_linux_amd64.tar.gz` | Linux x86-64 |
| `egpu_linux_arm64.tar.gz` | Linux ARM64 |
| `egpu_darwin_amd64.tar.gz` | macOS Intel |
| `egpu_darwin_arm64.tar.gz` | macOS Apple Silicon |
| `egpu_windows_amd64.zip` | Windows x86-64 |

Pushing the tag also triggers [`.github/workflows/release.yml`](.github/workflows/release.yml), which runs the same GoReleaser pipeline in CI.

#### Step 2 — Update the Homebrew formula

After the release is live, update the tap repository:

```bash
cd ../homebrew-gpu-cli-tap
./update-formula.sh 1.2.3
git add Formula/egpu.rb
git commit -m "egpu v1.2.3"
git push
```

`update-formula.sh` fetches `checksums.txt` from the release and patches `Formula/egpu.rb` with the new version and SHA256 values for all platforms. Users running `brew upgrade egpu` will receive the update automatically.

#### Dry run (no tag, no publish)

```bash
goreleaser build --snapshot --clean
```

Builds all binaries into `dist/` without creating a tag or GitHub release — useful for verifying the build matrix locally.

#### Install a released binary (end-users)

```bash
curl -fsSL https://raw.githubusercontent.com/exabits-xyz/gpu-cli/main/install.sh | sh
```

Or manually for a specific version:

```bash
VERSION=v1.0.0
OS=linux    # linux | darwin
ARCH=amd64  # amd64 | arm64

curl -L "https://github.com/exabits-xyz/gpu-cli/releases/download/${VERSION}/egpu_${OS}_${ARCH}.tar.gz" \
  | tar xz egpu

chmod +x egpu && sudo mv egpu /usr/local/bin/
```

---

## Project Structure

```
gpu-cli/
├── main.go                   # Entrypoint — calls cmd.Execute()
├── go.mod / go.sum
├── package.json              # npm distribution config for Agent Skills
│
├── cmd/
│   ├── root.go               # Root Cobra command, Viper init, --json flag,
│   │                         # printJSON / printError / exitAPIError / exitInvalidArgs
│   ├── auth.go               # egpu auth login — MD5 hash, saveTokens / saveConfigKeys
│   ├── vm.go                 # egpu vm list / create / get / start / stop / reboot /
│   │                         # metrics / volume attach|detach / delete
│   ├── token.go              # egpu token list / create / update / delete
│   ├── mcp.go                # egpu mcp — MCP stdio server (list_gpu_flavors,
│   │                         # list_os_images, create_gpu_vm, delete_gpu_vm,
│   │                         # check_billing_balance)
│   ├── volume.go             # egpu volume list / create / delete
│   ├── resource.go           # egpu flavor / image / region list — hardware flavors / images / regions
│   ├── billing.go            # egpu billing balance / usage / statement
│   ├── key.go                # egpu key list / create / delete
│   └── config.go             # egpu config show
│
├── internal/
│   ├── api/
│   │   └── client.go         # Authenticated HTTP client (Get / GetPaged / Post / Put /
│   │                         # Delete / DeleteParsed) + standalone Login()
│   └── types/
│       ├── vm.go             # VM, CreateVMRequest/Response, SSHKeyInput, …
│       ├── token.go          # APIToken, CreateTokenRequest, TokenListResult
│       ├── resource.go       # FlavorGroup, FlavorProduct, Image, Region
│       ├── volume.go         # Volume, CreateVolumeRequest, AttachVolumesRequest, …
│       ├── sshkey.go         # SSHKey
│       ├── metrics.go        # VMMetrics
│       └── billing.go        # CreditBalance, UsageRecord, Statement
│
└── skills/
    └── exabits-gpu-manager/
        └── SKILL.md          # Agent Skills — procedural knowledge for AI agents
```

### Key design points

**`internal/api/client.go`**

- `NewClient()` checks for `api_token` first; falls back to `access_token` + `refresh_token`.
- `do()` unwraps the Exabits envelope `{"status": bool, "message": string, "total": int, "data": ...}` internally — callers decode only the inner data type directly.
- `GetPaged` surfaces the `total` field via an out-param `*int` for list endpoints.
- `Login()` is a standalone package-level function — the login endpoint requires no auth headers.
- HTTP methods: `Get`, `GetPaged`, `Post`, `Put`, `Delete`.

**`cmd/auth.go`**

- `saveConfigKeys(map[string]any)` merges any key-value pairs into `~/.exabits/config.yaml` without overwriting existing keys. Used by `auth login` (saves JWT pair) and `token create --save` (saves `api_token`).

**`cmd/root.go`**

- `isJSONMode()` auto-enables JSON when stdout is not a TTY — pipes and agent invocations always get machine-readable output without passing `--json`.
- `exitInvalidArgs` / `exitAPIError` enforce consistent exit codes across all subcommands.

---

## Extending the CLI

### Adding a new resource (e.g., `egpu image list`)

1. Add structs to `internal/types/image.go`
2. Create `cmd/image.go` following the same pattern as `cmd/vm.go`
3. Register the command in its `init()` with `rootCmd.AddCommand(imageCmd)`

### Adding a new subcommand (e.g., `vm start`)

In `cmd/vm.go`, define a new `*cobra.Command` and add it via `vmCmd.AddCommand(vmStartCmd)` inside `init()`.

### Overriding the API base URL (staging/dev)

```bash
EXABITS_API_URL=https://staging.gpu-api.exascalelabs.ai egpu vm list
```
