using kai.Models;

namespace kai.Core.Configuration;

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
