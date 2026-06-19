using System.CommandLine;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using kai.Orchestrator;

namespace kai.Cli.Commands;

public static class PlanCommand
{
    public static void Configure(Command cmd)
    {
        var descriptionArg = new Argument<string>("description") { Description = "What you want to build" };
        cmd.Add(descriptionArg);

        var timeoutOpt = new Option<int?>("--timeout") { Description = "Pipeline timeout in seconds (default: no timeout)" };
        cmd.Add(timeoutOpt);

        cmd.SetAction(async (ParseResult ctx) =>
        {
            var description = ctx.GetValue(descriptionArg)!;
            var timeoutSecs = ctx.GetValue(timeoutOpt);

            var configPath = ServiceProviderFactory.FindConfigPath();
            var config = ServiceProviderFactory.LoadConfiguration(configPath);
            var services = ServiceProviderFactory.Create(config);

            var orchestrator = services.GetRequiredService<kai.Orchestrator.Orchestrator>();
            var logger = services.GetRequiredService<ILogger<Program>>();
            var workingDir = Directory.GetCurrentDirectory();

            using var cts = timeoutSecs.HasValue
                ? new CancellationTokenSource(TimeSpan.FromSeconds(timeoutSecs.Value))
                : new CancellationTokenSource();

            logger.LogInformation("Planning: {Goal}", description);

            var result = await orchestrator.RunAsync(description, workingDir, cts.Token);

            if (result.Success)
            {
                Console.WriteLine();
                Console.WriteLine($"  \u2713 {result.Message}");
            }
            else
            {
                Console.Error.WriteLine($"  \u2717 {result.Error}");
            }
        });
    }
}
