package repo

import (
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductFilters represents filters for product queries
type ProductFilters struct {
	CategoryID   *string  `json:"category_id,omitempty"`
	MinPrice     *float64 `json:"min_price,omitempty"`
	MaxPrice     *float64 `json:"max_price,omitempty"`
	HasPromotion *bool    `json:"has_promotion,omitempty"`
	HasSKU       *bool    `json:"has_sku,omitempty"`
	HasStock     *bool    `json:"has_stock,omitempty"`
	OutOfStock   *bool    `json:"out_of_stock,omitempty"`
}

// ProductRepository handles product data access
type ProductRepository struct {
	db *gorm.DB
}

// NewProductRepository creates a new product repository
func NewProductRepository(db *gorm.DB) *ProductRepository {
	return &ProductRepository{db: db}
}

// GetByID gets a product by ID
func (r *ProductRepository) GetByID(tenantID, id uuid.UUID) (*models.Product, error) {
	var product models.Product
	err := r.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

// Create creates a new product
func (r *ProductRepository) Create(product *models.Product) error {
	return r.db.Create(product).Error
}

// Update updates a product
func (r *ProductRepository) Update(product *models.Product) error {
	// Use Select to exclude EmbeddingHash from being overwritten
	return r.db.Omit("embedding_hash").Save(product).Error
}

// FindExistingProduct finds a product by unique keys (case insensitive)
func (r *ProductRepository) FindExistingProduct(tenantID uuid.UUID, name, sku, barcode string) (*models.Product, error) {
	var product models.Product

	// First try to find by name (case insensitive) - include soft deleted to handle upsert correctly
	if name != "" {
		err := r.db.Unscoped().Where("tenant_id = ? AND LOWER(name) = LOWER(?)", tenantID, name).First(&product).Error
		if err == nil {
			return &product, nil
		}
	}

	// Then try by SKU (case insensitive) - include soft deleted to handle upsert correctly
	if sku != "" {
		err := r.db.Unscoped().Where("tenant_id = ? AND LOWER(sku) = LOWER(?)", tenantID, sku).First(&product).Error
		if err == nil {
			return &product, nil
		}
	}

	// Finally try by barcode (case insensitive) - include soft deleted to handle upsert correctly
	if barcode != "" {
		err := r.db.Unscoped().Where("tenant_id = ? AND LOWER(barcode) = LOWER(?)", tenantID, barcode).First(&product).Error
		if err == nil {
			return &product, nil
		}
	}

	return nil, gorm.ErrRecordNotFound
}

// UpsertProduct creates or updates a product based on unique keys
func (r *ProductRepository) UpsertProduct(product *models.Product) (*models.Product, bool, error) {
	existing, err := r.FindExistingProduct(product.TenantID, product.Name, product.SKU, product.Barcode)

	if err == gorm.ErrRecordNotFound {
		// Product doesn't exist, create new one
		err = r.db.Create(product).Error
		return product, true, err // true = created
	}

	if err != nil {
		return nil, false, err
	}

	// Product exists, update it while preserving EmbeddingHash
	existing.Name = product.Name
	existing.Description = product.Description
	existing.Price = product.Price
	existing.SalePrice = product.SalePrice
	existing.SKU = product.SKU
	existing.Barcode = product.Barcode
	existing.Weight = product.Weight
	existing.Dimensions = product.Dimensions
	existing.Brand = product.Brand
	existing.Tags = product.Tags
	existing.StockQuantity = product.StockQuantity
	existing.LowStockThreshold = product.LowStockThreshold
	existing.CategoryID = product.CategoryID
	existing.DeletedAt = nil // Restaurar produto se estava soft deleted
	// Note: EmbeddingHash is preserved automatically (not overwritten)

	// Use Omit to exclude embedding_hash from being overwritten, but use Unscoped to update soft deleted records
	err = r.db.Unscoped().Omit("embedding_hash").Save(existing).Error
	return existing, false, err // false = updated
}

// PaginationResult represents paginated results
type PaginationResult[T any] struct {
	Data       []T   `json:"data"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
}

// List lists products with pagination
func (r *ProductRepository) List(tenantID uuid.UUID, limit, offset int) (*PaginationResult[models.Product], error) {
	var products []models.Product
	var total int64

	// Get total count with tenant filter
	if err := r.db.Model(&models.Product{}).Where("tenant_id = ?", tenantID).Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated data with tenant filter
	err := r.db.Where("tenant_id = ?", tenantID).Limit(limit).Offset(offset).Find(&products).Error
	if err != nil {
		return nil, err
	}

	page := (offset / limit) + 1
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &PaginationResult[models.Product]{
		Data:       products,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}

// ListWithSearch lists products with pagination and search functionality
func (r *ProductRepository) ListWithSearch(tenantID uuid.UUID, limit, offset int, search string) (*PaginationResult[models.Product], error) {
	var products []models.Product
	var total int64

	// Build base query with tenant filter and STOCK filter (only available products)
	query := r.db.Model(&models.Product{}).Where("tenant_id = ? AND stock_quantity > 0", tenantID)

	// Add search filter if provided
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where(
			"LOWER(name) LIKE LOWER(?) OR LOWER(sku) LIKE LOWER(?) OR LOWER(description) LIKE LOWER(?)",
			searchPattern, searchPattern, searchPattern,
		)
	}

	// Get total count with filters
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated data with filters, ordered by name
	err := query.Order("name ASC").Limit(limit).Offset(offset).Find(&products).Error
	if err != nil {
		return nil, err
	}

	page := (offset / limit) + 1
	if limit == 0 {
		page = 1
	}
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if limit == 0 {
		totalPages = 1
	}

	return &PaginationResult[models.Product]{
		Data:       products,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}

// ListWithSearchAdmin lists ALL products (including out of stock) for administrative purposes
func (r *ProductRepository) ListWithSearchAdmin(tenantID uuid.UUID, limit, offset int, search string) (*PaginationResult[models.Product], error) {
	var products []models.Product
	var total int64

	// Build base query with tenant filter ONLY (no stock filter for admin)
	query := r.db.Model(&models.Product{}).Where("tenant_id = ?", tenantID)

	// Add search filter if provided
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where(
			"LOWER(name) LIKE LOWER(?) OR LOWER(sku) LIKE LOWER(?) OR LOWER(description) LIKE LOWER(?)",
			searchPattern, searchPattern, searchPattern,
		)
	}

	// Get total count with filters
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated data with filters, ordered by name
	err := query.Order("name ASC").Limit(limit).Offset(offset).Find(&products).Error
	if err != nil {
		return nil, err
	}

	page := (offset / limit) + 1
	if limit == 0 {
		page = 1
	}
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if limit == 0 {
		totalPages = 1
	}

	return &PaginationResult[models.Product]{
		Data:       products,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}

// ListWithSearchAndFiltersAdmin lists ALL products with advanced filters for administrative purposes
func (r *ProductRepository) ListWithSearchAndFiltersAdmin(tenantID uuid.UUID, limit, offset int, search string, filters ProductFilters) (*PaginationResult[models.Product], error) {
	var products []models.Product
	var total int64

	// Build base query with tenant filter ONLY (no stock filter for admin)
	query := r.db.Model(&models.Product{}).Where("tenant_id = ?", tenantID)

	// Add search filter if provided
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where(
			"LOWER(name) LIKE LOWER(?) OR LOWER(sku) LIKE LOWER(?) OR LOWER(description) LIKE LOWER(?)",
			searchPattern, searchPattern, searchPattern,
		)
	}

	// Apply advanced filters
	if filters.CategoryID != nil && *filters.CategoryID != "" {
		query = query.Where("category_id = ?", *filters.CategoryID)
	}

	if filters.MinPrice != nil && *filters.MinPrice > 0 {
		query = query.Where("price::NUMERIC >= ?", *filters.MinPrice)
	}

	if filters.MaxPrice != nil && *filters.MaxPrice > 0 {
		query = query.Where("price::NUMERIC <= ?", *filters.MaxPrice)
	}

	if filters.HasPromotion != nil && *filters.HasPromotion {
		query = query.Where("sale_price IS NOT NULL AND sale_price != '' AND sale_price != '0'")
	}

	if filters.HasSKU != nil && *filters.HasSKU {
		query = query.Where("sku IS NOT NULL AND sku != ''")
	}

	if filters.HasStock != nil && *filters.HasStock {
		query = query.Where("stock_quantity IS NOT NULL AND stock_quantity > 0")
	}

	if filters.OutOfStock != nil && *filters.OutOfStock {
		query = query.Where("stock_quantity IS NULL OR stock_quantity <= 0")
	}

	// Get total count with filters
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated data with filters, ordered by name
	err := query.Order("name ASC").Limit(limit).Offset(offset).Find(&products).Error
	if err != nil {
		return nil, err
	}

	page := (offset / limit) + 1
	if limit == 0 {
		page = 1
	}
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if limit == 0 {
		totalPages = 1
	}

	return &PaginationResult[models.Product]{
		Data:       products,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}

// Delete deletes a product by ID
func (r *ProductRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Product{}, id).Error
}

// CustomerRepository handles customer data access
type CustomerRepository struct {
	db *gorm.DB
}

// NewCustomerRepository creates a new customer repository
func NewCustomerRepository(db *gorm.DB) *CustomerRepository {
	return &CustomerRepository{db: db}
}

// GetByID gets a customer by ID
func (r *CustomerRepository) GetByID(tenantID, id uuid.UUID) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.Where("id = ? AND tenant_id = ?", id, tenantID).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// GetByPhone gets a customer by phone
func (r *CustomerRepository) GetByPhone(tenantID uuid.UUID, phone string) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.Where("phone = ? AND tenant_id = ?", phone, tenantID).First(&customer).Error
	if err != nil {
		return nil, err
	}
	return &customer, nil
}

// Create creates a new customer
func (r *CustomerRepository) Create(customer *models.Customer) error {
	return r.db.Create(customer).Error
}

// Update updates a customer
func (r *CustomerRepository) Update(customer *models.Customer) error {
	return r.db.Save(customer).Error
}

// List lists customers with pagination
func (r *CustomerRepository) List(tenantID uuid.UUID, limit, offset int) (*PaginationResult[models.Customer], error) {
	var customers []models.Customer
	var total int64

	// Get total count with tenant filter
	if err := r.db.Model(&models.Customer{}).Where("tenant_id = ?", tenantID).Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated data with tenant filter
	err := r.db.Where("tenant_id = ?", tenantID).Limit(limit).Offset(offset).Find(&customers).Error
	if err != nil {
		return nil, err
	}

	page := (offset / limit) + 1
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &PaginationResult[models.Customer]{
		Data:       customers,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}

// ListWithSearch lists customers with pagination and search
func (r *CustomerRepository) ListWithSearch(tenantID uuid.UUID, limit, offset int, search string) (*PaginationResult[models.Customer], error) {
	var customers []models.Customer
	var total int64

	query := r.db.Model(&models.Customer{}).Where("tenant_id = ?", tenantID)

	// Apply search filter if provided
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("name ILIKE ? OR phone LIKE ? OR email ILIKE ? OR document LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	// Get total count with search filter
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated data with search filter
	err := query.Limit(limit).Offset(offset).Find(&customers).Error
	if err != nil {
		return nil, err
	}

	page := (offset / limit) + 1
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &PaginationResult[models.Customer]{
		Data:       customers,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}

// OrderRepository handles order data access
type OrderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository(db *gorm.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// GetDB returns the database instance for complex queries
func (r *OrderRepository) GetDB() *gorm.DB {
	return r.db
}

// GetByID gets an order by ID
func (r *OrderRepository) GetByID(tenantID, id uuid.UUID) (*models.Order, error) {
	var order models.Order
	err := r.db.Preload("Customer").
		Preload("Address").
		Preload("PaymentMethod").
		Preload("Items").
		Preload("Items.Product").
		Preload("Items.Attributes").
		Where("id = ? AND tenant_id = ?", id, tenantID).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// Create creates a new order
func (r *OrderRepository) Create(order *models.Order) error {
	return r.db.Create(order).Error
}

// Update updates an order - only updates non-zero fields to prevent data loss
func (r *OrderRepository) Update(order *models.Order) error {
	// Use Updates instead of Save to prevent overwriting existing fields with zero values
	// This ensures that fields not included in the update request are preserved
	return r.db.Model(order).Where("id = ? AND tenant_id = ?", order.ID, order.TenantID).Updates(order).Error
}

// List lists orders with pagination
func (r *OrderRepository) List(tenantID uuid.UUID, limit, offset int) (*PaginationResult[models.Order], error) {
	var orders []models.Order
	var total int64

	// Get total count with tenant filter
	if err := r.db.Model(&models.Order{}).Where("tenant_id = ?", tenantID).Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated data with tenant filter
	err := r.db.Preload("Customer").
		Preload("Address").
		Preload("PaymentMethod").
		Preload("Items").
		Where("tenant_id = ?", tenantID).Limit(limit).Offset(offset).Find(&orders).Error
	if err != nil {
		return nil, err
	}

	page := (offset / limit) + 1
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &PaginationResult[models.Order]{
		Data:       orders,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}

// OrderWithCustomer represents an order with customer information
type OrderWithCustomer struct {
	models.Order
	CustomerName      string `json:"customer_name"`
	CustomerEmail     string `json:"customer_email"`
	PaymentMethodName string `json:"payment_method_name"`
	ItemsCount        int    `json:"items_count"`
}

// ListWithCustomers lists orders with customer information and pagination
func (r *OrderRepository) ListWithCustomers(tenantID uuid.UUID, limit, offset int) (*PaginationResult[OrderWithCustomer], error) {
	var orders []OrderWithCustomer
	var total int64

	// Get total count with tenant filter
	if err := r.db.Model(&models.Order{}).Where("tenant_id = ?", tenantID).Count(&total).Error; err != nil {
		return nil, err
	}

	// Get paginated data with customer information
	err := r.db.Table("orders").
		Select(`orders.*, 
			customers.name as customer_name, 
			customers.email as customer_email,
			payment_methods.name as payment_method_name,
			COALESCE((SELECT SUM(quantity) FROM order_items WHERE order_id = orders.id), 0) as items_count`).
		Joins("LEFT JOIN customers ON customers.id = orders.customer_id").
		Joins("LEFT JOIN payment_methods ON payment_methods.id = orders.payment_method_id").
		Where("orders.tenant_id = ?", tenantID).
		Order("orders.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error

	if err != nil {
		return nil, err
	}

	page := (offset / limit) + 1
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &PaginationResult[OrderWithCustomer]{
		Data:       orders,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}

// ListWithCustomersAndSearch lists orders with customer information, pagination and search
func (r *OrderRepository) ListWithCustomersAndSearch(tenantID uuid.UUID, limit, offset int, search string) (*PaginationResult[OrderWithCustomer], error) {
	var orders []OrderWithCustomer
	var total int64

	// Base query for counting with search filter
	countQuery := r.db.Model(&models.Order{}).Where("orders.tenant_id = ?", tenantID)

	// Apply search filter if provided
	if search != "" {
		searchPattern := "%" + search + "%"
		countQuery = countQuery.Joins("LEFT JOIN customers ON customers.id = orders.customer_id").
			Where("(orders.order_number ILIKE ? OR customers.name ILIKE ? OR customers.phone ILIKE ? OR customers.email ILIKE ?)",
				searchPattern, searchPattern, searchPattern, searchPattern)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}

	// Build main query with customer information
	query := r.db.Table("orders").
		Select(`orders.*, 
			customers.name as customer_name, 
			customers.email as customer_email,
			payment_methods.name as payment_method_name,
			COALESCE((SELECT SUM(quantity) FROM order_items WHERE order_id = orders.id), 0) as items_count`).
		Joins("LEFT JOIN customers ON customers.id = orders.customer_id").
		Joins("LEFT JOIN payment_methods ON payment_methods.id = orders.payment_method_id").
		Where("orders.tenant_id = ?", tenantID)

	// Apply search filter if provided
	if search != "" {
		searchPattern := "%" + search + "%"
		query = query.Where("(orders.order_number ILIKE ? OR customers.name ILIKE ? OR customers.phone ILIKE ? OR customers.email ILIKE ?)",
			searchPattern, searchPattern, searchPattern, searchPattern)
	}

	err := query.Order("orders.created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&orders).Error

	if err != nil {
		return nil, err
	}

	page := (offset / limit) + 1
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return &PaginationResult[OrderWithCustomer]{
		Data:       orders,
		Total:      total,
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
	}, nil
}
