using kai.Abstractions.Git;
using LibGit2Sharp;
using Microsoft.Extensions.Logging;

namespace kai.Git;

public sealed class GitService : IGitService
{
    private readonly ILogger<GitService> _logger;

    public GitService(ILogger<GitService> logger)
    {
        _logger = logger;
    }

    public string Commit(string path, string message)
    {
        using var repo = new Repository(path);

        var author = repo.Config.BuildSignature(DateTimeOffset.UtcNow);
        if (author is null)
        {
            _logger.LogWarning("No git user configured, using fallback signature");
            author = new Signature("kai-code", "kai-code@local", DateTimeOffset.UtcNow);
        }

        Commands.Stage(repo, "*");
        var commit = repo.Commit(message, author, author);
        _logger.LogInformation("Committed: {Sha} - {Message}", commit.Sha[..8], message);
        return commit.Sha;
    }
}
