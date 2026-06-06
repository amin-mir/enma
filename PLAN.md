# Enma — Implementation Plan

## Phase 1 — Backend Foundation ✅

### Step 1: Project Setup ✅
- Initialize Go module under `/backend`
- Dependencies: Fiber, pgx (Postgres), gocql (Cassandra), golang-jwt, bcrypt
- Folder structure: `cmd/`, `internal/`, `migrations/`

### Step 2: Database Schema ✅ (Postgres) / ⏳ (Cassandra deferred)
- Postgres: `users`, `journal_entries`, `refresh_tokens` ✅
- Cassandra: `chat_messages` keyspace + table — deferred to Phase 3 (Step 10)
- Migration tooling: golang-migrate via Docker ✅

### Step 3: Auth ✅
- `POST /auth/register` — email + password, Argon2id hash ✅
- `POST /auth/login` — returns access token + refresh token ✅
- `POST /auth/refresh` — rotates refresh token via CTE ✅
- `POST /auth/logout` — invalidates refresh token ✅
- JWT middleware to protect all other routes ✅

### Step 4: Journal Endpoints ✅
- `POST /journals` — create entry ✅
- `GET /journals` — list all for authenticated user, descending ✅
- `GET /journals/:id` — get single entry ✅
- `PUT /journals/:id` — update entry ✅

---

## Phase 2 — Frontend Foundation 👈 NEXT

### Step 5: Auth Screens
- Register and login pages
- Store refresh token in Tauri keychain, access token in memory
- Redirect to journal list on successful login

### Step 6: Journal List
- Fetch and display all entries sorted descending
- Each entry shows a preview and date

### Step 7: Journal Form
- Dedicated form for writing a new entry
- Submit to backend, redirect to list on success

### Step 8: Journal View / Edit
- View a single journal entry
- Edit in place and save changes

---

## Phase 3 — AI Features

### Step 9: Embeddings Pipeline
- pgvector setup
- On journal save: embed content via OpenAI, store in `journal_chunks` table
- Chunking strategy: single chunk for short entries, paragraph-split with overlap for long entries

### Step 10: Chat Interface
- Cassandra schema for `chat_messages`
- SSE streaming endpoint on backend
- Chat UI in Svelte with streaming token display
- Conversation stored in Cassandra

### Step 11: Goal Tracking
- `goals` table in Postgres
- AI uses tool calls to add / update / remove goals
- Goals injected into every chat context

### Step 12: RAG for Journal Queries
- On each chat message: semantic search over user's journal chunks
- Top-k chunks injected into prompt alongside goals
- AI answers questions about personality, growth, past events
