using System.Diagnostics;
using Microsoft.Extensions.Logging;

namespace kai.Core.Testing;

public class TestResult
{
    public bool Passed { get; set; }
    public int Total { get; set; }
    public int PassedCount { get; set; }
    public int FailedCount { get; set; }
    public string Output { get; set; } = "";
    public List<string> FailedTests { get; set; } = [];
    public string? Error { get; set; }
}

public sealed class TestRunner
{
    private readonly ILogger<TestRunner> _logger;

    public TestRunner(ILogger<TestRunner> logger)
    {
        _logger = logger;
    }

    public async Task<TestResult> RunAsync(string workingDirectory, string command, CancellationToken ct = default)
    {
        var result = new TestResult();

        try
        {
            var parts = ParseCommand(command);
            var psi = new ProcessStartInfo
            {
                FileName = parts.FileName,
                Arguments = parts.Arguments,
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

            result.Output = output + "\n" + error;
            result.Passed = process.ExitCode == 0;

            ParseOutput(result, command);
        }
        catch (Exception ex)
        {
            _logger.LogError(ex, "Test execution failed: {Command}", command);
            result.Passed = false;
            result.Error = ex.Message;
        }

        return result;
    }

    private static void ParseOutput(TestResult result, string command)
    {
        if (command.Contains("dotnet test") || command.Contains("dotnet"))
        {
            ParseDotNetTestOutput(result);
        }
        else if (command.Contains("jest") || command.Contains("vitest"))
        {
            ParseJestOutput(result);
        }
        else if (command.Contains("pytest"))
        {
            ParsePytestOutput(result);
        }
        else if (command.Contains("go test"))
        {
            ParseGoTestOutput(result);
        }
        else if (command.Contains("cargo test"))
        {
            ParseCargoTestOutput(result);
        }
    }

    private static void ParseDotNetTestOutput(TestResult result)
    {
        foreach (var line in result.Output.Split('\n'))
        {
            if (line.Contains("passed") && line.Contains("failed"))
            {
                var passedMatch = System.Text.RegularExpressions.Regex.Match(line, @"(\d+)\s+passed");
                var failedMatch = System.Text.RegularExpressions.Regex.Match(line, @"(\d+)\s+failed");
                if (passedMatch.Success) result.PassedCount = int.Parse(passedMatch.Groups[1].Value);
                if (failedMatch.Success) result.FailedCount = int.Parse(failedMatch.Groups[1].Value);
            }
            if (line.TrimStart().StartsWith("Failed "))
                result.FailedTests.Add(line.Trim());
        }
        result.Total = result.PassedCount + result.FailedCount;
    }

    private static void ParseJestOutput(TestResult result)
    {
        var match = System.Text.RegularExpressions.Regex.Match(result.Output, @"Tests:\s+(\d+)\s+failed.*?(\d+)\s+passed");
        if (!match.Success)
            match = System.Text.RegularExpressions.Regex.Match(result.Output, @"Tests:\s+(\d+)\s+passed.*?(\d+)\s+failed");
        if (match.Success)
        {
            result.FailedCount = int.Parse(match.Groups[1].Value);
            result.PassedCount = int.Parse(match.Groups[2].Value);
            result.Total = result.PassedCount + result.FailedCount;
        }
    }

    private static void ParsePytestOutput(TestResult result)
    {
        var match = System.Text.RegularExpressions.Regex.Match(result.Output, @"(\d+)\s+passed.*?(\d+)\s+failed");
        if (!match.Success)
            match = System.Text.RegularExpressions.Regex.Match(result.Output, @"(\d+)\s+failed.*?(\d+)\s+passed");
        if (match.Success)
        {
            result.PassedCount = int.Parse(match.Groups[1].Value);
            result.FailedCount = int.Parse(match.Groups[2].Value);
            result.Total = result.PassedCount + result.FailedCount;
        }
        else
        {
            match = System.Text.RegularExpressions.Regex.Match(result.Output, @"(\d+)\s+passed");
            if (match.Success)
            {
                result.PassedCount = int.Parse(match.Groups[1].Value);
                result.Total = result.PassedCount;
            }
        }
    }

    private static void ParseGoTestOutput(TestResult result)
    {
        var match = System.Text.RegularExpressions.Regex.Match(result.Output, @"(FAIL|ok)\s+");
        result.Passed = !result.Output.Contains("FAIL");
    }

    private static void ParseCargoTestOutput(TestResult result)
    {
        var match = System.Text.RegularExpressions.Regex.Match(result.Output, @"(\d+)\s+passed.*?(\d+)\s+failed");
        if (match.Success)
        {
            result.PassedCount = int.Parse(match.Groups[1].Value);
            result.FailedCount = int.Parse(match.Groups[2].Value);
            result.Total = result.PassedCount + result.FailedCount;
        }
        result.Passed = result.FailedCount == 0;
    }

    private static (string FileName, string Arguments) ParseCommand(string command)
    {
        var parts = command.Split(' ', 2, StringSplitOptions.RemoveEmptyEntries);
        return (parts[0], parts.Length > 1 ? parts[1] : "");
    }
}
