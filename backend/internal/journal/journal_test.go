package journal

import (
	"fmt"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"github.com/amin-mir/enma/internal/testutil"
)

func seedUser(t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := pool.QueryRow(t.Context(),
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		uuid.New().String()+"@test.com", "hash",
	).Scan(&id)
	require.NoError(t, err)
	return id
}

func newTestApp(pool *pgxpool.Pool, userID uuid.UUID) *fiber.App {
	app := fiber.New()
	injectUser := func(c fiber.Ctx) error {
		c.Locals("user_id", userID.String())
		return c.Next()
	}
	New(pool).MountHTTPHandlers(app, injectUser)
	return app
}

func TestCreate(t *testing.T) {
	t.Parallel()
	pool := testutil.SetupPostgres(t, filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql"))

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(pool, seedUser(t, pool))

		status, got, err := testutil.Request[createResp](
			app, http.MethodPost, "/journals/",
			createReq{Content: "hello world"},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusCreated, status)
		require.NotEqual(t, uuid.UUID{}, got.ID)
		require.NotZero(t, got.CreatedAt)
		require.Equal(t, int32(1), got.Version)
	})

	t.Run("missing_content", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(pool, seedUser(t, pool))

		status, _, err := testutil.Request[struct{}](
			app, http.MethodPost, "/journals/",
			createReq{},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusBadRequest, status)
	})
}

func TestList(t *testing.T) {
	t.Parallel()
	pool := testutil.SetupPostgres(t, filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql"))

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(pool, seedUser(t, pool))

		for _, content := range []string{"first", "second", "third"} {
			status, _, err := testutil.Request[struct{}](
				app, http.MethodPost, "/journals/",
				createReq{Content: content},
			)
			require.NoError(t, err)
			require.Equal(t, fiber.StatusCreated, status)
		}

		status, got, err := testutil.Request[[]JournalEntry](app, http.MethodGet, "/journals/", nil)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, status)
		require.Len(t, got, 3)
		for i := 1; i < len(got); i++ {
			require.False(t, got[i].CreatedAt.After(got[i-1].CreatedAt))
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(pool, seedUser(t, pool))

		status, got, err := testutil.Request[[]JournalEntry](app, http.MethodGet, "/journals/", nil)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, status)
		require.Empty(t, got)
	})

	t.Run("isolated_by_user", func(t *testing.T) {
		t.Parallel()
		app1 := newTestApp(pool, seedUser(t, pool))
		status, _, err := testutil.Request[struct{}](
			app1, http.MethodPost, "/journals/",
			createReq{Content: "user1 entry"},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusCreated, status)

		app2 := newTestApp(pool, seedUser(t, pool))
		status, got, err := testutil.Request[[]JournalEntry](app2, http.MethodGet, "/journals/", nil)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, status)
		require.Empty(t, got)
	})
}

func TestGet(t *testing.T) {
	t.Parallel()
	pool := testutil.SetupPostgres(t, filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql"))

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(pool, seedUser(t, pool))

		_, created, err := testutil.Request[createResp](
			app, http.MethodPost, "/journals/",
			createReq{Content: "hello"},
		)
		require.NoError(t, err)

		status, got, err := testutil.Request[JournalEntry](
			app, http.MethodGet, fmt.Sprintf("/journals/%s", created.ID), nil,
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, status)
		require.Equal(t, created.ID, got.ID)
		require.Equal(t, "hello", got.Content)
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(pool, seedUser(t, pool))

		status, _, err := testutil.Request[struct{}](
			app, http.MethodGet, fmt.Sprintf("/journals/%s", uuid.New()), nil,
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusNotFound, status)
	})

	t.Run("wrong_user", func(t *testing.T) {
		t.Parallel()
		ownerApp := newTestApp(pool, seedUser(t, pool))
		_, created, err := testutil.Request[createResp](
			ownerApp, http.MethodPost, "/journals/",
			createReq{Content: "private"},
		)
		require.NoError(t, err)

		otherApp := newTestApp(pool, seedUser(t, pool))
		status, _, err := testutil.Request[struct{}](
			otherApp, http.MethodGet, fmt.Sprintf("/journals/%s", created.ID), nil,
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusNotFound, status)
	})
}

func TestUpdate(t *testing.T) {
	t.Parallel()
	pool := testutil.SetupPostgres(t, filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql"))

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(pool, seedUser(t, pool))

		_, created, err := testutil.Request[createResp](
			app, http.MethodPost, "/journals/",
			createReq{Content: "original"},
		)
		require.NoError(t, err)

		status, got, err := testutil.Request[updateResp](
			app, http.MethodPut, fmt.Sprintf("/journals/%s", created.ID),
			updateReq{Content: "updated", Version: created.Version},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, status)
		require.Equal(t, created.Version+1, got.Version)
	})

	t.Run("conflict", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(pool, seedUser(t, pool))

		_, created, err := testutil.Request[createResp](
			app, http.MethodPost, "/journals/",
			createReq{Content: "original"},
		)
		require.NoError(t, err)

		_, _, err = testutil.Request[struct{}](
			app, http.MethodPut, fmt.Sprintf("/journals/%s", created.ID),
			updateReq{Content: "first", Version: created.Version},
		)
		require.NoError(t, err)

		status, _, err := testutil.Request[struct{}](
			app, http.MethodPut, fmt.Sprintf("/journals/%s", created.ID),
			updateReq{Content: "stale", Version: created.Version},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusConflict, status)
	})

	t.Run("not_found", func(t *testing.T) {
		t.Parallel()
		app := newTestApp(pool, seedUser(t, pool))

		status, _, err := testutil.Request[struct{}](
			app, http.MethodPut, fmt.Sprintf("/journals/%s", uuid.New()),
			updateReq{Content: "x", Version: 1},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusConflict, status)
	})
}
