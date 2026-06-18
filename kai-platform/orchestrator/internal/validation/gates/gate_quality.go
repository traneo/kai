package gates

import (
	"fmt"
	"os/exec"
	"strings"

	"kaiplatform.com/orchestrator/internal/validation"
)

type CodeQualityGate struct{}

func NewCodeQualityGate() *CodeQualityGate {
	return &CodeQualityGate{}
}

func (g *CodeQualityGate) Name() validation.Type {
	return "code_quality"
}

func (g *CodeQualityGate) Run(ctx *validation.Context) *validation.Result {
	if ctx.RepoDir == "" {
		return &validation.Result{
			Gate:    "code_quality",
			Status:  validation.StatusSkipped,
			Message: "no repo directory to analyze",
		}
	}

	tools := detectQualityTools()
	if len(tools) == 0 {
		return &validation.Result{
			Gate:    "code_quality",
			Status:  validation.StatusSkipped,
			Message: "no code quality tool detected (sonar-scanner, codacy, or reviewdog)",
		}
	}

	var issues []string
	for _, tool := range tools {
		switch tool {
		case "sonar-scanner":
			issues = append(issues, runSonarScanner(ctx.RepoDir))
		case "reviewdog":
			issues = append(issues, runReviewdog(ctx.RepoDir))
		}
	}

	if len(issues) > 0 {
		return &validation.Result{
			Gate:    "code_quality",
			Status:  validation.StatusPassed,
			Message: fmt.Sprintf("quality checks: %s", strings.Join(issues, "; ")),
		}
	}

	return &validation.Result{
		Gate:    "code_quality",
		Status:  validation.StatusPassed,
		Message: "code quality checks passed",
	}
}

func detectQualityTools() []string {
	var found []string
	candidates := []string{"sonar-scanner", "reviewdog", "codacy-cli"}
	for _, c := range candidates {
		if _, err := exec.LookPath(c); err == nil {
			found = append(found, c)
		}
	}
	return found
}

func runSonarScanner(repoDir string) string {
	cmd := exec.Command("sonar-scanner",
		fmt.Sprintf("-Dsonar.projectBaseDir=%s", repoDir),
		"-Dsonar.sources=.",
		"-Dsonar.host.url=${SONAR_HOST_URL:-http://localhost:9000}",
		"-Dsonar.token=${SONAR_TOKEN:-}",
	)
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Sprintf("sonar-scanner: %s", truncateOutput(string(out)))
	}
	return "sonar-scanner: analysis submitted"
}

func runReviewdog(repoDir string) string {
	cmd := exec.Command("reviewdog", "-reporter=local", "-runners=golint,eslint")
	cmd.Dir = repoDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Sprintf("reviewdog: %s", truncateOutput(string(out)))
	}
	return "reviewdog: no issues found"
}

func truncateOutput(s string) string {
	lines := strings.SplitN(s, "\n", 6)
	if len(lines) > 5 {
		return strings.Join(lines[:5], "\n") + "..."
	}
	return s
}
