package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func SetupPostgres(t *testing.T, migrationScript string) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pgc, err := tcpostgres.Run(ctx, "postgres:18-alpine",
		tcpostgres.WithInitScripts(migrationScript),
		tcpostgres.WithDatabase("enma"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	testcontainers.CleanupContainer(t, pgc)

	connStr, err := pgc.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pool
}

// Request sends an HTTP request to the Fiber app and decodes the response body
// into T. Returns the status code, decoded value, and any error.
//
// Pass nil for body on requests with no payload (e.g. GET).
//
// Use struct{} as T when you only need the status code and don't care about
// the response body. This also handles empty-body responses (e.g. 204 No
// Content) where the decoder would otherwise return io.EOF.
func Request[T any](app *fiber.App, method, url string, body any) (int, T, error) {
	var zero T

	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return 0, zero, err
		}
		r = bytes.NewReader(b)
	}

	req := httptest.NewRequest(method, url, r)
	if r != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := app.Test(req)
	if err != nil {
		return 0, zero, err
	}
	defer resp.Body.Close()

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil && !errors.Is(err, io.EOF) {
		return resp.StatusCode, zero, err
	}

	return resp.StatusCode, result, nil
}
