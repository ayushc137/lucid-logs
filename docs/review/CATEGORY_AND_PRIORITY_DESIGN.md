# Category & Priority: Strategic Design & Usage

This document outlines the mental model, unified flow, and analytical value of **Categories** and **Priorities**. It addresses how to manage them optimally across Goals, Activities, and Tasks without creating friction.

## 1. The Mental Model

To make these fields useful (and not just "admin work"), we treat them as two different dimensions of your life:

### **Category = "Context & Identity" (Where?)**
*   **Question**: "Which area of my life is this serving?"
*   **Purpose**: **Balance**. Categories allow you to see if you are neglecting one area (e.g., Health) in favor of another (e.g., Work).
*   **Examples**: `Work`, `Health`, `Family`, `Learning`, `System/Admin`.
*   **Rule**: A Task/Goal usually belongs to **one** primary context. Even if I run with a friend (Health + Social), the *primary* context is likely Health (or Social, depending on user intent).

### **Priority = "Urgency & Impact" (When & Why?)**
*   **Question**: "How critical is this to my success right now?"
*   **Purpose**: **Focus**. Priority allows you to see if you are working on "High Impact" items or just "Busy Work".
*   **Values**:
    *   🔴 **High**: "Must do / Moves the needle significantly."
    *   🟡 **Medium**: "Should do / Maintenance."
    *   🟢 **Low**: "Nice to do / Admin / Low leverage."

---

## 2. The Golden Flow (Zero-Friction Management)

Users should rarely have to manually set these fields for Tasks. They should **Flow Down**.

### The Hierarchy
1.  **Goals** (The Source)
    *   User defines absolute Category & Priority here.
    *   *Example*: Goal "Launch App" -> **Category: Work**, **Priority: High**.
2.  **Activities** (The Habit/Method)
    *   Linked to Goal. Inherits traits.
    *   *Example*: Activity "Coding Session" (Linked to "Launch App") -> Auto-inherits **Work / High**.
3.  **Tasks** (The Execution unit)
    *   Generated from Activity OR Linked to Goal. Inherits traits.
    *   *Example*: Task "Fix login bug" (Created from "Coding Session") -> Auto-set to **Work / High**.

### Handling Edge Cases (The "Multiple Goals" Scenario)
A task can serve multiple goals.
*   *Scenario*: "Run with Boss" -> Linked to **Goal A (Marathon - Health)** and **Goal B (Promotion - Work)**.
*   **Strategy**:
    *   **Category**: **First-Link Wins**. If you link Health goal first, Task is Health. User can manually override if they feel it was more "Work" than "Health".
    *   **Priority**: **Highest Wins**. If linked to a High priority goal and a Low priority goal, the task is effectively **High Priority**.

---

## 3. Analytical Value (The "Why")

Why keep them? Because they answer the two most important questions in a Retrospective.

### A. The "Balance" Chart (Category)
*   **Visual**: Pie Chart or Stacked Bar.
*   **Insight**: "I spent 60 hours on **Work** this week but only 2 hours on **Health**. I am unbalanced."
*   **Without Category**: You only know *that* you worked, not *where* your energy went.

### B. The "Focus" Matrix (Priority)
*   **Visual**: The Eisenhower Breakdown (or simple Bar Chart).
*   **Insight**: "I completed 50 tasks, but 40 of them were **Low Priority**. I was busy, but not productive."
*   **Metric**: **High-Priority Focus Score** (% of time spent on High Priority tasks).
*   **Without Priority**: You treat "Check Email" (Low) the same as "Finish Presentation" (High).

---

## 4. Proposed Usage Scenarios

### Scenario 1: The "Flow" User (Ideal)
1.  User sets up Goal: "Get Fit" (**Category: Health**, **Priority: High**).
2.  User creates Activity: "Gym" (Linked to "Get Fit").
3.  **Daily Usage**:
    *   User clicks "Gym" activity -> Timer starts.
    *   **System**: Creates Task "Gym". Auto-tags **Category: Health**, **Priority: High**.
    *   **User Action**: Zero. Just clicks start.
4.  **Retro**: Shows "5 hours on Health (High Priority)".

### Scenario 2: The "Ad-Hoc" User (Manual)
1.  User creates solitary task: "Call Mom".
2.  **System**: Default Category (e.g., "Personal"), Default Priority (Medium).
3.  **User Action**: Specific override needed only if defaults are wrong.

### Scenario 3: The conflict
1.  User creates task "Read Book".
2.  Links to Goal "Relaxation" (Category: **Wellness**, Priority: **Low**).
3.  Task becomes **Wellness / Low**.
4.  User *also* links to Goal "Learn Spanish" (Category: **Learning**, Priority: **High**).
5.  **System Logic**:
    *   Keep **Category: Wellness** (Stability - don't change existing fields broadly).
    *   Upgrade **Priority: High** (Urgency principle - if it serves a high goal, it is high task).

---

## 5. Recommendation: Keep, but Automate

**Do not remove.** These are the pillars of "Self-Reflection". Without them, a log is just a list of text.

**Optimization Plan**:
1.  **Strict Inheritance**: Ensure creation from Activity/Goal copies these fields 100% of the time.
2.  **Visual Subtlety**:
    *   Don't make users select from a dropdown every time.
    *   If a Goal is selected, Hide the Category/Priority selectors in the UI (or collapse them) because they are handled. Show them only if "Custom" is needed.
3.  **Retro Integration**: Make the charts simpler.
    *   "Where did my time go?" (Category)
    *   "Did I do important things?" (Priority)

## 6. Revised Implementation Checklist
*   [ ] **Backend**: Service logic to "Inherit Category/Priority" on Task Create/Update when linked to Goal/Activity.
*   [ ] **Backend**: Logic for "Highest Priority Wins" when linking multiple goals.
*   [ ] **Frontend**: Task Modal - If Goal selected, auto-fill Category/Priority.
*   [ ] **Frontend**: Task Modal - visual indication "Inherited from [Goal Name]".

## 7. Update Propagation Rules (Handling Edits)

What happens when a user edits an existing Goal or Activity?

### Rule A: "Forward-Only" Inheritance (The Safe Default)
Changes to a **Goal** or **Activity** (Parent) only affect **Future** tasks. They do **NOT** retroactively change existing tasks.
*   *Scenario*: Goal "Launch App" was **Category: Work**. User changes Goal to **Category: Hobby**.
*   *Result*:
    *   Existing Tasks (Past & Pending): Remain **Work**. (History is preserved).
    *   New Tasks (Created tomorrow): Will be **Hobby**.
*   *Why?*: Preserves historical accuracy. "At the time I did this, it was Work."

### Rule B: Manual Override is King (The "Stickiness" Rule)
If a user manually changes a **Task's** Category/Priority to something different than its Goal, the system must respect that difference forever.
*   *Scenario*: Task linked to "Health Goal" (which is High Priority). User manually sets Task to **Low Priority**.
*   *Result*: Even if the Goal is updated later, the Task stays **Low**. The system does not "re-sync" and overwrite the user's manual choice.

### Rule C: Re-Linking Triggers Update
If a user *changes* the link (e.g., Unlinks Goal A, Links Goal B), the system **re-evaluates** inheritance.
*   *Scenario*: Task linked to Goal A. User switches link to Goal B.
*   *Result*: Task adopts Goal B's Category/Priority (unless user has manually set a specific override).

