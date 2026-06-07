package journal

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound = errors.New("journal entry not found")
	ErrConflict = errors.New("version conflict")
)

type JournalEntry struct {
	ID        uuid.UUID `db:"id"         json:"id"`
	UserID    uuid.UUID `db:"user_id"    json:"user_id"`
	Content   string    `db:"content"    json:"content"`
	Version   int32     `db:"version"    json:"version"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type JournalService struct{ pool *pgxpool.Pool }

func New(pool *pgxpool.Pool) *JournalService {
	return &JournalService{pool: pool}
}
