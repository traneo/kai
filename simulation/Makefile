ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/..)
SIM  := $(ROOT)/simulation

.PHONY: build clean start-dev docker-build env-files \
        env-orchestrator env-config-service env-agent-kai-code env-agent-opencode env-agent-claude \
        build-agent build-orchestrator build-config-service build-kaictl build-ui build-kai-code-cli \
        build-plugins \
        dist-agent-kai-code dist-agent-opencode dist-agent-claude dist-orchestrator

build: build-agent build-orchestrator build-config-service build-kaictl build-ui build-kai-code-cli build-plugins dist-orchestrator dist-agent-kai-code dist-agent-opencode dist-agent-claude

start-dev: build
	./start-dev.sh

docker-build: build env-files
	cd docker && docker compose build

# --- Environment variable reference files ---

env-files: env-orchestrator env-config-service env-agent-kai-code env-agent-opencode env-agent-claude

env-orchestrator:
	@mkdir -p $(SIM)/orchestrator
	@{\
	  echo "# Orchestrator - Environment Variables";\
	  echo "# Copy values below to .env and adjust as needed.";\
	  echo "";\
	  echo "# --- Server ---";\
	  echo "PORT=50051                           # gRPC listen port";\
	  echo "HTTP_PORT=8080                       # HTTP API port";\
	  echo "CONFIG_SERVICE_URL=http://localhost:8081";\
	  echo "";\
	  echo "# --- Plugins ---";\
	  echo "KAI_PLUGIN_DIR=./.kai-code/plugins     # plugin discovery directory";\
	  echo "";\
	  echo "# --- Auth ---";\
	  echo "INSECURE=true                        # set false in production";\
	  echo "AUTH_TOKEN=";\
	  echo "TLS_CERT_FILE=";\
	  echo "TLS_KEY_FILE=";\
	  echo "TLS_CA_FILE=";\
	  echo "";\
	  echo "# --- Audit ---";\
	  echo "DATABASE_URL=                        # PostgreSQL DSN; empty = in-memory";\
	  echo "";\
	  echo "# --- Archive storage ---";\
	  echo "ARCHIVE_DIR=data/state-archives";\
	  echo "AWS_BUCKET=";\
	  echo "AWS_REGION=";\
	  echo "AZURE_STORAGE_ACCOUNT=";\
	  echo "AZURE_STORAGE_CONTAINER=";\
	  echo "GCS_BUCKET=";\
	  echo "";\
	  echo "# --- Secrets ---";\
	  echo "OPENAI_API_KEY=";\
	  echo "ANTHROPIC_API_KEY=";\
	  echo "GIT_TOKEN=";\
	  echo "LLM_ENDPOINT=";\
	  echo "LLM_MODEL=";\
	  echo "LLM_API_KEY=";\
	  echo "VAULT_ADDR=";\
	  echo "VAULT_TOKEN=";\
	  echo "VAULT_MOUNT_PATH=secret";\
	  echo "VAULT_INSECURE=false";\
	} > $(SIM)/orchestrator/.env.docs
	@sed -n '/^[A-Z_]/ { s/ *  *#.*//; p; }' $(SIM)/orchestrator/.env.docs > $(SIM)/orchestrator/.env
	@echo "Generated .env and .env.docs for orchestrator"

env-config-service:
	@mkdir -p $(SIM)/config-service
	@{\
	  echo "# Config Service - Environment Variables";\
	  echo "# Copy values below to .env and adjust as needed.";\
	  echo "";\
	  echo "CONFIG_PORT=8081                     # HTTP listen port";\
	  echo "CONFIG_DATA_DIR=./tmp/config-service/data";\
	  echo "ORCHESTRATOR_URL=http://localhost:8080  # base URL for config reload push";\
	} > $(SIM)/config-service/.env.docs
	@sed -n '/^[A-Z_]/ { s/ *  *#.*//; p; }' $(SIM)/config-service/.env.docs > $(SIM)/config-service/.env
	@echo "Generated .env and .env.docs for config-service"

env-agent-kai-code:
	@mkdir -p $(SIM)/agent-kai-code
	@{\
	  echo "# Agent - Environment Variables";\
	  echo "# Copy values below to .env and adjust as needed.";\
	  echo "";\
	  echo "ORCHESTRATOR_ADDR=localhost:50051    # orchestrator gRPC address";\
	  echo "AGENT_ID=local-coder-1               # unique agent identifier";\
	  echo "AGENT_ADDR=localhost:50052           # address announced to orchestrator";\
	  echo "AGENT_LISTEN=:50052                  # gRPC server bind address";\
	  echo "MISSION_TIMEOUT=5m                   # per-mission timeout";\
	  echo "KAI_PLUGIN_DIR=./.kai-code/plugins   # plugin discovery directory";\
	} > $(SIM)/agent-kai-code/.env.docs
	@sed -n '/^[A-Z_]/ { s/ *  *#.*//; p; }' $(SIM)/agent-kai-code/.env.docs > $(SIM)/agent-kai-code/.env
	@echo "Generated .env and .env.docs for agent-kai-code"

env-agent-opencode:
	@mkdir -p $(SIM)/agent-opencode
	@{\
	  echo "# Agent - Environment Variables";\
	  echo "# Copy values below to .env and adjust as needed.";\
	  echo "";\
	  echo "ORCHESTRATOR_ADDR=localhost:50051    # orchestrator gRPC address";\
	  echo "AGENT_ID=local-coder-2               # unique agent identifier";\
	  echo "AGENT_ADDR=localhost:50053           # address announced to orchestrator";\
	  echo "AGENT_LISTEN=:50053                  # gRPC server bind address";\
	  echo "MISSION_TIMEOUT=5m                   # per-mission timeout";\
	  echo "KAI_PLUGIN_DIR=./.kai-code/plugins     # plugin discovery directory";\
	} > $(SIM)/agent-opencode/.env.docs
	@sed -n '/^[A-Z_]/ { s/ *  *#.*//; p; }' $(SIM)/agent-opencode/.env.docs > $(SIM)/agent-opencode/.env
	@echo "Generated .env and .env.docs for agent-opencode"

env-agent-claude:
	@mkdir -p $(SIM)/agent-claude
	@{\
	  echo "# Agent - Environment Variables";\
	  echo "# Copy values below to .env and adjust as needed.";\
	  echo "";\
	  echo "ORCHESTRATOR_ADDR=localhost:50051    # orchestrator gRPC address";\
	  echo "AGENT_ID=local-coder-3               # unique agent identifier";\
	  echo "AGENT_ADDR=localhost:50054           # address announced to orchestrator";\
	  echo "AGENT_LISTEN=:50054                  # gRPC server bind address";\
	  echo "MISSION_TIMEOUT=5m                   # per-mission timeout";\
	  echo "KAI_PLUGIN_DIR=./.kai-code/plugins     # plugin discovery directory";\
	} > $(SIM)/agent-claude/.env.docs
	@sed -n '/^[A-Z_]/ { s/ *  *#.*//; p; }' $(SIM)/agent-claude/.env.docs > $(SIM)/agent-claude/.env
	@echo "Generated .env and .env.docs for agent-claude"

# --- Go binaries ---

build-agent:
	@echo "=== Building agent ==="
	mkdir -p $(SIM)/agent-kai-code $(SIM)/agent-opencode $(SIM)/agent-claude
	cd $(ROOT)/kai-platform && go build -o $(SIM)/agent-kai-code/agent ./agent/cmd/agent
	cp $(SIM)/agent-kai-code/agent $(SIM)/agent-opencode/agent
	cp $(SIM)/agent-kai-code/agent $(SIM)/agent-claude/agent

build-orchestrator:
	@echo "=== Building orchestrator ==="
	mkdir -p $(SIM)/orchestrator
	cd $(ROOT)/kai-platform && go build -o $(SIM)/orchestrator/orchestrator ./orchestrator/cmd/server

build-config-service:
	@echo "=== Building config-service ==="
	mkdir -p $(SIM)/config-service
	cd $(ROOT)/kai-config-service/cmd/server && go build -o $(SIM)/config-service/config-service .

build-kaictl:
	@echo "=== Building kaictl ==="
	cd $(ROOT)/kai-cli-control && go build -o $(SIM)/kaictl .

# --- UI ---

build-ui:
	@echo "=== Building UI ==="
	mkdir -p $(SIM)/ui
	cd $(ROOT)/kai-platform-ui && npm run build && rm -rf $(SIM)/ui/dist && cp -r dist $(SIM)/ui/dist

# --- Kai C# agent ---

build-kai-code-cli:
	@echo "=== Building kai-code C# agent ==="
	mkdir -p $(SIM)/agent-kai-code/kai-code-cli
	dotnet publish -c Release -o $(SIM)/agent-kai-code/kai-code-cli $(ROOT)/kai-code/src/KaiCode.Cli/KaiCode.Cli.csproj 2>&1 | tail -3
	printf '#!/bin/bash\ndir="$$(dirname "$$0")"\nexec "$$dir/kai-code-cli/KaiCode.Cli" "$$@"\n' > $(SIM)/agent-kai-code/kai-code
	chmod +x $(SIM)/agent-kai-code/kai-code

# --- Plugins ---

build-plugins:
	@echo "=== Building plugins ==="
	$(MAKE) -C $(ROOT)/kai-plugins build

# --- Distribution ---

dist-orchestrator:
	@echo "=== Distributing orchestrator plugins ==="
	mkdir -p $(SIM)/orchestrator/.kai-code/plugins/conventional-commits
	cp $(ROOT)/kai-plugins/gate/conventional-commits-gate/plugin        $(SIM)/orchestrator/.kai-code/plugins/conventional-commits/
	cp $(ROOT)/kai-plugins/gate/conventional-commits-gate/plugin.json  $(SIM)/orchestrator/.kai-code/plugins/conventional-commits/
	mkdir -p $(SIM)/orchestrator/.kai-code/plugins/forgejo
	cp $(ROOT)/kai-plugins/gitprovider/forgejo-provider/forgejo-provider $(SIM)/orchestrator/.kai-code/plugins/forgejo/
	cp $(ROOT)/kai-plugins/gitprovider/forgejo-provider/plugin.json     $(SIM)/orchestrator/.kai-code/plugins/forgejo/
	mkdir -p $(SIM)/orchestrator/.kai-code/plugins/local-fs
	cp $(ROOT)/kai-plugins/archive/local-fs-plugin/local-fs-plugin $(SIM)/orchestrator/.kai-code/plugins/local-fs/
	cp $(ROOT)/kai-plugins/archive/local-fs-plugin/plugin.json     $(SIM)/orchestrator/.kai-code/plugins/local-fs/
	mkdir -p $(SIM)/orchestrator/.kai-code/plugins/env
	cp $(ROOT)/kai-plugins/secrets/env-plugin/env-plugin       $(SIM)/orchestrator/.kai-code/plugins/env/
	cp $(ROOT)/kai-plugins/secrets/env-plugin/plugin.json      $(SIM)/orchestrator/.kai-code/plugins/env/

dist-agent-kai-code:
	@echo "=== Distributing kai-code runner plugin ==="
	mkdir -p $(SIM)/agent-kai-code/.kai-code/plugins/kai-code
	cp $(ROOT)/kai-plugins/runner/kai-code-plugin/kai-code-plugin $(SIM)/agent-kai-code/.kai-code/plugins/kai-code/
	cp $(ROOT)/kai-plugins/runner/kai-code-plugin/plugin.json     $(SIM)/agent-kai-code/.kai-code/plugins/kai-code/

dist-agent-opencode:
	@echo "=== Distributing opencode runner plugin ==="
	mkdir -p $(SIM)/agent-opencode/.kai-code/plugins/opencode
	cp $(ROOT)/kai-plugins/runner/opencode-plugin/opencode-plugin $(SIM)/agent-opencode/.kai-code/plugins/opencode/
	cp $(ROOT)/kai-plugins/runner/opencode-plugin/plugin.json     $(SIM)/agent-opencode/.kai-code/plugins/opencode/

dist-agent-claude:
	@echo "=== Distributing claude runner plugin ==="
	mkdir -p $(SIM)/agent-claude/.kai-code/plugins/claude-code
	cp $(ROOT)/kai-plugins/runner/claude-plugin/claude-plugin $(SIM)/agent-claude/.kai-code/plugins/claude-code/
	cp $(ROOT)/kai-plugins/runner/claude-plugin/plugin.json   $(SIM)/agent-claude/.kai-code/plugins/claude-code/

# --- Clean ---

clean:
	rm -rf $(SIM)/orchestrator
	rm -rf $(SIM)/agent-kai-code
	rm -rf $(SIM)/agent-opencode
	rm -rf $(SIM)/agent-claude
	rm -rf $(SIM)/config-service
	rm -rf $(SIM)/ui
	rm -rf $(SIM)/tmp
	rm -f  $(SIM)/kaictl
	@echo "Cleaned simulation/"
