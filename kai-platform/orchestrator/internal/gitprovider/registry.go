package gitprovider

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Registry struct {
	dir       string
	builtins  map[string]Provider
	plugins   []PluginManifest
}

func NewRegistry(pluginDir string) *Registry {
	return &Registry{
		dir:      pluginDir,
		builtins: make(map[string]Provider),
	}
}

func (r *Registry) Register(p Provider) {
	r.builtins[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, bool) {
	if p, ok := r.builtins[name]; ok {
		return p, true
	}
	for _, m := range r.plugins {
		if m.Name == name {
			return NewPluginProvider(m), true
		}
	}
	return nil, false
}

func (r *Registry) Detect(repoURL string) string {
	lower := strings.ToLower(repoURL)
	switch {
	case strings.Contains(lower, "github.com"):
		return "github"
	case strings.Contains(lower, "gitlab.com") || strings.Contains(lower, "gitlab."):
		return "gitlab"
	case strings.Contains(lower, "bitbucket.org"):
		return "bitbucket"
	default:
		return ""
	}
}

func (r *Registry) RegisterBuiltins() {
	r.Register(NewGitHub())
	r.Register(NewGitLab())
	r.Register(NewBitbucket())
}

func (r *Registry) Discover() error {
	if r.dir == "" {
		return nil
	}

	entries, err := os.ReadDir(r.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read gitprovider dir %s: %w", r.dir, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifestPath := filepath.Join(r.dir, entry.Name(), "plugin.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}

		var manifest PluginManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}

		if manifest.Type != "gitprovider" {
			continue
		}

		manifest.Binary = filepath.Join(r.dir, entry.Name(), manifest.Binary)
		r.plugins = append(r.plugins, manifest)
		log.Printf("gitprovider: discovered plugin %q (%s)", manifest.Name, manifest.Binary)
	}

	return nil
}

func (r *Registry) Plugins() []PluginManifest {
	return r.plugins
}
