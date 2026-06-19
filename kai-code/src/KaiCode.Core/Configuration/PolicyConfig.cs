using System.Text.Json.Serialization;

namespace kai.Core.Configuration;

public class PolicyConfig
{
    private List<string> _allowedTools = [];
    private List<string> _allowedCommands = [];
    private List<string> _allowedDirs = [];
    private string _agent = "";

    [JsonPropertyName("allowed_tools")]
    public List<string> AllowedTools { get => _allowedTools; set => _allowedTools = value ?? []; }

    [JsonPropertyName("allowed_commands")]
    public List<string> AllowedCommands { get => _allowedCommands; set => _allowedCommands = value ?? []; }

    [JsonPropertyName("allowed_dirs")]
    public List<string> AllowedDirs { get => _allowedDirs; set => _allowedDirs = value ?? []; }

    [JsonPropertyName("agent")]
    public string Agent { get => _agent; set => _agent = value ?? ""; }
}
