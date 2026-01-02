# Lucid Logs Design Language

## Overview

This document defines the design language and component patterns for Lucid Logs. 
All UI components should follow these guidelines for consistency.

---

## Core Principles

1. **Clarity First** - Every element should have clear purpose
2. **Consistent Spacing** - Use 4px grid system (gap-1, gap-2, gap-3, gap-4...)
3. **Visual Hierarchy** - Use size and weight to establish importance
4. **Accessible Colors** - All text must have sufficient contrast
5. **Responsive by Default** - Components should work on all screen sizes

---

## Color System

### Semantic Colors (DaisyUI)
- `primary` - Main actions, active states
- `secondary` - Supporting actions
- `accent` - Highlights, goals
- `info` - Information, tasks
- `success` - Positive states, mood
- `warning` - Caution, streaks
- `error` - Destructive actions, errors

### Opacity Patterns
- Full opacity - Primary content
- `opacity-60` - Secondary labels, subtext
- `opacity-50` - Tertiary labels, hints
- `opacity-40` - Disabled/inactive

### Background Patterns
- Component backgrounds: `bg-{color}/10` (10% opacity of semantic color)
- Icon containers: `bg-{color}` with `text-{color}-content`
- Card backgrounds: `bg-base-100`
- Overlays: `bg-base-content/20` with `backdrop-blur-sm`

---

## Typography

### Headings
- **Page Title**: `text-3xl font-bold` or `text-2xl font-extrabold`
- **Section Title**: `text-lg font-bold` or `font-bold` (base size)
- **Card Title**: `font-semibold text-sm`
- **Label**: `text-xs font-medium uppercase opacity-50`

### Body Text
- **Primary**: Default size and weight
- **Secondary**: `text-sm`
- **Caption**: `text-xs` or `text-[10px]`
- **Muted**: Add `opacity-50` or `opacity-60`

---

## Spacing

### Standard Gaps
- `gap-1` (4px) - Tight inline elements
- `gap-2` (8px) - Related items
- `gap-3` (12px) - Component internals
- `gap-4` (16px) - Section separation
- `gap-6` (24px) - Major sections

### Padding
- **Cards**: `p-4` standard, `p-5` for larger cards
- **Buttons**: Default button padding, `btn-sm` for compact
- **Modal Content**: `px-6 py-4` for header/footer, `p-6` for body
- **Input**: Standard form control padding

---

## Component Patterns

### 1. Icon Container (`IconBox`)
Square containers for icons with semantic coloring.

```svelte
<IconBox size="md" variant="subtle" color="primary">
    <Icon class="w-5 h-5" />
</IconBox>

Size Options:
- sm: w-9 h-9 rounded-lg
- md: w-10 h-10 rounded-xl  (default)
- lg: w-16 h-16 rounded-full
- xl: w-20 h-20 rounded-full

Variant Options:
- solid: bg-{color} text-{color}-content
- subtle: bg-{color}/10 text-{color}
```

### 2. Stat Card (`StatCard`)
Display key metrics with icon and value.

```svelte
<StatCard value={7} label="Streak" unit="d" color="warning">
    {#snippet icon()}
        <Flame class="w-5 h-5" />
    {/snippet}
</StatCard>
```

### 3. Section Header (`SectionHeader`)
Consistent header pattern for sections.

```svelte
<SectionHeader title="Today's Timeline" subtitle="5 tasks scheduled" color="info">
    {#snippet icon()}
        <Calendar class="w-5 h-5" />
    {/snippet}
    {#snippet actions()}
        <button class="btn btn-sm">Action</button>
    {/snippet}
</SectionHeader>
```

### 4. Cards (`Card`)
Container components with consistent styling.

```svelte
<Card variant="bordered" shadow={true}>
    Content here
</Card>

Variants:
- default: Standard card
- bordered: With border-base-200
- compact: Smaller padding (p-3)
```

### 5. Buttons

```
Primary Action:
- btn btn-primary gap-2 shadow-lg shadow-primary/20

Secondary Action:
- btn btn-ghost

Danger:
- btn btn-error btn-sm

Icon Only:
- btn btn-ghost btn-sm btn-square

With badge:
- Add: <span class="badge badge-sm badge-secondary">Label</span>
```

### 6. Empty State (`EmptyState`)
Centered content when no data exists.

```svelte
<EmptyState
    title="No goals yet"
    description="Set goals to track your progress"
    buttonLabel="Create Your First Goal"
    onButtonClick={handleAdd}
>
    {#snippet icon()}
        <Target class="w-10 h-10 text-primary" />
    {/snippet}
</EmptyState>
```

### 7. Error State (`ErrorAlert`)
Alert for error conditions with retry capability.

```svelte
<ErrorAlert
    message="Failed to load data. Please try again."
    onRetry={() => query.refetch()}
/>
```

### 8. Loading State (`LoadingCard`)
Centered spinner with message.

```svelte
<LoadingCard message="Loading tasks..." />
```

---

## Modal Patterns

### 9. Modal (`Modal`)
Structured modal with header, content, and footer.

```svelte
<Modal
    bind:open={modalOpen}
    size="lg"
    title="Edit Task"
    subtitle="Capture your moment"
    onClose={handleClose}
>
    {#snippet icon()}
        <Sparkles class="w-5 h-5 text-primary" />
    {/snippet}

    <!-- Main content here -->

    {#snippet actions()}
        <button class="btn btn-ghost" onclick={handleClose}>Cancel</button>
        <button class="btn btn-primary" onclick={handleSave}>Save</button>
    {/snippet}
</Modal>

Sizes:
- sm: max-w-sm
- md: max-w-lg (default)
- lg: max-w-3xl
- xl: max-w-5xl
- full: max-w-5xl h-[85vh]
```

### 10. Confirm Dialog (`ConfirmDialog`)
Structured confirmation dialog with header/content/footer matching Modal design.

**Two Variants:**
- `destructive={true}` - Red styling for permanent actions (delete)
- `destructive={false}` - Warning/amber styling for reversible actions (uncomplete)

```svelte
<!-- Destructive example (delete) -->
<ConfirmDialog
    bind:open={deleteOpen}
    title="Delete Task?"
    message="This action cannot be undone. The task and all associated data will be permanently removed."
    confirmText="Delete"
    destructive={true}
    loading={isDeleting}
    onConfirm={handleDelete}
    onCancel={() => deleteOpen = false}
/>

<!-- Warning example (uncomplete) -->
<ConfirmDialog
    bind:open={uncompleteOpen}
    title="Mark as Incomplete?"
    message="Are you sure you want to mark this task as incomplete? This will remove the completion status."
    confirmText="Mark Incomplete"
    cancelText="Keep Complete"
    destructive={false}
    loading={isPending}
    onConfirm={handleUncomplete}
    onCancel={() => uncompleteOpen = false}
/>
```

**Features:**
- Header with icon box (error or warning themed)
- Title with semantic color
- Subtitle reflecting action type
- Content area with message
- Footer with cancel/confirm actions
- Consistent with Modal component styling

---

## Table Patterns

### 11. Data Table (`DataTable`)
Consistent table with card wrapper.

```svelte
<DataTable variant="lg" hover={true}>
    <thead class="bg-base-200">
        <tr>
            <SortableHeader 
                label="Title" 
                field="title" 
                {sortField} 
                {sortDirection}
                onSort={toggleSort}
            />
            <th>Category</th>
            <th>Actions</th>
        </tr>
    </thead>
    <tbody>
        {#each items as item}
            <tr>...</tr>
        {/each}
    </tbody>
</DataTable>

Variants:
- default: table
- lg: table table-lg
- compact: table table-sm
```

### 12. Sortable Header (`SortableHeader`)
Table column header with sort controls.

```svelte
<SortableHeader
    label="Date"
    field="start_date"
    sortField={currentSort}
    sortDirection="asc"
    onSort={(field) => toggleSort(field)}
/>
```

---

## Form Patterns

### 13. Color Picker (`ColorPicker`)
Color selection with presets and custom mode.

```svelte
<ColorPicker
    bind:value={selectedColor}
    allowCustom={true}
    cols={8}
    customMode={useCustomColor}
    onCustomModeChange={(v) => useCustomColor = v}
/>
```

### Input Groups
```
Label: text-xs font-semibold uppercase opacity-50
Input: input input-bordered input-sm w-full
Error: input-error class + alert message below
```

---

## Animation

### Standard Animations
- `animate-spin` - Loading spinners
- `animate-fade-in` - Content appearing
- `transition-all duration-300` - State changes

### Animation Keyframes (defined in CSS)
```css
@keyframes fade-in {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}
```

---

## Responsive Breakpoints

- **sm** (640px): Phone landscape / tablet portrait
- **md** (768px): Tablet landscape
- **lg** (1024px): Desktop
- **xl** (1280px): Large desktop

### Common Patterns
```
Grid: grid-cols-1 sm:grid-cols-2 lg:grid-cols-3
Text: text-2xl sm:text-3xl
Padding: p-4 lg:p-6
Hide/Show: hidden sm:inline
Modal: modal-bottom sm:modal-middle
Table Columns: hidden lg:table-cell
```

---

## File Organization

```
src/lib/
├── components/
│   ├── ui/                        # Core UI primitives
│   │   ├── IconBox.svelte         # Icon containers
│   │   ├── Card.svelte            # Card containers
│   │   ├── StatCard.svelte        # Metric display
│   │   ├── PageHeader.svelte      # Page titles
│   │   ├── SectionHeader.svelte   # Section titles
│   │   ├── EmptyState.svelte      # No data placeholder
│   │   ├── ErrorAlert.svelte      # Error with retry
│   │   ├── LoadingCard.svelte     # Loading spinner
│   │   ├── Modal.svelte           # Modal dialog
│   │   ├── ConfirmDialog.svelte   # Confirmation dialog
│   │   ├── DataTable.svelte       # Table wrapper
│   │   ├── SortableHeader.svelte  # Sortable column
│   │   ├── ColorPicker.svelte     # Color selection
│   │   ├── CategoryDropdown.svelte
│   │   ├── StatusDropdown.svelte
│   │   ├── PriorityDropdown.svelte
│   │   ├── FilterDropdown.svelte
│   │   └── index.ts               # Exports
│   ├── layout/                    # Layout components
│   ├── tasks/                     # Feature: Tasks
│   ├── timeline/                  # Feature: Timeline
│   └── settings/                  # Feature: Settings
├── constants.ts                   # Shared constants (colors, etc.)
└── utils.ts                       # Utility functions
```

---

## Import Examples

```svelte
// All UI components
import { 
    Modal, 
    ConfirmDialog, 
    DataTable, 
    SortableHeader,
    IconBox, 
    StatCard, 
    SectionHeader,
    Card,
    EmptyState, 
    ErrorAlert, 
    LoadingCard,
    ColorPicker,
    PageHeader,
} from "$lib/components/ui";

// Shared constants
import { COLOR_PRESETS, DEFAULT_COLOR, getContrastColor } from "$lib/constants";
```

---

## URL State Persistence

To maintain application state across refreshes and shareable links, use the `navigation.ts` utility. This ensures filters, sort options, and other view preferences are persisted in the URL query parameters.

### Usage Pattern

1. **Import Utilities**:
```typescript
import { getUrlParams, updateUrlParams, parsers } from "$lib/utils/navigation";
import { browser } from "$app/environment";
```

2. **Initialize State**:
Use `getUrlParams` to set initial values for reactive state variables.
```typescript
const initialParams = getUrlParams({
    q: parsers.string(""),           // String param
    page: parsers.number(1),         // Number param
    active: parsers.boolean(true),   // Boolean param
    date: parsers.dateOnly(new Date()), // Date (YYYY-MM-DD local)
});

let searchQuery = $state(initialParams.q);
let page = $state(initialParams.page);
```

3. **Sync to URL**:
Use `$effect` to automatically update the URL when state changes.
```typescript
$effect(() => {
    updateUrlParams({
        q: searchQuery,
        page: page
    }, { replace: true, keepFocus: true });
});
```

4. **Navigate Back**:
When constructing "Back" buttons from detail pages, use `sessionStorage` to properly return the user to their previous filtered state if applicable.
```typescript
// On navigation TO detail page
if (browser) {
    sessionStorage.setItem("referrer", $page.url.pathname + $page.url.search);
}

// On navigation BACK
const referrer = sessionStorage.getItem("referrer") || "/default-path";
goto(referrer);
```

