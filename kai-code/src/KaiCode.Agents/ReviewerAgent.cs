using System.Diagnostics;
using Microsoft.Extensions.Logging;
using kai.Core.Abstractions;
using kai.Core.Analysis;
using kai.Core.Configuration;
using kai.Core.Memory;
using kai.Core.Models;
using kai.Git;
using kai.LLM;

namespace kai.Agents;

public sealed partial class ReviewerAgent : IAgent
{
    private readonly IChatCompletion _chat;
    private readonly ProjectAnalyzer _analyzer;
    private readonly IGitService _git;
    private readonly ILogger<ReviewerAgent> _logger;
    private readonly ModelOptions? _modelOptions;

    public string Name => "reviewer";

    public ReviewerAgent(IChatCompletion chat, ProjectAnalyzer analyzer, IGitService git, ILogger<ReviewerAgent> logger, kaiConfig config)
    {
        _chat = chat;
        _analyzer = analyzer;
        _git = git;
        _logger = logger;
        _modelOptions = config.Agents.GetValueOrDefault("reviewer")?.ToModelOptions();
    }

    public async Task<AgentResult> ExecuteAsync(AgentContext context, CancellationToken ct = default)
    {
        _logger.LogInformation("ReviewerAgent started");
        var projectInfo = _analyzer.Analyze(context.WorkingDirectory);
        var diff = _git.GetDiff(context.WorkingDirectory);

        if (string.IsNullOrWhiteSpace(diff))
        {
            _logger.LogInformation("No diff to review");
            return AgentResult.Ok("No changes to review");
        }

        _logger.LogInformation("Reviewing diff ({Length} bytes), running build...", diff.Length);

        var buildOutput = await RunBuildAsync(context.WorkingDirectory, projectInfo.BuildCommand, ct);
        var buildPassed = string.IsNullOrEmpty(buildOutput) || IsBuildSuccessful(buildOutput);

        var conventions = ExtractConventions(context, projectInfo);
        var comment = projectInfo.Language.LineComment;

        var prompt = $"""
{comment} You are a code reviewer for {projectInfo.Language.Name} ({projectInfo.ProjectType}).
{comment} Project conventions: {conventions}

Review the following diff and build output.

Diff:
```diff
{diff}
```

Build output:
```
{buildOutput}
```

For each issue found, output on a separate line using this exact format:
ISSUE|severity|file|line|message|suggestion

Severity must be one of: error, warning, suggestion
File must be the file path from the diff
Line is the approximate line number (or 0 if unknown)
Message is a short description of the issue
Suggestion is how to fix it

Do NOT include preamble or summary. Only output ISSUE lines. If no issues, output: NO_ISSUES
""";

        var reviewText = await _chat.CompleteAsync(
            "You are a thorough code reviewer. Output only ISSUE lines.",
            prompt,
            _modelOptions,
            ct);

        var review = ParseReview(reviewText, buildOutput, buildPassed);

        var errorCount = review.Issues.Count(i => i.Severity == ReviewSeverity.Error);
        var warningCount = review.Issues.Count - errorCount;
        var summary = review.Passed
            ? $"Review passed: {errorCount} errors, {warningCount} warnings"
            : $"Review failed: {errorCount} errors, {warningCount} warnings";

        _logger.LogInformation("{Summary}", summary);

        foreach (var issue in review.Issues.Where(i => i.Severity == ReviewSeverity.Error))
            _logger.LogError("  [{Sev}] {File}:{Line} {Msg}", issue.Severity, issue.File, issue.Line, issue.Message);

        foreach (var issue in review.Issues.Where(i => i.Severity != ReviewSeverity.Error))
            _logger.LogWarning("  [{Sev}] {File}:{Line} {Msg}", issue.Severity, issue.File, issue.Line, issue.Message);

        var result = AgentResult.Ok(summary);
        result.Issues = review.Issues;
        return result;
    }

    private static string ExtractConventions(AgentContext context, ProjectInfo projectInfo)
    {
        var parts = new List<string>(projectInfo.Conventions);

        if (context.State.TryGetValue("projectMemory", out var memObj) &&
            memObj is ProjectMemory mem)
        {
            if (mem.Rules.Count > 0)
                parts.InsertRange(0, mem.Rules);
        }

        return parts.Count > 0 ? string.Join("; ", parts) : "standard " + projectInfo.Language.Name + " conventions";
    }

    private static ReviewResult ParseReview(string text, string buildOutput, bool buildPassed)
    {
        var result = new ReviewResult
        {
            BuildOutput = buildOutput,
            BuildPassed = buildPassed
        };

        if (text.Trim().Equals("NO_ISSUES", StringComparison.OrdinalIgnoreCase))
        {
            result.Passed = buildPassed;
            result.Summary = buildPassed ? "No issues found" : "Build failed, no code issues detected";
            return result;
        }

        foreach (var line in text.Split('\n', StringSplitOptions.RemoveEmptyEntries))
        {
            var trimmed = line.Trim();
            if (!trimmed.StartsWith("ISSUE|")) continue;

            var parts = trimmed.Split('|', 6);
            if (parts.Length < 5) continue;

            var issue = new ReviewIssue
            {
                Severity = parts[1] switch
                {
                    "error" => ReviewSeverity.Error,
                    "warning" => ReviewSeverity.Warning,
                    _ => ReviewSeverity.Suggestion
                },
                File = parts[2],
                Message = parts.Length > 4 ? parts[4] : "",
                Suggestion = parts.Length > 5 ? parts[5] : null
            };

            if (int.TryParse(parts[3], out var lineNum))
                issue.Line = lineNum;

            result.Issues.Add(issue);
        }

        result.Passed = buildPassed && !result.Issues.Any(i => i.Severity == ReviewSeverity.Error);
        result.Summary = result.Passed
            ? $"Passed ({result.Issues.Count} issues)"
            : $"Failed ({result.Issues.Count} issues, build: {(buildPassed ? "ok" : "failed")})";

        return result;
    }

    private async Task<string> RunBuildAsync(string workingDirectory, string command, CancellationToken ct)
    {
        if (string.IsNullOrWhiteSpace(command)) return "";

        var parts = command.Split(' ', 2, StringSplitOptions.RemoveEmptyEntries);
        var psi = new ProcessStartInfo
        {
            FileName = parts[0],
            Arguments = parts.Length > 1 ? parts[1] : "",
            WorkingDirectory = workingDirectory,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false
        };

        try
        {
            using var process = new Process { StartInfo = psi };
            process.Start();

            var output = await process.StandardOutput.ReadToEndAsync(ct);
            var error = await process.StandardError.ReadToEndAsync(ct);
            await process.WaitForExitAsync(ct);

            return (output + "\n" + error).Trim();
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Build command failed");
            return $"Build command failed: {ex.Message}";
        }
    }

    private static bool IsBuildSuccessful(string output)
    {
        if (string.IsNullOrWhiteSpace(output)) return true;

        if (output.Contains("BUILD FAILED", StringComparison.OrdinalIgnoreCase)) return false;
        if (output.Contains("error CS", StringComparison.OrdinalIgnoreCase)) return false;
        if (output.Contains("error TS", StringComparison.OrdinalIgnoreCase)) return false;
        if (output.Contains("Failed", StringComparison.OrdinalIgnoreCase)) return false;

        return true;
    }
}
