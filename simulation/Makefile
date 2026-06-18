ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/..)
SIM  := $(ROOT)/simulation

.PHONY: build clean start-dev docker-build env-files \
        env-orchestrator env-config-service env-agent-kai env-agent-opencode env-agent-claude \
        build-agent build-orchestrator build-config-service build-kaictl build-ui build-kai-cli \
        build-plugins \
        dist-agent-kai dist-agent-opencode dist-agent-claude dist-orchestrator

build: build-agent build-orchestrator build-config-service build-kaictl build-ui build-kai-cli build-plugins dist-orchestrator dist-agent-kai dist-agent-opencode dist-agent-claude

start-dev: build
	./start-dev.sh

docker-build: build env-files
	cd docker && docker compose build

# --- Environment variable reference files ---

env-files: env-orchestrator env-config-service env-agent-kai env-agent-opencode env-agent-claude

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
	  echo "KAI_PLUGIN_DIR=./.kai/plugins        # plugin discovery directory";\
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

env-agent-kai:
	@mkdir -p $(SIM)/agent-kai
	@{\
	  echo "# Agent - Environment Variables";\
	  echo "# Copy values below to .env and adjust as needed.";\
	  echo "";\
	  echo "ORCHESTRATOR_ADDR=localhost:50051    # orchestrator gRPC address";\
	  echo "AGENT_ID=local-coder-1               # unique agent identifier";\
	  echo "AGENT_ADDR=localhost:50052           # address announced to orchestrator";\
	  echo "AGENT_LISTEN=:50052                  # gRPC server bind address";\
	  echo "MISSION_TIMEOUT=5m                   # per-mission timeout";\
	  echo "KAI_PLUGIN_DIR=./.kai/plugins        # plugin discovery directory";\
	} > $(SIM)/agent-kai/.env.docs
	@sed -n '/^[A-Z_]/ { s/ *  *#.*//; p; }' $(SIM)/agent-kai/.env.docs > $(SIM)/agent-kai/.env
	@echo "Generated .env and .env.docs for agent-kai"

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
	  echo "KAI_PLUGIN_DIR=./.kai/plugins        # plugin discovery directory";\
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
	  echo "KAI_PLUGIN_DIR=./.kai/plugins        # plugin discovery directory";\
	} > $(SIM)/agent-claude/.env.docs
	@sed -n '/^[A-Z_]/ { s/ *  *#.*//; p; }' $(SIM)/agent-claude/.env.docs > $(SIM)/agent-claude/.env
	@echo "Generated .env and .env.docs for agent-claude"

# --- Go binaries ---

build-agent:
	@echo "=== Building agent ==="
	mkdir -p $(SIM)/agent-kai $(SIM)/agent-opencode $(SIM)/agent-claude
	cd $(ROOT)/kai-platform && go build -o $(SIM)/agent-kai/agent ./agent/cmd/agent
	cp $(SIM)/agent-kai/agent $(SIM)/agent-opencode/agent
	cp $(SIM)/agent-kai/agent $(SIM)/agent-claude/agent

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

build-kai-cli:
	@echo "=== Building kai C# agent ==="
	mkdir -p $(SIM)/agent-kai/kai-cli
	dotnet publish -c Release -o $(SIM)/agent-kai/kai-cli $(ROOT)/kai/src/Kai.Cli/Kai.Cli.csproj 2>&1 | tail -3
	printf '#!/bin/bash\ndir="$$(dirname "$$0")"\nexec "$$dir/kai-cli/Kai.Cli" "$$@"\n' > $(SIM)/agent-kai/kai
	chmod +x $(SIM)/agent-kai/kai

# --- Plugins ---

build-plugins:
	@echo "=== Building plugins ==="
	$(MAKE) -C $(ROOT)/kai-plugins build

# --- Distribution ---

dist-orchestrator:
	@echo "=== Distributing orchestrator plugins ==="
	mkdir -p $(SIM)/orchestrator/.kai/plugins/conventional-commits
	cp $(ROOT)/kai-plugins/gate/conventional-commits-gate/plugin        $(SIM)/orchestrator/.kai/plugins/conventional-commits/
	cp $(ROOT)/kai-plugins/gate/conventional-commits-gate/plugin.json  $(SIM)/orchestrator/.kai/plugins/conventional-commits/
	mkdir -p $(SIM)/orchestrator/.kai/plugins/forgejo
	cp $(ROOT)/kai-plugins/gitprovider/forgejo-provider/forgejo-provider $(SIM)/orchestrator/.kai/plugins/forgejo/
	cp $(ROOT)/kai-plugins/gitprovider/forgejo-provider/plugin.json     $(SIM)/orchestrator/.kai/plugins/forgejo/
	mkdir -p $(SIM)/orchestrator/.kai/plugins/local-fs
	cp $(ROOT)/kai-plugins/archive/local-fs-plugin/local-fs-plugin $(SIM)/orchestrator/.kai/plugins/local-fs/
	cp $(ROOT)/kai-plugins/archive/local-fs-plugin/plugin.json     $(SIM)/orchestrator/.kai/plugins/local-fs/
	mkdir -p $(SIM)/orchestrator/.kai/plugins/env
	cp $(ROOT)/kai-plugins/secrets/env-plugin/env-plugin       $(SIM)/orchestrator/.kai/plugins/env/
	cp $(ROOT)/kai-plugins/secrets/env-plugin/plugin.json      $(SIM)/orchestrator/.kai/plugins/env/

dist-agent-kai:
	@echo "=== Distributing kai runner plugin ==="
	mkdir -p $(SIM)/agent-kai/.kai/plugins/kai
	cp $(ROOT)/kai-plugins/runner/kai-plugin/kai-plugin   $(SIM)/agent-kai/.kai/plugins/kai/
	cp $(ROOT)/kai-plugins/runner/kai-plugin/plugin.json  $(SIM)/agent-kai/.kai/plugins/kai/

dist-agent-opencode:
	@echo "=== Distributing opencode runner plugin ==="
	mkdir -p $(SIM)/agent-opencode/.kai/plugins/opencode
	cp $(ROOT)/kai-plugins/runner/opencode-plugin/opencode-plugin $(SIM)/agent-opencode/.kai/plugins/opencode/
	cp $(ROOT)/kai-plugins/runner/opencode-plugin/plugin.json     $(SIM)/agent-opencode/.kai/plugins/opencode/

dist-agent-claude:
	@echo "=== Distributing claude runner plugin ==="
	mkdir -p $(SIM)/agent-claude/.kai/plugins/claude-code
	cp $(ROOT)/kai-plugins/runner/claude-plugin/claude-plugin $(SIM)/agent-claude/.kai/plugins/claude-code/
	cp $(ROOT)/kai-plugins/runner/claude-plugin/plugin.json   $(SIM)/agent-claude/.kai/plugins/claude-code/

# --- Clean ---

clean:
	rm -rf $(SIM)/orchestrator
	rm -rf $(SIM)/agent-kai
	rm -rf $(SIM)/agent-opencode
	rm -rf $(SIM)/agent-claude
	rm -rf $(SIM)/config-service
	rm -rf $(SIM)/ui
	rm -rf $(SIM)/tmp
	rm -f  $(SIM)/kaictl
	@echo "Cleaned simulation/"
