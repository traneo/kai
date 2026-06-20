package config

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type LLMConfig struct {
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
	APIKey   string `json:"api_key"`
}

type PlanBuilderConfig struct {
	LLM LLMConfig `json:"llm"`
}

type Config struct {
	PlanBuilder PlanBuilderConfig `json:"plan_builder"`
}

func Fetch(configServiceURL string) (*Config, error) {
	if configServiceURL == "" {
		return nil, fmt.Errorf("CONFIG_SERVICE_URL is required")
	}

	resp, err := http.Get(configServiceURL + "/api/v1/config")
	if err != nil {
		return nil, fmt.Errorf("fetch config from %s: %w", configServiceURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("config service returned status %d", resp.StatusCode)
	}

	var cfg Config
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config response: %w", err)
	}

	if cfg.PlanBuilder.LLM.Endpoint == "" {
		return nil, fmt.Errorf("config missing plan_builder.llm.endpoint")
	}
	if cfg.PlanBuilder.LLM.Model == "" {
		return nil, fmt.Errorf("config missing plan_builder.llm.model")
	}
	return &cfg, nil
}
