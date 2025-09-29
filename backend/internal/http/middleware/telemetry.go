package middleware

import (
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// Telemetry middleware adds OpenTelemetry tracing (optional)
func Telemetry() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Safely try to create tracer
			tracer := otel.Tracer("iafarma-api")

			// If telemetry is not configured, just proceed without tracing
			defer func() {
				if r := recover(); r != nil {
					// Telemetry failed, but don't break the request
				}
			}()

			ctx := c.Request().Context()

			spanName := c.Request().Method + " " + c.Path()
			ctx, span := tracer.Start(ctx, spanName)
			defer span.End()
			defer span.End()

			// Set request attributes
			span.SetAttributes(
				attribute.String("http.method", c.Request().Method),
				attribute.String("http.url", c.Request().URL.String()),
				attribute.String("http.route", c.Path()),
				attribute.String("user_agent", c.Request().UserAgent()),
			)

			// Set request ID if available
			if requestID := c.Get("request_id"); requestID != nil {
				span.SetAttributes(attribute.String("request.id", requestID.(string)))
			}

			// Set tenant ID if available
			if tenantID := c.Get("tenant_id"); tenantID != nil {
				span.SetAttributes(attribute.String("tenant.id", tenantID.(string)))
			}

			// Update request context
			c.SetRequest(c.Request().WithContext(ctx))

			err := next(c)

			// Set response attributes
			span.SetAttributes(attribute.Int("http.status_code", c.Response().Status))

			if err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}

			return err
		}
	}
}
