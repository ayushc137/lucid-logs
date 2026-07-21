-- 003_scheduler.sql: scheduler jobs/state and demo seed/reset state.

CREATE TABLE IF NOT EXISTS scheduler_jobs (
  name TEXT PRIMARY KEY,
  schedule TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0,1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS scheduler_state (
  job_name TEXT PRIMARY KEY REFERENCES scheduler_jobs(name) ON DELETE CASCADE,
  last_run_at TEXT,
  last_success_at TEXT,
  last_error TEXT,
  next_run_at TEXT,
  run_count INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);

INSERT INTO scheduler_jobs(name,schedule,description,created_at,updated_at) VALUES
  ('check_retro_times','* * * * *','Per-minute check for users whose daily retrospective time has arrived',datetime('now'),datetime('now')),
  ('daily_maintenance','0 3 * * *','Daily cleanup/maintenance at 3 AM UTC (aggregates, streaks, temp data)',datetime('now'),datetime('now'))
ON CONFLICT(name) DO UPDATE SET schedule=excluded.schedule, description=excluded.description, updated_at=datetime('now');

CREATE TABLE IF NOT EXISTS demo_seed_state (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  seeded_at TEXT,
  seed_version TEXT,
  admin_user_id TEXT REFERENCES users(id),
  reset_at TEXT,
  reset_count INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS scheduler_state_next_run ON scheduler_state(next_run_at);
