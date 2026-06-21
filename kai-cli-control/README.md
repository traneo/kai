# kaictl — CLI for kai-platform

`kaictl` is a standalone CLI that talks to the kai-platform REST API.
It is deliberately decoupled — no gRPC, no protobuf, no platform imports.
It communicates the same way a curl command or CI/CD script would.

## Quick start

```bash
# Build
make build

# Configure
kaictl login http://localhost:8080 <token>

# Use
kaictl status
kaictl pipeline list
kaictl pipeline create pipeline.yaml
```

## Commands

| Command | Description |
|---------|-------------|
| `kaictl login <host> <token>` | Save connection info to `~/.kaictl/config.json` |
| `kaictl status` | Platform health and agent count |
| `kaictl stats` | Aggregated usage statistics |
| `kaictl events` | Stream real-time platform events (SSE) |
| `kaictl audit` | Query audit log |
| `kaictl agent list` | List connected agents |
| `kaictl pipeline list` | List pipeline runs |
| `kaictl pipeline show <id>` | Pipeline detail with step statuses |
| `kaictl pipeline create <file.yaml>` | Create a pipeline from a YAML definition |
| `kaictl pipeline cancel <id>` | Cancel a running pipeline |
| `kaictl pipeline approve <id> <step>` | Approve or reject a blocked step |
| `kaictl pipeline logs <id> <step>` | Show step conversation logs |

All commands accept `--json` for machine-readable output.

## Global flags

| Flag | Description |
|------|-------------|
| `--host` | Override platform URL (default: from config) |
| `--token` | Override auth token (default: from config) |
| `--json` | Output raw JSON instead of formatted text |

## Configuration

Config is stored as JSON at `~/.kaictl/config.json`:

```json
{
  "host": "http://localhost:8080",
  "token": "your-auth-token"
}
```

Use `kaictl login` to create this file, or write it manually.
The `--host` and `--token` flags override the file at runtime.
