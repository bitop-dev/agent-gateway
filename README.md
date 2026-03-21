# Agent Gateway

Task routing, auth, webhooks, and scheduling for the [agent](https://github.com/bitop-dev/agent) distributed platform.

The gateway is the single entry point for all external interaction with the agent system.
It accepts tasks, routes them to workers, stores results in PostgreSQL, and provides
auth, webhooks, and cron scheduling.

## Quick start

```bash
# Start PostgreSQL
docker run -d --name agent-db -e POSTGRES_DB=agent -e POSTGRES_USER=agent \
  -e POSTGRES_PASSWORD=agent -p 5433:5432 postgres:17-alpine

# Start gateway
go run ./cmd/gateway \
  --addr :8080 \
  --dsn "postgres://agent:agent@localhost:5433/agent?sslmode=disable" \
  --registry "http://localhost:9080" \
  --admin-key "your-admin-key"

# Submit a task
curl -X POST http://localhost:8080/v1/tasks \
  -H "Authorization: Bearer your-admin-key" \
  -H "Content-Type: application/json" \
  -d '{"profile":"researcher","task":"Top AI story today"}'
```

## API

### Tasks

| Endpoint | Method | Auth | Description |
|---|---|---|---|
| `/v1/tasks` | POST | `tasks:write` | Submit a task (sync or async) |
| `/v1/tasks` | GET | `tasks:read` | List tasks (filter by `?status=`) |
| `/v1/tasks/{id}` | GET | `tasks:read` | Get task details + result |

Submit a task:
```json
POST /v1/tasks
{"profile": "researcher", "task": "Research Anthropic news", "async": false}
```

Async mode returns immediately with a task ID:
```json
POST /v1/tasks
{"profile": "researcher", "task": "...", "async": true}
→ {"id": "task-abc123", "status": "queued"}
```

### Workers

| Endpoint | Method | Auth | Description |
|---|---|---|---|
| `/v1/workers` | POST | none | Register/heartbeat a worker |
| `/v1/workers` | GET | none | List registered workers |
| `/v1/workers` | DELETE | none | Deregister (`?url=`) |

### Auth

| Endpoint | Method | Auth | Description |
|---|---|---|---|
| `/v1/auth/keys` | POST | `admin` | Create API key (returns raw key once) |
| `/v1/auth/keys` | GET | `admin` | List API keys |
| `/v1/auth/keys` | DELETE | `admin` | Revoke key (`?id=`) |

Scopes: `tasks:write`, `tasks:read`, `admin`

### Webhooks

| Endpoint | Method | Auth | Description |
|---|---|---|---|
| `/v1/webhooks` | POST | `admin` | Create webhook config |
| `/v1/webhooks` | GET | `admin` | List webhooks |
| `/v1/webhooks` | DELETE | `admin` | Delete webhook (`?id=`) |
| `/v1/webhooks/{path}` | POST | per-webhook token | Trigger a webhook |

Webhook template expansion:
```json
POST /v1/webhooks
{
  "name": "grafana-alerts",
  "path": "grafana",
  "profile": "researcher",
  "taskTemplate": "Alert: {{alertname}} on {{labels.host}}. Investigate.",
  "contextTemplate": {"team": "{{labels.team}}"},
  "authToken": "grafana-secret"
}
```

Trigger:
```json
POST /v1/webhooks/grafana
Authorization: Bearer grafana-secret
{"alertname": "High CPU", "labels": {"host": "prod-01", "team": "ict-aipe"}}
→ {"taskId": "task-xyz", "status": "queued"}
```

### Schedules

| Endpoint | Method | Auth | Description |
|---|---|---|---|
| `/v1/schedules` | POST | `admin` | Create schedule |
| `/v1/schedules` | GET | `admin` | List schedules |
| `/v1/schedules` | DELETE | `admin` | Delete schedule (`?id=`) |

```json
POST /v1/schedules
{
  "name": "daily-ops",
  "cron": "0 8 * * *",
  "timezone": "America/New_York",
  "profile": "grafana-alert-summary",
  "task": "Daily ops report for team ict-aipe",
  "context": {"team": "ict-aipe", "recipient": "nick@bitop.dev"}
}
```

### Discovery

| Endpoint | Method | Auth | Description |
|---|---|---|---|
| `/v1/agents` | GET | `tasks:read` | List available agents (workers + registry) |
| `/v1/health` | GET | none | Health check |

## Configuration

```
--addr          Listen address (default :8080)
--dsn           PostgreSQL connection string (or DATABASE_URL env)
--registry      agent-registry URL for profile discovery
--admin-key     Admin API key (or ADMIN_KEY env)
```

## Docker

```bash
docker run -p 8080:8080 ghcr.io/bitop-dev/agent-gateway:latest \
  --dsn "postgres://..." --admin-key "..."
```

## Related repos

| Repo | Purpose |
|---|---|
| [agent](https://github.com/bitop-dev/agent) | Framework, CLI, workers |
| [agent-registry](https://github.com/bitop-dev/agent-registry) | Plugin + profile packages |
| [agent-plugins](https://github.com/bitop-dev/agent-plugins) | Plugin packages |
| [agent-profiles](https://github.com/bitop-dev/agent-profiles) | Profile packages |
