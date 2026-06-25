namespace kai.Models;

public record LanguageDefinition(
    string Name,
    string[] FileExtensions,
    string[] ConfigFiles,
    string[] BuildFiles,
    string LineComment,
    string BlockCommentStart,
    string BlockCommentEnd,
    string DetectBuildCommand
);

public static class Languages
{
    public static readonly LanguageDefinition CSharp = new(
        Name: "C#",
        FileExtensions: ["*.cs"],
        ConfigFiles: ["*.csproj", "*.sln", "*.slnx"],
        BuildFiles: ["*.csproj", "*.sln", "*.slnx"],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        DetectBuildCommand: "dotnet build"
    );

    public static readonly LanguageDefinition TypeScript = new(
        Name: "TypeScript",
        FileExtensions: ["*.ts", "*.tsx"],
        ConfigFiles: ["package.json", "tsconfig.json"],
        BuildFiles: ["package.json"],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        DetectBuildCommand: "npx tsc --noEmit"
    );

    public static readonly LanguageDefinition JavaScript = new(
        Name: "JavaScript",
        FileExtensions: ["*.js", "*.jsx", "*.mjs"],
        ConfigFiles: ["package.json"],
        BuildFiles: ["package.json"],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        DetectBuildCommand: "node --check"
    );

    public static readonly LanguageDefinition Python = new(
        Name: "Python",
        FileExtensions: ["*.py"],
        ConfigFiles: ["pyproject.toml", "setup.py", "requirements.txt", "Pipfile"],
        BuildFiles: ["pyproject.toml", "setup.py"],
        LineComment: "#",
        BlockCommentStart: "\"\"\"",
        BlockCommentEnd: "\"\"\"",
        DetectBuildCommand: "python -m py_compile"
    );

    public static readonly LanguageDefinition Go = new(
        Name: "Go",
        FileExtensions: ["*.go"],
        ConfigFiles: ["go.mod"],
        BuildFiles: ["go.mod"],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        DetectBuildCommand: "go build ./..."
    );

    public static readonly LanguageDefinition Rust = new(
        Name: "Rust",
        FileExtensions: ["*.rs"],
        ConfigFiles: ["Cargo.toml"],
        BuildFiles: ["Cargo.toml"],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        DetectBuildCommand: "cargo check"
    );

    public static readonly LanguageDefinition Java = new(
        Name: "Java",
        FileExtensions: ["*.java"],
        ConfigFiles: ["pom.xml", "build.gradle", "build.gradle.kts"],
        BuildFiles: ["pom.xml", "build.gradle"],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        DetectBuildCommand: "mvn compile"
    );

    public static readonly LanguageDefinition Unknown = new(
        Name: "",
        FileExtensions: ["*.*"],
        ConfigFiles: [],
        BuildFiles: [],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        DetectBuildCommand: ""
    );

    public static LanguageDefinition[] All => [CSharp, TypeScript, JavaScript, Python, Go, Rust, Java];

    public static LanguageDefinition FindByName(string name) =>
        All.FirstOrDefault(l =>
            l.Name.Equals(name, StringComparison.OrdinalIgnoreCase)) ?? Unknown;
}
