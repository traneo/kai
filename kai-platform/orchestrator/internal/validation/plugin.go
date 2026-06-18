package validation

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PluginManifest struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	APIVersion string            `json:"api_version"`
	Type       string            `json:"type"`
	GateTypes  []string          `json:"gate_types,omitempty"`
	Binary     string            `json:"binary"`
	Config     map[string]string `json:"config,omitempty"`
}

type PluginGate struct {
	gateType Type
	binary   string
	manifest PluginManifest
}

func NewPluginGate(manifest PluginManifest) *PluginGate {
	return &PluginGate{
		gateType: Type(manifest.Name),
		binary:   manifest.Binary,
		manifest: manifest,
	}
}

func (g *PluginGate) Name() Type {
	return g.gateType
}

func (g *PluginGate) Run(ctx *Context) *Result {
	if g.binary == "" {
		return &Result{
			Gate:    g.gateType,
			Status:  StatusSkipped,
			Message: fmt.Sprintf("gate plugin %q has no binary path configured", g.gateType),
		}
	}

	args := []string{}
	if ctx.RepoDir != "" {
		args = append(args, "--repo-dir", ctx.RepoDir)
	}
	if ctx.WorkspaceDir != "" {
		args = append(args, "--workspace-dir", ctx.WorkspaceDir)
	}
	args = append(args, "--branch", ctx.BranchName)
	args = append(args, "--exit-code", fmt.Sprintf("%d", ctx.ExitCode))

	for k, v := range g.manifest.Config {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, g.binary, args...)
	out, err := cmd.CombinedOutput()

	if err != nil {
		return &Result{
			Gate:    g.gateType,
			Status:  StatusFailed,
			Message: fmt.Sprintf("plugin %s: %s", g.gateType, strings.TrimSpace(string(out))),
		}
	}

	var pluginResult Result
	if err := json.Unmarshal(out, &pluginResult); err != nil {
		return &Result{
			Gate:    g.gateType,
			Status:  StatusPassed,
			Message: strings.TrimSpace(string(out)),
		}
	}

	pluginResult.Gate = g.gateType
	return &pluginResult
}

type PluginManager struct {
	plugins []PluginManifest
	dir     string
}

func NewPluginManager(pluginDir string) *PluginManager {
	return &PluginManager{dir: pluginDir}
}

func (m *PluginManager) Discover() error {
	if m.dir == "" {
		return nil
	}

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read plugin dir %s: %w", m.dir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(m.dir, entry.Name(), "plugin.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var manifest PluginManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}

		if manifest.Type != "gate" {
			continue
		}

		manifest.Binary = filepath.Join(m.dir, entry.Name(), manifest.Binary)
		m.plugins = append(m.plugins, manifest)
	}

	return nil
}

func (m *PluginManager) LoadGates() []Gate {
	var gates []Gate
	for _, p := range m.plugins {
		gates = append(gates, NewPluginGate(p))
	}
	return gates
}

func (m *PluginManager) Plugins() []PluginManifest {
	return m.plugins
}

func DefaultPluginDir() string {
	if dir := os.Getenv("KAI_PLUGIN_DIR"); dir != "" {
		return dir
	}
	exe, err := os.Executable()
	if err != nil {
		return filepath.Join(".kai", "plugins")
	}
	return filepath.Join(filepath.Dir(exe), ".kai", "plugins")
}
