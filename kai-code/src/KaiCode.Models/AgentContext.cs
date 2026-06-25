namespace kai.Models;

public class AgentContext
{
    public string Goal { get; set; } = string.Empty;
    public Plan? Plan { get; set; }
    public string WorkingDirectory { get; set; } = string.Empty;
    public Dictionary<string, object> State { get; set; } = [];
}
