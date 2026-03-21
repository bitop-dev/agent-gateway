-- Tasks
CREATE TABLE IF NOT EXISTS tasks (
  id            TEXT PRIMARY KEY,
  profile       TEXT NOT NULL,
  task          TEXT NOT NULL,
  context       JSONB,
  status        TEXT NOT NULL DEFAULT 'queued',
  priority      TEXT NOT NULL DEFAULT 'normal',
  worker_url    TEXT,
  output        TEXT,
  error         TEXT,
  tool_calls    INTEGER DEFAULT 0,
  duration_ms   INTEGER,
  callback_url  TEXT,
  created_at    TIMESTAMPTZ DEFAULT now(),
  started_at    TIMESTAMPTZ,
  completed_at  TIMESTAMPTZ,
  metadata      JSONB
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_profile ON tasks(profile);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);

-- Workers
CREATE TABLE IF NOT EXISTS workers (
  url             TEXT PRIMARY KEY,
  profiles        TEXT[],
  capabilities    TEXT[],
  status          TEXT DEFAULT 'active',
  current_task    TEXT,
  tasks_completed INTEGER DEFAULT 0,
  registered_at   TIMESTAMPTZ DEFAULT now(),
  last_heartbeat  TIMESTAMPTZ DEFAULT now()
);

-- API keys
CREATE TABLE IF NOT EXISTS api_keys (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  key_hash    TEXT NOT NULL UNIQUE,
  scopes      TEXT[] NOT NULL DEFAULT '{"tasks:write","tasks:read"}',
  created_at  TIMESTAMPTZ DEFAULT now(),
  last_used   TIMESTAMPTZ,
  revoked     BOOLEAN DEFAULT false
);

-- Schedules
CREATE TABLE IF NOT EXISTS schedules (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  cron_expr   TEXT NOT NULL,
  timezone    TEXT DEFAULT 'UTC',
  profile     TEXT NOT NULL,
  task        TEXT NOT NULL,
  context     JSONB,
  enabled     BOOLEAN DEFAULT true,
  last_run    TIMESTAMPTZ,
  next_run    TIMESTAMPTZ,
  created_at  TIMESTAMPTZ DEFAULT now()
);

-- Webhooks
CREATE TABLE IF NOT EXISTS webhooks (
  id                TEXT PRIMARY KEY,
  name              TEXT NOT NULL,
  path              TEXT NOT NULL UNIQUE,
  profile           TEXT NOT NULL,
  task_template     TEXT NOT NULL,
  context_template  JSONB,
  auth_token        TEXT,
  enabled           BOOLEAN DEFAULT true,
  created_at        TIMESTAMPTZ DEFAULT now()
);

-- Audit log
CREATE TABLE IF NOT EXISTS audit_log (
  id          SERIAL PRIMARY KEY,
  action      TEXT NOT NULL,
  actor       TEXT,
  resource_id TEXT,
  resource    TEXT,
  details     JSONB,
  created_at  TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_created_at ON audit_log(created_at);

-- Cost tracking
CREATE TABLE IF NOT EXISTS cost_tracking (
  id             SERIAL PRIMARY KEY,
  task_id        TEXT,
  profile        TEXT NOT NULL,
  model          TEXT,
  input_tokens   INTEGER DEFAULT 0,
  output_tokens  INTEGER DEFAULT 0,
  total_tokens   INTEGER DEFAULT 0,
  estimated_cost NUMERIC(10,6) DEFAULT 0,
  created_at     TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cost_profile ON cost_tracking(profile);
CREATE INDEX IF NOT EXISTS idx_cost_created_at ON cost_tracking(created_at);

-- Agent memory (per-profile persistent knowledge)
CREATE TABLE IF NOT EXISTS agent_memory (
  id          SERIAL PRIMARY KEY,
  profile     TEXT NOT NULL,
  key         TEXT NOT NULL,
  value       TEXT NOT NULL,
  created_at  TIMESTAMPTZ DEFAULT now(),
  updated_at  TIMESTAMPTZ DEFAULT now(),
  UNIQUE(profile, key)
);

-- v0.5.2: Add token tracking to tasks
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS model TEXT;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS input_tokens INTEGER DEFAULT 0;
ALTER TABLE tasks ADD COLUMN IF NOT EXISTS output_tokens INTEGER DEFAULT 0;
