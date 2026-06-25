using kai.Abstractions.Tools;
using kai.Models;
using Microsoft.Extensions.Logging;

namespace kai.Core.Tools;

public sealed class WriteFileTool : ITool
{
    private readonly PolicyEnforcer _policy;
    private readonly ILogger<WriteFileTool> _logger;
    private static readonly string[] _extensions = [".cs", ".json", ".csproj", ".slnx", ".xml", ".yaml", ".yml", ".md", ".txt", ".js", ".ts", ".css", ".html"];

    public WriteFileTool(PolicyEnforcer policy, ILogger<WriteFileTool> logger)
    {
        _policy = policy;
        _logger = logger;
    }

    public string Name => "write_file";
    public string Description => "Write content to a file (creates directories as needed). Args: file path, newline, then file content.";

    public async Task<ToolResult> ExecuteAsync(string args, string workingDirectory, CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();
        var (filePath, content) = SplitPathAndContent(args);

        if (!_policy.IsAllowedTool("write_file"))
        {
            var msg = "Policy violation: tool 'write_file' is not allowed. Allowed tools: " + string.Join(", ", _policy.AllowedTools);
            _logger.LogWarning("{Msg}", msg);
            return ToolResult.Fail(msg);
        }

        if (!string.IsNullOrWhiteSpace(filePath) && !_policy.IsAllowedDir(filePath, workingDirectory))
        {
            var msg = "Policy violation: path '" + Truncate(filePath, 80) + "' is not in allowed directories. Allowed dirs: " + string.Join(", ", _policy.AllowedDirs);
            _logger.LogWarning("{Msg}", msg);
            return ToolResult.Fail(msg);
        }

        if (string.IsNullOrWhiteSpace(filePath) || filePath.Contains('<') || filePath.Contains('>') || filePath.Contains('|'))
            return ToolResult.Fail($"Invalid file path: '{Truncate(filePath, 80)}'. Must be a simple path like 'src/File.cs'.");

        var fullPath = Path.Combine(workingDirectory, filePath);

        if (Directory.Exists(fullPath))
            return ToolResult.Fail($"Cannot write: '{filePath}' is a directory. Remove it first: run | rm -rf '{filePath}'");

        var dir = Path.GetDirectoryName(fullPath);
        if (dir is not null) Directory.CreateDirectory(dir);

        content = StripCodeFences(content, out var stripped);
        await File.WriteAllTextAsync(fullPath, content, ct);
        if (stripped)
        {
            _logger.LogInformation("Removed markdown code fences from {FilePath}", filePath);
            return ToolResult.Ok($"Written {content.Length} bytes to {filePath} (removed markdown fences)");
        }
        return ToolResult.Ok($"Written {content.Length} bytes to {filePath}");
    }

    private static (string Path, string Content) SplitPathAndContent(string args)
    {
        var newlineIdx = args.IndexOf('\n');
        if (newlineIdx < 0)
        {
            var text = args.Trim();
            foreach (var ext in _extensions)
            {
                var extIdx = text.IndexOf(ext, StringComparison.OrdinalIgnoreCase);
                if (extIdx < 0) continue;
                var endIdx = extIdx + ext.Length;
                if (endIdx >= text.Length) continue;
                var rest = text[endIdx..].TrimStart();
                if (rest.Length > 0)
                    return (text[..endIdx], rest);
            }
            return (text, "");
        }

        var pathPart = args[..newlineIdx].Trim();
        var content = args[(newlineIdx + 1)..];

        var fenceStart = content.IndexOf("```");
        if (fenceStart >= 0)
        {
            var afterFence = content[(fenceStart + 3)..].TrimStart();
            var langNl = afterFence.IndexOf('\n');
            if (langNl > 0)
                afterFence = afterFence[(langNl + 1)..];

            var fenceEnd = afterFence.LastIndexOf("```");
            content = fenceEnd >= 0
                ? afterFence[..fenceEnd].TrimEnd()
                : afterFence;
        }

        return (pathPart, content);
    }

    private static string StripCodeFences(string content, out bool wasStripped)
    {
        var trimmed = content.AsSpan().TrimStart();
        wasStripped = false;

        if (trimmed.StartsWith("```"))
        {
            var firstNl = trimmed.IndexOf('\n');
            if (firstNl > 0)
                trimmed = trimmed[(firstNl + 1)..];
            wasStripped = true;
        }

        trimmed = trimmed.TrimEnd();
        if (trimmed.EndsWith("```"))
        {
            trimmed = trimmed[..^3];
            wasStripped = true;
        }

        return wasStripped ? trimmed.ToString() : content;
    }

    private static string Truncate(string text, int max) =>
        text.Length <= max ? text : text[..max] + "...";
}
