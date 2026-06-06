# Enma — Agent Notes

## References

The `references/` directory contains cloned repos used as examples and guidance. It is gitignored.

To set up on a new machine: `just refs-clone` (reads `references/repos.txt`).
To update all repos: `just refs-pull`.
To add a new repo: append its URL to `references/repos.txt`.

When researching a reference repo, use the **Explore subagent** — even when the file path is known. This keeps raw doc content out of the main context. Only read directly when the needed snippet is short (< ~50 lines) and already known.

When you discover a useful pattern, API, or gotcha from a reference repo during implementation, **add it to the relevant section below** so future sessions benefit without re-researching.

### Fiber recipes
- **Repo:** `references/recipes`
- **Use for:** Fiber SSE examples, JWT auth examples, middleware patterns, project structure

When implementing Fiber features, check `references/recipes` first for working examples before writing from scratch.

### pgx source
- **Repo:** `references/pgx`
- **Use for:** pgx API reference, row helper functions, type mapping, pool usage

#### Query vs QueryRow

`Query` returns `pgx.Rows`. `QueryRow` returns `pgx.Row`. They are different types.
`RowToStructByName` (and all `RowToFunc` variants) accept `CollectableRow`, which only
`pgx.Rows` implements — **not** `pgx.Row`. So `QueryRow` can only be used with manual `.Scan()`.

#### When to use what

| Goal | Pattern |
|------|---------|
| Single row → struct | `Query` + `CollectOneRow` + `RowToStructByName` |
| Multiple rows → []struct | `Query` + `CollectRows` + `RowToStructByName` |
| Single primitive / few fields | `QueryRow` + `.Scan(&a, &b)` |

`CollectOneRow` automatically closes rows and returns `pgx.ErrNoRows` if no row matched.

```go
// Single row into struct
rows, _ := pool.Query(ctx, "SELECT id, content, version, created_at, updated_at FROM journal_entries WHERE id=$1", id)
entry, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[JournalEntry])

// Multiple rows into slice
rows, _ := pool.Query(ctx, "SELECT id, content, version, created_at, updated_at FROM journal_entries WHERE user_id=$1", userID)
entries, err := pgx.CollectRows(rows, pgx.RowToStructByName[JournalEntry])

// Single primitive — QueryRow is fine here
var count int
pool.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
```

Struct field mapping uses `db` tag or auto-converts CamelCase → snake_case (e.g. `UserID` → `user_id`).
Variants: `RowToStructByName` (strict), `RowToStructByNameLax` (lenient), `RowToStructByPos` (positional).

### golang-jwt/jwt source
- **Repo:** `references/jwt`
- **Use for:** JWT signing, parsing, claims validation, parser options

Key usage rules:
- Always pass `jwt.WithValidMethods([]string{"HS256"})` to `ParseWithClaims` — the library strongly recommends this to prevent algorithm confusion attacks. It checks the signing method before the keyfunc is called, so the keyfunc only needs to return the key.
- Do not manually check `t.Method.(*jwt.SigningMethodHMAC)` in the keyfunc — `WithValidMethods` handles this at the parser level.
- Do not check `!token.Valid` after `ParseWithClaims` — if `err == nil`, `token.Valid` is always `true`. Just check `err`.

### UnoCSS source
- **Repo:** `references/unocss`
- **Use for:** UnoCSS config, preset APIs, Svelte/Vite integration, custom rules/shortcuts
- **Docs:** `references/unocss/docs/` — user-facing config and preset docs
- **Types:** `references/unocss/packages-engine/core/src/types.ts` — full plugin/programmatic API types

## Rules

### Testing
- Run `just test` (from `backend/`) before every commit and push — all tests must pass.
- After editing any file in `internal/postgres/`, run the postgres tests: `go test ./internal/postgres/...`. They use testcontainers and need no manual DB setup.

### postgres.DB interface
When adding a new method to `*Postgres`:
1. Add it to the `DB` interface in `internal/postgres/postgres.go`
2. If the method returns a sentinel error (`ErrNotFound`, `ErrUniqueViolation`), document it with a one-line comment on the interface method
3. Run `go generate ./internal/postgres/...` to regenerate the mock

## Docs

- `DECISIONS.md` — all architecture and tech stack decisions made so far
- `PLAN.md` — phased implementation plan
- `ARCHITECTURE.md` — backend package structure, DBTX interface, Store pattern, testing strategy, Phase 3 chat design
