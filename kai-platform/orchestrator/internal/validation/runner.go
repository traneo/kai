package validation

import (
	"fmt"
	"strings"
)

type Runner struct {
	gates []Gate
}

func NewRunner() *Runner {
	return &Runner{}
}

func (r *Runner) Register(gate Gate) {
	r.gates = append(r.gates, gate)
}

func (r *Runner) RegisterAll(gates ...Gate) {
	r.gates = append(r.gates, gates...)
}

func (r *Runner) Run(ctx *Context, gateTypes []string) []*Result {
	gateMap := make(map[Type]Gate, len(r.gates))
	for _, g := range r.gates {
		gateMap[g.Name()] = g
	}

	var results []*Result
	for _, gt := range gateTypes {
		gate, ok := gateMap[Type(gt)]
		if !ok {
			results = append(results, &Result{
				Gate:    Type(gt),
				Status:  StatusSkipped,
				Message: fmt.Sprintf("gate %q not registered", gt),
			})
			continue
		}

		res := gate.Run(ctx)
		results = append(results, res)
	}

	return results
}

func (r *Runner) AllPassed(results []*Result) bool {
	for _, res := range results {
		if res.Failed() {
			return false
		}
	}
	return true
}

func (r *Runner) FailedGates(results []*Result) []string {
	var failed []string
	for _, res := range results {
		if res.Failed() {
			failed = append(failed, string(res.Gate))
		}
	}
	return failed
}

func (r *Runner) Summary(results []*Result) string {
	var parts []string
	for _, res := range results {
		parts = append(parts, fmt.Sprintf("%s=%s", res.Gate, res.Status))
	}
	return strings.Join(parts, ", ")
}
