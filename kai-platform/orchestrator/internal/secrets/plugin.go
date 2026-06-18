package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

type PluginManifest struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	APIVersion string            `json:"api_version"`
	Type       string            `json:"type"`
	Backend    string            `json:"backend"`
	Binary     string            `json:"binary"`
	Config     map[string]string `json:"config,omitempty"`
}

type PluginLoader struct {
	plugins []PluginManifest
	dir     string
}

func NewPluginLoader(pluginDir string) *PluginLoader {
	return &PluginLoader{dir: pluginDir}
}

func (l *PluginLoader) Discover() error {
	if l.dir == "" {
		return nil
	}

	entries, err := os.ReadDir(l.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read plugin dir %s: %w", l.dir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(l.dir, entry.Name(), "plugin.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var manifest PluginManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}

		if manifest.Type != "secrets" {
			continue
		}

		manifest.Binary = filepath.Join(l.dir, entry.Name(), manifest.Binary)
		l.plugins = append(l.plugins, manifest)
	}

	return nil
}

func (l *PluginLoader) LoadManagers(ctx context.Context) []Manager {
	var managers []Manager
	for _, p := range l.plugins {
		mgr := &PluginSecretManager{manifest: p}
		managers = append(managers, mgr)
		log.Printf("secrets: loaded plugin backend %q (%s)", p.Backend, p.Binary)
	}
	return managers
}

func (l *PluginLoader) Plugins() []PluginManifest {
	return l.plugins
}

type PluginSecretManager struct {
	manifest PluginManifest
}

func (p *PluginSecretManager) GetSecret(ctx context.Context, path, key string) (string, error) {
	args := []string{"get", "--path", path, "--key", key}
	for k, v := range p.manifest.Config {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, p.manifest.Binary, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("secret plugin %q: %w", p.manifest.Backend, err)
	}

	return string(out), nil
}

func (p *PluginSecretManager) GetSecrets(ctx context.Context, path string) (map[string]string, error) {
	args := []string{"list", "--path", path}
	for k, v := range p.manifest.Config {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, p.manifest.Binary, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("secret plugin %q: %w", p.manifest.Backend, err)
	}

	var result map[string]string
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("secret plugin %q: invalid output: %w", p.manifest.Backend, err)
	}

	return result, nil
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
