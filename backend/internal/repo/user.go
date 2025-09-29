package repo

import (
	"iafarma/pkg/models"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserRepository handles user data access
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

// GetByEmail gets a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("LOWER(email) = LOWER(?)", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByID gets a user by ID
func (r *UserRepository) GetByID(id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Create creates a new user
func (r *UserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// Update updates a user
func (r *UserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// List lists users with pagination
func (r *UserRepository) List(tenantID *uuid.UUID, limit, offset int) ([]models.User, error) {
	var users []models.User
	query := r.db.Limit(limit).Offset(offset)

	if tenantID != nil {
		query = query.Where("tenant_id = ?", *tenantID)
	}

	err := query.Find(&users).Error
	return users, err
}

// TenantRepository handles tenant data access
type TenantRepository struct {
	db *gorm.DB
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *gorm.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

// GetDB returns the database connection for custom queries
func (r *TenantRepository) GetDB() *gorm.DB {
	return r.db
}

// GetByID gets a tenant by ID
func (r *TenantRepository) GetByID(id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	err := r.db.Preload("PlanInfo").Where("id = ?", id).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

// GetByTag gets a tenant by TAG
func (r *TenantRepository) GetByTag(tag string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := r.db.Where("tag = ?", tag).First(&tenant).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Return nil instead of error for not found
		}
		return nil, err
	}
	return &tenant, nil
}

// Create creates a new tenant
func (r *TenantRepository) Create(tenant *models.Tenant) error {
	return r.db.Create(tenant).Error
}

// Update updates a tenant
func (r *TenantRepository) Update(tenant *models.Tenant) error {
	// Clear the PlanInfo relationship to avoid GORM conflicts during update
	tenant.PlanInfo = nil

	result := r.db.Save(tenant)
	if result.Error != nil {
		return result.Error
	}

	// Reload the tenant with plan relationship for returning updated data
	return r.db.Preload("PlanInfo").Where("id = ?", tenant.ID).First(tenant).Error
}

// TenantWithAdmin represents a tenant with admin email information
type TenantWithAdmin struct {
	models.Tenant
	AdminEmail string `json:"admin_email"`
}

// List lists tenants with pagination and plan information including admin email
func (r *TenantRepository) List(limit, offset int) (*PaginationResult[TenantWithAdmin], error) {
	var tenantsWithAdmin []TenantWithAdmin
	var total int64

	// Get total count
	if err := r.db.Model(&models.Tenant{}).Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated data with plan information and admin email, ordered by created_at DESC
	err := r.db.Table("tenants").
		Select(`tenants.*, 
			plans.id as "plan_info__id",
			plans.name as "plan_info__name", 
			plans.description as "plan_info__description",
			plans.price as "plan_info__price",
			plans.currency as "plan_info__currency",
			plans.billing_period as "plan_info__billing_period",
			plans.max_conversations as "plan_info__max_conversations",
			plans.max_messages_per_month as "plan_info__max_messages_per_month",
			plans.max_products as "plan_info__max_products",
			plans.max_channels as "plan_info__max_channels",
			plans.max_credits_per_month as "plan_info__max_credits_per_month",
			plans.is_active as "plan_info__is_active",
			plans.is_default as "plan_info__is_default",
			plans.features as "plan_info__features",
			plans.stripe_url as "plan_info__stripe_url",
			plans.created_at as "plan_info__created_at",
			plans.updated_at as "plan_info__updated_at",
			COALESCE(users.email, '') as admin_email`).
		Joins("LEFT JOIN plans ON tenants.plan_id = plans.id").
		Joins("LEFT JOIN users ON tenants.id = users.tenant_id AND users.role = 'tenant_admin'").
		Order("tenants.created_at DESC").
		Limit(limit).
		Offset(offset).
		Scan(&tenantsWithAdmin).Error

	if err != nil {
		return nil, err
	}

	page := (offset / limit) + 1
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &PaginationResult[TenantWithAdmin]{
		Data:       tenantsWithAdmin,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}

// Delete deletes a tenant by ID
func (r *TenantRepository) Delete(id uuid.UUID) error {
	// Using transaction to ensure data integrity
	return r.db.Transaction(func(tx *gorm.DB) error {
		// First delete the tenant (cascade will handle related records)
		return tx.Delete(&models.Tenant{}, "id = ?", id).Error
	})
}

// Password Reset Token Methods

// CreatePasswordResetToken creates a new password reset token
func (r *UserRepository) CreatePasswordResetToken(token *models.PasswordResetToken) error {
	return r.db.Create(token).Error
}

// GetPasswordResetToken gets a password reset token by token string
func (r *UserRepository) GetPasswordResetToken(token string) (*models.PasswordResetToken, error) {
	var resetToken models.PasswordResetToken
	err := r.db.Preload("User").Where("token = ? AND is_used = false AND expires_at > NOW()", token).First(&resetToken).Error
	if err != nil {
		return nil, err
	}
	return &resetToken, nil
}

// MarkPasswordResetTokenAsUsed marks a password reset token as used
func (r *UserRepository) MarkPasswordResetTokenAsUsed(tokenID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.PasswordResetToken{}).
		Where("id = ?", tokenID).
		Updates(map[string]interface{}{
			"is_used": true,
			"used_at": &now,
		}).Error
}

// InvalidateUserPasswordResetTokens invalidates all unused tokens for a user
func (r *UserRepository) InvalidateUserPasswordResetTokens(userID uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND is_used = false", userID).
		Updates(map[string]interface{}{
			"is_used": true,
			"used_at": &now,
		}).Error
}
