using kai.Core.Language;

namespace kai.Core.Analysis;

public enum FileRole
{
    Unknown,
    EntryPoint,
    Controller,
    Service,
    Repository,
    Model,
    ViewModel,
    Dto,
    Configuration,
    Middleware,
    Test,
    Utility,
    Interface,
    Database,
    Event,
    Validator,
    Mapper,
    Extension,
    Constant,
    Enumeration
}

public record CodeSymbol(string Name, string Kind, string FilePath, int LineNumber, string? Signature = null);

public class ProjectInfo
{
    public LanguageDefinition Language { get; set; } = Languages.Unknown;
    public string ProjectType { get; set; } = "unknown";
    public string TargetFramework { get; set; } = "";
    public string RootNamespace { get; set; } = "";
    public List<string> Dependencies { get; set; } = [];
    public List<string> DirectoryStructure { get; set; } = [];
    public List<SourceFileInfo> KeyFiles { get; set; } = [];
    public string? EntryPointContent { get; set; }
    public List<string> Conventions { get; set; } = [];
    public string BuildCommand { get; set; } = "";
    public string TestCommand { get; set; } = "";
    public string TestFileSuffix { get; set; } = "_test";
    public string TestDirectory { get; set; } = "tests";
    public List<CodeSymbol> PublicTypes { get; set; } = [];
    public List<CodeSymbol> PublicMembers { get; set; } = [];
    public string? ProjectMap { get; set; }
}

public class SourceFileInfo
{
    public string RelativePath { get; set; } = "";
    public string? Namespace { get; set; }
    public bool IsEntryPoint { get; set; }
    public string? Content { get; set; }
    public FileRole Role { get; set; } = FileRole.Unknown;
    public List<string> Imports { get; set; } = [];
    public List<CodeSymbol> DeclaredTypes { get; set; } = [];
}
