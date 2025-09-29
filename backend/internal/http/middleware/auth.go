package middleware

import (
	"iafarma/internal/auth"
	"net/http"
	"strings"

	"github.com/labstack/echo/v4"
)

// JWTAuth middleware validates JWT tokens
func JWTAuth(authService *auth.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Get token from Authorization header
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
			}

			// Check if header starts with "Bearer "
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization header format")
			}

			// Extract token
			tokenString := authHeader[7:] // Remove "Bearer " prefix
			if tokenString == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing token")
			}

			// Validate token
			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
			}

			// Store claims in context
			c.Set("claims", claims)
			c.Set("user_id", claims.UserID)
			c.Set("user_email", claims.Email)
			c.Set("user_role", claims.Role)

			// Store tenant_id if present
			if claims.TenantID != nil {
				c.Set("tenant_id", *claims.TenantID)
			}

			return next(c)
		}
	}
}

// RequireRole middleware ensures user has required role
func RequireRole(roles ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole := c.Get("user_role")
			if userRole == nil {
				return echo.NewHTTPError(http.StatusForbidden, "User role not found")
			}

			roleStr := userRole.(string)
			for _, role := range roles {
				if roleStr == role {
					return next(c)
				}
			}

			return echo.NewHTTPError(http.StatusForbidden, "Insufficient permissions")
		}
	}
}

// SystemAdminOnly middleware ensures only system admins can access
func SystemAdminOnly() echo.MiddlewareFunc {
	return RequireRole("system_admin")
}

// TenantAdminOrAbove middleware allows tenant_admin and system_admin
func TenantAdminOrAbove() echo.MiddlewareFunc {
	return RequireRole("system_admin", "tenant_admin")
}

// TenantUserOrAbove middleware allows tenant_user, tenant_admin and system_admin
func TenantUserOrAbove() echo.MiddlewareFunc {
	return RequireRole("system_admin", "tenant_admin", "tenant_user")
}

// RequireSystemRole middleware ensures user has system-level access (no tenant)
func RequireSystemRole() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole := c.Get("user_role")
			if userRole == nil {
				return echo.NewHTTPError(http.StatusForbidden, "User role not found")
			}

			roleStr := userRole.(string)
			if roleStr != "system_admin" {
				return echo.NewHTTPError(http.StatusForbidden, "System admin access required")
			}

			// System admins should not have tenant_id
			tenantID := c.Get("tenant_id")
			if tenantID != nil {
				return echo.NewHTTPError(http.StatusForbidden, "System admin cannot have tenant context")
			}

			return next(c)
		}
	}
}

// RequireTenantRole middleware ensures user has tenant-level access
func RequireTenantRole() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole := c.Get("user_role")
			if userRole == nil {
				return echo.NewHTTPError(http.StatusForbidden, "User role not found")
			}

			roleStr := userRole.(string)
			// Permitir system_admin, tenant_admin e tenant_user
			if roleStr != "system_admin" && roleStr != "tenant_admin" && roleStr != "tenant_user" {
				return echo.NewHTTPError(http.StatusForbidden, "Tenant access required")
			}

			// Para system_admin, não é necessário tenant_id
			if roleStr == "system_admin" {
				return next(c)
			}

			// Tenant users must have tenant_id
			tenantID := c.Get("tenant_id")
			if tenantID == nil {
				return echo.NewHTTPError(http.StatusForbidden, "Tenant context required")
			}

			return next(c)
		}
	}
}

// RequireTenantAdminOnly middleware ensures only tenant admins can access (not tenant_user)
func RequireTenantAdminOnly() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			userRole := c.Get("user_role")
			if userRole == nil {
				return echo.NewHTTPError(http.StatusForbidden, "User role not found")
			}

			roleStr := userRole.(string)
			if roleStr != "tenant_admin" {
				return echo.NewHTTPError(http.StatusForbidden, "Tenant admin access required")
			}

			// Tenant admins must have tenant_id
			tenantID := c.Get("tenant_id")
			if tenantID == nil {
				return echo.NewHTTPError(http.StatusForbidden, "Tenant context required")
			}

			return next(c)
		}
	}
}
