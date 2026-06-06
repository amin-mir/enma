# Enma — Design Brief

## What the app does

Users write journal entries over time. An AI chat interface lets them query their history, track goals, reflect on personal growth, and get emotional support. v1 has no social features — it's a private, single-user desktop app.

---

## Screens (v1)

1. **Register** — email + password form, link to login
2. **Login** — email + password form, link to register
3. **Journal list** — all entries sorted newest-first, each showing a date and content preview
4. **New journal entry** — full-page writing form, submit button
5. **Journal entry view** — single entry, always in edit mode. Content is a plain editable text area; changes save automatically (debounced). No save button, no read/edit toggle. Google Docs-style seamless experience.
6. **AI chat** — chat interface where the user types messages and the AI streams responses back token by token (SSE). The AI can reference past journal entries, extract goals, and provide emotional support. Goals extracted from journals/chat appear somewhere in this view (sidebar, panel, etc.)

---

## Platform context

- **Desktop app** (macOS primarily) via Tauri — not a web app or mobile app
- Window size: typical desktop window, not full-screen by default
- No browser chrome — this is a native window

---

## Tone / feel

- Calm, focused, personal — like a private notebook
- Not clinical or productivity-tool-like
- Minimal UI; the writing and conversation should be the focus

---

## Deliverables

1. **Design tokens** (colors, typography, spacing scale, border radii, shadows) as CSS custom properties or a JSON token file — used directly in Svelte
2. **Component specs** for: buttons (primary/secondary/ghost), form inputs, text areas, navigation/sidebar, chat message bubbles (user vs AI), entry list cards
3. **Screen mockups or wireframes** for all 6 screens — ASCII, SVG, or descriptive spec
4. **Layout structure** — how the app is laid out (e.g. persistent sidebar + main content area, or page-based navigation)

---

## Tech stack

- Svelte 5
- UnoCSS
- No component library — building from scratch
