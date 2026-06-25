namespace kai.Models;

public class ProjectMemory
{
    public string ProjectHash { get; set; } = "";
    public string ProjectName { get; set; } = "";
    public string Language { get; set; } = "";
    public List<string> Conventions { get; set; } = [];
    public List<string> Rules { get; set; } = [];
    public List<PastTask> TaskHistory { get; set; } = [];
    public Dictionary<string, string> Preferences { get; set; } = [];
    public DateTime LastUpdated { get; set; } = DateTime.UtcNow;
    public int SessionCount { get; set; }
}

public class PastTask
{
    public string Goal { get; set; } = "";
    public string Summary { get; set; } = "";
    public int TaskCount { get; set; }
    public bool Success { get; set; }
    public string? BranchName { get; set; }
    public DateTime Timestamp { get; set; } = DateTime.UtcNow;
}
