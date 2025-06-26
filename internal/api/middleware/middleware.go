package middleware

import (
	"bytes"
	"io"
	"log/slog"

	"github.com/arnokay/arnobot-shared/apperror"
	"github.com/arnokay/arnobot-shared/applog"
	"github.com/arnokay/arnobot-shared/middlewares"
	"github.com/labstack/echo/v4"
	"github.com/scorfly/gokick"
)

type Middlewares struct {
	logger *slog.Logger

	AuthMiddlewares *middlewares.AuthMiddlewares
}

func New(
	authMiddlewares *middlewares.AuthMiddlewares,
) *Middlewares {
	logger := applog.NewServiceLogger("app-middleware")

	return &Middlewares{
		logger:          logger,
		AuthMiddlewares: authMiddlewares,
	}
}

func (m *Middlewares) VerifyKickWebhook(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// TODO: handle length of request body
		body, err := io.ReadAll(c.Request().Body)
		if err != nil {
			m.logger.ErrorContext(c.Request().Context(), "cannot read body", "err", err)
			return apperror.ErrUnauthorized
		}
		c.Request().Body.Close()
		c.Request().Body = io.NopCloser(bytes.NewReader(body))

		if !gokick.ValidateEvent(c.Request().Header, body) {
			m.logger.ErrorContext(c.Request().Context(), "unverified attempt to access webhook")
			return apperror.ErrUnauthorized
		}

		return next(c)
	}
}
