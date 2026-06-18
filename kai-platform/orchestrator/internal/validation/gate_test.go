package validation

import (
	"context"
	"testing"
)

func TestResultMethods(t *testing.T) {
	passed := &Result{Status: StatusPassed}
	failed := &Result{Status: StatusFailed}
	skipped := &Result{Status: StatusSkipped}

	if !passed.Passed() {
		t.Error("expected Passed() to be true")
	}
	if !failed.Failed() {
		t.Error("expected Failed() to be true")
	}
	if skipped.Passed() || skipped.Failed() {
		t.Error("expected skipped to be neither")
	}
}

func TestResultString(t *testing.T) {
	r := &Result{Gate: TypeExitCode, Status: StatusPassed, Message: "ok"}
	s := r.String()
	if s != "exit_zero: passed (ok)" {
		t.Errorf("unexpected string: %s", s)
	}
}

func TestNewExitCodeGate_Passed(t *testing.T) {
	g := NewExitCodeGate()
	ctx := &Context{Context: context.Background(), ExitCode: 0}
	res := g.Run(ctx)
	if res.Failed() {
		t.Errorf("expected passed, got %s", res.Status)
	}
}

func TestNewExitCodeGate_Failed(t *testing.T) {
	g := NewExitCodeGate()
	ctx := &Context{Context: context.Background(), ExitCode: 1}
	res := g.Run(ctx)
	if res.Passed() {
		t.Errorf("expected failed, got %s", res.Status)
	}
}

func TestApprovalGate_NotRequired(t *testing.T) {
	g := NewApprovalGate(false)
	ctx := &Context{Context: context.Background()}
	res := g.Run(ctx)
	if res.Status != StatusSkipped {
		t.Errorf("expected skipped, got %s", res.Status)
	}
}

func TestApprovalGate_Required(t *testing.T) {
	g := NewApprovalGate(true)
	ctx := &Context{Context: context.Background()}
	res := g.Run(ctx)
	if res.Status != StatusPending {
		t.Errorf("expected pending, got %s", res.Status)
	}
}

func TestDiffReviewGate_NoDiff(t *testing.T) {
	g := NewDiffReviewGate()
	ctx := &Context{Context: context.Background(), RepoDir: t.TempDir()}
	res := g.Run(ctx)
	if res.Failed() {
		t.Errorf("expected passed, got: %s", res.Message)
	}
}

func TestDiffReviewGate_DetectsSecret(t *testing.T) {
	g := NewDiffReviewGate()

	diff := []byte(`diff --git a/config.go b/config.go
new file mode 100644
--- /dev/null
+++ b/config.go
@@ -0,0 +1,5 @@
+package config
+
+func GetConfig() {
+    api_key = "sk-1234567890abcdef"
+}
`)

	issues, err := g.reviewDiffContent(diff)
	if err != nil {
		t.Fatalf("reviewDiffContent: %v", err)
	}
	if len(issues) == 0 {
		t.Fatal("expected to detect secret in diff")
	}
}

func TestDiffReviewGate_CleanDiff(t *testing.T) {
	g := NewDiffReviewGate()

	diff := []byte(`diff --git a/main.go b/main.go
new file mode 100644
--- /dev/null
+++ b/main.go
@@ -0,0 +1,3 @@
+package main
+
+func main() {}
`)

	issues, err := g.reviewDiffContent(diff)
	if err != nil {
		t.Fatalf("reviewDiffContent: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues, got %d", len(issues))
	}
}
