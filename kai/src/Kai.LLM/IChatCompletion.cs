using kai.Core.Configuration;

namespace kai.LLM;

public interface IChatCompletion
{
    Task<string> CompleteAsync(string systemPrompt, string userPrompt, ModelOptions? options = null, CancellationToken ct = default);
    IAsyncEnumerable<string> StreamAsync(string systemPrompt, string userPrompt, ModelOptions? options = null, CancellationToken ct = default);
}