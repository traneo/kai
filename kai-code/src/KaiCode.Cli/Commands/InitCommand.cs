using System.CommandLine;
using System.CommandLine.Parsing;
using System.Text.Json;
using kai.Core.Configuration;

namespace kai.Cli.Commands;

public static class InitCommand
{
    public static void Configure(Command cmd)
    {
        var forceOpt = new Option<bool>("--force") { Description = "Overwrite existing config" };
        cmd.Add(forceOpt);

        cmd.SetAction((ParseResult ctx) =>
        {
            var force = ctx.GetValue(forceOpt);
            var configPath = Path.Combine(Directory.GetCurrentDirectory(), "kai-code.json");

            if (File.Exists(configPath) && !force)
            {
                Console.Error.WriteLine("kai.json already exists. Use --force to overwrite.");
                return;
            }

            var config = new kaiConfig();
            var json = JsonSerializer.Serialize(config, new JsonSerializerOptions
            {
                WriteIndented = true,
                PropertyNamingPolicy = JsonNamingPolicy.CamelCase
            });

            File.WriteAllText(configPath, json);

            Console.WriteLine($"Created {configPath}");
            Console.WriteLine();
            Console.WriteLine("Edit the file to configure:");
            Console.WriteLine($"  language:  {config.Language ?? "(auto-detect)"} (C#, TypeScript, Python, Go, Rust, Java, or leave empty)");
            Console.WriteLine("  agents:   configure endpoint, model, apiKey for the coder agent");
        });
    }
}
