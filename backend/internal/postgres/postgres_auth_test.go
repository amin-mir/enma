package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCreateRefreshToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID, err := pg.CreateUser(ctx, "auth@example.com", "hash")
	require.NoError(t, err)

	err = pg.CreateRefreshToken(ctx, userID, "token_hash", time.Now().Add(30*24*time.Hour))
	require.NoError(t, err)
}

func TestRotateRefreshToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID, err := pg.CreateUser(ctx, "rotate@example.com", "hash")
	require.NoError(t, err)

	err = pg.CreateRefreshToken(ctx, userID, "old_hash", time.Now().Add(30*24*time.Hour))
	require.NoError(t, err)

	returnedUserID, err := pg.RotateRefreshToken(ctx, "old_hash", "new_hash", time.Now().Add(30*24*time.Hour))
	require.NoError(t, err)
	require.Equal(t, userID, returnedUserID)

	// Old token must be gone.
	_, err = pg.RotateRefreshToken(ctx, "old_hash", "another_hash", time.Now().Add(30*24*time.Hour))
	require.ErrorIs(t, err, ErrNotFound)
}

func TestRotateRefreshToken_InvalidToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	_, err := pg.RotateRefreshToken(ctx, "nonexistent_hash", "new_hash", time.Now().Add(30*24*time.Hour))
	require.ErrorIs(t, err, ErrNotFound)
}

func TestRotateRefreshToken_ExpiredToken(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID, err := pg.CreateUser(ctx, "expired@example.com", "hash")
	require.NoError(t, err)

	err = pg.CreateRefreshToken(ctx, userID, "expired_hash", time.Now().Add(-time.Hour))
	require.NoError(t, err)

	_, err = pg.RotateRefreshToken(ctx, "expired_hash", "new_hash", time.Now().Add(30*24*time.Hour))
	require.ErrorIs(t, err, ErrNotFound)
}

func TestDeleteRefreshTokenByHash(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID, err := pg.CreateUser(ctx, "logout@example.com", "hash")
	require.NoError(t, err)

	err = pg.CreateRefreshToken(ctx, userID, "delete_me", time.Now().Add(30*24*time.Hour))
	require.NoError(t, err)

	err = pg.DeleteRefreshTokenByHash(ctx, "delete_me")
	require.NoError(t, err)

	_, err = pg.RotateRefreshToken(ctx, "delete_me", "new_hash", time.Now().Add(30*24*time.Hour))
	require.ErrorIs(t, err, ErrNotFound)
}
