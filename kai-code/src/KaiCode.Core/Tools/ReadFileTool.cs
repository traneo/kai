using kai.Core.Configuration;

namespace kai.Core.Tools;

public sealed class ReadFileTool : ITool
{
    private readonly PolicyEnforcer _policy;
    private readonly LimitsConfig _limits;

    public ReadFileTool(PolicyEnforcer policy, LimitsConfig limits)
    {
        _policy = policy;
        _limits = limits;
    }

    public string Name => "read_file";
    public string Description => "Read the contents of a file. Args: file path relative to project root.";

    public async Task<ToolResult> ExecuteAsync(string args, string workingDirectory, CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();
        var path = Path.Combine(workingDirectory, args.Trim());

        if (!_policy.IsAllowedTool("read_file"))
        {
            var msg = "Policy violation: tool 'read_file' is not allowed. Allowed tools: " + string.Join(", ", _policy.AllowedTools);
            Console.Error.WriteLine(msg);
            return ToolResult.Fail(msg);
        }

        var relativePath = args.Trim();
        if (!_policy.IsAllowedDir(relativePath, workingDirectory))
        {
            var msg = "Policy violation: path '" + relativePath + "' is not in allowed directories. Allowed dirs: " + string.Join(", ", _policy.AllowedDirs);
            Console.Error.WriteLine(msg);
            return ToolResult.Fail(msg);
        }

        if (!File.Exists(path))
            return ToolResult.Fail($"File not found: {relativePath}");

        var maxChars = _limits.AgentLoop.ReadFileOutputChars;
        var content = await File.ReadAllTextAsync(path, ct);
        var lines = content.Split('\n').Length;

        if (content.Length > maxChars)
        {
            var truncated = content[..maxChars];
            return ToolResult.Ok($"{relativePath} ({lines} lines, {content.Length} chars, showing first {maxChars}):\n{truncated}\n... (truncated, {content.Length - maxChars} more chars)");
        }

        return ToolResult.Ok($"{relativePath} ({lines} lines, {content.Length} chars):\n{content}");
    }
}
