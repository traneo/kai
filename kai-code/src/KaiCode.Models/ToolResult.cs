namespace kai.Models;

public class ToolResult
{
    public bool Success { get; set; }
    public string Output { get; set; } = string.Empty;
    public string? Error { get; set; }

    public static ToolResult Ok(string output) => new() { Success = true, Output = output };
    public static ToolResult Fail(string error) => new() { Success = false, Error = error, Output = error };
}
