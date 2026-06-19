using System.Text;
using System.Text.RegularExpressions;
using kai.Core.Configuration;
using kai.Core.Language;

namespace kai.Core.Analysis;

public partial class ProjectAnalyzer
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
            TestCommand = DetectTestCommand(workingDirectory, lang),
            TestFileSuffix = lang.TestFileSuffix,
            TestDirectory = lang.TestDirectory
        };

        AnalyzeProjectConfig(workingDirectory, info);
        AnalyzeDirectory(workingDirectory, info);
        AnalyzeConventions(workingDirectory, info);
        AnalyzeCodeSymbols(workingDirectory, info);
        ClassifyFiles(info);
        BuildProjectMap(info);

        return info;
    }

    public string GetRelatedFilesContext(string workingDirectory, string? targetFilePath, int maxFiles = 0)
    {
        var lang = _languageOverride is not null
            ? Languages.FindByName(_languageOverride)
            : _detector.Detect(workingDirectory);
        var files = new List<string>();
        var limit = maxFiles > 0 ? maxFiles : _limits.Output.RelatedFilesCount;

        if (targetFilePath is not null)
        {
            var targetDir = Path.GetDirectoryName(Path.Combine(workingDirectory, targetFilePath));
            if (targetDir is not null && Directory.Exists(targetDir))
            {
                files.AddRange(Directory.GetFiles(targetDir, "*", SearchOption.TopDirectoryOnly)
                    .Where(f => lang.FileExtensions.Any(ext =>
                        f.EndsWith(ext[1..], StringComparison.OrdinalIgnoreCase)))
                    .Take(limit));
            }
        }

        var result = new List<string>();
        foreach (var file in files)
        {
            var relPath = Path.GetRelativePath(workingDirectory, file);
            var content = File.ReadAllText(file);
            var lines = content.Split('\n');
            var preview = string.Join("\n", lines.Take(_limits.Output.PreviewLines));
            result.Add($"--- {relPath} ---\n{preview}");
        }

        return string.Join("\n\n", result);
    }

    private void AnalyzeProjectConfig(string workingDirectory, ProjectInfo info)
    {
        var lang = info.Language;

        foreach (var configPattern in lang.ConfigFiles)
        {
            var configFiles = Directory.GetFiles(workingDirectory, configPattern, SearchOption.AllDirectories);
            if (configFiles.Length == 0) continue;

            var content = File.ReadAllText(configFiles[0]);
            info.ProjectType = DetectProjectType(content, lang);

            if (lang == Languages.CSharp)
                ParseCsprojMetadata(content, info);

            if (lang == Languages.TypeScript || lang == Languages.JavaScript)
                ParsePackageJson(content, info);

            break;
        }
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

        foreach (var file in sourceFiles.Take(_limits.Output.SourceFilesCount))
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

    private void AnalyzeConventions(string workingDirectory, ProjectInfo info)
    {
        var lang = info.Language;
        var sourceFiles = _detector.GetSourceFiles(workingDirectory, lang);

        if (sourceFiles.Length == 0) return;

        var sampleContent = string.Join("\n",
            sourceFiles.Take(_limits.Output.ConventionSamples)
                .Select(f => File.ReadAllText(Path.Combine(workingDirectory, f))));

        if (lang == Languages.CSharp)
        {
            if (sampleContent.Contains("namespace ") && sampleContent.Contains(';'))
                info.Conventions.Add("file-scoped namespaces");
            if (sampleContent.Contains("/// <summary>"))
                info.Conventions.Add("XML documentation comments");
            if (sampleContent.Contains("[ApiController]"))
                info.Conventions.Add("ASP.NET API controllers");
            if (sampleContent.Contains("await ") || sampleContent.Contains("async "))
                info.Conventions.Add("async/await pattern");
        }

        if (lang == Languages.TypeScript || lang == Languages.JavaScript)
        {
            if (sampleContent.Contains("import ") || sampleContent.Contains("export "))
                info.Conventions.Add("ES modules");
            if (sampleContent.Contains("interface "))
                info.Conventions.Add("TypeScript interfaces");
            if (sampleContent.Contains("async ") || sampleContent.Contains("await "))
                info.Conventions.Add("async/await pattern");
        }

        if (lang == Languages.Python)
        {
            if (sampleContent.Contains("def ") && sampleContent.Contains(" -> "))
                info.Conventions.Add("type hints");
            if (sampleContent.Contains("class "))
                info.Conventions.Add("classes");
            if (sampleContent.Contains("async def"))
                info.Conventions.Add("async/await pattern");
        }

        if (lang == Languages.Go)
        {
            if (sampleContent.Contains("func ") && char.IsUpper(sampleContent[sampleContent.IndexOf("func ") + 5]))
                info.Conventions.Add("exported functions");
            if (sampleContent.Contains("error"))
                info.Conventions.Add("error handling pattern");
        }

        if (lang.CommonConventions.Length > 0)
            info.Conventions.InsertRange(0, lang.CommonConventions);
    }

    private static string DetectProjectType(string content, LanguageDefinition lang)
    {
        if (lang == Languages.CSharp)
        {
            if (content.Contains("Microsoft.NET.Sdk.Web")) return "web";
            if (content.Contains("Microsoft.NET.Sdk")) return "library";
            return "unknown";
        }
        if (lang == Languages.TypeScript || lang == Languages.JavaScript)
        {
            if (content.Contains("\"next\"") || content.Contains("\"react\"") || content.Contains("\"vue\""))
                return "web framework";
            if (content.Contains("\"express\"")) return "web server";
            if (content.Contains("\"jest\"") || content.Contains("\"vitest\"")) return "test";
            return "node library";
        }
        if (lang == Languages.Python)
        {
            if (content.Contains("django") || content.Contains("flask") || content.Contains("fastapi"))
                return "web framework";
            return "library";
        }
        return "application";
    }

    private static void ParseCsprojMetadata(string content, ProjectInfo info)
    {
        var tfm = Regex.Match(content, "<TargetFramework>(.*?)</TargetFramework>");
        if (tfm.Success) info.TargetFramework = tfm.Groups[1].Value;

        var ns = Regex.Match(content, "<RootNamespace>(.*?)</RootNamespace>");
        if (ns.Success) info.RootNamespace = ns.Groups[1].Value;

        foreach (Match match in PackageRefRegex().Matches(content))
        {
            info.Dependencies.Add(match.Groups[1].Value);
        }
    }

    private static void ParsePackageJson(string content, ProjectInfo info)
    {
        try
        {
            var doc = System.Text.Json.JsonDocument.Parse(content);
            var root = doc.RootElement;

            if (root.TryGetProperty("dependencies", out var deps))
            {
                foreach (var dep in deps.EnumerateObject())
                    info.Dependencies.Add(dep.Name);
            }
            if (root.TryGetProperty("devDependencies", out var devDeps))
            {
                foreach (var dep in devDeps.EnumerateObject())
                    info.Dependencies.Add(dep.Name + " (dev)");
            }
        }
        catch { }
    }

    private static string DetectTestCommand(string workingDirectory, Language.LanguageDefinition lang)
    {
        if (lang == Languages.TypeScript || lang == Languages.JavaScript)
        {
            var pkgJson = Directory.GetFiles(workingDirectory, "package.json", SearchOption.TopDirectoryOnly);
            if (pkgJson.Length > 0)
            {
                try
                {
                    var content = File.ReadAllText(pkgJson[0]);
                    if (content.Contains("\"jest\"")) return "npx jest";
                    if (content.Contains("\"vitest\"")) return "npx vitest run";
                    if (content.Contains("\"mocha\"")) return "npx mocha";
                    if (content.Contains("\"test\":")) return "npm test";
                }
                catch { }
            }
        }
        if (lang == Languages.Python)
        {
            if (File.Exists(Path.Combine(workingDirectory, "pytest.ini"))) return "python -m pytest";
            if (File.Exists(Path.Combine(workingDirectory, "pyproject.toml")))
            {
                var content = File.ReadAllText(Path.Combine(workingDirectory, "pyproject.toml"));
                if (content.Contains("pytest")) return "python -m pytest";
            }
        }
        return lang.DetectTestCommand;
    }

    private void AnalyzeCodeSymbols(string workingDirectory, ProjectInfo info)
    {
        var sourceFiles = _detector.GetSourceFiles(workingDirectory, info.Language);
        var limit = _limits.Output.SourceFilesCount > 0 ? _limits.Output.SourceFilesCount : int.MaxValue;

        foreach (var relPath in sourceFiles.Take(limit))
        {
            var fullPath = Path.Combine(workingDirectory, relPath);
            if (!File.Exists(fullPath)) continue;

            var content = File.ReadAllText(fullPath);
            var lines = content.Split('\n');

            var fileInfo = info.KeyFiles.FirstOrDefault(f => f.RelativePath == relPath);
            if (fileInfo is null)
            {
                fileInfo = new SourceFileInfo { RelativePath = relPath };
                info.KeyFiles.Add(fileInfo);
            }

            switch (info.Language.Name)
            {
                case "C#":
                    ParseCSharpFile(content, lines, relPath, fileInfo, info);
                    break;
                case "TypeScript":
                    ParseTypeScriptFile(lines, relPath, fileInfo, info);
                    break;
                case "JavaScript":
                    ParseJavaScriptFile(lines, relPath, fileInfo, info);
                    break;
                case "Go":
                    ParseGoFile(lines, relPath, fileInfo, info);
                    break;
                case "Python":
                    ParsePythonFile(lines, relPath, fileInfo, info);
                    break;
                case "Rust":
                    ParseRustFile(lines, relPath, fileInfo, info);
                    break;
            }
        }
    }

    private static void ParseCSharpFile(string content, string[] lines, string relPath, SourceFileInfo fileInfo, ProjectInfo info)
    {
        for (var i = 0; i < lines.Length; i++)
        {
            var line = lines[i];

            if (fileInfo.Namespace is null)
            {
                var nsMatch = Regex.Match(line, @"namespace\s+([\w.]+)");
                if (nsMatch.Success)
                    fileInfo.Namespace = nsMatch.Groups[1].Value;
            }

            var usingMatch = Regex.Match(line, @"using\s+([\w.]+);");
            if (usingMatch.Success)
            {
                var ns = usingMatch.Groups[1].Value;
                if (!fileInfo.Imports.Contains(ns))
                    fileInfo.Imports.Add(ns);
            }

            var typeMatch = Regex.Match(line, @"^\s*public\s+(?:(?:partial|static|abstract|sealed|readonly)\s+)*(class|record|struct|interface|enum)\s+(\w+)");
            if (typeMatch.Success)
            {
                var kind = typeMatch.Groups[1].Value;
                var name = typeMatch.Groups[2].Value;
                var symbol = new CodeSymbol(name, kind, relPath, i + 1);
                fileInfo.DeclaredTypes.Add(symbol);
                info.PublicTypes.Add(symbol);
                continue;
            }

            var methodMatch = Regex.Match(line, @"^\s*public\s+(?:(?:async|static|override|virtual|abstract|sealed|unsafe|new)\s+)*(?!class\b|record\b|struct\b|interface\b|enum\b|partial\b)(?:\w+(?:<[^>]+>)?\s+)?(\w+)\s*\(");
            if (methodMatch.Success)
            {
                var name = methodMatch.Groups[1].Value;
                var signature = line.Trim();
                var symbol = new CodeSymbol(name, "method", relPath, i + 1, signature);
                info.PublicMembers.Add(symbol);
            }
        }
    }

    private static void ParseTypeScriptFile(string[] lines, string relPath, SourceFileInfo fileInfo, ProjectInfo info)
    {
        for (var i = 0; i < lines.Length; i++)
        {
            var line = lines[i];

            var importMatch = Regex.Match(line, @"import\s+(?:\{[^}]*\}\s+from\s+)?['""]([^'""]+)['""]");
            if (importMatch.Success)
            {
                var module = importMatch.Groups[1].Value;
                if (!fileInfo.Imports.Contains(module))
                    fileInfo.Imports.Add(module);
            }

            var typeMatch = Regex.Match(line, @"export\s+(?:default\s+)?(class|interface|type|enum)\s+(\w+)");
            if (typeMatch.Success)
            {
                var kind = typeMatch.Groups[1].Value;
                var name = typeMatch.Groups[2].Value;
                var symbol = new CodeSymbol(name, kind, relPath, i + 1);
                fileInfo.DeclaredTypes.Add(symbol);
                info.PublicTypes.Add(symbol);
                continue;
            }

            var funcMatch = Regex.Match(line, @"export\s+(?:async\s+)?function\s+(\w+)\s*\(");
            if (funcMatch.Success)
            {
                var name = funcMatch.Groups[1].Value;
                var signature = line.Trim();
                var symbol = new CodeSymbol(name, "function", relPath, i + 1, signature);
                info.PublicMembers.Add(symbol);
                continue;
            }

            var arrowMatch = Regex.Match(line, @"export\s+(?:const|let|var)\s+(\w+)\s*(?::\s*\w+(?:<[^>]+>)?\s*)?=\s*(?:async\s*)?\(");
            if (arrowMatch.Success)
            {
                var name = arrowMatch.Groups[1].Value;
                var signature = line.Trim();
                var symbol = new CodeSymbol(name, "function", relPath, i + 1, signature);
                info.PublicMembers.Add(symbol);
            }
        }
    }

    private static void ParseJavaScriptFile(string[] lines, string relPath, SourceFileInfo fileInfo, ProjectInfo info)
    {
        for (var i = 0; i < lines.Length; i++)
        {
            var line = lines[i];

            var importMatch = Regex.Match(line, @"(?:import|require)\s*\(?\s*['""]([^'""]+)['""]");
            if (importMatch.Success)
            {
                var module = importMatch.Groups[1].Value;
                if (!fileInfo.Imports.Contains(module))
                    fileInfo.Imports.Add(module);
            }

            var typeMatch = Regex.Match(line, @"(?:export\s+)?(?:default\s+)?class\s+(\w+)");
            if (typeMatch.Success)
            {
                var name = typeMatch.Groups[1].Value;
                var symbol = new CodeSymbol(name, "class", relPath, i + 1);
                fileInfo.DeclaredTypes.Add(symbol);
                info.PublicTypes.Add(symbol);
                continue;
            }

            var funcMatch = Regex.Match(line, @"(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(");
            if (funcMatch.Success)
            {
                var name = funcMatch.Groups[1].Value;
                var signature = line.Trim();
                var symbol = new CodeSymbol(name, "function", relPath, i + 1, signature);
                info.PublicMembers.Add(symbol);
                continue;
            }

            var arrowMatch = Regex.Match(line, @"(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s*)?\(");
            if (arrowMatch.Success)
            {
                var name = arrowMatch.Groups[1].Value;
                var signature = line.Trim();
                var symbol = new CodeSymbol(name, "function", relPath, i + 1, signature);
                info.PublicMembers.Add(symbol);
            }
        }
    }

    private static void ParseGoFile(string[] lines, string relPath, SourceFileInfo fileInfo, ProjectInfo info)
    {
        var inImportBlock = false;

        for (var i = 0; i < lines.Length; i++)
        {
            var line = lines[i].Trim();

            if (line.StartsWith("import ("))
            {
                inImportBlock = true;
                continue;
            }

            if (inImportBlock)
            {
                if (line == ")")
                {
                    inImportBlock = false;
                    continue;
                }

                var importMatch = Regex.Match(line, @"""([^""]+)""");
                if (importMatch.Success)
                {
                    var imp = importMatch.Groups[1].Value;
                    if (!fileInfo.Imports.Contains(imp))
                        fileInfo.Imports.Add(imp);
                }
                continue;
            }

            var singleImport = Regex.Match(line, @"import\s+""([^""]+)""");
            if (singleImport.Success)
            {
                var imp = singleImport.Groups[1].Value;
                if (!fileInfo.Imports.Contains(imp))
                    fileInfo.Imports.Add(imp);
            }

            var typeMatch = Regex.Match(line, @"type\s+(\w+)\s+(struct|interface)\b");
            if (typeMatch.Success)
            {
                var name = typeMatch.Groups[1].Value;
                var kind = typeMatch.Groups[2].Value;
                var symbol = new CodeSymbol(name, kind, relPath, i + 1);
                fileInfo.DeclaredTypes.Add(symbol);
                info.PublicTypes.Add(symbol);
                continue;
            }

            var funcMatch = Regex.Match(lines[i], @"func\s+(?:\([^)]+\)\s+)?(\w+)\s*\(");
            if (funcMatch.Success)
            {
                var name = funcMatch.Groups[1].Value;
                var signature = lines[i].Trim();
                var symbol = new CodeSymbol(name, "function", relPath, i + 1, signature);
                info.PublicMembers.Add(symbol);
            }
        }
    }

    private static void ParsePythonFile(string[] lines, string relPath, SourceFileInfo fileInfo, ProjectInfo info)
    {
        for (var i = 0; i < lines.Length; i++)
        {
            var line = lines[i].Trim();

            var importMatch = Regex.Match(line, @"^(?:import|from)\s+(\w+(?:\.\w+)*)");
            if (importMatch.Success)
            {
                var module = importMatch.Groups[1].Value;
                if (!fileInfo.Imports.Contains(module))
                    fileInfo.Imports.Add(module);
            }

            var classMatch = Regex.Match(line, @"^class\s+(\w+)");
            if (classMatch.Success)
            {
                var name = classMatch.Groups[1].Value;
                var symbol = new CodeSymbol(name, "class", relPath, i + 1);
                fileInfo.DeclaredTypes.Add(symbol);
                info.PublicTypes.Add(symbol);
                continue;
            }

            var funcMatch = Regex.Match(line, @"^(?:async\s+)?def\s+(\w+)\s*\(");
            if (funcMatch.Success)
            {
                var name = funcMatch.Groups[1].Value;
                var signature = lines[i].Trim();
                var symbol = new CodeSymbol(name, "function", relPath, i + 1, signature);
                info.PublicMembers.Add(symbol);
            }
        }
    }

    private static void ParseRustFile(string[] lines, string relPath, SourceFileInfo fileInfo, ProjectInfo info)
    {
        for (var i = 0; i < lines.Length; i++)
        {
            var line = lines[i].Trim();

            var useMatch = Regex.Match(line, @"^use\s+([\w:]+)");
            if (useMatch.Success)
            {
                var module = useMatch.Groups[1].Value;
                if (!fileInfo.Imports.Contains(module))
                    fileInfo.Imports.Add(module);
            }

            var typeMatch = Regex.Match(line, @"^(?:pub\s+)?(?:struct|enum|trait|union)\s+(\w+)");
            if (typeMatch.Success)
            {
                var kind = typeMatch.Groups[1].Value;
                var name = typeMatch.Groups[2].Value;
                var symbol = new CodeSymbol(name, kind, relPath, i + 1);
                fileInfo.DeclaredTypes.Add(symbol);
                info.PublicTypes.Add(symbol);
                continue;
            }

            var funcMatch = Regex.Match(lines[i], @"^(?:pub\s+)?(?:async\s+)?(?:unsafe\s+)?fn\s+(\w+)\s*\(");
            if (funcMatch.Success)
            {
                var name = funcMatch.Groups[1].Value;
                var signature = lines[i].Trim();
                var symbol = new CodeSymbol(name, "function", relPath, i + 1, signature);
                info.PublicMembers.Add(symbol);
            }
        }
    }

    private void ClassifyFiles(ProjectInfo info)
    {
        var testDir = info.TestDirectory;

        foreach (var file in info.KeyFiles)
        {
            if (file.IsEntryPoint)
            {
                file.Role = FileRole.EntryPoint;
                continue;
            }

            var relPath = file.RelativePath;
            var dirName = Path.GetDirectoryName(relPath)?.Replace('\\', '/') ?? "";
            var fileName = Path.GetFileNameWithoutExtension(relPath);

            if (file.Role != FileRole.Unknown) continue;

            // Directory-based classification
            if (dirName.Contains("controllers", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Controller;
            else if (dirName.Contains("services", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Service;
            else if (dirName.Contains("repositor", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Repository;
            else if (dirName.Contains("models", StringComparison.OrdinalIgnoreCase) ||
                     dirName.Contains("entities", StringComparison.OrdinalIgnoreCase) ||
                     dirName.Contains("domain", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Model;
            else if (dirName.Contains("dto", StringComparison.OrdinalIgnoreCase) ||
                     dirName.Contains("viewmodel", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Dto;
            else if (dirName.Contains("middleware", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Middleware;
            else if (dirName.Contains("config", StringComparison.OrdinalIgnoreCase) ||
                     dirName.Contains("settings", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Configuration;
            else if (dirName.Contains("mapper", StringComparison.OrdinalIgnoreCase) ||
                     dirName.Contains("mapping", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Mapper;
            else if (dirName.Contains("validator", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Validator;
            else if (dirName.Contains("event", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Event;
            else if (dirName.Contains("constant", StringComparison.OrdinalIgnoreCase) ||
                     dirName.Contains("enum", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Constant;
            else if (dirName.Contains("extension", StringComparison.OrdinalIgnoreCase) ||
                     dirName.Contains("helpers", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Extension;
            else if (dirName.Contains("interfaces", StringComparison.OrdinalIgnoreCase) ||
                     dirName.Contains("abstractions", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Interface;
            else if (dirName.Contains("database", StringComparison.OrdinalIgnoreCase) ||
                     dirName.Contains("migrations", StringComparison.OrdinalIgnoreCase) ||
                     dirName.Contains("data", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Database;

            if (file.Role != FileRole.Unknown) continue;

            // Name-based classification
            if (fileName.EndsWith("Controller", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Controller;
            else if (fileName.EndsWith("Service", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Service;
            else if (fileName.EndsWith("Repository", StringComparison.OrdinalIgnoreCase) ||
                     fileName.EndsWith("Repo", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Repository;
            else if (fileName.EndsWith("Model", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Model;
            else if (fileName.EndsWith("Middleware", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Middleware;
            else if (fileName.EndsWith("Mapper", StringComparison.OrdinalIgnoreCase) ||
                     fileName.EndsWith("MappingProfile", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Mapper;
            else if (fileName.EndsWith("Validator", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Validator;
            else if (fileName.EndsWith("Event", StringComparison.OrdinalIgnoreCase) ||
                     fileName.EndsWith("EventHandler", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Event;
            else if (fileName.EndsWith("Extensions", StringComparison.OrdinalIgnoreCase))
                file.Role = FileRole.Extension;
            else if (fileName.StartsWith("I", StringComparison.OrdinalIgnoreCase) &&
                     fileName.Length > 2 &&
                     char.IsUpper(fileName[1]) &&
                     info.Language.Name == "C#")
                file.Role = FileRole.Interface;

            if (file.Role != FileRole.Unknown) continue;

            // Content-based classification (C# attributes)
            if (info.Language.Name == "C#" && file.Content is not null)
            {
                if (file.Content.Contains("[ApiController]"))
                    file.Role = FileRole.Controller;
                else if (file.Content.Contains("[HttpGet]") || file.Content.Contains("[HttpPost]") ||
                         file.Content.Contains("[HttpPut]") || file.Content.Contains("[HttpDelete]") ||
                         file.Content.Contains("[HttpPatch]"))
                    file.Role = FileRole.Controller;
                else if (file.Content.Contains("[ApiController]"))
                    file.Role = FileRole.Controller;
            }

            if (file.Role != FileRole.Unknown) continue;

            // Test file detection
            var isInTestDir = !string.IsNullOrEmpty(testDir) &&
                dirName.Contains(testDir, StringComparison.OrdinalIgnoreCase);
            var hasTestSuffix = fileName.EndsWith(info.TestFileSuffix.Replace(".cs", "").Replace(".ts", "").Replace(".py", "").Replace(".go", ""), StringComparison.OrdinalIgnoreCase) ||
                                fileName.EndsWith("Test", StringComparison.OrdinalIgnoreCase) ||
                                fileName.EndsWith("Tests", StringComparison.OrdinalIgnoreCase);
            if (isInTestDir || hasTestSuffix)
                file.Role = FileRole.Test;
        }
    }

    private void BuildProjectMap(ProjectInfo info)
    {
        var sb = new StringBuilder();
        sb.AppendLine("Project Map:");

        var rolesWithFiles = info.KeyFiles
            .Where(f => f.Role != FileRole.Unknown && f.Role != FileRole.EntryPoint)
            .GroupBy(f => f.Role)
            .OrderBy(g => g.Key.ToString());

        var hasContent = false;
        foreach (var group in rolesWithFiles)
        {
            var roleName = group.Key.ToString();
            sb.AppendLine($"  {roleName}s/");
            hasContent = true;

            foreach (var file in group.Take(10))
            {
                var types = file.DeclaredTypes.Count > 0
                    ? " → " + string.Join(", ", file.DeclaredTypes.Select(t => t.Name))
                    : "";

                var roleHint = "";
                if (file.Role == FileRole.Controller && file.Content is not null)
                {
                    if (file.Content.Contains("[ApiController]")) roleHint = " [ApiController]";
                }

                sb.AppendLine($"    {file.RelativePath}{roleHint}{types}");

                var methods = info.PublicMembers
                    .Where(m => m.FilePath == file.RelativePath)
                    .Take(5)
                    .ToList();

                if (methods.Count > 0)
                {
                    var methodLines = methods.Select(m =>
                    {
                        var sig = m.Signature ?? m.Name;
                        return sig.Length > 100 ? sig[..100] + "..." : sig;
                    });
                    sb.AppendLine("      " + string.Join("\n      ", methodLines));
                }
            }

            var remaining = group.Count() - 10;
            if (remaining > 0)
                sb.AppendLine($"      ... and {remaining} more files");
        }

        // Entry points
        var entryPoints = info.KeyFiles.Where(f => f.IsEntryPoint).ToList();
        if (entryPoints.Count > 0)
        {
            sb.AppendLine("  Entry Points:");
            foreach (var ep in entryPoints)
                sb.AppendLine($"    {ep.RelativePath}");
        }

        // Unclassified
        var unknownFiles = info.KeyFiles
            .Where(f => f.Role == FileRole.Unknown && !f.IsEntryPoint)
            .ToList();
        if (unknownFiles.Count > 0)
        {
            sb.AppendLine("  Other Files:");
            foreach (var file in unknownFiles.Take(5))
                sb.AppendLine($"    {file.RelativePath}");
            if (unknownFiles.Count > 5)
                sb.AppendLine($"    ... and {unknownFiles.Count - 5} more");
        }

        if (info.Dependencies.Count > 0)
        {
            sb.AppendLine();
            sb.AppendLine("Dependencies: " + string.Join(", ", info.Dependencies.Take(15)));
        }

        info.ProjectMap = hasContent || entryPoints.Count > 0 || unknownFiles.Count > 0
            ? sb.ToString()
            : null;
    }

    [GeneratedRegex(@"<PackageReference Include=""([^""]+)""")]
    private static partial Regex PackageRefRegex();
}
