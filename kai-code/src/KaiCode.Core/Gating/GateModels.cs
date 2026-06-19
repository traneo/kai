namespace kai.Core.Gating;

public record GateRequest(
    Guid Id,
    string AgentName,
    string Reason,
    string Message,
    string? Context,
    string[]? Options);

public record GateResponse(
    Guid RequestId,
    string? Choice,
    string? Message);
