# egpu — GPU Cloud CLI

A command-line interface for managing resources on the [Exabits GPU Cloud](https://gpu.exabits.ai/cloud) platform.

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
git clone https://github.com/exabits/gpu-cli
cd gpu-cli
go build -o egpu .

# Move to PATH (optional)
mv egpu /usr/local/bin/egpu
```

Or install directly with `go install`:

```bash
go install github.com/exabits/gpu-cli@latest
```

---

## Authentication

The CLI supports two authentication methods. `api_token` takes precedence when both are present.

| Method | Headers sent | Expiry |
|---|---|---|
| **JWT** (`access_token` + `refresh_token`) | `Authorization: Bearer <access_token>` + `refresh-token: <refresh_token>` | 30 min / 2 h |
| **API Token** (`api_token`) | `Authorization: Bearer <api_token>` | Never |

All headers are injected automatically by the HTTP client on every request.

### Option 1 — `egpu auth login` (quickest start)

Run the login command with your Exabits account credentials. The password is MD5-hashed by the CLI before being sent — pass the plain-text value. On success, `access_token` and `refresh_token` are written to `~/.exabits/config.yaml`.

```bash
egpu auth login --username you@example.com --password yourpassword
```

> Tokens expire: `access_token` after **30 minutes**, `refresh_token` after **2 hours**. Re-run `auth login` to refresh them, or use an API Token (see Option 3) to avoid expiry entirely.

### Option 2 — JWT tokens (config file / env vars)

Obtain tokens via the Exabits platform and write them to `~/.exabits/config.yaml`:

```yaml
access_token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
refresh_token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."

# Optional — defaults to https://gpu-api.exabits.ai
# api_url: "https://gpu-api.exabits.ai"
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

### Option 3 — API Token (never expires)

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
| `access_token` | `EXABITS_ACCESS_TOKEN` | Yes (JWT mode) | — | Short-lived JWT. Expires after **30 minutes**. |
| `refresh_token` | `EXABITS_REFRESH_TOKEN` | Yes (JWT mode) | — | JWT refresh token. Expires after **2 hours**. |
| `api_url` | `EXABITS_API_URL` | No | `https://gpu-api.exabits.ai` | Override the API host (e.g. for staging). The `/api/v1` base path is appended automatically. |

Auth precedence: `api_token` → `access_token` + `refresh_token`. Environment variables take precedence over the config file.

---

## Commands

### Global flags

| Flag | Description |
|---|---|
| `--json` | Force JSON output even in an interactive terminal |
| `--help` | Show help for any command |

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

    "github.com/exabits/gpu-cli/internal/api"
    "github.com/exabits/gpu-cli/internal/types"
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

#### Manual release

```bash
VERSION=v0.1.0
COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE}"

mkdir -p dist
GOOS=linux   GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/egpu-linux-amd64 .
GOOS=linux   GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/egpu-linux-arm64 .
GOOS=darwin  GOARCH=arm64 go build -ldflags "$LDFLAGS" -o dist/egpu-darwin-arm64 .
GOOS=darwin  GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/egpu-darwin-amd64 .
GOOS=windows GOARCH=amd64 go build -ldflags "$LDFLAGS" -o dist/egpu-windows-amd64.exe .

cd dist && sha256sum egpu-* > checksums.txt
```

> `-s -w` strips the symbol table and DWARF debug info, shrinking the binary by ~30%.

#### Automated releases with GoReleaser

```bash
go install github.com/goreleaser/goreleaser/v2@latest
# or: brew install goreleaser
```

`.goreleaser.yaml`:

```yaml
version: 2

before:
  hooks:
    - go mod tidy
    - go test ./...

builds:
  - env:
      - CGO_ENABLED=0
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ignore:
      - goos: windows
        goarch: arm64
    ldflags:
      - -s -w
      - -X main.version={{.Version}}
      - -X main.commit={{.Commit}}
      - -X main.buildDate={{.Date}}

archives:
  - formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]
    name_template: "exabits_{{ .Version }}_{{ .Os }}_{{ .Arch }}"

checksum:
  name_template: "checksums.txt"

changelog:
  sort: asc
  filters:
    exclude: ["^docs:", "^test:", "^chore:"]
```

```bash
# Dry run
goreleaser build --snapshot --clean

# Full release (requires GITHUB_TOKEN)
git tag v0.1.0 && git push origin v0.1.0
GITHUB_TOKEN=<token> goreleaser release --clean
```

#### GitHub Actions

`.github/workflows/release.yml`:

```yaml
name: Release
on:
  push:
    tags: ["v*"]
permissions:
  contents: write
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true
      - run: go test -race ./...
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

`.github/workflows/ci.yml`:

```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
jobs:
  build-and-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
          cache: true
      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest
      - run: go build ./...
      - run: go test -v -race -coverprofile=coverage.out ./...
      - run: go tool cover -func=coverage.out
```

#### Install a released binary (end-users)

```bash
VERSION=v0.1.0
ARCH=linux-amd64   # linux-arm64 | darwin-arm64 | darwin-amd64

curl -L "https://github.com/exabits/gpu-cli/releases/download/${VERSION}/exabits_${VERSION}_${ARCH}.tar.gz" \
  | tar xz egpu

chmod +x egpu && sudo mv egpu /usr/local/bin/
```

---

## Project Structure

```
gpu-cli/
├── main.go                   # Entrypoint — calls cmd.Execute()
├── go.mod / go.sum
│
├── cmd/
│   ├── root.go               # Root Cobra command, Viper init, --json flag,
│   │                         # printJSON / printError / exitAPIError / exitInvalidArgs
│   ├── auth.go               # egpu auth login — MD5 hash, saveTokens / saveConfigKeys
│   ├── vm.go                 # egpu vm list / create / delete
│   └── token.go              # egpu token list / create / update / delete
│
└── internal/
    ├── api/
    │   └── client.go         # Authenticated HTTP client (Get / GetPaged / Post / Put / Delete)
    │                         # + standalone Login() for the unauthenticated login endpoint
    └── types/
        ├── vm.go             # VM, VMLogin, VMFlavor, VMImage, VMRegion, VMListResult,
        │                     # SSHKeyInput, CreateVMRequest, CreateVMResponse,
        │                     # LoginRequest, LoginData
        └── token.go          # APIToken, CreateTokenRequest, TokenListResult
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
EXABITS_API_URL=https://staging.gpu-api.exabits.ai egpu vm list
```
