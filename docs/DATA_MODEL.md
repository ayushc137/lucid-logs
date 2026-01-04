# Lucid Logs Data Model

This document provides a comprehensive overview of the Goals, Templates, and Tasks data model in Lucid Logs, including all parameters, relationships, and configuration options.

---

## Table of Contents

1. [Overview](#overview)
2. [Goals](#goals)
3. [Goal Actions](#goal-actions)
4. [Templates](#templates)
5. [Tasks](#tasks)
6. [Task-Goal Links](#task-goal-links)
7. [Relationships Diagram](#relationships-diagram)
8. [Common Workflows](#common-workflows)

---

## Overview

Lucid Logs uses a interconnected data model to help users track their activities, habits, and goals:

```
┌─────────────┐    creates    ┌──────────────┐
│  Template   │──────────────▶│     Task     │
└─────────────┘               └──────────────┘
      │                              │
      │ linked_to                    │ linked_to
      ▼                              ▼
┌─────────────┐◀─────────────┬──────────────┐
│    Goal     │              │  Task-Goal   │
│             │   contains   │    Link      │
│             │──────────────│              │
└─────────────┘              └──────────────┘
      │
      │ contains
      ▼
┌─────────────┐
│ Goal Action │
│  (subtask)  │
└─────────────┘
```

### Key Concepts

| Entity | Purpose | Example |
|--------|---------|---------|
| **Goal** | Define what you want to achieve | "Exercise 5x per week" |
| **Goal Action** | Subtasks/milestones within a goal | "Complete Week 1 of program" |
| **Template** | Reusable task blueprint for quick logging | "Morning Run" template |
| **Task** | A logged activity with time, emotions, reflections | "Ran 5km this morning" |
| **Task-Goal Link** | Connects a task to a goal with impact tracking | Task contributed 5km to running goal |

---

## Goals

Goals represent objectives the user wants to achieve. They can be one-time or recurring (habits).

### Goal Types

| Type | Description | Example |
|------|-------------|---------|
| `discrete` | One-time goal without measurable target | "Organize home office" |              ---> ### needed ? why cant epic hanfle this? 
| `measurable` | Goal with quantifiable target | "Run 100km this month" |              
| `avoidance` | Goal to NOT do something | "No social media before 10am" |                     ---->       needed ? can be handled in mesurabke ?  
| `epic` | Parent goal with child milestones | "Launch SaaS product" |    ---->  can this be better handled ? idk simolifyig thigns maybe ?  feels too complicated , tasks -> goals -> goals, how will fe show this ?? 

### Goal Statuses

| Status | Description |
|--------|-------------|
| `active` | Currently working on this goal |
| `completed` | Goal has been achieved |
| `paused` | Temporarily postponed |
| `abandoned` | Given up on this goal |

### Goal Parameters

#### Core Fields

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `id` | string | auto | Unique identifier | `"goals:abc123"` |
| `title` | string | ✅ | Goal name (1-500 chars) | `"Drink 3L water daily"` |
| `description` | string | | Detailed description (max 2000) | `"Stay hydrated for better focus"` |       
| `why` | string | | Why this matters - for retrospectives (max 1000) | `"Improve energy levels"` |               ---->  remove for now 
| `icon` | string | | Emoji icon (max 50) | `"💧"` |                                                               ---->  rethink this ??? emojie seems to be the best bet  ? maybe add icons option as well some publically available oens that fits our use case and ioensiurce  
| `color` | string | | Hex color code (max 20) | `"#3B82F6"` |                                 ---->  remove , instead link category like tasks
| `goal_type` | string | ✅ | Type of goal | `"measurable"` |  


When `recurrence` is set, the goal becomes a recurring habit. Omit for one-time goals.

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `recurrence.frequency` | int | Times per period (1-365) | `1` (once per day) |
| `recurrence.period` | string | `"day"`, `"week"`, `"month"` | `"day"` | 
| `recurrence.active_days` | string[] | Days when goal is active | `["mon", "tue", "wed"]` |
| `recurrence.before_time` | string | Complete before this time (HH:MM) | `"22:00"` |
| `recurrence.after_time` | string | Complete after this time (HH:MM) | `"06:00"` |
| `recurrence.grace_days` | int | Days allowed to miss without breaking streak (0-7) | `1` |
 
#### Target (for measurable goals)   ---->  needs to be reqworked with less then or equale, more then options and also the units should be uniqueue to goales yet user should be able to set it to any tasks auto linking the goals as well , also the linking mayve unit is not right work idk think this through, main thing to achieve is it should be able to be complex and user should be able to set it in any task auto linking goals to it 
---->  also to think if goal is connected to taks how are categories handled , does the task category change or how this happesn 

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `target.value` | float | Target amount | `3.0` |                                     
| `target.unit` | string | Unit of measurement | `"liters"` |                          
| `target.current_value` | float | Current progress (auto-computed) | `1.5` |
| `target.per_period` | bool | `true` = per period, `false` = total | `true` (3L per day) |

#### Timeline

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `start_date` | ISO datetime | When to start working on goal | `"2025-01-01T00:00:00Z"` |
| `deadline` | ISO datetime | Target completion date | `"2025-12-31T23:59:59Z"` |
| `completion_date` | ISO datetime | When goal was completed (auto) | |

#### Streak Tracking (auto-computed for recurring)

| Field | Type | Description |
|-------|------|-------------|
| `current_streak` | int | Current consecutive completions |
| `longest_streak` | int | Best streak ever achieved |
| `last_completed_date` | datetime | When goal was last completed |
| `grace_days_used` | int | Grace days used in current streak |

#### Organization

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `priority` | int | 1 (low), 2 (medium), 3 (high) | `3` |   ---->  see if we should remove from task as its already here , but then tasks without gials will miss this so maybe updating rasks based on this on linking ? , then need to think if multiple goals linked to tasks 
| `value_score` | int | How meaningful (1-5) | `5` | ---->  not needed , same as priprity 
| `category_id` | string | Link to category | `"categories:health123"` |
| `parent_goal` | string | Parent epic goal ID | `"goals:epic456"` | ---->  can be multiple ? or keep it to one for now ? 
| `life_domain` | string | Life area (max 50) | `"health"` | ---->  handeled by cateogiry not needed 
| `completion_mode` | string | For epics: `"all"` (AND) or `"any"` (OR) | `"all"` | ---->  maybe create ruleset with targets to better handle this ? 
| `is_private` | bool | Hide from shared views | `false` | ---->  remoe it for now all are hidden 

#### Auto-Generated Fields

| Field | Type | Description |
|-------|------|-------------|
| `activity_key` | string | Unique key for template/task auto-linking | ---->  what is this ??? whats the use ? 
| `linked_template` | string | Auto-created template ID |     
| `linked_tasks` | array | Tasks linked to this goal (via task_goals) |
| `child_goals` | array | Child goals for epic goals | 

---

## Goal Actions

Goal Actions are subtasks or milestones within a goal.

### Use Cases

- **Discrete goals**: Steps to complete (e.g., "Design wireframes", "Build prototype")
- **Epic goals**: Major milestones to track
- **Recurring goals**: Template activities that count toward the goal

### Goal Action Parameters   ----> rething based on new target , keep minimal items, tasks do action right ?   ----> whats the use of this ??? i am confused simplify things 

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `id` | string | auto | Unique identifier | `"goal_actions:abc123"` |
| `goal_id` | string | auto | Parent goal | `"goals:xyz789"` |
| `title` | string | ✅ | Action name (1-500) | `"Complete market research"` |  ----> what is this ?  hwo to get this ? 
| `description` | string | | Details (max 2000) | `"Analyze 5 competitors"` |          ----> remove 
| `order` | int | | Display order | `1` |                        
| `quantity_value` | float | | Value for measurable contribution | `5.0` |              
| `quantity_unit` | string | | Unit (max 50) | `"hours"` |
| `completed` | bool | | Is action done? | `false` |
| `completed_at` | datetime | | When completed | |
| `created_at` | datetime | auto | Creation timestamp | |

### Goal Action Operations

- **Create**: `POST /goals/:goalId/actions`
- **List**: `GET /goals/:goalId/actions`
- **Update**: `PUT /goals/:goalId/actions/:actionId`
- **Delete**: `DELETE /goals/:goalId/actions/:actionId`
- **Reorder**: `POST /goals/:goalId/actions/reorder`             ----> what is this ? 

---

## Templates

Templates are reusable blueprints for creating tasks quickly, especially useful for recurring activities.

### Template Sources

| Source | Description |
|--------|-------------|
| Auto-created | Generated when creating a goal with recurrence |
| User-created | Manually created for any recurring task |
| System defaults | Pre-populated templates (`is_default = true`) |        ----> remove this for now, not needed 

### Template Parameters

#### Core Fields

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `id` | string | auto | Unique identifier | `"templates:abc123"` |
| `title` | string | ✅ | Template name (1-500) | `"Morning Run"` |             ----> this should create ttiel for task (auto generated from goial same as goal title )
| `description` | string | | Description (max 2000) | `"Quick morning jog"` |    ----> auto (showing the goal linking )
| `icon` | string | | Emoji icon (max 50) | `"🏃"` |                            ----> same as goal 
| `color` | string | | Hex color (max 20) | `"#EF4444"` |           ----> remove keep category 

#### Task Defaults

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `default_duration` | int | Default task duration in seconds | `1800` (30 min) |          ----> consider on generation time of task based on this 
| `default_priority` | int | Default priority (0-3) | `2` |                                 ----> same discussion done for goal -> task 
| `default_category_id` | string | Default category | `"categories:fitness123"` |              ----> same discussion done for goal -> task  , discuss how to change this how to set this etc 

#### Quick Log Settings

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `is_quick_log` | bool | Show in quick log bar | `true` |
| `quick_log_order` | int | Position in quick log bar | `1` |

#### Quantity Settings       ----> check the task tracking for goials and this can be managed accordingly with right naming 

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `quantity_enabled` | bool | Enable quantity tracking | `true` |
| `quantity_default` | float | Default quantity value | `5.0` |
| `quantity_unit` | string | Quantity unit (max 50) | `"km"` |
| `quantity_step` | float | Increment step for UI | `0.5` |

#### Emotion Settings

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `expected_quadrant` | string | Expected emotion quadrant: `green`, `yellow`, `red`, `blue` | `"yellow"` |
| `default_emotion_id` | string | Default emotion selection | `"emotions:E16"` |     

#### Goal Linking

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `activity_key` | string | Key for auto-linking to goals | `"running"` |        ---->  rethink based on other changes 
| `goal_id` | string | Source goal if auto-created | `"goals:run100km"` |

#### Show Fields (UI control)  ---->  remove this not needed ?

| Field | Type | Description |
|-------|------|-------------|
| `show_fields.journal` | bool | Show journal entry field |
| `show_fields.duration` | bool | Show duration picker |
| `show_fields.quantity` | bool | Show quantity input |
| `show_fields.emotion` | bool | Show emotion selector |
| `show_fields.positives_negatives` | bool | Show reflection items |
| `show_fields.notes` | bool | Show notes field |

#### Metadata

| Field | Type | Description |
|-------|------|-------------|
| `is_default` | bool | System-provided template |   ---->  remove 
| `source_task_id` | string | Task this template was created from |
| `use_count` | int | Number of times used |
| `last_used_at` | datetime | Last usage time |

### Template Operations

- **Create**: `POST /templates`
- **List**: `GET /templates`
- **Get**: `GET /templates/:id`
- **Update**: `PUT /templates/:id`
- **Delete**: `DELETE /templates/:id`
- **Instantiate**: `POST /templates/:id/use` (create task from template)
- **Quick Log List**: `GET /templates/quick-log`

---

## Tasks

Tasks are the core activity logging entity - representing what the user actually did.

### Task Parameters

#### Core Fields

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `id` | string | auto | Unique identifier | `"tasks:abc123"` |
| `title` | string | ✅ | Task name (1-500) | `"Morning standup"` |
| `journal` | string | | Rich text journal entry (max 10000) | `"<p>Discussed blockers...</p>"` |
| `start_date` | ISO datetime | ✅ | When task started | `"2025-12-06T09:00:00Z"` |
| `end_date` | ISO datetime | ✅ | When task ended | `"2025-12-06T09:30:00Z"` |
| `completed` | bool | | Is task complete? | `true` |
| `priority` | int | | Priority level (-100 to 100) | `5` |
| `source` | string | | How task was created | `"manual"` |
| `note` | string | | Additional notes (max 5000) | `"Follow up needed"` |
| `category_id` | string | | Link to category | `"categories:work123"` |

#### Emotion Tracking

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `emotion_id` | string | User's primary emotion from mood grid | `"emotions:E16"` |
| `inferred_emotion` | object | Server-calculated emotion (auto) | See below |

##### Inferred Emotion (auto-computed)

| Field | Type | Description |
|-------|------|-------------|
| `average_energy` | float | Average energy from reflections (-1 to 1) |
| `average_pleasantness` | float | Average pleasantness (-1 to 1) |
| `dominant_quadrant` | string | Most common quadrant |
| `closest_emotion_id` | string | Nearest emotion on grid |

#### Reflections (Positives/Negatives)

```json
{
  "positives": [
    { "text": "Good team collaboration", "emotion_id": "emotions:E16" },
    { "text": "Made progress on project" }
  ],
  "negatives": [
    { "text": "Meeting ran over time", "emotion_id": "emotions:E61" }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `positives[].text` | string | What went well |
| `positives[].emotion_id` | string | Optional emotion tag |
| `negatives[].text` | string | What could be improved |
| `negatives[].emotion_id` | string | Optional emotion tag |

#### Goal Integration     ---->  chanfe if requieed based on discussion 

| Field | Type | Description | Example |
|-------|------|-------------|---------|
| `activity_key` | string | For auto-linking to goals | `"running"` |
| `template_id` | string | Source template if created from one | `"templates:xyz"` |
| `quantity` | object | Quantity tracked for measurable goals | `{ "value": 5.0, "unit": "km" }` |
| `linked_goals` | array | Goals linked to this task (via task_goals) | |

### Task Filter Parameters

| Parameter | Type | Description | Example |
|-----------|------|-------------|---------|
| `search` | string | Full-text search in title, journal, note | `"meeting"` |
| `category_id` | string | Filter by category | `"categories:work"` |
| `no_category` | bool | Filter uncategorized tasks | `true` |
| `status` | string | `"all"`, `"completed"`, `"pending"` | `"pending"` |
| `priority_min` | int | Minimum priority | `5` |  ---->  remove not needed, remove related code 
| `priority_max` | int | Maximum priority | `10` | ---->  remove not needed, remove related code 
| `start_date_from` | datetime | Tasks starting after | `"2025-01-01T00:00:00Z"` |
| `start_date_to` | datetime | Tasks starting before | `"2025-01-31T23:59:59Z"` |
| `sort_field` | string | `start_date`, `priority`, `title`, `created_at` | `"start_date"` |
| `sort_order` | string | `"asc"` or `"desc"` | `"desc"` |

---
 
## Task-Goal Links
  ---->  chjeck if needed any chanfes 
Task-Goal Links connect tasks to goals with impact tracking. This is a many-to-many relationship.

### Link Parameters

| Field | Type | Required | Description | Example |
|-------|------|----------|-------------|---------|
| `id` | string | auto | Edge ID | `"task_goals:abc123"` |
| `task_id` | string | auto | Source task | `"tasks:abc"` |
| `goal_id` | string | ✅ | Target goal | `"goals:xyz"` |
| `impact_type` | string | ✅ | `"positive"`, `"negative"`, `"neutral"` | `"positive"` |
| `impact_magnitude` | int | | Impact strength (1-5) | `3` |
| `quantity_value` | float | | Quantity contributed | `5.0` |
| `quantity_unit` | string | | Quantity unit | `"km"` |
| `notes` | string | | Context (max 1000) | `"Great morning run"` |
| `source` | string | | `"manual"` or `"auto"` | `"auto"` |

### Impact Types

| Type | Description | Example |
|------|-------------|---------|
| `positive` | Task contributes positively to goal | Completed a workout toward exercise goal |
| `negative` | Task negatively impacts goal | Ate junk food (for avoidance goal) |
| `neutral` | Task is related but neither + nor - | Researched goal without making progress |

### Link Operations

- **Link task to goal**: `POST /tasks/:taskId/goals`
- **Batch link**: `POST /tasks/:taskId/goals` with `links` array
- **Get task's goals**: `GET /tasks/:taskId/goals`
- **Get goal's tasks**: `GET /goals/:goalId/tasks`
- **Update link**: `PUT /tasks/:taskId/goals/:goalId`
- **Unlink**: `DELETE /tasks/:taskId/goals/:goalId`

---

## Relationships Diagram

----> why are we not using surrealdb to its fullest for complex  relationship managment , we only have task emotio nand task goal , are rest good with id refrences ? , also change based on dicusssion 

```
┌─────────────────────────────────────────────────────────────────────┐
│                          Categories                                  │
│  ┌──────────────────────────────────────────────────────────────┐  │
│  │ id, name, color                                              │  │
│  └──────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────┘
           │                            │
           │ category_id                │ default_category_id
           ▼                            ▼
┌──────────────────────┐      ┌──────────────────────┐
│        Goals         │      │      Templates       │
│ ──────────────────── │      │ ──────────────────── │
│ id                   │      │ id                   │
│ title                │◀─────│ goal_id              │
│ goal_type            │      │ activity_key ─────┐  │
│ recurrence           │      │ title              │ │
│ target               │      │ default_duration   │ │
│ activity_key ────────┼──────│ quantity_*         │ │
│ linked_template ─────┼─────▶│ is_quick_log      │ │
│ status               │      │ show_fields        │ │
│ streaks              │      └──────────────────────┘
└──────────────────────┘                │
           │                            │ instantiate
           │ goal_id                    ▼
           ▼                  ┌──────────────────────┐
┌──────────────────────┐      │        Tasks         │
│    Goal Actions      │      │ ──────────────────── │
│ ──────────────────── │      │ id                   │
│ id                   │      │ title                │
│ title                │      │ start_date, end_date │
│ order                │      │ journal              │
│ completed            │      │ emotion_id           │
│ quantity_*           │      │ positives, negatives │
└──────────────────────┘      │ activity_key ────────┼──┐
                              │ template_id ─────────┼──┼─▶ Templates
                              │ quantity             │  │
                              └──────────────────────┘  │
                                         │              │
                                         │              │ Auto-match via
                                         │              │ activity_key
                                         ▼              ▼
                              ┌──────────────────────────────────┐
                              │         Task-Goal Links          │
                              │ (task_goals relation table)      │
                              │ ────────────────────────────────│
                              │ task_id ──────▶ Tasks            │
                              │ goal_id ──────▶ Goals            │
                              │ impact_type                      │
                              │ impact_magnitude                 │
                              │ quantity_value, quantity_unit    │
                              │ source: "manual" | "auto"        │
                              └──────────────────────────────────┘
```

---

## Common Workflows

### 1. Setting Up a Daily Habit

```
1. Create Goal:
   POST /goals
   {
     "title": "Exercise 30 minutes daily",
     "goal_type": "measurable",
     "recurrence": { "frequency": 1, "period": "day" },
     "target": { "value": 30, "unit": "minutes", "per_period": true }
   }

2. System auto-creates Template with matching activity_key

3. User logs via Quick Log:
   POST /templates/:templateId/use
   { "start_date": "2025-01-15T07:00:00Z", "quantity": 35 }

4. System creates Task and auto-links to Goal via activity_key
```

### 2. Epic Goal with Milestones

```
1. Create Epic Goal:
   POST /goals
   {
     "title": "Launch Mobile App",
     "goal_type": "epic",
     "completion_mode": "all"
   }

2. Add Actions (milestones):
   POST /goals/:id/actions
   [
     { "title": "Design UI mockups", "order": 1 },
     { "title": "Build MVP", "order": 2 },
     { "title": "Beta testing", "order": 3 },
     { "title": "App Store submission", "order": 4 }
   ]

3. Complete actions as work progresses:
   PUT /goals/:id/actions/:actionId
   { "completed": true }

4. When all actions complete, goal auto-completes
```

### 3. Linking Tasks to Goals Manually

```
1. Create a task:
   POST /tasks
   { "title": "Weekly planning session", ... }

2. Link to goal:
   POST /tasks/:taskId/goals
   {
     "goal_id": "goals:productivity",
     "impact_type": "positive",
     "impact_magnitude": 3
   }
```

### 4. Avoidance Goal Tracking

```
1. Create Avoidance Goal:
   POST /goals
   {
     "title": "No junk food",
     "goal_type": "avoidance",
     "recurrence": { "frequency": 1, "period": "day" }
   }

2. If user slips, log with negative impact:
   POST /tasks
   { "title": "Ate pizza", ... }
   
   POST /tasks/:taskId/goals
   { "goal_id": "goals:no-junk", "impact_type": "negative" }
```

---

## API Quick Reference

### Goals
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/goals` | List goals (with filters) |
| POST | `/goals` | Create goal |
| GET | `/goals/:id` | Get goal by ID |
| PUT | `/goals/:id` | Update goal |
| DELETE | `/goals/:id` | Delete goal |
| GET | `/goals/today` | Get today's recurring goals |

### Goal Actions
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/goals/:id/actions` | List actions for goal |
| POST | `/goals/:id/actions` | Create action |
| PUT | `/goals/:id/actions/:actionId` | Update action |
| DELETE | `/goals/:id/actions/:actionId` | Delete action |
| POST | `/goals/:id/actions/reorder` | Reorder actions |

### Templates
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/templates` | List templates |
| POST | `/templates` | Create template |
| GET | `/templates/:id` | Get template |
| PUT | `/templates/:id` | Update template |
| DELETE | `/templates/:id` | Delete template |
| POST | `/templates/:id/use` | Create task from template |
| GET | `/templates/quick-log` | Get quick log templates |

### Tasks
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/tasks` | List tasks (with filters) |
| POST | `/tasks` | Create task |
| GET | `/tasks/:id` | Get task |
| PUT | `/tasks/:id` | Update task |
| DELETE | `/tasks/:id` | Delete task |
| GET | `/tasks/last-end-time` | Get last task's end time |

### Task-Goal Links
| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/tasks/:id/goals` | Get goals for task |
| POST | `/tasks/:id/goals` | Link task to goal(s) |
| PUT | `/tasks/:id/goals/:goalId` | Update link |
| DELETE | `/tasks/:id/goals/:goalId` | Remove link |
| GET | `/goals/:id/tasks` | Get tasks for goal |

---

## Emotion Reference

Emotions are organized in a 10x10 grid based on energy (vertical) and pleasantness (horizontal).

| Quadrant | Energy | Pleasantness | Example Emotions |
|----------|--------|--------------|------------------|
| Yellow | High | Pleasant | Happy, Excited, Proud, Motivated |
| Green | Low | Pleasant | Calm, Content, Relaxed, Peaceful |
| Red | High | Unpleasant | Anxious, Stressed, Frustrated, Angry |
| Blue | Low | Unpleasant | Sad, Tired, Bored, Disappointed |

Emotion IDs: `emotions:E01` to `emotions:E100`

---

*Last updated: January 2026*
