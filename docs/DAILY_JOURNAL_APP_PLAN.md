# Daily Journaling App - Comprehensive Planning Document

## Executive Summary

A cross-platform daily journaling application focused on task tracking, time management, mood logging, and goal visualization. The app emphasizes ease of use, high customizability, and extensibility through a robust plugin system.

**Target Platforms:** Web, iOS, Android  
**Core Philosophy:** Make journaling effortless, insightful, and personally meaningful

---

## 1. Technology Stack

### 1.1 Frontend Stack

#### Web Framework: **Next.js 14+ (App Router)**
- **Why:** Server-side rendering, excellent SEO, API routes, image optimization
- **Features:** React Server Components, Incremental Static Regeneration
- **TypeScript:** Type-safe development

#### Mobile Framework: **Tauri Mobile** (Emerging) or **Capacitor.js**

**Recommended: Capacitor.js**
- **Why:** Production-ready, maintained by Ionic team
- **Pros:**
  - Near-native performance
  - Access to all native APIs
  - Shares codebase with Next.js
  - Large plugin ecosystem
  - PWA support built-in
- **Native Features:** Camera, filesystem, notifications, biometrics, haptics

**Alternative: Tauri Mobile (Beta)**
- Rust-powered, smaller bundle sizes
- Better for desktop + mobile
- Still in beta for mobile (Q2 2025 stable)

**React Native (Traditional Alternative)**
- If you need fully native feel
- Expo Router + Next.js shared logic approach
- Separate mobile codebase but shared business logic

#### UI Component Library
- **shadcn/ui** - Unstyled, accessible components (Radix UI)
- **TailwindCSS** - Utility-first styling
- **Framer Motion** - Smooth animations
- **Recharts** / **Victory Native** - Data visualizations

### 1.2 Backend Stack

#### Rust Framework: **Axum** (Recommended)

**Why Axum:**
- Built by Tokio team (async runtime)
- Type-safe, minimal boilerplate
- Excellent performance
- Tower middleware ecosystem
- Best-in-class error handling
- Modern async/await patterns

**Alternative Rust Frameworks:**
1. **Actix-web** - Mature, battle-tested, slightly more boilerplate
2. **Rocket** - Ergonomic, but moving slower than Axum
3. **Loco.rs** - Rails-like framework for Rust (new, inspired by Rails)
4. **Poem** - OpenAPI-first approach

**Backend Stack Components:**
```rust
// Core Dependencies
axum = "0.7"           // Web framework
tokio = "1.35"         // Async runtime
sqlx = "0.7"           // SQL toolkit
tower = "0.4"          // Middleware
tower-http = "0.5"     // HTTP middleware
serde = "1.0"          // Serialization
```

#### Database
- **PostgreSQL** - Primary database
  - JSONB for flexible task schemas
  - Time series support for tracking
  - Full-text search for notes
  - Row-level security for multi-tenant
  
- **Redis** - Caching layer
  - Session management
  - Real-time features
  - Rate limiting

#### Additional Services
- **MinIO** / **S3** - Media storage (self-hosted option)
- **Typesense** / **Meilisearch** - Fast search (optional)
- **RabbitMQ** / **NATS** - Event streaming for plugins

### 1.3 DevOps & Infrastructure

```dockerfile
# Dockerfile structure
FROM rust:1.75-alpine AS backend-builder
FROM node:20-alpine AS frontend-builder
FROM alpine:latest AS runtime
```

**Container Services:**
- API server (Axum)
- PostgreSQL
- Redis
- MinIO (optional)
- Nginx (reverse proxy)

**Orchestration:**
- Docker Compose for development
- Kubernetes for production (optional)

---

## 2. Design System & UX

### 2.1 Design Language: "Mindful Minimalism"

**Inspiration from existing apps:**
- **Notion** - Flexible block-based content
- **Things 3** - Elegant task management
- **Daylio** - Quick mood logging
- **Reflect** - Beautiful journaling interface
- **Structured** - Timeline view excellence
- **Finch** - Gamification & emotional design

### 2.2 Color Palette

**Primary Theme: Adaptive Serenity**

```css
/* Light Mode */
--primary: 142 76% 36%      /* Sage Green - calm, growth */
--secondary: 221 83% 53%    /* Vivid Blue - focus, trust */
--accent: 45 93% 47%        /* Warm Amber - energy, highlight */
--success: 142 71% 45%      /* Green - achievement */
--warning: 38 92% 50%       /* Orange - attention */
--danger: 0 84% 60%         /* Coral Red - remove/alert */

/* Neutrals */
--background: 0 0% 99%      /* Off-white */
--surface: 0 0% 96%         /* Light gray */
--text-primary: 222 47% 11% /* Deep blue-gray */
--text-secondary: 215 14% 34%

/* Dark Mode */
--primary: 142 50% 45%      /* Lighter sage */
--secondary: 221 83% 60%    /* Brighter blue */
--background: 222 47% 8%    /* Deep blue-black */
--surface: 217 33% 12%      /* Elevated surface */
--text-primary: 210 40% 96% /* Off-white */
```

**Task Block Colors** (Psychology-based):
- Red: High energy tasks (exercise, urgent)
- Orange: Social/creative tasks
- Yellow: Learning/development
- Green: Health/wellness
- Blue: Work/productivity
- Purple: Personal growth/reflection
- Pink: Relationships/emotional
- Gray: Routine/neutral

### 2.3 Typography

```css
/* Font Stack */
--font-primary: 'Inter Variable', system-ui, sans-serif;
--font-mono: 'JetBrains Mono', 'Fira Code', monospace;
--font-display: 'Bricolage Grotesque', 'Inter Variable', sans-serif;

/* Type Scale (1.250 - Major Third) */
--text-xs: 0.64rem;    /* 10.24px */
--text-sm: 0.8rem;     /* 12.8px */
--text-base: 1rem;     /* 16px */
--text-lg: 1.25rem;    /* 20px */
--text-xl: 1.563rem;   /* 25px */
--text-2xl: 1.953rem;  /* 31.25px */
--text-3xl: 2.441rem;  /* 39px */
```

### 2.4 Spacing System (8px base)

```css
--space-1: 0.25rem;  /* 4px */
--space-2: 0.5rem;   /* 8px */
--space-3: 0.75rem;  /* 12px */
--space-4: 1rem;     /* 16px */
--space-6: 1.5rem;   /* 24px */
--space-8: 2rem;     /* 32px */
--space-12: 3rem;    /* 48px */
--space-16: 4rem;    /* 64px */
```

### 2.5 Animation Principles

**Timing Functions:**
```css
--ease-smooth: cubic-bezier(0.4, 0.0, 0.2, 1);
--ease-entrance: cubic-bezier(0.0, 0.0, 0.2, 1);
--ease-exit: cubic-bezier(0.4, 0.0, 1, 1);
--ease-bounce: cubic-bezier(0.68, -0.55, 0.265, 1.55);
```

**Animation Durations:**
- Micro-interactions: 150ms (hover, focus)
- Small transitions: 250ms (expand/collapse)
- Medium transitions: 350ms (page elements)
- Large transitions: 500ms (view changes)

**Key Animations:**
1. **Task Block Creation** - Scale + fade in with spring
2. **Task Completion** - Checkbox ✓ with confetti particles
3. **Timeline Scroll** - Parallax effect on time markers
4. **Mood Selection** - Emoji bounce + color ripple
5. **Goal Progress** - Smooth progress bar with celebration at milestones
6. **Quick Log** - Haptic feedback + scale pulse
7. **View Transitions** - Shared element transitions (FLIP technique)

### 2.6 Mood Board Elements

**Visual References:**
- Soft gradients (not harsh stops)
- Rounded corners (8px-16px radius)
- Subtle shadows (elevation system)
- Neumorphism for interactive elements
- Glass morphism for overlays
- Organic shapes for mood/emotion indicators
- Grid-based layouts with breathing room
- Generous whitespace

**Emotional Design:**
- Positive reinforcement animations
- Gentle error states (no aggressive reds)
- Progress visualization (streaks, charts)
- Achievement celebrations (subtle, non-intrusive)
- Empty states with encouraging messages

---

## 3. Core Features & User Experience

### 3.1 Timeline View (Primary Interface)

**Layout:**
```
┌─────────────────────────────────────┐
│  Today · Monday, Nov 17             │
│  [Week] [Day] [Month]    [+ Quick]  │
├─────────────────────────────────────┤
│ 06:00 ├─────────────────────────   │
│       │ 🌅 Morning Routine (45m)    │
│ 07:00 ├─────────────────────────   │
│       │                             │
│ 08:00 ├─────────────────────────   │
│       │ 💼 Deep Work (2h)           │
│ 10:00 ├─────────────────────────   │
│       │                             │
│ 11:00 ├─────────────────────────   │
│       │ ☕ Break (15m)              │
│ 12:00 ├─────────────────────────   │
│       │ 🍽️ Lunch (30m)             │
│ 13:00 ├─────────────────────────   │
│  ...  │                             │
└─────────────────────────────────────┘
```

**Features:**
- Drag-and-drop task blocks
- Visual time overlap detection
- Color-coded by category
- Tap to expand details
- Long-press for context menu
- Pinch-to-zoom time scale
- Swipe between days

**Week View:**
- 7-day horizontal scroll
- Condensed task blocks
- Heatmap overlay for intensity
- Quick pattern recognition

### 3.2 Task System Architecture

#### Task Schema (PostgreSQL JSONB)

```sql
CREATE TABLE tasks (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  template_id UUID,
  title VARCHAR(255) NOT NULL,
  
  -- Time tracking
  scheduled_start TIMESTAMPTZ,
  scheduled_end TIMESTAMPTZ,
  actual_start TIMESTAMPTZ,
  actual_end TIMESTAMPTZ,
  quick_log_duration INTERVAL, -- Default duration for quick log
  
  -- Quantity tracking
  quantity_value DECIMAL,
  quantity_unit VARCHAR(50), -- 'glasses', 'dollars', 'minutes', etc.
  quantity_direction CHAR(1) CHECK (quantity_direction IN ('+', '-')),
  
  -- Mood/emotion
  mood VARCHAR(50),
  energy_level INT CHECK (energy_level BETWEEN 1 AND 5),
  
  -- Content
  description TEXT,
  notes TEXT,
  
  -- Media & links
  links TEXT[],
  media_urls TEXT[],
  
  -- Organization
  tags TEXT[],
  color VARCHAR(7),
  icon VARCHAR(50),
  
  -- Time tracking methods
  tracking_method VARCHAR(20), -- 'pomodoro', 'flow', 'scheduled', 'stopwatch'
  pomodoro_config JSONB, -- {work: 25, break: 5, cycles: 4}
  
  -- Metadata
  is_template BOOLEAN DEFAULT false,
  is_quick_log_enabled BOOLEAN DEFAULT false,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  
  CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_template FOREIGN KEY (template_id) REFERENCES tasks(id)
);

CREATE INDEX idx_tasks_user_date ON tasks(user_id, scheduled_start);
CREATE INDEX idx_tasks_template ON tasks(user_id, is_template) WHERE is_template = true;
CREATE INDEX idx_tasks_tags ON tasks USING gin(tags);
```

#### Task Template Creation Flow

**Approach: Contextual Creation**

Instead of separate template screen:

1. **Quick Template from Current Task**
   - Any task can become a template
   - "Save as Template" button in task detail
   - Auto-suggests template name based on recurring patterns

2. **Inline Template Picker**
   - Command palette (⌘K / Ctrl+K)
   - Type-to-search templates
   - Recently used templates shown first
   - Smart suggestions based on time/context

3. **Template Editing**
   - Edit template properties inline
   - Changes don't affect existing tasks (unless opted in)
   - Visual diff for template updates

```typescript
// Template quick actions
interface TaskTemplate {
  id: string;
  name: string;
  icon: string;
  color: string;
  defaultDuration?: number;
  quickLogEnabled: boolean;
  quickLogDefaults: {
    quantityValue?: number;
    quantityUnit?: string;
    duration?: number; // seconds
  };
  fields: {
    time: boolean;
    quantity: boolean;
    mood: boolean;
    notes: boolean;
    links: boolean;
  };
}
```

### 3.3 Quick Log System

**Use Case:** User drinks water and wants to log it instantly.

**Flow:**
1. User taps "💧 Water" quick action (home screen widget or app)
2. App shows minimal interface:
   ```
   ┌────────────────────┐
   │  💧 Water          │
   │                    │
   │  [ - ]  2  [ + ]   │
   │  glasses           │
   │                    │
   │  [Log It!] ⚡      │
   └────────────────────┘
   ```
3. Tap quantity +/- or [Log It!]
4. Logged with default 10-second duration
5. Added to timeline at current time
6. Success haptic + visual feedback

**Quick Log Widget Actions:**
- Customizable up to 8 favorites
- Drag to reorder
- Color-coded circles with emoji/icon
- One-tap variants (e.g., "Water ×1" = instant log)

### 3.4 Mood & Emotion Tracking

**Based on Psychological Research:**

**Core Emotions (Ekman + Russell's Circumplex Model):**
- 😄 Joy (high pleasure, high arousal)
- 😌 Calm (high pleasure, low arousal)
- 😐 Neutral (low pleasure, low arousal)
- 😔 Sad (low pleasure, low arousal)
- 😰 Anxious (low pleasure, high arousal)
- 😤 Frustrated (low pleasure, high arousal)
- 😊 Grateful (high pleasure, medium arousal)
- 🤔 Reflective (medium, low arousal)

**Energy Levels:**
- ⚡⚡⚡⚡⚡ Energized
- ⚡⚡⚡⚡ Active
- ⚡⚡⚡ Moderate
- ⚡⚡ Low
- ⚡ Exhausted

**Mood Selection Interface:**
- Circular picker (emotion wheel)
- Quick emoji strip
- Voice input option
- Pattern detection: "You felt 😰 anxious 3 times this week"

### 3.5 Goal System

#### Goal Types

**1. Time-Based Goals**
```typescript
interface TimeGoal {
  type: 'time';
  target: {
    duration: number; // minutes
    frequency: 'daily' | 'weekly' | 'monthly';
  };
  linkedTasks: string[]; // Task template IDs
  visualization: 'progress-bar' | 'ring' | 'calendar-heatmap';
}
// Example: "Sleep 8 hours daily"
```

**2. Quantity Goals**
```typescript
interface QuantityGoal {
  type: 'quantity';
  target: {
    value: number;
    unit: string;
    period: 'day' | 'week' | 'month';
  };
  cumulative: boolean; // Sum up or maintain average
}
// Example: "Drink 8 glasses of water daily"
```

**3. Streak Goals**
```typescript
interface StreakGoal {
  type: 'streak';
  target: {
    minCount: number; // Days to maintain
    allowMissing: number; // Forgiveness days
  };
  currentStreak: number;
  longestStreak: number;
}
// Example: "Meditate 30 days in a row (1 day forgiveness)"
```

**4. Task Completion Goals**
```typescript
interface CompletionGoal {
  type: 'completion';
  target: {
    taskCount: number;
    taskTemplates: string[];
    period: 'day' | 'week' | 'month';
  };
}
// Example: "Complete 5 workout sessions per week"
```

**5. Descriptive/Milestone Goals (Epics)**
```typescript
interface EpicGoal {
  type: 'epic';
  description: string;
  milestones: Milestone[];
  endDate?: Date;
  priority: 1 | 2 | 3;
  subtasks: SubTask[];
}
// Example: "Learn Spanish - Conversational Fluency"
// Milestones: Complete course, 100 conversation sessions, pass test
```

**6. Mood/Wellbeing Goals**
```typescript
interface WellbeingGoal {
  type: 'wellbeing';
  target: {
    mood: string[];
    minOccurrence: number;
    period: 'week' | 'month';
  };
}
// Example: "Feel 😌 calm or 😄 joyful 5 days per week"
```

#### Goal Visualization Examples

**Sleep Goal Dashboard:**
```
┌────────────────────────────────────┐
│ 🛏️ Sleep 8 hours nightly          │
├────────────────────────────────────┤
│  This Week:                        │
│  ▓▓▓▓▓▓▓░░ 7.2 hrs avg            │
│                                    │
│  M  T  W  T  F  S  S               │
│  ●  ●  ◐  ●  ○  ●  ●              │
│  8h 7h 6h 8h 4h 9h 7.5h           │
│                                    │
│  [View Sleep Analysis] →           │
└────────────────────────────────────┘
```

**Streak Visualization:**
```
┌────────────────────────────────────┐
│ 🧘 Meditation Streak               │
├────────────────────────────────────┤
│  Current: 23 days 🔥               │
│  Longest: 45 days                  │
│                                    │
│  Calendar Heatmap:                 │
│  □ □ ■ ■ ■ ■ ■  Week 1             │
│  ■ ■ ■ ⚠ ■ ■ ■  Week 2            │
│  ■ ■ ■ ■ ■ ■ ■  Week 3             │
│  ■ ■                               │
│                                    │
│  ⚠ = Forgiveness day used          │
└────────────────────────────────────┘
```

**Epic Goal Progress:**
```
┌────────────────────────────────────┐
│ 🎯 Learn Spanish                   │
│ Priority: ⭐⭐⭐ High               │
├────────────────────────────────────┤
│  Overall Progress: 34%             │
│  ▓▓▓▓▓▓▓░░░░░░░░░░░░               │
│                                    │
│  ✅ Complete Duolingo Course       │
│  ⏳ 50 Conversation Sessions (12/50)│
│  ⬜ Pass DELE A2 Exam              │
│                                    │
│  Contributing Activities:          │
│  • 📚 Daily lessons: 45 days       │
│  • 💬 Conversations: 12 sessions   │
│  • 📝 Vocabulary: 450 words        │
└────────────────────────────────────┘
```

### 3.6 Retrospective System

**Psychological Framework: Gibbs' Reflective Cycle**

**Retrospective Types:**

1. **Task Retrospective** (Post-task)
   - What went well?
   - What was challenging?
   - What would you do differently?

2. **Daily Retrospective**
   - Gratitude (3 things)
   - Wins (accomplishments)
   - Lessons (learnings)
   - Tomorrow's focus

3. **Weekly Review**
   - Goal progress review
   - Energy patterns
   - Time allocation analysis
   - Mood trends
   - Adjustment plans

4. **Monthly Reflection**
   - Big picture progress
   - Habit formation success
   - Life balance assessment
   - Goal realignment

**Interface:**
- Accessible via floating "✨ Reflect" button
- Can be triggered any time
- Pre-filled with context (e.g., completed tasks, mood data)
- Optional prompts based on psychological frameworks
- Voice-to-text option
- Private by default

**Sample Prompts:**
- "What gave you energy today?"
- "What drained your energy?"
- "What are you grateful for?"
- "What's one thing you learned?"
- "How did you show up for yourself today?"

---

## 4. Backend API Architecture

### 4.1 API Structure (RESTful + WebSocket)

```rust
// Axum route structure
async fn main() {
    let app = Router::new()
        // Health check
        .route("/health", get(health_check))
        
        // Authentication
        .route("/auth/register", post(auth::register))
        .route("/auth/login", post(auth::login))
        .route("/auth/refresh", post(auth::refresh_token))
        .route("/auth/logout", post(auth::logout))
        
        // Tasks
        .route("/tasks", get(tasks::list).post(tasks::create))
        .route("/tasks/:id", get(tasks::get).put(tasks::update).delete(tasks::delete))
        .route("/tasks/:id/complete", post(tasks::complete))
        .route("/tasks/quick-log", post(tasks::quick_log))
        
        // Templates
        .route("/templates", get(templates::list).post(templates::create))
        .route("/templates/:id", get(templates::get).put(templates::update).delete(templates::delete))
        .route("/templates/:id/instantiate", post(templates::instantiate))
        
        // Goals
        .route("/goals", get(goals::list).post(goals::create))
        .route("/goals/:id", get(goals::get).put(goals::update).delete(goals::delete))
        .route("/goals/:id/progress", get(goals::progress))
        .route("/goals/dashboard", get(goals::dashboard))
        
        // Retrospectives
        .route("/retrospectives", get(retros::list).post(retros::create))
        .route("/retrospectives/:id", get(retros::get).put(retros::update))
        
        // Analytics
        .route("/analytics/timeline", get(analytics::timeline))
        .route("/analytics/mood-trends", get(analytics::mood_trends))
        .route("/analytics/time-distribution", get(analytics::time_distribution))
        .route("/analytics/streaks", get(analytics::streaks))
        
        // Tags
        .route("/tags", get(tags::list))
        .route("/tags/:name", get(tags::get_tasks))
        
        // Media
        .route("/media/upload", post(media::upload))
        .route("/media/:id", get(media::download).delete(media::delete))
        
        // Plugins (extensibility)
        .route("/plugins", get(plugins::list))
        .route("/plugins/:id/webhook", post(plugins::webhook))
        .route("/plugins/:id/export", post(plugins::export_data))
        .route("/plugins/:id/import", post(plugins::import_data))
        
        // WebSocket for real-time
        .route("/ws", get(websocket_handler))
        
        // Middleware
        .layer(auth_middleware())
        .layer(cors_layer())
        .layer(tracing_layer())
        .layer(rate_limit_layer());
        
    // Start server
    axum::Server::bind(&"0.0.0.0:8000".parse().unwrap())
        .serve(app.into_make_service())
        .await
        .unwrap();
}
```

### 4.2 Database Schema (PostgreSQL)

```sql
-- Users
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) UNIQUE NOT NULL,
  username VARCHAR(100) UNIQUE NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  timezone VARCHAR(50) DEFAULT 'UTC',
  preferences JSONB DEFAULT '{}',
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Tasks (already defined above)

-- Goals
CREATE TABLE goals (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  title VARCHAR(255) NOT NULL,
  description TEXT,
  goal_type VARCHAR(20) NOT NULL, -- 'time', 'quantity', 'streak', etc.
  target_config JSONB NOT NULL,
  current_progress JSONB DEFAULT '{}',
  visualization_type VARCHAR(50),
  priority INT CHECK (priority BETWEEN 1 AND 3),
  start_date DATE,
  end_date DATE,
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  
  CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id)
);

-- Goal tasks mapping
CREATE TABLE goal_tasks (
  goal_id UUID,
  task_id UUID,
  contributes_to_progress BOOLEAN DEFAULT true,
  PRIMARY KEY (goal_id, task_id),
  CONSTRAINT fk_goal FOREIGN KEY (goal_id) REFERENCES goals(id),
  CONSTRAINT fk_task FOREIGN KEY (task_id) REFERENCES tasks(id)
);

-- Retrospectives
CREATE TABLE retrospectives (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  retro_type VARCHAR(20) NOT NULL, -- 'task', 'daily', 'weekly', 'monthly'
  reference_date DATE,
  reference_task_id UUID,
  content JSONB NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  updated_at TIMESTAMPTZ DEFAULT NOW(),
  
  CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_task FOREIGN KEY (reference_task_id) REFERENCES tasks(id)
);

-- Media
CREATE TABLE media (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL,
  task_id UUID,
  file_path VARCHAR(500) NOT NULL,
  file_type VARCHAR(50),
  file_size BIGINT,
  mime_type VARCHAR(100),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  
  CONSTRAINT fk_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_task FOREIGN KEY (task_id) REFERENCES tasks(id)
);

-- Plugins
CREATE TABLE plugins (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(100) NOT NULL,
  slug VARCHAR(100) UNIQUE NOT NULL,
  version VARCHAR(20),
  author VARCHAR(100),
  description TEXT,
  webhook_url VARCHAR(500),
  capabilities JSONB, -- import, export, webhook, etc.
  is_active BOOLEAN DEFAULT true,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User plugin installations
