namespace kai.Core.Configuration;

public class AgentLoopLimits
{
    public int MaxIterations { get; set; } = 50;
    public int CompressThreshold { get; set; } = 70;
    public int ToolOutputChars { get; set; } = 2000;
}
