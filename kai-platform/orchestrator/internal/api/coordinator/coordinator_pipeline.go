package coordinator

import (
	"context"
	"fmt"
	"time"

	kaipb "kaiplatform.com/gen/kaiplatform/v1"
	sdkgokit "kaiplatform.com/observability-sdk"
	"kaiplatform.com/orchestrator/internal/agentpool"
	"kaiplatform.com/orchestrator/internal/audit"
	"kaiplatform.com/orchestrator/internal/gitops"
	"kaiplatform.com/orchestrator/internal/gitprovider"
	"kaiplatform.com/orchestrator/internal/secrets"
	"kaiplatform.com/orchestrator/internal/workflow"
)

func (c *Coordinator) CheckNextSteps(run *workflow.Run) {
	c.tryDispatchSteps(run)
}

func (c *Coordinator) resolveGitToken(ctx context.Context, tokenRef string) string {
	if tokenRef != "" && c.secretStore != nil {
		if val, err := c.secretStore.GetValue(ctx, tokenRef); err == nil && val != "" {
			return val
		}
		c.printf("token_ref %q not found in secret store, falling back to env", tokenRef)
	}

	if c.secMgr != nil {
		s := secrets.Resolve(ctx, c.secMgr)
		if s.GitToken != "" {
			return s.GitToken
		}
	}

	return ""
}

func (c *Coordinator) SetupWorkspace(runID string, run *workflow.Run, p *workflow.Pipeline) {
	ctx := context.Background()

	if c.obsLogger != nil {
		c.obsLogger.WithRunID(runID).Info("setup workspace started",
			sdkgokit.F("repo", p.Repo.URL),
			sdkgokit.F("branch", p.Repo.BaseBranch),
		)
	}

	token := c.resolveGitToken(ctx, p.Repo.TokenRef)

	providerName := p.Repo.Provider
	if providerName == "" && c.gitProviderReg != nil {
		providerName = c.gitProviderReg.Detect(p.Repo.URL)
	}

	var prov gitprovider.Provider
	if providerName != "" && c.gitProviderReg != nil {
		prov, _ = c.gitProviderReg.Get(providerName)
	}

	if providerName != "" && prov == nil {
		c.printf("git provider %q not found for run %s (falling back to push-only)", providerName, runID)
	}

	gc := gitops.New(gitops.Config{
		RepoURL:      p.Repo.URL,
		BaseBranch:   p.Repo.BaseBranch,
		BranchPrefix: p.Output.BranchPrefix,
		RunID:        runID,
		Token:        token,
		GitProvider:  prov,
	})

	if err := gc.Clone(ctx, p.Repo.BaseBranch); err != nil {
		c.printf("clone repo for run %s: %v", runID, err)
		if c.obsLogger != nil {
			c.obsLogger.WithRunID(runID).Error("workspace setup failed",
				sdkgokit.F("phase", "clone"),
				sdkgokit.F("error", err.Error()),
			)
		}
		return
	}

	if err := gc.SetupGitConfig(ctx); err != nil {
		c.printf("git config for run %s: %v", runID, err)
		if c.obsLogger != nil {
			c.obsLogger.WithRunID(runID).Error("workspace setup failed",
				sdkgokit.F("phase", "git_config"),
				sdkgokit.F("error", err.Error()),
			)
		}
		return
	}

	if err := gc.PushInitialBranch(ctx); err != nil {
		c.printf("push initial branch for run %s: %v", runID, err)
		if c.obsLogger != nil {
			c.obsLogger.WithRunID(runID).Error("workspace setup failed",
				sdkgokit.F("phase", "push_branch"),
				sdkgokit.F("error", err.Error()),
			)
		}
		return
	}

	c.mu.Lock()
	c.gitClients[runID] = gc
	c.mu.Unlock()

	if c.obsLogger != nil {
		c.obsLogger.WithRunID(runID).Info("workspace setup complete",
			sdkgokit.F("repo_dir", gc.RepoDir()),
			sdkgokit.F("branch", gc.BranchName()),
		)
	}

	c.printf("workspace ready for run %s: repo=%s branch=%s", runID, gc.RepoDir(), gc.BranchName())

	c.tryDispatchSteps(run)
}

func (c *Coordinator) finishRun(run *workflow.Run) {
	c.mu.Lock()
	gc, ok := c.gitClients[run.ID]
	c.mu.Unlock()
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	for _, state := range run.StepStates {
		if state.Status != workflow.StepPassed {
			c.printf("run %s not all steps passed, skipping finalize", run.ID)
			return
		}
	}

	switch run.Pipeline.Output.Type {
	case "pr":
		var stepIDs []string
		for _, s := range run.Pipeline.Steps {
			stepIDs = append(stepIDs, s.ID)
		}
		title := gitops.GeneratePRTitle(run.Pipeline.Project, run.ID)
		body := gitops.GeneratePRBody(run.ID, stepIDs)
		result, err := gc.StageCommitPushPR(ctx, title, body, fmt.Sprintf("[%s] finalize", run.ID))
		if err != nil {
			c.printf("create PR for run %s: %v", run.ID, err)
			return
		}
		if result != nil {
			run.OutputURL = result.URL
			c.printf("PR created for run %s: %s (branch: %s)", run.ID, result.URL, result.Branch)
		}
		c.saveRun(run)

	default:
		hasChanges, err := gc.HasChanges(ctx)
		if err != nil {
			c.printf("check changes for run %s: %v", run.ID, err)
			return
		}
		if !hasChanges {
			c.printf("run %s: no changes to commit", run.ID)
			return
		}
		result, err := gc.StageCommitPush(ctx, fmt.Sprintf("[%s] completed", run.ID))
		if err != nil {
			c.printf("commit for run %s: %v", run.ID, err)
			return
		}
		run.OutputSHA = result.CommitSHA
		c.printf("committed for run %s (branch: %s, sha: %s)", run.ID, result.Branch, result.CommitSHA)
		c.saveRun(run)
	}
}

func (c *Coordinator) tryDispatchSteps(run *workflow.Run) {
	snap := run.Snapshot()

	if snap.Status == workflow.PipelineCompleted {
		c.printf("pipeline run %s completed successfully!", run.ID)
		if c.obsLogger != nil {
			c.obsLogger.WithRunID(run.ID).Info("pipeline completed")
		}
		c.finishRun(run)
		return
	}

	if snap.Status != workflow.PipelineRunning {
		c.logf("RUN %s: tryDispatchSteps skipped (status=%v)", run.ID, snap.Status)
		if c.obsLogger != nil {
			c.obsLogger.WithRunID(run.ID).Warn("tryDispatchSteps skipped",
				sdkgokit.F("status", string(snap.Status)),
			)
		}
		return
	}

	ready := run.NextReadySteps()
	c.logf("RUN %s: tryDispatchSteps ready=%v", run.ID, ready)

	if c.obsLogger != nil {
		c.obsLogger.WithRunID(run.ID).Info("dispatching steps",
			sdkgokit.F("ready_steps", ready),
		)
	}

	for _, id := range ready {
		if id != "" {
			c.printf("step %s is ready, looking for idle agent", id)
			c.dispatchStep(run, id)
		}
	}

	snap2 := run.Snapshot()
	if snap2.Status == workflow.PipelineCompleted {
		c.printf("pipeline run %s completed successfully!", run.ID)
		if c.obsLogger != nil {
			c.obsLogger.WithRunID(run.ID).Info("pipeline completed")
		}
		c.finishRun(run)
	}
}

func (c *Coordinator) scheduleRetry(run *workflow.Run, stepID string) {
	snap := run.Snapshot()
	state, ok := snap.StepStates[stepID]
	if !ok || state.NextRetryAt == nil {
		return
	}

	delay := time.Until(*state.NextRetryAt)
	if delay < 0 {
		delay = 0
	}

	c.printf("step %s: retrying in %v (attempt %d/%d)", stepID, delay, state.Retries, state.Step.Policy.MaxRetries)

	go func() {
		select {
		case <-time.After(delay):
			c.auditLog(audit.EventStepRetry, run.ID, stepID, "", fmt.Sprintf("retry attempt %d/%d", state.Retries, state.Step.Policy.MaxRetries), nil)
			c.publish(map[string]any{
				"type": "step_retry_ready", "run_id": run.ID, "step_id": stepID,
				"retries": state.Retries,
			})
			c.tryDispatchSteps(run)
		}
	}()
}

func (c *Coordinator) DrainQueue() {
	for {
		agentID, ok := c.pool.GetIdleAgent()
		if !ok {
			return
		}
		req := c.pool.DequeueMission()
		if req == nil {
			return
		}
		run := c.GetRun(req.RunID)
		if run == nil {
			c.logf("DRAIN: dequeued mission for run %s but run not found, discarding", req.RunID)
			continue
		}
		snap := run.Snapshot()
		state, ok := snap.StepStates[req.StepID]
		if !ok {
			c.logf("DRAIN: dequeued mission for run %s step %s but step not found, discarding", req.RunID, req.StepID)
			continue
		}
		if state.Status != workflow.StepPending {
			c.logf("DRAIN: dequeued mission for run %s step %s but status=%v (not pending), discarding", req.RunID, req.StepID, state.Status)
			continue
		}
		c.printf("drainQueue: dispatching queued step %s (run %s) to agent %s", req.StepID, req.RunID, agentID)
		c.logf("DRAIN: dispatching step %s run %s to agent %s", req.StepID, req.RunID, agentID)
		c.dispatchStepToAgent(run, req.StepID, agentID)
	}
}

func (c *Coordinator) dispatchStep(run *workflow.Run, stepID string) {
	snap := run.Snapshot()
	stepState := snap.StepStates[stepID]
	step := stepState.Step
	prompt := step.Prompt

	if stepState.Feedback != "" {
		prompt = fmt.Sprintf("%s\n\n---\nThe reviewer provided this feedback:\n%s", prompt, stepState.Feedback)
	}

	if step.Policy.Agent != "" {
		agentID := step.Policy.Agent
		rec := c.pool.GetAgent(agentID)
		if rec != nil && rec.State == agentpool.AgentIdle {
			c.dispatchStepToAgent(run, stepID, agentID)
			return
		}
		c.pool.EnqueueMission(&agentpool.MissionRequest{
			RunID:   run.ID,
			StepID:  stepID,
			Prompt:  prompt,
			AgentID: agentID,
		})
		if rec == nil {
			c.logf("DISPATCH run=%s step=%s: agent %q not found, enqueued", run.ID, stepID, agentID)
			c.printf("agent %q not found, queued step %s for run %s", agentID, stepID, run.ID)
		} else {
			c.logf("DISPATCH run=%s step=%s: agent %q busy, enqueued", run.ID, stepID, agentID)
			c.printf("agent %q busy, queued step %s for run %s", agentID, stepID, run.ID)
		}
		return
	}

	agentID, ok := c.pool.GetIdleAgent()
	if !ok {
		c.pool.EnqueueMission(&agentpool.MissionRequest{
			RunID:  run.ID,
			StepID: stepID,
			Prompt: prompt,
		})
		c.logf("DISPATCH run=%s step=%s: no idle agents, enqueued", run.ID, stepID)
		c.printf("no idle agents, queued step %s for run %s", stepID, run.ID)
		return
	}
	c.dispatchStepToAgent(run, stepID, agentID)
}

func (c *Coordinator) dispatchStepToAgent(run *workflow.Run, stepID, agentID string) {
	snap := run.Snapshot()
	state, ok := snap.StepStates[stepID]
	if !ok {
		c.logf("DISPATCH run=%s step=%s agent=%s: step not found", run.ID, stepID, agentID)
		return
	}
	c.logf("DISPATCH run=%s step=%s agent=%s: status=%v prompt=%q...", run.ID, stepID, agentID, state.Status, truncate(state.Step.Prompt, 80))

	if err := run.StartStep(stepID); err != nil {
		c.logf("DISPATCH run=%s step=%s: StartStep failed: %v", run.ID, stepID, err)
		c.printf("start step %s: %v", stepID, err)
		return
	}
	c.auditLog(audit.EventStepStarted, run.ID, stepID, agentID, "", nil)
	c.auditLog(audit.EventAgentAssigned, run.ID, stepID, agentID, fmt.Sprintf("assigned to agent %s", agentID), nil)
	c.publish(map[string]any{"type": "step_started", "run_id": run.ID, "step_id": stepID, "agent_id": agentID})

	missionID := fmt.Sprintf("%s-%s", run.ID, stepID)

	gc := c.GetGitClient(run.ID)
	if gc != nil {
		if sha, err := gc.GetCurrentCommit(context.Background()); err == nil {
			run.SetStepBeforeSHA(stepID, sha)
		}
	}
	c.saveRun(run)

	var ws *kaipb.Workspace
	if gc != nil {
		repoURL := run.Pipeline.Repo.URL
		if token := c.resolveGitToken(context.Background(), run.Pipeline.Repo.TokenRef); token != "" {
			repoURL = gitprovider.InjectToken(repoURL, token)
		}
		ws = &kaipb.Workspace{
			RepoUrl: repoURL,
			Branch:  gc.BranchName(),
		}
	}

	if err := c.server.AssignMissionToAgent(context.Background(), agentID, missionID, state.Step.Prompt, &state.Step.Policy, ws); err != nil {
		c.logf("DISPATCH run=%s step=%s: AssignMissionToAgent failed: %v", run.ID, stepID, err)
		c.printf("assign mission %s to agent %s: %v", missionID, agentID, err)
		run.CompleteStep(stepID, false, fmt.Sprintf("agent assignment failed: %v", err))
		return
	}

	c.printf("dispatched step %s (mission %s) to agent %s", stepID, missionID, agentID)
	c.logf("DISPATCH run=%s step=%s: dispatched to agent %s (mission=%s)", run.ID, stepID, agentID, missionID)
}
