# Kai - System Architecture

## Overview

Kai is a pipeline platform for AI-assisted software development. Users define multi-step software tasks as YAML pipelines; the orchestrator dispatches those steps to AI coding agents, validates the results, optionally waits for human approval, and finally delivers the work as a feature branch or pull request.

The platform runs as a set of cooperating services:

- **Orchestrator** (Go) - workflow engine, API, agent pool, validation gates, git operations, audit, secrets
- **Agent workers** (Go) - hosts runner plugins that execute AI coding tools against an LLM
- **Runner plugins** (Go) - small wrapper binaries that wrap the actual AI coding tool (kai CLI, opencode, Claude Code)
- **kai CLI** (C# .NET) - the bundled AI agent runtime used by the kai runner plugin
- **Config service** (Go) - versioned platform configuration with hot-reload
- **Web UI** (React 19 + TypeScript + Vite) - pipeline management, live monitoring, audit, secrets, config
- **kaictl** (Go) - command-line client for the orchestrator

Everything is self-hosted. The whole system is started together via the `simulation/` bundle (`make start-dev`) or `docker compose up` from `simulation/docker/`.

## High-level flow

```
+------------------------------------------------------------------+
|                       USERS                                      |
|  +------------------+  +-----------------+  +------------------+ |
|  |  Web UI          |  |  kaictl         |  |  curl / HTTP     | |
|  |  (React)         |  |  (Go CLI)       |  |  (REST API)      | |
|  +--------+---------+  +--------+--------+  +---------+--------+ |
|           |                     |                     |          |
+------------------------------------------------------------------+
            | HTTP REST + SSE      |                     |
            +----------------------+---------------------+
                                  |
                                  v
+------------------------------------------------------------------+
|                Orchestrator (Go)                                 |
|                                                                  |
|  +-------------+  +-------------+  +-------------------------+  |
|  |  API Layer  |  | Coordinator |  |   Workflow Engine       |  |
|  |  (REST +    |--| (pipeline   |--|  (YAML -> DAG -> state) |  |
|  |   SSE)      |  |  lifecycle) |  |                         |  |
|  +-------------+  +------+------+  +-------------------------+  |
|                          |                                      |
|  +-------------+  +------v------+  +-------------------------+   |
|  | Agent Pool  |  | Validation  |  |   Git Operations        |   |
|  | (gRPC)      |--| (gates)     |  |   (clone, commit, push,  |   |
|  +-------------+  +-------------+  |    PR)                   |   |
|                                    +-------------------------+   |
|  +-------------+  +-------------+  +-------------------------+   |
|  | Audit Log   |  | Secrets     |  |   Versioned Config       |   |
|  |             |  | Manager     |  |   (hot-reload from       |   |
|  +-------------+  +-------------+  |    config service)       |   |
|                                   +-------------------------+   |
+----------------------------------+-------------------------------+
                                   | gRPC
                                   v
+------------------------------------------------------------------+
|                    Agent workers (Go)                            |
|                                                                  |
|  +-------------+  +--------------+  +-------------+              |
|  | agent-kai   |  | agent-       |  | agent-      |              |
|  |             |  | opencode     |  | claude-code |              |
|  +------+------+  +-------+------+  +------+------+              |
|         |                 |                |                     |
|         | subprocess (runner plugin)                              |
|         v                 v                v                     |
|  +-------------+  +--------------+  +-------------+              |
|  | kai CLI     |  | opencode     |  | Claude      |              |
|  | (C# .NET)   |  | binary       |  | Code CLI    |              |
|  | tool-use    |  |              |  |             |              |
|  | loop        |  |              |  |             |              |
|  +------+------+  +-------+------+  +------+------+              |
|         |                 |                |                     |
|         +-----------------+----------------+                     |
|                           v                                      |
|                   LLM provider                                   |
|           (OpenAI, Ollama, Anthropic, etc.)                      |
+------------------------------------------------------------------+
                                  |
                                  v
                        Git provider
              (GitHub / GitLab / Bitbucket / Forgejo)
                  branch push or pull request
```

## Pipeline execution

```
1. User submits YAML pipeline via REST API, Web UI, or kaictl
2. Orchestrator parses YAML, validates structure, creates PipelineRun
3. Coordinator resolves DAG dependencies, identifies ready steps
4. Orchestrator clones the target repo and creates a feature branch
5. Ready steps are dispatched to idle agents via gRPC AssignMission
6. Agent creates a sandboxed workspace (temp dir + repo clone)
7. Agent invokes its runner plugin, which runs the AI coding tool
8. Runner executes the tool-use loop against the LLM
   (read / write files, run commands, search, glob)
9. Agent streams logs and file changes back to orchestrator in real time
10. On step completion, orchestrator runs the configured validation gates
11. If approval is required, the step blocks and waits for human input
12. After all steps pass: git commit + push (branch) or create PR (PR mode)
13. Audit events are written at every stage
```

---

## Component details

### Orchestrator (`kai-platform/orchestrator`)

The central control plane. Exposes the HTTP REST API + SSE stream, owns the workflow engine and agent pool, and runs validation, git, audit, and secrets.

**Internal packages:**

| Package | Responsibility |
| --- | --- |
| `api/` | HTTP REST server + SSE event streaming. Handles pipeline CRUD, agent listing, stats, audit, secrets, approvals. |
| `coordinator/` | Pipeline lifecycle orchestrator - creates runs, dispatches steps, handles completion / failure. |
| `agentpool/` | Agent connection pool - tracks agent state (idle / busy / unhealthy / offline), health checks, FIFO mission queuing, idle-signal draining, stale agent pruning. |
| `workflow/` | Pipeline model, YAML parser, DAG dependency resolution, step state machine with retry / backoff. |
| `validation/` | Pluggable validation gates - `exit_zero`, `lint`, `typecheck`, `tests`, `diff_review`, `approval`, `security_scan`, `license_check`, `breaking_changes`, `code_quality`, plus external plugin gates. |
| `gitops/` | Git operations - clone, branch, commit, push, with git pathspec exclusions for Kai runtime files. |
| `gitprovider/` | Git platform abstractions - GitHub, GitLab, Bitbucket built in, plus external provider plugins (e.g. Forgejo). |
| `audit/` | Event-sourced audit trail - 12 event types stored in PostgreSQL with in-memory fallback. |
| `auth/` | Authentication - 3 modes: insecure, pre-shared Bearer token, TLS mutual auth. |
| `cost/` | Token usage and duration tracking per step, per run, per agent (in-memory only). |
| `secrets/` | Pluggable secrets backends - environment variables, HashiCorp Vault (functional), AWS / Azure / GCP (stubs), plugin, in-memory. |
| `archive/` | Step state archive - save and restore workspace snapshots (local filesystem functional, AWS S3 / Azure Blob / GCS are stubs). |

**Pipeline / step state machines:**

```
Pipeline:
  pending -> running -> completed
                      -> failed
                      -> cancelled

Step:
  pending -> ready -> running -> validating -> passed
                                          -> failed
                                          -> blocked  (awaiting human approval)
```

**Agent communication (gRPC):**

- Orchestrator exposes `Orchestrator` service - agents call `Heartbeat` (client streaming), `ReportLog` (client streaming), `ReportFileChange` (client streaming), `ReportResult` (unary).
- Agents expose `Agent` service - orchestrator calls `AssignMission` (server streaming for mission events), `CancelMission` (unary), `HealthCheck` (unary).
- Real-time log and file-change delivery via client-side gRPC streaming from agent to orchestrator.

**Protocol (`proto/kaiplatform/v1/`):**

| Proto | Service | Key RPCs |
| --- | --- | --- |
| `orchestrator.proto` | `Orchestrator` | `Heartbeat`, `ReportLog`, `ReportFileChange`, `ReportResult` |
| `agent.proto` | `Agent` | `AssignMission`, `CancelMission`, `HealthCheck` |
| `types.proto` | - | `Mission`, `Policy`, `Workspace`, `LogEntry`, `FileChange`, `MissionResult` |

---

### Agent worker (`kai-platform/agent`)

A Go worker that connects to the orchestrator via gRPC. When the orchestrator assigns a mission, the agent:

1. Sets up a sandboxed workspace (temp dir + repo clone if a repo is configured).
2. Resolves the runner plugin by name (default `kai`, can be overridden via the step's `policy.runner`).
3. Invokes the plugin binary, which spawns and supervises the actual AI coding tool.
4. Streams stdout / stderr / file changes back to the orchestrator as log entries.
5. Returns the final `MissionResult` (success, exit code, archive).

The agent itself does not implement the LLM tool-use loop - that is the runner plugin's and the underlying AI tool's job.

**Environment variables (from `simulation/` `.env`):**

| Var | Default | Purpose |
| --- | --- | --- |
| `ORCHESTRATOR_ADDR` | `localhost:50051` | Orchestrator gRPC address |
| `AGENT_ID` | `local-coder-N` | Agent identity |
| `AGENT_LISTEN` | `:5005N` | gRPC bind address for orchestrator callbacks |
| `KAI_PLUGIN_DIR` | `./.kai/plugins` | Where the agent looks up runner plugins |
| `MISSION_TIMEOUT` | `5m` | Per-mission timeout |

---

### Runner plugins (`kai-plugins/runner/`)

Small Go binaries that wrap an external AI coding tool. The agent worker spawns one of these per mission.

| Plugin | Wraps | Notes |
| --- | --- | --- |
| `kai-plugin` | kai CLI (C# .NET) | Bundled AI coding agent with its own tool-use loop |
| `opencode-plugin` | opencode CLI | Passes the mission prompt through |
| `claude-plugin` | Claude Code CLI | Passes the mission prompt through |

The plugin contract: the agent invokes the plugin with CLI flags describing the mission, and the plugin is responsible for writing any per-mission config, running the underlying tool, streaming its output, and returning a structured result.

---

### kai CLI (`kai/`)

The C# .NET agent runtime used by the `kai-plugin`. Implements the tool-use loop against an LLM (read / write / run / search / glob / list_dir), with:

- Multi-agent roles: `ToolCoderAgent` (coding), `TesterAgent` (test generation), `ReviewerAgent` (code review).
- Per-agent LLM model configuration (different model, temperature, etc. for coder / tester / reviewer).
- Context compression when estimated tokens exceed 85% of `MaxContextTokens`.
- Read-file caching (deduplicates repeated reads within a session).
- Project memory persisted in `.kai/` (SHA256-keyed by working dir).
- Policy enforcement: `allowed_tools`, `allowed_commands`, `allowed_dirs`.

**Projects:**

| Project | Responsibility |
| --- | --- |
| `Kai.Cli` | Entry point, command handling |
| `Kai.Core` | Core abstractions - agent interface, models, configuration, event bus, project memory, tools |
| `Kai.Orchestrator` | Local pipeline orchestration - runs coding -> testing -> review phases sequentially |
| `Kai.Agents` | Agent implementations - `ToolCoderAgent`, `TesterAgent`, `ReviewerAgent` |
| `Kai.LLM` | LLM abstraction - `IChatCompletion` with `OpenAiChatCompletion` implementation |
| `Kai.Git` | Git operations via LibGit2Sharp |

---

### Web UI (`kai-platform-ui`)

React 19 + TypeScript + Vite 6. Zero external dependencies (no router, no UI library, no state management) - all components are hand-rolled with React 19 hooks, native `fetch`, native `EventSource` (SSE), and pure CSS custom properties.

**Pages / components:**

| Component | Purpose |
| --- | --- |
| `LandingPage` | Marketing hero page with feature grid |
| `App` (Dashboard) | Pipeline list with search / status filters, agent list, queue widget, stats bar |
| `PipelineDetailView` | Pipeline run details - DAG visualization, step cards with gate results, tabbed conversation log (real-time SSE), approve / cancel controls, copy YAML |
| `PipelineBuilder` | Visual pipeline builder - interactive DAG graph, form-based step editor, validation gate toggles, approval settings, policy section, YAML import / export, live YAML preview |
| `NewPipeline` | Thin wrapper around `PipelineBuilder` |
| `AgentDetailPanel` | Modal showing agent health, state, address, uptime, missions completed, current mission |
| `AuditLog` | Searchable / filterable event log with type and run ID filters, SSE auto-refresh |
| `SecretsPage` | CRUD interface for managing stored secrets (git tokens, LLM API keys) |
| `PlatformConfigPage` | Config version management - browse history, create / edit drafts, publish & activate, rollback with hot-reload / restart tracking |
| `AgentConfigPage` | Per-pool LLM configuration - pool sidebar with live agent indicators, per-role LLM settings, add / remove roles, auto-import live agents |
| `NavBar` | Tab navigation |

**API communication:**

- REST API for CRUD operations (`/api/pipelines`, `/api/agents`, `/api/audit`, `/api/secrets`, `/api/queue`, `/api/stats`, `/api/policies`, `/api/status`, etc.)
- Server-Sent Events (SSE) for real-time updates (`/api/events`)
- Config service proxy at `/api/v1/config/*` for platform configuration management
- Vite dev proxy routes `/api/v1/config` -> config service (port 8081) and `/api` -> orchestrator (port 8080)

---

### kaictl (`kai-cli-control`)

A Go CLI for interacting with the orchestrator over HTTP REST. Built with Cobra. No gRPC dependency.

| Command | Description |
| --- | --- |
| `kaictl login <host> <token>` | Save connection info to `~/.kaictl/config.json` |
| `kaictl status` | Platform health, agent count, queue depth, uptime |
| `kaictl pipeline list` | List all pipeline runs (alias: `ls`) |
| `kaictl pipeline show <id>` | Full pipeline detail with per-step statuses and gate results |
| `kaictl pipeline create <file.yaml>` | Create a pipeline from YAML (aliases: `new`, `run`) |
| `kaictl pipeline cancel <id>` | Cancel a running pipeline |
| `kaictl pipeline approve <id> <step>` | Approve / reject blocked steps (use `--reject` to reject, `-m` for message) |
| `kaictl pipeline logs <id> <step>` | View step conversation logs (alias: `conversation`) |
| `kaictl agent list` | List connected agents (alias: `ls`) |
| `kaictl stats` | Usage statistics; `--runs` for per-run token breakdown |
| `kaictl audit` | Query audit log with `--limit` and `--run-id` filters |
| `kaictl events` | Stream real-time platform events via SSE |

All commands support `--json` for machine-readable output.

---

### Config service (`kai-config-service`)

A standalone REST service for managing platform configuration as versioned JSON documents on disk. It pushes changes to the orchestrator via `POST /api/platform/config/reload`.

**Configuration model:**

- `SystemConfig` - full platform configuration including server, auth, pool, backends, and agent pool definitions
- `KaiConfig` - per-pool kai CLI configuration (language, branch prefix, agents, rules, maxContextTokens, detailed limits)

**Version lifecycle:**

```
draft -> published -> active
                <-> rollback
```

**API endpoints:**

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/config` | Get active config |
| `GET` | `/api/v1/config/versions` | List versions (optional `?status=` filter) |
| `POST` | `/api/v1/config/versions` | Create new draft (based on active config) |
| `GET` | `/api/v1/config/versions/{id}` | Get specific version |
| `PUT` | `/api/v1/config/versions/{id}` | Update draft |
| `POST` | `/api/v1/config/versions/{id}/publish` | Publish draft |
| `POST` | `/api/v1/config/versions/{id}/activate` | Activate (pushes config to orchestrator, returns `ReloadResult` with `hot_reloaded` and `requires_restart` lists) |
| `POST` | `/api/v1/config/versions/{id}/rollback` | Rollback (creates new draft from target, auto-publishes & activates) |
| `GET` | `/api/v1/config/status` | Service status (active version info, total versions) |

**Orchestrator integration:**

- On startup, orchestrator fetches config from config service with exponential backoff (1s -> 30s retries)
- `POST /api/platform/config/reload` endpoint accepts config changes at runtime
- Most settings hot-reload without restart; auth / TLS changes require restart

---

### Plugin system

External plugins discovered from `$KAI_PLUGIN_DIR` (defaults to `./.kai/plugins` per service) via `plugin.json` manifests.

**Plugin types:**

- `runner` - AI agent runtime (kai, opencode, claude-code)
- `gate` - Custom validation gates (run after pipeline steps)
- `gitprovider` - Custom git platform integrations (create PRs)
- `secrets` - Custom secrets backend integrations
- `archive` - Custom archive storage backends

**Manifest format (`plugin.json`):**

```json
{
  "name": "my_gate",
  "version": "1.0.0",
  "api_version": "v1",
  "type": "gate",
  "gate_types": ["my_gate"],
  "binary": "my-gate-binary"
}
```

**Plugin discovery in `simulation/`:**

```
.kai/plugins/
|-- kai/              -> agent-kai runner plugin (type: runner)
|-- opencode/         -> agent-opencode runner plugin (type: runner)
|-- claude-code/      -> agent-claude runner plugin (type: runner)
|-- conventional-commits/  -> orchestrator gate plugin (type: gate)
|-- forgejo/          -> orchestrator gitprovider plugin (type: gitprovider)
|-- local-fs/         -> orchestrator archive plugin (type: archive)
`-- env/              -> orchestrator secrets plugin (type: secrets)
```

Each service only receives the plugins relevant to its role.

---

## Pipeline YAML format

Pipelines are declarative YAML files. Steps form a DAG via `depends_on`. Each step has a natural-language prompt, a policy, a list of validation gates, and an optional human-approval setting.

```yaml
version: 1
project: my-project

output:
  type: pr                # or "branch"
  branch_prefix: feat/kai-

repo:
  url: http://git.example.com/org/repo.git
  base_branch: main
  provider: forgejo
  token_ref: forgejo-token

steps:
  - id: scaffold
    prompt: |
      Create a Go HTTP server...
    depends_on: []
    policy:
      allowed_dirs: ["./"]
      allowed_tools: [read_file, write_file, run, glob, search, list_dir]
      allowed_commands: ["dotnet*", "go*", "npm*"]
      max_retries: 2
      retry_delay_seconds: 15
      retry_backoff: exponential
      timeout_seconds: 600
      save_state: true
    validation:
      - exit_zero
      - lint
      - tests
    approval: optional
```

---

## Deployment

| Mode | Setup |
| --- | --- |
| **Local dev (on host)** | `simulation/start-dev.sh` (or `make start-dev`) - builds everything, then starts config service (port 8081) + orchestrator (HTTP 8080, gRPC 50051) + 3 agents (50052-50054) + Web UI (5173) on the host |
| **Docker** | `cd simulation && make docker-build && docker compose up` - the same 6 services, each in its own container |
| **Individual components** | `dotnet build` in `kai/`, `make build` in `kai-platform/`, `make build` in `kai-cli-control/`, `npm run dev` in `kai-platform-ui/` |

---

## Security

| Layer | Mechanism |
| --- | --- |
| **Transport** | TLS / mTLS for gRPC and HTTP (configurable) |
| **Authentication** | Pre-shared Bearer token, TLS client certs, or insecure (dev only) |
| **Agent sandboxing** | Per-mission temp directories, restricted by policy (allowed dirs, tools, commands) |
| **Policy enforcement** | `allowed_dirs`, `allowed_tools`, `allowed_commands` enforced per step by the runner plugin and the underlying tool |
| **Secrets** | Pluggable backends - env vars, HashiCorp Vault (functional), AWS / Azure / GCP (stubs), plugin, in-memory |
| **Audit** | Immutable event log - 12 event types with timestamps (PostgreSQL or in-memory) |
| **Diff review gate** | Scans git diff for secrets (API keys, tokens, private keys); enforces max file / size limits |

---

## Key design decisions

1. **No task decomposition** - Steps are written as natural-language prompts. The runner tool decides how to fulfill them. There is no recursive planning layer on the platform side.
2. **Declarative YAML pipelines** - Version-controllable and human-readable, modelled on CI / CD systems.
3. **Pluggable runners** - The actual AI coding tool is decoupled from the platform via runner plugins. kai, opencode, and Claude Code are first-class supported runtimes; new ones are a small Go binary.
4. **Per-step LLM configuration** - Different models and parameters per agent role, set in the platform config.
5. **Pluggable everything** - Gates, git providers, secrets backends, archive backends all support binary plugins in any language.
6. **Self-hosted first** - Runs entirely on your infrastructure; no cloud dependency.
7. **gRPC for agent communication** - Client-streaming for log / file delivery from agents, server-streaming for mission assignment; health checks and cancellation are unary.
8. **In-memory defaults with Postgres upgrade path** - Audit, cost, conversation, and secret stores default to in-memory for easy development; PostgreSQL is available for production persistence.
