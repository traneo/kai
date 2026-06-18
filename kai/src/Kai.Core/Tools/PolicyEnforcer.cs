using kai.Core.Configuration;

namespace kai.Core.Tools;

public class PolicyEnforcer
{
    private readonly PolicyConfig _policy;

    public PolicyEnforcer(PolicyConfig policy)
    {
        _policy = policy;
    }

    public IReadOnlyList<string> AllowedTools => _policy.AllowedTools;
    public IReadOnlyList<string> AllowedCommands => _policy.AllowedCommands;
    public IReadOnlyList<string> AllowedDirs => _policy.AllowedDirs;
    public string Agent => _policy.Agent;

    public virtual bool IsAllowedTool(string tool)
    {
        foreach (var t in _policy.AllowedTools)
        {
            if (t == tool) return true;
        }
        return false;
    }

    public virtual bool IsAllowedCommand(string cmd)
    {
        foreach (var pattern in _policy.AllowedCommands)
        {
            if (MatchCommand(pattern, cmd)) return true;
        }
        return false;
    }

    public virtual bool IsAllowedDir(string path, string workingDirectory)
    {
        var fullPath = Path.GetFullPath(Path.Combine(workingDirectory, path));
        foreach (var allowed in _policy.AllowedDirs)
        {
            var allowedPath = Path.GetFullPath(Path.Combine(workingDirectory, allowed));
            if (fullPath.StartsWith(allowedPath, StringComparison.Ordinal))
                return true;
        }
        return false;
    }

    private static bool MatchCommand(string pattern, string cmd)
    {
        if (pattern.EndsWith('*'))
        {
            var prefix = pattern[..^1];
            return cmd == prefix.TrimEnd() || cmd.StartsWith(prefix, StringComparison.Ordinal);
        }
        return pattern == cmd;
    }
}
