package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"

	"github.com/amin-mir/enma/internal/model"
)

const sqlstateUniqueViolation = "23505"

var (
	ErrNotFound        = errors.New("not found")
	ErrUniqueViolation = errors.New("unique violation")
	ErrConflict        = errors.New("conflict")
)

func isUniqueViolation(err error) bool {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgErr.Code == sqlstateUniqueViolation
}

//go:generate go tool mockgen -source=postgres.go -destination=mocks/mock_db.go -package=mocks DB
type DB interface {
	// CreateUser returns ErrUniqueViolation if the email is already taken.
	CreateUser(ctx context.Context, email, hash string) (uuid.UUID, error)
	// GetUserByEmail returns ErrNotFound if no user exists with that email.
	GetUserByEmail(ctx context.Context, email string) (model.User, error)

	CreateRefreshToken(ctx context.Context, userID uuid.UUID, tokenHash string, expiresAt time.Time) error
	// RotateRefreshToken returns ErrNotFound if the token doesn't exist or has expired.
	RotateRefreshToken(ctx context.Context, oldTokenHash, newTokenHash string, expiresAt time.Time) (uuid.UUID, error)
	DeleteRefreshTokenByHash(ctx context.Context, tokenHash string) error

	CreateJournalEntry(ctx context.Context, userID uuid.UUID, content string) (CreateJournalEntryResult, error)
	ListJournalEntries(ctx context.Context, userID uuid.UUID) ([]model.JournalEntry, error)
	// GetJournalEntry returns ErrNotFound if no entry exists for that id and user.
	GetJournalEntry(ctx context.Context, entryID, userID uuid.UUID) (model.JournalEntry, error)
	// UpdateJournalEntry returns ErrConflict if the version does not match (stale write).
	UpdateJournalEntry(ctx context.Context, entryID, userID uuid.UUID, content string, version int32) (int32, error)
}

type Postgres struct {
	pool *pgxpool.Pool
}

func NewPool(url string) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create postgres pool")
	}
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to ping postgres")
	}
	return pool
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}
