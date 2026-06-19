namespace kai.Core.Tools;

public sealed class ListDirTool : ITool
{
    private readonly PolicyEnforcer _policy;

    public ListDirTool(PolicyEnforcer policy)
    {
        _policy = policy;
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
            Console.Error.WriteLine(msg);
            return ToolResult.Fail(msg);
        }

        if (!string.IsNullOrWhiteSpace(relativePath) && !_policy.IsAllowedDir(relativePath, workingDirectory))
        {
            var msg = "Policy violation: path '" + relativePath + "' is not in allowed directories. Allowed dirs: " + string.Join(", ", _policy.AllowedDirs);
            Console.Error.WriteLine(msg);
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
                .ToList(), ct);

        return ToolResult.Ok(
            items.Count > 0
                ? string.Join("\n", items)
                : "(empty directory)");
    }
}
