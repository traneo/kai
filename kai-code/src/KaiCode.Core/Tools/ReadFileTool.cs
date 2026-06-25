using kai.Abstractions.Tools;
using kai.Models;
using Microsoft.Extensions.Logging;

namespace kai.Core.Tools;

public sealed class ReadFileTool : ITool
{
    private readonly PolicyEnforcer _policy;
    private readonly ILogger<ReadFileTool> _logger;

    public ReadFileTool(PolicyEnforcer policy, ILogger<ReadFileTool> logger)
    {
        _policy = policy;
        _logger = logger;
    }

    public string Name => "read_file";
    public string Description => "Read a file. Args: path. Optional: --lines N-M to read a range (1-based).";

    public async Task<ToolResult> ExecuteAsync(string args, string workingDirectory, CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();
        var (relativePath, lineRange) = ParseArgs(args);
        var path = Path.Combine(workingDirectory, relativePath);

        if (!_policy.IsAllowedTool("read_file"))
        {
            var msg = "Policy violation: tool 'read_file' is not allowed. Allowed tools: " + string.Join(", ", _policy.AllowedTools);
            _logger.LogWarning("{Msg}", msg);
            return ToolResult.Fail(msg);
        }

        if (!_policy.IsAllowedDir(relativePath, workingDirectory))
        {
            var msg = "Policy violation: path '" + relativePath + "' is not in allowed directories. Allowed dirs: " + string.Join(", ", _policy.AllowedDirs);
            _logger.LogWarning("{Msg}", msg);
            return ToolResult.Fail(msg);
        }

        if (!File.Exists(path))
            return ToolResult.Fail($"File not found: {relativePath}");

        var allLines = await File.ReadAllLinesAsync(path, ct);
        var totalLines = allLines.Length;

        if (lineRange is not null)
        {
            var (start, end) = lineRange.Value;
            start = Math.Max(0, start);
            end = Math.Min(totalLines, end);
            if (start >= totalLines || start >= end)
                return ToolResult.Fail($"Invalid range: lines {start + 1}-{end} (file has {totalLines} lines)");

            var selected = allLines[start..end];
            return ToolResult.Ok($"{relativePath} (lines {start + 1}-{end} of {totalLines}):\n{string.Join("\n", selected)}");
        }

        var header = $"{relativePath} ({totalLines} lines):\n";
        return ToolResult.Ok(header + string.Join("\n", allLines));
    }

    private static (string Path, (int Start, int End)? LineRange) ParseArgs(string args)
    {
        var trimmed = args.Trim();
        var linesIdx = trimmed.LastIndexOf(" --lines ");
        if (linesIdx < 0)
            return (trimmed, null);

        var path = trimmed[..linesIdx].Trim();
        var rangeStr = trimmed[(linesIdx + 9)..].Trim();
        var parts = rangeStr.Split('-');

        if (parts.Length != 2)
            return (trimmed, null);

        var hasStart = int.TryParse(parts[0], out var start);
        var hasEnd = int.TryParse(parts[1], out var end);

        if (!hasStart && !hasEnd)
            return (trimmed, null);

        if (!hasStart) start = 1;
        if (!hasEnd) end = int.MaxValue;

        return (path, (start > 0 ? start - 1 : 0, end));
    }
}
