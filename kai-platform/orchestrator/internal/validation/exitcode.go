package validation

import "fmt"

type ExitCodeGate struct{}

func NewExitCodeGate() *ExitCodeGate {
	return &ExitCodeGate{}
}

func (g *ExitCodeGate) Name() Type {
	return TypeExitCode
}

func (g *ExitCodeGate) Run(ctx *Context) *Result {
	if ctx.ExitCode == 0 {
		return &Result{
			Gate:    TypeExitCode,
			Status:  StatusPassed,
			Message: "process exited with code 0",
		}
	}
	return &Result{
		Gate:   TypeExitCode,
		Status: StatusFailed,
		Message: fmt.Sprintf("process exited with code %d", ctx.ExitCode),
	}
}
