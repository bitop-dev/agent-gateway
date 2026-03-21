// Gateway API client

const API_BASE = "";

export interface Task {
  id: string;
  profile: string;
  task: string;
  status: "queued" | "running" | "completed" | "failed";
  priority?: string;
  output?: string;
  error?: string;
  workerUrl?: string;
  model?: string;
  inputTokens?: number;
  outputTokens?: number;
  toolCalls?: number;
  cost?: number;
  durationMs?: number;
  createdAt: string;
  startedAt?: string;
  completedAt?: string;
}

export interface Worker {
  url: string;
  status: string;
  profiles?: string[];
  currentTask?: string;
  lastSeen: string;
  completedTasks?: number;
}

export interface Agent {
  name: string;
  description?: string;
  capabilities?: string[];
  accepts?: string;
  returns?: string;
  extends?: string;
  mode?: string;
  model?: string;
  provider?: string;
  tools?: string[];
  source: string;
}

export interface Webhook {
  id: string;
  name: string;
  path: string;
  profile: string;
  taskTemplate: string;
  authToken?: string;
  enabled: boolean;
  createdAt: string;
}

export interface Schedule {
  id: string;
  name: string;
  cron: string;
  timezone?: string;
  profile: string;
  task: string;
  enabled: boolean;
  lastRun?: string;
  nextRun?: string;
  createdAt: string;
}

export interface MemoryEntry {
  key: string;
  value: string;
  updatedAt: string;
}

export interface CostSummary {
  profile: string;
  tasks: number;
  totalTasks?: number;
  inputTokens: number;
  outputTokens: number;
  totalTokens?: number;
  cost: number;
  totalCost?: number;
}

export interface ModelPricing {
  model: string;
  inputPerMillion: number;
  outputPerMillion: number;
  source: string;
}

export interface Plugin {
  name: string;
  version: string;
  description?: string;
  category?: string;
  runtime?: string;
  tools?: string[];
  dependencies?: string[];
  source: string;
}

export interface HealthResponse {
  status: string;
  workers: number;
  tasks: number;
}

export class AgentAPI {
  private key: string;

  constructor(key: string) {
    this.key = key;
  }

  setKey(key: string) {
    this.key = key;
  }

  private headers(): Record<string, string> {
    const h: Record<string, string> = { "Content-Type": "application/json" };
    if (this.key) h["Authorization"] = `Bearer ${this.key}`;
    return h;
  }

  private async get<T>(path: string): Promise<T> {
    const resp = await fetch(`${API_BASE}${path}`, { headers: this.headers() });
    if (!resp.ok) throw new Error(`${resp.status}: ${await resp.text()}`);
    return resp.json();
  }

  private async post<T>(path: string, body?: unknown): Promise<T> {
    const resp = await fetch(`${API_BASE}${path}`, {
      method: "POST",
      headers: this.headers(),
      body: body ? JSON.stringify(body) : undefined,
    });
    if (!resp.ok) throw new Error(`${resp.status}: ${await resp.text()}`);
    return resp.json();
  }

  private async del(path: string): Promise<void> {
    const resp = await fetch(`${API_BASE}${path}`, {
      method: "DELETE",
      headers: this.headers(),
    });
    if (!resp.ok) throw new Error(`${resp.status}: ${await resp.text()}`);
  }

  // Tasks
  async getTasks(opts?: {
    status?: string;
    limit?: number;
  }): Promise<{ tasks: Task[]; total: number }> {
    const params = new URLSearchParams();
    if (opts?.status) params.set("status", opts.status);
    if (opts?.limit) params.set("limit", String(opts.limit));
    const qs = params.toString();
    return this.get(`/v1/tasks${qs ? "?" + qs : ""}`);
  }

  async getTask(id: string): Promise<Task> {
    return this.get(`/v1/tasks/${id}`);
  }

  async submitTask(
    profile: string,
    task: string,
    async_?: boolean
  ): Promise<{ taskId: string; status: string; result?: string }> {
    return this.post("/v1/tasks", { profile, task, async: async_ });
  }

  async submitParallel(
    tasks: Array<{ profile: string; task: string }>
  ): Promise<{ results: Array<{ taskId: string; status: string }> }> {
    return this.post("/v1/tasks/parallel", { tasks });
  }

  // Workers
  async getWorkers(): Promise<{ workers: Worker[] }> {
    return this.get("/v1/workers");
  }

  // Agents
  async getAgents(): Promise<{ agents: Agent[] }> {
    return this.get("/v1/agents");
  }

  // Costs
  async getCosts(since?: string): Promise<{ costs: CostSummary[] }> {
    const qs = since ? `?since=${since}` : "";
    return this.get(`/v1/costs${qs}`);
  }

  async getPricing(): Promise<{ pricing: ModelPricing[] }> {
    return this.get("/v1/costs/pricing");
  }

  async setPricing(
    model: string,
    inputPerMillion: number,
    outputPerMillion: number
  ): Promise<void> {
    await this.post("/v1/costs/pricing", {
      model,
      inputPerMillion,
      outputPerMillion,
    });
  }

  // Memory
  async getMemory(
    profile: string,
    key?: string
  ): Promise<{ entries: MemoryEntry[] }> {
    const params = new URLSearchParams({ profile });
    if (key) params.set("key", key);
    const resp = await this.get<any>(`/v1/memory?${params}`);
    // Normalize: gateway returns { entries: [...] } or single { key, value }
    if (resp.entries) return { entries: resp.entries };
    if (resp.key) return { entries: [resp as MemoryEntry] };
    return { entries: [] };
  }

  async setMemory(profile: string, key: string, value: string): Promise<void> {
    await this.post(`/v1/memory?profile=${encodeURIComponent(profile)}`, {
      key,
      value,
    });
  }

  async deleteMemory(profile: string, key?: string): Promise<void> {
    const params = new URLSearchParams({ profile });
    if (key) params.set("key", key);
    await this.del(`/v1/memory?${params}`);
  }

  // Webhooks
  async getWebhooks(): Promise<{ webhooks: Webhook[]; count: number }> {
    return this.get("/v1/webhooks");
  }

  async createWebhook(
    config: Omit<Webhook, "id" | "createdAt">
  ): Promise<Webhook> {
    return this.post("/v1/webhooks", config);
  }

  async deleteWebhook(id: string): Promise<void> {
    await this.del(`/v1/webhooks?id=${id}`);
  }

  // Schedules
  async getSchedules(): Promise<{ schedules: Schedule[] }> {
    return this.get("/v1/schedules");
  }

  async createSchedule(
    config: Omit<Schedule, "id" | "createdAt" | "lastRun" | "nextRun">
  ): Promise<Schedule> {
    return this.post("/v1/schedules", config);
  }

  async updateSchedule(schedule: {
    id: string;
    name: string;
    cron: string;
    timezone?: string;
    profile: string;
    task: string;
    enabled: boolean;
  }): Promise<Schedule> {
    const resp = await fetch(`${API_BASE}/v1/schedules`, {
      method: "PUT",
      headers: this.headers(),
      body: JSON.stringify(schedule),
    });
    if (!resp.ok) throw new Error(`${resp.status}: ${await resp.text()}`);
    return resp.json();
  }

  async deleteSchedule(id: string): Promise<void> {
    await this.del(`/v1/schedules?id=${id}`);
  }

  // Plugins
  async getPlugins(): Promise<{ plugins: Plugin[] }> {
    return this.get("/v1/plugins");
  }

  // Health
  async getHealth(): Promise<HealthResponse> {
    return this.get("/v1/health");
  }
}

// Singleton
export const api = new AgentAPI("");
