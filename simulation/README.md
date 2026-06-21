# simulation/

Self-contained development environment for the kai Platform. After `make build`,
every directory here represents a Docker container with its binary, plugins, and
configuration — ready to run locally or package into an image.

## Quick start

```bash
# Local development (all services on host)
make build
make start-dev

# Docker (build images and run containers)
make docker-build
docker compose up
```

## Directory layout

```
simulation/
├── Makefile              # build, clean, start-dev, docker-build, env-files
├── start-dev.sh          # local dev: starts all services on host
├── README.md
├── docker/
│   ├── Dockerfile.*      # one per service
│   └── docker-compose.yml
│
├── orchestrator/         # container: orchestrator + plugins
│   ├── orchestrator      # binary
│   ├── .env              # env vars for Docker (clean KEY=VALUE)
│   ├── .env.docs         # env vars with comments (reference)
│   └── .kai-code/plugins/ # gate, gitprovider, archive, secrets
│
├── config-service/       # container: config-service
│   ├── config-service
│   ├── .env
│   └── .env.docs
│
├── agent-kai-code/       # container: agent + kai-code CLI
│   ├── agent             # agent binary
│   ├── kai-code          # wrapper -> kai-code-cli/KaiCode.Cli
│   ├── kai-code-cli/     # .NET publish output
│   ├── .env / .env.docs
│   └── .kai-code/plugins/kai-code/ # kai-code runner plugin
│
├── agent-opencode/       # container: agent for opencode
│   ├── agent
│   ├── .env / .env.docs
│   └── .kai-code/plugins/opencode/
│
├── agent-claude/         # container: agent for claude-code
│   ├── agent
│   ├── .env / .env.docs
│   └── .kai-code/plugins/claude-code/
│
├── ui/                   # platform UI static files
│   └── dist/
│
├── ui-observability/     # observability UI static files
│   └── dist/
│
├── ui-plan-builder/      # plan builder UI static files
│   └── dist/
│
├── observability/        # observability binary
│
├── plan-builder/         # plan builder binary
│
├── kaictl                # CLI tool (not containerized)
└── tmp/                  # runtime data (created by start-dev.sh)
```

## Makefile targets

| Target | Description |
|---|---|
| `make build` | Compiles all Go binaries, C# agent, plugins, and all 3 UIs |
| `make clean` | Removes all generated artifacts (binaries, plugins, tmp). Preserves `docker/` |
| `make start-dev` | Build + start all services locally on host (Ctrl+C to stop) |
| `make env-files` | Generate `.env` (Docker-compatible) and `.env.docs` (reference) per service |
| `make docker-build` | Build + env-files + `docker compose build` |
| `make check-prereqs` | Verify Go, Node, npm, dotnet, protoc, jq are installed |
| `make proto` | Generate protobuf code from `kai-platform/proto/` |
| `make npm-install` | Install UI dependencies for all 3 UI projects |
| `make build-ui` | Build all 3 UI applications |
| `make build-plugins` | Build runner, gate, gitprovider, secrets, and archive plugins |

## Environment variables

Each container has a `.env.docs` file documenting every supported env var with
defaults and descriptions. The `.env` file is the clean version (no comments)
compatible with Docker's `--env-file`.

```bash
docker run --env-file ./orchestrator/.env kai-orchestrator
```

### Key variables (defaults)

| Service | Env var | Default | Description |
|---|---|---|---|
| orchestrator | `PORT` | 50051 | gRPC port |
| orchestrator | `HTTP_PORT` | 8080 | HTTP API port |
| orchestrator | `CONFIG_SERVICE_URL` | http://localhost:8081 | Config service address |
| config-service | `CONFIG_PORT` | 8081 | HTTP port |
| config-service | `CONFIG_DATA_DIR` | ./tmp/config-service/data | Config data storage |
| config-service | `ORCHESTRATOR_URL` | http://localhost:8080 | Push target for config reload |
| agent | `ORCHESTRATOR_ADDR` | localhost:50051 | Orchestrator gRPC address |
| agent | `AGENT_ID` | local-coder-N | Agent identity |
| agent | `AGENT_LISTEN` | :5005N | gRPC bind address |
| agent | `MISSION_TIMEOUT` | 5m | Per-mission timeout |
| all | `KAI_PLUGIN_DIR` | `./.kai-code/plugins` | Plugin discovery directory |
| all | `OBSERVABILITY_URL` | http://localhost:8082 | Centralized logging endpoint |

## Plugin system

Each service discovers plugins at runtime via `KAI_PLUGIN_DIR`. The directory
contains subdirectories with `plugin.json` manifests:

```
.kai-code/plugins/
├── kai-code/              -> agent-kai-code runner plugin (type: runner)
├── opencode/              -> agent-opencode runner plugin (type: runner)
├── claude-code/           -> agent-claude runner plugin (type: runner)
├── conventional-commits/  -> orchestrator gate plugin (type: gate)
├── forgejo/               -> orchestrator gitprovider plugin (type: gitprovider)
├── local-fs/              -> orchestrator archive plugin (type: archive)
└── env/                   -> orchestrator secrets plugin (type: secrets)
```

Each container only receives the plugins relevant to its role.

## Local development (start-dev.sh)

Runs all services directly on the host (no Docker):

1. Observability on `:8082`
2. Config-service on `:8081`
3. Orchestrator HTTP `:8080`, gRPC `:50051`
4. Plan Builder on `:8083`
5. Agents on `:50052`, `:50053`, `:50054`
6. Platform UI (via vite preview) on `:5173`
7. Observability UI (via vite preview) on `:5174`
8. Plan Builder UI (via vite preview) on `:5175`

After startup, pushes platform config with 3 pools:
- `local-coder-1` -> kai-code runner
- `local-coder-2` -> opencode runner
- `local-coder-3` -> claude-code runner

Requires `jq` for API calls.

## Adding a new plugin

1. Create a new directory in `kai-plugins/<type>/<name>/` with `main.go` and `plugin.json`
2. Add a build target in `kai-plugins/Makefile`
3. Add a dist target in `simulation/Makefile` to copy it to the right container
4. Run `make build` — the plugin is now included
