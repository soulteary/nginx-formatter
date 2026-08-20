package server

import (
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v3"
)

func Launch(port int, indent int, char string, fn func(s string, indent int, char string) (string, error)) error {
	app := fiber.New()

	app.Get("/", func(c fiber.Ctx) error {
		c.Type("html")
		return c.Send([]byte(getDocCache()))
	})

	app.Post("/format", func(c fiber.Ctx) error {
		code := c.FormValue("code")
		updateDocCache(code, indent, char, fn)
		return c.Redirect().Status(http.StatusFound).To("/")
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
