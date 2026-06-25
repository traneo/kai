namespace kai.Abstractions.Git;

public interface IGitService
{
    string Commit(string path, string message);
}
