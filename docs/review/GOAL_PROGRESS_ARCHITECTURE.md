# Goal and Habit Progress Architecture

## 1. Current Implementation

### Dynamic Calculation
Currently, goal progress is **dynamically calculated on-read**. There is no persistent "progress" field stored on the goal record itself (except for denormalized streak data).

When a goal is fetched (e.g., via `FindByID` or `FindPaginated`), the backend executes a SurrealDB query that aggregates data from linked tasks.

#### The Data Flow
1.  **Tasks** are linked to **Goals** via the `task_goals` edge.
4.  This edge contains metadata:
    *   `quantity_value`: The numeric contribution of the task to the goal (e.g., "0.5" liters of water).
    *   `linked_at`: Timestamp.
5.  **Aggregation Query**:
    The system runs a subquery to sum up these `quantity_value`s.
    *   **Crucial Logic**: The query respects the `track_completed_only` flag on the goal target.
    *   If `track_completed_only` is **false/undefined**: It sums `quantity_value` from **ALL** linked tasks, regardless of their status.
    *   If `track_completed_only` is **true**: It sums `quantity_value` **ONLY** from tasks where `completed = true`.
    ```sql
    SELECT 
        math::sum(quantity_value) AS total,
        count() AS count
    FROM <-task_goals
    WHERE ($parent.target.track_completed_only IS NOT TRUE OR in.completed = true)
    AND ... time filters ...
    ```

### Handling Recurrence (Habits)

> [!NOTE]
> **`per_period` field is `true` by default**: For any goal with recurrence, filtering by the current period is the only sensible behavior. The `per_period` boolean on the `Target` struct is effectively always true for habits and can be considered for removal to simplify the data model.

For recurring goals (Habits), the aggregation is scoped to the **current period**.
*   **Daily Goals**: filters tasks where `time::day(completed_at) == time::day()`.
*   **Weekly Goals**: filters tasks where `time::week(completed_at) == time::week()`.
*   **Monthly Goals**: filters tasks where `time::month(completed_at) == time::month()`.

**Implication**: A daily habit's "progress" automatically resets to 0 at the start of the next day because the query only looks at *today's* tasks.

### Streaks
Unlike raw progress, **Streaks** are stored persistently on the Goal record to avoid expensive recursive queries.
*   Fields: `current_streak`, `longest_streak`, `last_completed_date`.
*   **Update Mechanism**: These are updated explicitly via the `UpdateStreaks` repository method when a task is completed or changed.

---

## 2. Issues with Current Approach

1.  **Read Heavy**: Every time a user views their dashboard, the DB must re-scan and sum all linked tasks. As history grows, this becomes slower.
2.  **No Historical Analytics**: Since progress is ephemeral (calculated for "now"), you cannot easily answer "What was my completion rate last week?" without running complex historical queries.
3.  **Limited "Day" Logic**: The `time::day() == time::day()` logic depends on the server/database timezone, which might differ from the user's timezone.

---

## 3. Optimizations & Future Analytics (SurrealDB Focused)

To utilize SurrealDB to its fullest and enable advanced analytics (streaks over time, historical charts), we propose the following changes.

### A. Pre-Computed Progress via Events (The "Goal Periods" Table)

While SurrealDB 2.0+ offers **Computed Views**, they are best for static aggregations. For time-windowed progress (like "Daily Progress" which technically resets at midnight even if data doesn't change), a **lookup table** managed by **Events** is more robust and performant.

#### 1. The `goal_periods` Table
This table acts as a materialized view for period buckets.
```sql
DEFINE TABLE goal_periods SCHEMAFULL;
DEFINE FIELD goal ON TABLE goal_periods TYPE record(goals);
DEFINE FIELD period_type ON TABLE goal_periods TYPE string; -- 'day', 'week', 'month'
DEFINE FIELD period_date ON TABLE goal_periods TYPE datetime; -- Truncated timestamp (e.g., 2026-01-27T00:00:00Z)
DEFINE FIELD current_value ON TABLE goal_periods TYPE float DEFAULT 0;
DEFINE FIELD target_met ON TABLE goal_periods TYPE bool DEFAULT false;
DEFINE INDEX unique_period ON TABLE goal_periods COLUMNS goal, period_type, period_date UNIQUE;
```

#### 2. Event-Based Updates
Instead of expensive recalculations, we use `DEFINE EVENT` on the `task_goals` edge to increment/decrement the relevant period record.

```sql
-- When a task is linked to a goal
DEFINE EVENT task_linked ON TABLE task_goals WHEN $event = "CREATE" THEN {
    -- 1. Identify valid periods for this goal (Day, Week, Month)
    -- 2. Upsert into goal_periods for those dates
    -- 3. Increment current_value by $after.quantity_value
};

-- When a task is updated (e.g. completed status changes)
DEFINE EVENT task_updated ON TABLE tasks WHEN $event = "UPDATE" AND $before.completed != $after.completed THEN {
    -- 1. Find all goals linked to this task
    -- 2. If goal.track_completed_only is true:
    --    - If now completed: Increment goal_periods value
    --    - If now uncompleted: Decrement goal_periods value
};
```

### B. Live Queries for Real-Time UI

With the `goal_periods` table populated, the frontend no longer needs to run complex aggregation queries.

*   **Dashboard View**:
    ```sql
    LIVE SELECT * FROM goal_periods 
    WHERE goal INSIDE $user_goals 
    AND period_date = time::floor(time::now(), 1d)
    ```
*   **Benefit**: The UI subscribes to this. As soon as a user ticks a task, the Event triggers, updates `goal_periods`, and the Live Query immediately pushes the new percentage to the UI. Zero refresh needed.

### C. Advanced Analytics (Time Series)
Since `goal_periods` essentially stores a history of every day/week's performance, we get analytics for free.
*   **"Show my performance last month"**: `SELECT * FROM goal_periods WHERE period_date > ...`
*   No need to re-scan thousands of tasks or calculate sums on the fly.

## 4. Proposed Refactoring Steps

1.  **Create `goal_logs` / `goal_periods` Table**: Start recording specific period progress.
2.  **Event Triggers**: Add SurrealDB events on `task_goals` to automatically update these log records.
3.  **Migration**: Backfill logs from existing task history.
4.  **Frontend Update**: Switch `goals/+page.svelte` to read from this new optimized table for current status, and use it for drawing history charts.
