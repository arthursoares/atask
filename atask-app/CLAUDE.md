# CLAUDE.md — atask Dioxus Desktop Client

## Quick Commands

```bash
cargo run              # Run in debug mode
cargo run --release    # Run optimized (much faster UI)
cargo build            # Build only
```

## Architecture

```
src/
  main.rs              → app shell, context providers, keyboard handler, SSE
  api/
    client.rs          → ApiClient wrapping reqwest, all endpoint methods
    types.rs           → serde structs matching Go API (PascalCase JSON)
    sse.rs             → SSE stream parser + reconnection
  state/
    navigation.rs      → ActiveView enum
    tasks.rs           → TaskState signals (inbox, today, upcoming, someday, logbook)
    projects.rs        → ProjectState signals (projects, areas, tags, sections)
    auth.rs            → AuthState
    command.rs         → CommandState for command palette
  components/          → one component per file, CSS classes only
  views/               → one view per file, reads from state signals
```

## Design Spec

Read `docs/design_specs/DESIGN.md` and `docs/design_specs/CLAUDE.md` before writing UI code. They are authoritative for tokens, components, views, and behavior.

Open `docs/design_specs/atask-screens-validation.html` in a browser to see the visual target.

## Go API Notes

- **PascalCase JSON**: The Go API returns `"ID"`, `"Title"`, `"ProjectID"`, etc. All serde structs use `#[serde(rename = "...")]`.
- **Status integers**: `0 = pending`, `1 = completed`, `2 = cancelled` (NOT 3 for completed).
- **Schedule integers**: `0 = inbox`, `1 = anytime`, `2 = someday`.
- **View endpoints** (`GET /views/*`, `GET /tasks`, `GET /projects`) return **bare JSON arrays**.
- **Mutation endpoints** (`POST /tasks`, `POST /tasks/{id}/complete`) return **event envelopes**: `{"event": "task.created", "data": {...}}`.
- **Notes field** is `String` not `Option<String>` — Go sends `""` for empty, not `null`.
- **Auth**: `POST /auth/login` → `{"token": "jwt..."}`. Use `Authorization: Bearer <jwt>`.

## Dioxus Reactivity Rules (CRITICAL)

These were discovered through debugging. Violating them causes silent failures where signal writes succeed but the UI never updates.

### 1. Signal reads MUST be inside `rsx!`

```rust
// WRONG — does not subscribe the component to changes
let has_token = token.read().is_some();
rsx! {
    if !has_token { ... }  // never re-evaluates when token changes
}

// CORRECT — read is inside rsx!, Dioxus tracks the dependency
rsx! {
    if token.read().is_none() { ... }  // re-renders when token changes
}
```

### 2. Signal writes from `spawn` async blocks may not trigger re-renders

Writing to a `Signal` from inside a `spawn` async task changes the data but may not notify the WebView renderer. The Dioxus desktop renderer uses a bridge between Rust and WebView — async signal updates can get lost crossing that bridge.

```rust
// UNRELIABLE — write succeeds but parent may not re-render
spawn(async move {
    let result = api.login(&email, &password).await;
    token.set(Some(result));  // data changes, UI may not update
});
```

Workaround: Use a shared `Signal` in a newtype wrapper via context, and read it inside `rsx!` in the consuming component.

### 3. `use_effect` depends on tracked signal reads

`use_effect` only re-fires when signals read inside its closure change AND Dioxus's tracking system detects the change. If the write came from `spawn`, the tracking chain can break.

### 4. Context state patterns

When sharing state between parent and child components:
- Provide via `use_context_provider`
- Read in `rsx!` (not in `let` bindings before `rsx!`)
- Prefer newtype wrappers for context types to avoid ambiguity

## Styling Rules

- **CSS only** — use classes from `assets/theme.css`, never inline `style:` attributes
- **No animations** — every state change is instant. Only `transition: background-color 80ms ease-out` on hover.
- **Design tokens** — all colors, spacing, fonts via CSS custom properties. Never hardcode values.

## Performance

`Cargo.toml` has `[profile.dev.package."*"] opt-level = 2` which optimizes dependencies in debug builds. This gives near-release UI performance with fast incremental compiles.
