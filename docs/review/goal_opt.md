# Goal & Habit Progress Tracking: Implementation & Optimization Guide

> Deep dive into how progress is calculated, tracked, and stored for goals and habits in Lucid Logs. Includes SurrealDB optimization strategies and future analytics roadmap.

---

## Table of Contents

1. [Current Implementation](#current-implementation)
2. [Progress Calculation Flow](#progress-calculation-flow)
3. [Streak Management](#streak-management)
4. [SurrealDB Optimization Opportunities](#surrealdb-optimization-opportunities)
5. [Proposed Schema Enhancements](#proposed-schema-enhancements)
6. [Future Analytics Roadmap](#future-analytics-roadmap)
7. [Implementation Priority Matrix](#implementation-priority-matrix)

---

## Current Implementation

### Goal Nature (Inferred from Structure)

Goals don't have an explicit `type` enum. Nature is inferred from fields:

| Structure | Type | Example |
|-----------|------|---------|
| Has `recurrence` | Habit | "Exercise 3x/week" |
| Has `target` | Measurable Goal | "Run 100km this month" |
| Has `target.operator = "lte"` | Avoidance Goal | "Max 2 coffees/day" |
| Has children via `goal_children` | Grouped Goal | "Q1 Objectives" |
| None of above | Simple Goal | "Learn Spanish" |

### Core Data Structures

```go
// Goal Target (for measurable goals)
type Target struct {
    Value              float64  // Target amount (e.g., 100)
    Operator           string   // "gte" (≥), "lte" (≤), "eq" (=)
    UnitID             *string  // e.g., "units:km"
    PerPeriod          bool     // Reset progress each period?
    TrackCompletedOnly bool     // Only count completed tasks?
}

// Goal Recurrence (makes it a habit)
type Recurrence struct {
    Frequency  int      // Times per period (e.g., 5)
    Period     string   // "day", "week", "month"
    ActiveDays []string // ["mon", "tue", ...] (optional)
    BeforeTime string   // Time window start
    AfterTime  string   // Time window end
    GraceDays  int      // Days allowed to miss without breaking streak
}

// Computed Stats (returned on read)
type GoalStats struct {
    CurrentValue       float64    // Sum of task contributions
    ProgressPercent    float64    // 0-100 (or >100)
    CurrentStreak      int        // Denormalized
    LongestStreak      int        // Denormalized
    LastCompletedDate  *time.Time // Denormalized
    TodayStatus        string     // "pending", "met", "exceeded"
    ChildrenTotal      int        // For grouped goals
    ChildrenCompleted  int        // For grouped goals
    TotalContributions int        // Count of task_goals links
}
```

### Task-Goal Linking (task_goals Edge)

Progress flows from tasks to goals via the `task_goals` relation:

```surql
-- Schema
DEFINE TABLE task_goals TYPE RELATION IN tasks OUT goals;
DEFINE FIELD quantity_value ON task_goals TYPE option<float>;  -- The progress contribution
DEFINE FIELD impact_type ON task_goals TYPE string;            -- "positive", "negative", "neutral"
DEFINE FIELD unit_id ON task_goals TYPE option<string>;
DEFINE FIELD is_milestone ON task_goals TYPE bool DEFAULT false;
DEFINE FIELD source ON task_goals TYPE string DEFAULT "manual";
```

---

## Progress Calculation Flow

### Current Approach: Query-Time Aggregation

Progress is computed when goals are fetched via SurrealDB subqueries:

```surql
SELECT
    *,
    -- Compute filtered stats inline
    (
        SELECT
            math::sum(quantity_value) AS total,
            count() AS count
        FROM <-task_goals
        WHERE
            -- Only completed tasks if configured
            ($parent.target.track_completed_only IS NOT TRUE OR in.completed = true)
            AND (
                -- No period filtering for non-periodic goals
                $parent.target.per_period IS NOT TRUE
                OR $parent.recurrence IS NONE
                OR (
                    -- Filter by current period
                    ($parent.recurrence.period = 'day' AND time::day(in.completed_at) = time::day())
                    OR ($parent.recurrence.period = 'week' AND time::week(in.completed_at) = time::week())
                    OR ($parent.recurrence.period = 'month'
                        AND time::month(in.completed_at) = time::month()
                        AND time::year(in.completed_at) = time::year())
                )
            )
        GROUP ALL
    ).total AS current_value
FROM goals
WHERE created_by = $user AND deleted_at IS NONE
```

### Progress Percent Calculation

```go
func calculateProgress(currentValue, targetValue float64, operator string) (percent float64, status string) {
    if targetValue == 0 {
        return 0, "pending"
    }

    percent = (currentValue / targetValue) * 100

    switch operator {
    case "gte": // Achievement goal (≥ target)
        if currentValue >= targetValue {
            status = "met"
        } else {
            status = "pending"
        }
    case "lte": // Avoidance goal (≤ limit)
        if currentValue <= targetValue {
            status = "met"
        } else {
            status = "exceeded"
        }
    case "eq": // Exact match
        if currentValue == targetValue {
            status = "met"
        } else if currentValue > targetValue {
            status = "exceeded"
        } else {
            status = "pending"
        }
    }

    return percent, status
}
```

### Issues with Current Approach

| Issue | Impact | Severity |
|-------|--------|----------|
| N+1 query pattern | Subquery runs per goal | Medium |
| No historical progress | Can't show progress over time | High |
| Period transitions are abrupt | Midnight resets lose context | Medium |
| No caching | Same calculation repeated on every read | Medium |

---

## Streak Management

### Current Approach: Denormalized on Write

Streaks are materialized when tasks complete, stored directly on the goal:

```go
// On task completion that affects a goal:
func (s *Service) RecordCompletion(goalID string, completionDate time.Time) {
    goal := s.repo.FindByID(goalID)

    // Normalize to date-only
    today := completionDate.Truncate(24 * time.Hour)
    lastDate := goal.LastCompletedDate.Truncate(24 * time.Hour)

    // Calculate expected previous date based on recurrence
    expectedPrev := calculateExpectedPreviousDate(today, goal.Recurrence)

    // Apply grace days
    if goal.Recurrence.GraceDays > 0 {
        expectedPrev = expectedPrev.AddDate(0, 0, -goal.Recurrence.GraceDays)
    }

    newStreak := goal.CurrentStreak
    if today.Equal(lastDate) {
        // Same day completion - no streak change
    } else if lastDate.After(expectedPrev) || lastDate.Equal(expectedPrev) {
        // Within acceptable window - increment streak
        newStreak++
    } else {
        // Gap too large - streak broken
        newStreak = 1
        logEvent("streak_broken", goal.ID)
    }

    // Update longest streak if needed
    longestStreak := goal.LongestStreak
    if newStreak > longestStreak {
        longestStreak = newStreak
    }

    // Persist denormalized fields
    s.repo.UpdateStreaks(goalID, newStreak, longestStreak, today)
}
```

### Streak Calculation Rules

| Recurrence | Streak Unit | Expected Gap |
|------------|-------------|--------------|
| Daily | Days | 1 day × frequency |
| Weekly | Weeks | 7 days × frequency |
| Monthly | Months | 1 month × frequency |

**Grace Days**: Allows missing up to N days without breaking streak.

---

## SurrealDB Optimization Opportunities

### 1. Pre-Aggregated Daily Snapshots (Recommended)

Create a materialized view pattern using SurrealDB events:

```surql
-- New table for pre-aggregated daily goal stats
DEFINE TABLE goal_daily_stats SCHEMAFULL PERMISSIONS FULL;
DEFINE FIELD goal_id ON goal_daily_stats TYPE record<goals>;
DEFINE FIELD date ON goal_daily_stats TYPE datetime;
DEFINE FIELD daily_value ON goal_daily_stats TYPE float DEFAULT 0;
DEFINE FIELD cumulative_value ON goal_daily_stats TYPE float DEFAULT 0;
DEFINE FIELD completion_count ON goal_daily_stats TYPE int DEFAULT 0;
DEFINE FIELD streak_at_date ON goal_daily_stats TYPE int DEFAULT 0;
DEFINE FIELD status ON goal_daily_stats TYPE string;  -- "met", "pending", "missed"
DEFINE FIELD created_by ON goal_daily_stats TYPE record<users>;
DEFINE INDEX idx_goal_daily ON goal_daily_stats COLUMNS goal_id, date UNIQUE;
DEFINE INDEX idx_user_date ON goal_daily_stats COLUMNS created_by, date;

-- Trigger: Auto-update on task_goals changes
DEFINE EVENT goal_progress_update ON TABLE task_goals WHEN $event IN ["CREATE", "UPDATE", "DELETE"] THEN {
    LET $goal = $after.out ?? $before.out;
    LET $task = $after.in ?? $before.in;
    LET $date = time::floor($task.completed_at ?? time::now(), 1d);

    -- Upsert daily stat
    UPSERT goal_daily_stats SET
        goal_id = $goal,
        date = $date,
        created_by = $goal.created_by,
        daily_value += IF $event = "DELETE" THEN -($before.quantity_value ?? 0)
                       ELSE ($after.quantity_value ?? 0) - ($before.quantity_value ?? 0) END,
        completion_count += IF $event = "DELETE" THEN -1 ELSE IF $event = "CREATE" THEN 1 ELSE 0 END
    WHERE goal_id = $goal AND date = $date;
};
```

**Benefits**:
- O(1) lookup for any date's progress
- Enables historical charts and trends
- Supports time-range queries efficiently

### 2. Streak Tracking via Daily Completions

Instead of denormalizing streaks on goals, compute from daily stats:

```surql
-- Calculate current streak from goal_daily_stats
DEFINE FUNCTION fn::calculate_streak($goal_id: record<goals>) {
    LET $today = time::floor(time::now(), 1d);
    LET $goal = (SELECT recurrence, target FROM $goal_id);

    LET $required_per_period = $goal.target.value;
    LET $period = $goal.recurrence.period;

    -- Get consecutive met days (simplified for daily habits)
    LET $streak = (
        SELECT count() as streak
        FROM goal_daily_stats
        WHERE goal_id = $goal_id
          AND status = "met"
          AND date <= $today
        ORDER BY date DESC
        -- Count consecutive days until gap
    );

    RETURN $streak;
};
```

### 3. Efficient Period-Based Queries with Indexes

Add specialized indexes for period filtering:

```surql
-- Index for weekly lookups
DEFINE INDEX idx_task_goals_week ON task_goals
    COLUMNS out, time::week(in.completed_at), time::year(in.completed_at);

-- Index for monthly lookups
DEFINE INDEX idx_task_goals_month ON task_goals
    COLUMNS out, time::month(in.completed_at), time::year(in.completed_at);

-- Composite index for goal progress queries
DEFINE INDEX idx_task_goals_progress ON task_goals
    COLUMNS out, in.completed, quantity_value;
```

### 4. Live Queries for Real-Time Progress

Enable real-time UI updates:

```surql
-- Frontend can subscribe to goal progress changes
LIVE SELECT
    id,
    title,
    (SELECT math::sum(quantity_value) FROM <-task_goals WHERE in.completed = true).total AS progress
FROM goals
WHERE created_by = $user AND status = "active";
```

### 5. Graph-Based Progress Aggregation

Leverage SurrealDB's graph traversal for grouped goals:

```surql
-- Calculate grouped goal progress via children
SELECT
    id,
    title,
    (
        SELECT
            count() AS total,
            count(IF status = "completed" THEN 1 ELSE NONE END) AS completed
        FROM ->goal_children->goals
    ) AS children_stats
FROM goals
WHERE created_by = $user AND deleted_at IS NONE;
```

### 6. Computed Fields (Future SurrealDB Feature)

When available, use computed fields for automatic progress:

```surql
-- FUTURE: Computed field (when SurrealDB supports it)
DEFINE FIELD progress ON goals VALUE {
    LET $total = (SELECT math::sum(quantity_value) FROM <-task_goals WHERE in.completed = true);
    IF target IS NONE THEN NONE
    ELSE ($total.total / target.value) * 100
    END
};
```

---

## Proposed Schema Enhancements

### 1. Goal Period Snapshots (Essential for Analytics)

```surql
-- Snapshot at period boundaries
DEFINE TABLE goal_period_snapshots PERMISSIONS FULL;
DEFINE FIELD goal_id ON goal_period_snapshots TYPE record<goals>;
DEFINE FIELD period_type ON goal_period_snapshots TYPE string;  -- "day", "week", "month"
DEFINE FIELD period_key ON goal_period_snapshots TYPE string;   -- "2026-W04", "2026-01-27"
DEFINE FIELD start_date ON goal_period_snapshots TYPE datetime;
DEFINE FIELD end_date ON goal_period_snapshots TYPE datetime;
DEFINE FIELD target_value ON goal_period_snapshots TYPE float;
DEFINE FIELD achieved_value ON goal_period_snapshots TYPE float;
DEFINE FIELD status ON goal_period_snapshots TYPE string;       -- "met", "missed", "exceeded"
DEFINE FIELD streak_at_close ON goal_period_snapshots TYPE int;
DEFINE FIELD contributions ON goal_period_snapshots TYPE array;  -- [{task_id, value, date}]
DEFINE FIELD created_by ON goal_period_snapshots TYPE record<users>;
DEFINE FIELD created_at ON goal_period_snapshots TYPE datetime DEFAULT time::now();

DEFINE INDEX idx_goal_period ON goal_period_snapshots
    COLUMNS goal_id, period_type, period_key UNIQUE;
DEFINE INDEX idx_user_period ON goal_period_snapshots
    COLUMNS created_by, period_type, start_date;
```

### 2. Streak History Table

```surql
-- Track streak changes over time
DEFINE TABLE streak_history PERMISSIONS FULL;
DEFINE FIELD goal_id ON streak_history TYPE record<goals>;
DEFINE FIELD date ON streak_history TYPE datetime;
DEFINE FIELD streak_before ON streak_history TYPE int;
DEFINE FIELD streak_after ON streak_history TYPE int;
DEFINE FIELD event ON streak_history TYPE string;  -- "increment", "broken", "reset"
DEFINE FIELD triggering_task_id ON streak_history TYPE option<record<tasks>>;
DEFINE FIELD created_by ON streak_history TYPE record<users>;

DEFINE INDEX idx_streak_goal ON streak_history COLUMNS goal_id, date;
DEFINE INDEX idx_user_streaks ON streak_history COLUMNS created_by, date;
```

### 3. Enhanced agg_daily for Goals

Extend existing `agg_daily` table:

```surql
-- Add goal-specific daily aggregates
DEFINE FIELD goal_metrics ON agg_daily TYPE object;
-- Structure:
-- {
--   "goals_met": 3,
--   "goals_missed": 1,
--   "total_contributions": 15,
--   "top_goal_progress": [
--     {"goal_id": "goals:xyz", "value": 5.0}
--   ],
--   "streak_updates": [
--     {"goal_id": "goals:abc", "new_streak": 7}
--   ]
-- }
```

---

## Future Analytics Roadmap

### Phase 1: Daily Progress Visualization

**Implementation**: Use `goal_daily_stats` table

```typescript
// API: GET /api/v1/goals/:id/progress?start=2026-01-01&end=2026-01-31
interface DailyProgressResponse {
    goal_id: string;
    target_per_period: number;
    data: {
        date: string;
        value: number;
        cumulative: number;
        status: 'met' | 'pending' | 'missed';
    }[];
}
```

**Charts**:
- Line chart: Progress over time
- Bar chart: Daily contributions
- Area chart: Cumulative vs target

### Phase 2: Streak Analytics

**Metrics**:
- Current streak length
- Longest streak ever
- Average streak before break
- Streak frequency (how often broken)
- Best day of week for streaks
- Streak recovery time (days to restart after break)

```surql
-- Streak analytics query
SELECT
    goal_id,
    title,
    current_streak,
    longest_streak,
    math::mean(
        SELECT streak_after - streak_before
        FROM streak_history
        WHERE goal_id = $parent.id AND event = "broken"
    ) AS avg_streak_length,
    count(
        SELECT * FROM streak_history
        WHERE goal_id = $parent.id AND event = "broken"
    ) AS total_breaks
FROM goals
WHERE created_by = $user AND recurrence IS NOT NONE;
```

### Phase 3: Time-Range Analysis

**Queries**:

```surql
-- Weekly goal performance
SELECT
    time::format(start_date, "%Y-W%V") AS week,
    count(IF status = "met" THEN 1 ELSE NONE END) AS met,
    count(IF status = "missed" THEN 1 ELSE NONE END) AS missed,
    math::mean(achieved_value / target_value * 100) AS avg_progress
FROM goal_period_snapshots
WHERE created_by = $user
  AND period_type = "week"
  AND start_date >= $range_start
  AND end_date <= $range_end
GROUP BY week
ORDER BY week;

-- Monthly trends
SELECT
    time::format(start_date, "%Y-%m") AS month,
    count() AS total_periods,
    math::sum(IF status = "met" THEN 1 ELSE 0 END) AS met_count,
    math::sum(IF status = "met" THEN 1 ELSE 0 END) / count() * 100 AS success_rate
FROM goal_period_snapshots
WHERE created_by = $user AND goal_id = $goal_id
GROUP BY month
ORDER BY month;
```

### Phase 4: Predictive Analytics

**Features**:
- Projected completion date
- Streak survival probability
- Optimal days/times for goal work
- Risk detection (likely to miss)

```surql
-- Simple projection: days to complete at current pace
LET $goal = (SELECT * FROM goals:xyz);
LET $daily_avg = (
    SELECT math::mean(daily_value)
    FROM goal_daily_stats
    WHERE goal_id = goals:xyz
      AND date >= time::now() - 30d
);
LET $remaining = $goal.target.value - $goal.stats.current_value;
LET $days_to_complete = math::ceil($remaining / $daily_avg);

RETURN {
    current_progress: $goal.stats.progress_percent,
    daily_average: $daily_avg,
    projected_days: $days_to_complete,
    projected_date: time::now() + ($days_to_complete * 1d)
};
```

### Phase 5: Comparative Analytics

**Features**:
- Goal vs goal performance
- This period vs last period
- Personal bests tracking
- Category performance comparison

```typescript
interface ComparativeAnalytics {
    goal_id: string;
    this_week: PeriodStats;
    last_week: PeriodStats;
    change: {
        value_delta: number;
        value_percent_change: number;
        status_improved: boolean;
    };
    personal_best: {
        best_day: { date: string; value: number };
        best_week: { week: string; value: number };
        best_streak: number;
    };
}
```

---

## Implementation Priority Matrix

| Enhancement | Effort | Impact | Priority |
|------------|--------|--------|----------|
| `goal_daily_stats` table + events | Medium | High | **P0** |
| Period snapshots (week/month close) | Medium | High | **P0** |
| Streak history table | Low | Medium | **P1** |
| Time-range API endpoints | Low | High | **P1** |
| Live query support | Low | Medium | **P2** |
| Enhanced indexes | Low | Medium | **P2** |
| Predictive analytics | High | Medium | **P3** |
| Comparative analytics | Medium | Medium | **P3** |

---

## Migration Path

### Step 1: Add New Tables (Non-Breaking)

```surql
-- Run in migration
DEFINE TABLE goal_daily_stats ...
DEFINE TABLE goal_period_snapshots ...
DEFINE TABLE streak_history ...
```

### Step 2: Backfill Historical Data

```go
// One-time migration script
func BackfillGoalDailyStats(ctx context.Context, db *database.DB) error {
    // Query all task_goals with completion dates
    // Group by goal_id + date
    // Insert into goal_daily_stats

    query := `
        SELECT
            out AS goal_id,
            time::floor(in.completed_at, 1d) AS date,
            math::sum(quantity_value) AS daily_value,
            count() AS count
        FROM task_goals
        WHERE in.completed = true
        GROUP BY goal_id, date
    `
    // Insert results...
}
```

### Step 3: Add Event Triggers

```surql
-- Enable automatic updates going forward
DEFINE EVENT goal_progress_update ON TABLE task_goals ...
```

### Step 4: Update API Layer

Add new endpoints while keeping existing ones working:

```go
// New endpoints
GET /api/v1/goals/:id/daily-progress
GET /api/v1/goals/:id/period-stats
GET /api/v1/goals/:id/streak-history
GET /api/v1/analytics/goals/trends
```

### Step 5: Deprecate Query-Time Calculation

Once daily stats are populated, switch to reading from pre-aggregated tables.

---

## Summary

### Current State
- Progress calculated at query time via subqueries
- Streaks denormalized on goal table
- Goal logs track events but no aggregation
- No historical progress data

### Recommended Improvements
1. **Pre-aggregate daily stats** for O(1) lookups and historical data
2. **Use SurrealDB events** for automatic stat maintenance
3. **Add period snapshots** for week/month analytics
4. **Track streak history** for streak analytics
5. **Optimize indexes** for time-based queries

### SurrealDB Features to Leverage
- `DEFINE EVENT` for triggers
- `UPSERT` for atomic updates
- `time::*` functions for date math
- Graph traversal for grouped goals
- `LIVE SELECT` for real-time updates
- Composite indexes for query optimization
