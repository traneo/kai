package main

import (
	"flag"
	"fmt"
	"log"

	"kaiplatform.com/orchestrator/internal/workflow"
)

func main() {
	pipelinePath := flag.String("pipeline", "", "path to pipeline.yaml")
	httpPort := flag.String("http-port", getEnv("HTTP_PORT", "8080"), "HTTP API port")
	configServiceURL := flag.String("config-service-url", getEnv("CONFIG_SERVICE_URL", "http://localhost:8081"), "config service URL for proxy")
	flag.Parse()

	if *pipelinePath != "" {
		runPipeline(*pipelinePath)
		return
	}

	startServer(*httpPort, *configServiceURL)
}

func runPipeline(path string) {
	p, err := workflow.ParsePipeline(path)
	if err != nil {
		log.Fatalf("parse pipeline: %v", err)
	}

	run, err := workflow.NewRun("manual-1", p)
	if err != nil {
		log.Fatalf("create run: %v", err)
	}

	fmt.Printf("Pipeline: %s (%d steps)\n\n", p.Project, len(p.Steps))

	run.Start()
	stepNum := 1
	for run.Status == workflow.PipelineRunning {
		ready := run.NextReadySteps()
		if len(ready) == 0 {
			break
		}
		for _, id := range ready {
			state := run.StepStates[id]
			fmt.Printf("  Step %d: %s\n", stepNum, id)
			fmt.Printf("    Prompt: %s\n", state.Step.Prompt)
			if len(state.Step.DependsOn) > 0 {
				fmt.Printf("    Depends on: %v\n", state.Step.DependsOn)
			}
			if state.Step.RequiresApproval() {
				fmt.Printf("    Approval: required\n")
			}
			fmt.Printf("    Validation gates: %v\n", state.Step.Validation)
			fmt.Println()

			run.StartStep(id)
			run.CompleteStep(id, true, "")
			run.ValidateStep(id, true, "")
			stepNum++
		}
	}

	if run.Status == workflow.PipelineCompleted {
		fmt.Printf("Pipeline completed successfully.\n")
	} else {
		fmt.Printf("Pipeline failed: %s\n", run.Status)
	}
}
