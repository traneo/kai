using kai.Models;

namespace kai.Abstractions.Tools;

public interface ITool
{
    string Name { get; }
    string Description { get; }
    Task<ToolResult> ExecuteAsync(string args, string workingDirectory, CancellationToken ct = default);
}
