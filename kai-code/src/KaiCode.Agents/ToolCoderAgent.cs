#define KAI_CW_DUMP   // Uncomment to dump context window to /tmp/kai-code.cw before each LLM call

using System.Diagnostics;
using Microsoft.Extensions.Logging;
using kai.Abstractions;
using kai.Abstractions.LLM;
using kai.Abstractions.Tools;
using kai.Core.Analysis;
using kai.Core.Compression;
using kai.Core.Configuration;
using kai.Core.Memory;
using kai.Core.Tools;
using kai.Models;

namespace kai.Agents;

public sealed class ToolCoderAgent : IAgent
{
    private readonly IChatCompletion _chat;
    private readonly ProjectAnalyzer _analyzer;
    private readonly IEnumerable<ITool> _tools;
    private readonly ILogger<ToolCoderAgent> _logger;
    private readonly int _maxContextTokens;
    private readonly ModelOptions? _modelOptions;
    private readonly LimitsConfig _limits;
    private readonly ContextCompressor _compressor;
    private readonly PolicyEnforcer _policy;

    public string Name => "coder";

    public ToolCoderAgent(
        IChatCompletion chat,
        ProjectAnalyzer analyzer,
        IEnumerable<ITool> tools,
        ILogger<ToolCoderAgent> logger,
        kaiConfig config,
        ContextCompressor compressor,
        PolicyEnforcer policy)
    {
        _chat = chat;
        _analyzer = analyzer;
        _tools = tools;
        _logger = logger;
        _maxContextTokens = config.MaxContextTokens;
        _limits = config.Limits;
        _compressor = compressor;
        _policy = policy;
        _modelOptions = config.Agents.GetValueOrDefault("coder")?.ToModelOptions();
    }

    public async Task<AgentResult> ExecuteAsync(AgentContext context, CancellationToken ct = default)
    {
        _logger.LogInformation("ToolCoderAgent started: {Goal}", context.Goal);
        if (!context.State.TryGetValue("currentTask", out var taskObj) || taskObj is not TaskItem task)
        {
            _logger.LogError("Agent abort: no task in context");
            return AgentResult.Fail("No task provided");
        }

        var projectInfo = context.State.TryGetValue("projectInfo", out var piObj) && piObj is ProjectInfo pi
            ? pi
            : _analyzer.Analyze(context.WorkingDirectory);
        var filteredTools = _tools.Where(t => _policy.IsAllowedTool(t.Name)).ToList();
        var toolsDesc = string.Join("\n", filteredTools.Select(t => $"  TOOL {t.Name}: {t.Description}"));

        var restrictionsNote = "";
        if (_policy.AllowedCommands.Count > 0)
            restrictionsNote += "\nAllowed commands: " + string.Join(", ", _policy.AllowedCommands);
        if (_policy.AllowedDirs.Count > 0)
            restrictionsNote += "\nAllowed directories: " + string.Join(", ", _policy.AllowedDirs);
        if (restrictionsNote.Length > 0)
            restrictionsNote = "\n\nRESTRICTIONS:" + restrictionsNote;

        var structure = projectInfo.DirectoryStructure.Count > 0
            ? "\nDirectory structure:\n" + string.Join("\n", projectInfo.DirectoryStructure.Select(d => $"  {d}/"))
            : "";

        var memorySection = BuildMemorySection(context);
        var existingProjects = BuildExistingProjectsList(context.WorkingDirectory);

        var systemPrompt = $"""
You are a senior software engineer working on a {projectInfo.Language.Name} project.
{structure}

You have access to these tools:

{toolsDesc}

Output format:
  TOOL: tool_name | arguments

For write_file, wrap content in a markdown code fence:
  TOOL: write_file | src/File.cs
  ```csharp
  public record Product(string Name, decimal Price);
  ```

For all other tools, everything after the pipe is the single-line argument:
  TOOL: run | go build ./...
  TOOL: read_file | src/main.go
  TOOL: glob | **/*.go
  TOOL: search | func main
{restrictionsNote}
CRITICAL RULES:
- One TOOL per response. Never append "DONE:" behind a TOOL line.
- When a tool returns SUCCESS, move forward — do NOT re-run the same command with different wrappers.
- glob and list_dir exclude node_modules/, bin/, obj/, and .git/ directories. Use `run | ls` to inspect those.
- Avoid re-running commands that already succeeded.
- Before scaffolding (go mod init, mkdir, etc.), verify the target doesn't already exist using `glob` or `list_dir`.
- When the task is complete, output only: DONE: summary of what was done
- Write_file content goes inside a markdown code fence (```). Only fenced content is written.

Guidance:
- 'read_file' before editing with write_file
- On FAILURE: do NOT retry the exact same command. Use diagnostic tools (glob, list_dir, read_file) to investigate the root cause, then try a corrected approach.
""";

        _logger.LogInformation("Tool coding: {Task}", task.Description);

        var messages = new List<(string Role, string Content)>
        {
            ("system", systemPrompt),
            ("user", $"""
Goal: {context.Goal}
{existingProjects}
{memorySection}

Use the available tools.
""")
        };

        var filesChanged = new List<string>();
        var readFileCache = new Dictionary<string, int>(StringComparer.OrdinalIgnoreCase);

        var fullContext = string.Join("\n\n", messages.Select(m => $"## {m.Role}\n{m.Content}\n"));
        var prevMessageCount = messages.Count;

        for (var i = 0; i < _limits.AgentLoop.MaxIterations; i++)
        {
            if (ct.IsCancellationRequested)
            {
                _logger.LogWarning("Agent cancelled via token at iteration {Iteration}", i);
                break;
            }

            if (messages.Count > prevMessageCount)
            {
                var newPart = string.Join("\n\n", messages.Skip(prevMessageCount)
                    .Select(m => $"## {m.Role}\n{m.Content}\n"));
                fullContext += "\n\n" + newPart;
            }
            else if (messages.Count < prevMessageCount)
            {
                fullContext = string.Join("\n\n", messages.Select(m => $"## {m.Role}\n{m.Content}\n"));
            }
            prevMessageCount = messages.Count;

            var estimatedTokens = fullContext.Length / 4;

            if (i > 0 && i % 5 == 0)
            {
                var pct = estimatedTokens * 100 / _maxContextTokens;
                _logger.LogInformation("Context: ~{EstTokens} tokens ({Pct}% of {MaxTokens}), {Pairs} pairs",
                    estimatedTokens, pct, _maxContextTokens, (messages.Count - 2) / 2);
            }

            if (estimatedTokens > _maxContextTokens * _limits.AgentLoop.CompressThreshold / 100)
            {
                try
                {
                    await _compressor.CompressAsync(messages, readFileCache, ct);
                    fullContext = string.Join("\n\n", messages.Select(m => $"## {m.Role}\n{m.Content}\n"));
                    prevMessageCount = messages.Count;
                    _logger.LogWarning("Context compressed to {NewTokens} tokens",
                        fullContext.Length / 4);
                }
                catch (Exception ex)
                {
                    _logger.LogError(ex, "Context compression failed at iteration {Iteration}", i);
                }
            }

#if KAI_CW_DUMP
            var cwPath = Environment.GetEnvironmentVariable("KAI_CW_DUMP") ?? "/tmp/kai-code.cw";
            File.WriteAllText(cwPath, fullContext);
#endif
            var response = await _chat.CompleteAsync(
                "You are a tool-using engineer. Be decisive. Output one TOOL: or DONE: per response. Never append DONE: after TOOL:. Never re-run a successful command.",
                fullContext,
                _modelOptions,
                ct);

            if (string.IsNullOrWhiteSpace(response))
            {
                _logger.LogWarning("LLM returned empty response at iteration {Iteration}", i);
                messages.Add(("assistant", ""));
                messages.Add(("user", "Empty response. Output TOOL: or DONE:."));
                continue;
            }

            var trimmed = response.Trim();

            if (trimmed.StartsWith("DONE:", StringComparison.OrdinalIgnoreCase))
            {
                var summary = trimmed[5..].Trim();
                projectInfo = _analyzer.Analyze(context.WorkingDirectory);

                if (!string.IsNullOrWhiteSpace(projectInfo.BuildCommand))
                {
                    var buildResult = await RunBuildAsync(context.WorkingDirectory, projectInfo.BuildCommand, ct);
                    if (!buildResult.Success)
                    {
                        _logger.LogWarning("Build failed at DONE, rejecting");
                        var errorOutput = buildResult.Output;
                        var changedList = filesChanged.Count > 0
                            ? "\nChanged files:\n" + string.Join("\n", filesChanged.Select(f => $"  - {f}"))
                            : "";
                        messages.Add(("assistant", trimmed));
                        messages.Add(("user",
                            $"BUILD FAILED\n" +
                            $"Build command: {projectInfo.BuildCommand}\n" +
                            $"Goal: {context.Goal}{changedList}\n" +
                            $"\n-- Build errors --\n{errorOutput}\n" +
                            $"\nFix the build errors. Use read_file to inspect each error file, then write_file to fix the code. Check the file syntax to ensure the code is correct for the language."));
                        continue;
                    }
                }

                _logger.LogInformation("Tool agent done: {Summary}", summary);
                var result = AgentResult.Ok(summary);
                result.FilesChanged = filesChanged;
                return result;
            }

            if (trimmed.StartsWith("TOOL:", StringComparison.OrdinalIgnoreCase))
            {
                var calls = ParseAllToolCalls(trimmed);

                if (calls.Count == 0)
                {
                    _logger.LogWarning("LLM response starts with TOOL but no valid calls parsed (iteration {Iteration})", i);
                    messages.Add(("assistant", trimmed));
                    messages.Add(("user", "Invalid format. Use: TOOL: name | args"));
                    continue;
                }

                var results = new List<(string Name, string Args, ToolResult Result)>();

                foreach (var (toolName, toolArgs) in calls)
                {
                    if (string.IsNullOrWhiteSpace(toolName))
                    {
                        results.Add((toolName, toolArgs, ToolResult.Fail("Invalid tool name")));
                        continue;
                    }

                    var tool = _tools.FirstOrDefault(t =>
                        t.Name.Equals(toolName, StringComparison.OrdinalIgnoreCase));

                    if (tool is null)
                    {
                        results.Add((toolName, toolArgs, ToolResult.Fail($"Unknown tool '{toolName}'")));
                        continue;
                    }

                    var sw = Stopwatch.StartNew();
                    _logger.LogInformation("Tool call: {Name} {Args}", toolName, Truncate(toolArgs, _limits.Display.LogChars));
                    var toolResult = await tool.ExecuteAsync(toolArgs, context.WorkingDirectory, ct);
                    sw.Stop();
                    results.Add((toolName, toolArgs, toolResult));

                    if (toolName.Equals("write_file", StringComparison.OrdinalIgnoreCase) && toolResult.Success)
                    {
                        var filePath = toolArgs.Split('\n', 2)[0].Trim();
                        if (!string.IsNullOrWhiteSpace(filePath) && !filesChanged.Contains(filePath))
                            filesChanged.Add(filePath);

                    }

                    var status = toolResult.Success ? "OK" : "FAIL";
                    _logger.LogInformation("Tool result: {Name} {Status} ({Elapsed}ms){Reason}", toolName, status, sw.ElapsedMilliseconds, toolResult.Success ? "" : $"\n  {Truncate(toolResult.Output, _limits.Display.LogChars)}");
                }

                var combinedOutput = string.Join("\n", results.Select(r =>
                {
                    var output = r.Result.Output;
                    var max = _limits.AgentLoop.ToolOutputChars;
                    string display;
                    if (output.Length > max && r.Name.Equals("read_file", StringComparison.OrdinalIgnoreCase))
                    {
                        var totalLines = output.Count(c => c == '\n');
                        display = output[..max] + $"\n... ({totalLines} lines total. Use --lines N-M to read specific sections.)";
                    }
                    else
                    {
                        display = output.Length > max ? output[..max] + "\n... (truncated)" : output;
                    }
                    return $"{(r.Result.Success ? "OK" : "FAIL")}\n{display}{(r.Result.Success ? "" : GetFailureGuidance(r.Name))}";
                }));

                var staleReadFiles = new List<(string Path, int OldIdx)>();
                foreach (var (name, args, _) in results)
                {
                    if (name.Equals("read_file", StringComparison.OrdinalIgnoreCase))
                    {
                        var path = args.Split('\n', 2)[0].Trim();
                        if (readFileCache.TryGetValue(path, out var oldIdx))
                            staleReadFiles.Add((path, oldIdx));
                    }
                }
                foreach (var (_, oldIdx) in staleReadFiles.OrderByDescending(x => x.OldIdx))
                {
                    messages.RemoveAt(oldIdx + 1);
                    messages.RemoveAt(oldIdx);
                    foreach (var k in readFileCache.Keys.Where(k => readFileCache[k] > oldIdx))
                        readFileCache[k] -= 2;
                }

                messages.Add(("assistant", trimmed));
                messages.Add(("user", combinedOutput));

                foreach (var (name, args, _) in results)
                {
                    if (name.Equals("read_file", StringComparison.OrdinalIgnoreCase))
                    {
                        var path = args.Split('\n', 2)[0].Trim();
                        readFileCache[path] = messages.Count - 2;
                    }
                }

                continue;
            }

            _logger.LogWarning("LLM response ignored: no TOOL or DONE found (iteration {Iteration}, length {Len})", i, trimmed.Length);
            messages.Add(("assistant", trimmed));
            messages.Add(("user", "Output must start with TOOL: or DONE:. Try again."));
        }

        _logger.LogWarning("Tool agent hit iteration limit ({Max})", _limits.AgentLoop.MaxIterations);
        var fallback = AgentResult.Fail($"Hit {_limits.AgentLoop.MaxIterations} iteration limit without completing the task");
        fallback.FilesChanged = filesChanged;
        return fallback;
    }

    private static List<(string Name, string Args)> ParseAllToolCalls(string text)
    {
        var calls = new List<(string Name, string Args)>();
        if (string.IsNullOrWhiteSpace(text)) return calls;

        var remaining = text.AsSpan();

        while (remaining.Length > 0)
        {
            remaining = remaining.TrimStart();
            if (remaining.Length == 0) break;

            if (!remaining.StartsWith("TOOL:", StringComparison.OrdinalIgnoreCase))
                break;

            remaining = remaining[5..];

            var pipeIdx = remaining.IndexOf('|');
            if (pipeIdx < 0) break;

            var name = remaining[..pipeIdx].ToString().Trim();
            remaining = remaining[(pipeIdx + 1)..];

            int endIdx;
            if (name.Equals("write_file", StringComparison.OrdinalIgnoreCase))
            {
                endIdx = FindNextBoundary(remaining);
            }
            else
            {
                var nlIdx = remaining.IndexOf('\n');
                endIdx = nlIdx >= 0 ? nlIdx : remaining.Length;
            }

            var args = remaining[..endIdx].ToString().TrimStart();
            remaining = remaining[endIdx..];

            calls.Add((name, StripDoneSuffix(args)));
        }

        return calls;
    }

    private static int FindNextBoundary(ReadOnlySpan<char> text)
    {
        for (int i = 0; i < text.Length; i++)
        {
            if (text[i] == '\n')
            {
                var rest = text[(i + 1)..];
                if (rest.StartsWith("TOOL:", StringComparison.OrdinalIgnoreCase) ||
                    rest.StartsWith("DONE:", StringComparison.OrdinalIgnoreCase))
                {
                    return i;
                }
            }
        }
        return text.Length;
    }

    private static (string Name, string Args) ParseSingleToolCall(string text)
    {
        var afterTool = text[5..].Trim();

        var pipeIdx = afterTool.IndexOf('|');
        var firstNewline = afterTool.IndexOf('\n');

        if (pipeIdx > 0)
        {
            var name = afterTool[..pipeIdx].Trim();
            var pipeArgs = afterTool[(pipeIdx + 1)..].TrimStart();

            if (name.Equals("write_file", StringComparison.OrdinalIgnoreCase) && firstNewline > pipeIdx)
            {
                var path = afterTool.Substring(pipeIdx + 1, firstNewline - pipeIdx - 1).Trim();
                var content = afterTool[(firstNewline + 1)..];
                return (name, StripDoneSuffix(path + "\n" + content));
            }

            return (name, StripDoneSuffix(pipeArgs));
        }

        if (firstNewline > 0)
        {
            var name = afterTool[..firstNewline].Trim();
            var args = afterTool[(firstNewline + 1)..].Trim();
            return (name, StripDoneSuffix(args));
        }

        var all = afterTool.Trim();
        var doneIdx = all.LastIndexOf(" DONE:", StringComparison.OrdinalIgnoreCase);
        return doneIdx > 0
            ? (all[..doneIdx].Trim(), "")
            : (all, "");
    }

    private static string GetFailureGuidance(string toolName) => toolName.ToLowerInvariant() switch
    {
        "run" => "\n\nThe command failed. Investigate before retrying: check if the command and prerequisites exist, verify paths are correct. Use `glob`, `list_dir`, or `read_file` to inspect the current state.",
        "read_file" => "\n\nFile not found. Use `glob` or `list_dir` to find the correct path before retrying.",
        "write_file" => "\n\nCould not write file. Ensure the parent directory exists (use `run | mkdir -p`) and the path is valid.",
        "glob" => "\n\nNo matches or invalid pattern. Verify the directory structure with `list_dir` or try a different glob pattern.",
        "search" => "\n\nSearch failed. Check the regex pattern or try a broader search.",
        "list_dir" => "\n\nDirectory not found. Use `list_dir` on a parent directory to find the correct path.",
        _ => "\n\nTool failed. Investigate the root cause with `glob`, `list_dir`, or `read_file` before retrying."
    };

    private static async Task<BuildResult> RunBuildAsync(string workingDirectory, string buildCommand, CancellationToken ct)
    {
        var parts = buildCommand.Split(' ', 2, StringSplitOptions.RemoveEmptyEntries);
        var psi = new ProcessStartInfo
        {
            FileName = parts[0],
            Arguments = parts.Length > 1 ? parts[1] : "",
            WorkingDirectory = workingDirectory,
            RedirectStandardOutput = true,
            RedirectStandardError = true,
            UseShellExecute = false
        };

        using var process = new Process { StartInfo = psi };
        process.Start();
        var output = await process.StandardOutput.ReadToEndAsync(ct);
        var error = await process.StandardError.ReadToEndAsync(ct);
        await process.WaitForExitAsync(ct);

        return new BuildResult
        {
            Success = process.ExitCode == 0,
            Output = output + "\n" + error
        };
    }

    private class BuildResult
    {
        public bool Success { get; set; }
        public string Output { get; set; } = "";
    }

    private static string StripDoneSuffix(string args)
    {
        var doneIdx = args.LastIndexOf(" DONE:", StringComparison.OrdinalIgnoreCase);
        if (doneIdx < 0) return args;

        var lineBeforeDone = args[..doneIdx].TrimEnd();
        if (lineBeforeDone.EndsWith(';') || lineBeforeDone.EndsWith('&'))
            lineBeforeDone += " true";

        return lineBeforeDone;
    }

    private string BuildExistingProjectsList(string workingDir)
    {
        try
        {
            var dirs = Directory.GetDirectories(workingDir)
                .Select(d => Path.GetFileName(d)!)
                .Where(d => !d.StartsWith('.') && !d.StartsWith("bin") && !d.StartsWith("obj"))
                .ToList();
            return dirs.Count > 0
                ? "\nExisting projects on disk: " + string.Join(", ", dirs)
                : "";
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "Failed to list projects in {Dir}", workingDir);
            return "";
        }
    }

    private static string BuildMemorySection(AgentContext context)
    {
        var parts = new List<string>();

        if (context.State.TryGetValue("projectMemory", out var memObj) &&
            memObj is ProjectMemory mem)
        {
            if (mem.Rules.Count > 0)
                parts.Add($"Project rules: {string.Join(", ", mem.Rules)}");
        }

        if (context.State.TryGetValue("recentGoals", out var goalsObj) &&
            goalsObj is List<string> goals && goals.Count > 0)
        {
            parts.Add("Recent work: " + string.Join(" → ", goals));
        }

        return parts.Count > 0 ? "\n" + string.Join("\n", parts) : "";
    }

    private static string Truncate(string text, int max) =>
        text.Length <= max ? text : text[..max] + "\n... (truncated)";


}
