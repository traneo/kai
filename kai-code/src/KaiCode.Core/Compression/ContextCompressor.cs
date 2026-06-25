using kai.Abstractions.LLM;
using kai.Core.Configuration;
using kai.Models;

namespace kai.Core.Compression;

public sealed class ContextCompressor
{
    private readonly IChatCompletion _chat;
    private readonly ModelOptions? _modelOptions;

    public ContextCompressor(IChatCompletion chat, kaiConfig config)
    {
        _chat = chat;
        _modelOptions = config.Agents.GetValueOrDefault("coder")?.ToModelOptions();
    }

    public async Task CompressAsync(
        List<(string Role, string Content)> messages,
        Dictionary<string, int> readFileCache,
        CancellationToken ct)
    {
        if (messages.Count <= 4) return;

        var keepFrom = messages.Count - 10;
        if (keepFrom < 2) keepFrom = 2;

        var toCompress = messages.GetRange(2, keepFrom - 2);
        if (toCompress.Count < 4) return;

        var segment = string.Join("\n\n", toCompress.Select(m => $"## {m.Role}\n{m.Content}\n"));

        var summary = await _chat.CompleteAsync(
            "You are a summarizer. Condense the following coding session into 2-3 sentences. " +
            "Include: what files were read and their contents, what errors occurred, " +
            "what changes were made, and what decisions were reached. Be factual and concise.",
            segment,
            _modelOptions, ct);

        messages.RemoveRange(2, keepFrom - 2);
        messages.Insert(2, ("system", $"[Context compressed] {summary.Trim()}"));

        var stale = new List<string>();
        foreach (var k in readFileCache.Keys)
        {
            var newIdx = readFileCache[k] - (keepFrom - 2) + 1;
            if (newIdx < 2) stale.Add(k);
            else readFileCache[k] = newIdx;
        }
        foreach (var k in stale) readFileCache.Remove(k);
    }
}
