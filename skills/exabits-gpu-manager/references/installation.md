# Installation & Authentication

Reference for installing the `egpu` CLI and configuring credentials for the Exabits GPU Cloud.

---

## Install the CLI

### Option A — Homebrew (macOS / Linux, recommended)

```bash
brew install exabits-xyz/gpu-cli-tap/egpu
```

### Option B — `go install` (requires Go 1.21+)

```bash
go install github.com/exabits-xyz/gpu-cli@latest
```

The binary is placed in `$(go env GOPATH)/bin`. Make sure that directory is on your `$PATH`:

```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Option C — Pre-built binary (no Go required)

```bash
curl -fsSL https://raw.githubusercontent.com/exabits-xyz/gpu-cli/main/install.sh | sh
```

Or install a specific version:

```bash
VERSION=v1.0.0 curl -fsSL https://raw.githubusercontent.com/exabits-xyz/gpu-cli/main/install.sh | sh
```

### Option D — Build from source

```bash
git clone https://github.com/exabits-xyz/gpu-cli
cd gpu-cli
go build -o egpu .
sudo mv egpu /usr/local/bin/
```

### Verify installation

```bash
egpu --help
```

---

## Authentication

The CLI supports two credential types. `api_token` takes precedence when both are present.

| Method | Headers sent | Expiry |
|---|---|---|
| **API Token** | `Authorization: Bearer <api_token>` | Never — preferred for agents and CI |
| **JWT pair** | `Authorization: Bearer <access_token>` + `refresh-token: <refresh_token>` | 30 min / 2 h |

---

### Option 1 — Interactive login (quickest start)

```bash
egpu auth login --username you@example.com --password yourpassword
```

Saves `access_token` and `refresh_token` to `~/.exabits/config.yaml`. Tokens expire: access after **30 minutes**, refresh after **2 hours**.

### Option 2 — Long-lived API Token (recommended for agents)

```bash
# Log in first, then create and immediately activate a token
egpu auth login --username you@example.com --password yourpassword
egpu token create --name my-agent --description "never-expires credential" --save
```

`--save` writes the token as `api_token` in `~/.exabits/config.yaml` and activates it immediately.

Or set it via environment variable (no config file needed):

```bash
export EXABITS_API_TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Option 3 — Environment variables (CI / Docker)

```bash
export EXABITS_API_TOKEN="eyJ..."                    # preferred
# or
export EXABITS_ACCESS_TOKEN="eyJ..."
export EXABITS_REFRESH_TOKEN="eyJ..."
```

Environment variables take precedence over the config file.

---

## Configuration Reference

Config file location: `~/.exabits/config.yaml`

| Key | Env var | Required | Default | Description |
|---|---|---|---|---|
| `api_token` | `EXABITS_API_TOKEN` | — | — | Long-lived API Token. When set, JWT fields are ignored. |
| `access_token` | `EXABITS_ACCESS_TOKEN` | JWT mode | — | Short-lived JWT. Expires after 30 minutes. |
| `refresh_token` | `EXABITS_REFRESH_TOKEN` | JWT mode | — | Refresh token. Expires after 2 hours. |
| `api_url` | `EXABITS_API_URL` | No | `https://gpu-api.exascalelabs.ai` | Override the API host (e.g. for staging). |

Auth precedence: `api_token` → `access_token` + `refresh_token`. Environment variables take precedence over the config file.

---

## Verify Authentication

```bash
egpu billing balance   # returns your credit balance if credentials are valid
```

Expected output:

```json
{
  "balance": 42.50,
  "currency": "USD"
}
```

If this returns an auth error, re-run `egpu auth login` or check that `EXABITS_API_TOKEN` is set correctly.
