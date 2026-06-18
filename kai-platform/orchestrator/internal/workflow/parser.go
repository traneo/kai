package workflow

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func ParsePipeline(path string) (*Pipeline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read pipeline file: %w", err)
	}

	var p Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse pipeline yaml: %w", err)
	}

	if len(p.Steps) == 0 {
		return nil, fmt.Errorf("pipeline must have at least one step")
	}

	for i, s := range p.Steps {
		if s.ID == "" {
			return nil, fmt.Errorf("step %d: missing id", i)
		}
		if s.Prompt == "" {
			return nil, fmt.Errorf("step %q: missing prompt", s.ID)
		}
	}

	return &p, nil
}

func ParsePipelineBytes(data []byte) (*Pipeline, error) {
	var p Pipeline
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse pipeline yaml: %w", err)
	}

	if len(p.Steps) == 0 {
		return nil, fmt.Errorf("pipeline must have at least one step")
	}

	for i, s := range p.Steps {
		if s.ID == "" {
			return nil, fmt.Errorf("step %d: missing id", i)
		}
		if s.Prompt == "" {
			return nil, fmt.Errorf("step %q: missing prompt", s.ID)
		}
	}

	return &p, nil
}
