using kai.Core.Configuration;

namespace kai.Core.Tools;

public sealed class PermissivePolicy : PolicyEnforcer
{
    public PermissivePolicy() : base(new PolicyConfig()) { }

    public override bool IsAllowedTool(string tool) => true;
    public override bool IsAllowedCommand(string cmd) => true;
    public override bool IsAllowedDir(string path, string workingDirectory) => true;
}
