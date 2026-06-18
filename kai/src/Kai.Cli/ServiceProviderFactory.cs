// #define KAI_UNSAFE  // Uncomment to bypass ALL tool restrictions (debug only)

using Microsoft.Extensions.Configuration;
using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;
using kai.Core;
using kai.Core.Abstractions;
using kai.Core.Analysis;
using kai.Core.Configuration;
using kai.Core.Events;
using kai.Core.Gating;
using kai.Core.Language;
using kai.Core.Memory;
using kai.Core.Tools;
using kai.Git;
using kai.LLM;
using kai.Orchestrator;
using kai.Agents;
using System.Text.Json;

namespace kai.Cli;

public static class ServiceProviderFactory
{
    public static ServiceProvider Create(kaiConfig config, PolicyConfig? policy = null, IEventBus? eventBus = null, GateService? gateService = null)
    {
        var services = new ServiceCollection();

        Console.WriteLine(JsonSerializer.Serialize(config));

        services.AddSingleton(config);
        services.AddSingleton(config.Limits);

#if KAI_UNSAFE
        services.AddSingleton<PolicyEnforcer>(new PermissivePolicy());
#else
        if (policy is not null)
        {
            services.AddSingleton(new PolicyEnforcer(policy));
        }
#endif

        services.AddLogging(builder =>
        {
            builder.ClearProviders();
            builder.AddConsole();
            builder.SetMinimumLevel(LogLevel.Information);
        });

        services.AddSingleton<IProjectMemory, JsonFileMemory>();
        services.AddSingleton<LanguageDetector>();
        services.AddSingleton(sp =>
        {
            var detector = sp.GetRequiredService<LanguageDetector>();
            return new ProjectAnalyzer(detector, config.Language, config.Limits);
        });
        services.AddSingleton<IChatCompletion, OpenAiChatCompletion>();
        services.AddSingleton<IGitService, GitService>();
        services.AddSingleton<kai.Orchestrator.Orchestrator>();

        services.AddSingleton<ITool, ReadFileTool>();
        services.AddSingleton<ITool, WriteFileTool>();
        services.AddSingleton<ITool, RunCommandTool>();
        services.AddSingleton<ITool, GlobTool>();
        services.AddSingleton<ITool, SearchTool>();
        services.AddTransient<IAgent, ToolCoderAgent>();
        services.AddTransient<IAgent, ReviewerAgent>();

        if (eventBus is not null)
            services.AddSingleton<IEventBus>(eventBus);
        else
            services.AddSingleton<IEventBus, InMemoryEventBus>();

        if (gateService is not null)
            services.AddSingleton(gateService);
        else
            services.AddSingleton<GateService>();

        return services.BuildServiceProvider();
    }

    public static kaiConfig LoadConfiguration(string configPath)
    {
        if (!File.Exists(configPath))
        {
            return new kaiConfig();
        }

        var configuration = new ConfigurationBuilder()
            .AddJsonFile(configPath, optional: false, reloadOnChange: false)
            .Build();

        var config = new kaiConfig();
        configuration.Bind(config);
        return config;
    }

    public static string FindConfigPath(string? startDir = null)
    {
        var dir = startDir ?? Directory.GetCurrentDirectory();
        var configFile = Path.Combine(dir, "kai.json");

        if (File.Exists(configFile)) return configFile;

        var parent = Directory.GetParent(dir);
        return parent is not null ? FindConfigPath(parent.FullName) : configFile;
    }
}
