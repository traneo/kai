<picture>
  <source media="(prefers-color-scheme: dark)" srcset="image/logo.png">
  <img alt="Kai" src="image/logo.png" width="120" align="right">
</picture>

<br>

# Kai

> **A pipeline platform for AI-assisted software development.**

Kai lets you describe multi-step software tasks as YAML pipelines. The platform orchestrates AI coding agents to implement, test, validate, and deliver the result — with validation gates, human approval, and a full audit trail.

---

## How it works

```
You define a pipeline in YAML
        |
        v
+----------------------------------------+
|          Orchestrator (Go)             |
|   parse -> DAG -> clone repo -> dispatch
|   gates -> approval -> push / PR       |
+----------------+-----------------------+
                 | gRPC
                 v
+----------------------------------------+
|           Agent workers (Go)           |
|  +--------+ +--------+ +--------+      |
|  |  kai   | |opencode| |claude- |      |
|  | agent  | | agent  | |  code  |      |
|  |        | |        | | agent  |      |
|  +----+---+ +----+---+ +----+---+      |
|       |         |         |            |
|       | runner plugins (Go)            |
|       v         v         v             |
|    kai-code CLI   opencode    claude        |
|   (C# .NET)  binary     code CLI       |
|       |         |         |            |
|       +---------+---------+            |
|                 v                       |
|         LLM provider                   |
|  (OpenAI, Ollama, Anthropic, etc.)     |
+----------------------------------------+
                 |
                 v
        Git: branch push or PR
```

A pipeline run goes through this flow:

1. You submit a YAML pipeline via the Web UI, `kaictl`, or `POST /api/pipelines`.
2. The orchestrator parses the YAML, builds a DAG of steps, and clones the target repo.
3. Ready steps are dispatched over gRPC to idle agents (parallel where the DAG allows).
4. Each agent spawns its runner plugin, which runs the AI coding tool against an LLM.
5. When a step completes, the orchestrator runs the configured validation gates.
6. If a step requires human approval, it blocks until someone approves via the UI or `kaictl`.
7. On success the orchestrator commits, pushes, or opens a PR — depending on the pipeline's `output.type`.
8. Every event is written to the audit log and streamed live over SSE.

---

## Pipeline YAML

```yaml
version: 1
project: my-service

repo:
  url: http://git.example.com/org/repo.git
  base_branch: main
  provider: forgejo
  token_ref: git-token

output:
  type: pr

steps:
  - id: impl
    prompt: |
      Add a POST /api/register endpoint to the Go HTTP server.
      Hash passwords with bcrypt and return 201 on success.
    policy:
      allowed_tools: [read_file, write_file, run, search, glob]
      allowed_commands: ["go *"]
      max_retries: 2
      timeout_seconds: 300
    validation:
      - exit_zero
      - lint
      - tests
    approval: optional
```

Steps form a DAG via `depends_on`. See [examples/](examples/) for full pipelines.

---

## Requirements

- **Go 1.25+** (orchestrator, config service, agent worker, plugins)
- **.NET 10 SDK** (kai-code CLI agent — the C# .NET agent runtime)
- **Node.js 20+** (web UI build)
- **Docker** (optional, for the containerized simulation)
- **jq** (used by `start-dev.sh`)

An OpenAI-compatible LLM endpoint is required to actually run pipelines.

---

## Quick start

The fastest way to try the platform is the simulation bundle. It builds and starts every component together (config service, orchestrator, three agent workers, the web UI).

```bash
cd simulation
make build          # compiles Go, .NET, plugins, and UI
make env-files      # generates per-service .env files
make start-dev      # runs everything on the host
```

Or with Docker:

```bash
cd simulation
make docker-build
docker compose up
```

Once everything is up:

| Service            | URL                          |
| ------------------ | ---------------------------- |
| Web UI             | http://localhost:5173        |
| Orchestrator HTTP  | http://localhost:8080        |
| Orchestrator gRPC  | localhost:50051              |
| Config service     | http://localhost:8081        |
| Agent kai          | localhost:50052              |
| Agent opencode     | localhost:50053              |
| Agent claude-code  | localhost:50054              |

Submit a pipeline:

```bash
# Using kaictl
kaictl login http://localhost:8080 ""
kaictl pipeline create --file examples/pipeline-forgejo.yaml
```

---

## What Kai gives you

- **DAG pipelines** — Steps run in parallel when the dependency graph allows it, with cycle detection.
- **Pluggable runners** — The C# .NET kai-code CLI, opencode, or Claude Code can each be used as the agent runtime for a pool. Add your own by writing a runner plugin.
- **10 validation gates** — `exit_zero`, `lint`, `typecheck`, `tests`, `diff_review`, `approval`, `security_scan`, `license_check`, `breaking_changes`, `code_quality`. Add custom gates as plugins.
- **Human-in-the-loop** — Per-step approval gates that block a step until a human approves via the UI or `kaictl`.
- **Policy enforcement** — Per-step `allowed_tools`, `allowed_commands`, `allowed_dirs`. Misuse is rejected before any code runs.
- **Pluggable git providers** — GitHub, GitLab, Bitbucket, and a Forgejo plugin ship in the box.
- **Pluggable secrets** — Environment variables, HashiCorp Vault, in-memory, and plugin backends.
- **Audit trail** — Event-sourced log of every pipeline and step transition, queryable from the API, `kaictl`, or the UI.
- **Live telemetry** — Server-Sent Events stream stdout, stderr, and step transitions in real time.
- **Versioned config** — Draft, publish, activate, roll back. Most settings hot-reload; auth/TLS changes flag for restart.
- **Self-hosted** — Everything runs on your infrastructure. No cloud dependency.

---

## Repository layout

```
kai-project/
|-- kai-code/                      # C# .NET agent runtime (the kai-code CLI)
|-- kai-platform/             # Go orchestrator + agent worker
|   |-- orchestrator/         #   workflow engine, API, agent pool, gates
|   |-- agent/                #   worker that hosts runner plugins
|-- kai-platform-ui/          # React 19 + TypeScript + Vite 6 web UI
|-- kai-cli-control/          # kaictl - Go CLI for the platform
|-- kai-config-service/       # Versioned config service
|-- kai-plugins/              # Runner, gate, git provider, secrets, archive plugins
|-- simulation/               # Build scripts, dev runner, Docker compose
|-- examples/                 # Sample pipeline YAML files
`-- docs/                     # Architecture, feature catalog, roadmap
```

---

## Plugin system

Plugins are standalone binaries discovered at startup from `$KAI_PLUGIN_DIR` (defaults to `./.kai/plugins` per service). Each plugin ships with a `plugin.json` manifest:

```json
{
  "name": "my-gate",
  "version": "1.0.0",
  "api_version": "v1",
  "type": "gate",
  "binary": "my-gate"
}
```

| Plugin type    | Role                               | Examples in the repo             |
| -------------- | ---------------------------------- | -------------------------------- |
| `runner`       | AI agent runtime                   | kai-code, opencode, claude-code       |
| `gate`         | Validation gate                    | conventional-commits             |
| `gitprovider`  | Git platform integration           | forgejo                          |
| `secrets`      | Secret store backend               | env                              |
| `archive`      | Workspace state storage            | local-fs                         |

See [kai-plugins/](kai-plugins/) and [docs/architecture.md](docs/architecture.md) for the contract and how to add your own.

---

## Development

```bash
# Build everything from source
cd simulation
make build

# Run all services in the foreground
make start-dev

# Clean
make clean
```

Individual components build with their native toolchains:

```bash
# Orchestrator / agent / plugins
cd kai-platform && make build

# kai-code CLI
cd kai-code && dotnet build

# Web UI
cd kai-platform-ui && npm install && npm run dev
```

---

## Documentation

- [docs/architecture.md](docs/architecture.md) - System architecture
- [kai-cli-control/README.md](kai-cli-control/README.md) - kaictl reference
- [simulation/README.md](simulation/README.md) - Simulation / dev environment

---

## License

MIT
