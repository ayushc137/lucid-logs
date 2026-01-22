# Graveyard - Deprecated Components

This folder contains components that are no longer in active use but have been
preserved for reference or potential future restoration.

## Components

### TimelineAgenda.svelte
**Deprecated:** 2026-01-22

Time-block based agenda view for displaying tasks organized by time of day
(Early Morning, Morning, Afternoon, Evening, Night). Replaced by using only
the TimelineGantt view to reduce UI clutter.

### AgendaTaskCard.svelte
**Deprecated:** 2026-01-22

Individual task card component used by TimelineAgenda. No longer needed since
the agenda view was removed.

### TimelineViewSwitcher.svelte
**Deprecated:** 2026-01-22

Toggle button component for switching between Timeline and Agenda views.
No longer needed since only the Timeline view is used.

### TaskModal.svelte
**Deprecated:** 2026-01-22

Modal dialog for creating/editing tasks inline. Replaced by dedicated pages:
- `/tasks/create` - for creating new tasks
- `/tasks/[id]` - for editing existing tasks

## Restoring Components

If you need to restore any of these components:

1. Move the component back to its original location
2. Update the corresponding `index.ts` exports
3. Fix any import paths that may have changed
4. Update CLAUDE.md documentation if needed
