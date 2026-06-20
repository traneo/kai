package validation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type LintGate struct {
	command string
	args    []string
	name    Type
}

func NewLintGate() *LintGate {
	return &LintGate{
		command: "",
		args:    nil,
		name:    TypeLint,
	}
}

func NewTypecheckGate() *LintGate {
	return &LintGate{
		command: "",
		args:    nil,
		name:    TypeTypecheck,
	}
}

func NewTestsGate() *LintGate {
	return &LintGate{
		command: "",
		args:    nil,
		name:    TypeTests,
	}
}

func (g *LintGate) Name() Type { return g.name }

func (g *LintGate) Run(ctx *Context) *Result {
	cmds := g.detectCommands(ctx.RepoDir)
	if len(cmds) == 0 {
		return &Result{
			Gate:    g.name,
			Status:  StatusSkipped,
			Message: fmt.Sprintf("no %s tool detected for this project", g.name),
		}
	}

	for _, cmd := range cmds {
		c := exec.Command(cmd[0], cmd[1:]...)
		if ctx.RepoDir != "" {
			c.Dir = ctx.RepoDir
		}
		out, err := c.CombinedOutput()
		if err != nil {
			return &Result{
				Gate:    g.name,
				Status:  StatusFailed,
				Message: fmt.Sprintf("%s: %s", err, string(out)),
			}
		}
	}

	return &Result{
		Gate:    g.name,
		Status:  StatusPassed,
		Message: fmt.Sprintf("%s passed", g.name),
	}
}

func (g *LintGate) detectCommands(repoDir string) [][]string {
	switch g.name {
	case TypeLint:
		if commandExists("golangci-lint") && hasAnyFile(repoDir, ".golangci.yml", ".golangci.yaml") {
			return [][]string{{"golangci-lint", "run", "./..."}}
		}
		if commandExists("eslint") && hasESLintConfig(repoDir) {
			return [][]string{{"eslint", "."}}
		}
		if commandExists("ruff") && hasAnyFile(repoDir, "pyproject.toml", ".ruff.toml", "ruff.toml") {
			return [][]string{{"ruff", "check", "."}}
		}
	case TypeTypecheck:
		if commandExists("go") && hasAnyFile(repoDir, "go.mod") {
			return [][]string{{"go", "vet", "./..."}}
		}
		if commandExists("tsc") && hasAnyFile(repoDir, "tsconfig.json") {
			return [][]string{{"tsc", "--noEmit"}}
		}
		if commandExists("pyright") && hasAnyFile(repoDir, "pyproject.toml") {
			return [][]string{{"pyright"}}
		}
	case TypeTests:
		if commandExists("go") && hasAnyFile(repoDir, "go.mod") {
			return [][]string{{"go", "test", "./...", "-count=1"}}
		}
		if commandExists("npm") && hasAnyFile(repoDir, "package.json") {
			return [][]string{{"npm", "test"}}
		}
		if commandExists("pytest") && hasAnyFile(repoDir, "pyproject.toml") {
			return [][]string{{"python", "-m", "pytest"}}
		}
	}
	return nil
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func hasAnyFile(dir string, names ...string) bool {
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}

func hasESLintConfig(dir string) bool {
	if hasAnyFile(dir,
		".eslintrc", ".eslintrc.json", ".eslintrc.yaml", ".eslintrc.yml",
		".eslintrc.js", ".eslintrc.cjs", ".eslintrc.mjs") {
		return true
	}
	pkgPath := filepath.Join(dir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "\"eslintConfig\"")
}


