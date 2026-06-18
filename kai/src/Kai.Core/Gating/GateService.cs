using System.Collections.Concurrent;
using System.Threading.Channels;
using kai.Core.Configuration;

namespace kai.Core.Gating;

public sealed class GateService
{
    private readonly ConcurrentDictionary<Guid, Channel<GateResponse>> _gates = new();
    private readonly LimitsConfig _limits;

    public GateService(LimitsConfig limits)
    {
        _limits = limits;
    }

    public event Action<GateRequest>? OnGateRequested;

    public async Task<GateResponse> RequestAsync(GateRequest request, TimeSpan timeout, CancellationToken ct = default)
    {
        var channel = Channel.CreateBounded<GateResponse>(1);
        _gates[request.Id] = channel;

        OnGateRequested?.Invoke(request);

        using var timeoutCts = new CancellationTokenSource(timeout);
        using var linked = CancellationTokenSource.CreateLinkedTokenSource(ct, timeoutCts.Token);

        try
        {
            return await channel.Reader.ReadAsync(linked.Token);
        }
        catch (OperationCanceledException) when (timeoutCts.Token.IsCancellationRequested)
        {
            return new GateResponse(request.Id, null, "TIMEOUT");
        }
        finally
        {
            _gates.TryRemove(request.Id, out _);
        }
    }

    public Task<GateResponse> RequestAsync(GateRequest request, CancellationToken ct = default)
    {
        return RequestAsync(request, TimeSpan.FromMinutes(_limits.Retries.GateTimeoutMinutes), ct);
    }

    public Task RespondAsync(Guid requestId, GateResponse response)
    {
        if (_gates.TryGetValue(requestId, out var channel))
            channel.Writer.TryWrite(response);
        return Task.CompletedTask;
    }

    public GateRequest? TryGetActiveGate(Guid requestId)
    {
        return null;
    }

    public int ActiveGateCount => _gates.Count;
}
