package httpserver

import "github.com/gofiber/fiber/v2"

const apiPrefix = "/api/v1"

type RouteRegistrar interface {
	RegisterRoutes(router fiber.Router)
}

type PublicRoutes interface {
	PublicPathPrefixes() []string
}

func NewRouter(app *fiber.App, admin AdminCredentials, registrars ...RouteRegistrar) {
	var publicPrefixes []string
	for _, r := range registrars {
		if p, ok := r.(PublicRoutes); ok {
			for _, prefix := range p.PublicPathPrefixes() {
				publicPrefixes = append(publicPrefixes, apiPrefix+prefix)
			}
		}
	}

	api := app.Group(apiPrefix, adminAuth(admin, publicPrefixes))
	for _, r := range registrars {
		r.RegisterRoutes(api)
	}

	app.Get("/healthz", func(c *fiber.Ctx) error {
		return c.SendString("ok")
	})
}
