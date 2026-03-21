# Agent Gateway

Task routing, worker management, and API gateway for the [agent](https://github.com/bitop-dev/agent) distributed platform.

## What it does

- Accepts task submissions via HTTP API
- Routes tasks to available workers (capability-based, load-aware)
- Tracks task lifecycle in PostgreSQL (queued → running → completed/failed)
- Worker registration and health monitoring
- Agent discovery (aggregates from workers + registry)
- Task history and audit

## Quick start

```bash
# Start PostgreSQL
docker run -d --name agent-db -e POSTGRES_DB=agent -e POSTGRES_USER=agent -e POSTGRES_PASSWORD=agent -p 5433:5432 postgres:17-alpine

# Start gateway
go run ./cmd/gateway --addr :8080 --dsn "postgres://agent:agent@localhost:5433/agent?sslmode=disable" --registry "http://localhost:9080"

# Submit a task
curl -X POST http://localhost:8080/v1/tasks \
  -H "Content-Type: application/json" \
  -d '{"profile":"researcher","task":"Top AI story today"}'
```

## API

| Endpoint | Method | Description |
|---|---|---|
| `/v1/health` | GET | Health check |
| `/v1/tasks` | POST | Submit a task |
| `/v1/tasks` | GET | List tasks (filter by ?status=) |
| `/v1/tasks/{id}` | GET | Get task details + result |
| `/v1/workers` | POST | Register/heartbeat a worker |
| `/v1/workers` | GET | List registered workers |
| `/v1/workers` | DELETE | Deregister a worker (?url=) |
| `/v1/agents` | GET | Discover available agents |

## Related repos

| Repo | Purpose |
|---|---|
| [agent](https://github.com/bitop-dev/agent) | Framework, CLI, workers |
| [agent-registry](https://github.com/bitop-dev/agent-registry) | Plugin + profile packages |
| [agent-plugins](https://github.com/bitop-dev/agent-plugins) | Plugin packages |
| [agent-profiles](https://github.com/bitop-dev/agent-profiles) | Profile packages |
