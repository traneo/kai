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

type kaiRunnerData struct {
	Language         string                    `json:"language,omitempty"`
	BranchPrefix     string                    `json:"branchPrefix,omitempty"`
	AutoCommit       *bool                     `json:"autoCommit,omitempty"`
	AutoPush         *bool                     `json:"autoPush,omitempty"`
	Rules            []string                  `json:"rules,omitempty"`
	MaxContextTokens int                       `json:"maxContextTokens,omitempty"`
	Limits           *kaiLimitsConfig          `json:"limits,omitempty"`
	Agents           map[string]*kaiAgentConfig `json:"agents,omitempty"`
}

type kaiConfig struct {
	Language         string                    `json:"language,omitempty"`
	BranchPrefix     string                    `json:"branchPrefix,omitempty"`
	AutoCommit       *bool                     `json:"autoCommit,omitempty"`
	AutoPush         *bool                     `json:"autoPush,omitempty"`
	Rules            []string                  `json:"rules,omitempty"`
	MaxContextTokens int                       `json:"maxContextTokens,omitempty"`
	Limits           *kaiLimitsConfig          `json:"limits,omitempty"`
	Agents           map[string]*kaiAgentConfig `json:"agents,omitempty"`
}

type kaiAgentConfig struct {
	Endpoint    string   `json:"endpoint,omitempty"`
	Model       string   `json:"model,omitempty"`
	Provider    string   `json:"provider,omitempty"`
	ApiKey      string   `json:"apiKey,omitempty"`
	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"topP,omitempty"`
	TopK        *int     `json:"topK,omitempty"`
}

type kaiLimitsConfig struct {
	AgentLoop *kaiAgentLoopLimits `json:"agentLoop,omitempty"`
	Retries   *kaiRetryLimits     `json:"retries,omitempty"`
	Output    *kaiOutputLimits    `json:"output,omitempty"`
	Llm       *kaiLlmLimits       `json:"llm,omitempty"`
	Display   *kaiDisplayLimits   `json:"display,omitempty"`
	Memory    *kaiMemoryLimits    `json:"memory,omitempty"`
}

type kaiAgentLoopLimits struct {
	MaxIterations       int `json:"maxIterations,omitempty"`
	MaxToolPairs        int `json:"maxToolPairs,omitempty"`
	CompressThreshold   int `json:"compressThreshold,omitempty"`
	KeepLastPairs       int `json:"keepLastPairs,omitempty"`
	ReadFileOutputChars int `json:"readFileOutputChars,omitempty"`
	ToolOutputChars     int `json:"toolOutputChars,omitempty"`
}

type kaiRetryLimits struct {
	TestFixAttempts      int   `json:"testFixAttempts,omitempty"`
	ReviewFixAttempts    int   `json:"reviewFixAttempts,omitempty"`
	LlmApiRetries        int   `json:"llmApiRetries,omitempty"`
	LlmRetryDelaySeconds []int `json:"llmRetryDelaySeconds,omitempty"`
	GateTimeoutMinutes   int   `json:"gateTimeoutMinutes,omitempty"`
}

type kaiOutputLimits struct {
	SearchResults       int `json:"searchResults,omitempty"`
	SearchFileSizeBytes int `json:"searchFileSizeBytes,omitempty"`
	FilePathMaxChars    int `json:"filePathMaxChars,omitempty"`
	TestOutputChars     int `json:"testOutputChars,omitempty"`
	GoalSummaryChars    int `json:"goalSummaryChars,omitempty"`
	KeyFilesCount       int `json:"keyFilesCount,omitempty"`
	DependenciesCount   int `json:"dependenciesCount,omitempty"`
	RelatedFilesCount   int `json:"relatedFilesCount,omitempty"`
	PreviewLines        int `json:"previewLines,omitempty"`
	SourceFilesCount    int `json:"sourceFilesCount,omitempty"`
	ConventionSamples   int `json:"conventionSamples,omitempty"`
	RecentGoalsCount    int `json:"recentGoalsCount,omitempty"`
}

type kaiLlmLimits struct {
	MaxTokens int `json:"maxTokens,omitempty"`
}

type kaiDisplayLimits struct {
	LogChars             int `json:"logChars,omitempty"`
	EventToolArgsChars   int `json:"eventToolArgsChars,omitempty"`
	EventOutputChars     int `json:"eventOutputChars,omitempty"`
	EventMessageChars    int `json:"eventMessageChars,omitempty"`
	SummaryToolsCount    int `json:"summaryToolsCount,omitempty"`
	SummaryToolLineChars int `json:"summaryToolLineChars,omitempty"`
}

type kaiMemoryLimits struct {
	MaxTaskHistoryEntries int `json:"maxTaskHistoryEntries,omitempty"`
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

	var kaiData kaiRunnerData
	if len(blob.Data) > 0 {
		if err := json.Unmarshal(blob.Data, &kaiData); err != nil {
			fmt.Fprintf(os.Stderr, "parse kai config: %v\n", err)
			os.Exit(1)
		}
	}

	cfg := kaiConfig{
		Language:         kaiData.Language,
		BranchPrefix:     kaiData.BranchPrefix,
		Rules:            kaiData.Rules,
		MaxContextTokens: kaiData.MaxContextTokens,
		Limits:           kaiData.Limits,
		Agents:           kaiData.Agents,
	}
	if kaiData.AutoCommit != nil && *kaiData.AutoCommit {
		cfg.AutoCommit = kaiData.AutoCommit
	}
	if kaiData.AutoPush != nil && *kaiData.AutoPush {
		cfg.AutoPush = kaiData.AutoPush
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal kai config: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(filepath.Join(workDir, "kai.json"), data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write kai.json: %v\n", err)
		os.Exit(1)
	}

	if input.Policy != nil {
		policyData, err := json.MarshalIndent(input.Policy, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "marshal policy: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(filepath.Join(workDir, "policy.json"), policyData, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "write policy.json: %v\n", err)
			os.Exit(1)
		}
	}

	json.NewEncoder(os.Stdout).Encode(runDefinition{
		Command: "{AGENTDIR}/kai",
		Args: []string{
			"run", "--json", "{PROMPT}",
			"--config", "{WORKDIR}/kai.json",
			"--policy", "{WORKDIR}/policy.json",
		},
	})
}
