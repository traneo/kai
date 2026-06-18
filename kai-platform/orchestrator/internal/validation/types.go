package validation

import (
	"context"
	"fmt"
	"time"
)

type Type string

const (
	TypeExitCode   Type = "exit_zero"
	TypeLint       Type = "lint"
	TypeTypecheck  Type = "typecheck"
	TypeTests      Type = "tests"
	TypeDiffReview Type = "diff_review"
	TypeApproval   Type = "approval"
)

type Status string

const (
	StatusPassed    Status = "passed"
	StatusFailed    Status = "failed"
	StatusSkipped   Status = "skipped"
	StatusPending   Status = "pending"
)

type Result struct {
	Gate     Type          `json:"gate"`
	Status   Status        `json:"status"`
	Message  string        `json:"message"`
	Duration time.Duration `json:"duration_ms"`
}

func (r *Result) Passed() bool  { return r.Status == StatusPassed }
func (r *Result) Failed() bool  { return r.Status == StatusFailed }

func (r *Result) String() string {
	return fmt.Sprintf("%s: %s (%s)", r.Gate, r.Status, r.Message)
}

type Context struct {
	context.Context
	ExitCode     int
	WorkspaceDir string
	RepoDir      string
	BranchName   string
}

type Gate interface {
	Name() Type
	Run(ctx *Context) *Result
}
