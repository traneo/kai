package gitops

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"kaiplatform.com/orchestrator/internal/gitprovider"
)

type Config struct {
	RepoURL      string
	BaseBranch   string
	BranchPrefix string
	WorkDir      string
	GitUserName  string
	GitUserEmail string
	RunID        string
	Token        string
	GitProvider  gitprovider.Provider
}

type Client struct {
	cfg     Config
	repoDir string
	branch  string
}

func New(cfg Config) *Client {
	return &Client{
		cfg:    cfg,
		branch: cfg.BranchPrefix + cfg.RunID,
	}
}

func (c *Client) BranchName() string {
	return c.branch
}

func (c *Client) RepoDir() string {
	return c.repoDir
}

func (c *Client) Clone(ctx context.Context, baseBranch string) error {
	dir, err := os.MkdirTemp("", "kai-git-*")
	if err != nil {
		return fmt.Errorf("create workdir: %w", err)
	}
	c.repoDir = filepath.Join(dir, "repo")

	if c.cfg.RepoURL == "" {
		if err := os.MkdirAll(c.repoDir, 0755); err != nil {
			return fmt.Errorf("create empty repo dir: %w", err)
		}
		if err := c.exec(ctx, c.repoDir, "git", "init"); err != nil {
			return fmt.Errorf("git init: %w", err)
		}
		return nil
	}

	cloneURL := gitprovider.InjectToken(c.cfg.RepoURL, c.cfg.Token)
	if err := c.exec(ctx, dir, "git", "clone", cloneURL, c.repoDir); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	if baseBranch == "" {
		baseBranch = "main"
	}

	out, err := c.output(ctx, c.repoDir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		currentBranch := strings.TrimSpace(out)
		if currentBranch != baseBranch {
			if err := c.exec(ctx, c.repoDir, "git", "checkout", baseBranch); err != nil {
				return fmt.Errorf("git checkout %s: %w", baseBranch, err)
			}
		}
	}

	if err := c.exec(ctx, c.repoDir, "git", "checkout", "-b", c.branch); err != nil {
		return fmt.Errorf("git checkout -b %s: %w", c.branch, err)
	}

	return nil
}

func (c *Client) SetupGitConfig(ctx context.Context) error {
	if c.cfg.GitUserName == "" {
		c.cfg.GitUserName = "kai-platform"
	}
	if c.cfg.GitUserEmail == "" {
		c.cfg.GitUserEmail = "kai@kai-platform.local"
	}
	if err := c.exec(ctx, c.repoDir, "git", "config", "user.name", c.cfg.GitUserName); err != nil {
		return fmt.Errorf("git config user.name: %w", err)
	}
	if err := c.exec(ctx, c.repoDir, "git", "config", "user.email", c.cfg.GitUserEmail); err != nil {
		return fmt.Errorf("git config user.email: %w", err)
	}
	return nil
}

func (c *Client) StageAll(ctx context.Context) error {
	return c.exec(ctx, c.repoDir, "git", "add", "-A", "--", ":!.kai-code/", ":!kai-code.json")
}

func (c *Client) Commit(ctx context.Context, message string) error {
	if err := c.exec(ctx, c.repoDir, "git", "commit", "-m", message); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	return nil
}

func (c *Client) PushInitialBranch(ctx context.Context) error {
	if c.cfg.RepoURL == "" {
		return nil
	}
	if err := c.exec(ctx, c.repoDir, "git", "push", "-u", "origin", c.branch); err != nil {
		return fmt.Errorf("git push -u origin %s: %w", c.branch, err)
	}
	return nil
}

func (c *Client) Pull(ctx context.Context) error {
	if c.cfg.RepoURL == "" {
		return nil
	}
	if err := c.exec(ctx, c.repoDir, "git", "pull", "--rebase", "origin", c.branch); err != nil {
		return fmt.Errorf("git pull --rebase origin %s: %w", c.branch, err)
	}
	return nil
}

func (c *Client) Push(ctx context.Context) error {
	if c.cfg.RepoURL == "" {
		return nil
	}
	if err := c.exec(ctx, c.repoDir, "git", "push", "origin", c.branch); err != nil {
		return fmt.Errorf("git push origin %s: %w", c.branch, err)
	}
	return nil
}

func (c *Client) PushForce(ctx context.Context) error {
	if c.cfg.RepoURL == "" {
		return nil
	}
	if err := c.exec(ctx, c.repoDir, "git", "push", "--force", "origin", c.branch); err != nil {
		return fmt.Errorf("git push --force origin %s: %w", c.branch, err)
	}
	return nil
}

func (c *Client) CreatePR(ctx context.Context, title, body string) (string, error) {
	if c.cfg.RepoURL == "" {
		return "no-repo", nil
	}

	if c.cfg.GitProvider == nil {
		return "", fmt.Errorf("no git provider configured - cannot create PR for %s", c.cfg.RepoURL)
	}

	ownerRepo, err := gitprovider.ParseRepoOwner(c.cfg.RepoURL)
	if err != nil {
		return "", fmt.Errorf("parse repo url: %w", err)
	}

	result, err := c.cfg.GitProvider.CreatePR(ctx, gitprovider.CreatePRRequest{
		RepoURL:    c.cfg.RepoURL,
		Owner:      ownerRepo.Owner,
		Repo:       ownerRepo.Repo,
		BaseBranch: c.cfg.BaseBranch,
		HeadBranch: c.branch,
		Title:      title,
		Body:       body,
		Token:      c.cfg.Token,
	})
	if err != nil {
		return "", err
	}

	return result.URL, nil
}

func (c *Client) GetDiff(ctx context.Context) (string, error) {
	diff, err := c.output(ctx, c.repoDir, "git", "diff", "--diff-filter=AM")
	if err != nil {
		return "", err
	}
	cached, err := c.output(ctx, c.repoDir, "git", "diff", "--cached", "--diff-filter=AM")
	if err == nil && cached != "" {
		if diff != "" {
			diff += "\n"
		}
		diff += cached
	}
	return diff, nil
}

// GetBranchDiff returns the diff of all changes on the current branch compared to the base branch.
// For repos with a remote, it compares against origin/<base> (agent commits are pushed).
// For local repos, it falls back to working tree diff (agent writes files directly).
func (c *Client) GetBranchDiff(ctx context.Context) (string, error) {
	base := c.cfg.BaseBranch
	if base == "" {
		base = "main"
	}

	var ref string
	if c.cfg.RepoURL != "" {
		ref = "origin/" + base
	} else {
		ref = base
	}

	diff, err := c.output(ctx, c.repoDir, "git", "diff", "--diff-filter=AM", ref+"...HEAD")
	if err != nil || diff == "" {
		// Fall back to working tree diff for local repos (agents write without committing)
		wdiff, werr := c.GetDiff(ctx)
		if werr == nil && wdiff != "" {
			return wdiff, nil
		}
		if err != nil {
			return "", err
		}
		return "", nil
	}
	return diff, nil
}

func (c *Client) HasChanges(ctx context.Context) (bool, error) {
	out, err := c.output(ctx, c.repoDir, "git", "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

func (c *Client) Reset(ctx context.Context) error {
	return c.exec(ctx, c.repoDir, "git", "reset", "--hard")
}

func (c *Client) GetCurrentCommit(ctx context.Context) (string, error) {
	out, err := c.output(ctx, c.repoDir, "git", "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// GetDiffBetween returns the diff of changes between two commits (from...to).
// This shows only what changed in that range, suitable for per-step review.
func (c *Client) GetDiffBetween(ctx context.Context, from, to string) (string, error) {
	return c.output(ctx, c.repoDir, "git", "diff", "--diff-filter=AM", from+"..."+to)
}

func (c *Client) Cleanup(ctx context.Context) {
	if c.repoDir != "" {
		parent := filepath.Dir(c.repoDir)
		if strings.HasPrefix(parent, os.TempDir()) {
			os.RemoveAll(parent)
		}
	}
}

func (c *Client) exec(ctx context.Context, dir, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = nil
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o BatchMode=yes -o ConnectTimeout=15",
	)
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w\n%s", name, args, err, cmd.Stderr.(*bytes.Buffer).String())
	}
	return nil
}

func (c *Client) output(ctx context.Context, dir, name string, args ...string) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir
	cmd.Stdin = nil
	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o BatchMode=yes -o ConnectTimeout=15",
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %v: %w\n%s", name, args, err, stderr.String())
	}
	return stdout.String(), nil
}

func GenerateStepCommitMessage(runID, stepID, prompt string) string {
	maxLen := 72
	msg := fmt.Sprintf("[%s] %s: %s", runID, stepID, prompt)
	if len(msg) > maxLen {
		msg = msg[:maxLen-3] + "..."
	}
	return msg
}

func GenerateBranchName(prefix, runID string) string {
	if prefix == "" {
		return fmt.Sprintf("kai/%s", runID)
	}
	return prefix + runID
}

func GeneratePRTitle(project, runID string) string {
	return fmt.Sprintf("[kai-platform] %s - %s", project, runID)
}

func GeneratePRBody(runID string, steps []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## kai Platform Pipeline Run\n\n")
	fmt.Fprintf(&b, "**Run ID:** %s\n\n", runID)
	fmt.Fprintf(&b, "**Steps completed:**\n\n")
	for _, s := range steps {
		fmt.Fprintf(&b, "- %s\n", s)
	}
	fmt.Fprintf(&b, "\n---\n")
	fmt.Fprintf(&b, "This PR was automatically generated by kai Platform.\n")
	return b.String()
}

type DirectCommitResult struct {
	Branch    string
	CommitSHA string
}

type PRResult struct {
	URL    string
	Branch string
}

func (c *Client) StageCommitPush(ctx context.Context, message string) (*DirectCommitResult, error) {
	if err := c.StageAll(ctx); err != nil {
		return nil, fmt.Errorf("stage: %w", err)
	}

	hasChanges, err := c.HasChanges(ctx)
	if err != nil {
		return nil, fmt.Errorf("check changes: %w", err)
	}
	if !hasChanges {
		return &DirectCommitResult{Branch: c.branch}, nil
	}

	if err := c.Commit(ctx, message); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	sha, err := c.GetCurrentCommit(ctx)
	if err != nil {
		return nil, fmt.Errorf("get commit: %w", err)
	}

	if err := c.Push(ctx); err != nil {
		return nil, fmt.Errorf("push: %w", err)
	}

	return &DirectCommitResult{
		Branch:    c.branch,
		CommitSHA: sha,
	}, nil
}

func (c *Client) StageCommitPushPR(ctx context.Context, title, body, stepMessagePrefix string) (*PRResult, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	commitMsg := fmt.Sprintf("%s - %s", stepMessagePrefix, now)

	if _, err := c.StageCommitPush(ctx, commitMsg); err != nil {
		return nil, err
	}

	url, err := c.CreatePR(ctx, title, body)
	if err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}

	return &PRResult{
		URL:    url,
		Branch: c.branch,
	}, nil
}
