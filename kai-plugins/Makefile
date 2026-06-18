.PHONY: all build install clean \
        build-runner-kai build-runner-opencode build-runner-claude \
        build-gate-commits build-gitprovider-forgejo \
        build-archive-local-fs build-secrets-env

PLUGIN_DIR := $(HOME)/.kai/plugins

all: build

build: build-runner-kai build-runner-opencode build-runner-claude \
       build-gate-commits build-gitprovider-forgejo \
       build-archive-local-fs build-secrets-env

# --- runner plugins ---

build-runner-kai:
	cd runner/kai-plugin && go build -o kai-plugin .

build-runner-opencode:
	cd runner/opencode-plugin && go build -o opencode-plugin .

build-runner-claude:
	cd runner/claude-plugin && go build -o claude-plugin .

# --- gate plugins ---

build-gate-commits:
	cd gate/conventional-commits-gate && go build -o plugin .

# --- gitprovider plugins ---

build-gitprovider-forgejo:
	cd gitprovider/forgejo-provider && go build -o forgejo-provider .

# --- archive plugins ---

build-archive-local-fs:
	cd archive/local-fs-plugin && go build -o local-fs-plugin .

# --- secrets plugins ---

build-secrets-env:
	cd secrets/env-plugin && go build -o env-plugin .

install: build
	@mkdir -p $(PLUGIN_DIR)/kai
	cp runner/kai-plugin/kai-plugin $(PLUGIN_DIR)/kai/
	cp runner/kai-plugin/plugin.json $(PLUGIN_DIR)/kai/
	@echo "Installed kai-plugin to $(PLUGIN_DIR)/kai/"

	@mkdir -p $(PLUGIN_DIR)/opencode
	cp runner/opencode-plugin/opencode-plugin $(PLUGIN_DIR)/opencode/
	cp runner/opencode-plugin/plugin.json $(PLUGIN_DIR)/opencode/
	@echo "Installed opencode-plugin to $(PLUGIN_DIR)/opencode/"

	@mkdir -p $(PLUGIN_DIR)/claude-code
	cp runner/claude-plugin/claude-plugin $(PLUGIN_DIR)/claude-code/
	cp runner/claude-plugin/plugin.json $(PLUGIN_DIR)/claude-code/
	@echo "Installed claude-plugin to $(PLUGIN_DIR)/claude-code/"

	@mkdir -p $(PLUGIN_DIR)/conventional-commits
	cp gate/conventional-commits-gate/plugin $(PLUGIN_DIR)/conventional-commits/
	cp gate/conventional-commits-gate/plugin.json $(PLUGIN_DIR)/conventional-commits/
	@echo "Installed conventional-commits-gate to $(PLUGIN_DIR)/conventional-commits/"

	@mkdir -p $(PLUGIN_DIR)/forgejo
	cp gitprovider/forgejo-provider/forgejo-provider $(PLUGIN_DIR)/forgejo/
	cp gitprovider/forgejo-provider/plugin.json $(PLUGIN_DIR)/forgejo/
	@echo "Installed forgejo-provider to $(PLUGIN_DIR)/forgejo/"

	@mkdir -p $(PLUGIN_DIR)/local-fs
	cp archive/local-fs-plugin/local-fs-plugin $(PLUGIN_DIR)/local-fs/
	cp archive/local-fs-plugin/plugin.json $(PLUGIN_DIR)/local-fs/
	@echo "Installed local-fs-plugin to $(PLUGIN_DIR)/local-fs/"

	@mkdir -p $(PLUGIN_DIR)/env
	cp secrets/env-plugin/env-plugin $(PLUGIN_DIR)/env/
	cp secrets/env-plugin/plugin.json $(PLUGIN_DIR)/env/
	@echo "Installed env-plugin to $(PLUGIN_DIR)/env/"

clean:
	rm -f runner/kai-plugin/kai-plugin
	rm -f runner/opencode-plugin/opencode-plugin
	rm -f runner/claude-plugin/claude-plugin
	rm -f gate/conventional-commits-gate/plugin
	rm -f gitprovider/forgejo-provider/forgejo-provider
	rm -f archive/local-fs-plugin/local-fs-plugin
	rm -f secrets/env-plugin/env-plugin
	rm -rf $(PLUGIN_DIR)/kai
	rm -rf $(PLUGIN_DIR)/opencode
	rm -rf $(PLUGIN_DIR)/claude-code
	rm -rf $(PLUGIN_DIR)/conventional-commits
	rm -rf $(PLUGIN_DIR)/forgejo
	rm -rf $(PLUGIN_DIR)/local-fs
	rm -rf $(PLUGIN_DIR)/env
