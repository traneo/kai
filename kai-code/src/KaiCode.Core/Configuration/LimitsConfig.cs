namespace kai.Core.Configuration;

public class LimitsConfig
{
    public AgentLoopLimits AgentLoop { get; set; } = new();
    public LlmLimits Llm { get; set; } = new();
    public DisplayLimits Display { get; set; } = new();
    public MemoryLimits Memory { get; set; } = new();
}
