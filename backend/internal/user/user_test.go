package user

import (
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/amin-mir/enma/internal/testutil"
)

func TestCreate(t *testing.T) {
	t.Parallel()
	pool := testutil.SetupPostgres(t, filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql"))

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		s := Store{DB: pool}
		id, err := s.Create(t.Context(), uuid.New().String()+"@test.com", "hash")
		require.NoError(t, err)
		require.NotEqual(t, uuid.UUID{}, id)
	})

	t.Run("duplicate_email", func(t *testing.T) {
		t.Parallel()
		email := uuid.New().String() + "@test.com"
		s := Store{DB: pool}
		_, err := s.Create(t.Context(), email, "hash")
		require.NoError(t, err)
		_, err = s.Create(t.Context(), email, "hash")
		require.ErrorIs(t, err, ErrEmailInUse)
	})
}

func TestGetByEmail(t *testing.T) {
	t.Parallel()
	pool := testutil.SetupPostgres(t, filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql"))

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		email := uuid.New().String() + "@test.com"
		s := Store{DB: pool}
		_, err := s.Create(t.Context(), email, "myhash")
		require.NoError(t, err)

		u, err := s.GetByEmail(t.Context(), email)
		require.NoError(t, err)
		require.Equal(t, email, u.Email)
		require.Equal(t, "myhash", u.PasswordHash)
		require.NotEqual(t, uuid.UUID{}, u.ID)
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()
		s := Store{DB: pool}
		_, err := s.GetByEmail(t.Context(), uuid.New().String()+"@test.com")
		require.ErrorIs(t, err, ErrNotFound)
	})
}
