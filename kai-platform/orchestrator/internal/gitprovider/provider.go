package gitprovider

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Provider interface {
	Name() string
	CreatePR(ctx context.Context, req CreatePRRequest) (*CreatePRResult, error)
}

type CreatePRRequest struct {
	RepoURL    string
	Owner      string
	Repo       string
	BaseBranch string
	HeadBranch string
	Title      string
	Body       string
	Token      string
}

type CreatePRResult struct {
	URL    string
	Number int
}

type RepoOwner struct {
	Owner string
	Repo  string
}

func ParseRepoOwner(rawURL string) (*RepoOwner, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse repo url: %w", err)
	}

	p := strings.TrimSuffix(u.Path, ".git")
	p = strings.Trim(p, "/")

	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		return nil, fmt.Errorf("unable to parse owner/repo from %s", rawURL)
	}

	repo := parts[len(parts)-1]
	owner := strings.Join(parts[:len(parts)-1], "/")

	return &RepoOwner{Owner: owner, Repo: repo}, nil
}

func InjectToken(rawURL, token string) string {
	if token == "" {
		return rawURL
	}
	if !strings.HasPrefix(rawURL, "https://") && !strings.HasPrefix(rawURL, "http://") {
		return rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	u.User = url.User(token)
	return u.String()
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
