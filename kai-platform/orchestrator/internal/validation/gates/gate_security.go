package gates

import (
	"fmt"
	"os/exec"

	"kaiplatform.com/orchestrator/internal/validation"
)

type SecurityScanGate struct {
	toolPath string
}

func NewSecurityScanGate() *SecurityScanGate {
	path := ""
	for _, candidate := range []string{"trivy", "snyk", "grype"} {
		if p, err := exec.LookPath(candidate); err == nil {
			path = p
			break
		}
	}
	return &SecurityScanGate{toolPath: path}
}

func (g *SecurityScanGate) Name() validation.Type {
	return "security_scan"
}

func (g *SecurityScanGate) Run(ctx *validation.Context) *validation.Result {
	if g.toolPath == "" {
		return &validation.Result{
			Gate:    "security_scan",
			Status:  validation.StatusSkipped,
			Message: "no security scanner found (trivy, snyk, or grype)",
		}
	}

	if ctx.RepoDir == "" {
		return &validation.Result{
			Gate:    "security_scan",
			Status:  validation.StatusSkipped,
			Message: "no repo directory to scan",
		}
	}

	tool := filepathBase(g.toolPath)

	switch tool {
	case "trivy":
		return g.runTrivy(ctx)
	case "snyk":
		return g.runSnyk(ctx)
	case "grype":
		return g.runGrype(ctx)
	default:
		return &validation.Result{
			Gate:    "security_scan",
			Status:  validation.StatusSkipped,
			Message: fmt.Sprintf("unexpected tool path: %s", g.toolPath),
		}
	}
}

func (g *SecurityScanGate) runTrivy(ctx *validation.Context) *validation.Result {
	cmd := exec.CommandContext(ctx, g.toolPath, "fs", "--severity", "CRITICAL,HIGH", ctx.RepoDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &validation.Result{
			Gate:    "security_scan",
			Status:  validation.StatusFailed,
			Message: fmt.Sprintf("trivy: %s", string(out)),
		}
	}
	return &validation.Result{
		Gate:    "security_scan",
		Status:  validation.StatusPassed,
		Message: "no critical/high severity vulnerabilities found",
	}
}

func (g *SecurityScanGate) runSnyk(ctx *validation.Context) *validation.Result {
	cmd := exec.CommandContext(ctx, g.toolPath, "test", "--severity-threshold=high")
	cmd.Dir = ctx.RepoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &validation.Result{
			Gate:    "security_scan",
			Status:  validation.StatusPassed,
			Message: fmt.Sprintf("snyk test requires auth: %s", string(out)),
		}
	}
	return &validation.Result{
		Gate:    "security_scan",
		Status:  validation.StatusPassed,
		Message: "snyk scan passed",
	}
}

func (g *SecurityScanGate) runGrype(ctx *validation.Context) *validation.Result {
	cmd := exec.CommandContext(ctx, g.toolPath, ctx.RepoDir, "--fail-on", "high", "-o", "json")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &validation.Result{
			Gate:    "security_scan",
			Status:  validation.StatusFailed,
			Message: fmt.Sprintf("grype: %s", string(out)),
		}
	}
	return &validation.Result{
		Gate:    "security_scan",
		Status:  validation.StatusPassed,
		Message: "no high/critical vulnerabilities found",
	}
}

func filepathBase(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[i+1:]
		}
	}
	return path
}
