package coordinator

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
	kaipb "kaiplatform.com/gen/kaiplatform/v1"
	"kaiplatform.com/orchestrator/internal/audit"
	"kaiplatform.com/orchestrator/internal/validation"
	"kaiplatform.com/orchestrator/internal/workflow"
)

func (c *Coordinator) HandleMissionResult(ctx context.Context, runID, stepID, agentID string, result *kaipb.MissionResult, repoDir string) {
	c.mu.Lock()
	run, ok := c.runs[runID]
	c.mu.Unlock()

	if !ok {
		c.printf("run %s not found", runID)
		return
	}

	state, ok := run.StepStates[stepID]
	if !ok {
		c.printf("step %s not found in run %s", stepID, runID)
		return
	}

	c.logf("MISSION_RESULT run=%s step=%s agent=%s success=%v exit=%d", runID, stepID, agentID, result.Success, result.ExitCode)
	c.logf("MISSION_RESULT run=%s step=%s: validation gates=%v approval=%q", runID, stepID, state.Step.Validation, state.Step.Approval)

	c.printf("handling result for step %s (success=%v, exit=%d)", stepID, result.Success, result.ExitCode)

	if len(result.StateArchive) > 0 && c.archiveStore != nil {
		if err := c.archiveStore.Save(ctx, runID, stepID, result.StateArchive); err != nil {
			c.printf("save state archive: %v", err)
		}
	}

	if agentID != "" {
		c.pool.CompleteMission(agentID)
	}

	if !result.Success || result.ExitCode != 0 {
		run.CompleteStep(stepID, false, fmt.Sprintf("exit code: %d", result.ExitCode))
		c.auditLog(audit.EventStepFailed, runID, stepID, agentID, fmt.Sprintf("exit code: %d", result.ExitCode), nil)

		snap := run.Snapshot()
		stepState := snap.StepStates[stepID]
		if stepState.Status == workflow.StepPending {
			c.publish(map[string]any{
				"type": "step_retrying", "run_id": runID, "step_id": stepID,
				"retries": stepState.Retries, "max_retries": stepState.Step.Policy.MaxRetries,
				"error": fmt.Sprintf("exit code: %d", result.ExitCode),
			})
			c.scheduleRetry(run, stepID)
		} else {
			c.publish(map[string]any{
				"type": "step_failed", "run_id": runID, "step_id": stepID,
				"error": stepState.Error, "retries": stepState.Retries,
			})
		}

		c.logf("RUN %s step %s: agent failed (exit=%d), status=%v", runID, stepID, result.ExitCode, stepState.Status)
		c.printf("step %s failed, retries=%d/%d", stepID, stepState.Retries, stepState.Step.Policy.MaxRetries)
		c.saveRun(run)
		c.publishAfterStateChange(run)
		c.DrainQueue()
		return
	}

	run.CompleteStep(stepID, true, "")

	if repoDir == "" {
		if gc := c.GetGitClient(runID); gc != nil {
			repoDir = gc.RepoDir()
		}
	}

	valCtx := &validation.Context{
		Context:    ctx,
		ExitCode:   int(result.ExitCode),
		RepoDir:    repoDir,
		BranchName: agentID,
	}

	gates := state.Step.Validation
	c.logf("RUN %s step %s: running validation gates=%v", runID, stepID, gates)
	results := c.valRunner.Run(valCtx, gates)

	allPassed := c.valRunner.AllPassed(results)
	summary := c.valRunner.Summary(results)
	c.logf("RUN %s step %s: validation results: %s (allPassed=%v)", runID, stepID, summary, allPassed)

	c.printf("validation for step %s: %s", stepID, summary)

	run.SetGateResults(stepID, toGateResults(results))

	if allPassed {
		if gc := c.GetGitClient(runID); gc != nil {
			if err := gc.Pull(ctx); err != nil {
				c.printf("sync step %s changes: %v", stepID, err)
			}
		}

		needsApproval := state.Step.RequiresApproval()
		c.logf("RUN %s step %s: all gates passed, needsApproval=%v", runID, stepID, needsApproval)
		if needsApproval {
			if gc := c.GetGitClient(runID); gc != nil {
				beforeSHA := state.BeforeSHA
				var diff string
				var err error
				if beforeSHA != "" {
					diff, err = gc.GetDiffBetween(ctx, beforeSHA, "HEAD")
				} else {
					diff, err = gc.GetBranchDiff(ctx)
				}
				if err == nil {
					run.SetStepDiff(stepID, diff)
				} else {
					c.printf("get diff for step %s: %v", stepID, err)
				}
			}
			run.BlockStep(stepID, "awaiting human approval")
			c.auditLog(audit.EventStepBlocked, runID, stepID, agentID, "awaiting human approval", nil)
			c.publish(map[string]any{"type": "step_blocked", "run_id": runID, "step_id": stepID})
			c.logf("RUN %s step %s: blocked for approval", runID, stepID)
			c.printf("step %s passed validation, awaiting human approval", stepID)
			c.DrainQueue()
		} else {
			run.ValidateStep(stepID, true, "")
			c.auditLog(audit.EventStepCompleted, runID, stepID, agentID, "passed all gates", nil)
			c.publish(map[string]any{"type": "step_passed", "run_id": runID, "step_id": stepID})
			c.logf("RUN %s step %s: passed, dispatching next steps", runID, stepID)
			c.printf("step %s passed all gates", stepID)
			c.tryDispatchSteps(run)
			c.DrainQueue()
		}
	} else {
		failed := c.valRunner.FailedGates(results)
		run.ValidateStep(stepID, false, fmt.Sprintf("validation failed: %v", failed))
		c.auditLog(audit.EventStepFailed, runID, stepID, agentID, fmt.Sprintf("validation failed: %v", failed), nil)

		snap := run.Snapshot()
		stepState := snap.StepStates[stepID]
		if stepState.Status == workflow.StepPending {
			c.publish(map[string]any{
				"type": "step_retrying", "run_id": runID, "step_id": stepID,
				"retries": stepState.Retries, "max_retries": stepState.Step.Policy.MaxRetries,
				"error": fmt.Sprintf("validation failed: %v", failed),
			})
			c.scheduleRetry(run, stepID)
		} else {
			c.publish(map[string]any{
				"type": "step_failed", "run_id": runID, "step_id": stepID,
				"error":   fmt.Sprintf("validation failed: %v", failed),
				"retries": stepState.Retries,
			})
		}

		c.logf("RUN %s step %s: validation FAILED: %v", runID, stepID, failed)
		c.printf("step %s failed validation: %v", stepID, failed)
		c.DrainQueue()
	}

	c.saveRun(run)
	c.publishAfterStateChange(run)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

func toGateResults(results []*validation.Result) []workflow.GateResult {
	gates := make([]workflow.GateResult, 0, len(results))
	for _, r := range results {
		gates = append(gates, workflow.GateResult{
			Gate:     string(r.Gate),
			Status:   string(r.Status),
			Message:  r.Message,
			Duration: r.Duration.String(),
		})
	}
	return gates
}

func (c *Coordinator) AuditLogApprove(runID, stepID, message string) {
	c.auditLog(audit.EventApprovalGranted, runID, stepID, "", message, nil)
}

func (c *Coordinator) AuditLogReject(runID, stepID, message string) {
	c.auditLog(audit.EventApprovalRejected, runID, stepID, "", message, nil)
}

func (c *Coordinator) CancelRun(id string) error {
	c.mu.Lock()
	run, ok := c.runs[id]
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("run %s not found", id)
	}
	if run.Status != workflow.PipelineRunning {
		return fmt.Errorf("run %s is not running (status: %s)", id, run.Status)
	}

	snap := run.Snapshot()
	for stepID, state := range snap.StepStates {
		if state.Status == workflow.StepRunning && state.AssignedTo != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			missionID := fmt.Sprintf("%s-%s", id, stepID)
			if err := c.pool.CancelMission(ctx, state.AssignedTo, missionID); err != nil {
				c.printf("cancel mission for step %s agent %s: %v", stepID, state.AssignedTo, err)
			}
			cancel()
		}
	}

	run.Cancel()
	c.saveRun(run)
	c.auditLog(audit.EventSystem, id, "", "", "pipeline cancelled", nil)
	c.publish(map[string]any{"type": "pipeline_cancelled", "run_id": id})
	return nil
}

func (c *Coordinator) RetryStep(runID, stepID string) error {
	c.mu.Lock()
	run, ok := c.runs[runID]
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("run %s not found", runID)
	}

	run.ResetStep(stepID)
	c.saveRun(run)
	c.auditLog(audit.EventSystem, runID, stepID, "", fmt.Sprintf("manual retry of step %s", stepID), nil)
	c.publish(map[string]any{"type": "step_retried", "run_id": runID, "step_id": stepID})
	c.tryDispatchSteps(run)
	return nil
}

func (c *Coordinator) RejectStep(runID, stepID, message string) {
	c.mu.Lock()
	run, ok := c.runs[runID]
	c.mu.Unlock()
	if !ok {
		return
	}

	run.ValidateStep(stepID, false, fmt.Sprintf("rejected by human: %s", message))
	c.saveRun(run)
	c.auditLog(audit.EventApprovalRejected, runID, stepID, "", message, nil)
	c.publish(map[string]any{"type": "step_rejected", "run_id": runID, "step_id": stepID, "message": message})
}

func (c *Coordinator) TryAgainStep(runID, stepID, feedback string) {
	c.mu.Lock()
	run, ok := c.runs[runID]
	c.mu.Unlock()
	if !ok {
		return
	}

	run.SetStepFeedback(stepID, feedback)
	run.ResetStep(stepID)
	c.saveRun(run)
	c.auditLog(audit.EventSystem, runID, stepID, "", fmt.Sprintf("try-again: %s", feedback), nil)
	c.publish(map[string]any{"type": "step_try_again", "run_id": runID, "step_id": stepID, "message": feedback})
	c.tryDispatchSteps(run)
}

func (c *Coordinator) publishAfterStateChange(run *workflow.Run) {
	switch run.Status {
	case workflow.PipelineCompleted:
		c.auditLog(audit.EventPipelineCompleted, run.ID, "", "", "pipeline completed", nil)
		c.publish(map[string]any{"type": "pipeline_completed", "run_id": run.ID})
	case workflow.PipelineFailed:
		c.auditLog(audit.EventPipelineFailed, run.ID, "", "", "pipeline failed", nil)
		c.publish(map[string]any{"type": "pipeline_failed", "run_id": run.ID})
	}
}

func entryFromLog(runID, stepID, missionID string, logEntry *kaipb.LogEntry) *ConversationEntry {
	t := time.Now()
	if logEntry.Timestamp != 0 {
		t = timestamppb.New(time.UnixMilli(logEntry.Timestamp)).AsTime()
	}
	return &ConversationEntry{
		MissionID: missionID,
		RunID:     runID,
		StepID:    stepID,
		Sequence:  logEntry.Sequence,
		Timestamp: t,
		Source:    logEntry.Source,
		Message:   logEntry.Message,
	}
}

func entryFromFileChange(runID, stepID, missionID string, fc *kaipb.FileChange) *ConversationEntry {
	return &ConversationEntry{
		MissionID: missionID,
		RunID:     runID,
		StepID:    stepID,
		Timestamp: time.Now(),
		Source:    "file_change",
		Message:   fmt.Sprintf("%s: %s", fc.Type.String(), fc.Path),
	}
}

func (c *Coordinator) StoreMissionEvent(missionID string, evt *kaipb.MissionEvent) {
	if c.convStore == nil {
		return
	}

	runID, stepID := c.resolveMissionRunStep(missionID)
	if runID == "" {
		return
	}

	var entry *ConversationEntry
	switch e := evt.Event.(type) {
	case *kaipb.MissionEvent_Log:
		entry = entryFromLog(runID, stepID, missionID, e.Log)
	case *kaipb.MissionEvent_FileChange:
		entry = entryFromFileChange(runID, stepID, missionID, e.FileChange)
	}

	if entry != nil {
		c.convStore.Append(entry)
		c.publish(map[string]any{
			"type":       "conversation",
			"run_id":     runID,
			"step_id":    stepID,
			"mission_id": missionID,
			"sequence":   entry.Sequence,
			"source":     entry.Source,
			"message":    entry.Message,
		})
	}
}

func (c *Coordinator) resolveMissionRunStep(missionID string) (runID, stepID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for id := range c.runs {
		prefix := id + "-"
		if strings.HasPrefix(missionID, prefix) {
			return id, strings.TrimPrefix(missionID, prefix)
		}
	}
	return "", ""
}
