using System.Security.Cryptography;
using System.Text;
using Microsoft.Extensions.Logging;
using kai.Core.Abstractions;
using kai.Core.Analysis;
using kai.Core.Configuration;
using kai.Core.Events;
using kai.Core.Memory;
using kai.Core.Models;
using kai.Git;

namespace kai.Orchestrator;

public sealed class Orchestrator
{
    private readonly IGitService _git;
    private readonly ProjectAnalyzer _analyzer;
    private readonly IProjectMemory _memory;
    private readonly ILogger<Orchestrator> _logger;
    private readonly IEnumerable<IAgent> _agents;
    private readonly IEventBus _eventBus;
    private readonly LimitsConfig _limits;

    public WorkflowState State { get; private set; } = WorkflowState.Idle;
    public Plan? CurrentPlan { get; private set; }

    public Orchestrator(
        IGitService git,
        ProjectAnalyzer analyzer,
        IProjectMemory memory,
        IEnumerable<IAgent> agents,
        IEventBus eventBus,
        ILogger<Orchestrator> logger,
        LimitsConfig limits)
    {
        _git = git;
        _analyzer = analyzer;
        _memory = memory;
        _agents = agents;
        _eventBus = eventBus;
        _logger = logger;
        _limits = limits;
    }

    public async Task<AgentResult> RunAsync(string goal, string workingDirectory, CancellationToken ct = default)
    {
        _logger.LogInformation("Starting: {Goal}", goal);
        await _eventBus.PublishAsync(new PipelineStartedEvent(goal));

        if (!_git.IsRepository(workingDirectory))
        {
            State = WorkflowState.Failed;
            await _eventBus.PublishAsync(new PipelineCompletedEvent(goal, 0, false));
            return AgentResult.Fail("Not a git repository");
        }

        var projectHash = ComputeHash(workingDirectory);
        var projectMemory = await _memory.LoadAsync(projectHash, ct);

        if (projectMemory.TaskHistory.Count > 0)
        {
            _logger.LogInformation("Loaded memory: {Count} past sessions, {Rules} rules",
                projectMemory.SessionCount, projectMemory.Rules.Count);
        }

        var projectInfo = _analyzer.Analyze(workingDirectory);

        var recentGoalsList = projectMemory.TaskHistory
            .OrderByDescending(t => t.Timestamp)
            .Take(_limits.Output.RecentGoalsCount)
            .Select(t => t.Goal)
            .ToList();

        var plan = new Plan
        {
            Goal = goal,
            Summary = goal.Length > _limits.Output.GoalSummaryChars ? goal[.._limits.Output.GoalSummaryChars] + "..." : goal,
            Tasks = [new TaskItem { Description = goal, FilePath = null }]
        };
        CurrentPlan = plan;

        var coder = _agents.FirstOrDefault(a =>
            a.Name.Equals("coder", StringComparison.OrdinalIgnoreCase));

        // --- Coding phase: single pass with the full goal ---
        State = WorkflowState.Coding;
        await _eventBus.PublishAsync(new PhaseChangedEvent(0, goal, "coding"));

        if (coder is null)
        {
            return Fail(workingDirectory, "No coder agent available");
        }

        var codeContext = new AgentContext
        {
            Goal = goal,
            Plan = plan,
            WorkingDirectory = workingDirectory,
            State = new Dictionary<string, object>
            {
                ["currentTask"] = plan.Tasks[0],
                ["projectMemory"] = projectMemory,
                ["projectInfo"] = projectInfo,
                ["recentGoals"] = recentGoalsList
            }
        };

        AgentResult coderResult;
        try
        {
            coderResult = await coder.ExecuteAsync(codeContext, ct);
        }
        catch (HttpRequestException)
        {
            return Fail(workingDirectory, "LLM connection lost. Check that Ollama is running.");
        }
        catch (OperationCanceledException)
        {
            return Fail(workingDirectory, "Coding phase timed out");
        }

        if (!coderResult.Success)
        {
            await _eventBus.PublishAsync(new PipelineCompletedEvent(goal, 1, false));
            return Fail(workingDirectory, coderResult.Error ?? "Coding failed");
        }

        plan.Tasks[0].Status = Core.Models.TaskStatus.Completed;
        plan.Tasks[0].Result = coderResult.Message;

        if (coderResult.FilesChanged.Count > 0)
        {
            plan.Tasks = coderResult.FilesChanged.Select(f => new TaskItem
            {
                FilePath = f,
                Status = Core.Models.TaskStatus.Pending
            }).ToList();

            try { _git.Commit(workingDirectory, "kai-code: " + goal); }
            catch (Exception ex) { _logger.LogWarning(ex, "Commit skipped (no changes to commit)"); }
        }

        // --- Review phase ---
        var reviewer = _agents.FirstOrDefault(a =>
            a.Name.Equals("reviewer", StringComparison.OrdinalIgnoreCase));

        if (reviewer is not null)
        {
            State = WorkflowState.Reviewing;
            await _eventBus.PublishAsync(new PhaseChangedEvent(1, "Reviewing changes", "reviewing"));

            for (var fixAttempt = 0; fixAttempt <= _limits.Retries.ReviewFixAttempts; fixAttempt++)
            {
                _logger.LogInformation("Reviewing changes...");
                var reviewContext = new AgentContext
                {
                    Goal = goal,
                    Plan = plan,
                    WorkingDirectory = workingDirectory,
                    State = new Dictionary<string, object>
                    {
                        ["projectMemory"] = projectMemory,
                        ["projectInfo"] = projectInfo,
                        ["recentGoals"] = recentGoalsList
                    }
                };

                AgentResult reviewResult;
                try
                {
                    reviewResult = await reviewer.ExecuteAsync(reviewContext, ct);
                    _logger.LogInformation("Review: {Msg}", reviewResult.Message);
                }
                catch (Exception ex) when (ex is HttpRequestException or TaskCanceledException)
                {
                    _logger.LogWarning(ex, "Review skipped: LLM unavailable or timed out");
                    break;
                }
                catch (Exception ex)
                {
                    _logger.LogError(ex, "Review failed unexpectedly");
                    break;
                }

                var errors = reviewResult.Issues.Where(i => i.Severity == ReviewSeverity.Error).ToList();
                if (errors.Count == 0) break;

                if (coder is null || fixAttempt >= _limits.Retries.ReviewFixAttempts) break;

                _logger.LogInformation("Fixing {Count} review errors (attempt {Attempt})...", errors.Count, fixAttempt + 1);

                var fixDesc = string.Join("\n", errors.Select(e =>
                    $"  - `{e.File}` line {e.Line}: {e.Message} — {e.Suggestion}"));

                var fixTask = new TaskItem
                {
                    Description = $"Fix review errors:\n{fixDesc}",
                    FilePath = null
                };

                var fixContext = new AgentContext
                {
                    Goal = $"Fix issues found during code review: {fixDesc}",
                    Plan = plan,
                    WorkingDirectory = workingDirectory,
                    State = new Dictionary<string, object>
                    {
                        ["currentTask"] = fixTask,
                        ["projectMemory"] = projectMemory,
                        ["projectInfo"] = projectInfo,
                        ["recentGoals"] = recentGoalsList
                    }
                };

                try
                {
                    var fixResult = await coder.ExecuteAsync(fixContext, ct);
                    if (fixResult.FilesChanged.Count > 0)
                    {
                        _git.Commit(workingDirectory, $"kai-code: fix review issues (attempt {fixAttempt + 1})");
                    }
                }
                catch (Exception ex)
                {
                    _logger.LogError(ex, "Auto-fix failed");
                }
            }
        }

        State = WorkflowState.Completed;
        _logger.LogInformation("Pipeline completed");
        await _eventBus.PublishAsync(new PipelineCompletedEvent(goal, 1, true));

        await SaveMemory(projectHash, projectMemory, plan, goal, success: true, ct);

        return AgentResult.Ok("Completed");
    }

    private AgentResult Fail(string workingDirectory, string message)
    {
        _logger.LogError("Pipeline failed: {Msg}", message);
        State = WorkflowState.Failed;
        return AgentResult.Fail(message);
    }

    private async Task SaveMemory(string projectHash, ProjectMemory projectMemory, Plan plan, string goal, bool success, CancellationToken ct = default)
    {
        try
        {
            await _memory.AddTaskAsync(projectHash, new PastTask
            {
                Goal = goal,
                Summary = plan.Summary,
                TaskCount = 1,
                Success = success,
                BranchName = "current",
                Timestamp = DateTime.UtcNow
            }, ct);

            projectMemory.SessionCount++;
            await _memory.SaveAsync(projectMemory, ct);
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "Failed to save memory (non-critical)");
        }
    }

    private static string ComputeHash(string path)
    {
        var bytes = SHA256.HashData(Encoding.UTF8.GetBytes(path));
        return Convert.ToHexString(bytes)[..16].ToLowerInvariant();
    }
}
