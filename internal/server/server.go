package server

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

func Launch(port int, indent int, char string, fn func(s string, indent int, char string) (string, error)) error {
	app := fiber.New()

	// GET / always serves the pristine document. The formatted result is
	// returned directly from POST /format instead of being parked in a
	// package-level variable and picked up by a redirect, so one visitor's
	// config is never served to another.
	app.Get("/", func(c fiber.Ctx) error {
		c.Type("html")
		return c.Send([]byte(PAGE_DOCUMENT))
	})

	app.Post("/format", func(c fiber.Ctx) error {
		c.Type("html")
		return c.Send([]byte(formatPage(c.FormValue("code"), indent, char, fn)))
	})

	app.Get("/base.css", func(c fiber.Ctx) error {
		c.Type("css")
		return c.Send(CACHE_STYLESHEET)
	})

	app.Get("/base.js", func(c fiber.Ctx) error {
		c.Type("js")
		return c.Send(CACHE_SCRIPT)
	})

	return app.Listen(fmt.Sprintf(":%d", port), fiber.ListenConfig{DisableStartupMessage: true})
}
