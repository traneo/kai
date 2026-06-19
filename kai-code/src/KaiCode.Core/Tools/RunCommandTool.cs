using System.Diagnostics;

namespace kai.Core.Tools;

public sealed class RunCommandTool : ITool
{
    private readonly PolicyEnforcer _policy;

    public RunCommandTool(PolicyEnforcer policy)
    {
        _policy = policy;
    }

    public string Name => "run";
    public string Description => "Execute a shell command in the project directory. Args: the full command string.";

    public async Task<ToolResult> ExecuteAsync(string args, string workingDirectory, CancellationToken ct = default)
    {
        var command = args.Trim();
        if (string.IsNullOrWhiteSpace(command))
            return ToolResult.Fail("No command provided");

        if (!_policy.IsAllowedTool("run"))
        {
            var msg = "Policy violation: tool 'run' is not allowed. Allowed tools: " + string.Join(", ", _policy.AllowedTools);
            Console.Error.WriteLine(msg);
            return ToolResult.Fail(msg);
        }

        if (!_policy.IsAllowedCommand(command))
        {
            var msg = "Policy violation: command '" + Truncate(command, 80) + "' is not in allowed list. Allowed commands: " + string.Join(", ", _policy.AllowedCommands);
            Console.Error.WriteLine(msg);
            return ToolResult.Fail(msg);
        }

        var subCommands = command.Split("&&", StringSplitOptions.TrimEntries | StringSplitOptions.RemoveEmptyEntries)
            .Where(s => s.Length > 0)
            .ToList();

        if (subCommands.Count == 1)
            return await RunSingleCommandAsync(subCommands[0], workingDirectory, ct);

        Console.Error.WriteLine($"[run] Splitting chained command into {subCommands.Count} steps");
        var outputs = new List<string>();
        for (var i = 0; i < subCommands.Count; i++)
        {
            var sw = Stopwatch.StartNew();
            var result = await RunSingleCommandAsync(subCommands[i], workingDirectory, ct);
            sw.Stop();
            var status = result.Success ? "OK" : "FAIL";
            Console.Error.WriteLine($"[run] Step {i + 1}/{subCommands.Count}: {subCommands[i]} → {status} ({sw.ElapsedMilliseconds}ms)");
            outputs.Add($"--- Step {i + 1}/{subCommands.Count}: {subCommands[i]} ---\n{result.Output}");
            if (!result.Success)
                return ToolResult.Fail(string.Join("\n\n", outputs));
        }
        return ToolResult.Ok(string.Join("\n\n", outputs));
    }

    private static async Task<ToolResult> RunSingleCommandAsync(string command, string workingDirectory, CancellationToken ct)
    {
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
            ct.ThrowIfCancellationRequested();

            var output = await process.StandardOutput.ReadToEndAsync(ct);
            var error = await process.StandardError.ReadToEndAsync(ct);
            await process.WaitForExitAsync(ct);

            var result = (output + "\n" + error).Trim();
            return process.ExitCode == 0
                ? ToolResult.Ok(result)
                : ToolResult.Fail($"Exit code {process.ExitCode}:\n{result}");
        }
        catch (Exception ex)
        {
            return ToolResult.Fail($"Failed to run command: {ex.Message}");
        }
    }

    private static string Truncate(string text, int max) =>
        text.Length <= max ? text : text[..max] + "...";
}
