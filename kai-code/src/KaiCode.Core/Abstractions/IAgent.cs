using kai.Core.Models;

namespace kai.Core.Abstractions;

public interface IAgent
{
    string Name { get; }
    Task<AgentResult> ExecuteAsync(AgentContext context, CancellationToken ct = default);
}

public class AgentContext
{
    public string Goal { get; set; } = string.Empty;
    public Plan? Plan { get; set; }
    public string WorkingDirectory { get; set; } = string.Empty;
    public Dictionary<string, object> State { get; set; } = [];
}
