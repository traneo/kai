using kai.Models;

namespace kai.Abstractions;

public interface IAgent
{
    string Name { get; }
    Task<AgentResult> ExecuteAsync(AgentContext context, CancellationToken ct = default);
}
