using System.Threading.Channels;
using kai.Core.Events;

namespace kai.Core;

public sealed class InMemoryEventBus : IEventBus, IDisposable
{
    private readonly Channel<kaiEvent> _channel = Channel.CreateUnbounded<kaiEvent>(new UnboundedChannelOptions { SingleReader = true });
    private Func<kaiEvent, CancellationToken, Task>? _handlers;
    private readonly CancellationTokenSource _cts = new();
    private readonly Task _processor;

    public InMemoryEventBus()
    {
        _processor = ProcessEventsAsync(_cts.Token);
    }

    public event Action<kaiEvent>? OnEvent
    {
        add
        {
            if (value is not null)
                _handlers += (e, _) => { value(e); return Task.CompletedTask; };
        }
        remove
        {
            if (value is not null)
                _handlers -= (e, _) => { value(e); return Task.CompletedTask; };
        }
    }

    public event Func<kaiEvent, CancellationToken, Task>? OnEventAsync
    {
        add { if (value is not null) _handlers += value; }
        remove { if (value is not null) _handlers -= value; }
    }

    public async Task PublishAsync<T>(T @event) where T : kaiEvent
    {
        await _channel.Writer.WriteAsync(@event, _cts.Token);
    }

    private async Task ProcessEventsAsync(CancellationToken ct)
    {
        var reader = _channel.Reader;
        while (await reader.WaitToReadAsync(ct))
        {
            while (reader.TryRead(out var @event))
            {
                var handler = _handlers;
                if (handler is null) continue;

                var tasks = handler.GetInvocationList()
                    .Cast<Func<kaiEvent, CancellationToken, Task>>()
                    .Select(h => SafeInvokeAsync(h, @event))
                    .ToArray();

                await Task.WhenAll(tasks);
            }
        }
    }

    private static async Task SafeInvokeAsync(Func<kaiEvent, CancellationToken, Task> handler, kaiEvent @event)
    {
        try
        {
            await handler(@event, CancellationToken.None);
        }
        catch
        {
            // Logged by the handler itself; never crash the event bus
        }
    }

    public void Dispose()
    {
        _cts.Cancel();
        _channel.Writer.TryComplete();
        try { _processor.GetAwaiter().GetResult(); } catch { }
        _cts.Dispose();
    }
}
