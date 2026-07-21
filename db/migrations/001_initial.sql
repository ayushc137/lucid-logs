PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL COLLATE NOCASE UNIQUE,
  pass TEXT NOT NULL,
  is_admin INTEGER NOT NULL DEFAULT 0 CHECK (is_admin IN (0,1)),
  preferences TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS categories (
  id TEXT PRIMARY KEY, created_by TEXT NOT NULL REFERENCES users(id),
  name TEXT NOT NULL, color TEXT NOT NULL,
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS categories_user_name_active ON categories(created_by, name COLLATE NOCASE) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS units (
  id TEXT PRIMARY KEY, created_by TEXT REFERENCES users(id), name TEXT NOT NULL,
  symbol TEXT NOT NULL, type TEXT NOT NULL, is_system INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT
);

CREATE TABLE IF NOT EXISTS emotions (
  id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, emoji TEXT, description TEXT,
  category TEXT, quadrant TEXT, valence REAL NOT NULL DEFAULT 0,
  arousal REAL NOT NULL DEFAULT 0, intensity REAL, x REAL, y REAL,
  color TEXT, synonyms TEXT NOT NULL DEFAULT '[]', metadata TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS goals (
  id TEXT PRIMARY KEY, created_by TEXT NOT NULL REFERENCES users(id),
  title TEXT NOT NULL, description TEXT, icon TEXT, color TEXT,
  status TEXT NOT NULL DEFAULT 'active', priority TEXT,
  category_id TEXT REFERENCES categories(id), target TEXT, recurrence TEXT, schedule TEXT,
  start_date TEXT, deadline TEXT, completed_at TEXT, current_value REAL,
  current_streak INTEGER NOT NULL DEFAULT 0, longest_streak INTEGER NOT NULL DEFAULT 0,
  last_completed_date TEXT, metadata TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT
);
CREATE INDEX IF NOT EXISTS goals_user_status ON goals(created_by,status);
CREATE INDEX IF NOT EXISTS goals_user_deleted ON goals(created_by,deleted_at);

CREATE TABLE IF NOT EXISTS activities (
  id TEXT PRIMARY KEY, created_by TEXT NOT NULL REFERENCES users(id),
  title TEXT NOT NULL, description TEXT, icon TEXT, color TEXT,
  mode TEXT NOT NULL DEFAULT 'instant', duration INTEGER, pinned INTEGER NOT NULL DEFAULT 0,
  category_id TEXT REFERENCES categories(id), priority TEXT,
  schedule TEXT, timer_config TEXT, default_task TEXT,
  use_count INTEGER NOT NULL DEFAULT 0, last_used_at TEXT,
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT
);
CREATE INDEX IF NOT EXISTS activities_user_deleted ON activities(created_by,deleted_at);

CREATE TABLE IF NOT EXISTS tasks (
  id TEXT PRIMARY KEY, created_by TEXT NOT NULL REFERENCES users(id),
  title TEXT NOT NULL, note TEXT, journal TEXT,
  start_date TEXT NOT NULL, end_date TEXT NOT NULL, duration INTEGER,
  completed INTEGER NOT NULL DEFAULT 0, completed_at TEXT,
  priority TEXT, status TEXT, category_id TEXT REFERENCES categories(id),
  activity_id TEXT REFERENCES activities(id), activity_mode TEXT,
  emotion_id TEXT REFERENCES emotions(id), positives TEXT, negatives TEXT,
  inferred_emotion TEXT, quantity_value REAL, unit_id TEXT REFERENCES units(id),
  metadata TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL, deleted_at TEXT
);
CREATE INDEX IF NOT EXISTS tasks_user_start ON tasks(created_by,start_date);
CREATE INDEX IF NOT EXISTS tasks_user_completed ON tasks(created_by,completed);
CREATE INDEX IF NOT EXISTS tasks_user_deleted ON tasks(created_by,deleted_at);

CREATE TABLE IF NOT EXISTS retrospectives (
  id TEXT PRIMARY KEY, created_by TEXT NOT NULL REFERENCES users(id),
  retro_type TEXT NOT NULL, start_date TEXT NOT NULL, end_date TEXT NOT NULL,
  responses TEXT NOT NULL DEFAULT '{}', generated_at TEXT, status TEXT,
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL, deleted_at TEXT
);
CREATE INDEX IF NOT EXISTS retrospectives_user_start ON retrospectives(created_by,start_date);

CREATE TABLE IF NOT EXISTS task_emotions (
  id TEXT PRIMARY KEY, task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  emotion_id TEXT NOT NULL REFERENCES emotions(id), type TEXT NOT NULL,
  text TEXT, created_at TEXT NOT NULL, UNIQUE(task_id,emotion_id,type)
);

CREATE TABLE IF NOT EXISTS task_goals (
  id TEXT PRIMARY KEY, task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  impact_type TEXT NOT NULL DEFAULT 'positive', quantity_value REAL,
  unit_id TEXT REFERENCES units(id), is_milestone INTEGER NOT NULL DEFAULT 0,
  milestone_label TEXT, milestone_order INTEGER, notes TEXT,
  source TEXT NOT NULL DEFAULT 'manual', created_at TEXT NOT NULL,
  UNIQUE(task_id,goal_id)
);
CREATE INDEX IF NOT EXISTS task_goals_goal ON task_goals(goal_id);

CREATE TABLE IF NOT EXISTS created_from_activity (
  task_id TEXT PRIMARY KEY REFERENCES tasks(id) ON DELETE CASCADE,
  activity_id TEXT NOT NULL REFERENCES activities(id), mode TEXT NOT NULL DEFAULT 'instant',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS activity_goals (
  id TEXT PRIMARY KEY, activity_id TEXT NOT NULL REFERENCES activities(id) ON DELETE CASCADE,
  goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  auto_link_tasks INTEGER NOT NULL DEFAULT 1, quantity_multiplier REAL NOT NULL DEFAULT 1,
  default_quantity REAL, default_impact TEXT NOT NULL DEFAULT 'positive',
  created_at TEXT NOT NULL, UNIQUE(activity_id,goal_id)
);

CREATE TABLE IF NOT EXISTS goal_children (
  parent_goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  child_goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  sort_order INTEGER NOT NULL DEFAULT 0, required INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL, PRIMARY KEY(parent_goal_id,child_goal_id),
  CHECK(parent_goal_id <> child_goal_id)
);

CREATE TABLE IF NOT EXISTS activity_logs (
  id TEXT PRIMARY KEY, entity_type TEXT, entity_id TEXT,
  activity_id TEXT REFERENCES activities(id), task_id TEXT REFERENCES tasks(id),
  created_by TEXT NOT NULL REFERENCES users(id), event_type TEXT NOT NULL,
  mode TEXT, started_at TEXT, ended_at TEXT, duration INTEGER, quantity REAL,
  changes TEXT, metadata TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS activity_logs_user_created ON activity_logs(created_by,created_at);

CREATE TABLE IF NOT EXISTS goal_logs (
  id TEXT PRIMARY KEY, goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  snapshot_id TEXT, created_by TEXT NOT NULL REFERENCES users(id),
  event_type TEXT NOT NULL, changes TEXT, triggered_by_task_id TEXT REFERENCES tasks(id),
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS goal_snapshots (
  id TEXT PRIMARY KEY, goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  created_by TEXT NOT NULL REFERENCES users(id), snapshot TEXT NOT NULL, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS goal_daily_stats (
  goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE, date TEXT NOT NULL,
  created_by TEXT NOT NULL REFERENCES users(id), daily_value REAL NOT NULL DEFAULT 0,
  cumulative_value REAL NOT NULL DEFAULT 0, contribution_count INTEGER NOT NULL DEFAULT 0,
  target_value REAL, streak_at_date INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'pending', created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
  PRIMARY KEY(goal_id,date)
);
CREATE TABLE IF NOT EXISTS goal_period_snapshots (
  id TEXT PRIMARY KEY, goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  created_by TEXT NOT NULL REFERENCES users(id), period_type TEXT NOT NULL,
  period_key TEXT NOT NULL, start_date TEXT NOT NULL, end_date TEXT NOT NULL,
  snapshot TEXT NOT NULL, UNIQUE(goal_id,period_type,period_key)
);
CREATE TABLE IF NOT EXISTS streak_history (
  id TEXT PRIMARY KEY, goal_id TEXT NOT NULL REFERENCES goals(id) ON DELETE CASCADE,
  created_by TEXT NOT NULL REFERENCES users(id), date TEXT NOT NULL,
  event TEXT NOT NULL, streak_value INTEGER, created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS agg_daily (
  user_id TEXT NOT NULL REFERENCES users(id), date TEXT NOT NULL,
  task_count INTEGER NOT NULL DEFAULT 0, completed_count INTEGER NOT NULL DEFAULT 0,
  duration INTEGER NOT NULL DEFAULT 0, metrics TEXT NOT NULL DEFAULT '{}',
  PRIMARY KEY(user_id,date)
);
CREATE TABLE IF NOT EXISTS timer_sessions (
  id TEXT PRIMARY KEY, activity_id TEXT REFERENCES activities(id),
  created_by TEXT NOT NULL REFERENCES users(id), status TEXT NOT NULL,
  started_at TEXT NOT NULL, ended_at TEXT, counters TEXT NOT NULL DEFAULT '{}',
  config TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
