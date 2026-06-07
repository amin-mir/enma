package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/amin-mir/enma/internal/db"
	"github.com/amin-mir/enma/internal/user"
)

const sqlstateUniqueViolation = "23505"

type Store struct{ DB db.DBTX }

func (s Store) CreateUserAndRefreshToken(ctx context.Context, email, passwordHash, tokenHash string, expiresAt time.Time) (uuid.UUID, error) {
	var userID uuid.UUID
	err := s.DB.QueryRow(ctx, `
		WITH new_user AS (
			INSERT INTO users (email, password_hash) VALUES ($1, $2)
			RETURNING id
		)
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		SELECT id, $3, $4 FROM new_user
		RETURNING user_id`,
		email, passwordHash, tokenHash, expiresAt,
	).Scan(&userID)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == sqlstateUniqueViolation {
			return uuid.UUID{}, user.ErrEmailInUse
		}
		return uuid.UUID{}, err
	}
	return userID, nil
}

func (s Store) CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error {
	_, err := s.DB.Exec(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

// Atomically deletes the old token and inserts the new one in a single CTE,
// returning user_id so the caller can issue a new access token without a second query.
// DELETE uses the unique index on token_hash; expires_at > NOW() is a post-index filter.
func (s Store) RotateRefreshToken(ctx context.Context, oldHash, newHash string, expiresAt time.Time) (uuid.UUID, error) {
	var userID uuid.UUID
	err := s.DB.QueryRow(ctx, `
		WITH deleted AS (
			DELETE FROM refresh_tokens
			WHERE token_hash = $1 AND expires_at > NOW()
			RETURNING user_id
		)
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		SELECT user_id, $2, $3
		FROM deleted
		RETURNING user_id`,
		oldHash, newHash, expiresAt,
	).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.UUID{}, ErrInvalidToken
	}
	return userID, err
}

// Uses the unique index on token_hash.
func (s Store) DeleteRefreshTokenByHash(ctx context.Context, tokenHash string) error {
	_, err := s.DB.Exec(ctx,
		`DELETE FROM refresh_tokens WHERE token_hash = $1`,
		tokenHash,
	)
	return err
}
