package middleware

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// TenantResolver middleware resolves tenant from JWT token or header
func TenantResolver(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var tenantID uuid.UUID
			var err error

			// Check if tenant_id was already set by JWT middleware
			if existingTenantID := c.Get("tenant_id"); existingTenantID != nil {
				if tid, ok := existingTenantID.(uuid.UUID); ok {
					tenantID = tid
				}
			}

			// If not set, try to get from X-Tenant-ID header
			if tenantID == uuid.Nil {
				tenantIDHeader := c.Request().Header.Get("X-Tenant-ID")
				if tenantIDHeader != "" {
					if tenantID, err = uuid.Parse(tenantIDHeader); err != nil {
						return echo.NewHTTPError(400, "Invalid tenant ID format")
					}
					c.Set("tenant_id", tenantID)
				}
			}

			// Set tenant context for RLS if tenant is present
			if tenantID != uuid.Nil {
				sql := fmt.Sprintf("SET LOCAL app.tenant = '%s'", tenantID.String())
				if err := db.Exec(sql).Error; err != nil {
					log.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("Failed to set tenant context")
					return echo.NewHTTPError(500, "Internal server error")
				}

				// Also set in context for handlers
				ctx := context.WithValue(c.Request().Context(), "tenant_id", tenantID)
				c.SetRequest(c.Request().WithContext(ctx))
			}

			return next(c)
		}
	}
}

// RequireTenant middleware ensures a tenant is present
func RequireTenant() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Para system_admin, não é necessário tenant_id
			userRole := c.Get("user_role")
			if userRole != nil && userRole.(string) == "system_admin" {
				return next(c)
			}

			tenantID := c.Get("tenant_id")
			if tenantID == nil {
				return echo.NewHTTPError(400, "Tenant ID is required")
			}

			if tenantID.(uuid.UUID) == uuid.Nil {
				return echo.NewHTTPError(400, "Valid tenant ID is required")
			}

			return next(c)
		}
	}
}

// SystemOnly middleware ensures only system-level access
func SystemOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantID := c.Get("tenant_id")
			if tenantID != nil && tenantID.(uuid.UUID) != uuid.Nil {
				return echo.NewHTTPError(403, "System-only access required")
			}

			return next(c)
		}
	}
}
