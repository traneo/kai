package gitprovider

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

type PluginManifest struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	APIVersion string            `json:"api_version"`
	Type       string            `json:"type"`
	Binary     string            `json:"binary"`
	Config     map[string]string `json:"config,omitempty"`
}

type PluginProvider struct {
	manifest PluginManifest
}

func NewPluginProvider(manifest PluginManifest) *PluginProvider {
	return &PluginProvider{manifest: manifest}
}

func (p *PluginProvider) Name() string {
	return p.manifest.Name
}

func (p *PluginProvider) CreatePR(ctx context.Context, req CreatePRRequest) (*CreatePRResult, error) {
	if p.manifest.Binary == "" {
		return nil, fmt.Errorf("gitprovider plugin %q has no binary path", p.manifest.Name)
	}

	args := []string{
		"create-pr",
		"--repo-url", req.RepoURL,
		"--owner", req.Owner,
		"--repo", req.Repo,
		"--base", req.BaseBranch,
		"--head", req.HeadBranch,
		"--title", req.Title,
		"--body", req.Body,
	}

	if req.Token != "" {
		args = append(args, "--token", req.Token)
	}

	for k, v := range p.manifest.Config {
		args = append(args, fmt.Sprintf("--%s=%s", k, v))
	}

	cmd := exec.CommandContext(ctx, p.manifest.Binary, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gitprovider plugin %q: %w\n%s", p.manifest.Name, err, out)
	}

	var result CreatePRResult
	if err := json.Unmarshal(out, &result); err != nil {
		return nil, fmt.Errorf("gitprovider plugin %q: invalid output: %w\n%s", p.manifest.Name, err, out)
	}

	return &result, nil
}
