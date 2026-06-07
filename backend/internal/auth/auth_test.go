package auth

import (
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/amin-mir/enma/internal/testutil"
	"github.com/amin-mir/enma/internal/user"
)

var testCfg = Config{
	Secret:               "test-secret",
	AccessTokenDuration:  15 * time.Minute,
	RefreshTokenDuration: 30 * 24 * time.Hour,
}

func newTestApp(pool *pgxpool.Pool) *fiber.App {
	app := fiber.New()
	New(pool, zerolog.Nop(), testCfg).MountHTTPHandlers(app)
	return app
}

func registerNewUser(t *testing.T, app *fiber.App, email, pass string) tokenResp {
	t.Helper()
	status, tokens, err := testutil.Request[tokenResp](
		app, http.MethodPost, "/auth/register",
		registerReq{Email: email, Password: pass},
	)
	require.NoError(t, err)
	require.Equal(t, fiber.StatusCreated, status)
	return tokens
}

func TestRegister(t *testing.T) {
	t.Parallel()
	pool := testutil.SetupPostgres(t, filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql"))
	app := newTestApp(pool)

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		status, got, err := testutil.Request[tokenResp](
			app, http.MethodPost, "/auth/register",
			registerReq{Email: uuid.New().String() + "@test.com", Password: "password123"},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusCreated, status)
		require.NotEmpty(t, got.AccessToken)
		require.NotEmpty(t, got.RefreshToken)
	})

	t.Run("missing_fields", func(t *testing.T) {
		t.Parallel()
		status, _, err := testutil.Request[struct{}](
			app, http.MethodPost, "/auth/register",
			registerReq{Email: "a@b.com"},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusBadRequest, status)
	})

	t.Run("duplicate_email", func(t *testing.T) {
		t.Parallel()
		email := uuid.New().String() + "@test.com"

		for i := range 2 {
			status, _, err := testutil.Request[struct{}](
				app, http.MethodPost, "/auth/register",
				registerReq{Email: email, Password: "pass"},
			)
			require.NoError(t, err)

			if i == 0 {
				require.Equal(t, fiber.StatusCreated, status)
			} else {
				require.Equal(t, fiber.StatusConflict, status)
			}
		}
	})
}

func TestLogin(t *testing.T) {
	t.Parallel()
	pool := testutil.SetupPostgres(t, filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql"))
	app := newTestApp(pool)

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		email := uuid.New().String() + "@test.com"
		registerNewUser(t, app, email, "pass")

		status, got, err := testutil.Request[tokenResp](
			app, http.MethodPost, "/auth/login",
			loginReq{Email: email, Password: "pass"},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, status)
		require.NotEmpty(t, got.AccessToken)
		require.NotEmpty(t, got.RefreshToken)
	})

	t.Run("wrong_password", func(t *testing.T) {
		t.Parallel()
		email := uuid.New().String() + "@test.com"
		registerNewUser(t, app, email, "correct")

		status, _, err := testutil.Request[struct{}](
			app, http.MethodPost, "/auth/login",
			loginReq{Email: email, Password: "wrong"},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, status)
	})

	t.Run("unknown_email", func(t *testing.T) {
		t.Parallel()
		status, _, err := testutil.Request[struct{}](
			app, http.MethodPost, "/auth/login",
			loginReq{Email: uuid.New().String() + "@test.com", Password: "pass"},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, status)
	})

	t.Run("malformed_hash", func(t *testing.T) {
		t.Parallel()
		email := uuid.New().String() + "@test.com"
		s := user.Store{DB: pool}
		_, err := s.Create(t.Context(), email, "not-a-valid-hash")
		require.NoError(t, err)

		status, _, err := testutil.Request[struct{}](
			app, http.MethodPost, "/auth/login",
			loginReq{Email: email, Password: "pass"},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, status)
	})
}

func TestRefresh(t *testing.T) {
	t.Parallel()
	pool := testutil.SetupPostgres(t, filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql"))
	app := newTestApp(pool)

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		tokens := registerNewUser(t, app, uuid.New().String()+"@test.com", "pass")

		status, got, err := testutil.Request[tokenResp](
			app, http.MethodPost, "/auth/refresh",
			refreshReq{RefreshToken: tokens.RefreshToken},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusOK, status)
		require.NotEmpty(t, got.AccessToken)
		require.NotEmpty(t, got.RefreshToken)
	})

	t.Run("invalid_token", func(t *testing.T) {
		t.Parallel()
		status, _, err := testutil.Request[struct{}](
			app, http.MethodPost, "/auth/refresh",
			refreshReq{RefreshToken: "invalid-token"},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, status)
	})
}

func TestLogout(t *testing.T) {
	t.Parallel()
	pool := testutil.SetupPostgres(t, filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql"))
	app := newTestApp(pool)

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		tokens := registerNewUser(t, app, uuid.New().String()+"@test.com", "pass")

		status, _, err := testutil.Request[struct{}](
			app, http.MethodPost, "/auth/logout",
			logoutReq{RefreshToken: tokens.RefreshToken},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusNoContent, status)
	})

	t.Run("token_reuse", func(t *testing.T) {
		t.Parallel()
		tokens := registerNewUser(t, app, uuid.New().String()+"@test.com", "pass")

		status, _, err := testutil.Request[struct{}](
			app, http.MethodPost, "/auth/logout",
			logoutReq{RefreshToken: tokens.RefreshToken},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusNoContent, status)

		status, _, err = testutil.Request[struct{}](
			app, http.MethodPost, "/auth/refresh",
			refreshReq{RefreshToken: tokens.RefreshToken},
		)
		require.NoError(t, err)
		require.Equal(t, fiber.StatusUnauthorized, status)
	})
}

func TestValidateAccessToken(t *testing.T) {
	t.Parallel()
	a := New(nil, zerolog.Nop(), testCfg)

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		tokenStr, err := a.newAccessToken(uuid.New())
		require.NoError(t, err)

		claims, err := a.validateAccessToken(tokenStr)
		require.NoError(t, err)
		require.NotEmpty(t, claims.UserID)
	})

	t.Run("invalid", func(t *testing.T) {
		t.Parallel()
		_, err := a.validateAccessToken("not.a.valid.token")
		require.ErrorIs(t, err, ErrInvalidToken)
	})

	t.Run("wrong_secret", func(t *testing.T) {
		t.Parallel()
		other := New(nil, zerolog.Nop(), Config{Secret: "other-secret", AccessTokenDuration: 15 * time.Minute})
		tokenStr, err := other.newAccessToken(uuid.New())
		require.NoError(t, err)

		_, err = a.validateAccessToken(tokenStr)
		require.ErrorIs(t, err, ErrInvalidToken)
	})
}
