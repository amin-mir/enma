# Enma — Design Decisions

## What We're Building

An AI-enhanced journaling application. Users write journal entries over time, and an AI chat interface lets them query their history, track goals, reflect on personal growth, and get support on difficult days.

Notes (shopping lists, to-do lists, routines) are a planned future feature. **v1 focuses entirely on journaling.**

---

## Features (v1)

- **Write journal entry** — dedicated form, free text, saved directly to the database
- **List journal entries** — sorted descending by date
- **AI chat** — single interface for everything:
  - Ask questions about past journals
  - Personality and growth analysis (on-demand, user must ask explicitly)
  - Goal tracking (AI manages automatically — see below)
  - Emotional support / motivational chat on hard days

## Deferred to v2

- Notes (quick notes, to-do lists, shopping lists, routines)
- Social login / OAuth

---

## Tech Stack

| Layer | Choice |
|---|---|
| Backend | Go |
| Frontend | Tauri + Svelte (skeleton already in `/frontend`) |
| AI provider | OpenAI |
| Primary database | Postgres |
| Write-heavy storage | Cassandra (chat messages only) |
| Vector search | pgvector (Postgres extension) |

---

## Architecture Decisions

### Database Split

- **Postgres**: users, journal_entries, goals, refresh_tokens, vector embeddings (pgvector)
- **Cassandra**: chat_messages (append-only, high write volume, always queried by user + time)

Cassandra is used only where the access pattern is truly write-heavy and append-only. Everything relational stays in Postgres.

### Journal Embeddings & Retrieval (RAG)

- Each journal entry is embedded and stored in Postgres via pgvector
- **Short entries (<~500 tokens)**: single chunk per entry
- **Long entries (>~500 tokens)**: split by paragraph with 1–2 sentence overlap, each chunk stored separately with a reference back to the parent `journal_entry_id`
- At chat time: top-k relevant chunks are retrieved semantically, deduplicated by entry if needed, and injected into the prompt

### Goal Tracking

- Goals are **not** entered explicitly by the user — the AI extracts them automatically from journal entries and chat messages
- Goals are stored in a dedicated `goals` table in Postgres (concise, easy to inject into every AI request without RAG)
- The AI manages goals via tool calls: `add_goal`, `update_goal`, `remove_goal`
- Example: user writes "I don't want to pursue that anymore" → AI calls `remove_goal`

### Chat Streaming

- OpenAI and Anthropic both use **SSE (Server-Sent Events)**, not WebSockets
- SSE fits the request/response pattern of LLM chat — client sends one message, server streams tokens back
- Go backend forwards OpenAI's SSE stream to the Tauri client
- Svelte client reads the stream via `fetch` with `ReadableStream`

### Auth — JWT with Rotating Refresh Tokens

- **Access token**: signed JWT, expires in 15 minutes, stateless (server validates signature only, no DB hit)
- **Refresh token**: opaque random string, expires in 30 days, stored as a hash in Postgres
- **Rotation**: each time the refresh token is used, a new one is issued and the old one is invalidated. Reuse of an old refresh token signals theft — all tokens for that user are invalidated.
- Tauri stores the refresh token in the OS keychain
- Access token lives in memory only
- No OAuth / social login in v1 — email + password only
- Pattern originates from OAuth 2.0 (RFC 6749); relevant reading: RFC 8725 (JWT Best Practices)

---

## Data Models (preliminary)

| Table | Database | Key Fields |
|---|---|---|
| `users` | Postgres | id, email, password_hash, created_at |
| `journal_entries` | Postgres | id, user_id, content, created_at |
| `journal_chunks` | Postgres (pgvector) | id, journal_entry_id, user_id, embedding, content, created_at |
| `goals` | Postgres | id, user_id, description, status (active/completed/abandoned), created_at, updated_at |
| `refresh_tokens` | Postgres | id, user_id, token_hash, expires_at, created_at |
| `chat_messages` | Cassandra | user_id (partition), created_at (clustering), message_id, role, content |

---

## Journal Autosave

The journal entry view is always in edit mode — no save button. Changes persist automatically.

- **Debounce:** client waits ~1-2s after the user stops typing before firing a PUT
- **Client-side hash check:** before firing the debounced PUT, compute a hash of the current content (e.g. a simple string hash or SHA). If it matches the hash of the last successfully saved content, skip the request — no-op saves cost nothing
- **Optimistic concurrency (version number):** each `journal_entries` row has a `version INTEGER` column. Every PUT sends the current version; the server increments it atomically. If the WHERE clause matches zero rows, the server returns 409 Conflict (a newer in-flight save already landed). The client discards the stale request silently — the newer save already has the correct content.
- **No WebSockets needed** — debounced PUT over HTTP is sufficient for single-user autosave

## Still To Decide

- Cassandra schema details (partition keys, clustering keys, TTL strategy)
- Postgres schema details (indexes, constraints)
- OpenAI model selection (chat model + embedding model)
- Go project structure and API design
- Tauri/Svelte app structure
