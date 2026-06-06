# Enma — Backend Architecture

## Philosophy

Package by feature (vertical slice). Each package owns its types, DB queries,
business logic, and HTTP handlers. `cmd/main.go` is pure assembly.

When a feature needs multiple domains (e.g. chat needs goals + messages + RAG),
create an umbrella package that imports and orchestrates the sub-packages.

---

## Package Structure

```
backend/
  cmd/
    main.go         — assembly only: create pool, mount packages, start server
  internal/
    db/             — DBTX interface + NewPool (no domain logic)
    config/         — env/config loading
    middleware/     — JWT auth middleware (shared, imported by packages that need protected routes)
    password/       — Argon2id hashing

    user/           — User type + Store{Create, GetByEmail}
    auth/           — Register/Login/Refresh/Logout, token types, handlers; imports user/
    journal/        — JournalEntry type, Store, CRUD, handlers

    # Phase 3
    goal/           — Goal type, Postgres Store, handlers (GET /goals etc.)
    message/        — Message type, Cassandra Store (no handlers — only used by chat/)
    chat/           — SSE handler, orchestrates: message/ + goal/ + journal/ + embedding/
    embedding/      — OpenAI embedding calls + pgvector storage (no handlers)
```

### Mounting convention

Each HTTP-facing package exposes a `Mount` function:

```go
// in cmd/main.go
auth.Mount(app, pool)
journal.Mount(app, pool, middleware.JWT(secret))
chat.Mount(app, pool, middleware.JWT(secret))
goal.Mount(app, pool, middleware.JWT(secret))
```

---

## DBTX Interface

Defined in `internal/db`. Both `*pgxpool.Pool` and `pgx.Tx` satisfy it structurally.
All Store methods and business logic functions accept `db.DBTX`, never a concrete type.
This allows callers to pass either a pool (auto-transaction per statement) or a tx
(coordinated transaction across multiple calls).

```go
package db

type DBTX interface {
    Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
    Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
    QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}
```

Use `pgx.BeginFunc` when multiple operations must be atomic:

```go
pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
    id, err := user.Store{tx}.Create(ctx, email, hash)
    if err != nil {
        return err
    }
    return auth.Store{tx}.CreateRefreshToken(ctx, id, tokenHash, expiresAt)
})
```

---

## Store Pattern

Each package defines a `Store` struct holding a `db.DBTX`. DB query methods live on
`Store`. Business logic functions take a `db.DBTX` directly and construct a Store
internally, or accept a Store if they need to call multiple methods.

```go
// user/user.go
type Store struct{ DB db.DBTX }

func (s Store) Create(ctx context.Context, email, hash string) (uuid.UUID, error) { ... }
func (s Store) GetByEmail(ctx context.Context, email string) (User, error) { ... }

// auth/auth.go
func Register(ctx context.Context, db db.DBTX, email, password string) (TokenPair, error) {
    hash, err := password.Hash(password)
    ...
    id, err := user.Store{db}.Create(ctx, email, hash)
    ...
}
```

When a function needs multiple stores or non-DB dependencies (e.g. JWT secret, OpenAI client),
group them in a config/deps struct rather than growing the parameter list:

```go
func Login(ctx context.Context, deps LoginDeps, email, password string) (TokenPair, error)

type LoginDeps struct {
    DB        db.DBTX
    JWTSecret string
}
```

---

## Package Dependency Rules

- `user/` — no imports of other internal packages
- `auth/` — may import `user/`
- `journal/` — no imports of other internal packages
- `goal/` — no imports of other internal packages
- `message/` — no imports of other internal packages
- `embedding/` — no imports of other internal packages
- `chat/` — may import `goal/`, `message/`, `journal/`, `embedding/`
- No package may import `handler/` (it no longer exists)
- Circular imports are forbidden; if two packages need each other, extract the shared type

---

## Testing

### Strategy
- Handler tests are integration tests — they test the full stack (HTTP → business logic → DB)
- Focus test coverage on handler/service layer; minimal redundant tests in Store layer
- No mocks; no generated mock files

### Shared container via TestMain
Each package that needs a DB spins up one testcontainer for the whole package test suite:

```go
var pool *pgxpool.Pool

func TestMain(m *testing.M) {
    ctx := context.Background()
    container, connStr := startPostgres(ctx)   // testcontainers
    runMigrations(connStr)                      // golang-migrate
    pool = newPool(connStr)
    code := m.Run()
    container.Terminate(ctx)
    os.Exit(code)
}
```

### Test isolation
All tests run in parallel (`t.Parallel()`). Isolation is by unique data — each test
generates unique identifiers (UUIDs for emails, IDs, etc.) so tests never interfere.
No transaction rollback tricks needed; the container is ephemeral.

```go
func TestRegister(t *testing.T) {
    t.Parallel()
    email := uuid.New().String() + "@test.com"
    ...
}
```

---

## Phase 3 — Chat Package Design

`chat/` is the umbrella for the AI chat feature. It owns the SSE HTTP handler and
orchestrates sub-packages:

```
chat/ ──▶ message/    store & retrieve chat messages (Cassandra)
     ──▶ goal/        update goals via AI tool calls
     ──▶ journal/     semantic search for RAG context
     ──▶ embedding/   OpenAI embedding calls + pgvector queries
```

- Goals are auto-extracted by the AI via tool calls (`add_goal`, `update_goal`, `remove_goal`)
- `goal/` has standalone handlers for reading goals (`GET /goals`) independently of chat
- `message/` has no handlers — messages are only surfaced through the chat interface
- Chat streaming uses SSE; the Go backend forwards OpenAI's SSE stream to the Tauri client
