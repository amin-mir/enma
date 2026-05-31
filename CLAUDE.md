# Enma — Agent Notes

## References

The `references/` directory contains cloned repos used as examples and guidance. It is gitignored and must be cloned manually on each machine.

### Fiber recipes
- **Clone:** `git clone https://github.com/gofiber/recipes references/recipes`
- **Use for:** Fiber SSE examples, JWT auth examples, middleware patterns, project structure

When implementing Fiber features, check `references/recipes` first for working examples before writing from scratch.

### pgx source
- **Clone:** `git clone https://github.com/jackc/pgx references/pgx`
- **Use for:** pgx API reference, row helper functions, type mapping, pool usage

Key row helper pattern (prefer over manual scanning):
```go
// Single row
rows, _ := pool.Query(ctx, "SELECT id, content, created_at FROM journal_entries WHERE id=$1", id)
entry, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[JournalEntry])

// Multiple rows
rows, _ := pool.Query(ctx, "SELECT id, content, created_at FROM journal_entries WHERE user_id=$1", userID)
entries, err := pgx.CollectRows(rows, pgx.RowToStructByName[JournalEntry])
```

Struct field mapping uses `db` tag or auto-converts CamelCase → snake_case (e.g. `UserID` → `user_id`).
Variants: `RowToStructByName` (strict), `RowToStructByNameLax` (lenient), `RowToStructByPos` (positional).

## Rules

### postgres.DB interface
When adding a new method to `*Postgres`:
1. Add it to the `DB` interface in `internal/postgres/postgres.go`
2. If the method returns a sentinel error (`ErrNotFound`, `ErrUniqueViolation`), document it with a one-line comment on the interface method
3. Run `go generate ./internal/postgres/...` to regenerate the mock

## Docs

- `DECISIONS.md` — all architecture and tech stack decisions made so far
- `PLAN.md` — phased implementation plan
