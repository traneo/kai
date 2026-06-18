package validation

import (
	"fmt"
	"os/exec"
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
		if commandExists("golangci-lint") {
			return [][]string{{"golangci-lint", "run", "./..."}}
		}
		if commandExists("eslint") {
			return [][]string{{"eslint", "."}}
		}
		if commandExists("ruff") {
			return [][]string{{"ruff", "check", "."}}
		}
	case TypeTypecheck:
		if commandExists("go") {
			return [][]string{{"go", "vet", "./..."}}
		}
		if commandExists("tsc") {
			return [][]string{{"tsc", "--noEmit"}}
		}
		if commandExists("pyright") {
			return [][]string{{"pyright"}}
		}
	case TypeTests:
		if commandExists("go") {
			return [][]string{{"go", "test", "./...", "-count=1"}}
		}
		if commandExists("npm") {
			return [][]string{{"npm", "test"}}
		}
		if commandExists("pytest") {
			return [][]string{{"python", "-m", "pytest"}}
		}
	}
	return nil
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
