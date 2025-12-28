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
- **Input**: Standard form control padding

---

## Component Patterns

### 1. Icon Container
Square containers for icons with semantic coloring.

```
Size Options:
- Small: w-9 h-9 rounded-lg
- Medium: w-10 h-10 rounded-xl  
- Large: w-16 h-16 rounded-full (for empty states)
- XLarge: w-20 h-20 rounded-full (for featured states)

Color Patterns:
- Solid: bg-{color} text-{color}-content
- Subtle: bg-{color}/10 text-{color}
```

### 2. Stat Card
Display key metrics with icon and value.

```
Structure:
[Icon Container] [Label + Value Stack]

Icon: 10x10 rounded-xl with semantic bg
Label: text-xs font-medium uppercase opacity-50
Value: text-xl font-bold text-{color}
Unit: text-sm font-medium opacity-60 ml-0.5
```

### 3. Section Header
Consistent header pattern for sections.

```
Structure:
[Icon Container] [Title + Subtitle]

Icon: w-10 h-10 rounded-xl bg-{color}/10
Title: font-semibold text-sm
Subtitle: text-xs opacity-50
```

### 4. Cards
Container components with consistent styling.

```
Standard Card:
- container: card bg-base-100 shadow-lg border border-base-200
- body: card-body p-4 or card-body py-12 (for empty states)

Simple Card:
- container: card bg-base-100 shadow-sm
- body: card-body p-4
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

### 6. Empty State
Centered content when no data exists.

```
Structure:
- Icon: w-20 h-20 rounded-full bg-primary/10 mx-auto mb-4
- Title: text-2xl font-bold
- Description: opacity-60 mt-2 max-w-md mx-auto
- CTA: btn btn-primary mt-6 gap-2 shadow-lg shadow-primary/20
```

### 7. Error State
Alert for error conditions with retry capability.

```
Structure:
- Container: alert alert-error shadow-lg
- Icon: CircleAlert w-5 h-5
- Message: span with text
- Action: btn btn-sm btn-ghost (Retry)
```

### 8. Loading State
Centered spinner with message.

```
Structure:
- Spinner: loading loading-spinner loading-lg text-primary
- Message: opacity-60 mt-4
```

---

## Form Patterns

### Input Groups
```
Label: text-xs font-semibold uppercase opacity-50
Input: input input-bordered input-sm w-full
Error: input-error class + alert message below
```

### Color Picker
```
Grid: grid-cols-8 gap-2 (for 16 colors)
Button: w-8 h-8 rounded-lg
Selected: ring-2 ring-primary ring-offset-2 scale-110
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
```

---

## File Organization

```
src/lib/
├── components/
│   ├── ui/                    # Core UI primitives
│   │   ├── PageHeader.svelte
│   │   ├── EmptyState.svelte
│   │   ├── ErrorAlert.svelte
│   │   ├── LoadingCard.svelte
│   │   ├── StatCard.svelte    # New
│   │   ├── IconBox.svelte     # New
│   │   ├── SectionHeader.svelte # New
│   │   └── index.ts
│   ├── layout/                # Layout components
│   ├── tasks/                 # Feature: Tasks
│   ├── timeline/              # Feature: Timeline
│   └── settings/              # Feature: Settings
├── constants.ts               # Shared constants (colors, etc.)
└── utils.ts                   # Utility functions
```
