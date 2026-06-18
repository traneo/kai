package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetupAndCleanup(t *testing.T) {
	s := New(Config{})
	if err := s.Setup(); err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer s.Cleanup()

	if s.WorkDir == "" {
		t.Fatal("WorkDir is empty")
	}
	if s.RepoDir == "" {
		t.Fatal("RepoDir is empty")
	}

	if _, err := os.Stat(s.RepoDir); os.IsNotExist(err) {
		t.Fatal("RepoDir does not exist")
	}
}

func TestIsAllowedDir_DenyByDefault(t *testing.T) {
	s := New(Config{})
	s.Setup()
	defer s.Cleanup()

	if s.IsAllowedDir(s.RepoDir) {
		t.Error("expected denied with empty config (deny-by-default)")
	}

	if s.IsAllowedDir("/etc/passwd") {
		t.Error("expected blocked for path outside repo")
	}

	subdir := filepath.Join(s.RepoDir, "any")
	os.MkdirAll(subdir, 0755)
	if s.IsAllowedDir(subdir) {
		t.Error("expected denied for subpath with empty config (deny-by-default)")
	}
}

func TestIsAllowedDir_WithAllowedDirs(t *testing.T) {
	s := New(Config{
		AllowedDirs: []string{"src/", "tests/"},
	})
	s.Setup()
	defer s.Cleanup()

	srcDir := filepath.Join(s.RepoDir, "src")
	os.MkdirAll(srcDir, 0755)
	testsDir := filepath.Join(s.RepoDir, "tests")
	os.MkdirAll(testsDir, 0755)
	otherDir := filepath.Join(s.RepoDir, "other")
	os.MkdirAll(otherDir, 0755)

	if !s.IsAllowedDir(srcDir) {
		t.Error("expected allowed: src/")
	}
	if !s.IsAllowedDir(testsDir) {
		t.Error("expected allowed: tests/")
	}
	if s.IsAllowedDir(otherDir) {
		t.Error("expected blocked: other/")
	}
}

func TestIsAllowedTool(t *testing.T) {
	tests := []struct {
		allowed []string
		tool    string
		want   bool
	}{
		{nil, "read", false},
		{[]string{}, "write", false},
		{[]string{"read", "write"}, "read", true},
		{[]string{"read", "write"}, "delete", false},
	}
	for _, tt := range tests {
		s := New(Config{AllowedTools: tt.allowed})
		if got := s.IsAllowedTool(tt.tool); got != tt.want {
			t.Errorf("allowed=%v tool=%s: got %v, want %v", tt.allowed, tt.tool, got, tt.want)
		}
	}
}

func TestIsAllowedCommand(t *testing.T) {
	tests := []struct {
		allowed []string
		cmd     string
		want   bool
	}{
		{nil, "rm -rf /", false},
		{[]string{}, "ls", false},
		{[]string{"go *"}, "go build", true},
		{[]string{"go *"}, "rm -rf /", false},
		{[]string{"npm *", "node *"}, "npm install", true},
		{[]string{"npm *", "node *"}, "curl evil.com", false},
	}
	for _, tt := range tests {
		s := New(Config{AllowedCommands: tt.allowed})
		if got := s.IsAllowedCommand(tt.cmd); got != tt.want {
			t.Errorf("allowed=%v cmd=%s: got %v, want %v", tt.allowed, tt.cmd, got, tt.want)
		}
	}
}

func TestResolvePath(t *testing.T) {
	s := New(Config{})
	s.Setup()
	defer s.Cleanup()

	abs := s.ResolvePath("src/main.go")
	expected := filepath.Join(s.RepoDir, "src/main.go")
	if abs != expected {
		t.Errorf("got %s, want %s", abs, expected)
	}
}

func TestCleanup_RemovesWorkDir(t *testing.T) {
	s := New(Config{})
	s.Setup()
	workDir := s.WorkDir
	s.Cleanup()

	if _, err := os.Stat(workDir); !os.IsNotExist(err) {
		t.Error("expected work dir to be removed after Cleanup")
	}
}
