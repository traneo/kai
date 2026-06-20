using System.Threading.Channels;
using System.Net.Http.Json;
using System.Text.Json;

namespace KaiObservability.Sdk;

internal sealed class LogEntry
{
    public string Service { get; set; } = "";
    public string Level { get; set; } = "info";
    public string Message { get; set; } = "";
    public long Timestamp { get; set; }
    public string? RunId { get; set; }
    public string? StepId { get; set; }
    public string? MissionId { get; set; }
    public string? AgentId { get; set; }
    public Dictionary<string, object>? Metadata { get; set; }
}

internal sealed class BatchRequest
{
    public List<LogEntry> Entries { get; set; } = [];
}

public sealed class KaiObservabilityLogger : IDisposable
{
    private readonly string _endpoint;
    private readonly string _service;
    private readonly int _batchSize;
    private readonly TimeSpan _flushInterval;

    private readonly HttpClient _http;
    private readonly Channel<LogEntry> _queue;
    private readonly CancellationTokenSource _cts = new();
    private readonly Task _flushTask;
    private bool _disposed;

    public KaiObservabilityLogger(string endpoint, string service, int batchSize = 50, int queueCap = 10000, TimeSpan? flushInterval = null)
    {
        _endpoint = endpoint.TrimEnd('/');
        _service = service;
        _batchSize = batchSize;
        _flushInterval = flushInterval ?? TimeSpan.FromSeconds(1);

        _http = new HttpClient { Timeout = TimeSpan.FromSeconds(5) };
        _queue = Channel.CreateBounded<LogEntry>(new BoundedChannelOptions(queueCap)
        {
            FullMode = BoundedChannelFullMode.DropOldest
        });
        _flushTask = Task.Run(FlushLoopAsync);
    }

    public void Log(string level, string message, Dictionary<string, object>? metadata = null)
    {
        var entry = new LogEntry
        {
            Service = _service,
            Level = level,
            Message = message,
            Timestamp = DateTimeOffset.UtcNow.ToUnixTimeMilliseconds(),
            Metadata = metadata
        };
        _queue.Writer.TryWrite(entry);
    }

    private async Task FlushLoopAsync()
    {
        var timer = new PeriodicTimer(_flushInterval);
        var batch = new List<LogEntry>(_batchSize);

        while (await timer.WaitForNextTickAsync(_cts.Token).ConfigureAwait(false))
        {
            while (_queue.Reader.TryRead(out var entry))
            {
                batch.Add(entry);
                if (batch.Count >= _batchSize)
                {
                    await FlushBatchAsync(batch).ConfigureAwait(false);
                    batch.Clear();
                }
            }
            if (batch.Count > 0)
            {
                await FlushBatchAsync(batch).ConfigureAwait(false);
                batch.Clear();
            }
        }
    }

    private async Task FlushBatchAsync(List<LogEntry> entries)
    {
        try
        {
            var req = new BatchRequest { Entries = entries };
            var resp = await _http.PostAsJsonAsync($"{_endpoint}/api/v1/logs/batch", req, _cts.Token).ConfigureAwait(false);
            resp.Dispose();
        }
        catch
        {
            // silently drop — non-blocking contract
        }
    }

    public void Dispose()
    {
        if (_disposed) return;
        _disposed = true;
        _cts.Cancel();
        try { _flushTask.GetAwaiter().GetResult(); } catch { }
        _http.Dispose();
        _cts.Dispose();
    }
}
