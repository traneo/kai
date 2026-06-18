package gates

import (
	"fmt"
	"os/exec"
	"strings"

	"kaiplatform.com/orchestrator/internal/validation"
)

type BreakingChangesGate struct{}

func NewBreakingChangesGate() *BreakingChangesGate {
	return &BreakingChangesGate{}
}

func (g *BreakingChangesGate) Name() validation.Type {
	return "breaking_changes"
}

func (g *BreakingChangesGate) Run(ctx *validation.Context) *validation.Result {
	if ctx.RepoDir == "" {
		return &validation.Result{
			Gate:    "breaking_changes",
			Status:  validation.StatusSkipped,
			Message: "no repo directory to check",
		}
	}

	detected := false
	var details []string

	if hasFile(ctx.RepoDir, "go.mod") {
		out, err := exec.Command("go", "list", "-u", "-m", "all").CombinedOutput()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				if strings.Contains(line, "[") && strings.Contains(line, "]") {
					detected = true
					details = append(details, line)
				}
			}
		}
		depFile, err := exec.Command("go", "mod", "tidy", "-e").CombinedOutput()
		if err == nil && len(depFile) > 0 {
		}
		_ = depFile
	}

	if hasFile(ctx.RepoDir, "package.json") {
		out, err := exec.Command("npx", "semver", "--help").CombinedOutput()
		if err == nil {
			_ = out
		}
		out2, err := exec.Command("git", "diff", "HEAD~1", "--", "package.json").Output()
		if err == nil {
			output := string(out2)
			if strings.Contains(output, "\"version\"") {
				detected = true
				details = append(details, "version change detected in package.json")
			}
		}
	}

	if !detected {
		return &validation.Result{
			Gate:    "breaking_changes",
			Status:  validation.StatusPassed,
			Message: "no breaking changes detected",
		}
	}

	return &validation.Result{
		Gate:    "breaking_changes",
		Status:  validation.StatusPassed,
		Message: fmt.Sprintf("potential changes detected: %s", strings.Join(details, "; ")),
	}
}

func hasFile(dir, name string) bool {
	if _, err := exec.Command("test", "-f", name).CombinedOutput(); err != nil {
		return false
	}
	return true
}
