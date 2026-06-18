namespace kai.Core.Language;

public record LanguageDefinition(
    string Name,
    string[] FileExtensions,
    string[] ConfigFiles,
    string[] BuildFiles,
    string LineComment,
    string BlockCommentStart,
    string BlockCommentEnd,
    string[] CommonConventions,
    string DetectBuildCommand,
    string DetectTestCommand,
    string TestFileSuffix,
    string TestDirectory
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
        CommonConventions: ["file-scoped namespaces", "PascalCase naming", "primary constructors", "XML documentation comments"],
        DetectBuildCommand: "dotnet build",
        DetectTestCommand: "dotnet test",
        TestFileSuffix: "Tests.cs",
        TestDirectory: "tests"
    );

    public static readonly LanguageDefinition TypeScript = new(
        Name: "TypeScript",
        FileExtensions: ["*.ts", "*.tsx"],
        ConfigFiles: ["package.json", "tsconfig.json"],
        BuildFiles: ["package.json"],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        CommonConventions: ["camelCase variables", "PascalCase types", "ES modules", "async/await"],
        DetectBuildCommand: "npx tsc --noEmit",
        DetectTestCommand: "npx vitest run",
        TestFileSuffix: ".test.ts",
        TestDirectory: "tests"
    );

    public static readonly LanguageDefinition JavaScript = new(
        Name: "JavaScript",
        FileExtensions: ["*.js", "*.jsx", "*.mjs"],
        ConfigFiles: ["package.json"],
        BuildFiles: ["package.json"],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        CommonConventions: ["camelCase variables", "ES modules", "async/await"],
        DetectBuildCommand: "node --check",
        DetectTestCommand: "npx vitest run",
        TestFileSuffix: ".test.js",
        TestDirectory: "tests"
    );

    public static readonly LanguageDefinition Python = new(
        Name: "Python",
        FileExtensions: ["*.py"],
        ConfigFiles: ["pyproject.toml", "setup.py", "requirements.txt", "Pipfile"],
        BuildFiles: ["pyproject.toml", "setup.py"],
        LineComment: "#",
        BlockCommentStart: "\"\"\"",
        BlockCommentEnd: "\"\"\"",
        CommonConventions: ["snake_case naming", "type hints", "PEP 8 style"],
        DetectBuildCommand: "python -m py_compile",
        DetectTestCommand: "python -m pytest",
        TestFileSuffix: "_test.py",
        TestDirectory: "tests"
    );

    public static readonly LanguageDefinition Go = new(
        Name: "Go",
        FileExtensions: ["*.go"],
        ConfigFiles: ["go.mod"],
        BuildFiles: ["go.mod"],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        CommonConventions: ["camelCase exports", "error handling", "interfaces", "defer pattern"],
        DetectBuildCommand: "go build ./...",
        DetectTestCommand: "go test ./...",
        TestFileSuffix: "_test.go",
        TestDirectory: ""
    );

    public static readonly LanguageDefinition Rust = new(
        Name: "Rust",
        FileExtensions: ["*.rs"],
        ConfigFiles: ["Cargo.toml"],
        BuildFiles: ["Cargo.toml"],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        CommonConventions: ["snake_case naming", "Result/Option pattern", "trait-based generics"],
        DetectBuildCommand: "cargo check",
        DetectTestCommand: "cargo test",
        TestFileSuffix: "_test.rs",
        TestDirectory: "tests"
    );

    public static readonly LanguageDefinition Java = new(
        Name: "Java",
        FileExtensions: ["*.java"],
        ConfigFiles: ["pom.xml", "build.gradle", "build.gradle.kts"],
        BuildFiles: ["pom.xml", "build.gradle"],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        CommonConventions: ["PascalCase classes", "camelCase methods", "package-based namespaces", "Javadoc"],
        DetectBuildCommand: "mvn compile",
        DetectTestCommand: "mvn test",
        TestFileSuffix: "Test.java",
        TestDirectory: "src/test/java"
    );

    public static readonly LanguageDefinition Unknown = new(
        Name: "Unknown",
        FileExtensions: ["*.*"],
        ConfigFiles: [],
        BuildFiles: [],
        LineComment: "//",
        BlockCommentStart: "/*",
        BlockCommentEnd: "*/",
        CommonConventions: [],
        DetectBuildCommand: "",
        DetectTestCommand: "",
        TestFileSuffix: "_test",
        TestDirectory: "tests"
    );

    public static LanguageDefinition[] All => [CSharp, TypeScript, JavaScript, Python, Go, Rust, Java];

    public static LanguageDefinition FindByName(string name) =>
        All.FirstOrDefault(l =>
            l.Name.Equals(name, StringComparison.OrdinalIgnoreCase)) ?? Unknown;
}
