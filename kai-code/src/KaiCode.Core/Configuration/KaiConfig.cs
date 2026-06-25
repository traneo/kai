namespace kai.Core.Configuration;

public class kaiConfig
{
    public string? Language { get; set; }
    public Dictionary<string, AgentConfig> Agents { get; set; } = [];
    public string BranchPrefix { get; set; } = "kai-code/";
    public bool AutoCommit { get; set; } = true;
    public bool AutoPush { get; set; } = false;
    public List<string> Rules { get; set; } = [];
    public int MaxContextTokens { get; set; } = 32768;
    public LimitsConfig Limits { get; set; } = new();
    public string? LogLevel { get; set; }
}
