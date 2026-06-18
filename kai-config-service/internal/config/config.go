package config

import "encoding/json"

// ConfigVersion stores a versioned opaque config blob.
type ConfigVersion struct {
	ID        string          `json:"id"`
	Version   int             `json:"version"`
	Status    string          `json:"status"`
	Config    json.RawMessage `json:"config"`
	Message   string          `json:"message"`
	CreatedBy string          `json:"created_by"`
	CreatedAt string          `json:"created_at"`
	UpdatedAt string          `json:"updated_at"`
}

type ReloadResult struct {
	Status          string   `json:"status"`
	AppliedAt       string   `json:"applied_at"`
	HotReloaded     []string `json:"hot_reloaded"`
	RequiresRestart []string `json:"requires_restart"`
	Errors          []string `json:"errors,omitempty"`
}
