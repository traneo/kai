using System.Collections.Concurrent;
using System.Text.RegularExpressions;
using kai.Core.Configuration;

namespace kai.Core.Tools;

public sealed class SearchTool : ITool
{
    private readonly PolicyEnforcer _policy;
    private readonly LimitsConfig _limits;

    public SearchTool(PolicyEnforcer policy, LimitsConfig limits)
    {
        _policy = policy;
        _limits = limits;
    }

    public string Name => "search";
    public string Description => "Search file contents for a regex pattern. Args: the regex pattern.";

    public Task<ToolResult> ExecuteAsync(string args, string workingDirectory, CancellationToken ct = default)
    {
        ct.ThrowIfCancellationRequested();
        var pattern = args.Trim();
        if (string.IsNullOrWhiteSpace(pattern))
            return Task.FromResult(ToolResult.Fail("No search pattern provided"));

        if (!_policy.IsAllowedTool("search"))
        {
            var msg = "Policy violation: tool 'search' is not allowed. Allowed tools: " + string.Join(", ", _policy.AllowedTools);
            Console.Error.WriteLine(msg);
            return Task.FromResult(ToolResult.Fail(msg));
        }

        try
        {
            var regex = new Regex(pattern, RegexOptions.Compiled);
            var results = new ConcurrentBag<string>();
            var maxSize = _limits.Output.SearchFileSizeBytes;
            var maxResults = _limits.Output.SearchResults;

            var files = Directory.EnumerateFiles(workingDirectory, "*", SearchOption.AllDirectories)
                .Where(f => !f.Contains("node_modules") && !f.Contains(".git") && !f.Contains("bin") && !f.Contains("obj"))
                .ToList();

            Parallel.ForEach(files, new ParallelOptions { CancellationToken = ct, MaxDegreeOfParallelism = 4 }, file =>
            {
                var fileInfo = new FileInfo(file);
                if (fileInfo.Length > maxSize)
                    return;

                var relPath = Path.GetRelativePath(workingDirectory, file);
                if (!_policy.IsAllowedDir(relPath, workingDirectory))
                    return;

                var content = File.ReadAllText(file);
                var matches = regex.Matches(content);
                if (matches.Count > 0)
                {
                    foreach (Match match in matches)
                    {
                        var line = GetLineNumber(content, match.Index);
                        results.Add($"{relPath}:{line}: {match.Value}");
                    }
                }
            });

            var sorted = results.OrderBy(r => r).ToList();

            if (sorted.Count == 0)
                return Task.FromResult(ToolResult.Ok($"No matches for '{pattern}'"));

            return Task.FromResult(ToolResult.Ok($"Found {sorted.Count} matches:\n" + string.Join("\n", sorted.Take(maxResults))));
        }
        catch (Exception ex)
        {
            return Task.FromResult(ToolResult.Fail($"Search error: {ex.Message}"));
        }
    }

    private static int GetLineNumber(string content, int index)
    {
        var lines = content[..index].Split('\n');
        return lines.Length;
    }
}
