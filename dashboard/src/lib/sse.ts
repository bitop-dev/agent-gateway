// SSE event stream connection

export interface AgentEvent {
  type: string;
  data: Record<string, unknown>;
  timestamp: string;
}

export type EventHandler = (event: AgentEvent) => void;

export class EventStream {
  private source: EventSource | null = null;
  private handlers: EventHandler[] = [];
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private apiKey: string = "";

  connect(apiKey: string) {
    this.apiKey = apiKey;
    this.disconnect();

    // SSE can't set headers — pass token as query param
    const url = `/v1/events${apiKey ? `?token=${encodeURIComponent(apiKey)}` : ""}`;
    this.source = new EventSource(url);

    // Named events from gateway
    const eventTypes = [
      "agent.task.submitted",
      "agent.task.started",
      "agent.task.completed",
      "agent.task.failed",
      "agent.task.retry",
      "agent.worker.joined",
      "agent.webhook.fired",
    ];

    for (const type of eventTypes) {
      this.source.addEventListener(type, (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data);
          const event: AgentEvent = {
            type,
            data,
            timestamp: data.timestamp || new Date().toISOString(),
          };
          this.handlers.forEach((h) => h(event));
        } catch {
          // ignore
        }
      });
    }

    // Also handle generic messages
    this.source.onmessage = (e) => {
      try {
        const event: AgentEvent = JSON.parse(e.data);
        if (!event.type) event.type = "message";
        if (!event.timestamp) event.timestamp = new Date().toISOString();
        this.handlers.forEach((h) => h(event));
      } catch {
        // ignore parse errors
      }
    };

    this.source.onerror = () => {
      this.source?.close();
      this.source = null;
      // Reconnect after 5s
      this.reconnectTimer = setTimeout(() => this.connect(this.apiKey), 5000);
    };
  }

  disconnect() {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
    if (this.source) {
      this.source.close();
      this.source = null;
    }
  }

  onEvent(handler: EventHandler) {
    this.handlers.push(handler);
    return () => {
      this.handlers = this.handlers.filter((h) => h !== handler);
    };
  }

  get connected(): boolean {
    return this.source?.readyState === EventSource.OPEN;
  }
}

export const eventStream = new EventStream();
