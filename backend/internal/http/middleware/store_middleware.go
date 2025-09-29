package middleware

import (
	"iafarma/internal/auth"
	"iafarma/pkg/models"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// StoreTagMiddleware valida e injeta o tenant_id a partir da TAG para rotas da loja
func StoreTagMiddleware(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tag := c.Param("tag")

			// Verificar se TAG foi fornecida
			if tag == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "TAG da loja é obrigatória",
				})
			}

			// Buscar tenant pela TAG
			var tenant models.Tenant
			if err := db.Where("tag = ? AND status = 'active' AND is_public_store = true", tag).First(&tenant).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return c.JSON(http.StatusNotFound, map[string]string{
						"error": "Loja não encontrada ou não disponível publicamente",
					})
				}
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Erro ao verificar loja",
				})
			}

			// Injetar tenant_id e tenant no context
			c.Set("tenant_id", tenant.ID)
			c.Set("tenant", &tenant)
			c.Set("tag", tag)

			return next(c)
		}
	}
}

// StoreTenantMiddleware valida e injeta o tenant_id para rotas da loja (versão antiga usando tenant_id)
func StoreTenantMiddleware(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			tenantIDStr := c.Param("tenant_id")

			// Verificar se tenant_id foi fornecido
			if tenantIDStr == "" {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "tenant_id é obrigatório para rotas da loja",
				})
			}

			// Validar formato UUID
			tenantID, err := uuid.Parse(tenantIDStr)
			if err != nil {
				return c.JSON(http.StatusBadRequest, map[string]string{
					"error": "tenant_id deve ser um UUID válido",
				})
			}

			// Verificar se o tenant existe e está ativo
			var tenant models.Tenant
			if err := db.Where("id = ? AND status = 'active'", tenantID).First(&tenant).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					return c.JSON(http.StatusNotFound, map[string]string{
						"error": "Loja não encontrada ou inativa",
					})
				}
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Erro ao verificar loja",
				})
			}

			// Injetar tenant_id e tenant no context
			c.Set("tenant_id", tenantID)
			c.Set("tenant", &tenant)

			return next(c)
		}
	}
}

// StoreAuthMiddleware middleware de autenticação específico para clientes da loja
func StoreAuthMiddleware(authService *auth.Service) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Obter token do header Authorization
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Token de autorização requerido",
				})
			}

			// Remover "Bearer " do início
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenString == authHeader {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Formato do token inválido. Use 'Bearer <token>'",
				})
			}

			// Validar token
			claims, err := authService.ValidateToken(tokenString)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Token inválido ou expirado",
				})
			}

			// Verificar se é um token de cliente (não de usuário admin)
			if claims.Role != "customer" {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Acesso negado. Token de cliente requerido",
				})
			}

			// Verificar se o tenant do token corresponde ao da URL
			tenantID := c.Get("tenant_id").(uuid.UUID)
			if claims.TenantID != nil && *claims.TenantID != tenantID {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Token não pertence a esta loja",
				})
			}

			// Injetar informações do cliente no context
			c.Set("customer_id", claims.UserID)
			c.Set("customer_phone", claims.Email) // Para clientes, usamos phone no lugar de email

			return next(c)
		}
	}
}

// StoreCustomerContext helper para extrair informações do cliente do context
type StoreCustomerContext struct {
	TenantID      uuid.UUID
	CustomerID    string
	CustomerPhone string
	Tenant        *models.Tenant
}

// GetStoreCustomerContext extrai as informações do cliente do echo.Context
func GetStoreCustomerContext(c echo.Context) *StoreCustomerContext {
	tenantID, _ := c.Get("tenant_id").(uuid.UUID)
	customerID, _ := c.Get("customer_id").(string)
	customerPhone, _ := c.Get("customer_phone").(string)
	tenant, _ := c.Get("tenant").(*models.Tenant)

	return &StoreCustomerContext{
		TenantID:      tenantID,
		CustomerID:    customerID,
		CustomerPhone: customerPhone,
		Tenant:        tenant,
	}
}

// StoreTenantContext helper para extrair informações do tenant (para rotas públicas)
type StoreTenantContext struct {
	TenantID uuid.UUID
	Tenant   *models.Tenant
}

// GetStoreTenantContext extrai as informações do tenant do echo.Context
func GetStoreTenantContext(c echo.Context) *StoreTenantContext {
	tenantID, _ := c.Get("tenant_id").(uuid.UUID)
	tenant, _ := c.Get("tenant").(*models.Tenant)

	return &StoreTenantContext{
		TenantID: tenantID,
		Tenant:   tenant,
	}
}

// StoreUnifiedMiddleware detecta automaticamente se o acesso é via TAG ou domínio
// Suporta duas formas de acesso:
// 1. Via TAG: /:tag/path (http://localhost:8082/farmacia-teste/path)
// 2. Via Domínio: /path com Host header (http://farmabrasil.com/path)
func StoreUnifiedMiddleware(db *gorm.DB) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var tenant models.Tenant
			var tag string
			var err error

			// Extrair o parâmetro tag da URL (pode estar vazio se for acesso via domínio)
			tagParam := c.Param("tag")

			// Se existe parâmetro TAG, usar busca por TAG
			if tagParam != "" {
				tag = tagParam
				err = db.Where("tag = ? AND status = 'active' AND is_public_store = true", tag).First(&tenant).Error
			} else {
				// Caso contrário, tentar buscar por domínio via Origin header (preferível)
				var originHost string

				// Tentar Origin primeiro (sempre presente em CORS requests)
				origin := c.Request().Header.Get("Origin")
				if origin != "" {
					// Extrair apenas o hostname do Origin (ex: "http://farmabrasil.localhost:8082" → "farmabrasil.localhost")
					if strings.HasPrefix(origin, "http://") {
						originHost = origin[7:] // Remove "http://"
					} else if strings.HasPrefix(origin, "https://") {
						originHost = origin[8:] // Remove "https://"
					}

					// Remover porta se presente
					if colonIndex := strings.Index(originHost, ":"); colonIndex != -1 {
						originHost = originHost[:colonIndex]
					}
				}

				// Se não encontrou no Origin, tentar Referer como fallback
				if originHost == "" {
					referer := c.Request().Header.Get("Referer")
					if referer != "" {
						if strings.HasPrefix(referer, "http://") {
							originHost = referer[7:]
						} else if strings.HasPrefix(referer, "https://") {
							originHost = referer[8:]
						}

						// Extrair apenas o hostname (remover path e porta)
						if slashIndex := strings.Index(originHost, "/"); slashIndex != -1 {
							originHost = originHost[:slashIndex]
						}
						if colonIndex := strings.Index(originHost, ":"); colonIndex != -1 {
							originHost = originHost[:colonIndex]
						}
					}
				}

				// Se ainda não encontrou, usar Host como último recurso
				if originHost == "" {
					originHost = c.Request().Host
					// Remover porta se presente
					if colonIndex := strings.Index(originHost, ":"); colonIndex != -1 {
						originHost = originHost[:colonIndex]
					}
				}

				err = db.Where("domain = ? AND status = 'active' AND is_public_store = true", originHost).First(&tenant).Error

				// Se encontrou via domínio, definir tag do tenant encontrado
				if err == nil {
					if tenant.Tag != nil {
						tag = *tenant.Tag // Tag do tenant encontrado via domínio
					} else {
						tag = "" // Tenant sem tag definida
					}
				}
			}

			// Verificar se encontrou o tenant
			if err != nil {
				if err == gorm.ErrRecordNotFound {
					errorMsg := "Loja não encontrada ou não disponível publicamente"
					if tagParam != "" {
						errorMsg = "Loja com TAG '" + tagParam + "' não encontrada ou não disponível publicamente"
					} else {
						errorMsg = "Loja com domínio '" + c.Request().Host + "' não encontrada ou não disponível publicamente"
					}
					return c.JSON(http.StatusNotFound, map[string]string{
						"error": errorMsg,
					})
				}
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error": "Erro ao verificar loja",
				})
			}

			// Injetar tenant_id, tenant e tag no context
			c.Set("tenant_id", tenant.ID)
			c.Set("tenant", &tenant)
			c.Set("tag", tag)
			c.Set("access_method", map[string]interface{}{
				"via_tag":      tagParam != "",
				"via_domain":   tagParam == "",
				"host":         c.Request().Host,
				"resolved_tag": tag,
			})

			return next(c)
		}
	}
}
