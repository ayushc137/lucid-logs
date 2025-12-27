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
- Pinch-to-zoom time scaleI’ll break it into: Let’s turn your broken thoughts into a clean concept for the app 😊 1. Big picture of the app 2. Goal system (designed for retros) 3. Logging & quick capture (minimal typing) 4. Retrospective logic (how app “analyzes”) 5. How tasks, streaks, flomodoro fit in 6. Example daily/weekly flows ⸻ 1. Big picture of the app Concept: An analytical life journal / journey where you: • Set goals & milestones • Log small tasks and moods quickly • Periodically do retros (day/week/month) • The app uses your goals + tasks + emotions to reflect: • What went well • What didn’t • What you learned • How your energy and mood behaved Key philosophy: • Friendly, non-robotic language • Very low friction to log things • Still structured enough to get real insight in retros. ⸻ 2. Goal system – designed for retros (core of your question) We want goals that: • Are meaningful and stable enough to evaluate later • But still flexible and human (you will change your mind) 2.1 Goal hierarchy You already hinted at this: • Epic / Big Goal / Wish • e.g. “Get fit”, “Learn React”, “Launch side project” • Milestones under epics • e.g. “Run 5km non-stop”, “Finish React basics course” • Day/Week Goals (commitments) – concrete actions • e.g. “Run 20 minutes today”, “Study React 45 mins” Conceptual model: • Epic (BigGoal) → has many Milestone • Milestone → appears in retros over weeks/months • DayGoal → appears in daily retro and rolls up to milestone/epic 2.2 Goal fields (minimal but powerful) For each goal/milestone/day goal, keep a simple structure: 1. Title 2. Type: Epic / Milestone / Daily 3. Time scope: • Day, Week, Month, Quarter, Year, or Exact Date 4. Priority: Low / Medium / High (or 1–3) 5. Value / Impact: • A simple 1–5 score for “how meaningful this is to me” 6. Why (short sentence): • “Why does this matter?” – helps in retro 7. Success signal: • “How will I know it’s done?” (e.g. “Run 20 min without stopping”) Optional (for analytics, if you want): • Estimated effort: 1–5 (points) • Category: Health / Work / Relationships / Learning / Fun etc. You can keep the UI very friendly: “What’s the goal?” “Why does it matter?” “When do you want to look back on this?” “Is it small, medium, or big for you?” ⸻ 3. What if goals are not met? Editing without breaking the retro This is the big pain you mentioned: If the user just keeps editing / resetting the goal or changing deadline, retro becomes meaningless. The trick: Goals are “locked per retro period”, not frozen forever. 3.1 Statuses instead of deleting For each goal within a retro period, allow these final statuses: • Done • Not done • In progress • Changed (scope changed) • Dropped / No longer important Each status can ask for one friendly sentence: • Done → “What helped you do this?” • Not done → “What got in the way?” • Changed → “What did you change and why?” • Dropped → “Why did you drop it?” This way: • User can change/edit, but it gets recorded as “Changed” (not invisible). • In retro you can show: • X goals done • Y goals not done • Z goals changed/dropped → these become “negative” or “learning” signals. 3.2 Changing deadlines = “rolled over”, not edited out Rule: once a retro period starts (e.g. the week has begun): • You can’t erase a goal from that period. • If you extend the deadline: • Mark original as “Rolled over” for that week. • Automatically create a new instance for next period. So in retro you see: • “Learn React basics – Rolled over from last week” • The app can highlight: • Number of rollovers per goal • “This goal has been rolled over 3 times” → great insight. ⸻ 4. Meaningful but friendly goal setting To keep it versatile but not robotic, you can use 3 lightweight templates: Template A – “Tiny Day Goal” Fields: • Title (e.g. “10-min walk”) • Scope: Today • Priority: default Medium • Optional: link to Epic (Health) UI copy: “What’s one small thing you want to do today?” “Optional: Which bigger dream is this connected to?” Template B – “Milestone” Fields: • Title • Due: Week/Month/Exact date • Value: 1–5 • Why: one line • Success signal UI copy: “Name the milestone.” “Why is it worth it?” “What does ‘done’ look like?” Template C – “Epic / Wish” Fields: • Title • Time horizon: Month / Quarter / Year • Value • Life area (Health, Work, Fun, Relationships…) UI copy: “What’s a direction you’d love to move in?” “How much does this matter to you right now?” You can let users convert: • Daily goal ↔ Milestone • Milestone ↔ Epic without losing history. ⸻ 5. Logging & quick capture (reduce things to enter) You wrote: Reduce things to enter Journal, description, comments, changes, outcome, retrospective, learning Quantity Mood Description notes thoughts Mood/emotion Link/media Quick log default… 5.1 Two main log types A. Quick Log (default) For fast capture: • What did you do? (short text) • How long / quantity? • Either user picks from presets (25 min, 1h, “1 unit”) • Or automatically from a flomodoro timer • Optional: • Mood (emoji picker) • Link to a goal/task (tap to attach) • Tag / category This Quick Log can represent: • A completed task • A mini event (e.g. call with friend) • A feeling (“Feeling anxious, took a walk”) B. Journal Entry A richer entry with: • Title (optional) • Free text • Mood / emotion • Attach: • Goals related • Media / links • Can be marked as: • Normal log • Retro log (e.g. weekly reflection) 5.2 Auto-derived stuff From quick logs + timers you can auto-compute: • Total time spent per goal / category • Number of tasks completed • Highlighted positive emotion moments • Highlighted negative emotion moments No need for the user to fill “outcomes” manually each time; the app infers from logs. ⸻ 6. Retrospective logic (how the app “thinks”) You want: In retro analyse based on goals from last retro + tasks + overall emotion ~ +ve or -ve or energy… Imagine each retro period (day/week/month) as a “snapshot” with: • Goals snapshot • Goals, their value/priority, status (done/not done/changed/rolled over) • Activity snapshot • Tasks done, time spent, streaks, quick logs • Emotion snapshot • Average mood, highs, lows, “energy” trend 6.1 Auto “What went well” Use rules like: • Goals with status Done, especially high value / high priority • Tasks completed linked to those goals • Periods with positive mood and good focus (flomodoro sessions) The app can generate friendly bullets: • “You completed 3/4 high-value goals.” • “You kept a 4-day streak for ‘Daily walk’.” • “Most positive mood entries happened after ‘coding sessions’.” 6.2 Auto “What didn’t go well” Look at: • Goals with Not done or Rolled over status • Goals changed/dropped frequently • Periods with mostly low mood or “drained” energy • Streaks that broke Bullets like: • “2 important goals were rolled over from last week.” • “You often felt low after 10pm work sessions.” • “The ‘Learn X’ epic has been pushed 3 times.” 6.3 User-defined learning & improvement After the app auto-summarizes, you let user add: 1. Learning – free text: • “What did you learn about yourself this period?” 2. Improvement – 1–3 simple commitments: • “Next week I’ll protect my sleep after 11pm.” • “I’ll choose max 2 big goals per week.” Those then become goals/constraints for the next retro cycle. ⸻ 7. Streaks, flomodoro, goals & value 7.1 Flomodoro sessions Each focus session = one unit of: • Duration (e.g. 25 min) • Linked goal/task • Mood before / after (optional) These feed: • Time spent per goal • A sense of focus quality during the period • Nice charts for retros (but you don’t need to overcomplicate in v1) 7.2 Streaks Track streaks on: • Habit-style daily goals (e.g. “Move body 10 min”) • Journal entries (e.g. “Daily reflection”) • Focus sessions (e.g. “At least 1 flomodoro per day”) Streaks become a positive reinforcement in retros: • “You journaled 5 days in a row.” • “You broke your streak on Thursday – what happened?” (soft nudge, not shaming) 7.3 “Value of goal” and comparing in retro You have the idea of value of goal with priority. Use that to show how well user aligned their effort with what they said matters: • Value alignment score (conceptual, you can design later): • (Time spent on high-value goals) ÷ (Total time) Retro can show: • “You spent 70% of your effort on high-value goals – nice.” • Or: “Most time went to low-value tasks while high-value goals were rolled over.” This gives meaningful negative insight without being judgmental. ⸻ 8. Example flows Daily flow 1. Morning: • Choose 1–3 Day Goals (linked to milestones/epics). 2. During day: • Use Quick Log or flomodoro to track tasks. • Optional: mood pings (“How are you feeling?”). 3. End of day mini retro: • Show: • Day goals: done / not done • Time spent per goal • Mood trend • Ask: • “What went well today?” • “One thing to improve tomorrow?” Weekly retro flow 1. Show week summary: • Goals statuses + rollovers • Tasks and flomodoro stats • Streaks • Mood / energy trend 2. Auto-generate bullets for: • What went well • What didn’t 3. Ask user to add: • Learning (text) • 2–3 improvements 4. Let user choose or adjust next week’s goals based on this. ⸻ If you’d like, next step I can: • Turn this into a simple data model (entities & fields) • Or sketch screen-by-screen flows (Goal creation, Quick Log, Retro screens) so you can start designing or prototyping. Nice, this is a really good design problem to think through. Let me untangle it step by step. ⸻ 1. What I meant by “locked per retro period” I didn’t mean: • “Goals become immutable” • “User must commit to strict weekly sprints” • “Retros can only be done at fixed times” What I actually mean conceptually is: For any retro, you should be able to see what the goal looked like and how it behaved during that period, even if the user edited it later. So: • Goals are editable • But their history is not lost – so retro is still meaningful. Think of it like: “Retro looks at the timeline of that goal”, not the final current version only. ⸻ 2. Are goals immutable? Do users have to delete and recreate? Short answer: No, they should be editable. Longer answer: Use this pattern: 2.1 Two layers: current state + history For each goal, you keep: 1. Current state (what the user sees/edits now) • Title, due date / recurrence, status, etc. 2. History events (internal, used for retros) • created_at • status_changed (active → completed/cancelled) • due_date_changed (old → new) • target_changed (e.g. “2L water” → “3L water”) • maybe renamed (optional) So the user can freely: • Change the due date • Change the target • Change the description • Mark completed / cancelled / paused But you log each important change as an event instead of overwriting and forgetting. This means: • You don’t force delete & recreate. • You can still say in retro: “This goal’s deadline moved twice this month.” ⸻ 3. No fixed sprints: retros = free date ranges You said: Main thing here is the retro are not locked they can be done any sec or can never be done… I dont want the user to lock into a sprint. So design retros like this: • Retro is not a fixed “sprint object”. • Retro is basically: User picks a date range (or you provide presets like “Last 3 days”, “Last 7 days”, “This month so far”), and you analyze everything between those dates. So the model: • Data is continuous (goals, tasks, water logs, React study logs, moods) • Retro is a lens over a time window: • “Show me how I did between X and Y.” That satisfies: • User can do retro: • Today, for last 2 days • Randomly on a Wednesday for last 10 days • Rarely, only when they feel like it • You don’t lock them into “week sprints”, but you can still suggest “Last week” as a quick preset. ⸻ 4. How this works with your examples Let’s plug in your concrete cases. 4.1 Recurring goal: drink 3L water daily Goal: • Type: recurring / habit • Target: 3L per day • Recurrence: daily • Start date: e.g. 1 Dec User logs tasks/check-ins: • Each time they drink water: • Quick task: “Drank 500ml” (quantity) • Linked to goal “Drink 3L water” For any retro date range (say 1–10 Dec): • For each day in the range: • Sum water quantity from tasks linked to this goal • Compare with target (3L) • Compute: • Days succeeded / days tracked • Streaks • Correlate with mood (did higher water correlate with better mood?) If user changes the target: • e.g. from 3L → 2.5L on Dec 7: • Log an event target_changed (3→2.5) at Dec 7 • For days before 7, retro uses old target = 3L • For days after 7, retro uses new target = 2.5L So edits are allowed, but past calculations still use the old values. No locking, just versioned logic. ⸻ 4.2 Time-bound goal: “Learn React by [date]” Goal: • Type: project / milestone • Title: Learn React basics • Due date: 31 Jan • Status: Active User logs study tasks: • “Watched React course – 45 mins” • “Built to-do app – 2h” • Attach those tasks to the React goal. Now: If they reach the goal At some point, they create a task like “Finish basic React course” and mark the goal as Completed. • You store an event: status_change: completed at 20 Jan • In any retro: • If date range covers 20 Jan → show “Goal completed”. • If range is earlier → it will still be “in progress” at that time. If they do not reach the goal User updates due date or status: • 1. They push the deadline • Old due: 31 Jan → New due: 15 Feb • Log: due_date_changed (31 Jan → 15 Feb) at 30 Jan In a retro for January: • You can say: • “React goal due on 31 Jan was not completed; deadline was pushed to 15 Feb.” • Retro is still meaningful: you see the missed original commitment. • 2. They mark it as ‘paused’ or ‘dropped’ • You log that too. • Retro can call this out as: • “You paused/dropped this goal this month.” ⸻ 5. But retro is not locked to that end date Important: none of this requires fixed sprints. Retro flow could be: 1. User hits “Retro”. 2. You ask: “What period do you want to look at?” • Quick options: • Last 3 days • Last 7 days • Last 30 days • Custom dates 3. App then: • Pulls goals that were active in that window • Pulls tasks / water logs / React logs in that window • Pulls goal history events in that window (deadline changes, completions, cancellations) So retro is always: “What did goals + tasks + moods do between date A and B?” Not “This was a fixed week you committed to earlier.” ⸻ 6. Design rules to avoid “meaningless” retros Here’s a clean set of conceptual rules you can implement: 6.1 Goals are editable, but history is append-only • You can edit fields, but important changes are recorded as events. • Retro never loses sight of: • Original due date • How often it was moved • When it was completed / cancelled • When recurrence changed 6.2 Retros are date-range-based, not sprint-based • User can open retro any time. • They choose a period; you analyze that period’s: • Goal states • Task logs • Moods/energy • Streaks 6.3 For recurring goals • Define them as templates: • “Daily water 3L” • “Practice React 30 mins on weekdays” • For each day in a chosen range: • Check whether the requirement was met using tasks. • If recurrence pattern changes mid-way: • Use the pattern relevant to each day based on change events. 6.4 For time-bound goals In a retro for date range [A, B]: • Show goals that were: • Active at any time in [A, B] • Or had due dates inside [A, B] • For each goal: • Was it completed before due date within this period? • Was it completed after due date? • Was it not completed and deadline passed inside this period? • Was its deadline pushed inside this period? This gives you automatic “what went well / didn’t go well” without manual filling. ⸻ 7. How to present it nicely in-app You can keep the UI very simple / human: When user edits a due date: “You moved this from 31 Jan to 15 Feb. That’s okay – life happens. We’ll remember this change for your future reflections.” In retro: • Show friendly cards like: Learn React basics • Due: 31 Jan → moved to 15 Feb • Status: In progress • Time spent this period: 5h • Note: This goal has been rescheduled 2 times. Drink 3L water daily • Success rate this period: 6/10 days • Best streak: 4 days in a row • On low-water days your mood trend was slightly lower. Everything stays flexible: user can change goals whenever. The “non-meaninglessness” comes from keeping the history and using date ranges to analyze. ⸻ If you want, I can next: • Sketch a minimal data model (tables/entities + fields) • Or write pseudo-logic like: “Given [startDate, endDate], compute stats for recurring + time-bound goals.”
- Swipe between days# Daily Journaling App - Comprehensive Planning Document

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
- Pinch-to-zoom time scaleI’ll break it into: Let’s turn your broken thoughts into a clean concept for the app 😊 1. Big picture of the app 2. Goal system (designed for retros) 3. Logging & quick capture (minimal typing) 4. Retrospective logic (how app “analyzes”) 5. How tasks, streaks, flomodoro fit in 6. Example daily/weekly flows ⸻ 1. Big picture of the app Concept: An analytical life journal / journey where you: • Set goals & milestones • Log small tasks and moods quickly • Periodically do retros (day/week/month) • The app uses your goals + tasks + emotions to reflect: • What went well • What didn’t • What you learned • How your energy and mood behaved Key philosophy: • Friendly, non-robotic language • Very low friction to log things • Still structured enough to get real insight in retros. ⸻ 2. Goal system – designed for retros (core of your question) We want goals that: • Are meaningful and stable enough to evaluate later • But still flexible and human (you will change your mind) 2.1 Goal hierarchy You already hinted at this: • Epic / Big Goal / Wish • e.g. “Get fit”, “Learn React”, “Launch side project” • Milestones under epics • e.g. “Run 5km non-stop”, “Finish React basics course” • Day/Week Goals (commitments) – concrete actions • e.g. “Run 20 minutes today”, “Study React 45 mins” Conceptual model: • Epic (BigGoal) → has many Milestone • Milestone → appears in retros over weeks/months • DayGoal → appears in daily retro and rolls up to milestone/epic 2.2 Goal fields (minimal but powerful) For each goal/milestone/day goal, keep a simple structure: 1. Title 2. Type: Epic / Milestone / Daily 3. Time scope: • Day, Week, Month, Quarter, Year, or Exact Date 4. Priority: Low / Medium / High (or 1–3) 5. Value / Impact: • A simple 1–5 score for “how meaningful this is to me” 6. Why (short sentence): • “Why does this matter?” – helps in retro 7. Success signal: • “How will I know it’s done?” (e.g. “Run 20 min without stopping”) Optional (for analytics, if you want): • Estimated effort: 1–5 (points) • Category: Health / Work / Relationships / Learning / Fun etc. You can keep the UI very friendly: “What’s the goal?” “Why does it matter?” “When do you want to look back on this?” “Is it small, medium, or big for you?” ⸻ 3. What if goals are not met? Editing without breaking the retro This is the big pain you mentioned: If the user just keeps editing / resetting the goal or changing deadline, retro becomes meaningless. The trick: Goals are “locked per retro period”, not frozen forever. 3.1 Statuses instead of deleting For each goal within a retro period, allow these final statuses: • Done • Not done • In progress • Changed (scope changed) • Dropped / No longer important Each status can ask for one friendly sentence: • Done → “What helped you do this?” • Not done → “What got in the way?” • Changed → “What did you change and why?” • Dropped → “Why did you drop it?” This way: • User can change/edit, but it gets recorded as “Changed” (not invisible). • In retro you can show: • X goals done • Y goals not done • Z goals changed/dropped → these become “negative” or “learning” signals. 3.2 Changing deadlines = “rolled over”, not edited out Rule: once a retro period starts (e.g. the week has begun): • You can’t erase a goal from that period. • If you extend the deadline: • Mark original as “Rolled over” for that week. • Automatically create a new instance for next period. So in retro you see: • “Learn React basics – Rolled over from last week” • The app can highlight: • Number of rollovers per goal • “This goal has been rolled over 3 times” → great insight. ⸻ 4. Meaningful but friendly goal setting To keep it versatile but not robotic, you can use 3 lightweight templates: Template A – “Tiny Day Goal” Fields: • Title (e.g. “10-min walk”) • Scope: Today • Priority: default Medium • Optional: link to Epic (Health) UI copy: “What’s one small thing you want to do today?” “Optional: Which bigger dream is this connected to?” Template B – “Milestone” Fields: • Title • Due: Week/Month/Exact date • Value: 1–5 • Why: one line • Success signal UI copy: “Name the milestone.” “Why is it worth it?” “What does ‘done’ look like?” Template C – “Epic / Wish” Fields: • Title • Time horizon: Month / Quarter / Year • Value • Life area (Health, Work, Fun, Relationships…) UI copy: “What’s a direction you’d love to move in?” “How much does this matter to you right now?” You can let users convert: • Daily goal ↔ Milestone • Milestone ↔ Epic without losing history. ⸻ 5. Logging & quick capture (reduce things to enter) You wrote: Reduce things to enter Journal, description, comments, changes, outcome, retrospective, learning Quantity Mood Description notes thoughts Mood/emotion Link/media Quick log default… 5.1 Two main log types A. Quick Log (default) For fast capture: • What did you do? (short text) • How long / quantity? • Either user picks from presets (25 min, 1h, “1 unit”) • Or automatically from a flomodoro timer • Optional: • Mood (emoji picker) • Link to a goal/task (tap to attach) • Tag / category This Quick Log can represent: • A completed task • A mini event (e.g. call with friend) • A feeling (“Feeling anxious, took a walk”) B. Journal Entry A richer entry with: • Title (optional) • Free text • Mood / emotion • Attach: • Goals related • Media / links • Can be marked as: • Normal log • Retro log (e.g. weekly reflection) 5.2 Auto-derived stuff From quick logs + timers you can auto-compute: • Total time spent per goal / category • Number of tasks completed • Highlighted positive emotion moments • Highlighted negative emotion moments No need for the user to fill “outcomes” manually each time; the app infers from logs. ⸻ 6. Retrospective logic (how the app “thinks”) You want: In retro analyse based on goals from last retro + tasks + overall emotion ~ +ve or -ve or energy… Imagine each retro period (day/week/month) as a “snapshot” with: • Goals snapshot • Goals, their value/priority, status (done/not done/changed/rolled over) • Activity snapshot • Tasks done, time spent, streaks, quick logs • Emotion snapshot • Average mood, highs, lows, “energy” trend 6.1 Auto “What went well” Use rules like: • Goals with status Done, especially high value / high priority • Tasks completed linked to those goals • Periods with positive mood and good focus (flomodoro sessions) The app can generate friendly bullets: • “You completed 3/4 high-value goals.” • “You kept a 4-day streak for ‘Daily walk’.” • “Most positive mood entries happened after ‘coding sessions’.” 6.2 Auto “What didn’t go well” Look at: • Goals with Not done or Rolled over status • Goals changed/dropped frequently • Periods with mostly low mood or “drained” energy • Streaks that broke Bullets like: • “2 important goals were rolled over from last week.” • “You often felt low after 10pm work sessions.” • “The ‘Learn X’ epic has been pushed 3 times.” 6.3 User-defined learning & improvement After the app auto-summarizes, you let user add: 1. Learning – free text: • “What did you learn about yourself this period?” 2. Improvement – 1–3 simple commitments: • “Next week I’ll protect my sleep after 11pm.” • “I’ll choose max 2 big goals per week.” Those then become goals/constraints for the next retro cycle. ⸻ 7. Streaks, flomodoro, goals & value 7.1 Flomodoro sessions Each focus session = one unit of: • Duration (e.g. 25 min) • Linked goal/task • Mood before / after (optional) These feed: • Time spent per goal • A sense of focus quality during the period • Nice charts for retros (but you don’t need to overcomplicate in v1) 7.2 Streaks Track streaks on: • Habit-style daily goals (e.g. “Move body 10 min”) • Journal entries (e.g. “Daily reflection”) • Focus sessions (e.g. “At least 1 flomodoro per day”) Streaks become a positive reinforcement in retros: • “You journaled 5 days in a row.” • “You broke your streak on Thursday – what happened?” (soft nudge, not shaming) 7.3 “Value of goal” and comparing in retro You have the idea of value of goal with priority. Use that to show how well user aligned their effort with what they said matters: • Value alignment score (conceptual, you can design later): • (Time spent on high-value goals) ÷ (Total time) Retro can show: • “You spent 70% of your effort on high-value goals – nice.” • Or: “Most time went to low-value tasks while high-value goals were rolled over.” This gives meaningful negative insight without being judgmental. ⸻ 8. Example flows Daily flow 1. Morning: • Choose 1–3 Day Goals (linked to milestones/epics). 2. During day: • Use Quick Log or flomodoro to track tasks. • Optional: mood pings (“How are you feeling?”). 3. End of day mini retro: • Show: • Day goals: done / not done • Time spent per goal • Mood trend • Ask: • “What went well today?” • “One thing to improve tomorrow?” Weekly retro flow 1. Show week summary: • Goals statuses + rollovers • Tasks and flomodoro stats • Streaks • Mood / energy trend 2. Auto-generate bullets for: • What went well • What didn’t 3. Ask user to add: • Learning (text) • 2–3 improvements 4. Let user choose or adjust next week’s goals based on this. ⸻ If you’d like, next step I can: • Turn this into a simple data model (entities & fields) • Or sketch screen-by-screen flows (Goal creation, Quick Log, Retro screens) so you can start designing or prototyping. Nice, this is a really good design problem to think through. Let me untangle it step by step. ⸻ 1. What I meant by “locked per retro period” I didn’t mean: • “Goals become immutable” • “User must commit to strict weekly sprints” • “Retros can only be done at fixed times” What I actually mean conceptually is: For any retro, you should be able to see what the goal looked like and how it behaved during that period, even if the user edited it later. So: • Goals are editable • But their history is not lost – so retro is still meaningful. Think of it like: “Retro looks at the timeline of that goal”, not the final current version only. ⸻ 2. Are goals immutable? Do users have to delete and recreate? Short answer: No, they should be editable. Longer answer: Use this pattern: 2.1 Two layers: current state + history For each goal, you keep: 1. Current state (what the user sees/edits now) • Title, due date / recurrence, status, etc. 2. History events (internal, used for retros) • created_at • status_changed (active → completed/cancelled) • due_date_changed (old → new) • target_changed (e.g. “2L water” → “3L water”) • maybe renamed (optional) So the user can freely: • Change the due date • Change the target • Change the description • Mark completed / cancelled / paused But you log each important change as an event instead of overwriting and forgetting. This means: • You don’t force delete & recreate. • You can still say in retro: “This goal’s deadline moved twice this month.” ⸻ 3. No fixed sprints: retros = free date ranges You said: Main thing here is the retro are not locked they can be done any sec or can never be done… I dont want the user to lock into a sprint. So design retros like this: • Retro is not a fixed “sprint object”. • Retro is basically: User picks a date range (or you provide presets like “Last 3 days”, “Last 7 days”, “This month so far”), and you analyze everything between those dates. So the model: • Data is continuous (goals, tasks, water logs, React study logs, moods) • Retro is a lens over a time window: • “Show me how I did between X and Y.” That satisfies: • User can do retro: • Today, for last 2 days • Randomly on a Wednesday for last 10 days • Rarely, only when they feel like it • You don’t lock them into “week sprints”, but you can still suggest “Last week” as a quick preset. ⸻ 4. How this works with your examples Let’s plug in your concrete cases. 4.1 Recurring goal: drink 3L water daily Goal: • Type: recurring / habit • Target: 3L per day • Recurrence: daily • Start date: e.g. 1 Dec User logs tasks/check-ins: • Each time they drink water: • Quick task: “Drank 500ml” (quantity) • Linked to goal “Drink 3L water” For any retro date range (say 1–10 Dec): • For each day in the range: • Sum water quantity from tasks linked to this goal • Compare with target (3L) • Compute: • Days succeeded / days tracked • Streaks • Correlate with mood (did higher water correlate with better mood?) If user changes the target: • e.g. from 3L → 2.5L on Dec 7: • Log an event target_changed (3→2.5) at Dec 7 • For days before 7, retro uses old target = 3L • For days after 7, retro uses new target = 2.5L So edits are allowed, but past calculations still use the old values. No locking, just versioned logic. ⸻ 4.2 Time-bound goal: “Learn React by [date]” Goal: • Type: project / milestone • Title: Learn React basics • Due date: 31 Jan • Status: Active User logs study tasks: • “Watched React course – 45 mins” • “Built to-do app – 2h” • Attach those tasks to the React goal. Now: If they reach the goal At some point, they create a task like “Finish basic React course” and mark the goal as Completed. • You store an event: status_change: completed at 20 Jan • In any retro: • If date range covers 20 Jan → show “Goal completed”. • If range is earlier → it will still be “in progress” at that time. If they do not reach the goal User updates due date or status: • 1. They push the deadline • Old due: 31 Jan → New due: 15 Feb • Log: due_date_changed (31 Jan → 15 Feb) at 30 Jan In a retro for January: • You can say: • “React goal due on 31 Jan was not completed; deadline was pushed to 15 Feb.” • Retro is still meaningful: you see the missed original commitment. • 2. They mark it as ‘paused’ or ‘dropped’ • You log that too. • Retro can call this out as: • “You paused/dropped this goal this month.” ⸻ 5. But retro is not locked to that end date Important: none of this requires fixed sprints. Retro flow could be: 1. User hits “Retro”. 2. You ask: “What period do you want to look at?” • Quick options: • Last 3 days • Last 7 days • Last 30 days • Custom dates 3. App then: • Pulls goals that were active in that window • Pulls tasks / water logs / React logs in that window • Pulls goal history events in that window (deadline changes, completions, cancellations) So retro is always: “What did goals + tasks + moods do between date A and B?” Not “This was a fixed week you committed to earlier.” ⸻ 6. Design rules to avoid “meaningless” retros Here’s a clean set of conceptual rules you can implement: 6.1 Goals are editable, but history is append-only • You can edit fields, but important changes are recorded as events. • Retro never loses sight of: • Original due date • How often it was moved • When it was completed / cancelled • When recurrence changed 6.2 Retros are date-range-based, not sprint-based • User can open retro any time. • They choose a period; you analyze that period’s: • Goal states • Task logs • Moods/energy • Streaks 6.3 For recurring goals • Define them as templates: • “Daily water 3L” • “Practice React 30 mins on weekdays” • For each day in a chosen range: • Check whether the requirement was met using tasks. • If recurrence pattern changes mid-way: • Use the pattern relevant to each day based on change events. 6.4 For time-bound goals In a retro for date range [A, B]: • Show goals that were: • Active at any time in [A, B] • Or had due dates inside [A, B] • For each goal: • Was it completed before due date within this period? • Was it completed after due date? • Was it not completed and deadline passed inside this period? • Was its deadline pushed inside this period? This gives you automatic “what went well / didn’t go well” without manual filling. ⸻ 7. How to present it nicely in-app You can keep the UI very simple / human: When user edits a due date: “You moved this from 31 Jan to 15 Feb. That’s okay – life happens. We’ll remember this change for your future reflections.” In retro: • Show friendly cards like: Learn React basics • Due: 31 Jan → moved to 15 Feb • Status: In progress • Time spent this period: 5h • Note: This goal has been rescheduled 2 times. Drink 3L water daily • Success rate this period: 6/10 days • Best streak: 4 days in a row • On low-water days your mood trend was slightly lower. Everything stays flexible: user can change goals whenever. The “non-meaninglessness” comes from keeping the history and using date ranges to analyze. ⸻ If you want, I can next: • Sketch a minimal data model (tables/entities + fields) • Or write pseudo-logic like: “Given [startDate, endDate], compute stats for recurring + time-bound goals.”
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
