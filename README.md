# Agent Gateway

Task routing, auth, webhooks, scheduling, cost tracking, agent memory, and
web dashboard for the [agent](https://github.com/bitop-dev/agent) platform.

## Quick start

```bash
docker run -p 8080:8080 ghcr.io/bitop-dev/agent-gateway:0.4.4 \
  --dsn "postgres://..." --admin-key "..." --nats "nats://nats:4222"
```

## API

| Endpoint | Method | Auth | Description |
|---|---|---|---|
| `/v1/tasks` | POST | `tasks:write` | Submit task (sync or async) |
| `/v1/tasks` | GET | `tasks:read` | List tasks |
| `/v1/tasks/{id}` | GET | `tasks:read` | Task details + result |
| `/v1/tasks/parallel` | POST | `tasks:write` | Parallel across workers |
| `/v1/workers` | POST/GET/DELETE | none | Worker registration |
| `/v1/auth/keys` | POST/GET/DELETE | `admin` | API key management |
| `/v1/webhooks` | POST/GET/DELETE | `admin` | Webhook CRUD |
| `/v1/webhooks/{path}` | POST | webhook token | Trigger webhook |
| `/v1/schedules` | POST/GET/DELETE | `admin` | Cron scheduling |
| `/v1/memory?profile=X` | POST/GET/DELETE | `tasks:*` | Agent memory |
| `/v1/costs` | GET | `tasks:read` | Cost summary by profile |
| `/v1/costs/pricing` | GET/POST | `admin` | Model pricing (models.dev) |
| `/v1/agents` | GET | `tasks:read` | Available agents |
| `/v1/events` | GET | `tasks:read` | SSE event stream |
| `/v1/health` | GET | none | Health check |
| `/` | GET | none | Web dashboard |

## Key features

- **Routing** — capability-based, load-aware, prefers idle workers
- **Retries** — auto-retry on timeouts/502s, pick different worker each retry
- **Dead worker eviction** — unreachable workers removed immediately
- **Cost tracking** — pricing from [models.dev](https://models.dev) (1800+ models, per-million-token)
- **Agent memory** — per-profile persistent knowledge in PostgreSQL
- **NATS events** — `agent.task.*`, `agent.worker.*`, `agent.webhook.*`
- **Dashboard** — embedded web UI at `/`

## Configuration

```
--addr          :8080
--dsn           postgres://... (or DATABASE_URL)
--nats          nats://... (or NATS_URL, optional)
--registry      http://registry:9080
--admin-key     ... (or ADMIN_KEY)
```

## Related repos

| Repo | Purpose |
|---|---|
| [agent](https://github.com/bitop-dev/agent) | Framework, CLI, workers |
| [agent-registry](https://github.com/bitop-dev/agent-registry) | Plugin + profile packages |
| [agent-docs](https://github.com/bitop-dev/agent-docs) | Documentation |
