namespace kai.Core.Models;

public class AgentResult
{
    public bool Success { get; set; }
    public string Message { get; set; } = string.Empty;
    public List<string> FilesChanged { get; set; } = [];
    public string? BranchName { get; set; }
    public string? Error { get; set; }
    public List<ReviewIssue> Issues { get; set; } = [];

    public static AgentResult Ok(string message) => new() { Success = true, Message = message };
    public static AgentResult Fail(string error) => new() { Success = false, Error = error, Message = error };
}
