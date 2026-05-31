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
	"github.com/amin-mir/enma/internal/handler"
	"github.com/amin-mir/enma/internal/middleware"
	"github.com/amin-mir/enma/internal/postgres"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.Load()

	pool := postgres.NewPool(cfg.PostgresURL)
	defer pool.Close()

	pg := postgres.New(pool)

	a := auth.New(pg, log.Logger, auth.Config{
		Secret:               cfg.JWTSecret,
		AccessTokenDuration:  cfg.AccessTokenDuration,
		RefreshTokenDuration: cfg.RefreshTokenDuration,
	})

	app := fiber.New(fiber.Config{
		CaseSensitive: true,
		StrictRouting: true,
	})

	app.Use(recover.New())
	app.Use(fiberlogger.New())
	app.Use(cors.New())

	api := app.Group("/api/v1")
	handler.NewAuthHandler(a).Mount(api)
	handler.NewJournalHandler(pg).Mount(api, middleware.Protected(a))

	if err := app.Listen(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
