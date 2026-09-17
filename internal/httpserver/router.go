package httpserver

import "github.com/gofiber/fiber/v2"

type RouteRegistrar interface {
	RegisterRoutes(router fiber.Router)
}

func NewRouter(app *fiber.App, registrars ...RouteRegistrar) {
	api := app.Group("/api/v1")
	for _, r := range registrars {
		r.RegisterRoutes(api)
	}

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
}
