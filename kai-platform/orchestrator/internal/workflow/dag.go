package workflow

import (
	"fmt"
)

type DAG struct {
	Pipeline  *Pipeline
	stepIndex map[string]int
	edges     map[string][]string
}

func BuildDAG(p *Pipeline) (*DAG, error) {
	stepIndex := make(map[string]int, len(p.Steps))
	for i, s := range p.Steps {
		stepIndex[s.ID] = i
	}

	edges := make(map[string][]string)
	for _, s := range p.Steps {
		for _, dep := range s.DependsOn {
			if _, ok := stepIndex[dep]; !ok {
				return nil, fmt.Errorf("step %q depends on %q which does not exist", s.ID, dep)
			}
			edges[dep] = append(edges[dep], s.ID)
		}
	}

	if _, err := topoSort(p.Steps); err != nil {
		return nil, err
	}

	return &DAG{
		Pipeline:  p,
		stepIndex: stepIndex,
		edges:     edges,
	}, nil
}

func (d *DAG) Dependencies(id string) []string {
	for _, s := range d.Pipeline.Steps {
		if s.ID == id {
			return s.DependsOn
		}
	}
	return nil
}

func (d *DAG) Dependents(id string) []string {
	return d.edges[id]
}

func (d *DAG) ExecutionOrder() []string {
	order, _ := topoSort(d.Pipeline.Steps)
	return order
}

func topoSort(steps []Step) ([]string, error) {
	inDegree := make(map[string]int, len(steps))
	for _, s := range steps {
		inDegree[s.ID] = len(s.DependsOn)
	}

	queue := make([]string, 0)
	for _, s := range steps {
		if inDegree[s.ID] == 0 {
			queue = append(queue, s.ID)
		}
	}

	result := make([]string, 0, len(steps))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		result = append(result, id)

		for _, dependent := range dependentsOf(id, steps) {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(result) != len(steps) {
		return nil, fmt.Errorf("pipeline has a circular dependency")
	}

	return result, nil
}

func dependentsOf(id string, steps []Step) []string {
	var deps []string
	for _, s := range steps {
		for _, dep := range s.DependsOn {
			if dep == id {
				deps = append(deps, s.ID)
			}
		}
	}
	return deps
}

func (d *DAG) ReadySteps(state map[string]StepStatus) []string {
	var ready []string
	for _, s := range d.Pipeline.Steps {
		if state[s.ID] != StepPending {
			continue
		}
		allDepsPassed := true
		for _, dep := range s.DependsOn {
			if state[dep] != StepPassed {
				allDepsPassed = false
				break
			}
		}
		if allDepsPassed {
			ready = append(ready, s.ID)
		}
	}
	return ready
}
