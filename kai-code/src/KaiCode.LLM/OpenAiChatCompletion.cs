using System.Net;
using System.Net.Http.Headers;
using System.Net.Http.Json;
using System.Text.Json;
using System.Text.Json.Serialization;
using Microsoft.Extensions.Logging;
using kai.Core.Configuration;

namespace kai.LLM;

public sealed class OpenAiChatCompletion : IChatCompletion, IDisposable
{
    private readonly HttpClient _http;
    private readonly ILogger<OpenAiChatCompletion> _logger;
    private readonly int _maxRetries;
    private readonly TimeSpan[] _retryDelays;
    private readonly int _maxTokens;

    private static readonly JsonSerializerOptions JsonOptions = new()
    {
        PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower,
        DefaultIgnoreCondition = JsonIgnoreCondition.WhenWritingNull
    };

    public OpenAiChatCompletion(ILogger<OpenAiChatCompletion> logger, LimitsConfig limits)
    {
        _logger = logger;
        _maxRetries = limits.Llm.MaxTokens > 0 ? limits.Retries.LlmApiRetries : 3;
        _maxTokens = limits.Llm.MaxTokens;
        _retryDelays = limits.Retries.LlmRetryDelaySeconds.Select(s => TimeSpan.FromSeconds(s)).ToArray();
        _http = new HttpClient();

        _logger.LogInformation("Chat client initialized");
    }

    public async Task<string> CompleteAsync(string systemPrompt, string userPrompt, ModelOptions? options = null, CancellationToken ct = default)
    {
        _logger.LogDebug("Complete: sys={SysLen} user={UserLen}", systemPrompt.Length, userPrompt.Length);

        var request = CreateRequest(systemPrompt, userPrompt, stream: false, options);
        var endpoint = options?.Endpoint ?? "";
        var apiKey = options?.ApiKey ?? "";

        for (var attempt = 0; attempt <= _maxRetries; attempt++)
        {
            try
            {
                var msg = BuildRequest(HttpMethod.Post, $"{endpoint.TrimEnd('/')}/chat/completions", request, apiKey);
                var response = await _http.SendAsync(msg, ct);

                if (response.StatusCode == HttpStatusCode.TooManyRequests ||
                    (response.StatusCode >= HttpStatusCode.InternalServerError && attempt < _maxRetries))
                {
                    await WaitAndRetry(attempt, ct);
                    continue;
                }

                response.EnsureSuccessStatusCode();

                var body = await response.Content.ReadFromJsonAsync<ChatResponse>(JsonOptions, ct);
                return body?.Choices?[0]?.Message?.Content ?? string.Empty;
            }
            catch (HttpRequestException ex) when (attempt < _maxRetries)
            {
                _logger.LogWarning(ex, "LLM call failed (attempt {Attempt}/{Max})", attempt + 1, _maxRetries);
                await WaitAndRetry(attempt, ct);
            }
            catch (TaskCanceledException) when (attempt < _maxRetries)
            {
                _logger.LogWarning("LLM call timed out (attempt {Attempt}/{Max})", attempt + 1, _maxRetries);
                await WaitAndRetry(attempt, ct);
            }
        }

        _logger.LogError("LLM call failed after {Max} retries", _maxRetries);
        throw new HttpRequestException($"LLM call failed after {_maxRetries} retries");
    }

    public async IAsyncEnumerable<string> StreamAsync(string systemPrompt, string userPrompt, ModelOptions? options = null, [System.Runtime.CompilerServices.EnumeratorCancellation] CancellationToken ct = default)
    {
        _logger.LogDebug("Stream: sys={SysLen} user={UserLen}", systemPrompt.Length, userPrompt.Length);

        var request = CreateRequest(systemPrompt, userPrompt, stream: true, options);
        var endpoint = options?.Endpoint ?? "";
        var apiKey = options?.ApiKey ?? "";
        var lastException = (Exception?)null;

        for (var attempt = 0; attempt <= _maxRetries; attempt++)
        {
            HttpResponseMessage? response = null;

            try
            {
                var msg = BuildRequest(HttpMethod.Post, $"{endpoint.TrimEnd('/')}/chat/completions", request, apiKey);
                response = await _http.SendAsync(msg, ct);
                response.EnsureSuccessStatusCode();
            }
            catch (Exception ex) when (ex is HttpRequestException or TaskCanceledException)
            {
                lastException = ex;
                if (attempt < _maxRetries)
                {
                    _logger.LogWarning(ex, "LLM stream failed (attempt {Attempt}/{Max})", attempt + 1, _maxRetries);
                    await WaitAndRetry(attempt, ct);
                }
                continue;
            }

            using var stream = await response.Content.ReadAsStreamAsync(ct);
            using var reader = new StreamReader(stream);

            while (true)
            {
                var line = await reader.ReadLineAsync(ct);
                if (line is null) break;
                if (!line.StartsWith("data: ")) continue;

                var data = line[6..];
                if (data == "[DONE]") break;

                if (TryParseChunk(data) is { } content)
                    yield return content;
            }

            yield break;
        }

        _logger.LogError("LLM stream failed after {Max} retries", _maxRetries);
        throw lastException ?? new HttpRequestException($"LLM stream failed after {_maxRetries} retries");
    }

    private HttpRequestMessage BuildRequest(HttpMethod method, string url, ChatRequest body, string apiKey)
    {
        var msg = new HttpRequestMessage(method, url)
        {
            Content = JsonContent.Create(body, options: JsonOptions)
        };
        if (!string.IsNullOrEmpty(apiKey))
        {
            msg.Headers.Authorization = new AuthenticationHeaderValue("Bearer", apiKey);
        }
        return msg;
    }

    private async Task WaitAndRetry(int attempt, CancellationToken ct)
    {
        var delay = attempt < _retryDelays.Length ? _retryDelays[attempt] : _retryDelays[^1];
        _logger.LogInformation("Retrying in {Delay}s...", delay.TotalSeconds);
        await Task.Delay(delay, ct);
    }

    private ChatRequest CreateRequest(string systemPrompt, string userPrompt, bool stream, ModelOptions? options = null)
    {
        return new ChatRequest
        {
            Model = options?.Model ?? "",
            Stream = stream,
            Messages =
            [
                new Message { Role = "system", Content = systemPrompt },
                new Message { Role = "user", Content = userPrompt }
            ],
            MaxTokens = _maxTokens,
            Temperature = options?.Temperature,
            TopP = options?.TopP,
            TopK = options?.TopK
        };
    }

    private string? TryParseChunk(string data)
    {
        try
        {
            var chunk = JsonSerializer.Deserialize<StreamChunk>(data, JsonOptions);
            return chunk?.Choices?[0]?.Delta?.Content;
        }
        catch (Exception ex)
        {
            _logger.LogWarning(ex, "Failed to parse SSE chunk: {Data}", data);
            return null;
        }
    }

    public void Dispose() => _http.Dispose();

    private sealed class ChatRequest
    {
        [JsonPropertyName("model")] public string Model { get; set; } = "";
        [JsonPropertyName("messages")] public Message[] Messages { get; set; } = [];
        [JsonPropertyName("stream")] public bool Stream { get; set; }
        [JsonPropertyName("max_tokens")] public int MaxTokens { get; set; } = 8192;
        [JsonPropertyName("temperature")] public double? Temperature { get; set; }
        [JsonPropertyName("top_p")] public double? TopP { get; set; }
        [JsonPropertyName("top_k")] public int? TopK { get; set; }
    }

    private sealed class Message
    {
        [JsonPropertyName("role")] public string Role { get; set; } = "";
        [JsonPropertyName("content")] public string Content { get; set; } = "";
    }

    private sealed class ChatResponse
    {
        [JsonPropertyName("choices")] public Choice[]? Choices { get; set; }
    }

    private sealed class Choice
    {
        [JsonPropertyName("message")] public Message? Message { get; set; }
    }

    private sealed class StreamChunk
    {
        [JsonPropertyName("choices")] public StreamChoice[]? Choices { get; set; }
    }

    private sealed class StreamChoice
    {
        [JsonPropertyName("delta")] public Message? Delta { get; set; }
    }
}