package main

import (
	"fmt"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/amin-mir/enma/internal/config"
	"github.com/amin-mir/enma/internal/postgres"
	"github.com/amin-mir/enma/internal/router"
)

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	cfg := config.Load()

	pool := postgres.NewPool(cfg.PostgresURL)
	defer pool.Close()

	pg := postgres.New(pool)

	app := fiber.New(fiber.Config{
		CaseSensitive: true,
		StrictRouting: true,
	})

	router.Setup(app, pg, log.Logger, cfg)

	if err := app.Listen(fmt.Sprintf(":%s", cfg.Port)); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
