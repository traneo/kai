using kai.Abstractions.Tools;
using kai.Models;
using Microsoft.Extensions.FileSystemGlobbing;
using Microsoft.Extensions.Logging;

namespace kai.Core.Tools;

public sealed class GlobTool : ITool
{
    private readonly PolicyEnforcer _policy;
    private readonly ILogger<GlobTool> _logger;

    public GlobTool(PolicyEnforcer policy, ILogger<GlobTool> logger)
    {
        _policy = policy;
        _logger = logger;
    }

    public string Name => "glob";
    public string Description => "Search for files matching a glob pattern. Supports ** (any directory depth), * (within a directory), and ? (single char). Args: the glob pattern (e.g. **/*.cs, src/**/*.ts).";

    public async Task<ToolResult> ExecuteAsync(string args, string workingDirectory, CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();
        var pattern = args.Trim();
        if (string.IsNullOrWhiteSpace(pattern))
            return ToolResult.Fail("No glob pattern provided");

        if (!_policy.IsAllowedTool("glob"))
        {
            var msg = "Policy violation: tool 'glob' is not allowed. Allowed tools: " + string.Join(", ", _policy.AllowedTools);
            _logger.LogWarning("{Msg}", msg);
            return ToolResult.Fail(msg);
        }

        var pathPart = pattern;
        if (pathPart.Contains('*') || pathPart.Contains('?'))
        {
            var starIdx = pathPart.IndexOfAny(['*', '?']);
            if (starIdx > 0)
                pathPart = pathPart[..(starIdx - 1)];
            else
                pathPart = "";
        }

        if (!string.IsNullOrWhiteSpace(pathPart) && !_policy.IsAllowedDir(pathPart, workingDirectory))
        {
            var msg = "Policy violation: path '" + pathPart + "' is not in allowed directories. Allowed dirs: " + string.Join(", ", _policy.AllowedDirs);
            _logger.LogWarning("{Msg}", msg);
            return ToolResult.Fail(msg);
        }

        try
        {
            var matcher = new Matcher();
            matcher.AddInclude(pattern);

            var files = await Task.Run(() =>
                matcher.GetResultsInFullPath(workingDirectory)
                    .Select(f => Path.GetRelativePath(workingDirectory, f))
                    .OrderBy(f => f)
                    .ToList(), ct);

            files = files.Where(f => !f.Contains("node_modules") && !f.Contains("bin/") && !f.Contains("obj/") && !f.Contains(".git/")).ToList();

            if (files.Count == 0)
                return ToolResult.Ok($"No files matching '{pattern}'");

            var grouped = files
                .GroupBy(f => Path.GetDirectoryName(f)?.Replace('\\', '/') ?? "")
                .OrderBy(g => g.Key)
                .Select(g =>
                {
                    var dir = string.IsNullOrEmpty(g.Key) ? "." : g.Key;
                    return $"  {dir}/: {string.Join(", ", g.Select(Path.GetFileName))}";
                });

            return ToolResult.Ok($"Found {files.Count} files:\n" + string.Join("\n", grouped));
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "Glob failed for pattern {Pattern}", pattern);
            return ToolResult.Fail($"Glob error: {ex.Message}");
        }
    }
}
