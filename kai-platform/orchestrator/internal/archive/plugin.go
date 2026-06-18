package archive

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

		if manifest.Type != "archive" && manifest.Type != "storage" {
			continue
		}

		manifest.Binary = filepath.Join(l.dir, entry.Name(), manifest.Binary)
		l.plugins = append(l.plugins, manifest)
	}

	return nil
}

func (l *PluginLoader) LoadStores(ctx context.Context) []Store {
	var stores []Store
	for _, p := range l.plugins {
		s := &PluginStore{manifest: p}
		stores = append(stores, s)
		log.Printf("archive: loaded plugin backend %q (%s)", p.Backend, p.Binary)
	}
	return stores
}

func (l *PluginLoader) Plugins() []PluginManifest {
	return l.plugins
}

type PluginStore struct {
	manifest PluginManifest
}

func (p *PluginStore) Save(ctx context.Context, runID, stepID string, data []byte) error {
	args := []string{"save", "--run-id", runID, "--step-id", stepID}
	for k, v := range p.manifest.Config {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, p.manifest.Binary, args...)
	cmd.Stdin = nil
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("archive plugin %q: stdin pipe: %w", p.manifest.Backend, err)
	}
	go func() {
		defer stdin.Close()
		stdin.Write(data)
	}()

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("archive plugin %q: %w", p.manifest.Backend, err)
	}

	log.Printf("archive plugin %q saved %s/%s.zip: %s", p.manifest.Backend, runID, stepID, string(out))
	return nil
}

func (p *PluginStore) Get(ctx context.Context, runID, stepID string) ([]byte, error) {
	args := []string{"get", "--run-id", runID, "--step-id", stepID}
	for k, v := range p.manifest.Config {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, p.manifest.Binary, args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("archive plugin %q: %w", p.manifest.Backend, err)
	}

	return out, nil
}

func (p *PluginStore) Close() error {
	return nil
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
