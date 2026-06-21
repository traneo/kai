<picture>
  <source media="(prefers-color-scheme: dark)" srcset="image/logo.png">
  <img alt="Kai" src="image/logo.png" width="120" align="right">
</picture>

<br>

# Kai

> **The open-source platform for AI-driven software pipelines.**

Kai is a self-hostable pipeline engine that orchestrates AI coding agents (kai-code, opencode, Claude Code) through multi-step workflows — with validation gates, human approvals, parallel DAG execution, git integration, and a full audit trail.

Think "CI/CD, but the steps are written and executed by AI."

---

## Why Kai?

| Problem | How Kai solves it |
|---------|-------------------|
| AI coding tools work on one task at a time | **DAG pipelines** — chain steps with dependencies; run in parallel where possible |
| No validation on AI output | **Validation gates** — lint, typecheck, test, diff review after every step |
| Black-box AI | **Audit trail** — every prompt, file change, and decision is logged |
| No human oversight | **Approval gates** — block steps until a human signs off |
| "Make it a PR" | **Git integration** — clone, branch, commit, push, PR — all automated |
| Multiple AI backends | **Pluggable runners** — kai-code, opencode, Claude Code, or your own |
| "Where do I even start?" | **Plan Builder** — describe your project in plain English, the LLM builds the pipeline for you |

---

## How it works

```
[Plan Builder]  ──conversation──>  spec  ──LLM──>  pipeline.yaml
                                                          |
                                                          v
                                          ┌──────────────────────────────┐
                                          │      Orchestrator (Go)       │
                                          │  parse → DAG → clone repo    │
                                          │  → dispatch → gates → PR     │
                                          └──────────────┬───────────────┘
                                                          │ gRPC + HTTP
                                                          v
                                          ┌──────────────────────────────┐
                                          │       Agent Workers          │
                                          │  ┌────────┐ ┌──────────┐    │
                                          │  │kai-code │ │ opencode │    │
                                          │  │  (C#)   │ │   (Go)   │    │
                                          │  └───┬────┘ └────┬─────┘    │
                                          │      │           │          │
                                          │      v           v          │
                                          │   LLM provider              │
                                          │  (Ollama / OpenAI / etc.)   │
                                          └──────────────────────────────┘

                                 Observability (Go) ←── SDK ── All services
                                       │
                                       v
                                 Observability UI (React)
```

---

## What's in the box

### kai-platform — Pipeline engine (Go)
The orchestrator and agent worker. Parses YAML into a DAG, dispatches steps to agents via gRPC, runs validation gates, manages retries, handles approvals, and pushes results to git. Built with:
- **DAG scheduler** — static analysis of `depends_on`, cycle detection, parallel execution
- **Step state machine** — `pending → ready → running → validating → passed/failed → blocked`
- **Retry with backoff** — linear or exponential, configurable per step
- **Hot-reload config** — pool config, auth, and backends reload without restart

### kai-platform-ui — Pipeline monitor (React 19)
Dashboard for tracking pipeline runs, agent status, queue depth, and audit logs. All real-time via SSE.

### kai-plan-builder — Spec-to-YAML via LLM (Go)
A conversational agent that walks you through building a complete pipeline spec. No YAML knowledge needed.
- **Chat interface** — tell it what you want, it asks clarifying questions
- **Spec editing** — LLM writes a structured spec, you edit it live
- **YAML generation** — converts the spec to a valid `pipeline.yaml` with retry + self-correction

### kai-plan-builder-ui — Spec builder UI (React 19)
Split-panel interface: spec editor on the left, LLM chat on the right. Transition to a visual pipeline editor with raw YAML tab for final review.

### kai-code — AI agent runtime (C# .NET)
The production agent that executes pipeline steps against an LLM. Reads the repo, writes code, runs commands, and reports results back to the orchestrator.

### kai-cli-control — CLI tool (Go)
`kaictl` — manage pipelines, agents, secrets, and config from the terminal. JSON output for scripting.

### kai-config-service — Versioned config (Go)
Draft → publish → activate → rollback lifecycle for platform config. Most settings hot-reload without restarting services.

### kai-observability — Centralized logs (Go)
Receives structured log entries from all services via the SDK. Query by service, level, run ID, step ID. Live SSE stream. PostgreSQL or in-memory storage.

### kai-observability-ui — Log viewer (React 19)
Filter, search, and stream logs in real time. Performance charts, run timelines, error dashboards, and auto-refresh.

### kai-observability-sdk — Logging SDKs (Go / TypeScript / C#)
Non-blocking batched logging with scoped loggers (`WithRunID()`, `WithStepID()`, etc.). Drop-in integration.

### kai-plugins — Plugin ecosystem (Go)
Extend every part of the platform without modifying core:
- **Runners** — `kai-code`, `opencode`, `claude-code` — AI tool integrations
- **Gates** — `conventional-commits` — custom validation logic
- **Git providers** — `forgejo` — platform-specific git + PR APIs
- **Secrets** — `env` — secret storage backends
- **Archive** — `local-fs` — step state persistence

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

See [examples/](examples/) for real pipelines and [docs/pipeline-schema.yaml](docs/pipeline-schema.yaml) for the full schema.

---

## Quick start

```bash
cd simulation
make build          # builds everything
make start-dev      # runs everything locally
```

| Service              | URL/Port                   |
|----------------------|----------------------------|
| Platform UI          | http://localhost:5173      |
| Plan Builder UI      | http://localhost:5175      |
| Observability UI     | http://localhost:5174      |
| Orchestrator API     | http://localhost:8080      |
| Config Service       | http://localhost:8081      |
| Observability API    | http://localhost:8082      |
| Plan Builder API     | http://localhost:8083      |

Submit a pipeline:
```bash
kaictl pipeline create --file examples/pipeline-forgejo.yaml
# Or open the Plan Builder: http://localhost:5175
```

---

## Requirements

- Go 1.25+, Node.js 20+, .NET 10 SDK, jq
- An OpenAI-compatible LLM endpoint (Ollama, OpenAI, etc.)

---

## Repository

```
kai/
├── kai-platform/             # Pipeline engine — orchestrator + agent worker
├── kai-platform-ui/          # Pipeline monitoring dashboard
├── kai-plan-builder/         # LLM-powered spec-to-YAML generator
├── kai-plan-builder-ui/      # Spec builder interface
├── kai-code/                 # C# .NET AI agent runtime
├── kai-cli-control/          # kaictl CLI
├── kai-config-service/       # Versioned configuration management
├── kai-observability/        # Centralized logging service
├── kai-observability-ui/     # Log viewer and analysis
├── kai-observability-sdk/    # Logging SDKs (Go / TS / C#)
├── kai-plugins/              # Plugin ecosystem
├── simulation/               # Build + dev runner
├── examples/                 # Sample pipeline YAMLs
├── docs/                     # Architecture + schema reference
├── image/                    # Project assets
├── Makefile                  # Top-level build targets
└── go.work                   # Go workspace
```

---

## License

MIT
