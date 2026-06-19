package sandbox

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Config struct {
	AllowedDirs     []string
	AllowedTools    []string
	AllowedCommands []string
}

type Sandbox struct {
	Config  Config
	WorkDir string
	RepoDir string
}

func New(cfg Config) *Sandbox {
	return &Sandbox{Config: cfg}
}

func (s *Sandbox) Setup() error {
	dir, err := os.MkdirTemp("", "kai-agent-*")
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	s.WorkDir = dir
	s.RepoDir = filepath.Join(dir, "repo")
	if err := os.MkdirAll(s.RepoDir, 0755); err != nil {
		return fmt.Errorf("create repo dir: %w", err)
	}
	return s.initRepo()
}

func (s *Sandbox) initRepo() error {
	git := func(args ...string) *exec.Cmd {
		all := append([]string{"-c", "credential.helper="}, args...)
		cmd := exec.Command("git", all...)
		cmd.Dir = s.RepoDir
		return cmd
	}

	git("init").Run()

	for _, cfg := range []struct{ key, val string }{
		{"user.email", "kai-code@agent"},
		{"user.name", "KaiCode Agent"},
	} {
		if err := git("config", cfg.key, cfg.val).Run(); err != nil {
			return fmt.Errorf("git config %s: %w", cfg.key, err)
		}
	}

	if err := s.ignoreKaiFiles(); err != nil {
		return fmt.Errorf("ignore kai files: %w", err)
	}

	if err := git("commit", "--allow-empty", "-m", "initial commit").Run(); err != nil {
		return fmt.Errorf("initial commit: %w", err)
	}

	git("branch", "-M", "main").Run()

	return nil
}

func (s *Sandbox) SetupWithRepo(ctx context.Context, repoURL, branch string) error {
	dir, err := os.MkdirTemp("", "kai-agent-*")
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	s.WorkDir = dir
	s.RepoDir = filepath.Join(dir, "repo")

	if repoURL == "" {
		if err := os.MkdirAll(s.RepoDir, 0755); err != nil {
			return fmt.Errorf("create repo dir: %w", err)
		}
		return s.initRepo()
	}

	if branch == "" {
		branch = "main"
	}

	gitCmd := func(args ...string) *exec.Cmd {
		all := append([]string{"-c", "credential.helper="}, args...)
		cmd := exec.CommandContext(ctx, "git", all...)
		cmd.Dir = s.RepoDir
		return cmd
	}

	if err := exec.CommandContext(ctx, "git", "-c", "credential.helper=", "clone", repoURL, s.RepoDir).Run(); err != nil {
		return fmt.Errorf("git clone %s: %w", repoURL, err)
	}

	var stdout, stderr bytes.Buffer
	checkout := gitCmd("checkout", branch)
	checkout.Stdout = &stdout
	checkout.Stderr = &stderr
	if err := checkout.Run(); err != nil {
		create := gitCmd("checkout", "-b", branch)
		if err2 := create.Run(); err2 != nil {
			return fmt.Errorf("git checkout %s: %w\nstderr: %s\nfallback create branch: %v", branch, err, stderr.String(), err2)
		}
	}

	if err := s.ignoreKaiFiles(); err != nil {
		return fmt.Errorf("ignore kai files: %w", err)
	}

	// Remove remote so subprocess cannot push
	if repoURL != "" {
		gitCmd("remote", "remove", "origin").Run()
	}

	return nil
}

func (s *Sandbox) ignoreKaiFiles() error {
	gitignorePath := filepath.Join(s.RepoDir, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read .gitignore: %w", err)
	}
	content := string(data)

	patterns := []string{".kai/", "kai.json", ".opencode/", "opencode.jsonc"}
	var need []string
	for _, p := range patterns {
		if !strings.Contains(content, p) {
			need = append(need, p)
		}
	}

	if len(need) == 0 {
		return nil
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open .gitignore: %w", err)
	}
	defer f.Close()

	if content != "" && !strings.HasSuffix(content, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	for _, p := range need {
		if _, err := fmt.Fprintf(f, "/%s\n", p); err != nil {
			return err
		}
	}
	return nil
}

func (s *Sandbox) Cleanup() {
	if s.WorkDir != "" {
		os.RemoveAll(s.WorkDir)
	}
}

func (s *Sandbox) ResolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(s.RepoDir, path)
}

func (s *Sandbox) IsAllowedDir(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	if s.RepoDir != "" && !strings.HasPrefix(abs, s.RepoDir) {
		return false
	}

	for _, allowed := range s.Config.AllowedDirs {
		allowedAbs := s.ResolvePath(allowed)
		if strings.HasPrefix(abs, allowedAbs) {
			return true
		}
	}
	return false
}

func (s *Sandbox) IsAllowedTool(tool string) bool {
	for _, t := range s.Config.AllowedTools {
		if t == tool {
			return true
		}
	}
	return false
}

func (s *Sandbox) IsAllowedCommand(cmd string) bool {
	for _, pattern := range s.Config.AllowedCommands {
		if matchCommand(pattern, cmd) {
			return true
		}
	}
	return false
}

func matchCommand(pattern, cmd string) bool {
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(cmd, prefix)
	}
	return pattern == cmd
}

var zipSkipDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	".angular":     true,
	"dist":         true,
	".cache":       true,
	"__pycache__":  true,
	".next":        true,
	"target":       true,
	"build":        true,
	".tox":         true,
	".venv":        true,
}

func ZipDir(dir string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	err := filepath.Walk(dir, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		base := filepath.Base(path)
		if fi.IsDir() && zipSkipDirs[base] {
			return filepath.SkipDir
		}

		if fi.IsDir() {
			_, err := zw.Create(rel + "/")
			return err
		}

		fh, err := zip.FileInfoHeader(fi)
		if err != nil {
			return err
		}
		fh.Name = rel
		fh.Method = zip.Deflate

		w, err := zw.CreateHeader(fh)
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(w, f)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("zip walk: %w", err)
	}

	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("zip close: %w", err)
	}

	return buf.Bytes(), nil
}
