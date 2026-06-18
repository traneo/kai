package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
	Binary     string            `json:"binary"`
	Config     map[string]string `json:"config,omitempty"`
}

type RunDefinition struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

type PluginRunner struct {
	baseRunner
	name     string
	binary   string
	manifest PluginManifest
	def      *RunDefinition
}

func NewPluginRunner(name, binary string, cfg Config) *PluginRunner {
	return &PluginRunner{
		baseRunner: baseRunner{cfg: cfg},
		name:       name,
		binary:     binary,
	}
}

func (r *PluginRunner) WriteConfig(input WriteConfigInput) error {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("plugin %q: marshal write-config input: %w", r.name, err)
	}

	cmd := exec.Command(r.binary, "write-config")
	cmd.Stdin = strings.NewReader(string(inputJSON))
	cmd.Env = append(os.Environ(),
		"KAI_WORKDIR="+r.cfg.WorkDir,
		"KAI_REPODIR="+r.cfg.RepoDir,
		"KAI_AGENT_DIR="+r.cfg.AgentDir,
	)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return fmt.Errorf("plugin %q write-config: %w\n%s", r.name, err, exitErr.Stderr)
		}
		return fmt.Errorf("plugin %q write-config: %w", r.name, err)
	}

	var def RunDefinition
	if err := json.Unmarshal(out, &def); err != nil {
		return fmt.Errorf("plugin %q write-config: invalid RunDefinition: %w\nraw: %s", r.name, err, string(out))
	}
	r.def = &def
	return nil
}

func (r *PluginRunner) Run(ctx context.Context, prompt string, onLine LineHandler) (*Result, error) {
	if r.def == nil {
		return nil, fmt.Errorf("plugin %q: WriteConfig must be called before Run", r.name)
	}

	if len(r.def.Env) > 0 {
		env := os.Environ()
		for k, v := range r.def.Env {
			env = append(env, k+"="+v)
		}
		r.baseRunner.env = env
	}

	command := resolveString(r.def.Command, prompt, r.cfg)
	args := resolvePlaceholders(r.def.Args, prompt, r.cfg)
	return r.baseRunner.Run(ctx, command, args, prompt, onLine)
}

func resolveString(s, prompt string, cfg Config) string {
	s = strings.ReplaceAll(s, "{PROMPT}", prompt)
	s = strings.ReplaceAll(s, "{WORKDIR}", cfg.WorkDir)
	s = strings.ReplaceAll(s, "{REPODIR}", cfg.RepoDir)
	s = strings.ReplaceAll(s, "{AGENTDIR}", cfg.AgentDir)
	return s
}

func resolvePlaceholders(args []string, prompt string, cfg Config) []string {
	resolved := make([]string, len(args))
	for i, arg := range args {
		resolved[i] = resolveString(arg, prompt, cfg)
	}
	return resolved
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

type RunnerPluginManager struct {
	plugins []PluginManifest
	dir     string
}

func NewRunnerPluginManager(pluginDir string) *RunnerPluginManager {
	return &RunnerPluginManager{dir: pluginDir}
}

func (m *RunnerPluginManager) Discover() error {
	if m.dir == "" {
		return nil
	}

	entries, err := os.ReadDir(m.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read runner plugin dir %s: %w", m.dir, err)
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

		if manifest.Type != "runner" {
			continue
		}

		manifest.Binary = filepath.Join(m.dir, entry.Name(), manifest.Binary)
		m.plugins = append(m.plugins, manifest)
		log.Printf("runner: discovered plugin %q (%s)", manifest.Name, manifest.Binary)
	}

	return nil
}

func (m *RunnerPluginManager) LoadRunners() {
	for _, p := range m.plugins {
		name := p.Name
		binary := p.Binary
		Register(name, func(cfg Config) Runner {
			return NewPluginRunner(name, binary, cfg)
		})
		log.Printf("runner: registered plugin %q", name)
	}
}

func DiscoverAndLoadPlugins(pluginDir string) {
	mgr := NewRunnerPluginManager(pluginDir)
	if err := mgr.Discover(); err != nil {
		log.Printf("runner plugin discovery: %v", err)
	}
	mgr.LoadRunners()
}
