namespace kai.Core.Models;

public enum ReviewSeverity
{
    Error,
    Warning,
    Suggestion
}

public class ReviewIssue
{
    public ReviewSeverity Severity { get; set; }
    public string File { get; set; } = string.Empty;
    public int? Line { get; set; }
    public string Message { get; set; } = string.Empty;
    public string? Suggestion { get; set; }
}

public class ReviewResult
{
    public bool Passed { get; set; }
    public List<ReviewIssue> Issues { get; set; } = [];
    public string Summary { get; set; } = string.Empty;
    public string BuildOutput { get; set; } = string.Empty;
    public bool BuildPassed { get; set; } = true;
}
