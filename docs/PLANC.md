# Comprehensive Feature Plan: Goals, Habits, Retrospectives & Analytics

## 1. Data Model & Schema (SurrealDB)

We will extend the existing schemaless architecture with record links.

### 1.1 New Entities

#### **Goals (`goals`)**
Represents both one-time goals and recurring habits.

```surql
DEFINE TABLE goals PERMISSIONS FULL;

-- Fields
DEFINE FIELD title ON goals TYPE string;
DEFINE FIELD description ON goals TYPE string;
DEFINE FIELD type ON goals TYPE string ASSERT $value IN ['discrete', 'measurable', 'habit'];
DEFINE FIELD status ON goals TYPE string ASSERT $value IN ['active', 'completed', 'abandoned', 'paused'];
DEFINE FIELD priority ON goals TYPE int DEFAULT 0;

-- Configuration for targets
DEFINE FIELD target_config ON goals TYPE object;
-- For measurable: { target_value: 1000, unit: 'km', deadline: '2025-12-31' }
-- For habit: { frequency: 'daily', count: 3, unit: 'times', days: ['Mon', 'Wed', 'Fri'] }

-- Hierarchy
DEFINE FIELD parent_goal ON goals TYPE option<record<goals>>; -- For sub-goals or milestones

-- Metadata
DEFINE FIELD created_at ON goals TYPE datetime DEFAULT time::now();
DEFINE FIELD updated_at ON goals TYPE datetime DEFAULT time::now();
DEFINE FIELD created_by ON goals TYPE record<users>;

-- Indexes
DEFINE INDEX idx_goals_user_status ON goals COLUMNS created_by, status;
```

#### **Goal History (`goal_history`)**
Tracks changes to goals for auditing and "honest" retrospectives.

```surql
DEFINE TABLE goal_history PERMISSIONS FULL;

DEFINE FIELD goal ON goal_history TYPE record<goals>;
DEFINE FIELD change_type ON goal_history TYPE string; -- 'status_change', 'deadline_change', 'target_change'
DEFINE FIELD old_value ON goal_history TYPE any;
DEFINE FIELD new_value ON goal_history TYPE any;
DEFINE FIELD reason ON goal_history TYPE string; -- Optional user reflection
DEFINE FIELD created_at ON goal_history TYPE datetime DEFAULT time::now();
```

#### **Task Templates (`task_templates`)**
Blueprints for quick task creation.

```surql
DEFINE TABLE task_templates PERMISSIONS FULL;

DEFINE FIELD title ON task_templates TYPE string;
DEFINE FIELD default_duration ON task_templates TYPE duration; -- e.g. 30m
DEFINE FIELD default_category ON task_templates TYPE option<record<categories>>;
DEFINE FIELD default_priority ON task_templates TYPE int;
DEFINE FIELD default_emotion_id ON task_templates TYPE option<string>; -- e.g. "emotions:E16"
DEFINE FIELD linked_goals ON task_templates TYPE array<record<goals>>;

DEFINE FIELD created_by ON task_templates TYPE record<users>;
```

#### **Retrospectives (`retrospectives`)**
Stores generated and user-edited retrospectives.

```surql
DEFINE TABLE retrospectives PERMISSIONS FULL;

DEFINE FIELD type ON retrospectives TYPE string ASSERT $value IN ['daily', 'weekly', 'monthly', 'custom'];
DEFINE FIELD date_range ON retrospectives TYPE object; -- { start: datetime, end: datetime }
DEFINE FIELD content ON retrospectives TYPE object; -- Structured analysis data
DEFINE FIELD user_notes ON retrospectives TYPE string; -- Free text reflection
DEFINE FIELD created_by ON retrospectives TYPE record<users>;
DEFINE FIELD created_at ON retrospectives TYPE datetime DEFAULT time::now();
```

#### **User Preferences (`user_preferences`)**
Stores user-specific settings like retro time.

```surql
DEFINE TABLE user_preferences PERMISSIONS FULL;

DEFINE FIELD user ON user_preferences TYPE record<users>;
DEFINE FIELD daily_retro_time ON user_preferences TYPE string; -- "21:00"
DEFINE FIELD timezone ON user_preferences TYPE string; -- "America/New_York"
DEFINE INDEX idx_prefs_user ON user_preferences COLUMNS user UNIQUE;
```

### 1.2 Relationships (Edges)

#### **Task Impact on Goals (`task_impacts`)**
Links tasks to goals with impact metadata.

```surql
DEFINE TABLE task_impacts TYPE RELATION IN tasks OUT goals PERMISSIONS FULL;

DEFINE FIELD impact_type ON task_impacts TYPE string ASSERT $value IN ['positive', 'negative', 'neutral'];
DEFINE FIELD magnitude ON task_impacts TYPE int DEFAULT 1; -- 1-5 scale of impact
DEFINE FIELD notes ON task_impacts TYPE string;
```

---

## 2. API Design

### 2.1 Goals & Habits
- `GET /goals` - List goals (filters: status, type).
- `POST /goals` - Create a new goal.
- `GET /goals/:id` - Get goal details + history.
- `PUT /goals/:id` - Update goal (automatically creates `goal_history` entry).
- `POST /goals/:id/impact` - Link a task to this goal (creates `task_impacts` edge).

### 2.2 Task Templates
- `GET /templates` - List templates.
- `POST /templates` - Create template.
- `POST /templates/:id/instantiate` - Create a task from a template.

### 2.3 Retrospectives
- `GET /retros` - List past retros.
- `POST /retros/generate` - Trigger generation for a date range (preview).
- `POST /retros` - Save a confirmed retrospective.
- `PUT /retros/:id` - Update user notes/reflection.

### 2.4 Analytics
- `GET /analytics/dashboard` - High-level metrics.
- `GET /analytics/goals/:id` - Specific goal progress/charts.
- `GET /analytics/emotions` - Heatmaps and mood trends.

---

## 3. Product Features & UX Breakdown

### 3.1 Goals: "The North Star"
*Philosophy: Goals should be inspiring, not intimidating. They are living documents that evolve with the user.*

*   **Visual Progress Tracking**:
    *   **Discrete Goals**: A beautiful progress ring or checklist. When completed, a "celebration" animation (confetti or gentle glow) triggers.
    *   **Measurable Goals**: A "burn-up" chart showing actual progress vs. the "ideal path" to the deadline. If the user falls behind, the "ideal path" gently adjusts to show a new, realistic slope, rather than a steep "impossible" climb.
*   **"Honest" Adjustments**:
    *   When a user moves a deadline, don't just change the date. Prompt: *"Life happens! Do you want to add a note about why?"* This note is saved in history and surfaces in the retrospective as a learning point, not a failure.
*   **Anti-Goals / Setbacks**:
    *   Allow users to log "Setbacks" (e.g., "Skipped gym"). Visually, these don't subtract progress but appear as "hurdles" on the timeline. This reframes them as obstacles overcome rather than negative points.

### 3.2 Habits: "Building Rhythm"
*Philosophy: Consistency > Intensity. Focus on the streak, but allow for human error.*

*   **Streak Visualization**:
    *   **Calendar Heatmap**: A GitHub-style contribution graph for each habit. Green squares for done, grey for missed.
    *   **"Grace Days"**: Allow users to configure "Grace Days" (e.g., 1 per week). If they miss a day, it uses a Grace Day token instead of breaking the streak. The UI shows a "Shield" icon protecting the streak.
*   **Habit Stacking**:
    *   Visual grouping of habits. If a user has "Morning Routine" (Water + Meditate + Stretch), show them as a linked chain. Completing one highlights the next.
*   **Smart Prompts**:
    *   If a user usually logs "Read" at 9 PM but hasn't yet, send a gentle notification: *"Ready to unwind with a book?"* (Not *"You forgot to read"*).

### 3.3 Task Templates: "Frictionless Flow"
*Philosophy: Logging should take 3 seconds, not 30.*

*   **"One-Tap" Widgets**:
    *   Home screen widgets or a "Quick Add" dock in the app.
    *   Example: A "Water" button that instantly logs "Drank Water" (30s duration) + "+1 Hydration Goal".
*   **Context-Aware Suggestions**:
    *   In the morning, suggest "Morning Routine" templates. In the evening, suggest "Reflection" or "Wind Down".
*   **"Magic" Conversion**:
    *   If a user types "Gym" manually 3 times, the app suggests: *"You log 'Gym' often. Want to make it a template?"*

### 3.4 Retrospectives: "The Mirror"
*Philosophy: Insight comes from pattern recognition, not just data dumping.*

*   **The "Daily Wrap"**:
    *   Presented like a "Story" (Instagram/Spotify Wrapped style).
    *   Slide 1: "You crushed 5 tasks today!" (Productivity)
    *   Slide 2: "Your mood was mostly 😌 Calm." (Emotion)
    *   Slide 3: "You missed 'Run', but that's okay. Tomorrow is a new day." (Compassionate accountability)
    *   Slide 4: "One thing to celebrate?" (User input)
*   **Deep Dive Mode (Weekly/Monthly)**:
    *   Side-by-side comparison: "This week vs. Last week".
    *   **Impact Analysis**: "On days you slept >7 hours, your mood was 20% higher." (Correlation insight).
    *   **Category Balance**: "You spent 60% of time on Work and 5% on Health. Want to adjust next week?"

---

## 4. Analytics & Charts: "Visualizing Life"

This section serves as the comprehensive reference for all analytics, metrics, and visualizations in the application.

### 4.1 Core Metrics (The "Pulse")

These metrics are calculated daily/weekly to give users a quick health check.

#### Productivity Metrics

| Metric | Description | Formula / Logic | Data Source |
| :--- | :--- | :--- | :--- |
| **Task Completion Rate** | Percentage of planned tasks completed. | `(completed_tasks / (completed + abandoned + postponed)) * 100` | `tasks` table |
| **Focus Score** | Quality of time spent based on task tags. | `Σ(duration * weight) / total_duration`. Weights: Deep Work=1.5, Shallow=0.5. | `tasks` (tags, duration) |
| **Procrastination Index** | Average delay between intention and action. | `AVG(actual_start - scheduled_start)` for tasks where `actual_start > scheduled_start`. | `tasks` |
| **Velocity** | Tasks completed per day/week. | `COUNT(completed_tasks)` per period. | `tasks` |
| **Interruption Rate** | Frequency of paused/stopped tasks. | `COUNT(task_pauses) / total_work_hours`. | `tasks` (audit log/events) |

#### Emotional Metrics

| Metric | Description | Formula / Logic | Data Source |
| :--- | :--- | :--- | :--- |
| **Average Valence** | General pleasantness of the period. | `AVG(valence)` (-1 to +1). | `task_emotions` |
| **Average Arousal** | General energy level. | `AVG(arousal)` (-1 to +1). | `task_emotions` |
| **Mood Stability** | Volatility of emotions. | `STD_DEV(valence)`. Lower = more stable. | `task_emotions` |
| **Emotional Diversity** | Richness of emotional experience. | Shannon Entropy of unique emotion IDs logged. | `task_emotions` |
| **Resilience Score** | Speed of recovery from negative states. | Avg time to return to `valence > 0` after a `valence < -0.5` event. | `task_emotions` (time series) |
| **Dissonance Score** | Conflict between mixed emotions. | Calculated from vector distance of simultaneous emotions. | `task_emotions` |

#### Habit & Lifestyle Metrics

| Metric | Description | Formula / Logic | Data Source |
| :--- | :--- | :--- | :--- |
| **Streak Robustness** | Consistency weighted by difficulty. | `current_streak * difficulty_multiplier`. | `goals` (habits) |
| **Time Consistency** | Adherence to a routine schedule. | `1 / VARIANCE(time_of_day)` for habit completion. | `goals` (habits) |
| **Sleep Correlation** | Impact of sleep on next-day mood. | Correlation Coeff between `sleep_duration` and `next_day_avg_valence`. | `tasks` (sleep category) |
| **Social Impact** | Impact of social connection on mood. | Avg Valence of tasks tagged "Social" vs. baseline. | `tasks` (social category) |

### 4.2 Advanced Visualizations

#### **A. The "Life Balance" Radar**
*   **Type**: Radar / Spider Chart.
*   **Axes**: Work, Health, Social, Growth, Rest (mapped from Categories).
*   **Data**: Total duration or task count per category.
*   **Goal**: Visualizes skew. Ideal = Balanced polygon.
*   **Action**: "Your 'Rest' axis is low. Schedule some downtime?"

#### **B. The "Emotional Landscape" Heatmap**
*   **Type**: 2D Heatmap Grid.
*   **X-Axis**: Day of Week (Mon-Sun).
*   **Y-Axis**: Time of Day (Hourly buckets).
*   **Color**: Average Valence (Red=-1, Green=+1). Opacity = Intensity/Arousal.
*   **Insight**: Identifies temporal patterns (e.g., "Sunday Scaries" or "Friday Joy").

#### **C. The "Consistency River" (Stream Graph)**
*   **Type**: Stream Graph / Stacked Area.
*   **X-Axis**: Time (Days/Weeks).
*   **Y-Axis**: Duration (Hours).
*   **Segments**: Categories.
*   **Insight**: Shows the "flow" of life allocation. Sudden narrowing of a stream indicates neglect.

#### **D. Goal Impact Scatter Plot**
*   **Type**: Scatter Plot.
*   **X-Axis**: Time.
*   **Y-Axis**: Cumulative Goal Progress.
*   **Points**: Individual Tasks.
    *   **Color**: Green (Positive Impact), Red (Negative/Setback).
    *   **Size**: Magnitude of impact (1-5).
*   **Insight**: Visualizes the journey. Clusters of red show struggle points.

#### **E. Correlation Matrix (The "Why")**
*   **Type**: Correlation Grid / Heatmap.
*   **Variables**: Sleep, Work Hours, Social Hours, Screen Time, Avg Mood, Stress Level.
*   **Insight**: Auto-generated insights like "Work Hours negatively correlates with Mood (-0.6)".

#### **F. The "Mood Meter" Distribution**
*   **Type**: Cartesian Scatter Plot.
*   **X-Axis**: Valence (-1 to +1).
*   **Y-Axis**: Arousal (-1 to +1).
*   **Quadrants**:
    *   Top-Right: Yellow (High Energy, Pleasant) - *Joy, Excited*
    *   Bottom-Right: Green (Low Energy, Pleasant) - *Calm, Serene*
    *   Bottom-Left: Blue (Low Energy, Unpleasant) - *Sad, Bored*
    *   Top-Left: Red (High Energy, Unpleasant) - *Angry, Anxious*
*   **Data**: All emotion logs for the period.
*   **Insight**: "Center of Gravity" for the user's emotional state.

### 4.3 Implementation Requirements

#### Aggregation Pipelines (SurrealDB)
To support these metrics efficiently, we need pre-computed aggregations or efficient live queries.

**Daily Summary View:**
```surql
SELECT 
    time::floor(created_at, 1d) as day,
    math::mean(valence) as avg_valence,
    count() as task_count,
    sum(duration) as total_duration
FROM tasks
GROUP BY day
```

**Category Rollup:**
```surql
SELECT 
    category,
    sum(duration) as total_time
FROM tasks
GROUP BY category
```

#### Caching Strategy
*   **Real-time**: Today's metrics are calculated on-the-fly.
*   **Historical**: Yesterday and older should be cached or materialized in a `daily_stats` table to ensure fast loading of charts (e.g., "Year in Review").

#### Privacy & Ethics
*   **Sentiment Analysis**: If using AI to infer mood from text, data must remain local or be anonymized.
*   **Non-Judgmental UI**: Metrics like "Procrastination Index" should be framed constructively (e.g., "Opportunity for Focus") rather than shaming.

---

## 5. Step-by-Step Implementation Plan

### Phase 1: Foundations (Goals & Templates)
1.  **Schema Migration**: Create `goals`, `task_templates`, `task_impacts` tables.
2.  **Backend - Templates**: Implement `features/templates` (CRUD + Instantiate).
3.  **Backend - Goals**: Implement `features/goals` (CRUD).
4.  **Task Integration**: Update `features/tasks` to support linking to goals (impacts).

### Phase 2: Habits & Tracking
1.  **Habit Logic**: Implement recurrence checking in `features/goals`.
2.  **Metrics Service**: Create `features/analytics` to aggregate data.
3.  **Goal History**: Implement the history tracking middleware/logic in `features/goals`.

### Phase 3: Retrospectives
1.  **Retro Engine**: Create `features/retros` service to query data and generate summaries.
2.  **User Preferences**: Implement `features/users/preferences` for retro time.
3.  **Job Scheduler**: Simple background worker to trigger daily retros.

### Phase 4: Analytics & Polish
1.  **Advanced Queries**: Optimize SurrealDB queries for analytics (aggregations).
2.  **Chart Data Endpoints**: Expose formatted data for frontend charts.
3.  **Refinement**: Edge cases, error handling, performance tuning.
