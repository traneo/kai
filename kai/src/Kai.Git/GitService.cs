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

    public bool IsRepository(string path)
    {
        try
        {
            return Repository.IsValid(path);
        }
        catch
        {
            return false;
        }
    }

    public string GetCurrentBranch(string path)
    {
        using var repo = new Repository(path);
        return repo.Head.FriendlyName;
    }

    public string CreateBranch(string path, string branchName)
    {
        using var repo = new Repository(path);

        var fullBranchName = repo.Head.FriendlyName.StartsWith(branchName)
            ? branchName
            : branchName;

        var branch = repo.Branches[fullBranchName];
        if (branch is null)
        {
            branch = repo.CreateBranch(fullBranchName);
            _logger.LogInformation("Created branch: {Branch}", fullBranchName);
        }

        Commands.Checkout(repo, branch);
        _logger.LogInformation("Checked out branch: {Branch}", fullBranchName);
        return fullBranchName;
    }

    public string Commit(string path, string message)
    {
        using var repo = new Repository(path);

        var author = repo.Config.BuildSignature(DateTimeOffset.UtcNow);
        if (author is null)
        {
            _logger.LogWarning("No git user configured, using fallback signature");
            author = new Signature("kai", "kai@local", DateTimeOffset.UtcNow);
        }

        Commands.Stage(repo, "*");
        var commit = repo.Commit(message, author, author);
        _logger.LogInformation("Committed: {Sha} - {Message}", commit.Sha[..8], message);
        return commit.Sha;
    }

    public void DeleteBranch(string path, string branchName)
    {
        using var repo = new Repository(path);

        var existing = repo.Branches[branchName];
        if (existing is null) return;

        if (repo.Head.FriendlyName == branchName)
            Commands.Checkout(repo, repo.Branches[repo.Refs.FirstOrDefault(r =>
                r.TargetIdentifier != branchName)?.TargetIdentifier ?? "main"]);

        repo.Branches.Remove(existing);
        _logger.LogInformation("Deleted branch: {Branch}", branchName);
    }

    public string GetDiff(string path)
    {
        using var repo = new Repository(path);

        var head = repo.Head.Tip;
        if (head is null) return string.Empty;

        var parent = head.Parents.FirstOrDefault();
        if (parent is null) return string.Empty;

        var changes = repo.Diff.Compare<TreeChanges>(parent.Tree, head.Tree);
        if (changes is null) return string.Empty;

        var patches = changes.Select(c =>
        {
            var patch = repo.Diff.Compare<Patch>(parent.Tree, head.Tree, new[] { c.Path });
            return $"--- a/{c.Path}\n+++ b/{c.Path}\n{patch.Content}";
        });

        return string.Join("\n", patches);
    }

    public List<string> GetChangedFiles(string path)
    {
        using var repo = new Repository(path);

        var head = repo.Head.Tip;
        if (head is null) return [];

        var parent = head.Parents.FirstOrDefault();
        if (parent is null) return [];

        var changes = repo.Diff.Compare<TreeChanges>(parent.Tree, head.Tree);
        if (changes is null) return [];

        return [..changes.Select(c => c.Path)];
    }
}
