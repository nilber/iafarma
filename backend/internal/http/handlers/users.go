package handlers

import (
	"fmt"
	"iafarma/internal/auth"
	"iafarma/internal/services"
	"iafarma/pkg/models"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserHandler struct {
	db           *gorm.DB
	authService  *auth.Service
	emailService *services.EmailService
}

func NewUserHandler(db *gorm.DB, authService *auth.Service) *UserHandler {
	emailService, err := services.NewEmailService(db)
	if err != nil {
		fmt.Printf("⚠️  Email service not available: %v\n", err)
	}
	return &UserHandler{
		db:           db,
		authService:  authService,
		emailService: emailService,
	}
}

// CreateTenantAdminRequest represents the request to create a tenant admin user
type CreateTenantAdminRequest struct {
	TenantID string `json:"tenant_id" validate:"required"`
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone"`
	Password string `json:"password" validate:"required,min=6"`
}

// CreateUserRequest represents the request to create a new tenant user
type CreateUserRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone"`
	Password string `json:"password" validate:"required,min=6"`
}

// UpdateUserRequest represents the request to update a tenant user
type UpdateUserRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone"`
	IsActive bool   `json:"is_active"`
}

// ChangeUserPasswordRequest represents the request to change a user's password
type ChangeUserPasswordRequest struct {
	Password string `json:"password" validate:"required,min=6"`
}

// SendTenantCredentialsRequest represents the request to send tenant credentials by email
type SendTenantCredentialsRequest struct {
	TenantID      string `json:"tenant_id" validate:"required"`
	TenantName    string `json:"tenant_name" validate:"required"`
	AdminName     string `json:"admin_name" validate:"required"`
	AdminEmail    string `json:"admin_email" validate:"required,email"`
	AdminPassword string `json:"admin_password" validate:"required"`
}

// UpdateTenantAdminRequest represents the request to update a tenant admin
type UpdateTenantAdminRequest struct {
	Name     string `json:"name" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
	Phone    string `json:"phone"`
	Password string `json:"password,omitempty"`
}

// CreateTenantAdmin creates a new tenant admin user (only system_admin can do this)
func (h *UserHandler) CreateTenantAdmin(c echo.Context) error {
	var req CreateTenantAdminRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Parse tenant ID
	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid tenant ID")
	}

	// Verify tenant exists
	var tenant models.Tenant
	if err := h.db.Where("id = ?", tenantUUID).First(&tenant).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Tenant not found")
	}

	// Check if email already exists (case-insensitive)
	var existingUser models.User
	if err := h.db.Where("LOWER(email) = LOWER(?)", req.Email).First(&existingUser).Error; err == nil {
		return echo.NewHTTPError(http.StatusConflict, "Este email já está sendo utilizado por outro usuário")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to hash password")
	}

	// Create tenant admin user
	user := models.User{
		TenantID: &tenantUUID,
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
		Phone:    req.Phone,
		Role:     "tenant_admin",
		IsActive: true,
	}

	if err := h.db.Create(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create tenant admin")
	}

	// Remove password from response
	user.Password = ""

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Tenant admin created successfully",
		"user":    user,
	})
}

// GetTenantUsersForAdmin lists all users of a specific tenant (system_admin only)
func (h *UserHandler) GetTenantUsersForAdmin(c echo.Context) error {
	tenantID := c.Param("tenant_id")

	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid tenant ID")
	}

	// Verify tenant exists
	var tenant models.Tenant
	if err := h.db.Where("id = ?", tenantUUID).First(&tenant).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Tenant not found")
	}

	// Get pagination parameters
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	page, _ := strconv.Atoi(c.QueryParam("page"))

	if limit <= 0 {
		limit = 20
	}
	if page <= 0 {
		page = 1
	}
	offset := (page - 1) * limit

	// Get users for this tenant
	var users []models.User
	var total int64

	if err := h.db.Model(&models.User{}).Where("tenant_id = ?", tenantUUID).Count(&total).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count users")
	}

	if err := h.db.Where("tenant_id = ?", tenantUUID).
		Limit(limit).
		Offset(offset).
		Find(&users).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch users")
	}

	// Remove passwords from response
	for i := range users {
		users[i].Password = ""
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// UpdateTenantAdminForAdmin updates a tenant admin user (system_admin only)
func (h *UserHandler) UpdateTenantAdminForAdmin(c echo.Context) error {
	userID := c.Param("user_id")

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID")
	}

	var req UpdateTenantAdminRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Get existing user
	var user models.User
	if err := h.db.Where("id = ? AND role = ?", userUUID, "tenant_admin").First(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Tenant admin user not found")
	}

	// Check if new email already exists (for other users)
	if req.Email != user.Email {
		var existingUser models.User
		if err := h.db.Where("LOWER(email) = LOWER(?) AND id != ?", req.Email, userUUID).First(&existingUser).Error; err == nil {
			return echo.NewHTTPError(http.StatusConflict, "Este email já está sendo utilizado por outro usuário")
		}
	}

	// Update user fields
	user.Name = req.Name
	user.Email = req.Email
	user.Phone = req.Phone

	// Update password if provided
	if req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, "Failed to hash password")
		}
		user.Password = string(hashedPassword)
	}

	// Save changes
	if err := h.db.Save(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update user")
	}

	// Remove password from response
	user.Password = ""

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Tenant admin updated successfully",
		"user":    user,
	})
}

// CreateTenantUser creates a new user for the tenant (tenant_user role)
func (h *UserHandler) CreateTenantUser(c echo.Context) error {
	var req CreateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Get tenant ID from context
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Tenant context required")
	}

	tenantUUID, ok := tenantID.(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid tenant ID")
	}

	// Check if email already exists
	var existingUser models.User
	if err := h.db.Where("LOWER(email) = LOWER(?)", req.Email).First(&existingUser).Error; err == nil {
		return echo.NewHTTPError(http.StatusConflict, "Este email já está sendo utilizado por outro usuário")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to hash password")
	}

	// Create user
	user := models.User{
		TenantID: &tenantUUID,
		Email:    req.Email,
		Password: string(hashedPassword),
		Name:     req.Name,
		Phone:    req.Phone,
		Role:     "tenant_user",
		IsActive: true,
	}

	if err := h.db.Create(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to create user")
	}

	// Remove password from response
	user.Password = ""

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "User created successfully",
		"user":    user,
	})
}

// GetTenantUsers retrieves all users for the current tenant
func (h *UserHandler) GetTenantUsers(c echo.Context) error {
	// Get tenant ID from context
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Tenant context required")
	}

	tenantUUID, ok := tenantID.(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid tenant ID")
	}

	// Get pagination parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}

	offset := (page - 1) * limit

	// Get users for the tenant
	var users []models.User
	var total int64

	// Count total users
	if err := h.db.Model(&models.User{}).Where("tenant_id = ?", tenantUUID).Count(&total).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to count users")
	}

	// Get paginated users (exclude password)
	if err := h.db.Select("id, tenant_id, email, name, phone, role, is_active, last_login_at, created_at, updated_at").
		Where("tenant_id = ?", tenantUUID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&users).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch users")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"users": users,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
			"pages": (total + int64(limit) - 1) / int64(limit),
		},
	})
}

// GetTenantUser retrieves a specific user for the current tenant
func (h *UserHandler) GetTenantUser(c echo.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "User ID is required")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID format")
	}

	// Get tenant ID from context
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Tenant context required")
	}

	tenantUUID, ok := tenantID.(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid tenant ID")
	}

	// Get user (exclude password)
	var user models.User
	if err := h.db.Select("id, tenant_id, email, name, phone, role, is_active, last_login_at, created_at, updated_at").
		Where("id = ? AND tenant_id = ?", userUUID, tenantUUID).
		First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch user")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"user": user,
	})
}

// UpdateTenantUser updates a user for the current tenant
func (h *UserHandler) UpdateTenantUser(c echo.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "User ID is required")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID format")
	}

	var req UpdateUserRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Get tenant ID from context
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Tenant context required")
	}

	tenantUUID, ok := tenantID.(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid tenant ID")
	}

	// Check if user exists and belongs to tenant
	var user models.User
	if err := h.db.Where("id = ? AND tenant_id = ?", userUUID, tenantUUID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch user")
	}

	// Check if email already exists (but not for this user)
	var existingUser models.User
	if err := h.db.Where("LOWER(email) = LOWER(?) AND id != ?", req.Email, userUUID).First(&existingUser).Error; err == nil {
		return echo.NewHTTPError(http.StatusConflict, "Este email já está sendo utilizado por outro usuário")
	}

	// Update user
	user.Name = req.Name
	user.Email = req.Email
	user.Phone = req.Phone
	user.IsActive = req.IsActive

	if err := h.db.Save(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update user")
	}

	// Remove password from response
	user.Password = ""

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User updated successfully",
		"user":    user,
	})
}

// ChangeUserPassword changes a user's password
func (h *UserHandler) ChangeUserPassword(c echo.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "User ID is required")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID format")
	}

	var req ChangeUserPasswordRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Get tenant ID from context
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Tenant context required")
	}

	tenantUUID, ok := tenantID.(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid tenant ID")
	}

	// Check if user exists and belongs to tenant
	var user models.User
	if err := h.db.Where("id = ? AND tenant_id = ?", userUUID, tenantUUID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch user")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to hash password")
	}

	// Update password
	user.Password = string(hashedPassword)
	if err := h.db.Save(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to update password")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Password updated successfully",
	})
}

// DeleteTenantUser deletes a user for the current tenant
func (h *UserHandler) DeleteTenantUser(c echo.Context) error {
	userID := c.Param("id")
	if userID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "User ID is required")
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid user ID format")
	}

	// Get tenant ID from context
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Tenant context required")
	}

	tenantUUID, ok := tenantID.(uuid.UUID)
	if !ok {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid tenant ID")
	}

	// Check if user exists and belongs to tenant
	var user models.User
	if err := h.db.Where("id = ? AND tenant_id = ?", userUUID, tenantUUID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return echo.NewHTTPError(http.StatusNotFound, "User not found")
		}
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to fetch user")
	}

	// Check if user is a tenant_admin (cannot delete tenant admins)
	if user.Role == "tenant_admin" {
		return echo.NewHTTPError(http.StatusForbidden, "Cannot delete tenant admin users")
	}

	// Delete user
	if err := h.db.Delete(&user).Error; err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to delete user")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": fmt.Sprintf("User %s deleted successfully", user.Name),
	})
}

// SendTenantCredentials sends tenant admin credentials via email
func (h *UserHandler) SendTenantCredentials(c echo.Context) error {
	var req SendTenantCredentialsRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request format")
	}

	if err := c.Validate(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Check if email service is available
	if h.emailService == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "Email service not configured")
	}

	// Parse tenant ID
	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid tenant ID")
	}

	// Verify tenant exists
	var tenant models.Tenant
	if err := h.db.Where("id = ?", tenantUUID).First(&tenant).Error; err != nil {
		return echo.NewHTTPError(http.StatusNotFound, "Tenant not found")
	}

	// Create email template
	subject := fmt.Sprintf("Suas credenciais de acesso - %s", req.TenantName)

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Credenciais de Acesso</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%); color: white; padding: 20px; border-radius: 8px 8px 0 0; }
        .content { background: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .credentials { background: white; padding: 20px; border-radius: 8px; margin: 20px 0; border-left: 4px solid #667eea; }
        .footer { text-align: center; margin-top: 30px; color: #666; font-size: 12px; }
        .button { display: inline-block; background: #667eea; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🎉 Bem-vindo ao Sistema!</h1>
        </div>
        <div class="content">
            <p>Olá <strong>%s</strong>,</p>
            
            <p>Sua empresa <strong>%s</strong> foi criada com sucesso em nosso sistema! Agora você tem acesso completo à plataforma de vendas e atendimento via WhatsApp.</p>
            
            <div class="credentials">
                <h3>🔐 Suas Credenciais de Acesso</h3>
                <p><strong>Email:</strong> %s</p>
                <p><strong>Senha:</strong> %s</p>
            </div>
            
            <p>⚠️ <strong>Importante:</strong> Recomendamos que você altere sua senha após o primeiro acesso por questões de segurança.</p>
            
            <p>Com seu acesso, você poderá:</p>
            <ul>
                <li>✅ Gerenciar produtos e estoque</li>
                <li>✅ Atender clientes via WhatsApp</li>
                <li>✅ Acompanhar vendas e relatórios</li>
                <li>✅ Configurar automações de IA</li>
                <li>✅ Gerenciar equipe e usuários</li>
            </ul>
            
            <a href="%s" class="button">🚀 Acessar Sistema</a>
            
            <p>Se tiver qualquer dúvida, nossa equipe de suporte está sempre disponível para ajudar!</p>
            
            <p>Bem-vindo a bordo! 🎊</p>
        </div>
        <div class="footer">
            <p>Este é um email automático, não responda a esta mensagem.<br>
            Em caso de dúvidas, entre em contato através do nosso suporte.</p>
        </div>
    </div>
</body>
</html>
	`, req.AdminName, req.TenantName, req.AdminEmail, req.AdminPassword, "https://painel.iafarma.net/")

	// Send email
	err = h.emailService.SendEmail([]string{req.AdminEmail}, subject, body)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Failed to send email: %v", err))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Credenciais enviadas por email com sucesso",
		"sent_to": req.AdminEmail,
	})
}
