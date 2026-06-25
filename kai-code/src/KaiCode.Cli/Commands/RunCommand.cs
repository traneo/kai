using System.CommandLine;
using System.Text.Json;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using kai.Core.Configuration;

namespace kai.Cli.Commands;

public static class RunCommand
{
    public static void Configure(Command cmd)
    {
        var descriptionArg = new Argument<string>("description") { Description = "What you want to build", Arity = ArgumentArity.ZeroOrOne };
        cmd.Add(descriptionArg);

        var fileOpt = new Option<FileInfo?>("--file") { Description = "Read goal description from a markdown file instead of the description argument" };
        cmd.Add(fileOpt);

        var configOpt = new Option<FileInfo?>("--config") { Description = "Path to kai.json config file" };
        cmd.Add(configOpt);

        var policyOpt = new Option<FileInfo?>("--policy") { Description = "Path to policy.json file" };
        cmd.Add(policyOpt);

        var jsonOpt = new Option<bool>("--json") { Description = "Output structured JSON instead of human-readable text" };
        cmd.Add(jsonOpt);

        var timeoutOpt = new Option<int?>("--timeout") { Description = "Pipeline timeout in seconds (default: no timeout)" };
        cmd.Add(timeoutOpt);

        cmd.SetAction(async (ParseResult ctx) =>
        {
            var description = ctx.GetValue(descriptionArg);
            var file = ctx.GetValue(fileOpt);
            var configFile = ctx.GetValue(configOpt);
            var policyFile = ctx.GetValue(policyOpt);
            var json = ctx.GetValue(jsonOpt);
            var timeoutSecs = ctx.GetValue(timeoutOpt);

            if (file is not null)
            {
                if (!file.Exists)
                {
                    WriteError(json, $"File not found: {file.FullName}");
                    return;
                }
                description = await File.ReadAllTextAsync(file.FullName);
            }

            if (string.IsNullOrWhiteSpace(description))
            {
                WriteError(json, "Provide a description argument or use --file <path>");
                return;
            }

            if (configFile is null || !configFile.Exists)
            {
                WriteError(json, configFile is null ? "--config is required" : $"Config file not found: {configFile.FullName}");
                return;
            }

            if (policyFile is null || !policyFile.Exists)
            {
                WriteError(json, policyFile is null ? "--policy is required" : $"Policy file not found: {policyFile.FullName}");
                return;
            }

            var config = ServiceProviderFactory.LoadConfiguration(configFile.FullName);
            var policyConfig = JsonSerializer.Deserialize<PolicyConfig>(await File.ReadAllTextAsync(policyFile.FullName)) ?? new PolicyConfig();

            File.Delete(configFile.FullName);
            File.Delete(policyFile.FullName);

            var services = ServiceProviderFactory.Create(config, policyConfig);
            var orchestrator = services.GetRequiredService<kai.Orchestrator.Orchestrator>();
            var logger = services.GetRequiredService<ILogger<Program>>();

            var workingDir = Directory.GetCurrentDirectory();

            using var cts = timeoutSecs.HasValue
                ? new CancellationTokenSource(TimeSpan.FromSeconds(timeoutSecs.Value))
                : new CancellationTokenSource();

            if (!json)
            {
                Console.WriteLine();
                Console.WriteLine($"  kai \u2014 {description[..Math.Min(description.Length, 80)]}");
                Console.WriteLine();
            }

            var result = await orchestrator.RunAsync(description, workingDir, cts.Token);

            if (json)
            {
                var output = new
                {
                    success = result.Success,
                    message = result.Message,
                    error = result.Error
                };
                Console.WriteLine(JsonSerializer.Serialize(output, new JsonSerializerOptions { WriteIndented = true }));
            }
            else
            {
                Console.WriteLine();
                if (result.Success)
                {
                    Console.WriteLine($"  \u2713 {result.Message}");
                    Console.WriteLine();
                    Console.WriteLine($"  Next steps:");
                    Console.WriteLine($"    cd {workingDir}");
                    Console.WriteLine($"    git diff");
                    Console.WriteLine($"    # Review and merge when ready");
                }
                else
                {
                    Console.Error.WriteLine($"  \u2717 {result.Error}");
                }
                Console.WriteLine();
            }
        });
    }

    private static void WriteError(bool json, string msg)
    {
        if (json)
        {
            var err = new { success = false, error = msg };
            Console.Error.WriteLine(JsonSerializer.Serialize(err));
        }
        else
        {
            Console.Error.WriteLine($"  \u2717 {msg}");
        }
    }
}
