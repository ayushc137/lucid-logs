# Activity System Design

## Overview

Activities are reusable task blueprints that make logging tasks fast and effortless. Every Activity supports three modes of use:

| Mode | Description | UX |
|------|-------------|-----|
| **Instant Log** | Log completed task in past | One tap → edit quantity only → done |
| **Schedule** | Plan task with defaults | Opens task form pre-filled with all defaults |
| **Timer (Flowmodoro)** | Track ongoing work | Start → work → stop → auto-creates task |

## Core Principles

1. **Goal-Activity Link**: Goals can auto-create an Activity for easy task logging
2. **Multi-Goal Support**: One Activity can link to multiple goals (e.g., "Morning Run" → Fitness + Cardio goals)
3. **Minimal Friction**: Instant log should be ONE action for quantity-based activities
4. **Smart Defaults**: Activities learn from usage to suggest better defaults

---

## Data Model

### Activity Table

```sql
DEFINE TABLE activities SCHEMAFULL;

-- Identity
DEFINE FIELD title ON activities TYPE string ASSERT $value != NONE;
DEFINE FIELD icon ON activities TYPE option<string>;  -- Emoji
DEFINE FIELD description ON activities TYPE option<string>;  -- Template for task journal

-- Task Defaults (pre-fill when creating task)
DEFINE FIELD default_duration ON activities TYPE option<int>;  -- seconds
DEFINE FIELD default_emotion_id ON activities TYPE option<record<emotions>>;
DEFINE FIELD default_priority ON activities TYPE int DEFAULT 3;
DEFINE FIELD default_completed ON activities TYPE bool DEFAULT true;  -- true for instant log

-- Quantity Settings (for measurable activities)
DEFINE FIELD quantity_enabled ON activities TYPE bool DEFAULT false;
DEFINE FIELD quantity_default ON activities TYPE option<float>;
DEFINE FIELD quantity_step ON activities TYPE option<float>;
DEFINE FIELD quantity_unit_id ON activities TYPE option<record<units>>;  -- fallback if no goal

-- Goal Link Defaults (applied when creating task)
DEFINE FIELD default_impact ON activities TYPE string DEFAULT "positive";  -- positive, negative, neutral

-- Display & Organization
DEFINE FIELD category_id ON activities TYPE option<record<categories>>;
DEFINE FIELD pinned ON activities TYPE bool DEFAULT false;  -- Show in quick access bar
DEFINE FIELD sort_order ON activities TYPE int DEFAULT 0;

-- Usage Statistics
DEFINE FIELD use_count ON activities TYPE int DEFAULT 0;
DEFINE FIELD last_used_at ON activities TYPE option<datetime>;

-- Timer/Flowmodoro State (for active sessions)
DEFINE FIELD active_session ON activities TYPE option<object>;  -- { task_id, started_at, breaks: [] }

-- Metadata
DEFINE FIELD created_by ON activities TYPE record<users>;
DEFINE FIELD created_at ON activities TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON activities TYPE datetime DEFAULT time::now();
DEFINE FIELD deleted_at ON activities TYPE option<datetime>;

-- Indexes
DEFINE INDEX idx_activities_user ON activities FIELDS created_by;
DEFINE INDEX idx_activities_pinned ON activities FIELDS created_by, pinned;
```

### Activity-Goal Link Table (Many-to-Many)

```sql
-- Relation: activity -> activity_goals -> goal
DEFINE TABLE activity_goals SCHEMAFULL;

DEFINE FIELD in ON activity_goals TYPE record<activities>;
DEFINE FIELD out ON activity_goals TYPE record<goals>;

-- Link Configuration
DEFINE FIELD auto_link_tasks ON activity_goals TYPE bool DEFAULT true;
DEFINE FIELD quantity_multiplier ON activity_goals TYPE float DEFAULT 1.0;
DEFINE FIELD default_impact ON activity_goals TYPE string DEFAULT "positive";

-- Metadata
DEFINE FIELD created_at ON activity_goals TYPE datetime DEFAULT time::now();

-- Usage: RELATE activities:abc -> activity_goals -> goals:xyz
```

---

## Go Backend Models

### `internal/features/activities/models.go`

```go
package activities

import (
    "time"
    "github.com/lucid-logs/go-backend/internal/features/categories"
    "github.com/lucid-logs/go-backend/internal/features/goals"
    "github.com/lucid-logs/go-backend/internal/features/units"
)

// Activity represents a reusable task blueprint.
type Activity struct {
    ID        string `json:"id,omitempty"`
    CreatedBy string `json:"-"`

    // Identity
    Title       string `json:"title"`
    Icon        string `json:"icon,omitempty"`
    Description string `json:"description,omitempty"`

    // Task Defaults
    DefaultDuration  int    `json:"default_duration,omitempty"`   // seconds
    DefaultEmotionID string `json:"default_emotion_id,omitempty"`
    DefaultPriority  int    `json:"default_priority"`
    DefaultCompleted bool   `json:"default_completed"`  // true = instant log creates completed task

    // Quantity Settings
    QuantityEnabled bool    `json:"quantity_enabled"`
    QuantityDefault float64 `json:"quantity_default,omitempty"`
    QuantityStep    float64 `json:"quantity_step,omitempty"`
    QuantityUnitID  string  `json:"quantity_unit_id,omitempty"`

    // Goal Link Defaults
    DefaultImpact string `json:"default_impact"` // positive, negative, neutral

    // Display
    Pinned    bool `json:"pinned"`
    SortOrder int  `json:"sort_order"`

    // Stats
    UseCount   int        `json:"use_count"`
    LastUsedAt *time.Time `json:"last_used_at,omitempty"`

    // Active Timer Session (if any)
    ActiveSession *TimerSession `json:"active_session,omitempty"`

    // Metadata
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
    DeletedAt *time.Time `json:"deleted_at,omitempty"`

    // Populated via graph queries
    Goals    []*ActivityGoalLink  `json:"goals,omitempty"`
    Category *categories.Category `json:"category,omitempty"`
}

// ActivityGoalLink represents link between activity and goal with config.
type ActivityGoalLink struct {
    GoalID             string  `json:"goal_id"`
    AutoLinkTasks      bool    `json:"auto_link_tasks"`
    QuantityMultiplier float64 `json:"quantity_multiplier"`
    DefaultImpact      string  `json:"default_impact"`

    // Populated goal details
    Goal *goals.Goal `json:"goal,omitempty"`
}

// TimerSession tracks an active Flowmodoro session.
type TimerSession struct {
    TaskID    string         `json:"task_id"`
    StartedAt time.Time      `json:"started_at"`
    Breaks    []TimerBreak   `json:"breaks,omitempty"`
}

// TimerBreak records a break during Flowmodoro.
type TimerBreak struct {
    StartedAt time.Time  `json:"started_at"`
    EndedAt   *time.Time `json:"ended_at,omitempty"`
    Duration  int        `json:"duration,omitempty"` // seconds
}
```

---

## API Endpoints

### Activities CRUD

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/activities` | List all activities |
| GET | `/activities/pinned` | Get pinned activities for quick bar |
| GET | `/activities/:id` | Get single activity |
| POST | `/activities` | Create activity |
| PUT | `/activities/:id` | Update activity |
| DELETE | `/activities/:id` | Delete activity |

### Activity Actions

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/activities/:id/instant` | **Instant Log** - Create completed task |
| POST | `/activities/:id/schedule` | **Schedule** - Get pre-filled task data |
| POST | `/activities/:id/timer/start` | **Timer Start** - Begin Flowmodoro session |
| POST | `/activities/:id/timer/stop` | **Timer Stop** - End session, create task |
| POST | `/activities/:id/timer/break` | **Timer Break** - Record break |

### Goal Link Management

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/activities/:id/goals` | Link activity to goal |
| DELETE | `/activities/:id/goals/:goalId` | Unlink goal |
| PUT | `/activities/:id/goals/:goalId` | Update link config |

---

## Request/Response Types

### Instant Log Request

```go
// InstantLogRequest creates a completed task immediately.
type InstantLogRequest struct {
    // Override defaults (all optional)
    Quantity  *float64 `json:"quantity,omitempty"`   // Override quantity
    Notes     string   `json:"notes,omitempty"`      // Add to description
    Timestamp *string  `json:"timestamp,omitempty"`  // Default: now
}

// InstantLogResponse returns the created task.
type InstantLogResponse struct {
    Task         *tasks.Task        `json:"task"`
    GoalsUpdated []GoalUpdateSummary `json:"goals_updated,omitempty"`
}

type GoalUpdateSummary struct {
    GoalID       string  `json:"goal_id"`
    GoalTitle    string  `json:"goal_title"`
    ValueAdded   float64 `json:"value_added"`
    NewTotal     float64 `json:"new_total"`
    TargetValue  float64 `json:"target_value"`
    IsCompleted  bool    `json:"is_completed"`
}
```

### Schedule Request

```go
// ScheduleRequest returns pre-filled task data (no task created).
type ScheduleRequest struct {
    StartDate string `json:"start_date,omitempty"` // Suggested start
}

// ScheduleResponse returns data to pre-fill task form.
type ScheduleResponse struct {
    Activity   *Activity            `json:"activity"`
    TaskDefaults TaskDefaults       `json:"task_defaults"`
    GoalLinks  []GoalLinkDefault    `json:"goal_links"`
}

type TaskDefaults struct {
    Title       string  `json:"title"`
    Journal     string  `json:"journal,omitempty"`
    Duration    int     `json:"duration,omitempty"`
    Priority    int     `json:"priority"`
    CategoryID  string  `json:"category_id,omitempty"`
    EmotionID   string  `json:"emotion_id,omitempty"`
    Quantity    float64 `json:"quantity,omitempty"`
    QuantityUnit string `json:"quantity_unit,omitempty"`
}

type GoalLinkDefault struct {
    GoalID        string  `json:"goal_id"`
    GoalTitle     string  `json:"goal_title"`
    GoalIcon      string  `json:"goal_icon,omitempty"`
    ImpactType    string  `json:"impact_type"`
    Quantity      float64 `json:"quantity,omitempty"`
    QuantityUnit  string  `json:"quantity_unit,omitempty"`
}
```

### Timer Endpoints

```go
// TimerStartRequest begins a Flowmodoro session.
type TimerStartRequest struct {
    Notes string `json:"notes,omitempty"` // Initial task notes
}

// TimerStartResponse returns the in-progress task.
type TimerStartResponse struct {
    Task      *tasks.Task   `json:"task"`
    SessionID string        `json:"session_id"`
    StartedAt time.Time     `json:"started_at"`
}

// TimerStopRequest ends the session.
type TimerStopRequest struct {
    Notes string `json:"notes,omitempty"` // Additional notes to append
}

// TimerStopResponse returns completed task with Flowmodoro breakdown.
type TimerStopResponse struct {
    Task           *tasks.Task         `json:"task"`
    TotalDuration  int                 `json:"total_duration"`   // seconds
    WorkDuration   int                 `json:"work_duration"`    // seconds (excluding breaks)
    Breaks         []TimerBreak        `json:"breaks"`
    GoalsUpdated   []GoalUpdateSummary `json:"goals_updated,omitempty"`
}
```

---

## Goal Auto-Creates Activity

When creating a goal, optionally auto-create an activity:

### Goal CreateRequest Extension

```go
type CreateRequest struct {
    // ... existing fields ...

    // Activity creation (optional)
    CreateActivity *CreateActivityFromGoal `json:"create_activity,omitempty"`
}

type CreateActivityFromGoal struct {
    Enabled         bool   `json:"enabled"`          // Create activity for this goal
    Pinned          bool   `json:"pinned,omitempty"` // Pin to quick bar
    DefaultDuration int    `json:"default_duration,omitempty"`
    Icon            string `json:"icon,omitempty"`   // Override goal icon
}
```

### Auto-Creation Logic

When `create_activity.enabled = true`:

1. Create Activity with:
   - `title` = Goal title (or custom)
   - `icon` = Goal icon
   - `quantity_enabled` = true if goal has target
   - `quantity_default` = 1 (or goal target for simple goals)
   - `quantity_unit_id` = goal target unit
   - `pinned` = from request
   - `default_completed` = true (instant log behavior)

2. Link activity to goal via `activity_goals` with:
   - `auto_link_tasks` = true
   - `quantity_multiplier` = 1.0

---

## Frontend Components

### ActivityBar (Dashboard Quick Access)

```svelte
<!-- Shows pinned activities for one-tap logging -->
<ActivityBar>
  <!-- Each activity shows icon + title -->
  <!-- Tap → InstantLogModal (quantity only) -->
  <!-- Long press → Action menu (instant/schedule/timer) -->
</ActivityBar>
```

### InstantLogModal (Minimal UI)

```svelte
<!-- Super minimal - quantity picker only -->
<InstantLogModal activity={activity}>
  <QuantityPicker 
    value={quantity}
    step={activity.quantity_step}
    unit={unitSymbol}
  />
  <Button>Log ✓</Button>
</InstantLogModal>
```

### TimerWidget (Floating)

```svelte
<!-- Shows during active timer session -->
<TimerWidget session={activeSession}>
  <Timer elapsed={elapsed} />
  <ActivityInfo icon={activity.icon} title={activity.title} />
  <Button onclick={takeBreak}>Break</Button>
  <Button onclick={stop}>Stop</Button>
</TimerWidget>
```

### Goal Form - Activity Section

```svelte
<!-- In GoalModal, add activity creation option -->
<ActivitySection>
  <Toggle bind:checked={createActivity}>
    Create quick-log activity for this goal
  </Toggle>
  
  {#if createActivity}
    <Toggle bind:checked={pinToBar}>Pin to quick bar</Toggle>
    <DurationPicker bind:value={defaultDuration} />
  {/if}
</ActivitySection>
```

---

## Migration from Templates

### Data Migration

```sql
-- Migrate templates to activities
INSERT INTO activities SELECT
    id,
    created_by,
    title,
    icon,
    description,
    default_duration,
    default_emotion_id,
    3 AS default_priority,
    is_quick_log AS default_completed,
    quantity_enabled,
    quantity_default,
    quantity_step,
    NULL AS quantity_unit_id,
    "positive" AS default_impact,
    is_quick_log AS pinned,
    quick_log_order AS sort_order,
    use_count,
    last_used_at,
    NULL AS active_session,
    created_at,
    updated_at,
    deleted_at
FROM templates;

-- Migrate template_goals to activity_goals
INSERT INTO activity_goals SELECT
    in,
    out,
    auto_link_tasks,
    quantity_multiplier,
    "positive" AS default_impact,
    created_at
FROM template_goals;
```

### API Compatibility

- Keep `/templates` endpoints as aliases to `/activities` (deprecated)
- Frontend gradually migrates to new API

---

## Implementation Order

### Phase 1: Backend Foundation (Day 1)
1. Create `activities` feature folder with models
2. Implement CRUD endpoints
3. Implement `/instant` endpoint (most important)
4. Migrate from templates table

### Phase 2: Frontend Instant Log (Day 1-2)
1. Create `activities` API module
2. Create `InstantLogModal` (quantity-only)
3. Create `ActivityBar` component
4. Integrate into dashboard

### Phase 3: Goal Integration (Day 2)
1. Add `create_activity` to goal creation
2. Update `GoalModal` with activity section
3. Auto-create activity when goal created

### Phase 4: Schedule Mode (Day 2-3)
1. Implement `/schedule` endpoint
2. Update `TaskForm` to accept pre-filled data via props (not sessionStorage)
3. Add "Schedule" action to activity cards

### Phase 5: Timer/Flowmodoro (Day 3-4)
1. Implement `/timer/start`, `/timer/stop`, `/timer/break`
2. Create `TimerWidget` floating component
3. Add timer state management
4. Add Flowmodoro breakdown to task description

---

## Success Metrics

- **Instant log**: < 2 taps to log a quantity-based activity
- **Timer accuracy**: Track work vs break time accurately
- **Goal progress**: Automatic goal linking reduces manual effort by 90%
- **Adoption**: 80% of recurring tasks created via activities
