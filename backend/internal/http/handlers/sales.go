package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"iafarma/internal/ai"
	"iafarma/internal/repo"
	"iafarma/internal/services"
	"iafarma/internal/zapplus"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// generateOrderNumber generates a unique order number
func generateOrderNumber() string {
	now := time.Now()
	return fmt.Sprintf("ORD-%d%02d%02d-%04d",
		now.Year(),
		now.Month(),
		now.Day(),
		now.Unix()%10000,
	)
}

// cleanPhoneNumber removes all non-numeric characters from phone number
func cleanPhoneNumber(phone string) string {
	reg := regexp.MustCompile(`[^0-9]`)
	return reg.ReplaceAllString(phone, "")
}

// generateUniqueSKU generates a unique SKU based on product name
func generateUniqueSKU(db *gorm.DB, tenantID uuid.UUID, productName string) string {
	// Clean product name to create a base SKU
	cleaned := strings.ToUpper(productName)
	// Remove special characters and keep only letters and numbers
	reg := regexp.MustCompile(`[^A-Z0-9]`)
	cleaned = reg.ReplaceAllString(cleaned, "")

	// Limit to first 8 characters
	if len(cleaned) > 8 {
		cleaned = cleaned[:8]
	}

	// If too short, pad with "PROD"
	if len(cleaned) < 4 {
		cleaned = "PROD" + cleaned
	}

	// Try with just the base name first
	baseSKU := cleaned
	if !skuExists(db, tenantID, baseSKU) {
		return baseSKU
	}

	// If exists, try with random suffix
	for i := 0; i < 10; i++ {
		// Generate random 4-character suffix
		randomBytes := make([]byte, 2)
		rand.Read(randomBytes)
		suffix := hex.EncodeToString(randomBytes)
		suffix = strings.ToUpper(suffix)

		candidate := baseSKU + suffix
		if !skuExists(db, tenantID, candidate) {
			return candidate
		}
	}

	// Fallback: use UUID-based SKU
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	return "SKU" + strings.ToUpper(hex.EncodeToString(randomBytes))
}

// skuExists checks if a SKU already exists for the tenant
func skuExists(db *gorm.DB, tenantID uuid.UUID, sku string) bool {
	var count int64
	db.Model(&models.Product{}).Where("tenant_id = ? AND sku = ?", tenantID, sku).Count(&count)
	return count > 0
}

// ProductHandler handles product operations
type ProductHandler struct {
	productRepo      *repo.ProductRepository
	categoryRepo     *repo.CategoryRepository
	embeddingService *services.EmbeddingService
	planLimitService *services.PlanLimitService
	storageService   *services.StorageService
	db               *gorm.DB
}

// NewProductHandler creates a new product handler
func NewProductHandler(productRepo *repo.ProductRepository, categoryRepo *repo.CategoryRepository, embeddingService *services.EmbeddingService, planLimitService *services.PlanLimitService, storageService *services.StorageService, db *gorm.DB) *ProductHandler {
	return &ProductHandler{
		productRepo:      productRepo,
		categoryRepo:     categoryRepo,
		embeddingService: embeddingService,
		planLimitService: planLimitService,
		storageService:   storageService,
		db:               db,
	}
}

// List godoc
// @Summary List products
// @Description Get list of products with pagination and search
// @Tags products
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Param search query string false "Search term for name, SKU, or description"
// @Success 200 {object} models.ProductListResponse
// @Failure 500 {object} map[string]string
// @Router /products [get]
// @Security BearerAuth
func (h *ProductHandler) List(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	search := strings.TrimSpace(c.QueryParam("search"))

	if limit <= 0 {
		limit = 20
	}

	result, err := h.productRepo.ListWithSearch(tenantID, limit, offset, search)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// ListAdmin godoc
// @Summary List all products (including out of stock) for admin
// @Description Get list of ALL products including those with zero stock for administrative purposes
// @Tags products
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Param search query string false "Search term for name, SKU, or description"
// @Success 200 {object} models.ProductListResponse
// @Failure 500 {object} map[string]string
// @Router /products/admin [get]
// @Security BearerAuth
func (h *ProductHandler) ListAdmin(c echo.Context) error {
	// Get user role from context
	userRole, _ := c.Get("user_role").(string)

	// Get tenant ID from context (for normal admins) or query parameter (for system_admin)
	var tenantID uuid.UUID
	var ok bool

	if userRole == "system_admin" {
		// For system_admin, tenant_id must be provided as query parameter
		tenantIDParam := c.QueryParam("tenant_id")
		if tenantIDParam == "" {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id query parameter required for system_admin"})
		}

		var err error
		tenantID, err = uuid.Parse(tenantIDParam)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid tenant_id format"})
		}
	} else {
		// For regular admins, get tenant from context
		tenantID, ok = c.Get("tenant_id").(uuid.UUID)
		if !ok {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
		}
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	search := strings.TrimSpace(c.QueryParam("search"))

	if limit <= 0 {
		limit = 20
	}

	// Parse filters from query parameters
	filters := repo.ProductFilters{}

	if categoryID := c.QueryParam("category_id"); categoryID != "" {
		filters.CategoryID = &categoryID
	}

	if minPriceStr := c.QueryParam("min_price"); minPriceStr != "" {
		if minPrice, err := strconv.ParseFloat(minPriceStr, 64); err == nil {
			filters.MinPrice = &minPrice
		}
	}

	if maxPriceStr := c.QueryParam("max_price"); maxPriceStr != "" {
		if maxPrice, err := strconv.ParseFloat(maxPriceStr, 64); err == nil {
			filters.MaxPrice = &maxPrice
		}
	}

	if hasPromotionStr := c.QueryParam("has_promotion"); hasPromotionStr == "true" {
		hasPromotion := true
		filters.HasPromotion = &hasPromotion
	}

	if hasSKUStr := c.QueryParam("has_sku"); hasSKUStr == "true" {
		hasSKU := true
		filters.HasSKU = &hasSKU
	}

	if hasStockStr := c.QueryParam("has_stock"); hasStockStr == "true" {
		hasStock := true
		filters.HasStock = &hasStock
	}

	if outOfStockStr := c.QueryParam("out_of_stock"); outOfStockStr == "true" {
		outOfStock := true
		filters.OutOfStock = &outOfStock
	}

	// Check if any filters are applied, if yes use the new method
	hasFilters := filters.CategoryID != nil || filters.MinPrice != nil || filters.MaxPrice != nil ||
		filters.HasPromotion != nil || filters.HasSKU != nil || filters.HasStock != nil || filters.OutOfStock != nil

	var result *repo.PaginationResult[models.Product]
	var err error

	if hasFilters {
		result, err = h.productRepo.ListWithSearchAndFiltersAdmin(tenantID, limit, offset, search, filters)
	} else {
		result, err = h.productRepo.ListWithSearchAdmin(tenantID, limit, offset, search)
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// Search godoc
// @Summary Semantic search products
// @Description Search products using semantic similarity
// @Tags products
// @Accept json
// @Produce json
// @Param query query string true "Search query"
// @Param limit query int false "Limit" default(10)
// @Success 200 {object} []services.ProductSearchResult
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/search [get]
// @Security BearerAuth
func (h *ProductHandler) Search(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	query := c.QueryParam("query")
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Query parameter is required"})
	}

	limitParam := c.QueryParam("limit")
	limit := uint64(10) // default
	if limitParam != "" {
		if l, err := strconv.ParseUint(limitParam, 10, 64); err == nil {
			limit = l
		}
	}

	if h.embeddingService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{"error": "Search service not available"})
	}

	results, err := h.embeddingService.SearchSimilarProducts(query, tenantID.String(), limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Se não há resultados semânticos, fazer busca tradicional como fallback
	if len(results) == 0 {
		return h.List(c) // Fallback para busca tradicional
	}

	return c.JSON(http.StatusOK, results)
}

// GetByID godoc
// @Summary Get product by ID
// @Description Get a product by its ID
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} models.Product
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /products/{id} [get]
// @Security BearerAuth
func (h *ProductHandler) GetByID(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	product, err := h.productRepo.GetByID(tenantID, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Product not found"})
	}

	return c.JSON(http.StatusOK, product)
}

// GetStats godoc
// @Summary Get product statistics
// @Description Get comprehensive product statistics for a tenant
// @Tags products
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /products/stats [get]
// @Security BearerAuth
func (h *ProductHandler) GetStats(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	var stats struct {
		Total           int64   `json:"total"`
		WithSku         int64   `json:"with_sku"`
		WithPromotion   int64   `json:"with_promotion"`
		AveragePrice    float64 `json:"average_price"`
		TotalValue      float64 `json:"total_value"`
		CategoriesCount int64   `json:"categories_count"`
		OutOfStock      int64   `json:"out_of_stock"`
		RecentAdded     int64   `json:"recent_added"`
	}

	// Total products
	if err := h.db.Model(&models.Product{}).Where("tenant_id = ? AND deleted_at IS NULL", tenantID).Count(&stats.Total).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get total products"})
	}

	// Products with SKU
	if err := h.db.Model(&models.Product{}).Where("tenant_id = ? AND deleted_at IS NULL AND sku IS NOT NULL AND sku != ''", tenantID).Count(&stats.WithSku).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get products with SKU"})
	}

	// Products with promotion (sale_price)
	if err := h.db.Model(&models.Product{}).Where("tenant_id = ? AND deleted_at IS NULL AND sale_price IS NOT NULL AND sale_price != '' AND sale_price != '0'", tenantID).Count(&stats.WithPromotion).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get products with promotion"})
	}

	// Average price and total value
	var result struct {
		AveragePrice float64
		TotalValue   float64
	}

	// Simplified query that handles numeric price strings
	query := `
		SELECT 
			COALESCE(AVG(CASE 
				WHEN price ~ '^[0-9]+(\.[0-9]+)?$' THEN price::NUMERIC
				ELSE NULL 
			END), 0) as average_price,
			COALESCE(SUM(CASE 
				WHEN price ~ '^[0-9]+(\.[0-9]+)?$' THEN price::NUMERIC
				ELSE 0 
			END), 0) as total_value
		FROM products 
		WHERE tenant_id = $1 AND deleted_at IS NULL AND price IS NOT NULL AND price != ''
	`

	if err := h.db.Raw(query, tenantID).Scan(&result).Error; err != nil {
		// Fallback to simpler calculation without regex
		fallbackQuery := `
			SELECT 
				COALESCE(AVG(price::NUMERIC), 0) as average_price,
				COALESCE(SUM(price::NUMERIC), 0) as total_value
			FROM products 
			WHERE tenant_id = $1 AND deleted_at IS NULL 
			AND price IS NOT NULL AND price != '' AND price ~ '^[0-9]+(\.[0-9]+)?$'
		`
		if err2 := h.db.Raw(fallbackQuery, tenantID).Scan(&result).Error; err2 != nil {
			// If both fail, just set to 0
			result.AveragePrice = 0
			result.TotalValue = 0
		}
	}

	stats.AveragePrice = result.AveragePrice
	stats.TotalValue = result.TotalValue

	// Categories count
	if err := h.db.Raw("SELECT COUNT(DISTINCT category_id) FROM products WHERE tenant_id = ? AND deleted_at IS NULL AND category_id IS NOT NULL", tenantID).Scan(&stats.CategoriesCount).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get categories count"})
	}

	// Out of stock products
	if err := h.db.Model(&models.Product{}).Where("tenant_id = ? AND deleted_at IS NULL AND (stock_quantity IS NULL OR stock_quantity <= 0)", tenantID).Count(&stats.OutOfStock).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get out of stock products"})
	}

	// Recent added (last 30 days)
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	if err := h.db.Model(&models.Product{}).Where("tenant_id = ? AND deleted_at IS NULL AND created_at > ?", tenantID, thirtyDaysAgo).Count(&stats.RecentAdded).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get recent added products"})
	}

	return c.JSON(http.StatusOK, stats)
}

// Create godoc
// @Summary Create product
// @Description Create a new product
// @Tags products
// @Accept json
// @Produce json
// @Param product body models.Product true "Product data"
// @Success 201 {object} models.Product
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products [post]
// @Security BearerAuth
func (h *ProductHandler) Create(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	// Validate plan limits before creating product
	if h.planLimitService != nil {
		if err := h.planLimitService.ValidateProductLimit(tenantID, 1); err != nil {
			return c.JSON(http.StatusForbidden, map[string]string{"error": err.Error()})
		}
	}

	var product models.Product
	if err := c.Bind(&product); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Set tenant ID for the product
	product.TenantID = tenantID

	// Clean empty numeric fields to prevent SQL errors
	h.cleanProductFields(&product)

	if err := h.productRepo.Create(&product); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Gerar embedding assincronamente com cache e rate limiting
	if h.embeddingService != nil {
		go func() {
			if err := h.storeProductEmbeddingWithCache(&product, tenantID.String()); err != nil {
				// Log error but don't fail the request
				log.Printf("Failed to store embedding for product %s: %v", product.ID, err)
			}
		}()
	}

	return c.JSON(http.StatusCreated, product)
}

// ImportProducts godoc
// @Summary Import products in bulk
// @Description Import multiple products from JSON data, with upsert logic
// @Tags products
// @Accept json
// @Produce json
// @Param products body models.ProductImportRequest true "Products import data"
// @Success 200 {object} models.ProductImportResult
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/import [post]
// @Security BearerAuth
func (h *ProductHandler) ImportProducts(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	var importRequest models.ProductImportRequest
	if err := c.Bind(&importRequest); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	if len(importRequest.Products) == 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No products provided"})
	}

	// Check how many products can be added
	var newProductsCount int
	availableProducts := 0

	if h.planLimitService != nil {
		var err error
		availableProducts, err = h.planLimitService.GetProductsAvailable(tenantID)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check plan limits"})
		}
	}

	result := models.ProductImportResult{
		TotalProcessed: len(importRequest.Products),
		Results:        make([]models.ProductImportItemResult, 0, len(importRequest.Products)),
	}

	// Process each product
	for i, item := range importRequest.Products {
		item.Name = strings.TrimSpace(strings.ReplaceAll(item.Name, "*", ""))
		item.Description = strings.TrimSpace(strings.ReplaceAll(item.Description, "*", ""))

		rowResult := models.ProductImportItemResult{
			RowNumber: i + 1,
			Name:      item.Name,
			SKU:       item.SKU,
		}

		// Validate required fields
		if item.Name == "" {
			rowResult.Status = "error"
			rowResult.Error = "Nome é obrigatório"
			result.Results = append(result.Results, rowResult)
			result.Errors++
			continue
		}

		if item.Price == "" {
			rowResult.Status = "error"
			rowResult.Error = "Preço é obrigatório"
			result.Results = append(result.Results, rowResult)
			result.Errors++
			continue
		}

		// Check if this would be a new product (by checking if SKU/name exists)
		isNewProduct := true
		if item.SKU != "" || item.Name != "" {
			existingProduct, err := h.productRepo.FindExistingProduct(tenantID, item.Name, item.SKU, "")
			if err == nil && existingProduct != nil {
				isNewProduct = false
			}
		}

		// If it's a new product, check plan limits
		if isNewProduct && h.planLimitService != nil {
			if newProductsCount >= availableProducts {
				rowResult.Status = "error"
				rowResult.Error = fmt.Sprintf("Limite de produtos atingido. Plano atual permite adicionar apenas %d produtos", availableProducts)
				result.Results = append(result.Results, rowResult)
				result.Errors++
				continue
			}
			newProductsCount++
		}

		// Handle category if provided
		var categoryID *uuid.UUID
		if item.CategoryName != "" {
			catID, err := h.findOrCreateCategory(tenantID, item.CategoryName)
			if err != nil {
				rowResult.Status = "error"
				rowResult.Error = fmt.Sprintf("Erro ao processar categoria '%s': %v", item.CategoryName, err)
				result.Results = append(result.Results, rowResult)
				result.Errors++
				continue
			}
			categoryID = catID
		}

		// Create product from import item
		product := models.Product{
			BaseTenantModel: models.BaseTenantModel{
				TenantID: tenantID,
			},
			CategoryID:        categoryID,
			Name:              item.Name,
			Description:       item.Description,
			Price:             item.Price,
			SalePrice:         item.SalePrice,
			SKU:               item.SKU,
			Barcode:           item.Barcode,
			Weight:            item.Weight,
			Dimensions:        item.Dimensions,
			Brand:             item.Brand,
			Tags:              item.Tags,
			StockQuantity:     item.StockQuantity,
			LowStockThreshold: item.LowStockThreshold,
		}

		// Clean fields
		h.cleanProductFields(&product)

		// Try to upsert the product
		savedProduct, isNew, err := h.productRepo.UpsertProduct(&product)
		if err != nil {
			rowResult.Status = "error"
			rowResult.Error = err.Error()
			result.Results = append(result.Results, rowResult)
			result.Errors++
			continue
		}

		// Success
		productIDStr := savedProduct.ID.String()
		rowResult.ProductID = &productIDStr

		if isNew {
			rowResult.Status = "created"
			rowResult.Message = "Produto criado com sucesso"
			result.Created++
		} else {
			rowResult.Status = "updated"
			rowResult.Message = "Produto atualizado com sucesso"
			result.Updated++
		}

		result.Results = append(result.Results, rowResult)
	}

	// Process embeddings in batch after all products are saved
	if h.embeddingService != nil && (result.Created > 0 || result.Updated > 0) {
		log.Printf("🔄 Starting async embedding processing for import - embeddingService available")
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("❌ PANIC in embedding processing goroutine: %v", r)
				}
			}()

			log.Printf("🚀 Starting embedding processing for %d results (created: %d, updated: %d)",
				len(result.Results), result.Created, result.Updated)

			if err := h.processBatchEmbeddingsForImport(tenantID, result.Results); err != nil {
				log.Printf("❌ Failed to process batch embeddings for import: %v", err)
			} else {
				log.Printf("✅ Successfully completed batch embedding processing for import")
			}
		}()
	} else if h.embeddingService == nil {
		log.Printf("⚠️  Embedding service is nil - skipping embedding processing")
	} else {
		log.Printf("ℹ️  No products to process embeddings for (created: %d, updated: %d)", result.Created, result.Updated)
	}

	log.Printf("✅ Import completed: %d created, %d updated, %d errors", result.Created, result.Updated, result.Errors)

	return c.JSON(http.StatusOK, result)
}

// processBatchEmbeddingsForImport processes embeddings in batches for imported products
func (h *ProductHandler) processBatchEmbeddingsForImport(tenantID uuid.UUID, results []models.ProductImportItemResult) error {
	log.Printf("🔄 Starting batch embedding processing for import (tenant: %s)", tenantID.String())

	// Collect all successful product IDs
	var productIDs []string
	for _, result := range results {
		if result.Status != "error" && result.ProductID != nil {
			productIDs = append(productIDs, *result.ProductID)
		}
	}

	log.Printf("📊 Import results analysis: %d total results, %d successful product IDs collected",
		len(results), len(productIDs))

	if len(productIDs) == 0 {
		log.Printf("ℹ️ No successful products to process embeddings for")
		return nil
	}

	// Fetch products from database with their current hashes
	var products []models.Product
	err := h.db.Where("id IN ? AND tenant_id = ?", productIDs, tenantID).Find(&products).Error
	if err != nil {
		log.Printf("❌ Failed to fetch products from database: %v", err)
		return fmt.Errorf("failed to fetch products for batch processing: %w", err)
	}

	log.Printf("📦 Found %d products in database for batch embedding processing", len(products))

	if len(products) == 0 {
		log.Printf("⚠️ No products found in database despite having successful import results")
		return fmt.Errorf("no products found in database for processing")
	}

	// Separate products that need embedding updates (cache miss) vs those that don't (cache hit)
	var productsNeedingEmbedding []models.Product
	cacheHits := 0

	for _, product := range products {
		searchText := product.GetSearchText()
		hash := h.calculateContentHash(searchText)

		log.Printf("🔍 Product %s: current hash='%s', calculated hash='%s'",
			product.ID, product.EmbeddingHash, hash)

		if product.EmbeddingHash != hash {
			// Cache miss - needs new embedding
			productsNeedingEmbedding = append(productsNeedingEmbedding, product)
			log.Printf("➕ Product %s needs embedding update (cache miss)", product.ID)
		} else {
			// Cache hit - embedding is up to date
			cacheHits++
			log.Printf("✅ Product %s has up-to-date embedding (cache hit)", product.ID)
		}
	}

	log.Printf("✅ Cache hits: %d, Cache misses: %d", cacheHits, len(productsNeedingEmbedding))

	if len(productsNeedingEmbedding) == 0 {
		log.Printf("✅ All products have up-to-date embeddings (100%% cache hit rate)")
		return nil
	}

	// Process in batches to avoid rate limits
	batchSize := 500 // Optimized batch size for better performance
	for i := 0; i < len(productsNeedingEmbedding); i += batchSize {
		end := i + batchSize
		if end > len(productsNeedingEmbedding) {
			end = len(productsNeedingEmbedding)
		}

		batch := productsNeedingEmbedding[i:end]

		log.Printf("🔄 Processing batch %d/%d (%d products)", (i/batchSize)+1, (len(productsNeedingEmbedding)+batchSize-1)/batchSize, len(batch))

		// Process batch with delay between batches to respect rate limits
		if err := h.processSingleBatch(batch, tenantID.String()); err != nil {
			log.Printf("❌ Failed to process batch %d: %v", (i/batchSize)+1, err)
			// Continue with next batch instead of failing completely
			continue
		}

		// Add delay between batches to respect rate limits (3000 RPM = 50 RPS max)
		if i+batchSize < len(productsNeedingEmbedding) {
			time.Sleep(2 * time.Second) // 2 second delay between batches
		}
	}

	log.Printf("✅ Batch embedding processing completed for tenant %s", tenantID.String())
	return nil
}

// processSingleBatch processes a single batch of products
func (h *ProductHandler) processSingleBatch(batch []models.Product, tenantID string) error {
	// Pre-filter products that need processing (hash changed or missing)
	var productsToProcess []models.Product
	var skippedCount int

	for _, product := range batch {
		searchText := product.GetSearchText()
		currentHash := h.calculateContentHash(searchText)

		// Check if product needs reprocessing
		if product.EmbeddingHash != "" && product.EmbeddingHash == currentHash {
			skippedCount++
			log.Printf("⏭️ Skipping product %s (hash unchanged: %s)", product.ID, currentHash)
			continue
		}

		productsToProcess = append(productsToProcess, product)
	}

	log.Printf("📊 Batch analysis: %d products to process, %d skipped (hash unchanged)",
		len(productsToProcess), skippedCount)

	// Skip batch if no products need processing
	if len(productsToProcess) == 0 {
		log.Printf("✅ Batch complete - all products up to date")
		return nil
	}

	// Prepare batch data for processing
	var batchProducts []services.BatchProductData
	var productHashes []struct {
		ProductID uuid.UUID
		Hash      string
	}

	for _, product := range productsToProcess {
		searchText := product.GetSearchText()
		hash := h.calculateContentHash(searchText)
		metadata := product.GetMetadata()

		batchProducts = append(batchProducts, services.BatchProductData{
			ID:       product.ID.String(),
			TenantID: product.TenantID.String(),
			Text:     searchText,
			Metadata: metadata,
		})

		productHashes = append(productHashes, struct {
			ProductID uuid.UUID
			Hash      string
		}{
			ProductID: product.ID,
			Hash:      hash,
		})
	}

	// Store batch embeddings (this will generate embeddings and store them)
	log.Printf("🚀 Calling embeddingService.StoreBatchProductEmbeddings with %d products for tenant %s",
		len(batchProducts), tenantID)

	err := h.embeddingService.StoreBatchProductEmbeddings(tenantID, batchProducts, len(batchProducts))
	if err != nil {
		log.Printf("❌ embeddingService.StoreBatchProductEmbeddings failed: %v", err)
		return fmt.Errorf("failed to store batch embeddings: %w", err)
	}

	log.Printf("✅ embeddingService.StoreBatchProductEmbeddings completed successfully")

	// Update hashes in database after successful embedding storage
	log.Printf("🔄 Updating embedding hashes for %d products", len(productHashes))
	for i, productHash := range productHashes {
		log.Printf("🔄 Updating hash for product %d/%d: %s -> %s",
			i+1, len(productHashes), productHash.ProductID, productHash.Hash)
		h.updateProductEmbeddingHash(productHash.ProductID, productHash.Hash)
	}

	log.Printf("✅ Processed batch of %d products", len(batch))
	return nil
}

// processBatchEmbeddingsForImageImport processes embeddings in batches for image-imported products
func (h *ProductHandler) processBatchEmbeddingsForImageImport(tenantID uuid.UUID, products []models.ProductImageImportProduct) error {
	log.Printf("🔄 Starting batch embedding processing for image import (tenant: %s)", tenantID.String())

	// Get IDs of successful products - we need to query database to get the actual Product objects
	// Since ProductImageImportProduct doesn't have ID, we'll search by name
	var dbProducts []models.Product
	for _, product := range products {
		if product.Status == "success" {
			var dbProduct models.Product
			err := h.db.Where("tenant_id = ? AND name = ?", tenantID, product.Name).Order("created_at DESC").First(&dbProduct).Error
			if err == nil {
				dbProducts = append(dbProducts, dbProduct)
			} else {
				log.Printf("⚠️ Could not find database product for '%s': %v", product.Name, err)
			}
		}
	}

	if len(dbProducts) == 0 {
		log.Printf("ℹ️ No products found in database for image import batch processing")
		return nil
	}

	log.Printf("📦 Found %d products for batch embedding processing", len(dbProducts))

	// All image-imported products are new, so they all need embeddings (no cache check needed)
	// Process in batches
	batchSize := 500
	for i := 0; i < len(dbProducts); i += batchSize {
		end := i + batchSize
		if end > len(dbProducts) {
			end = len(dbProducts)
		}

		batch := dbProducts[i:end]

		log.Printf("🔄 Processing image import batch %d/%d (%d products)", (i/batchSize)+1, (len(dbProducts)+batchSize-1)/batchSize, len(batch))

		if err := h.processSingleBatch(batch, tenantID.String()); err != nil {
			log.Printf("❌ Failed to process image import batch %d: %v", (i/batchSize)+1, err)
			continue
		}

		// Add delay between batches
		if i+batchSize < len(dbProducts) {
			time.Sleep(2 * time.Second)
		}
	}

	log.Printf("✅ Batch embedding processing completed for image import (tenant: %s)", tenantID.String())
	return nil
}

// GetImportTemplate godoc
// @Summary Get import template
// @Description Download a CSV template for product import
// @Tags products
// @Produce text/csv
// @Success 200 {file} string "CSV template file"
// @Router /products/import/template [get]
// @Security BearerAuth
func (h *ProductHandler) GetImportTemplate(c echo.Context) error {
	// CSV header and example data
	csvContent := `name,description,price,sale_price,sku,barcode,weight,dimensions,brand,tags,stock_quantity,low_stock_threshold,category_name
"Caneta Esferográfica Azul","Caneta esferográfica com tinta azul de alta qualidade","2.50","2.00","CAN001","7894567890123","10g","14cm","BIC","caneta,escolar,azul",100,10,"Canetas"
"Caderno Universitário 200 folhas","Caderno espiral universitário com 200 folhas pautadas","15.90","","CAD001","7894567890124","300g","28x20x1.5cm","Tilibra","caderno,universitário,espiral",50,5,"Cadernos"
"Lápis HB nº 2","Lápis grafite HB número 2 para escrita","1.20","0.90","LAP001","7894567890125","8g","18cm","Faber-Castell","lápis,grafite,escolar",200,20,"Lápis"`

	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", "attachment; filename=template_importacao_produtos.csv")

	return c.String(http.StatusOK, csvContent)
}

// Update godoc
// @Summary Update product
// @Description Update an existing product
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param product body models.Product true "Product data"
// @Success 200 {object} models.Product
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/{id} [put]
// @Security BearerAuth
func (h *ProductHandler) Update(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	existingProduct, err := h.productRepo.GetByID(tenantID, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Product not found"})
	}

	var updatedProduct models.Product
	if err := c.Bind(&updatedProduct); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Clean empty numeric fields
	h.cleanProductFields(&updatedProduct)

	// Preserve immutable fields
	updatedProduct.ID = existingProduct.ID
	updatedProduct.CreatedAt = existingProduct.CreatedAt
	updatedProduct.TenantID = existingProduct.TenantID
	updatedProduct.EmbeddingHash = existingProduct.EmbeddingHash // Preserve existing hash for cache comparison

	// Only update if fields are actually provided (not empty)
	if updatedProduct.Name == "" {
		updatedProduct.Name = existingProduct.Name
	}
	if updatedProduct.Description == "" {
		updatedProduct.Description = existingProduct.Description
	}
	if updatedProduct.Price == "" {
		updatedProduct.Price = existingProduct.Price
	}
	if updatedProduct.SKU == "" {
		updatedProduct.SKU = existingProduct.SKU
	}

	if err := h.productRepo.Update(&updatedProduct); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Reload product from database to get the current state including EmbeddingHash
	freshProduct, err := h.productRepo.GetByID(tenantID, id)
	if err != nil {
		log.Printf("Warning: Could not reload product after update: %v", err)
		freshProduct = &updatedProduct // Fallback to updated product
	}

	// Update product in RAG after successful database update with cache
	if h.embeddingService != nil {
		go func() {
			if err := h.storeProductEmbeddingWithCache(freshProduct, freshProduct.TenantID.String()); err != nil {
				log.Printf("Failed to update embedding for product %s: %v", freshProduct.ID, err)
			}
		}()
	}

	return c.JSON(http.StatusOK, updatedProduct)
}

// Delete godoc
// @Summary Delete product
// @Description Delete an existing product
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/{id} [delete]
// @Security BearerAuth
func (h *ProductHandler) Delete(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	// Check if product exists
	_, err = h.productRepo.GetByID(tenantID, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Product not found"})
	}

	if err := h.productRepo.Delete(id); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// Remove product from RAG after successful database deletion
	if h.embeddingService != nil {
		go func() {
			if err := h.embeddingService.DeleteProductEmbedding(id.String(), tenantID.String()); err != nil {
				log.Printf("Failed to delete embedding for product %s: %v", id, err)
			} else {
				log.Printf("Product embedding deleted successfully for product %s in tenant %s", id, tenantID)
			}
		}()
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Product deleted successfully"})
}

// storeProductEmbeddingWithCache stores product embedding with hash-based caching to avoid unnecessary OpenAI calls
func (h *ProductHandler) storeProductEmbeddingWithCache(product *models.Product, tenantID string) error {
	searchText := product.GetSearchText()

	// Calculate hash of the search text
	hash := h.calculateContentHash(searchText)

	// Check if embedding needs update
	if product.EmbeddingHash == hash {
		log.Printf("✅ CACHE HIT: Embedding already up-to-date for product %s (hash: %s)", product.ID, hash)
		return nil
	}

	log.Printf("🔄 CACHE MISS: Content changed for product %s, generating new embedding", product.ID)

	// Store embedding with retry logic
	metadata := product.GetMetadata()
	err := h.storeEmbeddingWithRetry(product.ID.String(), tenantID, searchText, metadata, 3)
	if err != nil {
		return err
	}

	// Update hash in database
	h.updateProductEmbeddingHash(product.ID, hash)

	log.Printf("Product embedding updated successfully for product %s in tenant %s", product.ID, tenantID)
	return nil
}

// storeEmbeddingWithRetry implements exponential backoff for rate limiting
func (h *ProductHandler) storeEmbeddingWithRetry(productID, tenantID, text string, metadata map[string]interface{}, maxRetries int) error {
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := h.embeddingService.StoreProductEmbedding(productID, tenantID, text, metadata)
		if err == nil {
			return nil
		}

		// Check if it's a rate limit error
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "Too Many Requests") {
			// Exponential backoff: 1s, 2s, 4s
			delay := time.Duration(1<<attempt) * time.Second
			log.Printf("Rate limit hit for product %s, retrying in %v (attempt %d/%d)", productID, delay, attempt+1, maxRetries)
			time.Sleep(delay)
			continue
		}

		// For non-rate-limit errors, return immediately
		return err
	}

	return fmt.Errorf("failed to store embedding after %d attempts due to rate limiting", maxRetries)
}

// calculateContentHash generates a hash of the product content for caching
func (h *ProductHandler) calculateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 chars
}

// updateProductEmbeddingHash updates the embedding hash in the database
func (h *ProductHandler) updateProductEmbeddingHash(productID uuid.UUID, hash string) {
	// Update synchronously for immediate effect
	if err := h.db.Model(&models.Product{}).Where("id = ?", productID).Update("embedding_hash", hash).Error; err != nil {
		log.Printf("Failed to update embedding hash for product %s: %v", productID, err)
	} else {
		log.Printf("✅ Updated embedding hash for product %s to: %s", productID, hash)
	}
}

// CustomerHandler handles customer operations
type CustomerHandler struct {
	customerRepo *repo.CustomerRepository
}

// NewCustomerHandler creates a new customer handler
func NewCustomerHandler(customerRepo *repo.CustomerRepository) *CustomerHandler {
	return &CustomerHandler{customerRepo: customerRepo}
}

// List godoc
// @Summary List customers
// @Description Get list of customers with pagination and search
// @Tags customers
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Param search query string false "Search term"
// @Success 200 {object} models.CustomerListResponse
// @Failure 500 {object} map[string]string
// @Router /customers [get]
// @Security BearerAuth
func (h *CustomerHandler) List(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	search := c.QueryParam("search")

	if limit <= 0 {
		limit = 20
	}

	result, err := h.customerRepo.ListWithSearch(tenantID, limit, offset, search)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// GetByID godoc
// @Summary Get customer by ID
// @Description Get a customer by their ID
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Success 200 {object} models.Customer
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /customers/{id} [get]
// @Security BearerAuth
func (h *CustomerHandler) GetByID(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	customer, err := h.customerRepo.GetByID(tenantID, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Customer not found"})
	}

	return c.JSON(http.StatusOK, customer)
}

// Create godoc
// @Summary Create customer
// @Description Create a new customer
// @Tags customers
// @Accept json
// @Produce json
// @Param customer body models.Customer true "Customer data"
// @Success 201 {object} models.Customer
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /customers [post]
// @Security BearerAuth
func (h *CustomerHandler) Create(c echo.Context) error {
	// Use intermediate struct to handle birth_date as string
	var customerInput struct {
		Phone     string `json:"phone" validate:"required,numeric"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		Document  string `json:"document"`
		BirthDate string `json:"birth_date"` // Accept as string to handle empty strings
		Gender    string `json:"gender"`
		Notes     string `json:"notes"`
		IsActive  bool   `json:"is_active"`
	}

	if err := c.Bind(&customerInput); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Tenant ID not found in context",
		})
	}

	// Convert to Customer model
	customer := models.Customer{
		Phone:    customerInput.Phone,
		Name:     customerInput.Name,
		Email:    customerInput.Email,
		Document: customerInput.Document,
		Gender:   customerInput.Gender,
		Notes:    customerInput.Notes,
		IsActive: customerInput.IsActive,
	}

	// Set tenant ID for the customer
	customer.TenantID = tenantID

	// Handle birth_date: convert empty string to nil
	if customerInput.BirthDate != "" {
		if birthDate, err := time.Parse("2006-01-02", customerInput.BirthDate); err == nil {
			customer.BirthDate = &birthDate
		} else if birthDate, err := time.Parse(time.RFC3339, customerInput.BirthDate); err == nil {
			customer.BirthDate = &birthDate
		}
		// If parsing fails, leave as nil (empty birth_date)
	}

	// Clean phone number to remove formatting
	customer.Phone = cleanPhoneNumber(customer.Phone)

	// Validate the converted data
	if err := c.Validate(&customer); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Validation failed: " + err.Error()})
	}

	if err := h.customerRepo.Create(&customer); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, customer)
}

// Update godoc
// @Summary Update customer
// @Description Update an existing customer
// @Tags customers
// @Accept json
// @Produce json
// @Param id path string true "Customer ID"
// @Param customer body models.Customer true "Customer data"
// @Success 200 {object} models.Customer
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /customers/{id} [put]
// @Security BearerAuth
func (h *CustomerHandler) Update(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	existingCustomer, err := h.customerRepo.GetByID(tenantID, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Customer not found"})
	}

	// Use intermediate struct to handle birth_date as string
	var customerInput struct {
		Phone     string `json:"phone" validate:"required,numeric"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		Document  string `json:"document"`
		BirthDate string `json:"birth_date"` // Accept as string to handle empty strings
		Gender    string `json:"gender"`
		Notes     string `json:"notes"`
		IsActive  bool   `json:"is_active"`
	}

	if err := c.Bind(&customerInput); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request format"})
	}

	// Convert to Customer model
	customer := models.Customer{
		Phone:    customerInput.Phone,
		Name:     customerInput.Name,
		Email:    customerInput.Email,
		Document: customerInput.Document,
		Gender:   customerInput.Gender,
		Notes:    customerInput.Notes,
		IsActive: customerInput.IsActive,
	}

	// Handle birth_date: convert empty string to nil
	if customerInput.BirthDate != "" {
		if birthDate, err := time.Parse("2006-01-02", customerInput.BirthDate); err == nil {
			customer.BirthDate = &birthDate
		} else if birthDate, err := time.Parse(time.RFC3339, customerInput.BirthDate); err == nil {
			customer.BirthDate = &birthDate
		}
		// If parsing fails, leave as nil (empty birth_date)
	}

	// Validate the converted data
	if err := c.Validate(&customer); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Validation failed: " + err.Error()})
	}

	// Clean phone number to remove formatting
	customer.Phone = cleanPhoneNumber(customer.Phone)

	customer.ID = existingCustomer.ID
	customer.TenantID = existingCustomer.TenantID
	customer.CreatedAt = existingCustomer.CreatedAt

	if err := h.customerRepo.Update(&customer); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, customer)
}

// OrderHandler handles order operations
type OrderHandler struct {
	orderRepo    *repo.OrderRepository
	customerRepo *repo.CustomerRepository
	productRepo  *repo.ProductRepository
	db           *gorm.DB
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderRepo *repo.OrderRepository, customerRepo *repo.CustomerRepository, productRepo *repo.ProductRepository, db *gorm.DB) *OrderHandler {
	return &OrderHandler{
		orderRepo:    orderRepo,
		customerRepo: customerRepo,
		productRepo:  productRepo,
		db:           db,
	}
}

// List godoc
// @Summary List orders
// @Description Get list of orders with pagination and customer information
// @Tags orders
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(20)
// @Param offset query int false "Offset" default(0)
// @Param search query string false "Search term"
// @Success 200 {object} models.OrderListResponse
// @Failure 500 {object} map[string]string
// @Router /orders [get]
// @Security BearerAuth
func (h *OrderHandler) List(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	offset, _ := strconv.Atoi(c.QueryParam("offset"))
	search := c.QueryParam("search")
	status := c.QueryParam("status")
	paymentStatus := c.QueryParam("payment_status")
	fulfillmentStatus := c.QueryParam("fulfillment_status")
	paymentMethodID := c.QueryParam("payment_method_id")
	customerID := c.QueryParam("customer_id")
	dateFrom := c.QueryParam("date_from")
	dateTo := c.QueryParam("date_to")

	if limit <= 0 {
		limit = 20
	}

	var result *repo.PaginationResult[repo.OrderWithCustomer]
	var err error

	// Check if any filters are applied
	hasFilters := status != "" || paymentStatus != "" || fulfillmentStatus != "" ||
		paymentMethodID != "" || customerID != "" || dateFrom != "" || dateTo != ""

	if hasFilters || search != "" {
		// Build filter query manually for now
		query := h.orderRepo.GetDB().Table("orders").
			Select(`orders.*, 
				customers.name as customer_name, 
				customers.email as customer_email,
				payment_methods.name as payment_method_name,
				COALESCE((SELECT SUM(quantity) FROM order_items WHERE order_id = orders.id), 0) as items_count`).
			Joins("LEFT JOIN customers ON customers.id = orders.customer_id").
			Joins("LEFT JOIN payment_methods ON payment_methods.id = orders.payment_method_id").
			Where("orders.tenant_id = ?", tenantID)

		// Apply search filter
		if search != "" {
			searchPattern := "%" + search + "%"
			query = query.Where("(orders.order_number ILIKE ? OR customers.name ILIKE ? OR customers.phone ILIKE ? OR customers.email ILIKE ?)",
				searchPattern, searchPattern, searchPattern, searchPattern)
		}

		// Apply status filters
		if status != "" {
			query = query.Where("orders.status = ?", status)
		}
		if paymentStatus != "" {
			query = query.Where("orders.payment_status = ?", paymentStatus)
		}
		if fulfillmentStatus != "" {
			query = query.Where("orders.fulfillment_status = ?", fulfillmentStatus)
		}

		// Apply ID filters
		if paymentMethodID != "" {
			query = query.Where("orders.payment_method_id = ?", paymentMethodID)
		}
		if customerID != "" {
			query = query.Where("orders.customer_id = ?", customerID)
		}

		// Apply date filters
		if dateFrom != "" {
			query = query.Where("DATE(orders.created_at) >= ?", dateFrom)
		}
		if dateTo != "" {
			query = query.Where("DATE(orders.created_at) <= ?", dateTo)
		}

		// Count total
		var total int64
		countQuery := h.orderRepo.GetDB().Model(&models.Order{}).Where("orders.tenant_id = ?", tenantID)

		// Apply same filters to count query
		if search != "" {
			searchPattern := "%" + search + "%"
			countQuery = countQuery.Joins("LEFT JOIN customers ON customers.id = orders.customer_id").
				Where("(orders.order_number ILIKE ? OR customers.name ILIKE ? OR customers.phone ILIKE ? OR customers.email ILIKE ?)",
					searchPattern, searchPattern, searchPattern, searchPattern)
		}
		if status != "" {
			countQuery = countQuery.Where("orders.status = ?", status)
		}
		if paymentStatus != "" {
			countQuery = countQuery.Where("orders.payment_status = ?", paymentStatus)
		}
		if fulfillmentStatus != "" {
			countQuery = countQuery.Where("orders.fulfillment_status = ?", fulfillmentStatus)
		}
		if paymentMethodID != "" {
			countQuery = countQuery.Where("orders.payment_method_id = ?", paymentMethodID)
		}
		if customerID != "" {
			countQuery = countQuery.Where("orders.customer_id = ?", customerID)
		}
		if dateFrom != "" {
			countQuery = countQuery.Where("DATE(orders.created_at) >= ?", dateFrom)
		}
		if dateTo != "" {
			countQuery = countQuery.Where("DATE(orders.created_at) <= ?", dateTo)
		}

		if err := countQuery.Count(&total).Error; err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Execute main query
		var orders []repo.OrderWithCustomer
		err = query.Order("orders.created_at DESC").
			Limit(limit).
			Offset(offset).
			Find(&orders).Error

		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
		}

		// Build pagination result
		page := (offset / limit) + 1
		totalPages := int((total + int64(limit) - 1) / int64(limit))

		result = &repo.PaginationResult[repo.OrderWithCustomer]{
			Data:       orders,
			Total:      total,
			Page:       page,
			PerPage:    limit,
			TotalPages: totalPages,
		}
	} else {
		// Use existing methods for simple cases
		if search != "" {
			result, err = h.orderRepo.ListWithCustomersAndSearch(tenantID, limit, offset, search)
		} else {
			result, err = h.orderRepo.ListWithCustomers(tenantID, limit, offset)
		}
	}

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusOK, result)
}

// GetByID godoc
// @Summary Get order by ID
// @Description Get an order by its ID
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Success 200 {object} models.Order
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /orders/{id} [get]
// @Security BearerAuth
func (h *OrderHandler) GetByID(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	order, err := h.orderRepo.GetByID(tenantID, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Order not found"})
	}

	return c.JSON(http.StatusOK, order)
}

// Create godoc
// @Summary Create order
// @Description Create a new order
// @Tags orders
// @Accept json
// @Produce json
// @Param order body models.Order true "Order data"
// @Success 201 {object} models.Order
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /orders [post]
// @Security BearerAuth
func (h *OrderHandler) Create(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	var order models.Order
	if err := c.Bind(&order); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Set tenant ID for the order
	order.TenantID = tenantID

	// Generate order number if not provided
	if order.OrderNumber == "" {
		order.OrderNumber = generateOrderNumber()
	}

	// Populate historical data for customer
	if order.CustomerID != nil {
		customer, err := h.customerRepo.GetByID(tenantID, *order.CustomerID)
		if err == nil {
			order.CustomerName = &customer.Name
			order.CustomerEmail = &customer.Email
			order.CustomerPhone = &customer.Phone
			order.CustomerDocument = &customer.Document
		}
	}

	// Populate historical data for order items
	for i := range order.Items {
		item := &order.Items[i]
		item.TenantID = tenantID

		if item.ProductID != nil {
			product, err := h.productRepo.GetByID(tenantID, *item.ProductID)
			if err == nil {
				item.ProductName = &product.Name
				item.ProductDescription = &product.Description
				item.ProductSKU = &product.SKU
				item.ProductCategoryID = product.CategoryID

				// Fetch category name via GORM if category exists
				if product.CategoryID != nil {
					var category models.Category
					if err := h.db.Where("id = ? AND tenant_id = ?", *product.CategoryID, tenantID).First(&category).Error; err == nil {
						item.ProductCategoryName = &category.Name
					}
				}

				item.UnitPrice = &item.Price
			}
		}
	}

	// Clean numeric fields - replace empty strings with "0" for numeric fields
	h.cleanOrderNumericFields(&order)

	if err := h.orderRepo.Create(&order); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	return c.JSON(http.StatusCreated, order)
}

// Update godoc
// @Summary Update order
// @Description Update an existing order
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param order body models.Order true "Order data"
// @Success 200 {object} models.Order
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /orders/{id} [put]
// @Security BearerAuth
func (h *OrderHandler) Update(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid ID format"})
	}

	existingOrder, err := h.orderRepo.GetByID(tenantID, id)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Order not found"})
	}

	var updateData models.Order
	if err := c.Bind(&updateData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request"})
	}

	// Start with the existing order to preserve all fields
	order := *existingOrder

	// Only update fields that were actually provided in the request
	// This prevents accidental data loss from partial updates

	// Update status fields only if they were provided and are not empty
	if updateData.Status != "" {
		order.Status = updateData.Status
	}
	if updateData.PaymentStatus != "" {
		order.PaymentStatus = updateData.PaymentStatus
	}
	if updateData.FulfillmentStatus != "" {
		order.FulfillmentStatus = updateData.FulfillmentStatus
	}

	// Update payment method ID - allow setting to nil
	if updateData.PaymentMethodID != existingOrder.PaymentMethodID {
		order.PaymentMethodID = updateData.PaymentMethodID
	}

	// Update monetary fields only if they were provided and are not empty
	if updateData.TotalAmount != "" {
		order.TotalAmount = updateData.TotalAmount
	}
	if updateData.Subtotal != "" {
		order.Subtotal = updateData.Subtotal
	}
	if updateData.TaxAmount != "" {
		order.TaxAmount = updateData.TaxAmount
	}
	if updateData.ShippingAmount != "" {
		order.ShippingAmount = updateData.ShippingAmount
	}
	if updateData.DiscountAmount != "" {
		order.DiscountAmount = updateData.DiscountAmount
	}
	if updateData.Currency != "" {
		order.Currency = updateData.Currency
	}

	// Update customer info only if provided (be very careful with these)
	if updateData.CustomerName != nil && *updateData.CustomerName != "" {
		order.CustomerName = updateData.CustomerName
	}
	if updateData.CustomerEmail != nil && *updateData.CustomerEmail != "" {
		order.CustomerEmail = updateData.CustomerEmail
	}
	if updateData.CustomerPhone != nil && *updateData.CustomerPhone != "" {
		order.CustomerPhone = updateData.CustomerPhone
	}
	if updateData.CustomerDocument != nil && *updateData.CustomerDocument != "" {
		order.CustomerDocument = updateData.CustomerDocument
	}

	// Update addresses only if provided
	if updateData.ShippingStreet != nil && *updateData.ShippingStreet != "" {
		order.ShippingStreet = updateData.ShippingStreet
	}
	if updateData.ShippingNumber != nil && *updateData.ShippingNumber != "" {
		order.ShippingNumber = updateData.ShippingNumber
	}
	if updateData.ShippingNeighborhood != nil && *updateData.ShippingNeighborhood != "" {
		order.ShippingNeighborhood = updateData.ShippingNeighborhood
	}
	if updateData.ShippingCity != nil && *updateData.ShippingCity != "" {
		order.ShippingCity = updateData.ShippingCity
	}
	if updateData.ShippingState != nil && *updateData.ShippingState != "" {
		order.ShippingState = updateData.ShippingState
	}
	if updateData.ShippingZipcode != nil && *updateData.ShippingZipcode != "" {
		order.ShippingZipcode = updateData.ShippingZipcode
	}

	// Update notes only if provided
	if updateData.Notes != "" {
		order.Notes = updateData.Notes
	}

	// Critical: Never allow these fields to be changed through this endpoint
	order.ID = existingOrder.ID
	order.TenantID = existingOrder.TenantID
	order.CreatedAt = existingOrder.CreatedAt
	order.OrderNumber = existingOrder.OrderNumber // Order number should never change

	// Preserve customer reference - critical to prevent losing customer association
	if order.CustomerID == nil || *order.CustomerID == uuid.Nil {
		order.CustomerID = existingOrder.CustomerID
	}

	// Preserve conversation_id - critical for tracking
	if order.ConversationID == nil && existingOrder.ConversationID != nil {
		order.ConversationID = existingOrder.ConversationID
	}

	// Clean numeric fields - ensure proper format for monetary values
	h.cleanOrderNumericFields(&order)

	// Calculate totals based on items if items were provided in the update
	if len(updateData.Items) > 0 {
		order.Items = updateData.Items
		h.calculateOrderTotals(&order)

		// Set tenant_id for all order items
		for i := range order.Items {
			order.Items[i].TenantID = tenantID
		}
	}

	// Store original fulfillment status to check for shipping notification
	originalFulfillmentStatus := existingOrder.FulfillmentStatus

	// Perform the update
	if err := h.orderRepo.Update(&order); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": err.Error()})
	}

	// 📨 Enviar notificação WhatsApp se o status mudou para "shipped"
	log.Printf("🔍 Checking fulfillment status change: original='%s', new='%s'", originalFulfillmentStatus, order.FulfillmentStatus)
	if originalFulfillmentStatus != "shipped" && order.FulfillmentStatus == "shipped" {
		log.Printf("🚚 TRIGGERING SHIPPING NOTIFICATION for order %s", order.OrderNumber)
		go h.sendShippingNotification(tenantID, &order)
	} else {
		log.Printf("⚠️ No shipping notification triggered - original='%s', new='%s'", originalFulfillmentStatus, order.FulfillmentStatus)
	}

	return c.JSON(http.StatusOK, order)
}

// cleanProductFields removes empty string values from numeric fields to prevent SQL errors
func (h *ProductHandler) cleanProductFields(product *models.Product) {
	// Generate SKU if empty
	if product.SKU == "" {
		product.SKU = generateUniqueSKU(h.db, product.TenantID, product.Name)
		log.Printf("Generated SKU '%s' for product '%s'", product.SKU, product.Name)
	}

	// Normalize price - convert formatted price (R$ 49,90) to numeric (49.90)
	product.Price = h.normalizePriceString(product.Price)

	// Normalize sale_price if provided
	if product.SalePrice != "" {
		product.SalePrice = h.normalizePriceString(product.SalePrice)
	}

	// Normalize weight if provided - convert comma to dot for decimal separator
	if product.Weight != "" {
		product.Weight = h.normalizeWeightString(product.Weight)
	}

	// Set empty numeric string fields to "0" or keep them empty based on field requirements
	if product.Price == "" {
		product.Price = "0"
	}
	// SalePrice can be empty for products without sale pricing

	// Ensure required fields have default values
	if product.CategoryID != nil && *product.CategoryID == uuid.Nil {
		// Set CategoryID to nil if it's the zero UUID
		product.CategoryID = nil
	}
}

// normalizePriceString converts formatted price strings to numeric format
// Examples: "R$ 49,90" -> "49.90", "29,50" -> "29.50", "100" -> "100"
func (h *ProductHandler) normalizePriceString(price string) string {
	if price == "" {
		return ""
	}

	// Remove currency symbols and spaces
	price = regexp.MustCompile(`[^\d,.]`).ReplaceAllString(price, "")

	// Replace comma with dot for decimal separator
	price = strings.ReplaceAll(price, ",", ".")

	// Validate if it's a valid number
	priceFloat, err := strconv.ParseFloat(price, 64)
	if err != nil {
		return "0"
	}

	// Format to ensure exactly 2 decimal places
	return fmt.Sprintf("%.2f", priceFloat)
}

// normalizeWeightString converts weight strings with comma to dot decimal separator
// Examples: "1,250" -> "1.250", "0,5" -> "0.500", "2.5" -> "2.500"
func (h *ProductHandler) normalizeWeightString(weight string) string {
	if weight == "" {
		return ""
	}

	// Remove any non-numeric characters except comma and dot
	weight = regexp.MustCompile(`[^\d,.]`).ReplaceAllString(weight, "")

	// Replace comma with dot for decimal separator
	weight = strings.ReplaceAll(weight, ",", ".")

	// Validate if it's a valid number
	weightFloat, err := strconv.ParseFloat(weight, 64)
	if err != nil {
		return weight // Return original if can't parse
	}

	// Format to ensure up to 3 decimal places, removing trailing zeros
	formatted := fmt.Sprintf("%.3f", weightFloat)
	// Remove trailing zeros after decimal point
	formatted = strings.TrimRight(formatted, "0")
	formatted = strings.TrimRight(formatted, ".")

	return formatted
}

// cleanOrderNumericFields cleans numeric fields in order to prevent SQL errors
func (h *OrderHandler) cleanOrderNumericFields(order *models.Order) {
	// Clean monetary fields - replace empty strings with "0"
	if order.TotalAmount == "" {
		order.TotalAmount = "0"
	}
	if order.Subtotal == "" {
		order.Subtotal = "0"
	}
	if order.TaxAmount == "" {
		order.TaxAmount = "0"
	}
	if order.ShippingAmount == "" {
		order.ShippingAmount = "0"
	}
	if order.DiscountAmount == "" {
		order.DiscountAmount = "0"
	}
	if order.Currency == "" {
		order.Currency = "BRL"
	}
}

// calculateOrderTotals calculates order totals based on items
func (h *OrderHandler) calculateOrderTotals(order *models.Order) {
	var subtotal float64 = 0

	// Calculate subtotal from items
	for i := range order.Items {
		// Ensure item has calculated total
		quantity := float64(order.Items[i].Quantity)
		price := h.parsePrice(order.Items[i].Price)
		itemTotal := quantity * price
		order.Items[i].Total = h.formatPrice(itemTotal)

		subtotal += itemTotal
	}

	// Set calculated values
	order.Subtotal = h.formatPrice(subtotal)

	// Calculate total: subtotal + shipping + tax - discount
	shipping := h.parsePrice(order.ShippingAmount)
	tax := h.parsePrice(order.TaxAmount)
	discount := h.parsePrice(order.DiscountAmount)

	total := subtotal + shipping + tax - discount
	if total < 0 {
		total = 0
	}

	order.TotalAmount = h.formatPrice(total)
}

// parsePrice safely parses a price string to float64
func (h *OrderHandler) parsePrice(priceStr string) float64 {
	if priceStr == "" {
		return 0
	}

	// Simple parsing - assume the string is already in correct format (like "49.90")
	var price float64
	if _, err := fmt.Sscanf(priceStr, "%f", &price); err != nil {
		return 0
	}
	return price
}

// formatPrice formats a float64 as a price string
func (h *OrderHandler) formatPrice(price float64) string {
	return fmt.Sprintf("%.2f", price)
}

// AddOrderItemRequest represents request to add item with attributes to order
type AddOrderItemRequest struct {
	ProductID  *uuid.UUID                     `json:"product_id" validate:"required"`
	Quantity   int                            `json:"quantity" validate:"min=1"`
	Price      string                         `json:"price,omitempty"`
	Attributes []AddOrderItemAttributeRequest `json:"attributes,omitempty"`
}

// AddOrderItemAttributeRequest represents an attribute for order item
type AddOrderItemAttributeRequest struct {
	AttributeID   uuid.UUID `json:"attribute_id" validate:"required"`
	OptionID      uuid.UUID `json:"option_id" validate:"required"`
	AttributeName string    `json:"attribute_name" validate:"required"`
	OptionName    string    `json:"option_name" validate:"required"`
	OptionPrice   string    `json:"option_price"`
}

// AddItem godoc
// @Summary Add item to order
// @Description Add a new item to an existing order
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param item body models.OrderItem true "Order item data"
// @Success 200 {object} models.Order
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /orders/{id}/items [post]
// @Security BearerAuth
func (h *OrderHandler) AddItem(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	orderID := c.Param("id")
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid order ID"})
	}

	// Get the order
	order, err := h.orderRepo.GetByID(tenantID, orderUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Order not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get order"})
	}

	// Parse the new item request
	var itemRequest AddOrderItemRequest
	if err := c.Bind(&itemRequest); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid item data"})
	}

	// Validate required fields
	if itemRequest.ProductID == nil || *itemRequest.ProductID == uuid.Nil || itemRequest.Quantity <= 0 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Product ID and quantity are required"})
	}

	// Get product to validate and get price
	product, err := h.productRepo.GetByID(tenantID, *itemRequest.ProductID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Product not found"})
	}

	// Create the order item
	newItem := models.OrderItem{
		BaseTenantModel: models.BaseTenantModel{
			ID:       uuid.New(),
			TenantID: tenantID,
		},
		OrderID:   orderUUID,
		ProductID: itemRequest.ProductID,
		Quantity:  itemRequest.Quantity,
		// Historical product data
		ProductName:        &product.Name,
		ProductDescription: &product.Description,
		ProductSKU:         &product.SKU,
	}

	// Use provided price or default to product price
	if itemRequest.Price != "" {
		newItem.Price = itemRequest.Price
	} else {
		newItem.Price = product.Price
	}

	// Calculate total (base price only, attributes will be added)
	basePrice := h.parsePrice(newItem.Price)

	// Calculate additional price from attributes
	var attributesPrice float64
	for _, attr := range itemRequest.Attributes {
		if attr.OptionPrice != "" {
			attributesPrice += h.parsePrice(attr.OptionPrice)
		}
	}

	// Total price per unit (base + attributes) * quantity
	totalPrice := (basePrice + attributesPrice) * float64(newItem.Quantity)
	newItem.Total = h.formatPrice(totalPrice)

	// Set unit price including attributes
	unitPriceWithAttributes := basePrice + attributesPrice
	unitPriceStr := h.formatPrice(unitPriceWithAttributes)
	newItem.UnitPrice = &unitPriceStr

	// Check if item with same product already exists
	existingItemIndex := -1
	for i, item := range order.Items {
		if item.ProductID != nil && newItem.ProductID != nil && *item.ProductID == *newItem.ProductID {
			existingItemIndex = i
			break
		}
	}

	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if existingItemIndex >= 0 {
		// Update existing item quantity
		existingItem := &order.Items[existingItemIndex]
		existingItem.Quantity += newItem.Quantity
		existingItemTotal := h.parsePrice(existingItem.Price) * float64(existingItem.Quantity)
		existingItem.Total = h.formatPrice(existingItemTotal)

		// Update in database
		if err := tx.Save(existingItem).Error; err != nil {
			tx.Rollback()
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update item"})
		}
	} else {
		// Add new item
		if err := tx.Create(&newItem).Error; err != nil {
			tx.Rollback()
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to add item"})
		}

		// Create item attributes if provided
		for _, attrReq := range itemRequest.Attributes {
			itemAttribute := models.OrderItemAttribute{
				BaseTenantModel: models.BaseTenantModel{
					ID:       uuid.New(),
					TenantID: tenantID,
				},
				OrderItemID:   newItem.ID,
				AttributeID:   attrReq.AttributeID,
				OptionID:      attrReq.OptionID,
				AttributeName: attrReq.AttributeName,
				OptionName:    attrReq.OptionName,
				OptionPrice:   attrReq.OptionPrice,
			}

			if err := tx.Create(&itemAttribute).Error; err != nil {
				tx.Rollback()
				return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to add item attributes"})
			}
		}

		order.Items = append(order.Items, newItem)
	}

	// Recalculate order totals
	h.recalculateOrderTotals(order)

	// Update order
	if err := tx.Save(order).Error; err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update order"})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to commit transaction"})
	}

	// Reload order with items
	updatedOrder, err := h.orderRepo.GetByID(tenantID, orderUUID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to reload order"})
	}

	return c.JSON(http.StatusOK, updatedOrder)
}

// UpdateItem godoc
// @Summary Update order item
// @Description Update quantity or price of an order item
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param item_id path string true "Item ID"
// @Param item body models.OrderItem true "Updated item data"
// @Success 200 {object} models.Order
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /orders/{id}/items/{item_id} [put]
// @Security BearerAuth
func (h *OrderHandler) UpdateItem(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	orderID := c.Param("id")
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid order ID"})
	}

	itemID := c.Param("item_id")
	itemUUID, err := uuid.Parse(itemID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid item ID"})
	}

	// Get the order
	order, err := h.orderRepo.GetByID(tenantID, orderUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Order not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get order"})
	}

	// Find the item
	var itemToUpdate *models.OrderItem
	for i := range order.Items {
		if order.Items[i].ID == itemUUID {
			itemToUpdate = &order.Items[i]
			break
		}
	}

	if itemToUpdate == nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Item not found"})
	}

	// Parse update data
	var updateData models.OrderItem
	if err := c.Bind(&updateData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid item data"})
	}

	// Update quantity if provided and valid
	if updateData.Quantity > 0 {
		itemToUpdate.Quantity = updateData.Quantity
	}

	// Update price if provided
	if updateData.Price != "" {
		itemToUpdate.Price = updateData.Price
	}

	// Recalculate item total
	price := h.parsePrice(itemToUpdate.Price)
	total := price * float64(itemToUpdate.Quantity)
	itemToUpdate.Total = h.formatPrice(total)

	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Update item in database
	if err := tx.Save(itemToUpdate).Error; err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update item"})
	}

	// Recalculate order totals
	h.recalculateOrderTotals(order)

	// Update order
	if err := tx.Save(order).Error; err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update order"})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to commit transaction"})
	}

	// Reload order with items
	updatedOrder, err := h.orderRepo.GetByID(tenantID, orderUUID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to reload order"})
	}

	return c.JSON(http.StatusOK, updatedOrder)
}

// RemoveItem godoc
// @Summary Remove item from order
// @Description Remove an item from an existing order
// @Tags orders
// @Accept json
// @Produce json
// @Param id path string true "Order ID"
// @Param item_id path string true "Item ID"
// @Success 200 {object} models.Order
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /orders/{id}/items/{item_id} [delete]
// @Security BearerAuth
func (h *OrderHandler) RemoveItem(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	orderID := c.Param("id")
	orderUUID, err := uuid.Parse(orderID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid order ID"})
	}

	itemID := c.Param("item_id")
	itemUUID, err := uuid.Parse(itemID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid item ID"})
	}

	// Get the order
	order, err := h.orderRepo.GetByID(tenantID, orderUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Order not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get order"})
	}

	// Find the item
	itemIndex := -1
	for i := range order.Items {
		if order.Items[i].ID == itemUUID {
			itemIndex = i
			break
		}
	}

	if itemIndex == -1 {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Item not found"})
	}

	// Start transaction
	tx := h.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Delete item from database
	if err := tx.Delete(&order.Items[itemIndex]).Error; err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete item"})
	}

	// Remove item from slice
	order.Items = append(order.Items[:itemIndex], order.Items[itemIndex+1:]...)

	// Recalculate order totals
	h.recalculateOrderTotals(order)

	// Update order
	if err := tx.Save(order).Error; err != nil {
		tx.Rollback()
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to update order"})
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to commit transaction"})
	}

	// Reload order with items
	updatedOrder, err := h.orderRepo.GetByID(tenantID, orderUUID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to reload order"})
	}

	return c.JSON(http.StatusOK, updatedOrder)
}

// recalculateOrderTotals recalculates order subtotal and total
func (h *OrderHandler) recalculateOrderTotals(order *models.Order) {
	subtotal := 0.0
	for _, item := range order.Items {
		subtotal += h.parsePrice(item.Total)
	}

	order.Subtotal = h.formatPrice(subtotal)

	// Calculate total (subtotal + shipping + tax - discount)
	shipping := h.parsePrice(order.ShippingAmount)
	tax := h.parsePrice(order.TaxAmount)
	discount := h.parsePrice(order.DiscountAmount)

	total := subtotal + shipping + tax - discount
	if total < 0 {
		total = 0
	}

	order.TotalAmount = h.formatPrice(total)
}

// ImportProductsFromImage godoc
// @Summary Import products from image using AI
// @Description Analyze an image (menu, catalog) using AI to extract product information and create products automatically
// @Tags products
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "Image file (JPG, PNG, WebP, PDF)"
// @Success 200 {object} models.ProductImageImportResult
// @Failure 400 {object} map[string]string
// @Failure 402 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/import-image [post]
// @Security BearerAuth
func (h *ProductHandler) ImportProductsFromImage(c echo.Context) error {
	// Get tenant and user ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	userID, ok := c.Get("user_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid user"})
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No file provided"})
	}

	// Validate file type
	validTypes := map[string]bool{
		"image/jpeg":      true,
		"image/jpg":       true,
		"image/png":       true,
		"image/webp":      true,
		"application/pdf": true,
	}

	if !validTypes[file.Header.Get("Content-Type")] {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid file type. Supported: JPG, PNG, WebP, PDF"})
	}

	// Check file size (max 10MB)
	maxSize := int64(10 * 1024 * 1024) // 10MB
	if file.Size > maxSize {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "File too large. Maximum size is 10MB"})
	}

	// Check if OpenAI key is configured
	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "AI service not configured"})
	}

	// Initialize AI services
	imageAnalyzer := ai.NewProductImageAnalysisService(openaiKey)
	aiCreditRepo := repo.NewAICreditRepository(h.db)

	// Check credit cost and availability
	creditCost := imageAnalyzer.GetImageAnalysisCreditCost()
	credits, err := aiCreditRepo.GetByTenantID(tenantID)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to check credits"})
	}

	if credits.RemainingCredits < creditCost {
		return c.JSON(http.StatusPaymentRequired, map[string]interface{}{
			"error":     "Insufficient AI credits",
			"required":  creditCost,
			"available": credits.RemainingCredits,
		})
	}

	// Use credits first
	err = aiCreditRepo.UseCredits(tenantID, &userID, creditCost, "Importação de produtos via imagem", "product_import", nil)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to use credits"})
	}

	// Save file temporarily for analysis
	tempDir := os.TempDir()
	tempFileName := fmt.Sprintf("product_import_%s_%s", tenantID.String(), uuid.New().String())
	tempFilePath := fmt.Sprintf("%s/%s", tempDir, tempFileName)

	// Open uploaded file
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read uploaded file"})
	}
	defer src.Close()

	// Create temporary file
	dst, err := os.Create(tempFilePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to create temporary file"})
	}
	defer dst.Close()

	// Copy file content
	_, err = io.Copy(dst, src)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save temporary file"})
	}

	// Clean up the temporary file after processing
	defer func() {
		if err := os.Remove(tempFilePath); err != nil {
			log.Printf("Warning: Failed to remove temporary file %s: %v", tempFilePath, err)
		}
	}()

	// For AI analysis, we need to convert the file to base64 or use a different approach
	// Since GPT-4 Vision needs a URL, we'll read the file and encode it as base64 data URL
	fileBytes, err := os.ReadFile(tempFilePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to read temporary file"})
	}

	// Create data URL for the image
	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// For PDF files, we'll need to handle differently - for now let's focus on images
	if mimeType == "application/pdf" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "PDF analysis not yet supported. Please use image files (JPG, PNG, WebP)"})
	}

	base64Data := base64.StdEncoding.EncodeToString(fileBytes)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

	// Analyze image with AI
	ctx := c.Request().Context()
	analysis, err := imageAnalyzer.AnalyzeProductImage(ctx, dataURL)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to analyze image: %v", err),
		})
	}

	// Prepare result
	result := models.ProductImageImportResult{
		Success:      true,
		CreatedCount: 0,
		CreditsUsed:  creditCost,
		Products:     make([]models.ProductImageImportProduct, 0),
		Errors:       make([]string, 0),
	}

	// Create products from analysis
	for _, detectedProduct := range analysis.Products {
		// Validate product data
		if detectedProduct.Name == "" {
			result.Errors = append(result.Errors, "Produto sem nome detectado")
			continue
		}

		// Create product
		productData := models.Product{
			BaseTenantModel: models.BaseTenantModel{
				ID:       uuid.New(),
				TenantID: tenantID,
			},
			Name:              detectedProduct.Name,
			Description:       detectedProduct.Description,
			SKU:               generateUniqueSKU(h.db, tenantID, detectedProduct.Name),
			StockQuantity:     10000, // Fixed stock as requested
			LowStockThreshold: 100,
		}

		// Set price if detected
		if detectedProduct.Price != "" {
			// Clean price string (remove currency symbols, etc.)
			priceStr := strings.ReplaceAll(detectedProduct.Price, "R$", "")
			priceStr = strings.ReplaceAll(priceStr, ",", ".")
			priceStr = strings.TrimSpace(priceStr)
			productData.Price = priceStr
		}

		// Set tags
		if len(detectedProduct.Tags) > 0 {
			tagsStr := strings.Join(detectedProduct.Tags, ",")
			productData.Tags = tagsStr
		}

		// Save product to database with retry logic for SKU conflicts
		var createErr error
		maxRetries := 3
		for attempt := 0; attempt < maxRetries; attempt++ {
			createErr = h.db.Create(&productData).Error
			if createErr == nil {
				break // Success
			}

			// Check if it's a SKU duplicate error
			if strings.Contains(createErr.Error(), "duplicate key") && strings.Contains(createErr.Error(), "sku") {
				// Generate a new SKU and try again
				productData.SKU = generateUniqueSKU(h.db, tenantID, detectedProduct.Name)
				log.Printf("SKU conflict detected for product '%s', retrying with new SKU: %s", detectedProduct.Name, productData.SKU)
				continue
			}

			// For other errors, don't retry
			break
		}

		if createErr != nil {
			errorMsg := fmt.Sprintf("Erro ao criar produto '%s': %v", detectedProduct.Name, createErr)
			result.Errors = append(result.Errors, errorMsg)

			// Add to result as error
			result.Products = append(result.Products, models.ProductImageImportProduct{
				Name:        detectedProduct.Name,
				Description: detectedProduct.Description,
				Price:       detectedProduct.Price,
				Tags:        strings.Join(detectedProduct.Tags, ","),
				Status:      "error",
			})
			continue
		}

		// Success
		result.CreatedCount++
		result.Products = append(result.Products, models.ProductImageImportProduct{
			Name:        detectedProduct.Name,
			Description: detectedProduct.Description,
			Price:       detectedProduct.Price,
			Tags:        strings.Join(detectedProduct.Tags, ","),
			Status:      "success",
		})
	}

	// Update success status
	if len(result.Errors) > 0 && result.CreatedCount == 0 {
		result.Success = false
	}

	// Process embeddings in batch for created products
	if h.embeddingService != nil && result.CreatedCount > 0 {
		go func() {
			if err := h.processBatchEmbeddingsForImageImport(tenantID, result.Products); err != nil {
				log.Printf("❌ Failed to process batch embeddings for image import: %v", err)
			}
		}()
	}

	return c.JSON(http.StatusOK, result)
}

// sendShippingNotification envia notificação WhatsApp quando pedido é marcado como enviado
func (h *OrderHandler) sendShippingNotification(tenantID uuid.UUID, order *models.Order) {
	notificationService := zapplus.NewNotificationService(h.db)

	err := notificationService.SendShippingNotification(tenantID, order)
	if err != nil {
		log.Printf("❌ Error sending shipping notification for order %s: %v", order.OrderNumber, err)
	} else {
		log.Printf("✅ Shipping notification sent successfully for order %s", order.OrderNumber)
	}
}

// findOrCreateCategory finds an existing category by name or creates a new one
func (h *ProductHandler) findOrCreateCategory(tenantID uuid.UUID, categoryName string) (*uuid.UUID, error) {
	if categoryName == "" {
		return nil, nil
	}

	// Try to find existing category
	existing, err := h.categoryRepo.FindExistingCategory(tenantID, categoryName)
	if err == nil && existing != nil {
		return &existing.ID, nil
	}

	// If not found, create new category
	if err == gorm.ErrRecordNotFound {
		category := &models.Category{
			BaseTenantModel: models.BaseTenantModel{
				TenantID: tenantID,
			},
			Name:        categoryName,
			Description: "",
			IsActive:    true,
			SortOrder:   0,
		}

		if createErr := h.categoryRepo.Create(category); createErr != nil {
			return nil, createErr
		}

		return &category.ID, nil
	}

	// Return other errors
	return nil, err
}

// UploadProductImage godoc
// @Summary Upload product image
// @Description Upload an image for a product
// @Tags products
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "Product ID"
// @Param image formData file true "Product image file"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/{id}/upload-image [post]
// @Security BearerAuth
func (h *ProductHandler) UploadProductImage(c echo.Context) error {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
	}

	tenantID := c.Get("tenant_id").(uuid.UUID)

	// Check if product exists
	_, err = h.productRepo.GetByID(tenantID, productID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Product not found"})
	}

	// Get file from request
	file, err := c.FormFile("image")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "No image file provided"})
	}

	// Validate file type
	src, err := file.Open()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to open uploaded file"})
	}
	defer src.Close()

	// Check if it's an image
	buffer := make([]byte, 512)
	_, err = src.Read(buffer)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Failed to read file"})
	}
	src.Seek(0, 0) // Reset file pointer

	contentType := http.DetectContentType(buffer)
	if !strings.HasPrefix(contentType, "image/") {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "File is not an image"})
	}

	// Check if storage service is available
	if h.storageService == nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Storage service not available"})
	}

	// Upload to S3 in products folder
	publicURL, err := h.storageService.UploadMultipartFile(file, tenantID.String(), "products")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Failed to upload image: %v", err)})
	}

	// Create ProductMedia record
	productMedia := &models.ProductMedia{
		ProductID: productID,
		Type:      "image",
		URL:       publicURL,
		SortOrder: 0, // You might want to increment this based on existing images
	}
	productMedia.TenantID = tenantID

	// Save to database
	if err := h.db.Create(productMedia).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to save image record"})
	}

	return c.JSON(http.StatusCreated, map[string]string{
		"url":     publicURL,
		"id":      productMedia.ID.String(),
		"type":    productMedia.Type,
		"message": "Image uploaded successfully",
	})
}

// GetProductImages godoc
// @Summary Get product images
// @Description Get all images for a product
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {array} models.ProductMedia
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/{id}/images [get]
// @Security BearerAuth
func (h *ProductHandler) GetProductImages(c echo.Context) error {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
	}

	tenantID := c.Get("tenant_id").(uuid.UUID)

	// Check if product exists
	_, err = h.productRepo.GetByID(tenantID, productID)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Product not found"})
	}

	// Get product images
	var images []models.ProductMedia
	err = h.db.Where("product_id = ? AND tenant_id = ? AND type = ?", productID, tenantID, "image").
		Order("sort_order ASC, created_at ASC").
		Find(&images).Error
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get product images"})
	}

	return c.JSON(http.StatusOK, images)
}

// DeleteProductImage godoc
// @Summary Delete product image
// @Description Delete a product image
// @Tags products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param image_id path string true "Image ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /products/{id}/images/{image_id} [delete]
// @Security BearerAuth
func (h *ProductHandler) DeleteProductImage(c echo.Context) error {
	productID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid product ID"})
	}

	imageID, err := uuid.Parse(c.Param("image_id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid image ID"})
	}

	tenantID := c.Get("tenant_id").(uuid.UUID)

	// Get the product media record
	var productMedia models.ProductMedia
	err = h.db.Where("id = ? AND product_id = ? AND tenant_id = ?", imageID, productID, tenantID).First(&productMedia).Error
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]string{"error": "Image not found"})
	}

	// Delete from S3 if storage service is available
	if h.storageService != nil && productMedia.S3Key != "" {
		// Extract S3 key from URL if needed
		s3Key := productMedia.S3Key
		if s3Key == "" {
			// Extract from URL if S3Key is not set
			// This assumes the URL format: https://bucket/key
			parts := strings.Split(productMedia.URL, "/")
			if len(parts) > 3 {
				s3Key = strings.Join(parts[3:], "/")
			}
		}
		if s3Key != "" {
			if err := h.storageService.DeleteFile(s3Key); err != nil {
				// Log error but don't fail the request
				log.Printf("Failed to delete file from S3: %v", err)
			}
		}
	}

	// Delete from database
	if err := h.db.Delete(&productMedia).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to delete image record"})
	}

	return c.JSON(http.StatusOK, map[string]string{"message": "Image deleted successfully"})
}
