# Dashboard Plan

## Architecture

```
Development:
  Svelte + Vite + Bun → localhost:5173 (HMR)
    ↓ proxies API calls to
  Gateway → localhost:8080

Production:
  bun run build → static files → cmd/gateway/web/
    ↓ go:embed
  Gateway binary serves both API + dashboard on :8080
```

**Stack:**
- **Svelte 5** — reactive UI, small bundle, no virtual DOM
- **Vite** — fast dev server with HMR
- **Bun** — package manager and build runner
- **Tailwind CSS** — utility-first styling
- **Chart.js** or **lightweight SVG** — cost charts
- **EventSource API** — SSE for real-time events

**No separate service.** The built dashboard is embedded in the gateway
binary via `go:embed`. One binary, one port, one deployment.

---

## Pages

### 1. Overview (/)

The landing page. At-a-glance status of the entire platform.

```
┌─────────────────────────────────────────────────────┐
│  🤖 Agent Platform                     [API Key: •••]│
├─────────────────────────────────────────────────────┤
│                                                      │
│  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐  ┌──────┐  │
│  │  5   │  │  47  │  │  42  │  │   3  │  │ $0.12│  │
│  │Workers│  │Tasks │  │ Done │  │Failed│  │ Cost │  │
│  └──────┘  └──────┘  └──────┘  └──────┘  └──────┘  │
│                                                      │
│  Recent tasks                          [View all →]  │
│  ┌─────────────────────────────────────────────────┐ │
│  │ task-abc  researcher   completed  12.3s  2m ago │ │
│  │ task-def  orchestrator completed  28.1s  5m ago │ │
│  │ task-ghi  researcher   failed     0.2s   8m ago │ │
│  └─────────────────────────────────────────────────┘ │
│                                                      │
│  Live events                                         │
│  ┌─────────────────────────────────────────────────┐ │
│  │ 15:04:23 task.completed  researcher  12.3s      │ │
│  │ 15:04:11 task.started    researcher  worker-2   │ │
│  │ 15:04:10 task.submitted  researcher             │ │
│  └─────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

**Data sources:**
- `GET /v1/health` — worker count
- `GET /v1/tasks?limit=10` — recent tasks
- `GET /v1/costs?since=today` — daily cost
- `GET /v1/events` (SSE) — live event feed

**Auto-refresh:** Stats every 10s, events via SSE stream.

---

### 2. Tasks (/tasks)

Full task history with filtering, search, and detail view.

```
┌─────────────────────────────────────────────────────┐
│  Tasks                                               │
│                                                      │
│  [All ▾]  [Status ▾]  [Profile ▾]  [Search...]      │
│                                                      │
│  ┌─────────────────────────────────────────────────┐ │
│  │ ID          Profile       Status   Duration  TS │ │
│  │ task-abc123 researcher    ✅ done   12.3s   2m  │ │
│  │ task-def456 orchestrator  ✅ done   28.1s   5m  │ │
│  │ task-ghi789 researcher    ❌ fail   0.2s    8m  │ │
│  │ task-jkl012 alert-monitor ✅ done   2.7s   12m  │ │
│  └─────────────────────────────────────────────────┘ │
│                                                      │
│  ← 1 2 3 ... 10 →                                   │
└─────────────────────────────────────────────────────┘
```

**Click on a task → detail view:**

```
┌─────────────────────────────────────────────────────┐
│  ← Back to tasks                                     │
│                                                      │
│  Task: task-abc123                                   │
│  Profile: researcher                                 │
│  Status: ✅ completed                                │
│  Worker: 10.244.1.95:9898                            │
│  Duration: 12.3s                                     │
│  Model: gpt-oss-120b                                 │
│  Tokens: 1,234 in / 567 out                          │
│  Cost: $0.00                                         │
│  Created: 2026-03-21 15:04:10                        │
│                                                      │
│  Task:                                               │
│  ┌─────────────────────────────────────────────────┐ │
│  │ Top Anthropic story March 2026. 1 story, 2 sent │ │
│  └─────────────────────────────────────────────────┘ │
│                                                      │
│  Output:                                             │
│  ┌─────────────────────────────────────────────────┐ │
│  │ **Topic:** Anthropic                             │ │
│  │ **Key stories:**                                 │ │
│  │ - *DOD says Anthropic's 'red lines'...*          │ │
│  │   Source: https://techcrunch.com/...             │ │
│  └─────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

**Data source:** `GET /v1/tasks/{id}`

---

### 3. Submit task (/tasks/new)

Submit a task directly from the dashboard.

```
┌─────────────────────────────────────────────────────┐
│  New Task                                            │
│                                                      │
│  Profile:  [researcher        ▾]                     │
│                                                      │
│  Task:                                               │
│  ┌─────────────────────────────────────────────────┐ │
│  │ Research the latest Anthropic news and summarize │ │
│  │                                                  │ │
│  └─────────────────────────────────────────────────┘ │
│                                                      │
│  ☐ Async (return immediately)                        │
│                                                      │
│  [Submit]                                            │
│                                                      │
│  Available agents:                                   │
│  • researcher — web search + summarization           │
│  • orchestrator — discovery + delegation             │
│  • alert-monitor — reactive alert investigation      │
└─────────────────────────────────────────────────────┘
```

**Data sources:**
- `GET /v1/agents` — populate profile dropdown
- `POST /v1/tasks` — submit

---

### 4. Workers (/workers)

Worker status and health.

```
┌─────────────────────────────────────────────────────┐
│  Workers                                  [3 active] │
│                                                      │
│  ┌─────────────────────────────────────────────────┐ │
│  │ URL                  Status  Task     Completed  │ │
│  │ 10.244.0.83:9898     🟢 idle  —        12       │ │
│  │ 10.244.1.95:9898     🔵 busy  task-abc  8       │ │
│  │ 10.244.2.59:9898     🟢 idle  —        15       │ │
│  └─────────────────────────────────────────────────┘ │
│                                                      │
│  Registered profiles: researcher, orchestrator       │
└─────────────────────────────────────────────────────┘
```

**Data source:** `GET /v1/workers` — poll every 10s

---

### 5. Costs (/costs)

Cost breakdown by profile, model, and time.

```
┌─────────────────────────────────────────────────────┐
│  Costs                    [Last 7 days ▾]            │
│                                                      │
│  Total: $1.23  (45,678 tokens)                       │
│                                                      │
│  ┌─────────────────────────────────────────────────┐ │
│  │  [Bar chart: cost per day over last 7 days]     │ │
│  └─────────────────────────────────────────────────┘ │
│                                                      │
│  By profile:                                         │
│  ┌─────────────────────────────────────────────────┐ │
│  │ researcher      23 tasks  34,000 tokens  $0.89  │ │
│  │ orchestrator     5 tasks  11,000 tokens  $0.34  │ │
│  │ alert-monitor    2 tasks     678 tokens  $0.00  │ │
│  └─────────────────────────────────────────────────┘ │
│                                                      │
│  Model pricing:                      [Edit pricing]  │
│  ┌─────────────────────────────────────────────────┐ │
│  │ gpt-4o          $2.50/M in   $10.00/M out       │ │
│  │ gpt-oss-120b    $0.00/M in   $0.00/M out        │ │
│  │ claude-sonnet    $3.00/M in   $15.00/M out       │ │
│  └─────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

**Data sources:**
- `GET /v1/costs?since=...` — summary by profile
- `GET /v1/costs/pricing` — model pricing table
- `POST /v1/costs/pricing` — update pricing

---

### 6. Memory (/memory)

View and manage agent memory.

```
┌─────────────────────────────────────────────────────┐
│  Agent Memory                                        │
│                                                      │
│  Profile: [researcher ▾]                             │
│                                                      │
│  ┌─────────────────────────────────────────────────┐ │
│  │ Key                    Value              Updated│ │
│  │ last_report_date       2026-03-21         2m ago │ │
│  │ preferred_sources      techcrunch, verge  1h ago │ │
│  │ known_issues           OOM on k8s prod    3h ago │ │
│  └─────────────────────────────────────────────────┘ │
│                                                      │
│  [+ Add entry]                    [Clear all]        │
└─────────────────────────────────────────────────────┘
```

**Data source:** `GET /v1/memory?profile=X`

---

### 7. Webhooks (/webhooks)

Manage webhook configurations.

```
┌─────────────────────────────────────────────────────┐
│  Webhooks                              [+ New]       │
│                                                      │
│  ┌─────────────────────────────────────────────────┐ │
│  │ Name             Path         Profile    Enabled│ │
│  │ grafana-alerts   /grafana     researcher   ✅   │ │
│  │ alert-monitor    /alert-fired alert-mon.   ✅   │ │
│  └─────────────────────────────────────────────────┘ │
│                                                      │
│  Trigger URL: POST http://gateway:8080/v1/webhooks/  │
│               {path}                                 │
└─────────────────────────────────────────────────────┘
```

---

### 8. Schedules (/schedules)

Manage cron schedules.

```
┌─────────────────────────────────────────────────────┐
│  Schedules                             [+ New]       │
│                                                      │
│  ┌─────────────────────────────────────────────────┐ │
│  │ Name           Cron          Profile   Next Run │ │
│  │ daily-ops      0 8 * * *     grafana   8:00 AM  │ │
│  │ weekly-report  0 9 * * 1     research  Mon 9AM  │ │
│  └─────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

---

## Navigation

Sidebar navigation:

```
🤖 Agent Platform
─────────────────
📊 Overview
📋 Tasks
👷 Workers
🔍 Agents
💰 Costs
🧠 Memory
🔗 Webhooks
⏰ Schedules
⚙️ Settings
```

---

## Technical implementation

### Project structure

```
agent-gateway/
  dashboard/                    # Svelte project
    src/
      lib/
        api.ts                  # Gateway API client
        stores.ts               # Svelte stores (tasks, workers, events)
        sse.ts                  # SSE event stream connection
      routes/
        +page.svelte            # Overview
        tasks/
          +page.svelte          # Task list
          [id]/+page.svelte     # Task detail
          new/+page.svelte      # Submit task
        workers/+page.svelte
        costs/+page.svelte
        memory/+page.svelte
        webhooks/+page.svelte
        schedules/+page.svelte
      components/
        Sidebar.svelte
        StatCard.svelte
        TaskTable.svelte
        EventFeed.svelte
        CostChart.svelte
        StatusBadge.svelte
      app.html
      app.css                   # Tailwind
    static/
    vite.config.ts
    svelte.config.js
    tailwind.config.js
    package.json
    bun.lockb
  cmd/gateway/
    web/                        # Build output (go:embed target)
      index.html
      assets/
    main.go                     # go:embed web/*
  ...
```

### Build pipeline

```bash
# Development (hot reload)
cd dashboard
bun install
bun run dev          # Vite dev server on :5173, proxies /v1/* to :8080

# Production build
bun run build        # outputs to ../cmd/gateway/web/
cd ..
go build ./cmd/gateway  # embeds the built dashboard
```

### Vite config

```typescript
// vite.config.ts
export default defineConfig({
  build: {
    outDir: '../cmd/gateway/web',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/v1': 'http://localhost:8080',
    },
  },
});
```

### API client

```typescript
// src/lib/api.ts
const API_BASE = '';  // same origin in production

class AgentAPI {
  private key: string;

  constructor(key: string) { this.key = key; }

  private headers() {
    return {
      'Authorization': `Bearer ${this.key}`,
      'Content-Type': 'application/json',
    };
  }

  async getTasks(opts?: { status?: string; limit?: number }) { ... }
  async getTask(id: string) { ... }
  async submitTask(profile: string, task: string, async?: boolean) { ... }
  async submitParallel(tasks: Array<{profile: string; task: string}>) { ... }
  async getWorkers() { ... }
  async getAgents() { ... }
  async getCosts(since?: string) { ... }
  async getPricing() { ... }
  async setPricing(model: string, input: number, output: number) { ... }
  async getMemory(profile: string, key?: string) { ... }
  async setMemory(profile: string, key: string, value: string) { ... }
  async deleteMemory(profile: string, key?: string) { ... }
  async getWebhooks() { ... }
  async createWebhook(config: WebhookConfig) { ... }
  async getSchedules() { ... }
  async createSchedule(config: ScheduleConfig) { ... }
  async getHealth() { ... }
}
```

### SSE event stream

```typescript
// src/lib/sse.ts
// Note: SSE doesn't support custom headers.
// Workaround: pass token as query param, gateway checks both.
function connectEvents(apiKey: string) {
  const es = new EventSource(`/v1/events?token=${apiKey}`);
  es.addEventListener('agent.task.completed', (e) => { ... });
  es.addEventListener('agent.task.failed', (e) => { ... });
  es.addEventListener('agent.worker.joined', (e) => { ... });
  return es;
}
```

**SSE auth workaround:** The gateway's `/v1/events` endpoint should also
accept `?token=` as a query parameter since EventSource can't set headers.

### Theme

Dark theme matching the current dashboard aesthetic:
- Background: `#0d1117`
- Cards: `#161b22` with `#30363d` borders
- Text: `#c9d1d9`
- Accent: `#58a6ff`
- Success: `#3fb950`
- Error: `#f85149`
- Warning: `#d29922`

Responsive: works on desktop and tablet. Mobile is secondary.

---

## Implementation phases

### Phase 1: Foundation
- Svelte project setup with Vite + Bun + Tailwind
- API client library
- Sidebar navigation + routing
- Overview page with stat cards
- Build pipeline → `cmd/gateway/web/`

### Phase 2: Core pages
- Task list with filtering and pagination
- Task detail view with output rendering
- Worker status page
- Agent discovery page

### Phase 3: Actions
- Submit task form
- SSE event stream (real-time feed)
- Webhook CRUD
- Schedule CRUD

### Phase 4: Insights
- Cost charts (daily, by profile, by model)
- Memory browser and editor
- Pricing management

### Phase 5: Polish
- Markdown rendering for task output
- Responsive layout
- Loading states and error handling
- Toast notifications for events
- Keyboard shortcuts

---

## SSE auth gateway change needed

Add query parameter auth for SSE:

```go
// In requireAuth middleware, also check ?token= query param
token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
if token == "" {
    token = r.URL.Query().Get("token")
}
```

This is a small gateway change needed before the dashboard SSE works.
