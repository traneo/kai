using Microsoft.Extensions.Logging;

namespace KaiObservability.Sdk;

public sealed class KaiObservabilityLoggerProvider : ILoggerProvider
{
    private readonly KaiObservabilityLogger _logger;
    private bool _disposed;

    public KaiObservabilityLoggerProvider(string endpoint, string service)
    {
        _logger = new KaiObservabilityLogger(endpoint, service);
    }

    public ILogger CreateLogger(string categoryName)
    {
        return new KaiObservabilityILogger(_logger, categoryName);
    }

    public void Dispose()
    {
        if (_disposed) return;
        _disposed = true;
        _logger.Dispose();
    }
}

internal sealed class KaiObservabilityILogger : ILogger
{
    private readonly KaiObservabilityLogger _inner;
    private readonly string _category;

    private static readonly Dictionary<LogLevel, string> LevelMap = new()
    {
        [LogLevel.Trace] = "debug",
        [LogLevel.Debug] = "debug",
        [LogLevel.Information] = "info",
        [LogLevel.Warning] = "warn",
        [LogLevel.Error] = "error",
        [LogLevel.Critical] = "error",
    };

    public KaiObservabilityILogger(KaiObservabilityLogger inner, string category)
    {
        _inner = inner;
        _category = category;
    }

    public IDisposable? BeginScope<TState>(TState state) where TState : notnull => null;

    public bool IsEnabled(LogLevel logLevel) => logLevel >= LogLevel.Information;

    public void Log<TState>(LogLevel logLevel, EventId eventId, TState state, Exception? exception, Func<TState, Exception?, string> formatter)
    {
        if (!IsEnabled(logLevel)) return;

        var level = LevelMap.GetValueOrDefault(logLevel, "info");
        var message = formatter(state, exception);
        var metadata = new Dictionary<string, object>
        {
            ["category"] = _category,
        };
        if (exception != null)
        {
            metadata["exception"] = exception.ToString();
        }

        _inner.Log(level, message, metadata);
    }
}
