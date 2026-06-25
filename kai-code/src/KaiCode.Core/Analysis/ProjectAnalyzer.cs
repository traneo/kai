using System.Text.RegularExpressions;
using kai.Core.Configuration;
using kai.Core.Language;
using kai.Models;

namespace kai.Core.Analysis;

public sealed partial class ProjectAnalyzer
{
    private readonly LanguageDetector _detector;
    private readonly LimitsConfig _limits;
    private readonly string? _languageOverride;

    public ProjectAnalyzer(LanguageDetector detector, string? languageOverride = null, LimitsConfig? limits = null)
    {
        _detector = detector;
        _languageOverride = languageOverride;
        _limits = limits ?? new LimitsConfig();
    }

    public ProjectInfo Analyze(string workingDirectory)
    {
        var lang = _languageOverride is not null
            ? Languages.FindByName(_languageOverride)
            : _detector.Detect(workingDirectory);

        var info = new ProjectInfo
        {
            Language = lang,
            BuildCommand = lang.DetectBuildCommand,
        };

        AnalyzeDirectory(workingDirectory, info);

        return info;
    }

    private void AnalyzeDirectory(string workingDirectory, ProjectInfo info)
    {
        var srcDir = Path.Combine(workingDirectory, "src");
        var baseDir = Directory.Exists(srcDir) ? srcDir : workingDirectory;

        var dirs = Directory.GetDirectories(baseDir, "*", SearchOption.AllDirectories)
            .Where(d => !d.Contains("node_modules") && !d.Contains("bin") && !d.Contains("obj") && !d.Contains(".git"))
            .Select(d => Path.GetRelativePath(baseDir, d))
            .OrderBy(d => d)
            .ToList();
        info.DirectoryStructure = dirs;

        var sourceFiles = _detector.GetSourceFiles(workingDirectory, info.Language);

        foreach (var file in sourceFiles)
        {
            var fullPath = Path.Combine(workingDirectory, file);
            var fileName = Path.GetFileName(file);

            var entryPointNames = new[] { "Program.cs", "main.go", "index.ts", "index.js", "main.py", "main.rs", "main.java", "App.tsx", "app.tsx" };
            var srcFile = new SourceFileInfo
            {
                RelativePath = file,
                IsEntryPoint = entryPointNames.Contains(fileName, StringComparer.OrdinalIgnoreCase) ||
                                fileName.StartsWith("main.", StringComparison.OrdinalIgnoreCase),
            };

            if (srcFile.IsEntryPoint && File.Exists(fullPath))
            {
                srcFile.Content = File.ReadAllText(fullPath);
                info.EntryPointContent = srcFile.Content;
            }

            info.KeyFiles.Add(srcFile);
        }
    }
}
