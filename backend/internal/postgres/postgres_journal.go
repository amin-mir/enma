package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/amin-mir/enma/internal/model"
)

type CreateJournalEntryResult struct {
	ID        uuid.UUID `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	Version   int32     `db:"version"`
}

func (pg *Postgres) CreateJournalEntry(ctx context.Context, userID uuid.UUID, content string) (CreateJournalEntryResult, error) {
	rows, _ := pg.pool.Query(ctx,
		`INSERT INTO journal_entries (user_id, content)
		 VALUES ($1, $2)
		 RETURNING id, created_at, version`,
		userID, content,
	)
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[CreateJournalEntryResult])
}

// Uses composite index (user_id, created_at DESC) — covers both the WHERE and ORDER BY.
func (pg *Postgres) ListJournalEntries(ctx context.Context, userID uuid.UUID) ([]model.JournalEntry, error) {
	rows, _ := pg.pool.Query(ctx,
		`SELECT id, user_id, content, version, created_at, updated_at
		 FROM journal_entries
		 WHERE user_id = $1
		 ORDER BY created_at DESC`,
		userID,
	)
	return pgx.CollectRows(rows, pgx.RowToStructByName[model.JournalEntry])
}

// Uses the primary key index on id; user_id is a post-index filter.
func (pg *Postgres) GetJournalEntry(ctx context.Context, entryID, userID uuid.UUID) (model.JournalEntry, error) {
	rows, _ := pg.pool.Query(ctx,
		`SELECT id, user_id, content, version, created_at, updated_at
		 FROM journal_entries
		 WHERE id = $1 AND user_id = $2`,
		entryID, userID,
	)
	entry, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.JournalEntry])
	if errors.Is(err, pgx.ErrNoRows) {
		return model.JournalEntry{}, ErrNotFound
	}
	return entry, err
}

// Uses the primary key index on id; user_id and version are post-index filters.
func (pg *Postgres) UpdateJournalEntry(ctx context.Context, entryID, userID uuid.UUID, content string, version int32) (int32, error) {
	var newVersion int32
	err := pg.pool.QueryRow(ctx,
		`UPDATE journal_entries
		 SET content = $1, updated_at = NOW(), version = version + 1
		 WHERE id = $2 AND user_id = $3 AND version = $4
		 RETURNING version`,
		content, entryID, userID, version,
	).Scan(&newVersion)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrConflict
	}
	return newVersion, err
}
