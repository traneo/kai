using Microsoft.Extensions.DependencyInjection;
using Microsoft.Extensions.Logging;

namespace KaiObservability.Sdk;

public static class ServiceCollectionExtensions
{
    public static ILoggingBuilder AddKaiObservability(this ILoggingBuilder builder, string endpoint, string service)
    {
        builder.Services.AddSingleton<ILoggerProvider>(sp =>
            new KaiObservabilityLoggerProvider(endpoint, service));
        return builder;
    }
}
