namespace kai.Models;

public class ProjectInfo
{
    public LanguageDefinition Language { get; set; } = Languages.Unknown;
    public List<string> DirectoryStructure { get; set; } = [];
    public List<SourceFileInfo> KeyFiles { get; set; } = [];
    public string? EntryPointContent { get; set; }
    public string BuildCommand { get; set; } = "";
}

public class SourceFileInfo
{
    public string RelativePath { get; set; } = "";
    public bool IsEntryPoint { get; set; }
    public string? Content { get; set; }
}
