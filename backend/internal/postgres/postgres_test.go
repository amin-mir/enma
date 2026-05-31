package postgres

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func newTestDB(t *testing.T) *Postgres {
	t.Helper()

	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:18-alpine",
		tcpostgres.WithInitScripts(filepath.Join("..", "..", "migrations", "000001_initial_schema.up.sql")),
		tcpostgres.WithDatabase("enma"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	testcontainers.CleanupContainer(t, pgContainer)
	require.NoError(t, err)

	pgURL, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, pgURL)
	require.NoError(t, err)

	return New(pool)
}
