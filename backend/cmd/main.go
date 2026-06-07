package main

import (
	"fmt"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	fiberlogger "github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/amin-mir/enma/internal/auth"
	"github.com/amin-mir/enma/internal/config"
	"github.com/amin-mir/enma/internal/db"
	"github.com/amin-mir/enma/internal/journal"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.Load()

	pool := db.NewPool(cfg.PostgresURL)
	defer pool.Close()

	authService := auth.New(pool, log.Logger, auth.Config{
		Secret:               cfg.JWTSecret,
		AccessTokenDuration:  cfg.AccessTokenDuration,
		RefreshTokenDuration: cfg.RefreshTokenDuration,
	})
	journalService := journal.New(pool)

	app := fiber.New(fiber.Config{
		CaseSensitive: true,
		StrictRouting: true,
	})

	app.Use(recover.New())
	app.Use(fiberlogger.New())
	app.Use(cors.New())

	protected := authService.JWTMiddleware()

	api := app.Group("/api/v1")
	authService.MountHTTPHandlers(api)
	journalService.MountHTTPHandlers(api, protected)

	if err := app.Listen(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
