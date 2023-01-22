package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/labstack/echo/v4"
)

// ContextTimeoutConfig defines the config for ContextTimeout middleware.
type ContextTimeoutConfig struct {
	// Skipper defines a function to skip middleware.
	Skipper Skipper

	// ErrorHandler is a function when error aries in middeware execution.
	ErrorHandler func(err error, c echo.Context) error

	// Timeout configures a timeout for the middleware, defaults to 0 for no timeout
	Timeout time.Duration
}

var (
	// DefaultContextTimeoutErrorHandler is default error handler of ContextTimeout middleware.
	DefaultContextTimeoutErrorHandler = func(err error, c echo.Context) error {
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return echo.ErrServiceUnavailable
			}
			return err
		}
		return nil
	}
)

// ContextTimeout returns a middleware which returns error (503 Service Unavailable error) to client
// when underlying method returns context.DeadlineExceeded error.
func ContextTimeout(timeout time.Duration) echo.MiddlewareFunc {
	config := ContextTimeoutConfig{
		Skipper:      DefaultSkipper,
		ErrorHandler: DefaultContextTimeoutErrorHandler,
		Timeout:      timeout,
	}
	return ContextTimeoutWithConfig(config)
}

// ContextTimeoutWithConfig returns a Timeout middleware with config.
func ContextTimeoutWithConfig(config ContextTimeoutConfig) echo.MiddlewareFunc {
	return config.ToMiddleware()
}

// ToMiddleware converts Config to middleware.
func (config ContextTimeoutConfig) ToMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if (config.Skipper != nil && config.Skipper(c)) || config.Timeout == 0 {
				return next(c)
			}

			timeoutContext, cancel := context.WithTimeout(c.Request().Context(), config.Timeout)
			defer cancel()

			c.SetRequest(c.Request().WithContext(timeoutContext))

			err := next(c)
			if err != nil {
				if config.ErrorHandler != nil {
					return config.ErrorHandler(err, c)
				} else {
					return DefaultContextTimeoutErrorHandler(err, c)
				}
			}
			return nil
		}
	}
}
