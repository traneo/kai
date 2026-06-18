namespace kai.Core.Configuration;

public class LimitsConfig
{
    public AgentLoopLimits AgentLoop { get; set; } = new();
    public RetryLimits Retries { get; set; } = new();
    public OutputLimits Output { get; set; } = new();
    public LlmLimits Llm { get; set; } = new();
    public DisplayLimits Display { get; set; } = new();
    public MemoryLimits Memory { get; set; } = new();
}

public class AgentLoopLimits
{
    public int MaxIterations { get; set; } = 50;
    public int MaxToolPairs { get; set; } = 25;
    public int CompressThreshold { get; set; } = 85;
    public int KeepLastPairs { get; set; } = 10;
    public int ReadFileOutputChars { get; set; } = 8000;
    public int ToolOutputChars { get; set; } = 300;
}

public class RetryLimits
{
    public int TestFixAttempts { get; set; } = 3;
    public int ReviewFixAttempts { get; set; } = 3;
    public int LlmApiRetries { get; set; } = 3;
    public List<int> LlmRetryDelaySeconds { get; set; } = [1, 3, 10];
    public int GateTimeoutMinutes { get; set; } = 5;
}

public class OutputLimits
{
    public int SearchResults { get; set; } = 50;
    public int SearchFileSizeBytes { get; set; } = 1_048_576;
    public int FilePathMaxChars { get; set; } = 200;
    public int TestOutputChars { get; set; } = 3000;
    public int GoalSummaryChars { get; set; } = 100;
    public int KeyFilesCount { get; set; } = 30;
    public int DependenciesCount { get; set; } = 15;
    public int RelatedFilesCount { get; set; } = 5;
    public int PreviewLines { get; set; } = 30;
    public int SourceFilesCount { get; set; } = 30;
    public int ConventionSamples { get; set; } = 5;
    public int RecentGoalsCount { get; set; } = 3;
}

public class LlmLimits
{
    public int MaxTokens { get; set; } = 4096;
}

public class DisplayLimits
{
    public int LogChars { get; set; } = 120;
    public int EventToolArgsChars { get; set; } = 200;
    public int EventOutputChars { get; set; } = 300;
    public int EventMessageChars { get; set; } = 100;
    public int SummaryToolsCount { get; set; } = 5;
    public int SummaryToolLineChars { get; set; } = 80;
}

public class MemoryLimits
{
    public int MaxTaskHistoryEntries { get; set; } = 50;
}
