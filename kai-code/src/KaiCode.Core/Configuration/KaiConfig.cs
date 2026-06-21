namespace kai.Core.Configuration;

public record ModelOptions(string? Model = null, double? Temperature = null, double? TopP = null, int? TopK = null, string? Endpoint = null, string? ApiKey = null, string? Provider = null);

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

public class AgentConfig
{
    public string? Endpoint { get; set; }
    public string? Model { get; set; }
    public string? ApiKey { get; set; }
    public string? Provider { get; set; }
    public double? Temperature { get; set; }
    public double? TopP { get; set; }
    public int? TopK { get; set; }

    public ModelOptions ToModelOptions() => new(
        Model: Model,
        Temperature: Temperature,
        TopP: TopP,
        TopK: TopK,
        Endpoint: Endpoint,
        ApiKey: ApiKey,
        Provider: Provider
    );
}