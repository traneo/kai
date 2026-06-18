namespace kai.Core.Events;

public interface IEventBus
{
    Task PublishAsync<T>(T @event) where T : kaiEvent;
    event Action<kaiEvent>? OnEvent;
    event Func<kaiEvent, CancellationToken, Task>? OnEventAsync;
}
