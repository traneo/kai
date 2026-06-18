package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type writeConfigInput struct {
	ConfigBlob string  `json:"config_blob"`
	Policy     *policy `json:"policy,omitempty"`
}

type policy struct {
	AllowedTools    []string `json:"allowed_tools"`
	AllowedCommands []string `json:"allowed_commands"`
	AllowedDirs     []string `json:"allowed_dirs"`
}

type opencodeConfig struct {
	Runner string          `json:"runner"`
	Data   json.RawMessage `json:"data"`
}

type runDefinition struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "write-config" {
		fmt.Fprintf(os.Stderr, "usage: %s write-config\n", os.Args[0])
		os.Exit(1)
	}

	workDir := os.Getenv("KAI_WORKDIR")
	if workDir == "" {
		fmt.Fprintf(os.Stderr, "KAI_WORKDIR not set\n")
		os.Exit(1)
	}

	var input writeConfigInput
	if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
		fmt.Fprintf(os.Stderr, "decode input: %v\n", err)
		os.Exit(1)
	}

	var blob opencodeConfig
	if err := json.Unmarshal([]byte(input.ConfigBlob), &blob); err != nil {
		fmt.Fprintf(os.Stderr, "parse config blob: %v\n", err)
		os.Exit(1)
	}

	if len(blob.Data) > 0 {
		configPath := filepath.Join(workDir, "opencode.jsonc")
		if err := os.WriteFile(configPath, blob.Data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write opencode.jsonc: %v\n", err)
			os.Exit(1)
		}
	}

	json.NewEncoder(os.Stdout).Encode(runDefinition{
		Command: "opencode",
		Args: []string{
			"run", "{PROMPT}",
			"--dir", "{REPODIR}",
			"--format", "json",
			"--dangerously-skip-permissions",
		},
	})
}
