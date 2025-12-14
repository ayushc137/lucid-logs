User Review Required
IMPORTANT

Technology Stack Decisions - Please confirm the following choices:

UI Framework: shadcn-svelte (Tailwind CSS based, copy-paste components) vs Skeleton UI
Svelte Version: Svelte 5 with runes (cutting-edge) or Svelte 4 (stable)
Deployment Target: Static generation, Node adapter, or Vercel/Cloudflare?
CAUTION

Setting up Svelte 5 with runes requires svelte@next and may have some ecosystem compatibility considerations.

Proposed Tech Stack
Core Framework
Package	Purpose	Why This Choice
SvelteKit 2.x	Full-stack framework	SSR, file-based routing, optimal DX
Svelte 5 (runes)	Reactivity	$state, $derived, $effect - cleaner, more explicit
TypeScript 5.x	Type safety	End-to-end types from backend to frontend
Vite 5.x	Build tool	Lightning-fast HMR, modern ES modules
UI & Styling
Package	Purpose	Why This Choice
shadcn-svelte	Component library	Beautiful, accessible, fully customizable
Bits UI	Headless primitives	Underlying shadcn components (accessible)
Tailwind CSS 3.4	Utility CSS	Rapid styling, dark mode, responsive
Lucide Svelte	Icons	1400+ consistent icons, tree-shaking
Mode Watcher	Dark mode	Svelte 5 compatible theme management
Vaul Svelte	Drawer/sheets	Mobile-friendly bottom sheets
cmdk-sv	Command palette	Cmd+K search experience
Data & Forms
Package	Purpose	Why This Choice
TanStack Query	Server state	Caching, refetching, optimistic updates
Superforms	Form management	Type-safe, validation, SvelteKit native
Zod	Schema validation	Runtime validation + TypeScript inference
ky	HTTP client	Tiny, modern fetch wrapper
Charts & Visualization
Package	Purpose	Why This Choice
Apache ECharts	Charts library	Rich chart types, great for analytics
svelte-echarts	ECharts wrapper	Svelte integration
date-fns	Date utilities	Lightweight, tree-shakeable
@internationalized/date	Calendar dates	shadcn calendar dependency
Developer Experience
Package	Purpose
ESLint + Prettier	Code quality
@sveltejs/enhanced-img	Optimized images
unplugin-icons	Auto-import icons
svelte-motion	Animations (framer-motion port)
Proposed Changes
[Component: Project Setup]
[NEW] 
SvelteKit Project
Initialize SvelteKit project with:

npx -y sv create ./apps/frontend \
  --template minimal \
  --types ts \
  --add tailwindcss,prettier,eslint
Configure shadcn-svelte:

npx shadcn-svelte@latest init
Install dependencies:

npm i @tanstack/svelte-query zod sveltekit-superforms ky date-fns echarts svelte-echarts lucide-svelte svelte-motion mode-watcher
[Component: Project Structure]
apps/frontend/
├── src/
│   ├── lib/
│   │   ├── api/                    # API client & types
│   │   │   ├── client.ts           # ky-based HTTP client
│   │   │   ├── auth.ts             # Auth endpoints
│   │   │   ├── tasks.ts            # Tasks endpoints
│   │   │   ├── goals.ts            # Goals endpoints
│   │   │   ├── emotions.ts         # Emotions endpoints
│   │   │   ├── templates.ts        # Templates endpoints
│   │   │   ├── retrospectives.ts   # Retros endpoints
│   │   │   ├── analytics.ts        # Analytics endpoints
│   │   │   └── types.ts            # Shared API types (auto-gen from Go)
│   │   │
│   │   ├── components/
│   │   │   ├── ui/                 # shadcn components
│   │   │   ├── layout/             # App shell, nav, sidebar
│   │   │   ├── tasks/              # Task-specific components
│   │   │   ├── goals/              # Goal-specific components
│   │   │   ├── emotions/           # Emotion grid, picker
│   │   │   ├── charts/             # Chart wrappers
│   │   │   └── shared/             # Reusable custom components
│   │   │
│   │   ├── stores/                 # Global stores
│   │   │   ├── auth.svelte.ts      # Auth state (runes)
│   │   │   ├── preferences.svelte.ts
│   │   │   └── ui.svelte.ts        # UI state (sidebar, modals)
│   │   │
│   │   ├── utils/                  # Utilities
│   │   │   ├── date.ts             # Date helpers
│   │   │   ├── cn.ts               # className helper
│   │   │   └── format.ts           # Formatters
│   │   │
│   │   └── config/                 # App config
│   │       └── constants.ts
│   │
│   ├── routes/
│   │   ├── +layout.svelte          # Root layout (sidebar, nav)
│   │   ├── +layout.ts              # Load user, prefetch
│   │   ├── +page.svelte            # Dashboard
│   │   │
│   │   ├── (auth)/                 # Auth group
│   │   │   ├── login/+page.svelte
│   │   │   └── register/+page.svelte
│   │   │
│   │   ├── (app)/                  # Protected routes
│   │   │   ├── tasks/
│   │   │   │   ├── +page.svelte    # Task timeline/list
│   │   │   │   └── [id]/+page.svelte
│   │   │   ├── goals/
│   │   │   │   ├── +page.svelte    # Goals dashboard
│   │   │   │   └── [id]/+page.svelte
│   │   │   ├── habits/+page.svelte # Habit tracker
│   │   │   ├── templates/+page.svelte
│   │   │   ├── emotions/+page.svelte
│   │   │   ├── retrospectives/
│   │   │   │   ├── +page.svelte
│   │   │   │   └── [id]/+page.svelte
│   │   │   ├── analytics/+page.svelte
│   │   │   └── settings/+page.svelte
│   │   │
│   │   └── api/                    # API proxy (optional)
│   │
│   └── app.css                     # Global styles + Tailwind
│
├── static/                         # Static assets
├── tailwind.config.js
├── svelte.config.js
├── vite.config.ts
└── package.json
[Component: API Client Layer]
[NEW] 
client.ts
import ky from 'ky';
import { browser } from '$app/environment';
import { goto } from '$app/navigation';
const API_BASE = import.meta.env.VITE_API_URL || 'http://localhost:8080';
export const api = ky.create({
  prefixUrl: `${API_BASE}/api/v1`,
  hooks: {
    beforeRequest: [
      (request) => {
        const token = browser ? localStorage.getItem('token') : null;
        if (token) {
          request.headers.set('Authorization', `Bearer ${token}`);
        }
      }
    ],
    afterResponse: [
      async (_request, _options, response) => {
        if (response.status === 401 && browser) {
          localStorage.removeItem('token');
          await goto('/login');
        }
      }
    ]
  }
});
[Component: TanStack Query Setup]
[NEW] 
+layout.svelte
<script lang="ts">
  import { QueryClient, QueryClientProvider } from '@tanstack/svelte-query';
  import { ModeWatcher } from 'mode-watcher';
  import '../app.css';
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 1000 * 60 * 5, // 5 minutes
        refetchOnWindowFocus: true,
      }
    }
  });
</script>
<QueryClientProvider client={queryClient}>
  <ModeWatcher />
  <slot />
</QueryClientProvider>
[Component: Key UI Components]
Emotion Grid (100 emotions in 4 quadrants)
Beautiful 10×10 grid with color-coded quadrants (🟡 Yellow, 🟢 Green, 🔴 Red, 🔵 Blue), hover previews, and click-to-select.

Task Timeline
Vertical timeline view showing tasks by time with overlap handling, drag-to-reschedule, and quick-add.

Quick Log Widget
Floating action button with frequently-used templates for one-tap task creation.

Goal Progress Cards
Visual progress indicators with streak counts, completion percentages, and target tracking.

Analytics Dashboard
ECharts-powered charts: mood trends, habit heatmaps, productivity metrics, category distributions.

[Component: Design System]
Colors (Tailwind Extended)
// tailwind.config.js colors
{
  // Emotion quadrants
  'mood-yellow': { 50: '#fffbeb', 500: '#f59e0b', 900: '#78350f' },
  'mood-green':  { 50: '#ecfdf5', 500: '#10b981', 900: '#064e3b' },
  'mood-red':    { 50: '#fef2f2', 500: '#ef4444', 900: '#7f1d1d' },
  'mood-blue':   { 50: '#eff6ff', 500: '#3b82f6', 900: '#1e3a8a' },
  
  // Accent
  'primary': { 50: '#f5f3ff', 500: '#8b5cf6', 900: '#4c1d95' }
}
Typography
Font: Inter (Google Fonts) - clean, modern, excellent readability
Scale: 12, 14, 16, 20, 24, 32, 48px
Animations
Subtle micro-interactions on all interactive elements
Page transitions using Svelte transitions
Loading states with skeleton screens
Toast notifications with slide-in
Verification Plan
Automated Tests
After implementation, run:

# Build verification
cd apps/frontend && npm run build
# Type checking
npm run check
# Linting
npm run lint
Manual Verification
Since this is frontend setup, verification will be done by:

Start the dev server:

cd /home/enduser/Documents/code/task/apps/frontend
npm run dev
Verify homepage loads at http://localhost:5173

Test shadcn components render correctly

Test dark mode toggle works

Verify API connection to Go backend (requires backend running)

NOTE

Full E2E testing with Playwright can be added later. For initial setup, visual verification is sufficient.

Implementation Phases
Phase	Focus	Duration
1. Foundation	Project setup, config, shadcn	~30 min
2. Auth Flow	Login, register, protected routes	~30 min
3. Layout Shell	Sidebar, navigation, theme	~20 min
4. Tasks Module	Timeline, CRUD, emotions	~1 hour
5. Goals Module	Goals, habits, progress	~1 hour
6. Templates	Quick log, template management	~30 min
7. Retrospectives	Daily/weekly retros	~30 min
8. Analytics	Charts, dashboards	~45 min
User Experience Principles
Minimal Input: Use templates, defaults, and smart suggestions
Visual Feedback: Every action shows immediate feedback
Keyboard First: Full keyboard navigation + Cmd+K
Mobile Ready: Touch-friendly, responsive from day 1
Delightful Details: Micro-animations, emoji integration, color-coded emotions
Dark Mode: First-class dark theme support
Fast: Optimistic updates, skeleton loaders, prefetching
