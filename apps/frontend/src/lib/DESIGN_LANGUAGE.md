# Lucid Logs Design Language

> **This document is the law, not a suggestion.** If a pattern you need isn't here,
> add it to this doc and build it as a shared primitive — don't one-off it in a page.
> UI primitives live in `$lib/components/ui/`. Feature components compose them.

---

## 1. Principles

1. **One way to do each thing.** One modal. One button. One page header. One empty state.
   If two exist, one is a bug.
2. **Mobile-first, touch-first.** 44px min touch targets. Bottom sheets on mobile,
   centered dialogs on desktop. Same component handles both.
3. **Calm surfaces.** Neutral backgrounds, one accent color, generous radius,
   soft tinted shadows. No pure black, no neon.
4. **Theme tokens only.** Never hardcode hex/rgb in components — use DaisyUI
   semantic colors (`primary`, `base-100`, `base-content`, …). Exceptions live
   in the chart palette only (see §3.4).
5. **Motion is subtle.** 150–250ms, `ease-out`. Everything interactive has
   hover + active + focus-visible states.

---

## 2. Typography

Fonts: **Inter** (UI, loaded 400–800 with `opsz` axis) · **JetBrains Mono** (numbers, code).

### Scale (the only sizes allowed)

| Role | Classes |
|---|---|
| Page title | `text-2xl sm:text-[1.7rem] font-bold tracking-tight leading-tight` |
| Section title | `text-lg font-semibold` |
| Card title | `text-sm font-semibold` |
| Body | `text-sm` (default) |
| Caption / meta | `text-xs text-base-content/60` |
| Overline / label | `text-xs font-medium uppercase tracking-wide text-base-content/50` |
| Numbers / stats | `font-mono tabular-nums` (never proportional digits in stats) |

Rules: page titles come from `<PageHeader>` — never hand-write an `<h1>` in a page.
Muted text uses `text-base-content/60` (or `/50`, `/40`) — **not** `opacity-*` on the parent
(that dims children too) and **not** `#6b7280`.

---

## 3. Color

### 3.1 Themes
Only two themes ship: **`lucid-light`** (default) and **`lucid-dark`**.
No others. The theme switcher offers exactly these two (+ system).

### 3.2 Semantic mapping
| Token | Use |
|---|---|
| `primary` | The one accent: primary buttons, active nav, selected states, links |
| `secondary` | Teal — goals/progress accents only |
| `success/warning/error/info` | Status only (streaks, alerts, destructive) |
| `base-100` | Cards, sheets, raised surfaces |
| `base-200` | Page background |
| `base-300` | Inset wells, dividers on dark |
| `base-content` | Text (with `/60`, `/50`, `/40` for hierarchy) |
| `neutral` | Rarely — high-emphasis neutral chips only |

### 3.3 Surfaces & elevation
- Card: `bg-base-100 rounded-box border border-base-content/5` (+ optional `shadow-sm`)
- Page bg: `bg-base-200`
- Inset/well: `bg-base-200` on light, `bg-base-300/50` on dark — or simply `bg-base-content/5` (works both)
- Overlay (all modals/drawers): `bg-base-content/20 backdrop-blur-sm` — via `modal-backdrop`, never custom
- Borders: prefer `border-base-content/5` (subtle) over `border-base-300`; hover can bump to `/10`

### 3.4 Chart palette (the only hardcoded values allowed)
Charts may use a fixed categorical palette (defined in `$lib/utils/chart-colors.ts`),
chosen to read in both themes. Feature CSS may not introduce other hexes.

---

## 4. Shape & spacing

- Radius: selectors `rounded-selector` · fields `rounded-field` · cards/sheets `rounded-box` (1rem).
  Don't hand-pick `rounded-lg/xl/2xl` — use the tokens.
- Spacing: 4px grid. Common rhythms: `gap-2` tight, `gap-3` default, `gap-4` sections, `gap-6` page blocks.
- Page padding: `px-4 sm:px-6 lg:px-8`, vertical `py-4 sm:py-6`. Max content width `max-w-6xl mx-auto`.
- z-index scale (the only values allowed):
  `z-10` sticky in-page · `z-20` dropdowns · `z-30` sticky headers inside modals ·
  `z-40` drawers · `z-50` modals/toasts. No `z-[100]`, no `z-[9999]`.

---

## 5. Components — the canonical set

### 5.1 Button — `<Button>` (wraps DaisyUI `btn`)
Variants: `primary | neutral | ghost | outline | error` · Sizes: `xs | sm | md`.
Rules:
- **Every** clickable thing is a `<Button>` or a `<button class="btn …">`. Raw styled
  `<button>`/`<div onclick>` are banned outside primitives.
- Primary action per view: **one**. Everything else is `ghost` or `outline`.
- Icon-only buttons: `btn-ghost btn-square` (`btn-circle` only for the FAB).
- Destructive: `btn-error btn-outline` inline; `btn-error` solid only in confirm dialogs.
- Always `:active:scale-[0.98]` (built into Button), `transition`, visible `focus-visible` ring.

### 5.2 Modal — `<Modal>`
One component, bottom sheet on mobile → centered dialog on desktop
(`modal modal-bottom sm:modal-middle`, backdrop `bg-base-content/20 backdrop-blur-sm`).
- Props: `open`, `size` (`sm|md|lg|xl|full`), `title`, `subtitle`, `icon`, `actions` snippet, `onClose`.
- Header: title + optional icon, close `X` button, `border-b border-base-content/5`.
- Body: `px-6 py-5` default padding, scrollable.
- Footer (`actions` snippet): right-aligned `justify-end gap-2`, `border-t border-base-content/5`.
- **All feature modals must use `<Modal>`.** Confirm flows use `<ConfirmDialog>` (same shell,
  error-solid action). No `<dialog>` elements outside `ui/`.

### 5.3 Card — `<Card>`
`title?`, `icon?`, `actions?` snippet, `padding` (`sm|md|lg`, default md = `p-4 sm:p-5`).
Stat tiles use `<StatCard>` (value `font-mono`, label caption, delta arrow).
No bespoke `div class="bg-base-100 rounded-…"` cards in pages.

### 5.4 Page header — `<PageHeader>`
`title`, `subtitle?`, `icon?`, `actions?` snippet. Handles the title scale in §2.
Every route starts with exactly one.

### 5.5 Inputs
Text/select/textarea: DaisyUI `input input-bordered w-full` / `select` / `textarea`,
always wrapped in a `label` with `text-xs font-medium text-base-content/60`.
Errors: `input-error` + inline `text-xs text-error` below. No alert() for validation.

### 5.6 Badges / chips
Status: `badge badge-{semantic} badge-sm`. Category chips use the category color at
`/10` bg + full-color text. Priority: colored dot + label, not colored background walls.

### 5.7 States
- Loading: `<LoadingCard>` skeletons (pulse), never spinners alone.
- Empty: `<EmptyState icon title body>` + one primary CTA.
- Error: `<ErrorAlert>` inline at top of the failing region.

### 5.8 Dropdowns / popovers
`dropdown` pattern from DaisyUI; popovers `z-20`, `bg-base-100 rounded-box border border-base-content/5 shadow-lg`.
One open at a time; close on outside click (use the shared `clickOutside` action).

---

## 6. Layout

- **Desktop (lg+):** fixed left sidebar (`w-64`), content column `max-w-6xl`.
- **Mobile:** top app bar (compact, page title + primary action) + bottom tab bar
  (5 tabs max) with center-lifted FAB. `safe-area-inset-bottom` padding on the tab bar.
- Drawers/sheets slide with 200ms ease; content never jumps when they open.
- Lists on mobile render as cards; tables are desktop-only (`hidden md:table` + card list below).

---

## 7. Do / Don't

| ✅ Do | ❌ Don't |
|---|---|
| `<Modal>` for every dialog | hand-rolled `<dialog>` / `fixed inset-0` overlays |
| `btn btn-primary` via `<Button>` | `<button class="bg-indigo-600 …">` one-offs |
| `text-base-content/60` for muted text | `text-[#6b7280]`, `opacity-60` wrappers |
| `rounded-box` token | `rounded-2xl` sprinkled per-component |
| `<PageHeader>` on every route | per-page `<h1>` styling |
| z-10/20/30/40/50 from §4 | `z-[9999]`, `z-[100000]` |
| lucid-light / lucid-dark themes | shipping retro/dracula/lofi we never designed |
| 150–250ms ease-out transitions | 500ms+ bounces, or none at all |

---

## 8. Migration notes (July 2026 coherence pass)

This doc replaced a 486-line aspirational version. The audit found: 6 hand-rolled
modals (canonical `<Modal>` unused), 261 raw `<button>` elements, 22 button class
combos, 12 shipped themes, 3 competing page-title styles, z-index chaos
(`z-[100]`, `z-[9999]`, `z-[100000]`). The pass converts feature modals to `<Modal>`,
routes all page titles through `<PageHeader>`, prunes themes to lucid-light/dark,
moves chart hexes to one palette module, and normalizes the z-index scale.
