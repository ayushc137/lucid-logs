# Journal App Feature Planning Document

## Executive Summary

This document provides an exhaustive specification for extending the existing journal application with:
- **Goals** (one-time and recurring/habits)
- **Metrics and Analytics**
- **Task Templates for Quick Logging**
- **Retrospectives** (daily auto-generated and custom date ranges)
- **Rich Visualization and Charts**

All features integrate with the existing entities: `tasks`, `categories`, `emotions`, `task_emotions`, and `users`.

---

## Table of Contents

1. [Current System Analysis](#1-current-system-analysis)
2. [Data Model & Schema](#2-data-model--schema)
3. [API Design](#3-api-design)
4. [Feature Breakdown](#4-feature-breakdown)
5. [Analytics & Insights](#5-analytics--insights)
6. [Step-by-Step Implementation Plan](#6-step-by-step-implementation-plan)
7. [Appendices](#7-appendices)

---

## 1. Current System Analysis

### 1.1 Existing Entities

Based on the current codebase in `apps/go_backend/internal/features/`:

| Entity | Table | Key Fields | Notes |
|--------|-------|------------|-------|
| **User** | `users` | id, email, pass, is_admin, created_at, updated_at | SCHEMAFULL; auth via JWT |
| **Task** | `tasks` | id, title, journal, start_date, end_date, completed, priority, source, note, positives, negatives, emotion_id, inferred_emotion, category (record link), created_by | Schemaless; soft delete support |
| **Category** | `categories` | id, name, color, created_by | Unique name per user |
| **Emotion** | `emotions` | id (E01-E100), name, emoji, quadrant (yellow/green/red/blue), x, y, valence, arousal, dominance, intensity, certainty, social, description | 100 emotions based on Yale RULER |
| **Task-Emotion Edge** | `task_emotions` | in (task), out (emotion), type (primary/positive/negative), text | Relation table for analytics |

### 1.2 Existing Patterns

- **Repository Pattern**: Each feature has handler -> service -> repository layers
- **Soft Delete**: `deleted_at` field; queries filter with `deleted_at = NONE`
- **Pagination**: Standard `pagination.Params` with limit/offset
- **Record Links**: SurrealDB record IDs (e.g., `categories:abc123`)
- **Ownership**: All entities have `created_by` field; enforced in Go layer
- **Emotion Inference**: `InferredEmotion` computed from positives/negatives on write
- **Edge Tables**: `task_emotions` as TYPE RELATION for graph queries

### 1.3 Database Patterns

```surql
-- Record links (current)
task.category = categories:work123

-- Hydration
SELECT * FROM tasks FETCH category

-- Edge queries
SELECT ->task_emotions->emotions.* FROM tasks:abc
SELECT <-task_emotions<-tasks.* FROM emotions:E16
```

---

## 2. Data Model & Schema

### 2.1 New Tables Overview

| Table | Type | Purpose |
|-------|------|---------|
| `goals` | Schemaless | One-time and recurring goal definitions |
| `habits` | Schemaless | Recurring pattern definitions (frequency, quantity, time-bound, avoidance) |
| `task_goal_impacts` | RELATION | Edge: task -> goal with polarity/magnitude |
| `task_habit_logs` | RELATION | Edge: task -> habit for completion tracking |
| `goal_history` | Schemaless | Append-only changelog for goals |
| `habit_history` | Schemaless | Append-only changelog for habits |
| `templates` | Schemaless | Reusable task blueprints |
| `retros_daily` | Schemaless | Auto-generated daily retrospectives |
| `retros_range` | Schemaless | Custom date-range retrospectives |
| `daily_summaries` | Schemaless | Pre-computed daily aggregates for analytics |
| `user_preferences` | SCHEMAFULL | User settings (timezone, retro time, etc.) |

### 2.2 Goals Table

```surql
-- db/migrations/004_goals.surql
DEFINE TABLE IF NOT EXISTS goals PERMISSIONS FULL;

-- Indexes
DEFINE INDEX IF NOT EXISTS idx_goals_user ON TABLE goals COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_goals_user_status ON TABLE goals COLUMNS created_by, status;
DEFINE INDEX IF NOT EXISTS idx_goals_user_deadline ON TABLE goals COLUMNS created_by, deadline;
DEFINE INDEX IF NOT EXISTS idx_goals_user_type ON TABLE goals COLUMNS created_by, goal_type;
DEFINE INDEX IF NOT EXISTS idx_goals_category ON TABLE goals COLUMNS category;
```

**Goal Model Fields:**

```go
// internal/features/goals/models.go
type Goal struct {
    ID          string     `json:"id"`
    Title       string     `json:"title"`
    Description string     `json:"description"`
    
    // Type: "discrete" (binary done/not done) or "quantitative" (measurable target)
    GoalType    string     `json:"goal_type"`
    
    // For quantitative goals
    TargetValue   *float64  `json:"target_value,omitempty"`   // e.g., 1000 (km)
    CurrentValue  float64   `json:"current_value"`            // Computed from task impacts
    Unit          *string   `json:"unit,omitempty"`           // km, hours, pages, count, etc.
    
    // Deadline (optional for open-ended goals)
    Deadline    *time.Time `json:"deadline,omitempty"`
    
    // Status lifecycle
    // not_started | in_progress | completed | abandoned | postponed | paused
    Status      string     `json:"status"`
    
    // Optional category link for domain organization
    Category    *Category  `json:"category,omitempty"`
    
    // Privacy: public | private | sensitive
    Privacy     string     `json:"privacy"`
    
    // Ordering/importance
    Priority    int        `json:"priority"`
    
    // Optional color/icon for UI
    Color       *string    `json:"color,omitempty"`
    Icon        *string    `json:"icon,omitempty"`
    
    // Metadata
    CreatedAt   time.Time  `json:"created_at"`
    UpdatedAt   time.Time  `json:"updated_at"`
    DeletedAt   *time.Time `json:"deleted_at,omitempty"`
    CreatedBy   string     `json:"-"`
    UpdatedBy   string     `json:"-"`
    
    // Computed fields (populated on read)
    PositiveImpactCount  int     `json:"positive_impact_count"`
    NegativeImpactCount  int     `json:"negative_impact_count"`
    NetImpact            float64 `json:"net_impact"`
    ProgressPercent      float64 `json:"progress_percent"` // For quantitative goals
}
```

**Goal Types with Examples:**

| Type | Subtype | Examples |
|------|---------|----------|
| **Discrete** | Binary | "Finish learning React", "Complete a TV show", "Finish Duolingo course" |
| **Discrete** | Project-based | "Plan and execute Europe trip", "Clear backlog in Work category" |
| **Quantitative** | Distance | "Run 1000 km by Dec 31", "Cycle 500 km this quarter" |
| **Quantitative** | Count | "Read 24 books this year", "Complete 100 workouts" |
| **Quantitative** | Duration | "Meditate 1000 hours total", "Practice piano 200 hours" |
| **Quantitative** | Days | "Sleep by 10 PM at least 20 days this month" |
| **Quantitative** | Financial | "Save $5000 by year end", "Reduce expenses by $500/month" |
| **Quantitative** | Weight | "Lose 10 kg by summer", "Gain 5 kg muscle mass" |
| **Quantitative** | Learning | "Learn 2000 vocabulary words", "Complete 50 lessons" |

**Supported Units:**

```go
var ValidUnits = []string{
    // Distance
    "km", "miles", "meters", "steps",
    // Duration
    "hours", "minutes", "days",
    // Count
    "count", "times", "sessions", "reps", "sets",
    // Learning
    "pages", "books", "lessons", "words", "chapters",
    // Financial
    "usd", "eur", "gbp", "amount",
    // Weight
    "kg", "lbs",
    // Volume
    "liters", "glasses", "cups", "ml",
    // Custom
    "tasks", "habits", "percentage",
}
```

### 2.3 Habits Table

```surql
-- db/migrations/005_habits.surql
DEFINE TABLE IF NOT EXISTS habits PERMISSIONS FULL;

DEFINE INDEX IF NOT EXISTS idx_habits_user ON TABLE habits COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_habits_user_active ON TABLE habits COLUMNS created_by, is_active;
DEFINE INDEX IF NOT EXISTS idx_habits_category ON TABLE habits COLUMNS category;
```

**Habit Model Fields:**

```go
// internal/features/habits/models.go
type Habit struct {
    ID          string     `json:"id"`
    Title       string     `json:"title"`
    Description string     `json:"description"`
    
    // Pattern type: "frequency" | "quantity" | "time_bound" | "avoidance"
    PatternType string     `json:"pattern_type"`
    
    // Frequency-based: "X times per day/week/month"
    FrequencyTarget  *int     `json:"frequency_target,omitempty"`   // e.g., 3
    FrequencyPeriod  *string  `json:"frequency_period,omitempty"`   // "day", "week", "month"
    
    // Quantity-based: "X units per period"
    QuantityTarget   *float64 `json:"quantity_target,omitempty"`    // e.g., 3.0
    QuantityUnit     *string  `json:"quantity_unit,omitempty"`      // e.g., "liters"
    QuantityPeriod   *string  `json:"quantity_period,omitempty"`    // "day", "week", "month"
    
    // Time-bound: "do X before/after/between times"
    TimeWindowStart  *string  `json:"time_window_start,omitempty"`  // "HH:MM" format
    TimeWindowEnd    *string  `json:"time_window_end,omitempty"`    // "HH:MM" format
    TimeBoundType    *string  `json:"time_bound_type,omitempty"`    // "before", "after", "between"
    
    // Day-of-week restrictions (nil = all days)
    DaysOfWeek       []int    `json:"days_of_week,omitempty"`       // 0=Sun, 1=Mon, ..., 6=Sat
    
    // Avoidance habits: success = no violating event
    IsAvoidance      bool     `json:"is_avoidance"`
    
    // Grace days: streak not broken if within grace allowance
    GraceDays        int      `json:"grace_days"`                   // per period
    
    // Phase: "ramp" (building up) or "maintain" (steady state)
    Phase            string   `json:"phase"`
    
    // Streak tracking
    StreakCurrent    int      `json:"streak_current"`
    StreakBest       int      `json:"streak_best"`
    LastCompletedAt  *time.Time `json:"last_completed_at,omitempty"`
    
    // Status
    IsActive         bool     `json:"is_active"`
    
    // Optional category
    Category         *Category `json:"category,omitempty"`
    
    // Privacy
    Privacy          string   `json:"privacy"`
    
    // UI
    Color            *string  `json:"color,omitempty"`
    Icon             *string  `json:"icon,omitempty"`
    Emoji            *string  `json:"emoji,omitempty"`
    
    // Linked template for quick logging
    TemplateID       *string  `json:"template_id,omitempty"`
    
    // For habit stacking (parent-child routines)
    ParentHabitID    *string  `json:"parent_habit_id,omitempty"`
    OrderInParent    *int     `json:"order_in_parent,omitempty"`
    
    // Metadata
    CreatedAt        time.Time  `json:"created_at"`
    UpdatedAt        time.Time  `json:"updated_at"`
    DeletedAt        *time.Time `json:"deleted_at,omitempty"`
    CreatedBy        string     `json:"-"`
    
    // Computed (on read)
    TodayProgress    float64    `json:"today_progress"`
    PeriodProgress   float64    `json:"period_progress"`
    SuccessRate      float64    `json:"success_rate"` // last 30 days
}
```

**Habit Pattern Examples:**

| Pattern | Example | Fields Used |
|---------|---------|-------------|
| **Frequency** | "Go to gym 5x per week" | frequency_target=5, frequency_period="week" |
| **Frequency** | "Call mom 3x per week" | frequency_target=3, frequency_period="week" |
| **Quantity** | "Drink 3L water daily" | quantity_target=3, quantity_unit="liters", quantity_period="day" |
| **Quantity** | "Run 100km per month" | quantity_target=100, quantity_unit="km", quantity_period="month" |
| **Time-bound** | "Wake up before 8 AM" | time_window_end="08:00", time_bound_type="before" |
| **Time-bound** | "Take bath by 3 PM" | time_window_end="15:00", time_bound_type="before" |
| **Time-bound** | "Meditate between 6-7 AM" | time_window_start="06:00", time_window_end="07:00", time_bound_type="between" |
| **Avoidance** | "No smoking" | is_avoidance=true |
| **Avoidance** | "No overeating" | is_avoidance=true |
| **Avoidance** | "No social media after 9 PM" | is_avoidance=true, time_window_start="21:00", time_bound_type="after" |

**More Realistic Habit Examples:**

- "Read for at least 30 minutes daily" (quantity, 30 min/day)
- "Exercise at least 4 times per week" (frequency, 4/week)
- "Finish work tasks by 6 PM on weekdays" (time-bound, before 18:00, days=[1,2,3,4,5])
- "Complete all assigned tasks in Work category daily" (frequency + category filter)
- "Journal for 10 minutes before bed" (quantity, time-bound combo)
- "Take vitamins every morning before 9 AM" (frequency=1/day, time_bound before 09:00)
- "Practice coding problems 5x per week" (frequency 5/week)
- "Walk 10,000 steps daily" (quantity 10000 steps/day)
- "No alcohol on weekdays" (avoidance, days=[1,2,3,4,5])
- "Floss teeth every night" (frequency 1/day, time-bound after 20:00)

### 2.4 Task-Goal Impact Table (Edge)

```surql
-- db/migrations/006_task_goal_impacts.surql
DEFINE TABLE task_goal_impacts TYPE RELATION IN tasks OUT goals PERMISSIONS FULL;

DEFINE FIELD polarity ON task_goal_impacts TYPE string 
    ASSERT $value IN ["positive", "negative", "neutral"];
DEFINE FIELD magnitude ON task_goal_impacts TYPE float 
    ASSERT $value >= -1.0 AND $value <= 1.0;
DEFINE FIELD confidence ON task_goal_impacts TYPE float 
    ASSERT $value >= 0.0 AND $value <= 1.0;
DEFINE FIELD notes ON task_goal_impacts TYPE option<string>;
DEFINE FIELD auto_inferred ON task_goal_impacts TYPE bool DEFAULT false;
DEFINE FIELD quantity_delta ON task_goal_impacts TYPE option<float>; -- for quantitative goals
DEFINE FIELD created_at ON task_goal_impacts TYPE datetime DEFAULT time::now();

DEFINE INDEX idx_task_goal_impacts_in ON TABLE task_goal_impacts COLUMNS in;
DEFINE INDEX idx_task_goal_impacts_out ON TABLE task_goal_impacts COLUMNS out;
DEFINE INDEX idx_task_goal_impacts_polarity ON TABLE task_goal_impacts COLUMNS polarity;
```

**Impact Model:**

```go
// internal/features/goals/impact.go
type TaskGoalImpact struct {
    ID            string    `json:"id"`
    TaskID        string    `json:"task_id"`        // in
    GoalID        string    `json:"goal_id"`        // out
    
    // Polarity: positive (helps goal), negative (harms goal), neutral
    Polarity      string    `json:"polarity"`
    
    // Magnitude: -1.0 to +1.0 (intensity of impact)
    // Positive task + high magnitude = strongly helps
    // Negative task + high magnitude = strongly harms
    Magnitude     float64   `json:"magnitude"`
    
    // Confidence: 0.0 to 1.0 (how sure the user is)
    Confidence    float64   `json:"confidence"`
    
    // For quantitative goals: actual delta logged
    // e.g., "Planned 10km, did 4km" -> quantity_delta = 4
    QuantityDelta *float64  `json:"quantity_delta,omitempty"`
    
    // Optional notes for reflection
    Notes         string    `json:"notes,omitempty"`
    
    // Whether system auto-inferred this impact
    AutoInferred  bool      `json:"auto_inferred"`
    
    CreatedAt     time.Time `json:"created_at"`
}
```

**UX/Logic for Honest Impact Marking:**

1. **Non-binary scale**: Use a slider from -1 to +1 instead of just positive/negative
2. **Gentle language**: "This helped a little" vs "strongly supported"
3. **Partial credit**: For quantitative goals, enter actual vs planned
   - "Goal: Run 10km | Logged: Ran 4km" -> Positive impact with delta=4
4. **Quick chips**: After completing a task, show top 3-5 likely impacted goals
5. **Anti-goal prompt**: "Did this work against any goal?" with soft framing
6. **Learning focus**: "What got in the way?" prompt for negative impacts
7. **Confidence slider**: "How sure are you about this impact?"
8. **No shame defaults**: Default magnitude = 0.5 (moderate impact)
9. **Undo/edit**: Allow easy editing of impact assessments

### 2.5 Task-Habit Log Table (Edge)

```surql
-- db/migrations/007_task_habit_logs.surql
DEFINE TABLE task_habit_logs TYPE RELATION IN tasks OUT habits PERMISSIONS FULL;

DEFINE FIELD completion_type ON task_habit_logs TYPE string 
    ASSERT $value IN ["full", "partial", "attempted", "violated"];
DEFINE FIELD quantity_logged ON task_habit_logs TYPE option<float>;
DEFINE FIELD time_logged ON task_habit_logs TYPE option<datetime>;
DEFINE FIELD auto_matched ON task_habit_logs TYPE bool DEFAULT false;
DEFINE FIELD created_at ON task_habit_logs TYPE datetime DEFAULT time::now();

DEFINE INDEX idx_task_habit_logs_in ON TABLE task_habit_logs COLUMNS in;
DEFINE INDEX idx_task_habit_logs_out ON TABLE task_habit_logs COLUMNS out;
```

### 2.6 Goal History Table (Versioning)

```surql
-- db/migrations/008_goal_history.surql
DEFINE TABLE IF NOT EXISTS goal_history PERMISSIONS FULL;

DEFINE INDEX IF NOT EXISTS idx_goal_history_goal ON TABLE goal_history COLUMNS goal_id;
DEFINE INDEX IF NOT EXISTS idx_goal_history_changed ON TABLE goal_history COLUMNS goal_id, changed_at;
```

**Goal History Model:**

```go
type GoalHistory struct {
    ID          string    `json:"id"`
    GoalID      string    `json:"goal_id"`
    ChangedAt   time.Time `json:"changed_at"`
    ChangeType  string    `json:"change_type"`
    // Change types: title, description, deadline, status, target_value, unit, 
    //               category, priority, privacy, task_linked, task_unlinked
    FieldName   string    `json:"field_name,omitempty"`
    OldValue    any       `json:"old_value,omitempty"`
    NewValue    any       `json:"new_value,omitempty"`
    ActorID     string    `json:"actor_id"`
    Notes       string    `json:"notes,omitempty"`
}
```

### 2.7 Habit History Table

```surql
-- db/migrations/009_habit_history.surql
DEFINE TABLE IF NOT EXISTS habit_history PERMISSIONS FULL;

DEFINE INDEX IF NOT EXISTS idx_habit_history_habit ON TABLE habit_history COLUMNS habit_id;
DEFINE INDEX IF NOT EXISTS idx_habit_history_changed ON TABLE habit_history COLUMNS habit_id, changed_at;
```

### 2.8 Templates Table

```surql
-- db/migrations/010_templates.surql
DEFINE TABLE IF NOT EXISTS templates PERMISSIONS FULL;

DEFINE INDEX IF NOT EXISTS idx_templates_user ON TABLE templates COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_templates_category ON TABLE templates COLUMNS default_category;
```

**Template Model:**

```go
// internal/features/templates/models.go
type Template struct {
    ID                  string     `json:"id"`
    Title               string     `json:"title"`
    Description         string     `json:"description"`
    
    // Defaults for quick task creation
    DefaultDuration     *int       `json:"default_duration,omitempty"`     // seconds
    DefaultCategory     *Category  `json:"default_category,omitempty"`
    DefaultPriority     *int       `json:"default_priority,omitempty"`
    
    // Default emotion or expected emotional zone
    DefaultEmotionID    *string    `json:"default_emotion_id,omitempty"`
    ExpectedQuadrant    *string    `json:"expected_quadrant,omitempty"` // yellow/green/red/blue
    
    // Default positives/negatives prompts
    DefaultPositives    []TaskItem `json:"default_positives,omitempty"`
    DefaultNegatives    []TaskItem `json:"default_negatives,omitempty"`
    
    // Auto-linked goals and habits
    LinkedGoalIDs       []string   `json:"linked_goal_ids,omitempty"`
    LinkedHabitIDs      []string   `json:"linked_habit_ids,omitempty"`
    
    // Expected impacts (pre-filled when using template)
    ExpectedImpacts     []ExpectedImpact `json:"expected_impacts,omitempty"`
    
    // Quick log settings
    IsQuickLog          bool       `json:"is_quick_log"`
    QuickLogQuantity    *float64   `json:"quick_log_quantity,omitempty"`
    QuickLogUnit        *string    `json:"quick_log_unit,omitempty"`
    
    // If created from an existing task
    CreatedFromTaskID   *string    `json:"created_from_task_id,omitempty"`
    
    // UI customization
    Color               *string    `json:"color,omitempty"`
    Icon                *string    `json:"icon,omitempty"`
    Emoji               *string    `json:"emoji,omitempty"`
    
    // Ordering for favorites
    SortOrder           int        `json:"sort_order"`
    IsFavorite          bool       `json:"is_favorite"`
    
    // Usage stats
    UsageCount          int        `json:"usage_count"`
    LastUsedAt          *time.Time `json:"last_used_at,omitempty"`
    
    // Metadata
    CreatedAt           time.Time  `json:"created_at"`
    UpdatedAt           time.Time  `json:"updated_at"`
    DeletedAt           *time.Time `json:"deleted_at,omitempty"`
    CreatedBy           string     `json:"-"`
}

type ExpectedImpact struct {
    GoalID        string  `json:"goal_id"`
    Polarity      string  `json:"polarity"`       // positive/negative
    Magnitude     float64 `json:"magnitude"`      // typical magnitude
    QuantityDelta *float64 `json:"quantity_delta,omitempty"` // for quantitative goals
}
```

**Default Templates (System-Provided):**

| Template | Duration | Category | Linked Habits | Expected Impact |
|----------|----------|----------|---------------|-----------------|
| "Drink water" | 30s | Health | hydration_habit | +0.1 per glass |
| "Exercise session" | 60min | Health | exercise_habit | +0.8 fitness goal |
| "Deep work block" | 90min | Work | focus_habit | +0.7 productivity goal |
| "Daily reflection" | 15min | Personal | journaling_habit | +0.5 wellbeing goal |
| "Call a loved one" | 30min | Relationships | social_habit | +0.6 connection goal |
| "Prepare for tomorrow" | 15min | Productivity | planning_habit | +0.4 organization goal |
| "Sleep wind-down" | 30min | Health | sleep_habit | +0.5 sleep goal |
| "Morning routine" | 45min | Personal | morning_habit | Varies |
| "Meal prep" | 60min | Health | nutrition_habit | +0.5 health goal |
| "Read a book" | 30min | Learning | reading_habit | +0.5 reading goal |
| "Meditation" | 15min | Wellness | meditation_habit | +0.6 mindfulness goal |
| "Workout" | 60min | Health | exercise_habit | +0.8 fitness goal |

### 2.9 Task Extension

Extend the existing `tasks` table:

```surql
-- db/migrations/011_tasks_extend.surql
-- Add new indexes for extended functionality
DEFINE INDEX IF NOT EXISTS idx_tasks_template ON TABLE tasks COLUMNS template_id;
DEFINE INDEX IF NOT EXISTS idx_tasks_status ON TABLE tasks COLUMNS created_by, status;
```

**Extended Task Fields:**

```go
// Extend existing Task struct
type Task struct {
    // ... existing fields ...
    
    // Template link (optional)
    TemplateID      *string   `json:"template_id,omitempty"`
    
    // Extended status: completed | postponed | canceled | not_started | in_progress
    Status          string    `json:"status"`
    
    // Computed: matched habits (populated on read)
    MatchedHabits   []string  `json:"matched_habits,omitempty"`
    
    // Computed: goal impacts summary (populated on read)
    GoalImpacts     []TaskGoalImpact `json:"goal_impacts,omitempty"`
    
    // For postponement tracking
    OriginalDate    *time.Time `json:"original_date,omitempty"`
    PostponedCount  int        `json:"postponed_count"`
}
```

### 2.10 Daily Retrospective Table

```surql
-- db/migrations/012_retros_daily.surql
DEFINE TABLE IF NOT EXISTS retros_daily PERMISSIONS FULL;

DEFINE INDEX IF NOT EXISTS idx_retros_daily_user_date ON TABLE retros_daily COLUMNS created_by, date UNIQUE;
```

**Daily Retro Model:**

```go
// internal/features/retros/daily.go
type DailyRetro struct {
    ID              string    `json:"id"`
    Date            string    `json:"date"`            // YYYY-MM-DD
    GeneratedAt     time.Time `json:"generated_at"`
    
    // User's configured retro time snapshot
    ConfiguredTime  string    `json:"configured_time"` // HH:MM
    Timezone        string    `json:"timezone"`
    
    // ===== AUTO-GENERATED AGGREGATES =====
    
    // Task summary
    TasksCompleted      int       `json:"tasks_completed"`
    TasksPostponed      int       `json:"tasks_postponed"`
    TasksCanceled       int       `json:"tasks_canceled"`
    TasksNotStarted     int       `json:"tasks_not_started"`
    TotalTaskTime       int       `json:"total_task_time"`       // seconds
    
    // Emotion summary
    MoodAverage         *EmotionSummary `json:"mood_average,omitempty"`
    MoodDistribution    map[string]int  `json:"mood_distribution"` // quadrant -> count
    EmotionSpikes       []EmotionSpike  `json:"emotion_spikes,omitempty"`
    
    // Habit summary
    HabitsTracked       int              `json:"habits_tracked"`
    HabitsMet           int              `json:"habits_met"`
    HabitsPartial       int              `json:"habits_partial"`
    HabitsMissed        int              `json:"habits_missed"`
    HabitResults        []HabitResult    `json:"habit_results"`
    StreaksUpdated      []StreakUpdate   `json:"streaks_updated,omitempty"`
    
    // Goal impact summary
    GoalImpactsPositive int             `json:"goal_impacts_positive"`
    GoalImpactsNegative int             `json:"goal_impacts_negative"`
    NetGoalImpact       float64         `json:"net_goal_impact"`
    GoalsSummary        []GoalDayStatus `json:"goals_summary"`
    
    // Category breakdown
    TimeByCategory      map[string]int  `json:"time_by_category"` // category_id -> seconds
    TasksByCategory     map[string]int  `json:"tasks_by_category"`
    NeglectedCategories []string        `json:"neglected_categories,omitempty"`
    
    // ===== AUTO-GENERATED INSIGHTS =====
    AutoInsights        []Insight       `json:"auto_insights"`
    
    // ===== USER INPUTS =====
    WhatWentWell        string          `json:"what_went_well,omitempty"`
    WhatDidntGoWell     string          `json:"what_didnt_go_well,omitempty"`
    WhatLearned         string          `json:"what_learned,omitempty"`
    Gratitude           []string        `json:"gratitude,omitempty"`
    TomorrowFocus       string          `json:"tomorrow_focus,omitempty"`
    UserNotes           string          `json:"user_notes,omitempty"`
    
    // User can override/adjust auto insights
    UserOverrides       map[string]any  `json:"user_overrides,omitempty"`
    
    // Metadata
    CreatedBy           string          `json:"-"`
    CreatedAt           time.Time       `json:"created_at"`
    UpdatedAt           time.Time       `json:"updated_at"`
}

type EmotionSummary struct {
    Valence     float64 `json:"valence"`
    Arousal     float64 `json:"arousal"`
    Dominance   float64 `json:"dominance"`
    Quadrant    string  `json:"quadrant"`
    ClosestEmotion string `json:"closest_emotion"`
}

type EmotionSpike struct {
    Time     string  `json:"time"`
    Emotion  string  `json:"emotion"`
    TaskID   string  `json:"task_id"`
    IsHigh   bool    `json:"is_high"` // true = positive spike, false = negative dip
}

type HabitResult struct {
    HabitID        string  `json:"habit_id"`
    HabitTitle     string  `json:"habit_title"`
    Status         string  `json:"status"` // met | partial | missed | not_applicable
    Progress       float64 `json:"progress"` // 0.0 to 1.0+
    Target         string  `json:"target"` // e.g., "3 times" or "2L"
    Achieved       string  `json:"achieved"` // e.g., "2 times" or "1.5L"
}

type StreakUpdate struct {
    HabitID      string `json:"habit_id"`
    HabitTitle   string `json:"habit_title"`
    Action       string `json:"action"` // continued | broken | started | grace_used
    StreakBefore int    `json:"streak_before"`
    StreakAfter  int    `json:"streak_after"`
}

type GoalDayStatus struct {
    GoalID            string  `json:"goal_id"`
    GoalTitle         string  `json:"goal_title"`
    ImpactToday       float64 `json:"impact_today"`
    ProgressDelta     float64 `json:"progress_delta"` // for quantitative goals
    TotalProgress     float64 `json:"total_progress"`
}

type Insight struct {
    Type       string `json:"type"`       // positive | negative | neutral | suggestion
    Category   string `json:"category"`   // productivity | emotion | habits | goals | balance
    Title      string `json:"title"`
    Message    string `json:"message"`
    Confidence float64 `json:"confidence"` // 0.0 to 1.0
}
```

### 2.11 Range Retrospective Table

```surql
-- db/migrations/013_retros_range.surql
DEFINE TABLE IF NOT EXISTS retros_range PERMISSIONS FULL;

DEFINE INDEX IF NOT EXISTS idx_retros_range_user ON TABLE retros_range COLUMNS created_by;
DEFINE INDEX IF NOT EXISTS idx_retros_range_dates ON TABLE retros_range COLUMNS created_by, start_date, end_date;
```

**Range Retro Model:**

```go
type RangeRetro struct {
    ID              string    `json:"id"`
    StartDate       string    `json:"start_date"`      // YYYY-MM-DD
    EndDate         string    `json:"end_date"`        // YYYY-MM-DD
    RangeType       string    `json:"range_type"`      // week | month | quarter | year | custom
    
    // ===== AGGREGATED DATA =====
    
    // Task aggregates
    TotalTasks          int     `json:"total_tasks"`
    TasksCompleted      int     `json:"tasks_completed"`
    TasksPostponed      int     `json:"tasks_postponed"`
    TasksCanceled       int     `json:"tasks_canceled"`
    CompletionRate      float64 `json:"completion_rate"`
    TotalTime           int     `json:"total_time"` // seconds
    AvgDailyTime        int     `json:"avg_daily_time"`
    
    // Emotion aggregates
    AvgValence          float64            `json:"avg_valence"`
    AvgArousal          float64            `json:"avg_arousal"`
    AvgDominance        float64            `json:"avg_dominance"`
    EmotionVariability  float64            `json:"emotion_variability"`
    QuadrantDistribution map[string]float64 `json:"quadrant_distribution"` // quadrant -> percentage
    TopEmotions         []EmotionCount     `json:"top_emotions"`
    
    // Goal aggregates
    GoalsAdvanced       []GoalProgress     `json:"goals_advanced"`
    GoalsNegativeImpact []GoalProgress     `json:"goals_negative_impact"`
    NetImpactByGoal     map[string]float64 `json:"net_impact_by_goal"`
    
    // Habit aggregates
    HabitSuccessRates   []HabitStats       `json:"habit_success_rates"`
    BestStreaks         []StreakInfo       `json:"best_streaks"`
    HabitsConsistency   float64            `json:"habits_consistency"`
    
    // Category aggregates
    TimeByCategory      map[string]int     `json:"time_by_category"`
    TasksByCategory     map[string]int     `json:"tasks_by_category"`
    CategoryBalance     float64            `json:"category_balance"` // 0-1, higher = more balanced
    
    // Pattern insights
    MostProductiveDays  []string           `json:"most_productive_days"`
    MostProductiveHours []int              `json:"most_productive_hours"`
    CommonObstacles     []string           `json:"common_obstacles"`
    AntiGoalPatterns    []AntiGoalPattern  `json:"anti_goal_patterns,omitempty"`
    
    // ===== AUTO INSIGHTS =====
    AutoInsights        []Insight          `json:"auto_insights"`
    
    // ===== USER INPUTS =====
    Reflection          string             `json:"reflection,omitempty"`
    KeyLearnings        []string           `json:"key_learnings,omitempty"`
    Improvements        []string           `json:"improvements,omitempty"`
    NextPeriodGoals     []string           `json:"next_period_goals,omitempty"`
    UserNotes           string             `json:"user_notes,omitempty"`
    
    // Metadata
    GeneratedAt         time.Time          `json:"generated_at"`
    CreatedBy           string             `json:"-"`
    CreatedAt           time.Time          `json:"created_at"`
    UpdatedAt           time.Time          `json:"updated_at"`
}

type EmotionCount struct {
    EmotionID   string `json:"emotion_id"`
    EmotionName string `json:"emotion_name"`
    Emoji       string `json:"emoji"`
    Count       int    `json:"count"`
}

type GoalProgress struct {
    GoalID        string  `json:"goal_id"`
    GoalTitle     string  `json:"goal_title"`
    GoalType      string  `json:"goal_type"`
    NetImpact     float64 `json:"net_impact"`
    ProgressDelta float64 `json:"progress_delta"` // for quantitative
    ProgressPercent float64 `json:"progress_percent"`
}

type HabitStats struct {
    HabitID      string  `json:"habit_id"`
    HabitTitle   string  `json:"habit_title"`
    SuccessRate  float64 `json:"success_rate"` // 0.0 to 1.0
    TotalDays    int     `json:"total_days"`
    DaysMet      int     `json:"days_met"`
    CurrentStreak int    `json:"current_streak"`
}

type StreakInfo struct {
    HabitID     string `json:"habit_id"`
    HabitTitle  string `json:"habit_title"`
    StreakDays  int    `json:"streak_days"`
}

type AntiGoalPattern struct {
    GoalID       string   `json:"goal_id"`
    GoalTitle    string   `json:"goal_title"`
    PatternDesc  string   `json:"pattern_desc"`
    Frequency    int      `json:"frequency"`
    TaskExamples []string `json:"task_examples"`
}
```

### 2.12 Daily Summaries Table (Pre-computed)

```surql
-- db/migrations/014_daily_summaries.surql
DEFINE TABLE IF NOT EXISTS daily_summaries PERMISSIONS FULL;

DEFINE INDEX IF NOT EXISTS idx_daily_summaries_user_date ON TABLE daily_summaries COLUMNS created_by, date UNIQUE;
```

**Daily Summary Model (for fast analytics):**

```go
type DailySummary struct {
    ID                  string                 `json:"id"`
    Date                string                 `json:"date"` // YYYY-MM-DD
    
    // Task counts
    TasksCreated        int                    `json:"tasks_created"`
    TasksCompleted      int                    `json:"tasks_completed"`
    TasksPostponed      int                    `json:"tasks_postponed"`
    TasksCanceled       int                    `json:"tasks_canceled"`
    
    // Time metrics (seconds)
    TotalTime           int                    `json:"total_time"`
    TimeByCategory      map[string]int         `json:"time_by_category"`
    TimeByHour          map[int]int            `json:"time_by_hour"` // hour -> seconds
    
    // Emotion metrics
    EmotionCentroid     EmotionSummary         `json:"emotion_centroid"`
    QuadrantCounts      map[string]int         `json:"quadrant_counts"`
    ValenceSum          float64                `json:"valence_sum"`
    ArousalSum          float64                `json:"arousal_sum"`
    EmotionCount        int                    `json:"emotion_count"`
    
    // Goal impacts
    PositiveImpactSum   float64                `json:"positive_impact_sum"`
    NegativeImpactSum   float64                `json:"negative_impact_sum"`
    ImpactsByGoal       map[string]float64     `json:"impacts_by_goal"`
    
    // Habit results
    HabitResults        map[string]string      `json:"habit_results"` // habit_id -> met|partial|missed
    
    // For fast aggregation
    CreatedBy           string                 `json:"-"`
    ComputedAt          time.Time              `json:"computed_at"`
}
```

### 2.13 User Preferences Table

```surql
-- db/migrations/015_user_preferences.surql
DEFINE TABLE IF NOT EXISTS user_preferences SCHEMAFULL PERMISSIONS FULL;

DEFINE FIELD user_id ON user_preferences TYPE string;
DEFINE FIELD timezone ON user_preferences TYPE string DEFAULT "UTC";
DEFINE FIELD daily_retro_time ON user_preferences TYPE string DEFAULT "21:00"; -- HH:MM local
DEFINE FIELD week_start_day ON user_preferences TYPE int DEFAULT 1; -- 0=Sun, 1=Mon
DEFINE FIELD notifications_enabled ON user_preferences TYPE bool DEFAULT true;
DEFINE FIELD privacy_default ON user_preferences TYPE string DEFAULT "private";
DEFINE FIELD theme ON user_preferences TYPE string DEFAULT "auto";
DEFINE FIELD quick_log_favorites ON user_preferences TYPE array DEFAULT [];
DEFINE FIELD created_at ON user_preferences TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON user_preferences TYPE datetime DEFAULT time::now();

DEFINE INDEX idx_user_preferences_user ON TABLE user_preferences COLUMNS user_id UNIQUE;
```

### 2.14 Complete Migration Order

```
db/migrations/
├── 001_core.surql           (existing: users, categories, tasks)
├── 002_task_emotions.surql  (existing: task_emotions edge)
├── 003_seed_emotions.surql  (existing: 100 emotions seed)
├── 004_goals.surql
├── 005_habits.surql
├── 006_task_goal_impacts.surql
├── 007_task_habit_logs.surql
├── 008_goal_history.surql
├── 009_habit_history.surql
├── 010_templates.surql
├── 011_tasks_extend.surql
├── 012_retros_daily.surql
├── 013_retros_range.surql
├── 014_daily_summaries.surql
├── 015_user_preferences.surql
```

---

## 3. API Design

### 3.1 Goals API

```
POST   /api/v1/goals                     Create goal
GET    /api/v1/goals                     List goals (filters: status, type, deadline, category)
GET    /api/v1/goals/:id                 Get goal by ID
PUT    /api/v1/goals/:id                 Update goal (logs to history)
DELETE /api/v1/goals/:id                 Soft delete goal
PATCH  /api/v1/goals/:id/status          Update status only
GET    /api/v1/goals/:id/history         Get goal change history
GET    /api/v1/goals/:id/impacts         Get all task impacts for goal
POST   /api/v1/goals/:id/progress        Log progress for quantitative goal
```

**Create Goal Request:**

```go
type CreateGoalRequest struct {
    Title        string   `json:"title" validate:"required,min=1,max=500"`
    Description  string   `json:"description" validate:"max=5000"`
    GoalType     string   `json:"goal_type" validate:"required,oneof=discrete quantitative"`
    TargetValue  *float64 `json:"target_value,omitempty" validate:"omitempty,gt=0"`
    Unit         *string  `json:"unit,omitempty" validate:"omitempty,valid_unit"`
    Deadline     *string  `json:"deadline,omitempty" validate:"omitempty,datetime_flexible"`
    CategoryID   *string  `json:"category_id,omitempty"`
    Priority     int      `json:"priority" validate:"min=-100,max=100"`
    Privacy      string   `json:"privacy" validate:"required,oneof=public private sensitive"`
    Color        *string  `json:"color,omitempty" validate:"omitempty,hexcolor"`
    Icon         *string  `json:"icon,omitempty"`
}
```

### 3.2 Habits API

```
POST   /api/v1/habits                    Create habit
GET    /api/v1/habits                    List habits (filters: is_active, pattern_type, category)
GET    /api/v1/habits/:id                Get habit by ID
PUT    /api/v1/habits/:id                Update habit (logs to history)
DELETE /api/v1/habits/:id                Soft delete habit
PATCH  /api/v1/habits/:id/active         Toggle active status
GET    /api/v1/habits/:id/history        Get habit change history
GET    /api/v1/habits/:id/logs           Get habit completion logs
GET    /api/v1/habits/:id/streak         Get streak details
POST   /api/v1/habits/:id/log            Manual habit log entry
```

**Create Habit Request:**

```go
type CreateHabitRequest struct {
    Title            string   `json:"title" validate:"required,min=1,max=500"`
    Description      string   `json:"description" validate:"max=5000"`
    PatternType      string   `json:"pattern_type" validate:"required,oneof=frequency quantity time_bound avoidance"`
    
    // Frequency pattern
    FrequencyTarget  *int     `json:"frequency_target,omitempty" validate:"omitempty,min=1"`
    FrequencyPeriod  *string  `json:"frequency_period,omitempty" validate:"omitempty,oneof=day week month"`
    
    // Quantity pattern
    QuantityTarget   *float64 `json:"quantity_target,omitempty" validate:"omitempty,gt=0"`
    QuantityUnit     *string  `json:"quantity_unit,omitempty" validate:"omitempty,valid_unit"`
    QuantityPeriod   *string  `json:"quantity_period,omitempty" validate:"omitempty,oneof=day week month"`
    
    // Time-bound pattern
    TimeWindowStart  *string  `json:"time_window_start,omitempty" validate:"omitempty,time_hhmm"`
    TimeWindowEnd    *string  `json:"time_window_end,omitempty" validate:"omitempty,time_hhmm"`
    TimeBoundType    *string  `json:"time_bound_type,omitempty" validate:"omitempty,oneof=before after between"`
    
    // Day restrictions
    DaysOfWeek       []int    `json:"days_of_week,omitempty" validate:"omitempty,dive,min=0,max=6"`
    
    // Avoidance
    IsAvoidance      bool     `json:"is_avoidance"`
    
    // Grace and phase
    GraceDays        int      `json:"grace_days" validate:"min=0,max=7"`
    Phase            string   `json:"phase" validate:"oneof=ramp maintain"`
    
    // Links
    CategoryID       *string  `json:"category_id,omitempty"`
    TemplateID       *string  `json:"template_id,omitempty"`
    
    // UI
    Color            *string  `json:"color,omitempty"`
    Icon             *string  `json:"icon,omitempty"`
    Emoji            *string  `json:"emoji,omitempty"`
    
    Privacy          string   `json:"privacy" validate:"required,oneof=public private sensitive"`
}
```

### 3.3 Task Impacts API

```
POST   /api/v1/tasks/:id/impacts              Add goal impact to task
GET    /api/v1/tasks/:id/impacts              Get all impacts for task
PUT    /api/v1/tasks/:id/impacts/:impactId    Update impact
DELETE /api/v1/tasks/:id/impacts/:impactId    Remove impact
POST   /api/v1/tasks/:id/impacts/suggest      Auto-suggest impacts (from template/category/keywords)
```

**Add Impact Request:**

```go
type AddTaskImpactRequest struct {
    GoalID        string   `json:"goal_id" validate:"required"`
    Polarity      string   `json:"polarity" validate:"required,oneof=positive negative neutral"`
    Magnitude     float64  `json:"magnitude" validate:"min=-1,max=1"`
    Confidence    float64  `json:"confidence" validate:"min=0,max=1"`
    QuantityDelta *float64 `json:"quantity_delta,omitempty"`
    Notes         string   `json:"notes,omitempty" validate:"max=1000"`
}
```

### 3.4 Templates API

```
POST   /api/v1/templates                       Create template
GET    /api/v1/templates                       List templates (filters: is_favorite, category)
GET    /api/v1/templates/:id                   Get template by ID
PUT    /api/v1/templates/:id                   Update template
DELETE /api/v1/templates/:id                   Delete template
POST   /api/v1/templates/:id/instantiate       Create task from template
POST   /api/v1/templates/from-task/:taskId     Create template from existing task
PATCH  /api/v1/templates/:id/favorite          Toggle favorite
```

**Instantiate Template Request:**

```go
type InstantiateTemplateRequest struct {
    // Overrides for template defaults
    StartDate  *string    `json:"start_date,omitempty" validate:"omitempty,datetime_flexible"`
    EndDate    *string    `json:"end_date,omitempty" validate:"omitempty,datetime_flexible"`
    Title      *string    `json:"title,omitempty"`
    CategoryID *string    `json:"category_id,omitempty"`
    Quantity   *float64   `json:"quantity,omitempty"` // for quick log quantity
    
    // Auto-apply template's linked impacts
    ApplyImpacts bool     `json:"apply_impacts"`
}
```

### 3.5 Retrospectives API

```
# User preferences
PUT    /api/v1/user/preferences               Update preferences (timezone, daily_retro_time)
GET    /api/v1/user/preferences               Get preferences

# Daily retrospectives
GET    /api/v1/retros/daily/:date             Get daily retro (auto-generates if missing)
POST   /api/v1/retros/daily/:date/regenerate  Force regenerate daily retro
PATCH  /api/v1/retros/daily/:id               Update user notes/overrides
GET    /api/v1/retros/daily                   List daily retros (paginated, date range)

# Range retrospectives
POST   /api/v1/retros/range                   Generate range retro
GET    /api/v1/retros/range/:id               Get range retro
PATCH  /api/v1/retros/range/:id               Update user notes
GET    /api/v1/retros/range                   List range retros
DELETE /api/v1/retros/range/:id               Delete range retro
```

**Update Preferences Request:**

```go
type UpdatePreferencesRequest struct {
    Timezone           *string  `json:"timezone,omitempty" validate:"omitempty,timezone"`
    DailyRetroTime     *string  `json:"daily_retro_time,omitempty" validate:"omitempty,time_hhmm"`
    WeekStartDay       *int     `json:"week_start_day,omitempty" validate:"omitempty,min=0,max=6"`
    NotificationsEnabled *bool  `json:"notifications_enabled,omitempty"`
    PrivacyDefault     *string  `json:"privacy_default,omitempty" validate:"omitempty,oneof=public private sensitive"`
}
```

**Create Range Retro Request:**

```go
type CreateRangeRetroRequest struct {
    StartDate string `json:"start_date" validate:"required,date"`
    EndDate   string `json:"end_date" validate:"required,date,gtfield=StartDate"`
    RangeType string `json:"range_type" validate:"oneof=week month quarter year custom"`
}
```

### 3.6 Analytics API

```
# Goal analytics
GET    /api/v1/analytics/goals/progress        Goal progress over time
GET    /api/v1/analytics/goals/impacts         Net impacts by goal
GET    /api/v1/analytics/goals/completion      Goal completion stats

# Habit analytics
GET    /api/v1/analytics/habits/streaks        Current and best streaks
GET    /api/v1/analytics/habits/success-rate   Success rate by habit
GET    /api/v1/analytics/habits/calendar       Calendar heatmap data
GET    /api/v1/analytics/habits/by-time        Success by time-of-day and day-of-week

# Category analytics
GET    /api/v1/analytics/categories/time       Time spent by category
GET    /api/v1/analytics/categories/tasks      Task count by category
GET    /api/v1/analytics/categories/balance    Category balance (radar data)
GET    /api/v1/analytics/categories/emotions   Emotion correlation by category

# Task analytics
GET    /api/v1/analytics/tasks/completion      Completion trends
GET    /api/v1/analytics/tasks/postponements   Postponement chains
GET    /api/v1/analytics/tasks/duration        Duration distribution

# Emotion analytics
GET    /api/v1/analytics/emotions/trends       Mood time series
GET    /api/v1/analytics/emotions/heatmap      Emotion heatmap (hour x quadrant)
GET    /api/v1/analytics/emotions/distribution Emotion distribution
GET    /api/v1/analytics/emotions/correlations Emotion-category/goal correlations

# Meta analytics
GET    /api/v1/analytics/honesty              Honesty ratio (setbacks logged vs wins)
GET    /api/v1/analytics/balance              Life domain balance
GET    /api/v1/analytics/summary              Overall summary for date range
```

**Common Analytics Query Params:**

```
?start_date=2024-01-01
&end_date=2024-12-31
&granularity=day|week|month
&goal_id=goals:xxx (filter)
&habit_id=habits:xxx (filter)
&category_id=categories:xxx (filter)
```

### 3.7 Middleware and Validation

**New Validators:**

```go
// internal/shared/validator/validators.go

// valid_unit: checks against ValidUnits list
// time_hhmm: validates HH:MM format
// timezone: validates IANA timezone
// date: validates YYYY-MM-DD format
// hexcolor: validates #RRGGBB format
```

**Rate Limiting (analytics):**

```go
// Suggested limits for analytics endpoints
// - 60 requests/minute per user for summary endpoints
// - 20 requests/minute per user for heavy aggregation endpoints
```

---

## 4. Feature Breakdown

### 4.1 Goals

#### Core (MVP)
- [ ] Goal CRUD with title, description, type (discrete/quantitative), target, unit
- [ ] Status lifecycle: not_started -> in_progress -> completed/abandoned/postponed
- [ ] Deadline support (optional)
- [ ] Link tasks to goals with polarity (positive/negative/neutral) and magnitude
- [ ] History logging for all goal changes
- [ ] Category association for domain organization
- [ ] Privacy levels (public/private/sensitive)
- [ ] Basic progress calculation for quantitative goals

#### Nice to Have
- [ ] Priority ordering for goals
- [ ] Rollover tracking when deadline is extended (log as goal_history event)
- [ ] Partial credit input for quantitative goals ("planned 10km, did 4km")
- [ ] Goal templates (pre-defined goal structures)
- [ ] Goal tags/labels
- [ ] Goal milestones (sub-goals)
- [ ] Goal dependencies (goal X depends on goal Y)
- [ ] Goal sharing with privacy controls
- [ ] Color/icon customization

#### Advanced
- [ ] Auto-suggestion of impacted goals when completing a task (NLP/keyword matching)
- [ ] Goal confidence scoring based on historical performance
- [ ] Drift detection and reminder nudges ("You haven't worked on X in 5 days")
- [ ] Goal forecasting ("At current rate, you'll complete by...")
- [ ] Goal comparison (vs previous periods, vs other users optionally)
- [ ] Smart goal recommendations based on patterns

### 4.2 Habits

#### Core (MVP)
- [ ] Habit CRUD with pattern types: frequency, quantity, time-bound, avoidance
- [ ] Frequency: X times per day/week/month
- [ ] Quantity: X units per day/week/month
- [ ] Day-of-week restrictions
- [ ] Streak tracking (current and best)
- [ ] Active/inactive toggle
- [ ] Link habits to templates for quick logging
- [ ] Passive task matching (by category, template, explicit link)
- [ ] History logging for habit changes

#### Nice to Have
- [ ] Time-bound patterns: do X before/after/between times
- [ ] Avoidance habits: success = no violating event
- [ ] Grace days: streak not broken within grace allowance
- [ ] Phase support: ramp (building up) vs maintain (steady state)
- [ ] Habit stacking/routines: parent habit with ordered sub-habits
- [ ] Manual habit log entry (for habits not tied to tasks)
- [ ] Habit reminders and notifications
- [ ] Habit calendar view
- [ ] Color/icon/emoji customization

#### Advanced
- [ ] Adaptive targets: system suggests increasing/decreasing based on success
- [ ] Anomaly detection: late completions, sudden streak breaks
- [ ] Contextual nudges: time-of-day suggestions
- [ ] Habit correlations: "When you do X, you're more likely to do Y"
- [ ] Habit difficulty scoring
- [ ] Habit challenge mode (30-day challenges)
- [ ] Cross-midnight time windows
- [ ] Backfill logging for missed days

### 4.3 Metrics

#### Core (MVP)
- [ ] Task counts: created, completed, postponed, canceled per period
- [ ] Time spent per category
- [ ] Mood averages (valence, arousal, dominance) per period
- [ ] Emotion distribution by quadrant
- [ ] Habit success rate
- [ ] Streak lengths
- [ ] Net positive vs negative impact per goal

#### Nice to Have
- [ ] Emotional variability (how much mood fluctuates)
- [ ] Time-of-day success patterns for habits
- [ ] Balance score across life domains (categories)
- [ ] Honesty ratio: negative/anti-goal logs vs positive logs
- [ ] Productivity score (composite metric)
- [ ] Consistency score for habits
- [ ] Average task duration
- [ ] Postponement frequency

#### Advanced
- [ ] Flow proximity metrics (based on emotion research)
- [ ] Emotional inertia (how long moods persist)
- [ ] Burnout/anxiety risk indicators
- [ ] Personality trait inference (Big Five from long-term patterns)
- [ ] EQ assessment from emotion patterns
- [ ] Predictive metrics (streak break risk, goal completion probability)
- [ ] Correlation discovery (what activities correlate with good moods)

### 4.4 Task Templates

#### Core (MVP)
- [ ] Template CRUD with title, default duration, category, priority
- [ ] Instantiate task from template with overrides
- [ ] Convert existing task to template
- [ ] Link template to goals and habits
- [ ] Template_id stored on tasks for reference
- [ ] Favorite templates for quick access
- [ ] Default system templates (drink water, exercise, etc.)

#### Nice to Have
- [ ] Default emotions/expected emotional zone
- [ ] Default positives/negatives prompts
- [ ] Expected impact suggestions (auto-fill impact on linked goals)
- [ ] Quick log mode (one-tap task creation with defaults)
- [ ] Template usage tracking (count, last used)
- [ ] Template categories/folders
- [ ] Template import/export

#### Advanced
- [ ] Template performance analytics (avg mood after using template)
- [ ] Smart template suggestions based on time of day/context
- [ ] Template versioning (track changes to templates)
- [ ] Community templates (share with others)
- [ ] Template automation (auto-create task at certain times)

### 4.5 Retrospectives

#### Core (MVP)
- [ ] User preference: daily retro time (HH:MM) and timezone
- [ ] Auto-generate daily retro at configured time
- [ ] Daily retro includes: task summary, emotion avg, habit results, goal impacts
- [ ] Retrieve daily retro by date
- [ ] Edit daily retro: user notes, what went well/didn't, learnings
- [ ] Manual retro regeneration

#### Nice to Have
- [ ] Custom date-range retros (week, month, quarter, year, custom)
- [ ] Auto-insights: most productive day, common emotion, biggest impact
- [ ] Reflection prompts library (rotating prompts)
- [ ] Streak updates in daily retro
- [ ] Category breakdown in retros
- [ ] Neglected category detection
- [ ] User can override/adjust auto-generated insights
- [ ] Export retro as PDF/markdown

#### Advanced
- [ ] Adaptive prompts based on detected patterns
- [ ] Anomaly alerts in retros (unusual spikes/drops)
- [ ] Comparative retros ("this week vs last week")
- [ ] Shareable retros (with privacy controls)
- [ ] Voice-to-text for retro notes
- [ ] Retro reminders/notifications

### 4.6 Analytics & Charts

#### Core (MVP)
- [ ] Time-series: goal progress over time
- [ ] Time-series: mood (valence) over time
- [ ] Bar chart: tasks completed/postponed/canceled by day
- [ ] Pie chart: emotion quadrant distribution
- [ ] Calendar heatmap: habit streaks

#### Nice to Have
- [ ] Stacked bar: positive vs negative impacts per goal
- [ ] Heatmap: emotion by hour of day
- [ ] Heatmap: habit success by day of week
- [ ] Radar/spider: category balance
- [ ] Histogram: task duration distribution
- [ ] Line chart: habit success rate trend
- [ ] Tables with sparklines: top goals, top habits

#### Advanced
- [ ] Scatter plot: mood vs time spent in category
- [ ] Predictive trend lines
- [ ] Postponement chain visualization
- [ ] Correlation matrix: categories vs emotions
- [ ] Flow state detection visualization
- [ ] Comparative charts (vs previous period)
- [ ] Custom chart builder

---

## 5. Analytics & Insights

### 5.1 Metrics Library

| Category | Metric | Formula/Description | Chart Type |
|----------|--------|---------------------|------------|
| **Productivity** | Tasks completed | COUNT(tasks WHERE completed=true) per period | Time-series |
| **Productivity** | Completion rate | completed / (completed + postponed + canceled) | Percentage |
| **Productivity** | Postponement count | COUNT(tasks WHERE postponed_count > 0) | Bar chart |
| **Productivity** | Avg task duration | AVG(end_date - start_date) | Histogram |
| **Time** | Time by category | SUM(duration) GROUP BY category | Stacked bar |
| **Time** | Time by goal | SUM(duration) for tasks linked to goal | Pie chart |
| **Time** | Peak hours | COUNT(tasks) GROUP BY hour | Bar chart |
| **Emotion** | Avg valence | MEAN(inferred_emotion.valence) | Time-series |
| **Emotion** | Avg arousal | MEAN(inferred_emotion.arousal) | Time-series |
| **Emotion** | Quadrant distribution | COUNT BY quadrant | Pie chart |
| **Emotion** | Variability | STDDEV(valence) | Single value |
| **Emotion** | Top emotions | COUNT BY emotion_id ORDER DESC | Bar chart |
| **Goals** | Net impact | SUM(magnitude WHERE polarity=positive) - SUM(magnitude WHERE polarity=negative) | Per goal |
| **Goals** | Progress % | current_value / target_value * 100 | Progress bar |
| **Goals** | Anti-goal ratio | negative_impacts / positive_impacts | Percentage |
| **Habits** | Success rate | days_met / total_days | Percentage |
| **Habits** | Current streak | consecutive days met | Number |
| **Habits** | Best streak | MAX(streak) | Number |
| **Habits** | On-time rate | met_on_time / total_met (for time-bound) | Percentage |
| **Balance** | Category spread | Gini coefficient of time by category | Radar chart |
| **Balance** | Neglected domains | Categories with < X% of time | List |
| **Honesty** | Setback ratio | negative_logs / total_logs | Percentage |

### 5.2 Pre-computation Strategy

**On-Write Denormalization:**
- Store task duration on task record
- Store primary category_id on task
- Store net_impact on task (sum of linked impacts)
- Update goal.current_value when impact is added

**Scheduled Jobs (Cron):**
- Daily at user's retro time: Generate daily retro
- Daily at midnight UTC: Compute daily_summary for all users
- Daily: Update habit streaks
- Weekly: Generate weekly insights cache

**Materialized Daily Summaries:**

```go
// Computed daily and stored
type DailySummary struct {
    Date                string
    TasksCompleted      int
    TasksPostponed      int
    TasksCanceled       int
    TotalTime           int
    TimeByCategory      map[string]int
    TimeByHour          map[int]int
    EmotionCentroid     EmotionSummary
    QuadrantCounts      map[string]int
    ImpactsByGoal       map[string]float64
    HabitResults        map[string]string
}
```

**Query Patterns (SurrealDB):**

```surql
-- Daily retro aggregation
LET $day_start = time::floor($target_date, 1d);
LET $day_end = $day_start + 1d;

SELECT 
    count() as total,
    count(completed = true) as completed,
    math::sum(duration) as total_time
FROM tasks 
WHERE created_by = $user_id 
    AND start_date >= $day_start 
    AND start_date < $day_end 
    AND deleted_at = NONE;

-- Habit evaluation for day
SELECT *, 
    (SELECT count() FROM task_habit_logs 
     WHERE out = habits.id 
       AND time::floor(created_at, 1d) = $target_date) as day_count
FROM habits 
WHERE created_by = $user_id AND is_active = true;

-- Goal impact aggregation
SELECT 
    out.id as goal_id,
    out.title as goal_title,
    math::sum(IF polarity = "positive" THEN magnitude ELSE 0 END) as positive_sum,
    math::sum(IF polarity = "negative" THEN magnitude ELSE 0 END) as negative_sum
FROM task_goal_impacts
WHERE in.created_by = $user_id 
    AND time::floor(created_at, 1d) = $target_date
GROUP BY out;

-- Emotion distribution
SELECT 
    inferred_emotion.quadrant as quadrant,
    count() as count
FROM tasks
WHERE created_by = $user_id 
    AND start_date >= $day_start 
    AND start_date < $day_end
    AND inferred_emotion != NONE
GROUP BY inferred_emotion.quadrant;
```

### 5.3 Chart Specifications

| Chart | Data Structure | Endpoint |
|-------|----------------|----------|
| **Goal Progress Line** | `[{date, value, target}]` | `/analytics/goals/progress` |
| **Mood Time-Series** | `[{date, valence, arousal, dominance}]` | `/analytics/emotions/trends` |
| **Task Status Stacked Bar** | `[{date, completed, postponed, canceled}]` | `/analytics/tasks/completion` |
| **Quadrant Pie** | `{yellow: n, green: n, red: n, blue: n}` | `/analytics/emotions/distribution` |
| **Habit Calendar Heatmap** | `[{date, status: met|partial|missed}]` | `/analytics/habits/calendar` |
| **Category Time Stacked Bar** | `[{date, category_a: mins, category_b: mins, ...}]` | `/analytics/categories/time` |
| **Emotion Hour Heatmap** | `[{hour, quadrant, count}]` | `/analytics/emotions/heatmap` |
| **Category Balance Radar** | `{category_a: score, category_b: score, ...}` | `/analytics/categories/balance` |
| **Habit DOW Heatmap** | `[{day_of_week, habit_id, success_rate}]` | `/analytics/habits/by-time` |
| **Impact per Goal Bar** | `[{goal_id, title, positive, negative, net}]` | `/analytics/goals/impacts` |

---

## 6. Step-by-Step Implementation Plan

### Phase 0: Foundation (Week 1-2)

**Goal**: Set up infrastructure for new features

1. **User Preferences**
   - Migration: `015_user_preferences.surql`
   - Model: `internal/features/users/preferences.go`
   - API: `PUT/GET /api/v1/user/preferences`
   - Add timezone and daily_retro_time fields

2. **Extend Tasks**
   - Migration: `011_tasks_extend.surql` (add status field, template_id index)
   - Update task model with Status field
   - Update task service to handle status transitions

3. **Goals Foundation**
   - Migration: `004_goals.surql`
   - Model: `internal/features/goals/models.go`
   - Repository: `internal/features/goals/repository.go`
   - Service: `internal/features/goals/service.go`
   - Handler: `internal/features/goals/handler.go`
   - API: Basic CRUD only

4. **Task-Goal Impact Edge**
   - Migration: `006_task_goal_impacts.surql`
   - Model: `internal/features/goals/impact.go`
   - API: `POST/GET/PUT/DELETE /tasks/:id/impacts`

**Deliverable**: Goals CRUD + Task-Goal linking working

### Phase 1: Habits MVP (Week 3-4)

1. **Habits Foundation**
   - Migration: `005_habits.surql`
   - Model: `internal/features/habits/models.go`
   - Repository, Service, Handler
   - API: Basic CRUD

2. **Task-Habit Logging**
   - Migration: `007_task_habit_logs.surql`
   - Passive matching: when task is created/updated, check if it matches habit criteria

3. **Streak Calculation**
   - Service method to calculate current streak
   - Background job to update streaks daily

4. **Goal/Habit History**
   - Migrations: `008_goal_history.surql`, `009_habit_history.surql`
   - Auto-log changes in service layer
   - API: `GET /goals/:id/history`, `GET /habits/:id/history`

**Deliverable**: Habits with streak tracking, history logging

### Phase 2: Templates (Week 5)

1. **Templates Foundation**
   - Migration: `010_templates.surql`
   - Model: `internal/features/templates/models.go`
   - Repository, Service, Handler
   - API: CRUD + instantiate + from-task

2. **Template Linking**
   - Store template_id on instantiated tasks
   - Link templates to goals/habits

3. **Default Templates**
   - Seed migration with system templates
   - Or lazy-load on first user access

**Deliverable**: Templates with quick task creation

### Phase 3: Daily Retro MVP (Week 6-7)

1. **Daily Summary Infrastructure**
   - Migration: `014_daily_summaries.surql`
   - Background job: compute daily summary at end of day

2. **Daily Retro Generation**
   - Migration: `012_retros_daily.surql`
   - Model: `internal/features/retros/daily.go`
   - Service: aggregate from daily summary + raw tasks
   - Scheduler: trigger at user's configured retro time

3. **Retro API**
   - API: `GET /retros/daily/:date` (auto-generate if missing)
   - API: `PATCH /retros/daily/:id` (user notes)
   - API: `POST /retros/daily/:date/regenerate`

4. **Auto-Insights Engine**
   - Rules-based insight generation
   - Insert into retro.auto_insights

**Deliverable**: Daily retros auto-generated with insights

### Phase 4: Analytics Core (Week 8-9)

1. **Analytics Service**
   - `internal/features/analytics/service.go`
   - Query methods for each metric type

2. **Core Endpoints**
   - Goal progress time-series
   - Task completion trends
   - Emotion trends
   - Habit streak calendar

3. **Pre-computation**
   - Ensure daily_summaries are computed
   - Cache commonly queried aggregates

**Deliverable**: Basic analytics endpoints working

### Phase 5: Range Retros (Week 10)

1. **Range Retro**
   - Migration: `013_retros_range.surql`
   - Model: `internal/features/retros/range.go`
   - Service: aggregate from daily summaries
   - API: `POST /retros/range`, `GET /retros/range/:id`

2. **Insights for Ranges**
   - Trend detection (improving/declining)
   - Pattern identification

**Deliverable**: Week/month/custom range retros

### Phase 6: Habits Advanced (Week 11)

1. **Time-Bound Habits**
   - Before/after/between time window support
   - Compliance evaluation

2. **Avoidance Habits**
   - Success = no violating task

3. **Grace Days**
   - Streak logic with grace allowance

4. **Habit Stacking**
   - Parent/child habit relationships

**Deliverable**: Full habit pattern support

### Phase 7: Goals Advanced (Week 12)

1. **Quantitative Goals**
   - Progress accumulation from impacts
   - Progress percentage calculation

2. **Partial Credit**
   - quantity_delta on impacts
   - "Planned X, did Y" UX

3. **Rollover Tracking**
   - Deadline extension logged to history
   - Surface in retros

**Deliverable**: Full goal feature set

### Phase 8: Advanced Analytics (Week 13-14)

1. **Emotion Analytics**
   - Heatmaps (hour x quadrant)
   - Correlation with categories

2. **Habit Analytics**
   - Success by day-of-week
   - Success by time-of-day

3. **Balance Metrics**
   - Category radar
   - Honesty ratio

4. **Caching Layer**
   - Redis or in-memory cache for analytics
   - Cache invalidation on data changes

**Deliverable**: Rich analytics dashboard

### Phase 9: Polish & Advanced Features (Week 15-16)

1. **Auto-Suggestions**
   - Suggest impacted goals from template/category
   - Suggest habits matching task

2. **Notifications Infrastructure**
   - Retro reminders
   - Streak risk alerts

3. **Performance Optimization**
   - Query optimization
   - Index tuning

4. **Testing & Documentation**
   - Integration tests for complex flows
   - API documentation updates

**Deliverable**: Production-ready feature set

---

## 7. Appendices

### 7.1 Validation Rules Summary

| Field | Rule |
|-------|------|
| `goal.title` | required, 1-500 chars |
| `goal.goal_type` | required, oneof: discrete, quantitative |
| `goal.target_value` | gt 0 if quantitative |
| `goal.unit` | valid_unit if quantitative |
| `goal.status` | oneof: not_started, in_progress, completed, abandoned, postponed, paused |
| `goal.privacy` | required, oneof: public, private, sensitive |
| `habit.pattern_type` | required, oneof: frequency, quantity, time_bound, avoidance |
| `habit.frequency_period` | oneof: day, week, month |
| `habit.days_of_week` | array of 0-6 |
| `habit.grace_days` | 0-7 |
| `impact.polarity` | required, oneof: positive, negative, neutral |
| `impact.magnitude` | -1.0 to 1.0 |
| `impact.confidence` | 0.0 to 1.0 |
| `preferences.timezone` | valid IANA timezone |
| `preferences.daily_retro_time` | HH:MM format |

### 7.2 Directory Structure

```
apps/go_backend/internal/features/
├── auth/
├── categories/
├── emotions/
├── goals/
│   ├── handler.go
│   ├── models.go
│   ├── impact.go
│   ├── repository.go
│   └── service.go
├── habits/
│   ├── handler.go
│   ├── models.go
│   ├── repository.go
│   ├── service.go
│   └── streak.go
├── health/
├── templates/
│   ├── handler.go
│   ├── models.go
│   ├── repository.go
│   └── service.go
├── retros/
│   ├── handler.go
│   ├── daily.go
│   ├── range.go
│   ├── insights.go
│   ├── repository.go
│   └── service.go
├── analytics/
│   ├── handler.go
│   ├── models.go
│   ├── service.go
│   └── queries.go
├── tasks/
└── users/
    └── preferences.go
```

### 7.3 Quick Reference: API Routes

```
# Goals
POST   /api/v1/goals
GET    /api/v1/goals
GET    /api/v1/goals/:id
PUT    /api/v1/goals/:id
DELETE /api/v1/goals/:id
PATCH  /api/v1/goals/:id/status
GET    /api/v1/goals/:id/history
GET    /api/v1/goals/:id/impacts

# Habits
POST   /api/v1/habits
GET    /api/v1/habits
GET    /api/v1/habits/:id
PUT    /api/v1/habits/:id
DELETE /api/v1/habits/:id
PATCH  /api/v1/habits/:id/active
GET    /api/v1/habits/:id/history
GET    /api/v1/habits/:id/streak

# Task Impacts
POST   /api/v1/tasks/:id/impacts
GET    /api/v1/tasks/:id/impacts
PUT    /api/v1/tasks/:id/impacts/:impactId
DELETE /api/v1/tasks/:id/impacts/:impactId

# Templates
POST   /api/v1/templates
GET    /api/v1/templates
GET    /api/v1/templates/:id
PUT    /api/v1/templates/:id
DELETE /api/v1/templates/:id
POST   /api/v1/templates/:id/instantiate
POST   /api/v1/templates/from-task/:taskId

# Retros
PUT    /api/v1/user/preferences
GET    /api/v1/user/preferences
GET    /api/v1/retros/daily/:date
POST   /api/v1/retros/daily/:date/regenerate
PATCH  /api/v1/retros/daily/:id
POST   /api/v1/retros/range
GET    /api/v1/retros/range/:id
PATCH  /api/v1/retros/range/:id

# Analytics
GET    /api/v1/analytics/goals/progress
GET    /api/v1/analytics/goals/impacts
GET    /api/v1/analytics/habits/streaks
GET    /api/v1/analytics/habits/calendar
GET    /api/v1/analytics/categories/time
GET    /api/v1/analytics/categories/balance
GET    /api/v1/analytics/tasks/completion
GET    /api/v1/analytics/emotions/trends
GET    /api/v1/analytics/emotions/heatmap
GET    /api/v1/analytics/summary
```

### 7.4 SurrealDB Query Patterns

```surql
-- Fetch goal with impact stats
SELECT *, 
    (SELECT count() FROM task_goal_impacts WHERE out = $parent.id AND polarity = "positive") as positive_count,
    (SELECT count() FROM task_goal_impacts WHERE out = $parent.id AND polarity = "negative") as negative_count,
    (SELECT math::sum(magnitude) FROM task_goal_impacts WHERE out = $parent.id AND polarity = "positive") as positive_sum,
    (SELECT math::sum(magnitude) FROM task_goal_impacts WHERE out = $parent.id AND polarity = "negative") as negative_sum
FROM goals WHERE id = $goal_id;

-- Habit streak calculation
SELECT habit_id, 
    array::len(
        array::filter(
            (SELECT date FROM daily_summaries 
             WHERE habit_results[$habit_id] = "met"
             ORDER BY date DESC),
            |d| d.date >= $streak_start
        )
    ) as streak_days
FROM habits WHERE id = $habit_id;

-- Time by category for range
SELECT 
    category.id as category_id,
    category.name as category_name,
    math::sum(duration) as total_time
FROM tasks
WHERE created_by = $user_id 
    AND start_date >= $start 
    AND start_date < $end
    AND deleted_at = NONE
GROUP BY category;

-- Emotion heatmap data
SELECT 
    time::hour(start_date) as hour,
    inferred_emotion.quadrant as quadrant,
    count() as count
FROM tasks
WHERE created_by = $user_id 
    AND start_date >= $start 
    AND start_date < $end
    AND inferred_emotion != NONE
GROUP BY time::hour(start_date), inferred_emotion.quadrant;
```

---

## Document Metadata

- **Version**: 1.0
- **Created**: December 2024
- **Last Updated**: December 2024
- **Author**: AI Planning Assistant
- **Target Codebase**: `apps/go_backend` (Gin + SurrealDB)
- **Related Docs**: 
  - `docs/EMOTION_RESEARCH.md` (Yale RULER emotion framework)
  - `docs/CONCEPT.md` (Product concept)
  - `docs/DAILY_JOURNAL_APP_PLAN.md` (Initial planning)
  - `db/schema.surql` (Current schema)
