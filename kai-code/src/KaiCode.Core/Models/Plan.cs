namespace kai.Core.Models;

public class Plan
{
    public string Id { get; set; } = Guid.NewGuid().ToString("N")[..8];
    public string Goal { get; set; } = string.Empty;
    public string Summary { get; set; } = string.Empty;
    public List<TaskItem> Tasks { get; set; } = [];
    public DateTime CreatedAt { get; set; } = DateTime.UtcNow;
}
