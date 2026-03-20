# CLAUDE.md — atask Dioxus App

## What This Project Is

atask is an AI-first personal task manager. This repo is the **Dioxus desktop client** — a cross-platform app (macOS primary) that consumes the atask Go REST API.

## Design System

**Read `DESIGN.md` before writing any UI code.** It is the authoritative spec for:
- All design tokens (colors, spacing, typography, shadows, radii)
- Component specs with RSX examples and CSS
- View layouts and behaviors for every screen (Today, Inbox, Upcoming, Someday, Logbook, Project)
- Command palette spec
- Keyboard shortcut map
- Motion policy (no animations)
- API-to-view data mapping with Rust types

**`theme.css`** is the production stylesheet. Copy it to `assets/theme.css`. All visual tokens are CSS custom properties — do not hardcode colors, sizes, or spacing in RSX.

**`atask-screens-validation.html`** is the interactive reference mockup. Open it in a browser to see how every view should look and behave.

## Key Architectural Rules

1. **Styling is CSS, not inline.** Use CSS classes from `theme.css`. Dioxus renders in a WebView, so standard CSS works. Use `class:` attribute in RSX, not `style:`.

2. **State is signals.** Use Dioxus signals (`use_signal`, `use_memo`) for all state. Global app state lives in `src/state/`. No `use_state` — that's React.

3. **Optimistic updates.** Update local signals immediately on user action, fire API call async, rollback on failure. See §10 in DESIGN.md.

4. **SSE for real-time.** Subscribe to `GET /events/stream` on app launch. Update signals when events arrive. This keeps multiple views in sync.

5. **Components are small.** One component per file in `src/components/`. Props derive `Clone + PartialEq`. Use `#[component]` macro.

6. **API client is separate.** `src/api/client.rs` wraps reqwest. All endpoints return `Result<T, ApiError>`. Never call HTTP from components directly — go through state layer.

## Aesthetic Guidelines

- **Theme:** "Bone" — ivory canvas (`#f6f5f2`), desaturated blue accent (`#4670a0`). Warm but not Anthropic-beige.
- **Font:** Atkinson Hyperlegible. Load from `assets/fonts/`. Never fall back to Inter, Roboto, or Arial.
- **Task rows:** Single-line, 32px height. Checkbox, title, and metadata all on one horizontal axis. Title truncates with ellipsis; metadata pins to right edge.
- **Checkboxes:** Circular for tasks (20px), square for checklist items (16px). State change is instant — no animation.
- **Agent elements:** Use `--agent-tint` (`#7868a8`) purple. Never elevate agent UI above human UI — same stream, different tint.
- **Today:** Amber (`--today-star` `#c88c30`), not red or blue. Checkbox borders are amber in Today view.
- **No priority flags.** Position in list = priority. This is a deliberate design constraint.
- **No animations.** Every state change is instant. No entrance animations, no stagger delays, no slide-ins, no scale-ins, no checkPop. The only CSS transitions allowed are hover feedback on interactive elements: `background-color 80ms ease-out`. If it moves, it's wrong.

## File Naming

- Components: `snake_case.rs` (e.g., `task_item.rs`, `command_palette.rs`)
- CSS classes: `kebab-case` (e.g., `task-item`, `command-palette`)
- Signals: `snake_case` (e.g., `selected_task`, `inbox_count`)

## API Base URL

Default: `http://localhost:8080` — configurable via environment variable `ATASK_API_URL`.

## Don't

- Don't use Tailwind utility classes in RSX. Use the semantic CSS classes from `theme.css`.
- Don't hardcode any color, size, or spacing value. Use CSS variables.
- Don't create modals for task editing. Use the 340px detail panel (right side).
- Don't add priority fields, labels, or UI. This app has no priority system by design.
- Don't use serif fonts, emoji as icons, or decorative elements. The aesthetic is restrained.
- Don't add any CSS animations, keyframes, transitions (except 80ms hover smoothing), or stagger delays. State changes are instant.
- Don't make task items two-line. Title and metadata share one 32px row.
