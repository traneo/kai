using kai.Abstractions.Tools;
using kai.Models;
using Microsoft.Extensions.Logging;

namespace kai.Core.Tools;

public sealed class ListDirTool : ITool
{
    private readonly PolicyEnforcer _policy;
    private readonly ILogger<ListDirTool> _logger;

    public ListDirTool(PolicyEnforcer policy, ILogger<ListDirTool> logger)
    {
        _policy = policy;
        _logger = logger;
    }

    public string Name => "list_dir";
    public string Description => "List files and directories in a path. Args: directory path relative to project root (or empty for root).";

    public async Task<ToolResult> ExecuteAsync(string args, string workingDirectory, CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();
        var relativePath = args.Trim();

        if (!_policy.IsAllowedTool("list_dir"))
        {
            var msg = "Policy violation: tool 'list_dir' is not allowed. Allowed tools: " + string.Join(", ", _policy.AllowedTools);
            _logger.LogWarning("{Msg}", msg);
            return ToolResult.Fail(msg);
        }

        if (!string.IsNullOrWhiteSpace(relativePath) && !_policy.IsAllowedDir(relativePath, workingDirectory))
        {
            var msg = "Policy violation: path '" + relativePath + "' is not in allowed directories. Allowed dirs: " + string.Join(", ", _policy.AllowedDirs);
            _logger.LogWarning("{Msg}", msg);
            return ToolResult.Fail(msg);
        }

        var dirPath = string.IsNullOrWhiteSpace(relativePath) ? workingDirectory : Path.Combine(workingDirectory, relativePath);

        if (!Directory.Exists(dirPath))
            return ToolResult.Fail($"Directory not found: {relativePath}");

        var items = await Task.Run(() =>
            Directory.GetFileSystemEntries(dirPath)
                .Select(f => new
                {
                    Name = Path.GetRelativePath(workingDirectory, f),
                    IsDir = Directory.Exists(f)
                })
                .OrderBy(f => f.IsDir ? 0 : 1)
                .ThenBy(f => f.Name)
                .Select(f => f.IsDir ? $"{f.Name}/" : f.Name)
                .Where(i => !i.Contains("node_modules") && !i.Contains("bin/") && !i.Contains("obj/") && !i.Contains(".git/"))
                .ToList(), ct);

        if (items.Count == 0)
            return ToolResult.Ok("(empty directory)");

        var dirs = items.Where(i => i.EndsWith('/')).ToList();
        var plainFiles = items.Where(i => !i.EndsWith('/')).ToList();

        var parts = new List<string>();
        if (dirs.Count > 0)
            parts.Add("dirs: " + string.Join(", ", dirs));
        if (plainFiles.Count > 0)
            parts.Add("files: " + string.Join(", ", plainFiles));

        return ToolResult.Ok(string.Join("\n", parts));
    }
}
