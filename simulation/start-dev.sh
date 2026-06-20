#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
PIDS=()

ORCHESTRATOR_PORT="${ORCHESTRATOR_PORT:-50051}"
HTTP_PORT="${HTTP_PORT:-8080}"
CONFIG_PORT="${CONFIG_PORT:-8081}"
OBSERVABILITY_PORT="${OBSERVABILITY_PORT:-8082}"
PLAN_BUILDER_PORT="${PLAN_BUILDER_PORT:-8083}"

KAI_ENDPOINT="${KAI_ENDPOINT:-http://192.168.0.44:11434/v1/}"
KAI_MODEL="${KAI_MODEL:-gemma4-12b-256k}"
CLAUDE_API_URL="${CLAUDE_API_URL:-http://192.168.0.44:11434/v1}"
CLAUDE_MODEL="${CLAUDE_MODEL:-gemma444-128k}"
CLAUDE_API_KEY="${CLAUDE_API_KEY:-}"
MISSION_TIMEOUT="${MISSION_TIMEOUT:-5m}"

cleanup() {
	echo ""
	echo "Shutting down..."
	kill "${PIDS[@]}" 2>/dev/null
	wait 2>/dev/null
	echo "Done."
}
trap cleanup INT TERM

if ! command -v jq &>/dev/null; then
	echo "ERROR: jq is required. Install it with: brew install jq"
	exit 1
fi

echo "=== Cleanning the binaries ==="
make clean

echo "=== Build full project ==="
make build

echo "=== Creating tmp directories ==="
mkdir -p "$ROOT/tmp"/{config-service,orchestrator,agent-kai-code,agent-opencode,agent-claude,observability-logs,plan-builder}

# ---- Config service ----

echo "=== Starting config-service ==="
"$ROOT/config-service/config-service" \
	--port "$CONFIG_PORT" \
	--data-dir "$ROOT/tmp/config-service/data" \
	--orchestrator-url "http://localhost:${HTTP_PORT}" &
CFG_PID=$!
PIDS+=($CFG_PID)

for i in $(seq 1 15); do
	if curl -sf "http://localhost:${CONFIG_PORT}/api/v1/config/status" >/dev/null 2>&1; then
		echo "  Config service ready"
		break
	fi
	if [ "$i" -eq 15 ]; then echo "Config service failed to start"; exit 1; fi
	sleep 1
done

# ---- Observability ----

echo "=== Starting observability ==="
OBSERVABILITY_FILE_DUMP=true \
OBSERVABILITY_FILE_DUMP_PATH="$ROOT/tmp/observability-logs" \
	"$ROOT/observability/observability" --port "$OBSERVABILITY_PORT" &
OBS_PID=$!
PIDS+=($OBS_PID)

for i in $(seq 1 15); do
	if curl -sf "http://localhost:${OBSERVABILITY_PORT}/healthz" >/dev/null 2>&1; then
		echo "  Observability ready"
		break
	fi
	if [ "$i" -eq 15 ]; then echo "Observability failed to start"; exit 1; fi
	sleep 1
done

OBSERVABILITY_URL="http://localhost:${OBSERVABILITY_PORT}"

# ---- Orchestrator ----

echo "=== Starting orchestrator ==="
KAI_PLUGIN_DIR="$ROOT/orchestrator/.kai-code/plugins" \
	PORT="$ORCHESTRATOR_PORT" \
	HTTP_PORT="$HTTP_PORT" \
	CONFIG_SERVICE_URL="http://localhost:${CONFIG_PORT}" \
	OBSERVABILITY_URL="${OBSERVABILITY_URL}" \
	"$ROOT/orchestrator/orchestrator" \
	--http-port "$HTTP_PORT" \
	--config-service-url "http://localhost:${CONFIG_PORT}" &
ORCH_PID=$!
PIDS+=($ORCH_PID)

for i in $(seq 1 30); do
	if curl -sf "http://localhost:${HTTP_PORT}/healthz" >/dev/null 2>&1; then
		echo "  Orchestrator ready"
		break
	fi
	if [ "$i" -eq 30 ]; then echo "Orchestrator failed to start"; exit 1; fi
	sleep 1
done

# ---- Push platform config ----

echo "=== Pushing platform config ==="
DRAFT=$(curl -sf -X POST "http://localhost:${CONFIG_PORT}/api/v1/config/versions")
DRAFT_ID=$(echo "$DRAFT" | jq -r '.id')
echo "  Draft: $DRAFT_ID"

CONFIG_JSON=$(cat <<ENDJSON
{
  "config": {
    "plan_builder": {
      "llm": {
        "endpoint": "${KAI_ENDPOINT}",
        "model": "${KAI_MODEL}",
        "api_key": ""
      }
    },
    "platform": {
      "auth": {"pre_shared_token": ""},
      "pool": {"heartbeat_timeout": "30s", "health_interval": "10s"},
      "server": {"http_port": "${HTTP_PORT}", "grpc_port": "${ORCHESTRATOR_PORT}", "version": "0.1.0"},
      "backends": {"secrets": "memory", "audit": "memory", "archive": "local", "plugin_dir": ""}
    },
    "pools": [
      {
        "name": "local-coder-1",
        "blob": {
          "runner": "kai-code",
          "data": {
            "branchPrefix": "kai-code/",
            "autoCommit": true,
            "maxContextTokens": 256000,
            "agents": {
              "coder": {
                "endpoint": "${KAI_ENDPOINT}",
                "model": "${KAI_MODEL}",
                "provider": "openai-compatible",
                "temperature": 0.6, "topP": 0.95, "topK": 25
              },
              "reviewer": {
                "endpoint": "${KAI_ENDPOINT}",
                "model": "${KAI_MODEL}",
                "provider": "openai-compatible",
                "temperature": 0.6, "topP": 0.95, "topK": 25
              },
              "tester": {
                "endpoint": "${KAI_ENDPOINT}",
                "model": "${KAI_MODEL}",
                "provider": "openai-compatible",
                "temperature": 0.6, "topP": 0.95, "topK": 25
              }
            },
            "limits": {
              "agentLoop": {"maxIterations": 300, "maxToolPairs": 25, "compressThreshold": 85, "keepLastPairs": 10, "readFileOutputChars": 8000, "toolOutputChars": 600},
              "retries": {"testFixAttempts": 10, "reviewFixAttempts": 10, "llmApiRetries": 3, "llmRetryDelaySeconds": [1,3,10], "gateTimeoutMinutes": 5},
              "output": {"searchResults": 50, "searchFileSizeBytes": 1048576, "filePathMaxChars": 200, "testOutputChars": 3000, "goalSummaryChars": 100, "keyFilesCount": 30, "dependenciesCount": 15, "relatedFilesCount": 5, "previewLines": 30, "sourceFilesCount": 30, "conventionSamples": 5, "recentGoalsCount": 3},
              "llm": {"maxTokens": 4096},
              "display": {"logChars": 120, "eventToolArgsChars": 200, "eventOutputChars": 300, "eventMessageChars": 100, "summaryToolsCount": 5, "summaryToolLineChars": 80},
              "memory": {"maxTaskHistoryEntries": 100}
            }
          }
        }
      },
      {
        "name": "local-coder-2",
        "blob": {"runner": "opencode", "data": {}}
      },
      {
        "name": "local-coder-3",
        "blob": {
          "runner": "claude-code",
          "data": {
            "apiUrl": "${CLAUDE_API_URL}",
            "model": "${CLAUDE_MODEL}",
            "apiKey": "${CLAUDE_API_KEY}"
          }
        }
      }
    ]
  },
  "message": "simulation config"
}
ENDJSON
)

curl -sf -X PUT "http://localhost:${CONFIG_PORT}/api/v1/config/versions/${DRAFT_ID}" \
	-H "Content-Type: application/json" \
	-d "$CONFIG_JSON" | jq -c '.version, .status'

curl -sf -X POST "http://localhost:${CONFIG_PORT}/api/v1/config/versions/${DRAFT_ID}/publish" >/dev/null
ACTIVATE=$(curl -sf -X POST "http://localhost:${CONFIG_PORT}/api/v1/config/versions/${DRAFT_ID}/activate")
echo "  Activated: $(echo "$ACTIVATE" | jq -c '.status, .hot_reloaded')"

# ---- Plan Builder ----

echo "=== Starting plan-builder ==="
PORT="$PLAN_BUILDER_PORT" \
CONFIG_SERVICE_URL="http://localhost:${CONFIG_PORT}" \
OBSERVABILITY_URL="${OBSERVABILITY_URL}" \
	"$ROOT/plan-builder/plan-builder" &
PB_PID=$!
PIDS+=($PB_PID)

for i in $(seq 1 15); do
	if curl -sf "http://localhost:${PLAN_BUILDER_PORT}/api/v1/plan-builder/health" >/dev/null 2>&1; then
		echo "  Plan builder ready"
		break
	fi
	if [ "$i" -eq 15 ]; then echo "Plan builder failed to start"; exit 1; fi
	sleep 1
done

# ---- Agents ----

echo "=== Starting agent-kai-code ==="
KAI_PLUGIN_DIR="$ROOT/agent-kai-code/.kai-code/plugins" \
	"$ROOT/agent-kai-code/agent" \
	--orchestrator "localhost:${ORCHESTRATOR_PORT}" \
	--agent-id "local-coder-1" \
	--agent-addr "localhost:50052" \
	--listen ":50052" \
	--timeout "${MISSION_TIMEOUT}" &
PIDS+=($!)

echo "=== Starting agent-opencode ==="
KAI_PLUGIN_DIR="$ROOT/agent-opencode/.kai-code/plugins" \
	"$ROOT/agent-opencode/agent" \
	--orchestrator "localhost:${ORCHESTRATOR_PORT}" \
	--agent-id "local-coder-2" \
	--agent-addr "localhost:50053" \
	--listen ":50053" \
	--timeout "${MISSION_TIMEOUT}" &
PIDS+=($!)

echo "=== Starting agent-claude ==="
KAI_PLUGIN_DIR="$ROOT/agent-claude/.kai-code/plugins" \
	"$ROOT/agent-claude/agent" \
	--orchestrator "localhost:${ORCHESTRATOR_PORT}" \
	--agent-id "local-coder-3" \
	--agent-addr "localhost:50054" \
	--listen ":50054" \
	--timeout "${MISSION_TIMEOUT}" &
PIDS+=($!)

# ---- UI ----

echo "=== Starting platform UI ==="
cd "$ROOT/../kai-platform-ui" && npx vite preview --port 5173 &
UI_PID=$!
PIDS+=($UI_PID)

echo "=== Starting observability UI ==="
cd "$ROOT/../kai-observability-ui" && npx vite preview --port 5174 &
OBS_UI_PID=$!
PIDS+=($OBS_UI_PID)

echo "=== Starting plan-builder UI ==="
cd "$ROOT/../kai-plan-builder-ui" && npx vite preview --port 5175 &
PB_UI_PID=$!
PIDS+=($PB_UI_PID)

# ---- Summary ----

echo ""
echo "============================================"
echo "  kai Platform — Simulation"
echo "============================================"
echo ""
echo "  local-coder-1  →  kai-code    (gRPC :50052)"
echo "  local-coder-2  →  opencode    (gRPC :50053)"
echo "  local-coder-3  →  claude-code (gRPC :50054)"
echo ""
echo "  Observability:    http://localhost:${OBSERVABILITY_PORT}"
echo "  Config-service:   http://localhost:${CONFIG_PORT}"
echo "  Orchestrator:     http://localhost:${HTTP_PORT}  |  gRPC :${ORCHESTRATOR_PORT}"
echo "  Plan Builder:     http://localhost:${PLAN_BUILDER_PORT}"
echo "  UI:               http://localhost:5173"
echo "  Observability UI: http://localhost:5174"
echo "  Plan Builder UI:  http://localhost:5175"
echo "  kaictl:           $ROOT/kaictl"
echo ""
echo "  Submit a test:"
echo "    curl -X POST http://localhost:${HTTP_PORT}/api/pipelines \\"
echo '      -H "Content-Type: application/json" \'
echo '      -d '\''{"yaml": "version: 1\nproject: test\nsteps:\n  - id: step-1\n    prompt: hello\n    approval: optional"}'\'
echo ""
echo "  Check agents:"
echo "    curl http://localhost:${HTTP_PORT}/api/agents | jq ."
echo ""
echo "Press Ctrl+C to stop all services."
echo ""

wait
