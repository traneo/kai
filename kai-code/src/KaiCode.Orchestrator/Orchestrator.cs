using System.Security.Cryptography;
using System.Text;
using Microsoft.Extensions.Logging;
using kai.Abstractions;
using kai.Abstractions.Git;
using kai.Abstractions.Memory;
using kai.Core.Analysis;
using kai.Enums;
using kai.Core.Configuration;
using kai.Core.Memory;
using kai.Models;

namespace kai.Orchestrator;

public sealed class Orchestrator
{
    private readonly IGitService _git;
    private readonly ProjectAnalyzer _analyzer;
    private readonly IProjectMemory _memory;
    private readonly ILogger<Orchestrator> _logger;
    private readonly IEnumerable<IAgent> _agents;
    private readonly LimitsConfig _limits;

    public WorkflowState State { get; private set; } = WorkflowState.Idle;
    public Plan? CurrentPlan { get; private set; }

    public Orchestrator(
        IGitService git,
        ProjectAnalyzer analyzer,
        IProjectMemory memory,
        IEnumerable<IAgent> agents,
        ILogger<Orchestrator> logger,
        LimitsConfig limits)
    {
        _git = git;
        _analyzer = analyzer;
        _memory = memory;
        _agents = agents;
        _logger = logger;
        _limits = limits;
    }

    public async Task<AgentResult> RunAsync(string goal, string workingDirectory, CancellationToken ct = default)
    {
        _logger.LogInformation("Starting: {Goal}", goal);

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
            .Select(t => t.Goal)
            .ToList();

        var plan = new Plan
        {
            Goal = goal,
            Summary = goal,
            Tasks = [new TaskItem { Description = goal, FilePath = null }]
        };
        CurrentPlan = plan;

        var coder = _agents.FirstOrDefault(a =>
            a.Name.Equals("coder", StringComparison.OrdinalIgnoreCase));

        _logger.LogInformation("State transition: Idle → Coding");
        State = WorkflowState.Coding;

        if (coder is null)
        {
            return Fail("No coder agent available");
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
        catch (HttpRequestException ex)
        {
            _logger.LogError(ex, "LLM connection lost during coding phase");
            return Fail("LLM connection lost. Check that Ollama is running.");
        }
        catch (OperationCanceledException ex)
        {
            _logger.LogError(ex, "Coding phase timed out");
            return Fail("Coding phase timed out");
        }

        if (!coderResult.Success)
        {
            return Fail(coderResult.Error ?? "Coding failed");
        }

        plan.Tasks[0].Status = kai.Enums.TaskStatus.Completed;
        plan.Tasks[0].Result = coderResult.Message;

        if (coderResult.FilesChanged.Count > 0)
        {
            plan.Tasks = coderResult.FilesChanged.Select(f => new TaskItem
            {
                FilePath = f,
                Status = kai.Enums.TaskStatus.Pending
            }).ToList();

            try { _git.Commit(workingDirectory, "kai-code: " + goal); }
            catch (Exception ex) { _logger.LogWarning(ex, "Commit skipped (no changes to commit)"); }
        }

        _logger.LogInformation("Pipeline completed");
        State = WorkflowState.Completed;

        await SaveMemory(projectHash, projectMemory, plan, goal, success: true, ct);

        return AgentResult.Ok("Completed");
    }

    private AgentResult Fail(string message)
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
