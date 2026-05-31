package postgres

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	id, err := pg.CreateUser(ctx, "test@example.com", "hashed_password")
	require.NoError(t, err)
	require.NotEmpty(t, id)
}

func TestCreateUser_DuplicateEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	_, err := pg.CreateUser(ctx, "dupe@example.com", "hashed_password")
	require.NoError(t, err)

	_, err = pg.CreateUser(ctx, "dupe@example.com", "other_hash")
	require.ErrorIs(t, err, ErrUniqueViolation)
}

func TestGetUserByEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	_, err := pg.CreateUser(ctx, "get@example.com", "hashed_password")
	require.NoError(t, err)

	user, err := pg.GetUserByEmail(ctx, "get@example.com")
	require.NoError(t, err)
	require.Equal(t, "get@example.com", user.Email)
	require.Equal(t, "hashed_password", user.PasswordHash)
	require.NotEmpty(t, user.ID)
	require.NotZero(t, user.CreatedAt)
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	_, err := pg.GetUserByEmail(ctx, "nobody@example.com")
	require.ErrorIs(t, err, ErrNotFound)
}
