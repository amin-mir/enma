package postgres

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestCreateJournalEntry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID, err := pg.CreateUser(ctx, "journal@example.com", "hash")
	require.NoError(t, err)

	res, err := pg.CreateJournalEntry(ctx, userID, "my first entry")
	require.NoError(t, err)
	require.NotEmpty(t, res.ID)
	require.NotZero(t, res.CreatedAt)
}

func TestListJournalEntries(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID, err := pg.CreateUser(ctx, "list@example.com", "hash")
	require.NoError(t, err)

	contents := []string{"first", "second", "third"}
	for _, c := range contents {
		_, err := pg.CreateJournalEntry(ctx, userID, c)
		require.NoError(t, err)
	}

	entries, err := pg.ListJournalEntries(ctx, userID)
	require.NoError(t, err)
	require.Len(t, entries, 3)

	// Verify descending order by created_at.
	for i := 1; i < len(entries); i++ {
		require.False(t, entries[i].CreatedAt.After(entries[i-1].CreatedAt))
	}
}

func TestListJournalEntries_Empty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID, err := pg.CreateUser(ctx, "empty@example.com", "hash")
	require.NoError(t, err)

	entries, err := pg.ListJournalEntries(ctx, userID)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestListJournalEntries_IsolatedByUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID1, err := pg.CreateUser(ctx, "user1@example.com", "hash")
	require.NoError(t, err)
	userID2, err := pg.CreateUser(ctx, "user2@example.com", "hash")
	require.NoError(t, err)

	_, err = pg.CreateJournalEntry(ctx, userID1, "user1 entry")
	require.NoError(t, err)

	entries, err := pg.ListJournalEntries(ctx, userID2)
	require.NoError(t, err)
	require.Empty(t, entries)
}

func TestGetJournalEntry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID, err := pg.CreateUser(ctx, "get-entry@example.com", "hash")
	require.NoError(t, err)

	res, err := pg.CreateJournalEntry(ctx, userID, "hello world")
	require.NoError(t, err)

	entry, err := pg.GetJournalEntry(ctx, res.ID, userID)
	require.NoError(t, err)
	require.Equal(t, res.ID, entry.ID)
	require.Equal(t, userID, entry.UserID)
	require.Equal(t, "hello world", entry.Content)
	require.NotZero(t, entry.CreatedAt)
	require.NotZero(t, entry.UpdatedAt)
}

func TestGetJournalEntry_NotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID, err := pg.CreateUser(ctx, "get-notfound@example.com", "hash")
	require.NoError(t, err)

	_, err = pg.GetJournalEntry(ctx, uuid.New(), userID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestGetJournalEntry_WrongUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID1, err := pg.CreateUser(ctx, "owner@example.com", "hash")
	require.NoError(t, err)
	userID2, err := pg.CreateUser(ctx, "other@example.com", "hash")
	require.NoError(t, err)

	res, err := pg.CreateJournalEntry(ctx, userID1, "private entry")
	require.NoError(t, err)

	_, err = pg.GetJournalEntry(ctx, res.ID, userID2)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateJournalEntry(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID, err := pg.CreateUser(ctx, "update@example.com", "hash")
	require.NoError(t, err)

	res, err := pg.CreateJournalEntry(ctx, userID, "original content")
	require.NoError(t, err)

	err = pg.UpdateJournalEntry(ctx, res.ID, userID, "updated content")
	require.NoError(t, err)

	entry, err := pg.GetJournalEntry(ctx, res.ID, userID)
	require.NoError(t, err)
	require.Equal(t, "updated content", entry.Content)
}

func TestUpdateJournalEntry_NotFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID, err := pg.CreateUser(ctx, "update-notfound@example.com", "hash")
	require.NoError(t, err)

	err = pg.UpdateJournalEntry(ctx, uuid.New(), userID, "content")
	require.ErrorIs(t, err, ErrNotFound)
}

func TestUpdateJournalEntry_WrongUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	pg := newTestDB(t)

	userID1, err := pg.CreateUser(ctx, "update-owner@example.com", "hash")
	require.NoError(t, err)
	userID2, err := pg.CreateUser(ctx, "update-other@example.com", "hash")
	require.NoError(t, err)

	res, err := pg.CreateJournalEntry(ctx, userID1, "original")
	require.NoError(t, err)

	err = pg.UpdateJournalEntry(ctx, res.ID, userID2, "hacked")
	require.ErrorIs(t, err, ErrNotFound)
}
