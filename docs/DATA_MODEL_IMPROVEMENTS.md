# Lucid Logs Data Model - Proposed Improvements

This document outlines proposed changes to simplify the data model, make it more robust, and leverage SurrealDB's graph capabilities for complex relationship management.

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Proposed Architecture](#proposed-architecture)
3. [Goals Model](#goals-model)
4. [Goal Logs (History)](#goal-logs-history)
5. [Templates Model](#templates-model)
6. [Tasks Model](#tasks-model)
7. [Relationship Graph Design](#relationship-graph-design)
8. [Unit System](#unit-system)
9. [Real-Life Scenarios](#real-life-scenarios)
10. [API Changes](#api-changes)
11. [Migration Path](#migration-path)

---

## Executive Summary

### Key Principles

1. **Graph-Defined Goal Types** - No `goal_type` enum. A goal's nature is defined by its graph structure:
   - Has `->goal_children->` edges → **Grouped goal**
   - Has `target` field → **Measurable goal**
   - Has `recurrence` field → **Habit**
   - Has `target.operator = "lte"` or `"eq"` with low value → **Avoidance**

2. **SurrealDB Graph Relations** - Use proper graph edges for all relationships

3. **Unified Category Handling** - Single `in_category` edge for all entities

4. **Goal History Tracking** - `goal_logs` relation records all goal state changes

5. **Simplified Status** - Only 3 statuses: `active`, `completed`, `archived`

6. **Stats Object** - Move computed statistics to separate `GoalStats` object

### Impact Summary

| Area | Current | Proposed | Benefit |
|------|---------|----------|---------|
| Goal Types | 4 enum values | Graph-inferred | Flexible, no artificial constraints |
| Templates | 25+ fields | 15 fields | Faster setup |
| Linking | activity_key matching | Explicit graph edges | Consistent, queryable |
| Categories | ID fields per entity | Single `in_category` edge | No sync issues |
| Goal History | None | `goal_logs` relation | Full audit trail |

---

## Proposed Architecture

### Entity Relationship Diagram (SurrealDB Graph)

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              SURREALDB GRAPH MODEL                              │
└─────────────────────────────────────────────────────────────────────────────────┘

                              ┌──────────────┐
                              │  categories  │
                              │  ─────────── │
                              │  id, name,   │
                              │  color, icon │
                              └──────────────┘
                                     │
                    ┌────────────────┼────────────────┐
                    │                │                │
                    ▼                ▼                ▼
          ┌───────────────┐  ┌───────────────┐  ┌───────────────┐
          │     goals     │  │   templates   │  │     tasks     │
          │   ─────────   │  │   ─────────   │  │   ─────────   │
          │ id, title     │  │ id, title     │  │ id, title     │
          │ target        │  │ defaults      │  │ start/end     │
          │ recurrence    │  │ quantity_cfg  │  │ journal       │
          │ status        │  │ is_quick_log  │  │ quantity      │
          │ stats         │  │               │  │ completed     │
          └───────────────┘  └───────────────┘  └───────────────┘
                 │                   │                  │
    ┌────────────┼────────────┐      │     ┌───────────┼───────────┐
    │            │            │      │     │           │           │
    ▼            ▼            ▼      ▼     ▼           ▼           ▼
┌─────────┐ ┌─────────┐ ┌──────────────────────┐ ┌───────────┐ ┌─────────────┐
│goal_    │ │goal_    │ │     in_category      │ │ task_goals│ │task_emotions│
│children │ │logs     │ │     (RELATE)         │ │ (RELATE)  │ │ (RELATE)    │
│(RELATE) │ │(RELATE) │ │ ──────────────────── │ │───────────│ │─────────────│
│─────────│ │─────────│ │ task/goal/template   │ │in -> tasks│ │in -> tasks  │
│in->goals│ │in->goals│ │ -> category          │ │out-> goals│ │out-> emotion│
│out->goal│ │out->snap│ │                      │ │impact,qty │ │type         │
│order    │ │event    │ │                      │ │           │ │             │
└─────────┘ └─────────┘ └──────────────────────┘ └───────────┘ └─────────────┘
                                                       │
                                               ┌───────┴───────┐
                                               ▼               ▼
                                       ┌─────────────┐ ┌─────────────┐
                                       │created_from │ │    units    │
                                       │ (RELATE)    │ │ ─────────── │
                                       │─────────────│ │ id, name    │
                                       │ task ->     │ │ symbol,type │
                                       │ template    │ │             │
                                       └─────────────┘ └─────────────┘
```

---

## Goals Model

### Graph-Inferred Goal Nature

Instead of a `goal_type` enum, the goal's nature is determined by its structure:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                         GOAL NATURE (GRAPH-INFERRED)                        │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│   HAS CHILDREN?  ──yes──►  GROUPED GOAL (project with sub-goals)           │
│        │                                                                    │
│        no                                                                   │
│        │                                                                    │
│        ▼                                                                    │
│   HAS RECURRENCE?  ──yes──►  HABIT (daily/weekly/monthly goal)             │
│        │                                                                    │
│        no                                                                   │
│        │                                                                    │
│        ▼                                                                    │
│   HAS TARGET?  ──yes──►  MEASURABLE GOAL                                   │
│        │                    │                                               │
│        │                    ├── operator = "gte" → Achievement goal         │
│        │                    ├── operator = "lte" → Limit/Avoidance goal    │
│        │                    └── operator = "eq"  → Exact target goal       │
│        │                                                                    │
│        no                                                                   │
│        │                                                                    │
│        ▼                                                                    │
│   SIMPLE GOAL (just complete it, target.value = 1 implied)                 │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Goal Model

```go
// Goal represents a goal, habit, or project in the system.
// Its nature is inferred from its structure, not an enum.
type Goal struct {
    ID        string `json:"id,omitempty"` // goals:xxx
    CreatedBy string `json:"-"`
    
    // Core fields
    Title       string `json:"title"`        // Required
    Description string `json:"description,omitempty"`
    Icon        string `json:"icon,omitempty"` // Emoji
    
    // Target (optional - if absent, implies simple completion goal)
    Target *Target `json:"target,omitempty"`
    
    // Recurrence (optional - if present, this is a habit)
    Recurrence *Recurrence `json:"recurrence,omitempty"`
    
    // Status: only 3 states
    Status string `json:"status"` // "active", "completed", "archived"
    
    // Computed statistics (populated on read)
    Stats *GoalStats `json:"stats,omitempty"`
    
    // Organization
    Priority int `json:"priority"` // 1-3
    
    // Timeline
    StartDate *time.Time `json:"start_date,omitempty"`
    Deadline  *time.Time `json:"deadline,omitempty"`
    
    // Metadata
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    CompletedAt *time.Time `json:"completed_at,omitempty"`
    
    // Populated via graph queries (not stored on goal record)
    Category    *Category      `json:"category,omitempty"`     // From in_category edge
    LinkedTasks []TaskGoalLink `json:"linked_tasks,omitempty"` // From task_goals
    Children    []Goal         `json:"children,omitempty"`     // From goal_children
    Parent      *Goal          `json:"parent,omitempty"`       // Reverse of goal_children
}

// Target defines what success looks like for a measurable goal
type Target struct {
    Value    float64 `json:"value"`    // Target amount (e.g., 100, 3, 0)
    Operator string  `json:"operator"` // "gte" (≥), "lte" (≤), "eq" (=)
    UnitID   string  `json:"unit_id"`  // Reference to units table
    PerPeriod bool   `json:"per_period"` // true = per recurrence period
}

// GoalStats contains computed statistics (populated on read, not stored)
type GoalStats struct {
    // Progress
    CurrentValue  float64    `json:"current_value"`   // Sum from task_goals
    ProgressPercent float64  `json:"progress_percent"` // 0-100 (or >100 if exceeded)
    
    // Streak tracking (for habits)
    CurrentStreak     int        `json:"current_streak"`
    LongestStreak     int        `json:"longest_streak"`
    LastCompletedDate *time.Time `json:"last_completed_date,omitempty"`
    TodayStatus       string     `json:"today_status,omitempty"` // "pending", "met", "exceeded"
    
    // For grouped goals
    ChildrenTotal     int `json:"children_total,omitempty"`
    ChildrenCompleted int `json:"children_completed,omitempty"`
    
    // Overall
    TotalContributions int `json:"total_contributions"` // Count of task_goals links
}

// Recurrence defines habit frequency
type Recurrence struct {
    Frequency  int      `json:"frequency"`             // Times per period
    Period     string   `json:"period"`                // "day", "week", "month"
    ActiveDays []string `json:"active_days,omitempty"` // ["mon", "tue", ...]
    BeforeTime string   `json:"before_time,omitempty"` // "22:00"
    AfterTime  string   `json:"after_time,omitempty"`  // "06:00"
    GraceDays  int      `json:"grace_days,omitempty"`  // 0-7
}
```

### Target Operator Examples

| Operator | Example Goal | Target | Meaning |
|----------|--------------|--------|---------|
| `gte` (≥) | "Run 100km this month" | `{value: 100, operator: "gte", unit_id: "units:km"}` | Achieve at least 100km |
| `lte` (≤) | "Max 2 coffees per day" | `{value: 2, operator: "lte", unit_id: "units:count", per_period: true}` | Don't exceed 2 |
| `eq` (=) | "Zero cigarettes" | `{value: 0, operator: "eq", unit_id: "units:count", per_period: true}` | Exactly zero (strict avoidance) |
| `gte` | "Drink 3L water daily" | `{value: 3, operator: "gte", unit_id: "units:l", per_period: true}` | At least 3L per day |

### Goal Status

Only 3 statuses:

| Status | Description | Can transition to |
|--------|-------------|------------------|
| `active` | Currently working on | `completed`, `archived` |
| `completed` | Goal achieved | `archived` |
| `archived` | Hidden from active views | `active` |

---

## Goal Logs (History)

The `goal_logs` relation tracks all significant events in a goal's lifecycle, providing full audit history.

### goal_logs Relation

```surql
DEFINE TABLE goal_logs SCHEMAFULL TYPE RELATION IN goal OUT goal_snapshot;
DEFINE FIELD in ON goal_logs TYPE record<goal>;
DEFINE FIELD out ON goal_logs TYPE record<goal_snapshot>;
DEFINE FIELD event ON goal_logs TYPE string ASSERT $value IN [
    "created", "updated", "completed", "archived", "reactivated",
    "streak_updated", "target_met", "target_exceeded", "period_reset"
];
DEFINE FIELD changes ON goal_logs TYPE object;  -- What changed
DEFINE FIELD triggered_by ON goal_logs TYPE record<task> | NONE;  -- Which task caused this
DEFINE FIELD created_at ON goal_logs TYPE datetime DEFAULT time::now();
DEFINE FIELD created_by ON goal_logs TYPE string;
```

### goal_snapshot Table

Stores point-in-time snapshots of goal state:

```surql
DEFINE TABLE goal_snapshot SCHEMAFULL;
DEFINE FIELD goal_id ON goal_snapshot TYPE record<goal>;
DEFINE FIELD target ON goal_snapshot TYPE object;
DEFINE FIELD stats ON goal_snapshot TYPE object;  -- Snapshot of GoalStats
DEFINE FIELD status ON goal_snapshot TYPE string;
DEFINE FIELD created_at ON goal_snapshot TYPE datetime DEFAULT time::now();
```

### Log Events

| Event | When Triggered | Data Captured |
|-------|----------------|---------------|
| `created` | Goal created | Initial state |
| `updated` | Goal properties modified | Changed fields |
| `completed` | Status → completed | Final stats |
| `archived` | Status → archived | Reason (optional) |
| `reactivated` | archived → active | — |
| `streak_updated` | Daily habit check | New streak value |
| `target_met` | current_value reaches target | Task that completed it |
| `target_exceeded` | Avoidance goal exceeded | Task that broke it |
| `period_reset` | Recurring goal period ends | Period stats snapshot |

### Example: Goal History Query

```surql
-- Get full history for a goal
SELECT 
    event,
    changes,
    triggered_by.title AS task_title,
    out.stats AS snapshot_stats,
    created_at
FROM goal_logs 
WHERE in = goals:g_hydration
ORDER BY created_at DESC
LIMIT 20;
```

**Result**:
```json
[
  { "event": "target_met", "task_title": "Log Water", "snapshot_stats": { "current_value": 3.25 }, "created_at": "2026-01-04T18:30:00Z" },
  { "event": "streak_updated", "changes": { "streak": 6 }, "created_at": "2026-01-04T18:30:00Z" },
  { "event": "period_reset", "snapshot_stats": { "current_value": 0 }, "created_at": "2026-01-04T00:00:00Z" },
  { "event": "target_met", "task_title": "Log Water", "snapshot_stats": { "current_value": 3.5 }, "created_at": "2026-01-03T20:15:00Z" }
]
```

---

## Templates Model

### Template Model

```go
type TaskTemplate struct {
    ID        string `json:"id,omitempty"`
    CreatedBy string `json:"-"`
    
    // Core
    Title       string `json:"title"`
    Description string `json:"description,omitempty"`
    Icon        string `json:"icon,omitempty"`
    
    // Defaults for tasks
    DefaultDuration int `json:"default_duration,omitempty"` // seconds
    
    // Quick log settings
    IsQuickLog    bool `json:"is_quick_log"`
    QuickLogOrder int  `json:"quick_log_order,omitempty"`
    
    // Quantity settings
    QuantityEnabled bool    `json:"quantity_enabled"`
    QuantityDefault float64 `json:"quantity_default,omitempty"`
    QuantityStep    float64 `json:"quantity_step,omitempty"`
    
    // Emotion defaults
    ExpectedQuadrant string `json:"expected_quadrant,omitempty"`
    DefaultEmotionID string `json:"default_emotion_id,omitempty"`
    
    // Usage stats
    UseCount   int        `json:"use_count"`
    LastUsedAt *time.Time `json:"last_used_at,omitempty"`
    
    // Metadata
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
    
    // Populated via graph
    Goals    []Goal    `json:"goals,omitempty"`    // From template_goals edge
    Category *Category `json:"category,omitempty"` // From in_category OR inherited
}
```

### template_goals Relation

```surql
DEFINE TABLE template_goals SCHEMAFULL TYPE RELATION IN template OUT goal;
DEFINE FIELD auto_link_tasks ON template_goals TYPE bool DEFAULT true;
DEFINE FIELD quantity_multiplier ON template_goals TYPE float DEFAULT 1.0;
DEFINE FIELD created_at ON template_goals TYPE datetime DEFAULT time::now();
```

---

## Tasks Model

### Task Model

```go
type Task struct {
    ID        string    `json:"id,omitempty"`
    Title     string    `json:"title"`
    Journal   string    `json:"journal,omitempty"`
    StartDate time.Time `json:"start_date"`
    EndDate   time.Time `json:"end_date"`
    Completed bool      `json:"completed"`
    Note      string    `json:"note,omitempty"`
    Source    string    `json:"source"` // "manual", "template", "quick"
    
    // Reflections
    Positives []TaskItem `json:"positives"`
    Negatives []TaskItem `json:"negatives"`
    
    // Emotion
    EmotionID       *string          `json:"emotion_id,omitempty"`
    InferredEmotion *InferredEmotion `json:"inferred_emotion,omitempty"`
    
    // Quantity (for measurable goals)
    Quantity *Quantity `json:"quantity,omitempty"`
    
    // Metadata
    CreatedAt time.Time  `json:"created_at"`
    UpdatedAt time.Time  `json:"updated_at"`
    CreatedBy string     `json:"-"`
    
    // Populated via graph
    Category    *Category      `json:"category,omitempty"`
    Template    *Template      `json:"template,omitempty"`
    LinkedGoals []TaskGoalLink `json:"linked_goals,omitempty"`
}

type Quantity struct {
    Value  float64 `json:"value"`
    UnitID string  `json:"unit_id"`
}
```

---

## Relationship Graph Design

### All Graph Relations

| Relation | In → Out | Purpose | Key Fields |
|----------|----------|---------|------------|
| `in_category` | task/goal/template → category | Category assignment | `inherited: bool` |
| `task_goals` | task → goal | Task contributes to goal | `impact_type`, `quantity_value`, `unit_id` |
| `task_emotions` | task → emotion | Emotion analytics | `type` (primary/positive/negative) |
| `template_goals` | template → goal | Auto-link configuration | `auto_link_tasks`, `quantity_multiplier` |
| `created_from` | task → template | Source tracking | `used_defaults` |
| `goal_children` | goal → goal | Parent-child grouping | `order`, `required` |
| `goal_logs` | goal → goal_snapshot | History tracking | `event`, `changes` |

### in_category Relation

Replaces all `category_id` fields with a single graph edge:

```surql
DEFINE TABLE in_category SCHEMAFULL TYPE RELATION 
    IN record<task|goal|template> 
    OUT record<category>;
DEFINE FIELD inherited ON in_category TYPE bool DEFAULT false;
DEFINE FIELD created_at ON in_category TYPE datetime DEFAULT time::now();

-- Examples
RELATE tasks:abc -> in_category -> categories:work;
RELATE goals:fitness -> in_category -> categories:health SET { inherited: false };

-- Query: Get category for any entity
SELECT ->in_category->categories AS category FROM tasks:xxx;
SELECT ->in_category->categories AS category FROM goals:xxx;

-- Query: All entities in a category
SELECT {
    tasks: <-in_category<-tasks,
    goals: <-in_category<-goals,
    templates: <-in_category<-templates
} FROM categories:work;
```

### goal_children Relation

Replaces the old `parent_goal` field and `epic` goal type:

```surql
DEFINE TABLE goal_children SCHEMAFULL TYPE RELATION IN goal OUT goal;
DEFINE FIELD order ON goal_children TYPE int DEFAULT 0;
DEFINE FIELD required ON goal_children TYPE bool DEFAULT true;
DEFINE FIELD weight ON goal_children TYPE float DEFAULT 1.0;
DEFINE FIELD created_at ON goal_children TYPE datetime DEFAULT time::now();

-- Parent links to children
RELATE goals:launch_saas -> goal_children -> goals:design_ui SET { order: 1, required: true };
RELATE goals:launch_saas -> goal_children -> goals:build_mvp SET { order: 2, required: true };

-- Query children with progress
SELECT 
    *,
    ->goal_children->goals AS children,
    count(->goal_children->goals[WHERE status = "completed"]) AS completed_count
FROM goals:launch_saas;

-- Query parent (reverse)
SELECT <-goal_children<-goals AS parents FROM goals:design_ui;
```

### task_goals Relation (Enhanced)

```surql
DEFINE TABLE task_goals SCHEMAFULL TYPE RELATION IN task OUT goal;
DEFINE FIELD impact_type ON task_goals TYPE string 
    ASSERT $value IN ["positive", "negative", "neutral"];
DEFINE FIELD impact_magnitude ON task_goals TYPE int DEFAULT 3 
    ASSERT $value >= 1 AND $value <= 5;
DEFINE FIELD quantity_value ON task_goals TYPE float;
DEFINE FIELD unit_id ON task_goals TYPE record<unit>;
DEFINE FIELD source ON task_goals TYPE string DEFAULT "manual" 
    ASSERT $value IN ["manual", "auto"];
DEFINE FIELD notes ON task_goals TYPE string;
DEFINE FIELD is_milestone ON task_goals TYPE bool DEFAULT false;
DEFINE FIELD milestone_label ON task_goals TYPE string;
DEFINE FIELD milestone_order ON task_goals TYPE int;
DEFINE FIELD created_at ON task_goals TYPE datetime DEFAULT time::now();
```

---

## Unit System

### units Table

```surql
DEFINE TABLE units SCHEMAFULL;
DEFINE FIELD name ON units TYPE string;
DEFINE FIELD symbol ON units TYPE string;
DEFINE FIELD type ON units TYPE string; -- "distance", "time", "volume", "count", "custom"
DEFINE FIELD is_system ON units TYPE bool DEFAULT false;
DEFINE FIELD created_by ON units TYPE string;
DEFINE FIELD created_at ON units TYPE datetime DEFAULT time::now();

-- System units (seeded)
CREATE units:km SET name = "kilometers", symbol = "km", type = "distance", is_system = true;
CREATE units:mi SET name = "miles", symbol = "mi", type = "distance", is_system = true;
CREATE units:min SET name = "minutes", symbol = "min", type = "time", is_system = true;
CREATE units:hr SET name = "hours", symbol = "hr", type = "time", is_system = true;
CREATE units:count SET name = "count", symbol = "×", type = "count", is_system = true;
CREATE units:l SET name = "liters", symbol = "L", type = "volume", is_system = true;
CREATE units:ml SET name = "milliliters", symbol = "ml", type = "volume", is_system = true;
CREATE units:pages SET name = "pages", symbol = "pg", type = "count", is_system = true;
```

---

## Real-Life Scenarios

This section demonstrates complete user journeys over extended time periods, showing how the system evolves with real usage.

---

### Scenario 1: 30-Day Hydration Habit Journey

**User Goal**: Build a habit of drinking 3L water daily

#### Day 1: Goal Creation

**User Input**:
```
Title: "Stay Hydrated 💧"
Target: At least 3 liters per day
Category: Health
```

**API Request**:
```json
POST /goals
{
  "title": "Stay Hydrated 💧",
  "icon": "💧",
  "target": {
    "value": 3,
    "operator": "gte",
    "unit_id": "units:l",
    "per_period": true
  },
  "recurrence": { "frequency": 1, "period": "day" }
}
```

**System Creates**:
```surql
-- Goal
CREATE goals:hydration SET
  title = "Stay Hydrated 💧",
  icon = "💧",
  target = { value: 3, operator: "gte", unit_id: units:l, per_period: true },
  recurrence = { frequency: 1, period: "day" },
  status = "active",
  created_at = time::now();

-- Category link
RELATE goals:hydration -> in_category -> categories:health;

-- Auto-create quick-log template
CREATE templates:log_water SET
  title = "Log Water",
  icon = "💧",
  quantity_enabled = true,
  quantity_default = 0.5,
  quantity_step = 0.25,
  is_quick_log = true;

RELATE templates:log_water -> template_goals -> goals:hydration SET { auto_link_tasks: true };
RELATE templates:log_water -> in_category -> categories:health SET { inherited: true };

-- Initial log entry
RELATE goals:hydration -> goal_logs -> goal_snapshot:snap_001 SET {
  event: "created",
  created_at: time::now()
};
```

#### Day 1: First Water Logs

**User logs water 4 times**:

| Time | Quantity | Running Total |
|------|----------|---------------|
| 8:00 AM | 0.5L | 0.5L |
| 11:00 AM | 0.75L | 1.25L |
| 2:00 PM | 0.5L | 1.75L |
| 6:00 PM | 0.75L | 2.5L |

**System after 4th log** (goal NOT met yet):
```json
{
  "id": "goals:hydration",
  "stats": {
    "current_value": 2.5,
    "progress_percent": 83.3,
    "current_streak": 0,
    "today_status": "pending"
  }
}
```

#### Day 1: Goal Met

**User logs 0.75L at 8:00 PM** (5th entry):

**System Updates**:
```surql
-- Create task
CREATE tasks:water_005 SET title = "Log Water", quantity = { value: 0.75, unit_id: units:l };
RELATE tasks:water_005 -> task_goals -> goals:hydration SET { quantity_value: 0.75, source: "auto" };

-- Update stats (now 3.25L)
-- Goal met! Log the event
RELATE goals:hydration -> goal_logs -> goal_snapshot:snap_002 SET {
  event: "target_met",
  triggered_by: tasks:water_005,
  changes: { current_value: 3.25, streak: 1 }
};

-- Update streak
-- (computed on next read)
```

**Goal State**:
```json
{
  "stats": {
    "current_value": 3.25,
    "progress_percent": 108.3,
    "current_streak": 1,
    "longest_streak": 1,
    "today_status": "met"
  }
}
```

#### Day 5: Maintaining Streak

**After 5 days of consistent logging**:

```json
{
  "id": "goals:hydration",
  "stats": {
    "current_value": 3.5,  // Today's progress
    "current_streak": 5,
    "longest_streak": 5,
    "total_contributions": 23,  // Total water logs
    "today_status": "met"
  }
}
```

**Goal Log History**:
```json
[
  { "event": "target_met", "created_at": "2026-01-05", "changes": { "streak": 5 } },
  { "event": "period_reset", "created_at": "2026-01-05T00:00:00Z" },
  { "event": "target_met", "created_at": "2026-01-04", "changes": { "streak": 4 } },
  { "event": "period_reset", "created_at": "2026-01-04T00:00:00Z" },
  // ... more entries
]
```

#### Day 8: Missed Day (Streak Broken)

**User forgets to log on Day 8**:

**Midnight System Check**:
```surql
-- Daily cron job checks all active habits
LET $yesterday = time::floor(time::now() - 1d, 1d);

FOR $goal IN (SELECT * FROM goals WHERE recurrence != NONE AND status = "active") {
  LET $yesterday_value = (
    SELECT math::sum(quantity_value) FROM task_goals 
    WHERE out = $goal.id 
    AND in.start_date >= $yesterday
    AND in.start_date < $yesterday + 1d
  )[0] ?? 0;
  
  IF $yesterday_value < $goal.target.value {
    -- Streak broken!
    RELATE $goal.id -> goal_logs -> goal_snapshot:new SET {
      event: "streak_updated",
      changes: { streak: 0, previous_streak: $goal.stats.current_streak }
    };
  }
};
```

**Goal State After Missed Day**:
```json
{
  "stats": {
    "current_value": 0,  // New day, reset
    "current_streak": 0,  // Broken!
    "longest_streak": 7,  // Preserved
    "today_status": "pending"
  }
}
```

#### Day 30: Monthly Review

**User views goal after 30 days**:

**API Request**:
```json
GET /goals/hydration?include=logs&log_limit=50
```

**Response**:
```json
{
  "id": "goals:hydration",
  "title": "Stay Hydrated 💧",
  "status": "active",
  "stats": {
    "current_value": 2.75,
    "current_streak": 12,
    "longest_streak": 12,
    "total_contributions": 142
  },
  "logs_summary": {
    "days_met": 26,
    "days_missed": 4,
    "average_daily": 3.2,
    "best_day": { "date": "2026-01-22", "value": 4.5 }
  }
}
```

---

### Scenario 2: Avoidance Goal - Zero Cigarettes

**User Goal**: Quit smoking completely (strict zero tolerance)

#### Goal Creation

**User Input**:
```
Title: "Stay Smoke-Free 🚭"
Target: Exactly 0 cigarettes per day
Category: Health
```

**API Request**:
```json
POST /goals
{
  "title": "Stay Smoke-Free 🚭",
  "icon": "🚭",
  "target": {
    "value": 0,
    "operator": "eq",
    "unit_id": "units:count",
    "per_period": true
  },
  "recurrence": { "frequency": 1, "period": "day" }
}
```

**System Creates**:
```surql
CREATE goals:no_smoking SET
  title = "Stay Smoke-Free 🚭",
  target = { value: 0, operator: "eq", unit_id: units:count, per_period: true },
  recurrence = { frequency: 1, period: "day" },
  status = "active";
```

#### Day 1-5: Success Streak

**No tasks logged against this goal** = Goal met each day!

**Daily Check Logic**:
```surql
-- For "eq" with value 0, goal is met if current_value = 0
LET $today_value = (SELECT math::sum(quantity_value) FROM task_goals WHERE out = goals:no_smoking)[0] ?? 0;
LET $met = IF $goal.target.operator = "eq" THEN $today_value = $goal.target.value ELSE ...;
-- $met = true (0 = 0)
```

**Goal State After 5 Days**:
```json
{
  "stats": {
    "current_value": 0,
    "current_streak": 5,
    "today_status": "met"
  }
}
```

#### Day 6: Slip-Up

**User has a cigarette and logs it honestly**:

**User Input**:
```
Title: "Had a cigarette at party"
Quantity: 1
Negatives: ["Peer pressure", "Stressful day"]
```

**API Request**:
```json
POST /tasks
{
  "title": "Had a cigarette at party",
  "quantity": { "value": 1, "unit_id": "units:count" },
  "negatives": [
    { "text": "Peer pressure" },
    { "text": "Stressful day", "emotion_id": "emotions:E61" }
  ]
}

POST /tasks/{id}/goals
{
  "goal_id": "goals:no_smoking",
  "impact_type": "negative"
}
```

**System Updates**:
```surql
RELATE tasks:slip -> task_goals -> goals:no_smoking SET {
  impact_type: "negative",
  quantity_value: 1,
  source: "manual"
};

-- Goal exceeded (for avoidance, current > target is "exceeded")
RELATE goals:no_smoking -> goal_logs -> goal_snapshot:slip SET {
  event: "target_exceeded",
  triggered_by: tasks:slip,
  changes: { 
    current_value: 1, 
    streak_broken: true,
    previous_streak: 5
  }
};
```

**Goal State**:
```json
{
  "stats": {
    "current_value": 1,
    "current_streak": 0,  // Reset
    "longest_streak": 5,
    "today_status": "exceeded"
  }
}
```

#### Day 7+: Recovery

**User recommits, no more slips**:

```json
// After 30 more days clean
{
  "stats": {
    "current_value": 0,
    "current_streak": 30,
    "longest_streak": 30,
    "total_contributions": 1  // Only that one slip
  }
}
```

---

### Scenario 3: Limit Goal - Coffee Intake

**User Goal**: Limit coffee to max 3 cups per day

#### Goal Creation

```json
POST /goals
{
  "title": "Moderate Coffee ☕",
  "icon": "☕",
  "target": {
    "value": 3,
    "operator": "lte",
    "unit_id": "units:count",
    "per_period": true
  },
  "recurrence": { "frequency": 1, "period": "day" }
}
```

#### Day 1: Within Limit

**User logs coffees**:

| Time | Cup # | Status |
|------|-------|--------|
| 7:00 AM | 1 | ✅ Good (1 ≤ 3) |
| 10:00 AM | 2 | ✅ Good (2 ≤ 3) |
| 2:00 PM | 3 | ✅ At limit (3 ≤ 3) |

**Goal State**:
```json
{
  "stats": {
    "current_value": 3,
    "today_status": "met"  // 3 ≤ 3
  }
}
```

#### Day 2: Exceeded Limit

**User has 4 coffees**:

| Time | Cup # | Status |
|------|-------|--------|
| 7:00 AM | 1 | ✅ |
| 10:00 AM | 2 | ✅ |
| 2:00 PM | 3 | ✅ At limit |
| 5:00 PM | 4 | ⚠️ Exceeded! |

**System on 4th coffee**:
```surql
-- current_value becomes 4
-- 4 > 3, so for "lte" this is exceeded
RELATE goals:coffee -> goal_logs -> goal_snapshot:exceeded SET {
  event: "target_exceeded",
  changes: { current_value: 4 }
};
```

**Goal State**:
```json
{
  "stats": {
    "current_value": 4,
    "today_status": "exceeded",  // 4 > 3
    "current_streak": 0  // Broken
  }
}
```

---

### Scenario 4: Grouped Goal - Launch Side Project

**User Goal**: Launch a side project with multiple sub-goals

#### Create Parent Goal

```json
POST /goals
{
  "title": "Launch Blog 🚀",
  "icon": "🚀",
  "description": "Personal tech blog with 10 articles",
  "deadline": "2026-03-31T23:59:59Z"
}
```

*Note: No `target` or `recurrence` - this goal will become "grouped" when children are added.*

#### Add Child Goals

```json
// Child 1
POST /goals
{
  "title": "Design Blog",
  "icon": "🎨",
  "target": { "value": 1, "operator": "gte", "unit_id": "units:count" }
}
POST /goals/{parent_id}/children
{ "child_goal_id": "goals:design", "order": 1, "required": true }

// Child 2
POST /goals
{
  "title": "Write 10 Articles",
  "icon": "✍️",
  "target": { "value": 10, "operator": "gte", "unit_id": "units:count" }
}
POST /goals/{parent_id}/children
{ "child_goal_id": "goals:articles", "order": 2, "required": true }

// Child 3
POST /goals
{
  "title": "Deploy Site",
  "icon": "🌐",
  "target": { "value": 1, "operator": "gte", "unit_id": "units:count" }
}
POST /goals/{parent_id}/children
{ "child_goal_id": "goals:deploy", "order": 3, "required": true }
```

**Graph Structure**:
```
goals:launch_blog
    ├─[goal_children]─► goals:design (order: 1)
    ├─[goal_children]─► goals:articles (order: 2)
    └─[goal_children]─► goals:deploy (order: 3)
```

**Query Parent with Children**:
```surql
SELECT 
    *,
    ->goal_children->goals.{
        id, title, status, 
        target.value AS target,
        stats.current_value AS current
    } AS children,
    count(->goal_children->goals[WHERE status = "completed"]) AS completed,
    count(->goal_children->goals) AS total
FROM goals:launch_blog;
```

#### Week 1: Design Complete

**User completes design tasks and marks "Design Blog" as complete**:

```json
PUT /goals/design
{ "status": "completed" }
```

**System Updates**:
```surql
UPDATE goals:design SET status = "completed", completed_at = time::now();

RELATE goals:design -> goal_logs -> goal_snapshot:complete SET { event: "completed" };

-- Parent goal progress recalculated automatically
```

**Parent State**:
```json
{
  "id": "goals:launch_blog",
  "children": [
    { "id": "goals:design", "status": "completed", "current": 1, "target": 1 },
    { "id": "goals:articles", "status": "active", "current": 0, "target": 10 },
    { "id": "goals:deploy", "status": "active", "current": 0, "target": 1 }
  ],
  "stats": {
    "children_completed": 1,
    "children_total": 3,
    "progress_percent": 33.3
  }
}
```

#### Week 2-6: Writing Articles

**User logs article completions over 6 weeks**:

| Week | Articles Written | Total | Progress |
|------|------------------|-------|----------|
| 2 | 2 | 2 | 20% |
| 3 | 1 | 3 | 30% |
| 4 | 2 | 5 | 50% |
| 5 | 3 | 8 | 80% |
| 6 | 2 | 10 | 100% ✅ |

**Each article logged as task**:
```json
POST /tasks
{
  "title": "Article: Getting Started with Go",
  "completed": true
}
POST /tasks/{id}/goals
{
  "goal_id": "goals:articles",
  "impact_type": "positive",
  "quantity_value": 1,
  "is_milestone": true,
  "milestone_label": "Article #5"
}
```

**After 10 articles**:
```json
{
  "id": "goals:articles",
  "status": "completed",  // Auto-completed when target met
  "stats": { "current_value": 10 }
}
```

#### Week 8: Deployment and Launch

**User deploys and marks complete**:

```json
PUT /goals/deploy
{ "status": "completed" }
```

**System Checks Parent Completion**:
```surql
-- All required children complete?
LET $parent = goals:launch_blog;
LET $required = (SELECT * FROM ->goal_children->goals WHERE required = true);
LET $completed = (SELECT * FROM ->goal_children->goals WHERE required = true AND status = "completed");

IF count($required) = count($completed) {
  UPDATE $parent SET status = "completed", completed_at = time::now();
  RELATE $parent -> goal_logs -> goal_snapshot:final SET { event: "completed" };
}
```

**Final Parent State**:
```json
{
  "id": "goals:launch_blog",
  "status": "completed",
  "completed_at": "2026-02-28T15:30:00Z",
  "stats": {
    "children_completed": 3,
    "children_total": 3,
    "progress_percent": 100
  }
}
```

---

### Scenario 5: Multi-Goal Task - Gym Session

**User has multiple fitness goals**:
1. "Exercise 150 min/week"
2. "Burn 2000 calories/week"
3. "Strength training 3x/week"

#### Log a Gym Session

**User logs 1-hour gym workout**:

```json
POST /tasks
{
  "title": "Morning gym - cardio + weights",
  "start_date": "2026-01-06T06:00:00Z",
  "end_date": "2026-01-06T07:00:00Z",
  "completed": true,
  "positives": [
    { "text": "New bench press PR!", "emotion_id": "emotions:E25" }
  ],
  "emotion_id": "emotions:E16"
}
```

**Link to Multiple Goals**:
```json
POST /tasks/{id}/goals
{
  "links": [
    {
      "goal_id": "goals:exercise_150",
      "quantity_value": 60,
      "unit_id": "units:min",
      "notes": "Full hour workout"
    },
    {
      "goal_id": "goals:burn_2000",
      "quantity_value": 450,
      "unit_id": "units:cal",
      "notes": "Estimated from heart rate monitor"
    },
    {
      "goal_id": "goals:strength_3x",
      "quantity_value": 1,
      "unit_id": "units:count",
      "notes": "Included weight training"
    }
  ]
}
```

**System Creates 3 Edges**:
```surql
RELATE tasks:gym_001 -> task_goals -> goals:exercise_150 SET { quantity_value: 60, unit_id: units:min };
RELATE tasks:gym_001 -> task_goals -> goals:burn_2000 SET { quantity_value: 450, unit_id: units:cal };
RELATE tasks:gym_001 -> task_goals -> goals:strength_3x SET { quantity_value: 1, unit_id: units:count };

-- Update all three goals' stats
-- (computed on next read)
```

**After Week 1** (3 gym sessions):

| Goal | Target | Current | Status |
|------|--------|---------|--------|
| Exercise 150 min/week | 150 | 180 | ✅ Met |
| Burn 2000 cal/week | 2000 | 1350 | 🔄 67% |
| Strength 3x/week | 3 | 3 | ✅ Met |

---

### Scenario 6: Category Inheritance Chain

**Setup**:
- Category: "Learning" (blue)
- Goal: "Complete 5 courses" → in Learning
- Template: "Study Session" → linked to goal

#### User Uses Template

```json
POST /templates/study_session/use
{
  "start_date": "2026-01-06T19:00:00Z",
  "end_date": "2026-01-06T20:00:00Z",
  "quantity": { "value": 1, "unit_id": "units:hr" }
}
```

**System Inheritance Chain**:
```surql
-- 1. Create task
CREATE tasks:study_001 SET title = "Study Session", ...;

-- 2. Link to template
RELATE tasks:study_001 -> created_from -> templates:study_session;

-- 3. Get goal from template->goal edge
LET $goal = (SELECT ->template_goals->goals FROM templates:study_session)[0];

-- 4. Auto-link task to goal (if auto_link_tasks = true)
RELATE tasks:study_001 -> task_goals -> $goal.id SET { source: "auto" };

-- 5. Inherit category: Task has none, check goal
LET $goal_category = (SELECT ->in_category->categories FROM $goal.id)[0];
IF $goal_category {
    RELATE tasks:study_001 -> in_category -> $goal_category.id SET { inherited: true };
}
```

**Final Task State**:
```json
{
  "id": "tasks:study_001",
  "title": "Study Session",
  "category": { 
    "id": "categories:learning", 
    "name": "Learning", 
    "color": "#3B82F6",
    "_inherited": true  // UI can show this differently
  },
  "linked_goals": [
    { "goal_id": "goals:5_courses", "goal_title": "Complete 5 courses" }
  ],
  "template": { "id": "templates:study_session" }
}
```

---

### Scenario Summary

| Scenario | Duration | Key Features Demonstrated |
|----------|----------|---------------------------|
| 1. Hydration Habit | 30 days | Habit tracking, streaks, daily resets, period logs |
| 2. Zero Cigarettes | 30+ days | `eq` operator, strict avoidance, slip recovery |
| 3. Coffee Limit | Ongoing | `lte` operator, "at limit" vs "exceeded" |
| 4. Blog Launch | 8 weeks | Grouped goals, child completion, milestones |
| 5. Gym Session | 1 week | Multi-goal tasks, different units |
| 6. Study Session | Single task | Template→Goal→Category inheritance |

---

## API Changes

### Removed Endpoints

| Endpoint | Replacement |
|----------|-------------|
| `POST /goals/:id/actions` | Link tasks with `is_milestone: true` |
| `GET /goals/:id/actions` | Query `task_goals WHERE is_milestone = true` |

### New Endpoints

| Endpoint | Purpose |
|----------|---------|
| `GET /goals/:id/logs` | Get goal history |
| `POST /goals/:id/children` | Add child goal to grouped goal |
| `GET /goals/:id/children` | Get children with progress |
| `GET /units` | List available units |
| `POST /units` | Create custom unit |

### Modified Endpoints

| Endpoint | Changes |
|----------|---------|
| `POST /tasks` | No `category_id`, category via in_category after create |
| `PUT /entities/:id/category` | Generic endpoint to set/change category |
| `POST /templates` | No `goal_id`, use template_goals edge |

---

## Migration Path

### Phase 1: Add New Relations

1. Create `in_category`, `goal_children`, `goal_logs`, `goal_snapshot` tables
2. Add `units` table with system units
3. Dual-write to both old fields and new edges

### Phase 2: Migrate Data

```surql
-- Migrate categories
FOR $task IN (SELECT * FROM tasks WHERE category != NONE) {
    RELATE $task.id -> in_category -> $task.category SET { inherited: false };
};

-- Migrate goal types to graph structure
-- (epic goals get children, others just use target)

-- Create initial goal logs
FOR $goal IN (SELECT * FROM goals) {
    CREATE goal_snapshot SET goal_id = $goal.id, stats = {}, status = $goal.status;
    RELATE $goal.id -> goal_logs -> (SELECT id FROM goal_snapshot WHERE goal_id = $goal.id) SET {
        event: "created", created_at: $goal.created_at
    };
};
```

### Phase 3: Remove Old Fields

1. Remove `goal_type` field
2. Remove `category_id` fields
3. Update all queries to use graph traversal
4. Remove deprecated statuses (paused, abandoned → archived)

---

## Summary

| Aspect | Before | After |
|--------|--------|-------|
| Goal Nature | `goal_type` enum (4 values) | Graph-inferred |
| Goal Status | 4 statuses | 3: active, completed, archived |
| Goal Stats | Mixed into goal fields | Separate `GoalStats` object |
| Goal History | None | `goal_logs` relation |
| Category Link | `category_id` field | `in_category` edge |
| Avoidance | Separate type | `operator: "lte"` or `"eq"` |
| Grouped Goals | `epic` type + `parent_goal` | `goal_children` edges |

---

*Document Created: January 2026*
*Status: Proposal - Ready for Implementation*
