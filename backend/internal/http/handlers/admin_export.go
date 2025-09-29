package handlers

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"time"

	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

type AdminExportHandler struct {
	db *gorm.DB
}

func NewAdminExportHandler(db *gorm.DB) *AdminExportHandler {
	return &AdminExportHandler{
		db: db,
	}
}

// ExportTenantProducts godoc
// @Summary Export tenant products to CSV
// @Description Export all products from a specific tenant to CSV format for backup purposes
// @Tags admin-export
// @Produce text/csv
// @Param tenant_id path string true "Tenant ID"
// @Success 200 {string} string "CSV file content"
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /admin/export/tenants/{tenant_id}/products [get]
func (h *AdminExportHandler) ExportTenantProducts(c echo.Context) error {
	tenantIDStr := c.Param("tenant_id")
	if tenantIDStr == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "tenant_id is required"})
	}

	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid tenant_id format"})
	}

	// Verificar se o tenant existe
	var tenant models.Tenant
	if err := h.db.Where("id = ?", tenantID).First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "tenant not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to check tenant"})
	}

	// Buscar todos os produtos do tenant com suas categorias
	var products []models.Product
	var categories []models.Category

	// Primeiro, buscar todas as categorias do tenant
	if err := h.db.Where("tenant_id = ?", tenantID).Find(&categories).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch categories"})
	}

	// Criar um mapa para lookup rápido de categorias
	categoryMap := make(map[uuid.UUID]string)
	for _, category := range categories {
		categoryMap[category.ID] = category.Name
	}

	// Buscar todos os produtos do tenant
	if err := h.db.Where("tenant_id = ?", tenantID).Find(&products).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch products"})
	}

	// Configurar response para download CSV
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("produtos_tenant_%s_%s.csv", tenant.Name, timestamp)

	c.Response().Header().Set(echo.HeaderContentType, "text/csv")
	c.Response().Header().Set(echo.HeaderContentDisposition, fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Response().WriteHeader(http.StatusOK)

	// Criar writer CSV
	writer := csv.NewWriter(c.Response().Writer)
	defer writer.Flush()

	// Escrever cabeçalho (mesmo formato do arquivo de importação)
	header := []string{
		"name",
		"description",
		"price",
		"sale_price",
		"sku",
		"barcode",
		"weight",
		"dimensions",
		"brand",
		"tags",
		"stock_quantity",
		"low_stock_threshold",
		"category_name",
	}
	if err := writer.Write(header); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to write CSV header"})
	}

	// Escrever dados dos produtos
	for _, product := range products {
		// Buscar nome da categoria
		categoryName := ""
		if product.CategoryID != nil {
			if name, exists := categoryMap[*product.CategoryID]; exists {
				categoryName = name
			}
		}

		record := []string{
			product.Name,
			product.Description,
			product.Price,
			product.SalePrice,
			product.SKU,
			product.Barcode,
			product.Weight,
			product.Dimensions,
			product.Brand,
			product.Tags,
			fmt.Sprintf("%d", product.StockQuantity),
			fmt.Sprintf("%d", product.LowStockThreshold),
			categoryName,
		}

		if err := writer.Write(record); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to write CSV record"})
		}
	}

	return nil
}

// GetTenantsList godoc
// @Summary Get all tenants for export selection
// @Description Get a list of all tenants to choose from for product export
// @Tags admin-export
// @Produce json
// @Success 200 {array} map[string]interface{}
// @Failure 500 {object} map[string]string
// @Router /admin/export/tenants [get]
func (h *AdminExportHandler) GetTenantsList(c echo.Context) error {
	var tenants []models.Tenant

	if err := h.db.Select("id, name, created_at").Find(&tenants).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to fetch tenants"})
	}

	// Transformar em formato mais limpo para o frontend
	result := make([]map[string]interface{}, len(tenants))
	for i, tenant := range tenants {
		// Contar produtos do tenant
		var productCount int64
		h.db.Model(&models.Product{}).Where("tenant_id = ?", tenant.ID).Count(&productCount)

		result[i] = map[string]interface{}{
			"id":            tenant.ID,
			"name":          tenant.Name,
			"created_at":    tenant.CreatedAt,
			"product_count": productCount,
		}
	}

	return c.JSON(http.StatusOK, result)
}
