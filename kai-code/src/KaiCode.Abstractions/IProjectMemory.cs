using kai.Models;

namespace kai.Abstractions.Memory;

public interface IProjectMemory
{
    Task<ProjectMemory> LoadAsync(string projectHash, CancellationToken ct = default);
    Task SaveAsync(ProjectMemory memory, CancellationToken ct = default);
    Task AddTaskAsync(string projectHash, PastTask task, CancellationToken ct = default);
    Task<string[]> GetRecentGoalsAsync(string projectHash, int count = 5, CancellationToken ct = default);
}
