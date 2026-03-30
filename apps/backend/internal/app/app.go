package app

import (
	"saferoute-backend/config"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

type RouteRegistrar func(fiber.Router)

func New(cfg config.Config, registrars ...RouteRegistrar) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName: cfg.AppName,
	})

	app.Use(recover.New())
	app.Use(logger.New())

	api := app.Group("/api/v1")
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":      "ok",
			"service":     cfg.AppName,
			"environment": cfg.Environment,
		})
	})

	for _, register := range registrars {
		if register != nil {
			register(api)
		}
	}

	return app
}
