# kaictl User Guide

## Configuration

### `kaictl login`

Save the platform URL and auth token so you don't need to pass them every time.

```bash
kaictl login http://localhost:8080 my-secret-token
```

This writes `~/.kaictl/config.json`:

```json
{
  "host": "http://localhost:8080",
  "token": "my-secret-token"
}
```

You can also create this file manually. At runtime, `--host` and `--token` override
whatever is in the file.

## Platform

### `kaictl status`

Check the platform health — agent count, queue depth, pipeline count, version.

```bash
# Text output
kaictl status

# JSON
kaictl status --json
```

Example text output:

```
Version    0.1.0
Agents     3 (idle: 2, busy: 1)
Queue      0 missions
Pipelines  5 total
Uptime     2026-06-14T12:00:00Z
```

## Pipelines

### `kaictl pipeline list` (aliases: `ls`)

List all pipeline runs with their status and step counts.

```bash
kaictl pipeline list
kaictl pipeline list --json
```

Columns: ID, project, status, steps, passed, failed, blocked, queued.

### `kaictl pipeline show <id>` (alias: `get`)

Full detail of a pipeline run, including every step's status, retries,
gate results, and error messages.

```bash
kaictl pipeline show run-abc-1234567890000
kaictl pipeline show run-abc-1234567890000 --json
```

Text output shows each step with a status icon:

| Icon | Status |
|------|--------|
| ✓ | passed |
| ✗ | failed |
| ▶ | running |
| ⊘ | blocked |
| ○ | pending/ready |
| – | cancelled |

### `kaictl pipeline create <file.yaml>` (aliases: `new`, `run`)

Create a pipeline run by submitting a YAML definition file.

```bash
kaictl pipeline create my-pipeline.yaml
kaictl pipeline create my-pipeline.yaml --json
```

The command reads the YAML file and sends it to `POST /api/pipelines`
wrapped in `{"yaml": "..."}`. On success it prints the new pipeline ID
and status.

Example YAML:

```yaml
version: 1
project: my-project
output:
  type: pr
  branch_prefix: feat/

steps:
  - id: scaffold
    prompt: "Initialize project structure"
    validation:
      - exit_zero
  - id: implement
    prompt: "Implement the feature"
    depends_on:
      - scaffold
    validation:
      - exit_zero
      - lint
      - tests
```

### `kaictl pipeline cancel <id>`

Cancel a running pipeline. Idempotent — safe to call on already-finished
pipelines.

```bash
kaictl pipeline cancel run-abc-1234567890000
```

### `kaictl pipeline approve <id> <step>` (alias: `reject`)

Approve or reject a step that is blocked waiting for human review.

```bash
# Approve
kaictl pipeline approve run-abc-1234567890000 step-implement

# Approve with a message
kaictl pipeline approve run-abc-1234567890000 step-implement -m "verified manually"

# Reject
kaictl pipeline approve run-abc-1234567890000 step-implement --reject \
  -m "does not meet requirements"
```

The API expects `{"action": "approve"|"reject", "message": "..."}`.
The `reject` alias is provided as a convenience — both forms work the same.

### `kaictl pipeline logs <id> <step>` (alias: `conversation`)

View the log and file-change conversation for a specific step. Useful for
debugging what the agent did during a mission.

```bash
kaictl pipeline logs run-abc-1234567890000 step-implement
kaictl pipeline logs run-abc-1234567890000 step-implement --limit 500 --json
```

Each log entry shows the source (`agent`, `system`, `file_change`) and message.

## Agents

### `kaictl agent list` (alias: `ls`)

List all agents connected to the platform with their state, health,
mission count, and uptime.

```bash
kaictl agent list
kaictl agent list --json
```

## Statistics

### `kaictl stats`

Aggregated usage statistics — agent counts, queue depth, pipeline count,
steps, token usage, and total duration.

```bash
kaictl stats
kaictl stats --runs       # include per-run token breakdown
kaictl stats --json
```

## Audit

### `kaictl audit`

Query the audit event log. Supports filtering and pagination.

```bash
# Last 50 events (default)
kaictl audit

# Custom limit
kaictl audit --limit 100

# Filter by run
kaictl audit --run-id run-abc-1234567890000

# Combined
kaictl audit --limit 200 --run-id run-abc-1234567890000

# Machine-readable
kaictl audit --json
```

Each audit event shows: ID, time, type, run ID, and message.

## Events

### `kaictl events`

Connect to the platform's SSE endpoint and stream real-time events to
stdout. Press Ctrl+C to disconnect.

```bash
kaictl events
kaictl events --json
```

Events appear as they happen:

```
connected — waiting for events (Ctrl+C to stop)
[pipeline_created] run-abc-1234567890000: created
[step_started] run-abc-1234567890000: starting step-implement
[step_completed] run-abc-1234567890000: step-implement passed
```

## Automation

Every command supports `--json` for structured output, making the CLI
easy to use in scripts:

```bash
# Capture pipeline ID from created run
ID=$(kaictl pipeline create my-pipeline.yaml --json | jq -r '.id')
echo "Created $ID"

# Wait for completion
while true; do
  STATUS=$(kaictl pipeline show "$ID" --json | jq -r '.status')
  [ "$STATUS" = "completed" ] && break
  sleep 5
done
```
