# Goal Analytics Implementation Status

## Overview

This document tracks the implementation status of goal progress analytics optimizations as outlined in `GOAL_PROGRESS_ARCHITECTURE.md` and `goal_opt.md`.

## Implementation Status

### ✅ Completed

#### 1. Database Schema (P0)
- **`goal_daily_stats` table**: Pre-computed daily progress snapshots for O(1) lookups
- **`goal_period_snapshots` table**: Week/month analytics snapshots  
- **`streak_history` table**: Track streak changes over time for analytics
- **Enhanced indexes**: Added indexes for faster goal progress lookups
- **Migration file**: `db/migrations/004_goal_analytics.surql`

#### 2. SurrealDB Events (P0)
- **`goal_stats_on_create`**: Auto-updates daily stats when task_goals link is created
- **`goal_stats_on_update`**: Updates stats with delta when link is modified
- **`goal_stats_on_delete`**: Decrements stats when link is removed
- **`streak_change_log`**: Logs streak changes to streak_history table

#### 3. Backend Analytics API (P1)
- **Repository methods**: `GetDailyProgress`, `GetPeriodSnapshots`, `GetStreakHistory`, `GetStreakAnalytics`
- **Service layer**: Added corresponding service methods with authorization checks
- **REST endpoints**:
  - `GET /api/v1/goals/:id/daily-progress` - Daily progress history
  - `GET /api/v1/goals/:id/period-stats` - Period snapshots for trends
  - `GET /api/v1/goals/:id/streak-history` - Streak change events

#### 4. Backfill Support
- **`BackfillDailyStats`**: One-time migration helper to populate historical data
- **`RecalculateDailyStats`**: Manual correction for specific goal/date

#### 5. Frontend Integration (P1)
- **New API types**: `DailyStats`, `PeriodSnapshot`, `StreakEvent`, `DailyProgressResponse`, `StreakAnalytics`
- **New API functions**: `getGoalDailyProgress`, `getGoalPeriodStats`, `getGoalStreakHistory`
- **New components**:
  - `GoalProgressChart.svelte` - Bar chart for daily progress visualization
  - `GoalStreakHistory.svelte` - Timeline of streak changes
- **GoalModal updates**: Added "Analytics" tab that displays progress chart and streak history

#### 6. Seed Data
- **Seed file**: `db/migrations/005_seed_goal_analytics.surql`
- Includes commented sample data and backfill utility script

#### 7. per_period Removal (2026-01-28)
- **Removed `per_period` from Target model** - Recurring goals ALWAYS filter by current period
- **Added `progress_date` query param** to `GET /api/v1/goals` for timeline views
- **List view**: Shows current period progress (default behavior)
- **Timeline view**: Can pass `progress_date` to get progress as of a specific date
- **Updated frontend**: Removed per_period from Goal types and forms

#### 8. Task List Pagination Improvement
- **Reduced page size** from 100 to 25 for faster initial loads
- **Added Load More button** for infinite scroll pattern
- **Accumulated results** in memory for better UX

### 🔲 Pending / Future Work

#### Additional Frontend Enhancements
- [ ] Period trend charts (weekly/monthly aggregations)
- [ ] Best days analysis visualization
- [ ] Predictive analytics (projected completion date)
- [ ] Comparative analytics (goal vs goal, period vs period)

---

## How to Apply the Migration

### 1. Run the Migration

```bash
# Start SurrealDB if not running
task db:start

# Apply the migration
surreal sql -e http://localhost:8000 -u root -p root -ns lucid -db logs < db/migrations/004_goal_analytics.surql
```

### 2. Backfill Historical Data

The migration creates empty tables and events. To populate existing data:

```go
// In your application code or a one-time script:
repo := goals.NewRepository(db)
err := repo.BackfillDailyStats(ctx, userID)
```

Or via SurrealQL:

```surql
-- Backfill for all users
LET $aggregated = (
    SELECT
        out AS goal_id,
        out.created_by AS created_by,
        time::floor(
            IF in.completed_at IS NOT NONE 
            THEN in.completed_at 
            ELSE in.start_date END,
            1d
        ) AS date,
        math::sum(quantity_value) AS daily_value,
        count() AS contribution_count,
        out.target.value AS target_value
    FROM task_goals
    WHERE out.deleted_at IS NONE
    GROUP BY goal_id, date
);

FOR $row IN $aggregated {
    UPSERT goal_daily_stats SET
        goal_id = $row.goal_id,
        date = $row.date,
        created_by = $row.created_by,
        daily_value = $row.daily_value,
        contribution_count = $row.contribution_count,
        target_value = $row.target_value,
        status = IF $row.target_value IS NOT NONE AND $row.daily_value >= $row.target_value 
            THEN "met" 
            ELSE "pending" 
        END,
        updated_at = time::now()
    WHERE goal_id = $row.goal_id AND date = $row.date;
};
```

---

## API Usage Examples

### Get Daily Progress

```bash
# Last 30 days by default
GET /api/v1/goals/goals:abc123/daily-progress

# Custom date range
GET /api/v1/goals/goals:abc123/daily-progress?start_date=2026-01-01T00:00:00Z&end_date=2026-01-31T23:59:59Z
```

**Response:**
```json
{
  "goal_id": "goals:abc123",
  "target_per_period": 3.0,
  "data": [
    {
      "date": "2026-01-27T00:00:00Z",
      "daily_value": 2.5,
      "cumulative_value": 45.0,
      "contribution_count": 3,
      "status": "pending"
    }
  ]
}
```

### Get Period Stats

```bash
# Weekly trends
GET /api/v1/goals/goals:abc123/period-stats?period_type=week

# Monthly trends
GET /api/v1/goals/goals:abc123/period-stats?period_type=month&start_date=2025-01-01T00:00:00Z
```

### Get Streak History

```bash
GET /api/v1/goals/goals:abc123/streak-history?limit=20
```

**Response:**
```json
[
  {
    "goal_id": "goals:abc123",
    "date": "2026-01-27T18:00:00Z",
    "streak_before": 6,
    "streak_after": 7,
    "event": "increment"
  },
  {
    "goal_id": "goals:abc123",
    "date": "2026-01-20T10:00:00Z",
    "streak_before": 12,
    "streak_after": 0,
    "event": "broken"
  }
]
```

---

## Performance Impact

### Before (Query-Time Aggregation)
- Every goal fetch runs subquery to aggregate task_goals
- O(N) complexity where N = number of linked tasks
- No historical analytics possible without expensive recalculation

### After (Pre-Aggregated Tables)
- O(1) lookup for current day's progress via events
- Historical charts via simple SELECT from `goal_daily_stats`
- Streak analytics via `streak_history` table
- Events maintain consistency automatically

---

## Notes on `per_period` Field

As noted in `GOAL_PROGRESS_ARCHITECTURE.md`:

> **`per_period` field is deprecated**: For any goal with recurrence, filtering by the current period is the only sensible behavior.

### Implementation (2026-01-28)

The `per_period` flag is now **ignored** for recurring goals. The progress calculation automatically:
- **Recurring goals (habits)**: Always filter contributions by current period (day/week/month)
- **Non-recurring goals**: Include all contributions from linked tasks

This fix ensures:
1. Daily habits show today's progress, not cumulative all-time values
2. Weekly habits show this week's progress only
3. Monthly habits show this month's progress only
4. Progress percentages no longer exceed 100% unexpectedly

The `per_period` field is kept in the API for backwards compatibility but has no effect on recurring goals.

