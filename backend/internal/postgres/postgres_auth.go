package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (pg *Postgres) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := pg.pool.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

// RotateRefreshToken atomically deletes the old refresh token and inserts the new one in a single
// CTE, returning the user_id so the caller can issue a new access token without a separate query.
// Returns ErrNotFound if the old token doesn't exist or has expired.
// DELETE uses the unique index on token_hash; expires_at > NOW() is a post-index filter.
func (pg *Postgres) RotateRefreshToken(ctx context.Context, oldTokenHash, newTokenHash string, expiresAt time.Time) (uuid.UUID, error) {
	var userID uuid.UUID
	err := pg.pool.QueryRow(ctx, `
		WITH deleted AS (
			DELETE FROM refresh_tokens
			WHERE token_hash = $1 AND expires_at > NOW()
			RETURNING user_id
		)
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		SELECT user_id, $2, $3
		FROM deleted
		RETURNING user_id`,
		oldTokenHash, newTokenHash, expiresAt,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, ErrNotFound
	}
	return userID, err
}

// Uses the unique index on token_hash.
func (pg *Postgres) DeleteRefreshTokenByHash(ctx context.Context, tokenHash string) error {
	_, err := pg.pool.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	)
	return err
}
