# Observability Architecture — Centralized Log Flow

## Design Decisions

### Centralized API (kai-observability)
All logs converge into a single HTTP API service (`kai-observability`, port 8082). Every service sends structured log entries there. The API stores them (PostgreSQL in prod, in-memory in dev) and streams them live via SSE.

### Orchestrator as the Forwarding Hub
Agents do **not** know about kai-observability. They send log entries via gRPC (`ReportLog` RPC and `MissionEvent` stream) to the orchestrator. The orchestrator is the single point that forwards those entries to the logging API. This gives us:
- Central control over what gets logged and with what context
- Correlation IDs (run_id, step_id, mission_id) resolved at the orchestrator level
- No need to configure every agent with an observability endpoint

### Plugin Strategy (Option A)
Plugins (kai-code, opencode, claude-code) write to stdout/stderr only. The agent captures those lines, wraps them as `LogEntry` protobuf messages, and sends them through the gRPC mission stream to the orchestrator. The orchestrator then forwards them to kai-observability. Plugins never call the SDK.

### Non-Blocking Everywhere
Every SDK client uses a buffered channel and background goroutine to flush batches asynchronously. If the observability API is down or slow, calling services do not block. Logs are dropped silently if the buffer is full (bounded capacity).

---

## Log Entry Schema

```json
{
  "id":         "service-message-1718000000000",
  "service":    "orchestrator | agent | config-service | kai-code | cli",
  "level":      "info | warn | error | debug",
  "message":    "human-readable log line",
  "timestamp":  1718000000000,
  "run_id":     "pipeline run identifier",
  "step_id":    "step identifier",
  "mission_id": "agent mission identifier",
  "agent_id":   "agent identifier",
  "metadata":   { "duration_ms": 123, "exit_code": 0, ... },
  "received_at": "2025-06-20T12:00:00Z"
}
```

---

## Log Flow by Component

```
┌──────────────────────────────────────────────────────────────────────────┐
│                         kai-observability API                            │
│                        (port 8082, Go service)                           │
│                                                                           │
│  POST /api/v1/logs         ← single entry                                 │
│  POST /api/v1/logs/batch   ← batch (from SDK flush)                      │
│  GET  /api/v1/logs         ← query (filters: service, level, time, ...)  │
│  GET  /api/v1/logs/stream  ← SSE live stream                             │
│                                                                           │
│  Store: PostgreSQL | in-memory ring buffer                                │
│  Debug: optional file dump to tmp/observability-logs/*.jsonl              │
└──────────────────────────────────────────────────────────────────────────┘
            ▲                    ▲                    ▲
            │ HTTP               │ HTTP               │ HTTP
            │                    │                    │
    ┌───────┴───────┐   ┌───────┴────────┐   ┌───────┴──────┐
    │  Orchestrator │   │ Config Service │   │  Agent (Go)  │
    │    (Go SDK)   │   │    (Go SDK)    │   │   (Go SDK)   │
    └───────┬───────┘   └────────────────┘   └───────┬──────┘
            │                                        │
            │ gRPC streams                           │ subprocess capture
            │                                        │
    ┌───────┴───────────────────────────────────────┴──────┐
    │                    Agent Workers                      │
    │  (send LogEntry via AssignMission event stream)       │
    └───────────────────────────┬──────────────────────────┘
                                │
                    ┌───────────┴───────────┐
                    │      Plugins           │
                    │  (stdout only)         │
                    └───────────────────────┘

┌──────────────────────────────────────────────────────────────────────────┐
│                           kai-code (C#)                                  │
│                                                                            │
│  Existing ILogger<T> calls in business logic → KaiObservabilityProvider   │
│  → batched HTTP POST to kai-observability API                            │
│  Zero business code changes required.                                    │
└──────────────────────────────────────────────────────────────────────────┘
```

### 1. Orchestrator

| What | Where | SDK call |
|------|-------|----------|
| Agent log entry via `ReportLog` gRPC | `internal/api/server.go:128-138` | `obsLogger.Info/Warn/Error/Debug(msg, source, mission_id, sequence)` |
| Agent log entry via `MissionEvent` stream | `internal/api/coordinator/coordinator_mission.go:326-346` | `obsLogger.WithRunID().WithStepID().WithMissionID().Info/Warn/Error/Debug(msg, source, sequence)` |
| Agent `FileChange` via `MissionEvent` | `internal/api/coordinator/coordinator_mission.go:344-347` | `obsLogger.WithRunID().WithStepID().WithMissionID().Info("file change", type, path)` |
| Mission result | `internal/api/server.go:158-161` | `obsLogger.WithMissionID().Info("mission result", success, tokens, duration)` |
| Mission stream error | `internal/api/server.go:267-271` | `obsLogger.WithMissionID().Error("mission stream error", err)` |
| Startup/shutdown | `cmd/server/main.go` | `obsLogger.Info(...)` |

### 2. Agent

| What | Where | SDK call |
|------|-------|----------|
| Sandbox failure | `cmd/agent/main.go:146` | `obsLogger.WithMissionID().Error("sandbox setup failed")` |
| Unknown runner | `cmd/agent/main.go:220` | `obsLogger.WithMissionID().Error("unknown runner")` |
| Write config failure | `cmd/agent/main.go:250-251` | `obsLogger.WithMissionID().Error("write config failed")` |
| Runner execution failure | `cmd/agent/main.go:287-288` | `obsLogger.WithMissionID().Error("runner run failed")` |
| Report result failure | `cmd/agent/main.go:78-80` | `obsLogger.WithMissionID().Error("report result to orchestrator failed")` |
| Heartbeat failure | `cmd/agent/main.go:60` | `obsLogger.Error("heartbeat loop ended")` |
| Server startup/shutdown | `cmd/agent/main.go:96-98,107-109` | `obsLogger.Info(...)` |

### 3. Config Service

| What | Where | SDK call |
|------|-------|----------|
| Server startup | `cmd/server/main.go:41` | `obsLogger.Info("config service listening")` |

### 4. kai-code (C#)

| What | Where | SDK call |
|------|-------|----------|
| All existing `ILogger<T>` calls | Every class with `ILogger<T>` | Automatically routed through `KaiObservabilityProvider` — no code changes |

### 5. Plugins (Option A)

| What | How |
|------|-----|
| Plugin stdout/stderr | Captured by agent's `onLine` callback → wrapped as `LogEntry` → gRPC → orchestrator → SDK |
| Plugin never calls SDK | No SDK dependency in any plugin |

---

## SDK Architecture

All three SDKs (Go, C#, TypeScript) share the same design:

```
log() → buffered channel → background goroutine
                              │
                    flush every 1s or 50 entries
                              │
                    POST /api/v1/logs/batch
```

- **Non-blocking writes**: `log()` uses `select { case ch <- e: default: }` — drops if buffer full
- **Configurable**: batch size (default 50), flush interval (default 1s), queue capacity (default 10K)
- **Scoped loggers**: `WithRunID()`, `WithStepID()`, `WithMissionID()`, `WithAgentID()` return a new logger with those fields pre-set on every entry
- **Close**: flushes remaining entries before shutdown

---

## Storage

### PostgreSQL (production)
Table `log_entries`:
```sql
CREATE TABLE log_entries (
    id          TEXT PRIMARY KEY,
    service     TEXT NOT NULL,
    level       TEXT NOT NULL,
    message     TEXT NOT NULL,
    ts          BIGINT NOT NULL,          -- unix millis
    run_id      TEXT DEFAULT '',
    step_id     TEXT DEFAULT '',
    mission_id  TEXT DEFAULT '',
    agent_id    TEXT DEFAULT '',
    metadata    JSONB DEFAULT '{}',
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON log_entries(ts DESC);
CREATE INDEX ON log_entries(service, ts DESC);
CREATE INDEX ON log_entries(level, ts DESC);
```

### In-Memory (dev fallback)
Ring buffer with 50K entry capacity. Same pattern as `MemoryConversationStore`.

### File Dump (debug)
Non-blocking decorator — writes JSONL files to `tmp/observability-logs/logs_YYYY-MM-DD.jsonl`. Enabled via `OBSERVABILITY_FILE_DUMP=true` (default on in `start-dev.sh`).

---

## SSE Live Stream

`GET /api/v1/logs/stream` reuses the orchestrator's exact subscriber pattern:
- `subscribers map[chan LogEntry]struct{}`
- Non-blocking publish to all subscribers
- Subscribe on connect, unsubscribe on disconnect
- Server-Sent Events with `data: <json>\n\n` frames

All services' logs appear in a single unified live stream.

---

## Simulation / Dev Setup

In `start-dev.sh`:
1. Observability starts first on port 8082
2. File dump enabled, writing to `tmp/observability-logs/`
3. `OBSERVABILITY_URL=http://localhost:8082` set for all other services
4. Summary shows observability endpoint alongside other services
