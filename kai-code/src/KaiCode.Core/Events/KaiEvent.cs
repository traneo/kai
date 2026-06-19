using kai.Core.Gating;

namespace kai.Core.Events;

public abstract record kaiEvent(DateTime Timestamp, string AgentName, string Type);

public record PipelineStartedEvent(string Goal) : kaiEvent(DateTime.UtcNow, "pipeline", "pipeline_started");
public record PipelineCompletedEvent(string Goal, int TaskCount, bool Success) : kaiEvent(DateTime.UtcNow, "pipeline", "pipeline_completed");
public record PhaseChangedEvent(int TaskIndex, string TaskDescription, string Phase) : kaiEvent(DateTime.UtcNow, "pipeline", "phase_changed");
public record ToolEvent(string ToolName, string Args, bool Success, string? Output) : kaiEvent(DateTime.UtcNow, "coder", "tool");
public record GateRequestedEvent(GateRequest Request) : kaiEvent(DateTime.UtcNow, Request.AgentName, "gate_requested");
public record GateResolvedEvent(Guid RequestId, string? Choice) : kaiEvent(DateTime.UtcNow, "pipeline", "gate_resolved");
public record AgentMessageEvent(string Role, string Content) : kaiEvent(DateTime.UtcNow, "coder", "agent_message");
