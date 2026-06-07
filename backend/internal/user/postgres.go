package user

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/amin-mir/enma/internal/db"
)

const sqlstateUniqueViolation = "23505"

type Store struct{ DB db.DBTX }

func (s Store) Create(ctx context.Context, email, hash string) (uuid.UUID, error) {
	var id uuid.UUID
	err := s.DB.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		email, hash,
	).Scan(&id)
	if err != nil {
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == sqlstateUniqueViolation {
			return uuid.UUID{}, ErrEmailInUse
		}
		return uuid.UUID{}, err
	}
	return id, nil
}

// Uses the unique index on email.
func (s Store) GetByEmail(ctx context.Context, email string) (User, error) {
	rows, _ := s.DB.Query(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`,
		email,
	)
	u, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[User])
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}
