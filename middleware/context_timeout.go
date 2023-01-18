package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/labstack/echo/v4"
)

// Create handler that checks for context deadline and runs actual task in separate coroutine
// Note: separate coroutine may not be even if you do not want to process continue executing and
// just want to stop long-running handler to stop and you are using "context aware" methods (ala db queries with ctx)
// 	e.GET("/", func(c echo.Context) error {
//
//		doneCh := make(chan error)
//		go func(ctx context.Context) {
//			doneCh <- myPossiblyLongRunningBackgroundTaskWithCtx(ctx)
//		}(c.Request().Context())
//
//		select { // wait for task to finish or context to timeout/cancelled
//		case err := <-doneCh:
//			if err != nil {
//				return err
//			}
//			return c.String(http.StatusOK, "OK")
//		case <-c.Request().Context().Done():
//			if c.Request().Context().Err() == context.DeadlineExceeded {
//				return c.String(http.StatusServiceUnavailable, "timeout")
//			}
//			return c.Request().Context().Err()
//		}
//
//	})
//

// ContextTimeoutConfig defines the config for ContextTimeout middleware.
type ContextTimeoutConfig struct {
	// Skipper defines a function to skip middleware.
	Skipper Skipper

	// Timeout configures a timeout for the middleware, defaults to 0 for no timeout
	// NOTE: when difference between timeout duration and handler execution time is almost the same (in range of 100microseconds)
	// the result of timeout does not seem to be reliable - could respond timeout, could respond handler output
	// difference over 500microseconds (0.5millisecond) response seems to be reliable
	Timeout time.Duration
}

var (
	// DefaultContextTimeoutConfig is the default ContextTimeoutConfig middleware config.
	DefaultContextTimeoutConfig = ContextTimeoutConfig{
		Skipper: DefaultSkipper,
		Timeout: 0,
	}
)

// ContextTimeout returns a middleware which returns error (503 Service Unavailable error) to client immediately when handler
// call runs for longer than its time limit. NB: timeout does not stop handler execution.
func ContextTimeout() echo.MiddlewareFunc {
	return ContextTimeoutWithConfig(DefaultContextTimeoutConfig)
}

// ContextTimeoutWithConfig returns a Timeout middleware with config.
func ContextTimeoutWithConfig(config ContextTimeoutConfig) echo.MiddlewareFunc {
	return config.ToMiddleware()
}

// ToMiddleware converts Config to middleware or returns an error for invalid configuration
func (config ContextTimeoutConfig) ToMiddleware() echo.MiddlewareFunc {
	if config.Skipper == nil {
		config.Skipper = DefaultTimeoutConfig.Skipper
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if config.Skipper(c) || config.Timeout == 0 {
				return next(c)
			}

			timeoutContext, cancel := context.WithTimeout(c.Request().Context(), config.Timeout)
			defer cancel()

			timeoutRequest := c.Request().WithContext(timeoutContext)

			c.SetRequest(timeoutRequest)

			if err := next(c); err != nil {
				if errors.Is(err, context.DeadlineExceeded) {
					return echo.ErrServiceUnavailable
				}

				return err
			}
			return nil

		}
	}
}
