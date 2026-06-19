namespace kai.Core.Language;

public sealed class LanguageDetector
{
    public LanguageDefinition Detect(string workingDirectory)
    {
        foreach (var lang in Languages.All)
        {
            foreach (var configFile in lang.ConfigFiles)
            {
                if (configFile.Contains('*'))
                {
                    var pattern = configFile.Replace("*.", "*.");
                    if (Directory.GetFiles(workingDirectory, configFile, SearchOption.AllDirectories).Length > 0)
                        return lang;
                }
                else
                {
                    var path = Path.Combine(workingDirectory, configFile);
                    if (File.Exists(path))
                        return lang;
                }
            }
        }

        var allFiles = Directory.GetFiles(workingDirectory, "*", SearchOption.AllDirectories);

        foreach (var lang in Languages.All)
        {
            foreach (var ext in lang.FileExtensions)
            {
                var pattern = ext.Replace("*.", "*.");
                if (allFiles.Any(f => f.EndsWith(ext[1..])))
                    return lang;
            }
        }

        return Languages.Unknown;
    }

    public string[] GetSourceFiles(string workingDirectory, LanguageDefinition lang)
    {
        var files = new List<string>();
        foreach (var ext in lang.FileExtensions)
        {
            files.AddRange(Directory.GetFiles(workingDirectory, ext, SearchOption.AllDirectories));
        }
        return [..files
            .Where(f => !f.Contains("node_modules") && !f.Contains("bin/") && !f.Contains("obj/") && !f.Contains(".git/"))
            .Select(f => Path.GetRelativePath(workingDirectory, f))
            .OrderBy(f => f)];
    }
}
