using System.Text.Json;
using kai.Core.Configuration;

namespace kai.Core.Memory;

public sealed class JsonFileMemory : IProjectMemory
{
    private readonly string _basePath;
    private readonly LimitsConfig _limits;
    private static readonly JsonSerializerOptions JsonOptions = new()
    {
        WriteIndented = true,
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase
    };

    public JsonFileMemory(string? basePath = null, LimitsConfig? limits = null)
    {
        _basePath = basePath ?? Path.Combine(Environment.CurrentDirectory, ".kai");
        _limits = limits ?? new LimitsConfig();
        Directory.CreateDirectory(_basePath);
    }

    public async Task<ProjectMemory> LoadAsync(string projectHash, CancellationToken ct = default)
    {
        var path = GetMemoryPath(projectHash);
        if (!File.Exists(path)) return new ProjectMemory { ProjectHash = projectHash };

        try
        {
            var json = await File.ReadAllTextAsync(path, ct);
            return JsonSerializer.Deserialize<ProjectMemory>(json, JsonOptions) ?? new ProjectMemory { ProjectHash = projectHash };
        }
        catch
        {
            return new ProjectMemory { ProjectHash = projectHash };
        }
    }

    public async Task SaveAsync(ProjectMemory memory, CancellationToken ct = default)
    {
        var path = GetMemoryPath(memory.ProjectHash);
        memory.LastUpdated = DateTime.UtcNow;
        var json = JsonSerializer.Serialize(memory, JsonOptions);
        await File.WriteAllTextAsync(path, json, ct);
    }

    public async Task AddTaskAsync(string projectHash, PastTask task, CancellationToken ct = default)
    {
        var memory = await LoadAsync(projectHash, ct);
        memory.TaskHistory.Add(task);
        memory.SessionCount++;
        if (memory.TaskHistory.Count > _limits.Memory.MaxTaskHistoryEntries)
            memory.TaskHistory = memory.TaskHistory.TakeLast(_limits.Memory.MaxTaskHistoryEntries).ToList();
        await SaveAsync(memory, ct);
    }

    public async Task<string[]> GetRecentGoalsAsync(string projectHash, int count = 5, CancellationToken ct = default)
    {
        var memory = await LoadAsync(projectHash, ct);
        return memory.TaskHistory
            .OrderByDescending(t => t.Timestamp)
            .Take(count)
            .Select(t => t.Goal)
            .ToArray();
    }

    private string GetMemoryPath(string projectHash) =>
        Path.Combine(_basePath, $"{projectHash}.json");
}
