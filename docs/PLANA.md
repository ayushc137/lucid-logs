# planning.md

## 0) Purpose and Principles
- Build on existing entities (tasks, categories, emotions) and SurrealDB schema.
- Keep logging friction low; encourage honest reflection (non-judgmental UX, partial credit).
- Make goals/habits/metrics first-class but interoperable with tasks/templates/retros.
- Prefer additive extensions (new tables/edges, versioning) over rewrites.
- Timezone-aware, user-level preferences (daily retro time).

## 1) Data Model & Schema (SurrealDB-oriented)

### 1.1 Core Entities (new/extended)
- `goal`
  - Fields: `id`, `user_id`, `title`, `description`, `status` (not_started|in_progress|completed|abandoned|postponed|paused), `type` (discrete|quantitative), `target` (number), `unit` (km, min, count, pages, kcal, tasks, hours_sleep, etc.), `deadline` (optional), `category_id` (optional), `privacy` (public|private|sensitive), `created_at`, `updated_at`.
  - Derived flags: `is_recurring` (false for one-time).
- `habit` (recurring goal profile)
  - Fields: `id`, `user_id`, `title`, `description`, `pattern` (frequency, quantity, time-bound, avoidance), `frequency` (times per day/week/month), `quantity_target` (number), `unit`, `time_window` (e.g., before 15:00), `days_of_week`, `grace_days`, `phase` (ramp|maintain), `streak_current`, `streak_best`, `category_id`, `privacy`, `created_at`, `updated_at`.
- `task_goal_impact` (edge between task and goal/habit)
  - Fields: `id`, `task_id`, `goal_id` (or `habit_id`), `polarity` (positive|negative|neutral), `magnitude` (-1.0..+1.0), `confidence` (0-1), `notes`, `auto_inferred` (bool), `created_at`.
- `template`
  - Fields: `id`, `user_id`, `title`, `default_duration`, `default_category_id`, `default_priority`, `default_emotions` (ids or zones), `default_goal_ids`, `default_habit_ids`, `default_positives`, `default_negatives`, `expected_impact` (goal_id → polarity/magnitude), `created_from_task_id` (optional), `created_at`, `updated_at`.
- `task` (extend existing)
  - Add: `template_id` (optional), `goal_impacts` (array of `task_goal_impact` refs), `habit_matches` (computed/denormalized list of habit ids matched), `status` (completed|postponed|canceled|not_started).
- `retro_daily`
  - Fields: `id`, `user_id`, `date`, `generated_at`, `config_time` (snapshot), `timezone`, aggregates (mood_avg, mood_distribution, tasks_done, tasks_postponed, tasks_canceled, habit_results, goal_impacts_pos/neg, category_time, emotion_spikes), `auto_insights` (array), `user_notes`, `overrides`.
- `retro_range`
  - Fields: `id`, `user_id`, `start_date`, `end_date`, aggregates (goal_progress, habit_performance, emotion_stats, category_balance, impact_stats), `insights`, `user_notes`, `created_at`.
- `user_pref`
  - Fields: `user_id`, `timezone`, `daily_retro_time` (local time), `notifications_enabled`, `privacy_defaults`.

### 1.2 Versioning / History
- `goal_history`
  - Fields: `id`, `goal_id`, `changed_at`, `change_type` (title|description|deadline|status|target|unit|privacy|tasks_linked|tasks_unlinked), `old_value`, `new_value`, `actor_id`.
- `habit_history`
  - Fields: `id`, `habit_id`, `changed_at`, `change_type` (frequency|quantity_target|time_window|grace_days|phase), `old_value`, `new_value`.
- `task_status_history` (optional)
  - Fields: `id`, `task_id`, `changed_at`, `old_status`, `new_status`.

### 1.3 Relationships
- `task` → `category` (existing)
- `task` → `emotions` (existing)
- `task` → `goal` via `task_goal_impact` (many-to-many with polarity/magnitude)
- `task` → `habit` (matched passively or spawned actively)
- `task` → `template` (optional)
- `goal` → `category` (optional) for domain balance
- `habit` → `category`
- `retro_daily`/`retro_range` references tasks, goals, habits via IDs in aggregates.

### 1.4 Indexes (SurrealDB)
- On `task`: `start_datetime`, `end_datetime`, `user_id`, `category_id`, `status`.
- On `task_goal_impact`: `task_id`, `goal_id`, `polarity`.
- On `goal`: `user_id`, `deadline`, `status`, `type`.
- On `habit`: `user_id`, `days_of_week`, `time_window`.
- On `goal_history`/`habit_history`: `goal_id`/`habit_id`, `changed_at`.
- On `retro_daily`: `user_id`, `date`.
- On `retro_range`: `user_id`, `start_date`, `end_date`.
- On `user_pref`: `user_id`.

### 1.5 Modeling patterns
- Many-to-many via edge tables (`task_goal_impact`, `task_template`, `task_habit_match` if needed).
- Versioning: append-only history tables; current state stored on main entity.
- Privacy: per-goal/habit flag; hide sensitive links in shared views.
- Timezone: store UTC + user timezone; for scheduling daily retro use `daily_retro_time` + timezone to compute UTC trigger.

## 2) API Design (Gin, high-level)

### 2.1 Goals
- `POST /goals` (create)
- `GET /goals` (list with filters: status, type, deadline range)
- `GET /goals/:id`
- `PUT /goals/:id` (updates; log to `goal_history`)
- `PATCH /goals/:id/status`
- `POST /goals/:id/links/tasks` (link tasks with impact payload)
- `DELETE /goals/:id/links/tasks/:taskId`
- `GET /goals/:id/history`
- Validation: ownership, privacy rules.

### 2.2 Habits
- `POST /habits`
- `GET /habits`
- `GET /habits/:id`
- `PUT /habits/:id` (log to `habit_history`)
- `PATCH /habits/:id/status` (active/paused)
- `GET /habits/:id/history`
- `POST /habits/:id/generate-tasks` (optional; spawn tasks based on schedule)
- Validation: ownership, pattern correctness (frequency, quantity, time windows).

### 2.3 Task impacts
- `POST /tasks/:id/impacts` (body: goal_id, polarity, magnitude, confidence, notes)
- `PUT /tasks/:id/impacts/:impactId`
- `DELETE /tasks/:id/impacts/:impactId`
- Auto-inference endpoint (optional advanced): `POST /tasks/:id/impacts/auto` to suggest from template/habit/keywords.

### 2.4 Templates
- `POST /templates` (create or from task via `from_task_id`)
- `GET /templates`
- `GET /templates/:id`
- `PUT /templates/:id`
- `POST /templates/:id/instantiate` (creates task; allows overrides)
- `POST /tasks/:id/save-as-template`
- Validation: ensure goal/habit associations belong to user.

### 2.5 Retrospectives
- Preferences: `PUT /user/prefs` (timezone, daily_retro_time)
- Daily retro:
  - `GET /retros/daily/:date`
  - `POST /retros/daily/:date/regenerate` (re-run aggregation)
  - `PATCH /retros/daily/:id` (user edits/notes/overrides)
- Custom range:
  - `POST /retros/range` (body: start_date, end_date)
  - `GET /retros/range/:id`
  - `PATCH /retros/range/:id` (user notes)
- Validation: date boundaries, ownership, max range limits for performance.

### 2.6 Analytics
- `GET /analytics/goals/progress?range=...`
- `GET /analytics/habits/streaks?range=...`
- `GET /analytics/categories/time?range=...`
- `GET /analytics/emotions/heatmap?range=...`
- `GET /analytics/tasks/completion-trends?range=...`
- `GET /analytics/impacts/net?goal_id=...&range=...`
- Paginate or bucket results for charts.

### 2.7 Middleware/Patterns
- Auth middleware (existing): ensure user_id context.
- Validation middleware: ensure numeric ranges (magnitude -1..1), time window formats, day-of-week enums.
- Rate limit on analytics endpoints.
- Error handling: consistent JSON error shape.

## 3) Feature Breakdown (Core / Nice / Advanced)

### 3.1 Goals
- Core: CRUD goals; link tasks with polarity/magnitude; status lifecycle; deadline; history logging.
- Nice: Privacy levels; partial progress inputs for quantitative goals; rollovers/reschedules tracking; category linkage.
- Advanced: Auto-suggestions of goals impacted by a task (NLP/keyword matching); goal “confidence” scoring; reminders based on drift.

### 3.2 Habits
- Core: Habit definitions (frequency/quantity/time-bound/avoidance), streak tracking, grace days, days-of-week, habit vs task linkage (passive match).
- Nice: Habit phases (ramp/maintain), habit stacking/routines, passive detection rules (e.g., if task title matches), per-habit dashboards.
- Advanced: Adaptive targets (increase/decrease based on success), anomaly detection (late completions), contextual nudges (time-of-day).

### 3.3 Metrics
- Core: Counts per category; time spent per category; mood averages; emotion distribution; habit success rate; streaks; net positive vs negative impact per goal.
- Nice: Emotional variability; time-of-day success for habits; balance across domains; honesty ratio (setbacks vs wins logged).
- Advanced: Flow proximity metrics, inertia/volatility, risk indicators (burnout/anxiety), personality/EQ estimates (later).

### 3.4 Task Templates
- Core: Template CRUD; instantiate task from template; link template to goals/habits; store originating template on task.
- Nice: Default emotions/expected zone; default positives/negatives prompts; default impact suggestions.
- Advanced: Template performance analytics (avg mood after using template); smart template suggestions based on usage; versioned templates.

### 3.5 Retrospectives
- Core: User-set daily retro time (timezone-aware); auto daily retro aggregation; editable retro entries.
- Nice: Custom range retros with insights; auto-insights (top positive/negative impacts, habits met/missed); prompts library.
- Advanced: Adaptive prompts based on patterns; anomaly alerts (spikes/drops); collaborative/shareable retros (respect privacy).

### 3.6 Analytics & Charts
- Core: Time-series (goal progress, tasks completed), stacked bars (impact pos/neg), heatmaps (emotion vs hour), calendar heatmap (habit streaks), radar (category balance).
- Nice: Postponement chains; success by day-of-week/time-of-day; impact per goal trend; emotion-category correlation.
- Advanced: Predictive streak break risk; anti-goal detection; cohort comparisons (per user personal history).

## 4) UX/Logic for Honest Goal Impact
- Impact scale: polarity (positive/negative/neutral) + magnitude slider (e.g., -1..+1) + confidence.
- Gentle prompts when marking negative: “What got in the way? Any learning?” avoid shaming.
- Partial credit: allow quantitative delta entry (planned 10 km, did 4 km).
- Anti-goal logging: quick toggle “This worked against X” with optional emotion tag.
- Privacy: sensitive goals/tasks hidden from shared views; per-goal privacy flag.
- Post-task nudge: “Did this help any goal?” with quick chips of top goals/habits.
- Auto-suggestions: from template/habit/category to prefill likely impacts; user can override.

## 5) Habit Model Details
- Patterns:
  - Frequency: `times_per_period` (period: day|week|month).
  - Quantity: `quantity_target`, `unit`, optional `per_period`.
  - Time-bound: `before_time`, `after_time`, `window_start`/`window_end`.
  - Avoidance: `avoid = true`; success if no violating task tagged/linked.
- Grace: `grace_days` per week/month; streak not broken if within grace.
- Phases: `phase` (ramp → maintain) with different targets.
- Stacking: parent habit with `subhabit_ids` to execute in sequence; optional `routine_template_id`.
- Matching tasks:
  - Passive: match by category, template, keywords, or explicit habit link on task.
  - Active: habit generates tasks (e.g., “Drink water” quick task every morning).
- Edge cases: timezones; crossing midnight windows; skip days; backfill logs.

## 6) Metrics Library (examples)
- Productivity: tasks completed, completion rate, postponement count, chain length.
- Time use: total/avg time per category, per goal, per habit.
- Emotional: avg valence/arousal/dominance; distribution by quadrant; variability; spikes.
- Impact: net impact per goal (Σ magnitude), anti-goal ratio, positive vs negative task counts.
- Habits: success rate, streak length, best streak, on-time vs late, success by DOW/time.
- Balance: category radar (normalized time or task count); neglected categories.
- Honesty: ratio of negative/anti-goal logs to total logs; presence of setbacks; edits vs deletes.
- Sleep/energy (if applicable): bedtime tasks, wake-up compliance, correlation with mood.

## 7) Analytics & Charts (suggested)
- Time-series line: goal progress %, quantitative target accumulation, mood over time.
- Stacked bar: positive/negative impacts per goal per day/week; task statuses (completed/postponed/canceled).
- Heatmap: emotion quadrant by hour; habit success by hour or DOW.
- Calendar heatmap: habit streaks; daily completion count.
- Radar/spider: category balance; life domains vs time.
- Histogram: task durations; bedtime deviations.
- Scatter: mood vs time spent in category; impact magnitude vs mood delta.
- Tables with sparklines: top goals advanced; top anti-goal behaviors.

## 8) Computation & Aggregation Strategy
- On-write denorm:
  - Store task duration, primary category, template_id.
  - Store goal impact net per task.
- Periodic jobs (cron):
  - Daily retro generation at user-local time -> compute: tasks summary, habits, goal impacts, emotion stats.
  - Habit streak recompute daily.
  - Materialized summaries per day per user: `{date, tasks_done, time_by_cat, net_impact_by_goal, emotion_stats}`.
- On-demand:
  - Custom range retro queries aggregate from daily summaries + raw tasks if needed.
- Indexing for ranges; cache/bucket results by day/week.
- Avoid heavy fan-out by precomputing daily aggregates.

## 9) Query Patterns (SurrealDB)
- Daily retro aggregation:
  - Fetch tasks for user where `start_datetime` within day (UTC adjust by timezone).
  - Join `task_goal_impact` for net impact.
  - Join emotions to compute distribution.
  - Match habits: evaluate pattern over day (frequency/quantity/time-bound/avoidance).
- Custom range:
  - Sum daily aggregates; fallback to raw tasks if missing.
- Streak calc:
  - Walk days from most recent backwards using daily habit result flags.

## 10) Step-by-Step Implementation Plan

### Phase 0: Foundations (Core)
1. Add `user_pref` (timezone, daily_retro_time).
2. Extend `task` statuses; add `task_goal_impact` table; indexes.
3. Goal CRUD + history logging; minimal fields.
4. Habit CRUD (frequency/quantity/time-bound/avoidance) + streak fields (computed later).
5. Template CRUD + instantiate; link template_id on tasks.

### Phase 1: Daily Retro MVP
1. Scheduler: per-user daily retro at configured time (compute UTC trigger).
2. Daily retro aggregation: tasks summary, emotion avg/distribution, basic habit evaluation (frequency count), net goal impact.
3. Retro retrieval & edit endpoints.

### Phase 2: Metrics & Analytics Core
1. Daily materialized summary (time_by_category, task_status_counts, net_impact_by_goal, emotion_stats).
2. Analytics endpoints for goals, habits, categories, emotions using daily summaries.
3. Charts: time-series, stacked bars, heatmaps (data endpoints only).

### Phase 3: Habits Upgrade
1. Grace days, phases, time-window compliance, avoidance evaluation.
2. Streak calculation job; calendar heatmap data endpoint.
3. Passive habit-task matching (category/keyword/template).

### Phase 4: Goals Upgrade
1. Quantitative targets accumulation; progress %.
2. Rollovers/reschedules tracking; status transitions; partial credit inputs.
3. Anti-goal logging support (negative impacts UX prompt).

### Phase 5: Templates Upgrade
1. Defaults for emotions, positives/negatives prompts, expected impacts.
2. Template performance analytics (avg mood after template tasks).

### Phase 6: Custom Range Retros
1. Range retro generation; aggregated insights; prompts library.
2. User notes/overrides persistence.

### Phase 7: Advanced Analytics
1. Postponement chains; honesty ratio; category balance radar.
2. Emotion-category correlation; success by DOW/time-of-day.
3. Optional predictive/alerting (experimental).

## 11) Edge Cases & Pitfalls
- Timezone/day-boundary: ensure consistent day slicing; store both UTC and user TZ.
- Habit windows crossing midnight; avoidance habits need “no event” detection.
- Privacy: hide sensitive goals/habits in shared/exported analytics.
- Magnitude misuse: clamp -1..1; default magnitude if omitted.
- Template changes not retroactively mutating existing tasks (immutable link snapshot).
- History growth: consider pruning or archiving old history if needed.

## 12) Data Validation Rules
- Goal quantitative: `target > 0`, `unit` from controlled list.
- Habit frequency: integers >0; grace_days ≤ period length.
- Time windows: HH:MM; days_of_week subset.
- Impact magnitude: -1..1; confidence 0..1.
- Retro date range: enforce max span (e.g., 1 year) for performance.

## 13) Ties to Emotional Wellbeing
- Always surface mood with tasks/goals/habits (emotion correlations).
- Prompts emphasize learning, not blame; highlight positive streaks and recoveries.
- Honesty metrics encourage logging setbacks; normalize mixed results.
- Balance view across categories (life domains) to prevent tunnel vision.

## 14) Quick Wins
- Add task → goal impact edge with magnitude; simple UI prompt after task complete.
- User prefs for retro time; daily retro cron that summarizes tasks/emotions.
- Goal CRUD + status + deadline; habit CRUD with simple frequency.
- Daily summaries table to accelerate analytics endpoints.