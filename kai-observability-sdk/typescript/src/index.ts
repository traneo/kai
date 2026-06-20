type LogLevel = "info" | "warn" | "error" | "debug";

interface LogEntry {
  service: string;
  level: LogLevel;
  message: string;
  timestamp: number;
  run_id?: string;
  step_id?: string;
  mission_id?: string;
  agent_id?: string;
  metadata?: Record<string, unknown>;
}

interface BatchRequest {
  entries: LogEntry[];
}

export class LogClient {
  private endpoint: string;
  private service: string;
  private batchSize: number;
  private flushInterval: number;
  private queue: LogEntry[] = [];
  private timer: ReturnType<typeof setInterval> | null = null;
  private runId?: string;
  private stepId?: string;
  private missionId?: string;
  private agentId?: string;

  constructor(endpoint: string, service: string, opts?: {
    batchSize?: number;
    flushInterval?: number;
  }) {
    this.endpoint = endpoint.replace(/\/$/, "");
    this.service = service;
    this.batchSize = opts?.batchSize ?? 50;
    this.flushInterval = opts?.flushInterval ?? 1000;
  }

  private start(): void {
    if (this.timer) return;
    this.timer = setInterval(() => this.flush(), this.flushInterval);
  }

  private log(level: LogLevel, message: string, metadata?: Record<string, unknown>): void {
    this.start();
    const entry: LogEntry = {
      service: this.service,
      level,
      message,
      timestamp: Date.now(),
      run_id: this.runId,
      step_id: this.stepId,
      mission_id: this.missionId,
      agent_id: this.agentId,
      metadata,
    };
    this.queue.push(entry);
    if (this.queue.length >= this.batchSize) {
      this.flush();
    }
  }

  info(message: string, metadata?: Record<string, unknown>): void { this.log("info", message, metadata); }
  warn(message: string, metadata?: Record<string, unknown>): void { this.log("warn", message, metadata); }
  error(message: string, metadata?: Record<string, unknown>): void { this.log("error", message, metadata); }
  debug(message: string, metadata?: Record<string, unknown>): void { this.log("debug", message, metadata); }

  withRunId(runId: string): LogClient {
    const c = this.clone(); c.runId = runId; return c;
  }
  withStepId(stepId: string): LogClient {
    const c = this.clone(); c.stepId = stepId; return c;
  }
  withMissionId(missionId: string): LogClient {
    const c = this.clone(); c.missionId = missionId; return c;
  }
  withAgentId(agentId: string): LogClient {
    const c = this.clone(); c.agentId = agentId; return c;
  }

  private clone(): LogClient {
    const c = new LogClient(this.endpoint, this.service);
    c.batchSize = this.batchSize;
    c.flushInterval = this.flushInterval;
    return c;
  }

  private flush(): void {
    if (this.queue.length === 0) return;
    const batch = this.queue.splice(0, this.batchSize);
    const body: BatchRequest = { entries: batch };
    fetch(`${this.endpoint}/api/v1/logs/batch`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }).catch(() => {});
  }

  subscribe(url?: string): EventSource {
    const source = new EventSource((url ?? this.endpoint) + "/api/v1/logs/stream");
    return source;
  }

  close(): void {
    if (this.timer) {
      clearInterval(this.timer);
      this.timer = null;
    }
    this.flush();
  }
}
