package main

import (
	"encoding/json"
	"fmt"
	"os"
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

	var data struct {
		ApiURL string `json:"apiUrl"`
		Model  string `json:"model"`
		ApiKey string `json:"apiKey"`
	}
	if len(blob.Data) > 0 {
		if err := json.Unmarshal(blob.Data, &data); err != nil {
			fmt.Fprintf(os.Stderr, "parse claude config: %v\n", err)
			os.Exit(1)
		}
	}

	def := runDefinition{
		Command: "claude",
		Args:    []string{"-p", "{PROMPT}"},
		Env:     make(map[string]string),
	}
	if data.Model != "" {
		def.Args = append(def.Args, "--model", data.Model)
	}
	def.Args = append(def.Args, "--allow-dangerously-skip-permissions", "--print")

	if data.ApiURL != "" {
		def.Env["ANTHROPIC_BASE_URL"] = data.ApiURL
	}
	if data.ApiKey != "" {
		def.Env["ANTHROPIC_API_KEY"] = data.ApiKey
	}

	json.NewEncoder(os.Stdout).Encode(def)
}
