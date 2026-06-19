namespace kai.Git;

public interface IGitService
{
    bool IsRepository(string path);
    string GetCurrentBranch(string path);
    string CreateBranch(string path, string branchName);
    string Commit(string path, string message);
    void DeleteBranch(string path, string branchName);
    string GetDiff(string path);
    List<string> GetChangedFiles(string path);
}
