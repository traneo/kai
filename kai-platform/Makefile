.PHONY: build build-orchestrator build-agent proto docker-up docker-down dev

proto:
	protoc --proto_path=proto --go_out=gen --go_opt=paths=source_relative \
		--go-grpc_out=gen --go-grpc_opt=paths=source_relative \
		kaiplatform/v1/types.proto \
		kaiplatform/v1/orchestrator.proto \
		kaiplatform/v1/agent.proto

build-orchestrator:
	go build -o bin/orchestrator ./orchestrator/cmd/server

build-agent:
	go build -o bin/agent ./agent/cmd/agent

build: build-orchestrator build-agent build-plugins

build-plugins:
	$(MAKE) -C ../kai-plugins build
	mkdir -p bin/plugins
	cp ../kai-plugins/runner/kai-plugin/kai-plugin        bin/plugins/kai-plugin
	cp ../kai-plugins/runner/opencode-plugin/opencode-plugin bin/plugins/opencode-plugin
	cp ../kai-plugins/runner/claude-plugin/claude-plugin     bin/plugins/claude-plugin

install-plugins: build-plugins
	@mkdir -p $(HOME)/.kai/plugins/kai $(HOME)/.kai/plugins/opencode $(HOME)/.kai/plugins/claude
	cp ../kai-plugins/runner/kai-plugin/kai-plugin      $(HOME)/.kai/plugins/kai/
	cp ../kai-plugins/runner/opencode-plugin/opencode-plugin $(HOME)/.kai/plugins/opencode/
	cp ../kai-plugins/runner/claude-plugin/claude-plugin    $(HOME)/.kai/plugins/claude/
	cp ../kai-plugins/runner/kai-plugin/plugin.json        $(HOME)/.kai/plugins/kai/plugin.json
	cp ../kai-plugins/runner/opencode-plugin/plugin.json   $(HOME)/.kai/plugins/opencode/plugin.json
	cp ../kai-plugins/runner/claude-plugin/plugin.json     $(HOME)/.kai/plugins/claude/plugin.json

dev:
	docker compose up --build

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

clean:
	rm -rf bin/
