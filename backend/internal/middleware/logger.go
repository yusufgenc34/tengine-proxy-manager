package middleware

import (
	"log"
	"time"

	"github.com/labstack/echo/v4"
)

func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			log.Printf("[%s] %s %s %d %s",
				c.Request().Method,
				c.Request().URL.Path,
				c.RealIP(),
				c.Response().Status,
				time.Since(start),
			)

			return err
		}
	}
}
