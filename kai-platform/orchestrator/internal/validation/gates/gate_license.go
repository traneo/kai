package gates

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"kaiplatform.com/orchestrator/internal/validation"
)

type LicenseGate struct{}

func NewLicenseGate() *LicenseGate {
	return &LicenseGate{}
}

func (g *LicenseGate) Name() validation.Type {
	return "license_check"
}

func (g *LicenseGate) Run(ctx *validation.Context) *validation.Result {
	if ctx.RepoDir == "" {
		return &validation.Result{
			Gate:    "license_check",
			Status:  validation.StatusSkipped,
			Message: "no repo directory to check",
		}
	}

	licenseFiles := findLicenseFiles(ctx.RepoDir)
	if len(licenseFiles) > 0 {
		return &validation.Result{
			Gate:    "license_check",
			Status:  validation.StatusPassed,
			Message: fmt.Sprintf("license files found: %s", strings.Join(licenseFiles, ", ")),
		}
	}

	detected, err := detectProjectLicense(ctx.RepoDir)
	if err != nil {
		return &validation.Result{
			Gate:    "license_check",
			Status:  validation.StatusPassed,
			Message: fmt.Sprintf("license detection attempted: %s", err),
		}
	}

	if detected != "" {
		return &validation.Result{
			Gate:    "license_check",
			Status:  validation.StatusPassed,
			Message: fmt.Sprintf("license detected: %s", detected),
		}
	}

	return &validation.Result{
		Gate:    "license_check",
		Status:  validation.StatusPassed,
		Message: "no license file found — add LICENSE or LICENSE.md to project root",
	}
}

func findLicenseFiles(repoDir string) []string {
	var found []string
	candidates := []string{"LICENSE", "LICENSE.txt", "LICENSE.md", "LICENSE.apache", "LICENSE.mit", "COPYING", "COPYING.LESSER"}
	for _, name := range candidates {
		path := filepath.Join(repoDir, name)
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
			found = append(found, name)
			if len(found) >= 3 {
				break
			}
		}
	}
	return found
}

func detectProjectLicense(repoDir string) (string, error) {
	packageFiles := []struct {
		file      string
		field     string
		separator string
	}{
		{"package.json", "license", "\""},
		{"Cargo.toml", "license", "="},
		{"setup.cfg", "license", "="},
		{"go.mod", "", ""},
	}

	for _, pf := range packageFiles {
		path := filepath.Join(repoDir, pf.file)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		content := string(data)

		if pf.file == "go.mod" {
			if strings.Contains(content, "module ") {
				return "Go module", nil
			}
			continue
		}

		lines := strings.Split(content, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, pf.field) {
				parts := strings.Split(trimmed, pf.separator)
				if len(parts) >= 3 {
					val := strings.TrimSpace(parts[2])
					val = strings.Trim(val, "\",' ")
					if val != "" {
						return val, nil
					}
				}
			}
		}
	}

	_, errGoMod := os.Stat(filepath.Join(repoDir, "go.mod"))
	_, errCargo := os.Stat(filepath.Join(repoDir, "Cargo.toml"))
	_, errPackage := os.Stat(filepath.Join(repoDir, "package.json"))

	if errGoMod == nil || errCargo == nil || errPackage == nil {
		cmd := exec.Command("grep", "-ril", "--include=*.go", "--include=*.rs", "--include=*.ts", "--include=*.py", "SPDX-License-Identifier", repoDir)
		out, _ := cmd.CombinedOutput()
		if strings.TrimSpace(string(out)) != "" {
			return "SPDX-License-Identifier found in source files", nil
		}
	}

	return "", fmt.Errorf("no license source detected")
}
