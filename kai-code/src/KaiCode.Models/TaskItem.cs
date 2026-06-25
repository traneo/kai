namespace kai.Models;

public class TaskItem
{
    public string Id { get; set; } = Guid.NewGuid().ToString("N")[..8];
    public string Description { get; set; } = string.Empty;
    public string? FilePath { get; set; }
    public kai.Enums.TaskStatus Status { get; set; } = kai.Enums.TaskStatus.Pending;
    public List<string> Dependencies { get; set; } = [];
    public string? Result { get; set; }
}
