package gitops

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, path)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestNewAndBranchName(t *testing.T) {
	c := New(Config{RunID: "run-1", BranchPrefix: "feat/kai-"})
	if c.BranchName() != "feat/kai-run-1" {
		t.Errorf("expected feat/kai-run-1, got %s", c.BranchName())
	}
}

func TestClone_EmptyRepo(t *testing.T) {
	c := New(Config{RunID: "test-1", BranchPrefix: "kai/"})
	ctx := context.Background()

	if err := c.Clone(ctx, ""); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	defer c.Cleanup(ctx)

	if c.RepoDir() == "" {
		t.Fatal("RepoDir is empty")
	}

	if _, err := os.Stat(filepath.Join(c.RepoDir(), ".git")); err != nil {
		t.Errorf("expected git repo: %v", err)
	}
}

func TestClone_WithBranch(t *testing.T) {
	c := New(Config{RunID: "test-2", BranchPrefix: "kai/"})
	ctx := context.Background()

	if err := c.Clone(ctx, ""); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	defer c.Cleanup(ctx)

	if !strings.HasSuffix(c.BranchName(), "test-2") {
		t.Errorf("expected branch suffix test-2, got %s", c.BranchName())
	}
}

func TestStageCommitPush_EmptyRepo(t *testing.T) {
	c := New(Config{RunID: "test-3", BranchPrefix: "kai/"})
	ctx := context.Background()

	if err := c.Clone(ctx, ""); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	defer c.Cleanup(ctx)

	if err := c.SetupGitConfig(ctx); err != nil {
		t.Fatalf("SetupGitConfig: %v", err)
	}

	writeFile(t, c.RepoDir(), "test.txt", "hello")

	result, err := c.StageCommitPush(ctx, "test commit")
	if err != nil {
		t.Fatalf("StageCommitPush: %v", err)
	}
	if result.CommitSHA == "" {
		t.Fatal("expected non-empty commit SHA")
	}
	if result.Branch != "kai/test-3" {
		t.Errorf("expected branch kai/test-3, got %s", result.Branch)
	}
}

func TestHasChanges_True(t *testing.T) {
	c := New(Config{RunID: "test-4"})
	ctx := context.Background()

	if err := c.Clone(ctx, ""); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	defer c.Cleanup(ctx)

	writeFile(t, c.RepoDir(), "new.txt", "content")

	has, err := c.HasChanges(ctx)
	if err != nil {
		t.Fatalf("HasChanges: %v", err)
	}
	if !has {
		t.Error("expected changes")
	}
}

func TestHasChanges_False(t *testing.T) {
	c := New(Config{RunID: "test-5"})
	ctx := context.Background()

	if err := c.Clone(ctx, ""); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	defer c.Cleanup(ctx)

	has, err := c.HasChanges(ctx)
	if err != nil {
		t.Fatalf("HasChanges: %v", err)
	}
	if has {
		t.Error("expected no changes in empty repo")
	}
}

func TestGetDiff(t *testing.T) {
	c := New(Config{RunID: "test-6"})
	ctx := context.Background()

	if err := c.Clone(ctx, ""); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	defer c.Cleanup(ctx)

	writeFile(t, c.RepoDir(), "file.txt", "content")

	if err := c.StageAll(ctx); err != nil {
		t.Fatalf("StageAll: %v", err)
	}

	diff, err := c.GetDiff(ctx)
	if err != nil {
		t.Fatalf("GetDiff: %v", err)
	}
	if !strings.Contains(diff, "file.txt") {
		t.Errorf("expected diff to contain file.txt, got: %s", diff)
	}
}

func TestReset(t *testing.T) {
	c := New(Config{RunID: "test-7"})
	ctx := context.Background()

	if err := c.Clone(ctx, ""); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	defer c.Cleanup(ctx)

	if err := c.SetupGitConfig(ctx); err != nil {
		t.Fatalf("SetupGitConfig: %v", err)
	}

	writeFile(t, c.RepoDir(), "f.txt", "v1")
	if _, err := c.StageCommitPush(ctx, "initial"); err != nil {
		t.Fatalf("StageCommitPush: %v", err)
	}

	writeFile(t, c.RepoDir(), "f.txt", "v2")
	if _, err := c.StageCommitPush(ctx, "second"); err != nil {
		t.Fatalf("StageCommitPush: %v", err)
	}

	sha, err := c.GetCurrentCommit(ctx)
	if err != nil {
		t.Fatalf("GetCurrentCommit: %v", err)
	}

	if err := c.Reset(ctx); err != nil {
		t.Fatalf("Reset: %v", err)
	}

	sha2, err := c.GetCurrentCommit(ctx)
	if err != nil {
		t.Fatalf("GetCurrentCommit: %v", err)
	}
	if sha != sha2 {
		t.Errorf("expected same SHA after reset, got %s vs %s", sha, sha2)
	}
}

func TestGenerateHelpers(t *testing.T) {
	msg := GenerateStepCommitMessage("run-1", "scaffold", "Initialize project")
	if !strings.Contains(msg, "run-1") || !strings.Contains(msg, "scaffold") {
		t.Errorf("unexpected commit message: %s", msg)
	}

	branch := GenerateBranchName("feat/kai-", "run-1")
	if branch != "feat/kai-run-1" {
		t.Errorf("expected feat/kai-run-1, got %s", branch)
	}

	title := GeneratePRTitle("my-service", "run-1")
	if !strings.Contains(title, "my-service") {
		t.Errorf("expected title to contain project, got %s", title)
	}

	body := GeneratePRBody("run-1", []string{"scaffold", "implement-api"})
	if !strings.Contains(body, "scaffold") || !strings.Contains(body, "implement-api") {
		t.Errorf("body missing steps: %s", body)
	}
}

func TestReprocess(t *testing.T) {
	c := New(Config{RunID: "test-8"})
	ctx := context.Background()

	if err := c.Clone(ctx, ""); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	defer c.Cleanup(ctx)

	if err := c.SetupGitConfig(ctx); err != nil {
		t.Fatalf("SetupGitConfig: %v", err)
	}

	writeFile(t, c.RepoDir(), "a.go", "package main")
	result, err := c.StageCommitPush(ctx, "first step")
	if err != nil {
		t.Fatalf("first commit: %v", err)
	}
	if result.CommitSHA == "" {
		t.Fatal("expected SHA")
	}

	writeFile(t, c.RepoDir(), "b.go", "package main")
	result2, err := c.StageCommitPush(ctx, "second step")
	if err != nil {
		t.Fatalf("second commit: %v", err)
	}
	if result2.CommitSHA == result.CommitSHA {
		t.Error("expected different SHA")
	}
}
