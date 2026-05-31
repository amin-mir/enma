package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/amin-mir/enma/internal/model"
)

func (pg *Postgres) CreateUser(ctx context.Context, email, hash string) (uuid.UUID, error) {
	var id uuid.UUID
	err := pg.pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		email, hash,
	).Scan(&id)
	if err != nil {
		if isUniqueViolation(err) {
			return uuid.UUID{}, ErrUniqueViolation
		}
		return uuid.UUID{}, err
	}
	return id, nil
}

// Uses the unique index on email (implicit from UNIQUE constraint).
func (pg *Postgres) GetUserByEmail(ctx context.Context, email string) (model.User, error) {
	rows, _ := pg.pool.Query(ctx,
		`SELECT id, email, password_hash, created_at FROM users WHERE email = $1`,
		email,
	)
	user, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[model.User])
	if errors.Is(err, pgx.ErrNoRows) {
		return model.User{}, ErrNotFound
	}
	return user, err
}
