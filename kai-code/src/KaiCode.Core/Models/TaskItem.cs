namespace kai.Core.Models;

public class TaskItem
{
    public string Id { get; set; } = Guid.NewGuid().ToString("N")[..8];
    public string Description { get; set; } = string.Empty;
    public string? FilePath { get; set; }
    public TaskStatus Status { get; set; } = TaskStatus.Pending;
    public List<string> Dependencies { get; set; } = [];
    public string? Result { get; set; }
}

public enum TaskStatus
{
    Pending,
    InProgress,
    Completed,
    Failed
}
