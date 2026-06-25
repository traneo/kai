namespace kai.Models;

public record ModelOptions(string? Model = null, double? Temperature = null, double? TopP = null, int? TopK = null, string? Endpoint = null, string? ApiKey = null, string? Provider = null);
