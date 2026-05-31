package router

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/rs/zerolog"

	"github.com/amin-mir/enma/internal/auth"
	"github.com/amin-mir/enma/internal/config"
	"github.com/amin-mir/enma/internal/handler"
	"github.com/amin-mir/enma/internal/middleware"
	"github.com/amin-mir/enma/internal/postgres"
)

func Setup(app *fiber.App, pg *postgres.Postgres, log zerolog.Logger, cfg *config.Config) {
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New())

	a := auth.New(pg, log, auth.Config{
		Secret:               cfg.JWTSecret,
		AccessTokenDuration:  cfg.AccessTokenDuration,
		RefreshTokenDuration: cfg.RefreshTokenDuration,
	})
	authHandler := handler.NewAuthHandler(a)
	journalHandler := handler.NewJournalHandler(pg)

	api := app.Group("/api/v1")

	authRoutes := api.Group("/auth")
	authRoutes.Post("/register", authHandler.Register)
	authRoutes.Post("/login", authHandler.Login)
	authRoutes.Post("/refresh", authHandler.Refresh)
	authRoutes.Post("/logout", authHandler.Logout)

	journals := api.Group("/journals", middleware.Protected(a))
	journals.Post("/", journalHandler.Create)
	journals.Get("/", journalHandler.List)
	journals.Get("/:id", journalHandler.Get)
	journals.Put("/:id", journalHandler.Update)
}
