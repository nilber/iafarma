package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"iafarma/internal/repo"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// AnalyticsHandler handles analytics and reports operations
type AnalyticsHandler struct {
	db           *gorm.DB
	orderRepo    *repo.OrderRepository
	productRepo  *repo.ProductRepository
	customerRepo *repo.CustomerRepository
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(db *gorm.DB, orderRepo *repo.OrderRepository, productRepo *repo.ProductRepository, customerRepo *repo.CustomerRepository) *AnalyticsHandler {
	return &AnalyticsHandler{
		db:           db,
		orderRepo:    orderRepo,
		productRepo:  productRepo,
		customerRepo: customerRepo,
	}
}

// SalesAnalyticsResponse represents sales analytics data
type SalesAnalyticsResponse struct {
	TotalRevenue   float64          `json:"total_revenue"`
	TotalOrders    int              `json:"total_orders"`
	NewCustomers   int              `json:"new_customers"`
	AverageTicket  float64          `json:"average_ticket"`
	MonthlyData    []ReportDataItem `json:"monthly_data"`
	TopProducts    []TopProductItem `json:"top_products"`
	ConversionRate float64          `json:"conversion_rate"`
	GrowthRate     float64          `json:"growth_rate"`
}

type ReportDataItem struct {
	Period       string  `json:"period"`
	Revenue      float64 `json:"revenue"`
	Orders       int     `json:"orders"`
	Customers    int     `json:"customers"`
	ProductsSold int     `json:"products_sold"`
}

// ReportsResponse includes both data and comparison metadata
type ReportsResponse struct {
	Data       interface{}        `json:"data"`
	Comparison ComparisonMetadata `json:"comparison"`
}

type TopProductItem struct {
	ProductID    string  `json:"product_id"`
	ProductName  string  `json:"product_name"`
	SalesCount   int     `json:"sales_count"`
	TotalRevenue float64 `json:"total_revenue"`
}

// GetSalesAnalytics godoc
// @Summary Get sales analytics
// @Description Get comprehensive sales analytics and metrics
// @Tags analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Param period query string false "Period (daily, weekly, monthly)" default(monthly)
// @Success 200 {object} SalesAnalyticsResponse
// @Failure 500 {object} map[string]string
// @Router /analytics/sales [get]
// @Security BearerAuth
func (h *AnalyticsHandler) GetSalesAnalytics(c echo.Context) error {
	// Get tenant ID from context
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	// Parse date parameters
	startDateStr := c.QueryParam("start_date")
	endDateStr := c.QueryParam("end_date")
	period := c.QueryParam("period")
	if period == "" {
		period = "monthly"
	}

	// Default to last 6 months if no dates provided
	endDate := time.Now()
	startDate := endDate.AddDate(0, -6, 0)

	if startDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = parsed
		}
	}
	if endDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = parsed
		}
	}

	// Get basic metrics with tenant filtering
	var totalRevenue float64
	var totalOrders int64

	// Calculate total revenue and orders
	h.db.Table("orders").
		Where("tenant_id = ?", tenantID).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("status NOT IN ?", []string{"cancelled", "refunded"}).
		Select("COALESCE(SUM(CAST(total_amount AS DECIMAL)), 0)").
		Row().Scan(&totalRevenue)

	h.db.Table("orders").
		Where("tenant_id = ?", tenantID).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Where("status NOT IN ?", []string{"cancelled", "refunded"}).
		Count(&totalOrders)

	// Calculate new customers in period
	var newCustomers int64
	h.db.Table("customers").
		Where("tenant_id = ?", tenantID).
		Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Count(&newCustomers)

	// Calculate average ticket
	averageTicket := float64(0)
	if totalOrders > 0 {
		averageTicket = totalRevenue / float64(totalOrders)
	}

	// Get monthly data
	monthlyData := []ReportDataItem{}
	for i := 0; i < 6; i++ {
		monthStart := endDate.AddDate(0, -i-1, 0)
		monthEnd := endDate.AddDate(0, -i, 0)

		var monthRevenue float64
		var monthOrders int64
		var monthCustomers int64

		h.db.Table("orders").
			Where("tenant_id = ?", tenantID).
			Where("created_at BETWEEN ? AND ?", monthStart, monthEnd).
			Where("status NOT IN ?", []string{"cancelled", "refunded"}).
			Select("COALESCE(SUM(CAST(total_amount AS DECIMAL)), 0)").
			Row().Scan(&monthRevenue)

		h.db.Table("orders").
			Where("tenant_id = ?", tenantID).
			Where("created_at BETWEEN ? AND ?", monthStart, monthEnd).
			Where("status NOT IN ?", []string{"cancelled", "refunded"}).
			Count(&monthOrders)

		h.db.Table("customers").
			Where("tenant_id = ?", tenantID).
			Where("created_at BETWEEN ? AND ?", monthStart, monthEnd).
			Count(&monthCustomers)

		// Count products sold in this month
		var productsSold int64
		h.db.Table("order_items").
			Joins("JOIN orders ON orders.id = order_items.order_id").
			Where("orders.tenant_id = ?", tenantID).
			Where("orders.created_at BETWEEN ? AND ?", monthStart, monthEnd).
			Where("orders.status NOT IN ?", []string{"cancelled", "refunded"}).
			Select("COALESCE(SUM(quantity), 0)").
			Row().Scan(&productsSold)

		monthlyData = append([]ReportDataItem{{
			Period:       monthStart.Format("January"),
			Revenue:      monthRevenue,
			Orders:       int(monthOrders),
			Customers:    int(monthCustomers),
			ProductsSold: int(productsSold),
		}}, monthlyData...)
	}

	// Get real top products from database
	type TopProductQuery struct {
		ProductID    string  `gorm:"column:product_id"`
		ProductName  string  `gorm:"column:product_name"`
		SalesCount   int     `gorm:"column:sales_count"`
		TotalRevenue float64 `gorm:"column:total_revenue"`
	}

	var topProductsQuery []TopProductQuery
	h.db.Table("order_items").
		Select(`
			products.id as product_id,
			products.name as product_name,
			COALESCE(SUM(order_items.quantity), 0) as sales_count,
			COALESCE(SUM(CAST(order_items.total AS DECIMAL)), 0) as total_revenue
		`).
		Joins("JOIN products ON products.id = order_items.product_id").
		Joins("JOIN orders ON orders.id = order_items.order_id").
		Where("orders.tenant_id = ?", tenantID).
		Where("orders.created_at BETWEEN ? AND ?", startDate, endDate).
		Where("orders.status NOT IN ?", []string{"cancelled", "refunded"}).
		Group("products.id, products.name").
		Order("total_revenue DESC").
		Limit(5).
		Find(&topProductsQuery)

	// Convert to response format
	topProducts := make([]TopProductItem, len(topProductsQuery))
	for i, item := range topProductsQuery {
		topProducts[i] = TopProductItem{
			ProductID:    item.ProductID,
			ProductName:  item.ProductName,
			SalesCount:   item.SalesCount,
			TotalRevenue: item.TotalRevenue,
		}
	}

	// Calculate growth rate (compare with previous period)
	previousStartDate := startDate.AddDate(0, -6, 0)
	var previousRevenue float64
	h.db.Table("orders").
		Where("tenant_id = ?", tenantID).
		Where("created_at BETWEEN ? AND ?", previousStartDate, startDate).
		Where("status NOT IN ?", []string{"cancelled", "refunded"}).
		Select("COALESCE(SUM(CAST(total_amount AS DECIMAL)), 0)").
		Row().Scan(&previousRevenue)

	growthRate := float64(0)
	if previousRevenue > 0 {
		growthRate = ((totalRevenue - previousRevenue) / previousRevenue) * 100
	}

	// Calculate conversion rate (orders / customers)
	var totalCustomers int64
	h.db.Table("customers").
		Where("tenant_id = ?", tenantID).
		Count(&totalCustomers)

	conversionRate := float64(0)
	if totalCustomers > 0 {
		conversionRate = (float64(totalOrders) / float64(totalCustomers)) * 100
	}

	response := SalesAnalyticsResponse{
		TotalRevenue:   totalRevenue,
		TotalOrders:    int(totalOrders),
		NewCustomers:   int(newCustomers),
		AverageTicket:  averageTicket,
		MonthlyData:    monthlyData,
		TopProducts:    topProducts,
		ConversionRate: conversionRate,
		GrowthRate:     growthRate,
	}

	return c.JSON(http.StatusOK, response)
}

// GetReportsData godoc
// @Summary Get reports data
// @Description Get specific report data by type
// @Tags reports
// @Accept json
// @Produce json
// @Param type query string true "Report type (revenue, orders, customers, products)"
// @Param period query string false "Period (daily, weekly, monthly)" default(monthly)
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Success 200 {array} ReportDataItem
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /reports [get]
// @Security BearerAuth
func (h *AnalyticsHandler) GetReportsData(c echo.Context) error {

	// Get tenant ID from context
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	reportType := c.QueryParam("type")
	if reportType == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "type parameter is required"})
	}

	period := c.QueryParam("period")
	if period == "" {
		period = "monthly"
	}

	// Parse date parameters
	startDateStr := c.QueryParam("start_date")
	endDateStr := c.QueryParam("end_date")

	// Default period
	endDate := time.Now()
	startDate := time.Time{}

	if period == "monthly" {
		// Para monthly, sempre últimos 6 meses
		startDate = endDate.AddDate(0, -6, 0)
	} else {
		// Para daily/weekly, usar datas fornecidas ou padrão de 30 dias
		startDate = endDate.AddDate(0, 0, -30)

		if startDateStr != "" {
			if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
				startDate = parsed
			}
		}
		if endDateStr != "" {
			if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
				endDate = parsed
			}
		}
	}

	switch period {
	case "monthly":
		var monthlyData []ReportDataItem

		// Buscar dados reais dos últimos 6 meses
		for i := 0; i < 6; i++ {
			monthStart := time.Now().AddDate(0, -i, 0)
			monthStart = time.Date(monthStart.Year(), monthStart.Month(), 1, 0, 0, 0, 0, monthStart.Location())
			monthEnd := monthStart.AddDate(0, 1, 0).Add(-time.Nanosecond)

			type MonthResult struct {
				Revenue      float64 `gorm:"column:revenue"`
				Orders       int64   `gorm:"column:orders"`
				Customers    int64   `gorm:"column:customers"`
				ProductsSold int64   `gorm:"column:products_sold"`
			}

			var result MonthResult

			// Query usando JOINS para buscar todos os dados em uma consulta
			err := h.db.Table("orders o").
				Select(`
					COALESCE(SUM(CAST(o.total_amount AS DECIMAL)), 0) as revenue,
					COUNT(DISTINCT o.id) as orders,
					COUNT(DISTINCT o.customer_id) as customers,
					COALESCE(SUM(oi.quantity), 0) as products_sold
				`).
				Joins("LEFT JOIN order_items oi ON o.id = oi.order_id").
				Where("o.tenant_id = ?", tenantID).
				Where("o.created_at BETWEEN ? AND ?", monthStart, monthEnd).
				Where("o.status NOT IN ?", []string{"cancelled", "refunded"}).
				Scan(&result).Error

			if err != nil {
				// Fallback: Try query without order_items join to see if table exists
				var fallbackResult MonthResult
				fallbackErr := h.db.Table("orders o").
					Select(`
						COALESCE(SUM(CAST(o.total_amount AS DECIMAL)), 0) as revenue,
						COUNT(DISTINCT o.id) as orders,
						COUNT(DISTINCT o.customer_id) as customers,
						0 as products_sold
					`).
					Where("o.tenant_id = ?", tenantID).
					Where("o.created_at BETWEEN ? AND ?", monthStart, monthEnd).
					Where("o.status NOT IN ?", []string{"cancelled", "refunded"}).
					Scan(&fallbackResult).Error

				if fallbackErr != nil {
					continue
				} else {
					result = fallbackResult
				}
			}

			// Se products_sold estiver zero, tentar consulta alternativa
			if result.ProductsSold == 0 && result.Orders > 0 {
				var itemCount int64
				itemErr := h.db.Table("order_items").
					Where("tenant_id = ?", tenantID).
					Where("created_at BETWEEN ? AND ?", monthStart, monthEnd).
					Count(&itemCount).Error

				if itemErr == nil {
					result.ProductsSold = itemCount
				} else {
					// Try with direct SUM of quantities
					var quantitySum int64
					quantityErr := h.db.Table("order_items oi").
						Joins("LEFT JOIN orders o ON oi.order_id = o.id").
						Select("COALESCE(SUM(oi.quantity), 0)").
						Where("o.tenant_id = ?", tenantID).
						Where("o.created_at BETWEEN ? AND ?", monthStart, monthEnd).
						Where("o.status NOT IN ?", []string{"cancelled", "refunded"}).
						Scan(&quantitySum).Error

					if quantityErr == nil {
						result.ProductsSold = quantitySum
					}
				}
			} // Adicionar dados ao array (apenas se houver dados)
			if result.Revenue > 0 || result.Orders > 0 {
				monthlyData = append(monthlyData, ReportDataItem{
					Period:       monthStart.Format("2006-01"),
					Revenue:      result.Revenue,
					Orders:       int(result.Orders),
					Customers:    int(result.Customers),
					ProductsSold: int(result.ProductsSold),
				})
			}
		}

		// Verificar se há dados suficientes para comparação
		var totalOrdersAllTime int64
		h.db.Table("orders").
			Where("tenant_id = ?", tenantID).
			Count(&totalOrdersAllTime)

		// Contar número de meses com dados
		var monthsWithData int64
		h.db.Table("orders").
			Select("COUNT(DISTINCT DATE_TRUNC('month', created_at))").
			Where("tenant_id = ?", tenantID).
			Row().Scan(&monthsWithData)

		hasSufficientData := monthsWithData >= 2 && totalOrdersAllTime >= 5
		var comparisonMessage string
		if !hasSufficientData {
			if totalOrdersAllTime < 5 {
				comparisonMessage = "Dados insuficientes para comparação. Realize mais pedidos para ver tendências."
			} else {
				comparisonMessage = "Aguarde mais alguns dias para ver comparações com períodos anteriores."
			}
		} else {
			comparisonMessage = "Comparação disponível com períodos anteriores"
		}

		// Se não há dados reais, retornar array vazio com estrutura simples para compatibilidade
		if len(monthlyData) == 0 {
			// Para compatibilidade com frontend existente, retornar array vazio simples
			return c.JSON(http.StatusOK, []ReportDataItem{})
		}

		// Verificar se cliente quer nova estrutura (com header Accept específico) ou manter compatibilidade
		acceptHeader := c.Request().Header.Get("Accept")
		if acceptHeader == "application/json+metadata" {
			// Nova estrutura com metadados
			comparison := ComparisonMetadata{
				HasSufficientData: hasSufficientData,
				ComparisonPeriod:  "Últimos 6 meses",
				DataPoints:        int(totalOrdersAllTime),
				Message:           comparisonMessage,
			}

			response := ReportsResponse{
				Data:       map[string]interface{}{"monthly": monthlyData},
				Comparison: comparison,
			}
			return c.JSON(http.StatusOK, response)
		} else {
			// Estrutura original para compatibilidade
			return c.JSON(http.StatusOK, monthlyData)
		}

	case "daily":
		fmt.Printf("DEBUG: In daily case - implementing inline\n")

		// Implementação melhorada para incluir products_sold
		type PeriodResult struct {
			Revenue      float64 `gorm:"column:revenue"`
			Orders       int64   `gorm:"column:orders"`
			Customers    int64   `gorm:"column:customers"`
			ProductsSold int64   `gorm:"column:products_sold"`
		}

		var result PeriodResult

		// Query com JOIN para incluir order_items
		err := h.db.Table("orders o").
			Select(`
				COALESCE(SUM(CAST(o.total_amount AS DECIMAL)), 0) as revenue,
				COUNT(DISTINCT o.id) as orders,
				COUNT(DISTINCT o.customer_id) as customers,
				COALESCE(SUM(oi.quantity), 0) as products_sold
			`).
			Joins("LEFT JOIN order_items oi ON o.id = oi.order_id").
			Where("o.tenant_id = ?", tenantID).
			Where("o.created_at BETWEEN ? AND ?", startDate, endDate).
			Where("o.status NOT IN ?", []string{"cancelled", "refunded"}).
			Scan(&result).Error

		if err != nil {
			fmt.Printf("DEBUG: Error in daily query with JOIN: %v\n", err)
			// Fallback para query sem JOIN
			err = h.db.Table("orders").
				Select("COALESCE(SUM(CAST(total_amount AS DECIMAL)), 0) as revenue, COUNT(*) as orders, COUNT(DISTINCT customer_id) as customers, 0 as products_sold").
				Where("tenant_id = ?", tenantID).
				Where("created_at BETWEEN ? AND ?", startDate, endDate).
				Where("status NOT IN ?", []string{"cancelled", "refunded"}).
				Scan(&result).Error

			if err != nil {
				fmt.Printf("DEBUG: Error in daily fallback query: %v\n", err)
				result = PeriodResult{Revenue: 0, Orders: 0, Customers: 0, ProductsSold: 0}
			}
		}

		// Se products_sold está zero mas há orders, tentar calcular separadamente
		if result.ProductsSold == 0 && result.Orders > 0 {
			var productsSold int64
			err := h.db.Table("order_items oi").
				Joins("LEFT JOIN orders o ON oi.order_id = o.id").
				Select("COALESCE(SUM(oi.quantity), 0)").
				Where("o.tenant_id = ?", tenantID).
				Where("o.created_at BETWEEN ? AND ?", startDate, endDate).
				Where("o.status NOT IN ?", []string{"cancelled", "refunded"}).
				Scan(&productsSold).Error

			if err == nil {
				result.ProductsSold = productsSold
				fmt.Printf("DEBUG: Alternative products_sold calculation: %d\n", productsSold)
			}
		}

		fmt.Printf("DEBUG: Daily result - Revenue: %.2f, Orders: %d, Customers: %d, Products: %d\n",
			result.Revenue, result.Orders, result.Customers, result.ProductsSold)

		dailyData := []ReportDataItem{
			{
				Period:       "Período Selecionado",
				Revenue:      result.Revenue,
				Orders:       int(result.Orders),
				Customers:    int(result.Customers),
				ProductsSold: int(result.ProductsSold),
			},
		}

		// Para compatibilidade, retornar estrutura simples
		return c.JSON(http.StatusOK, dailyData)
	default:
		return c.JSON(http.StatusOK, []ReportDataItem{})
	}
}

// getMonthlyReportsData returns monthly aggregated data (for the chart)
// getDailyReportsData returns aggregated data for the selected period (for KPIs)
func (h *AnalyticsHandler) getDailyReportsData(c echo.Context, tenantID string, startDate, endDate time.Time) error {
	fmt.Printf("DEBUG: getDailyReportsData called for tenant: %s\n", tenantID)
	fmt.Printf("DEBUG: Date range: %s to %s\n", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	type PeriodResult struct {
		Revenue      float64 `gorm:"column:revenue"`
		Orders       int64   `gorm:"column:orders"`
		Customers    int64   `gorm:"column:customers"`
		ProductsSold int64   `gorm:"column:products_sold"`
	}

	var result PeriodResult

	// Consulta com timeout para evitar hang
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT 
			COALESCE(SUM(CAST(total_amount AS DECIMAL)), 0) as revenue,
			COUNT(id) as orders,
			COUNT(DISTINCT customer_id) as customers,
			0 as products_sold
		FROM orders 
		WHERE tenant_id = $1 
		  AND created_at BETWEEN $2 AND $3
		  AND status NOT IN ('cancelled', 'refunded')
	`

	err := h.db.WithContext(ctx).Raw(query, tenantID, startDate, endDate).Scan(&result).Error
	if err != nil {
		fmt.Printf("DEBUG: Error in daily query: %v\n", err)
		// Se der timeout, retornar dados básicos
		result = PeriodResult{Revenue: 0, Orders: 0, Customers: 0, ProductsSold: 0}
	}

	fmt.Printf("DEBUG: Period data - Revenue: %.2f, Orders: %d, Customers: %d, Products: %d\n",
		result.Revenue, result.Orders, result.Customers, result.ProductsSold)

	// Retornar dados agregados do período
	dailyData := []ReportDataItem{
		{
			Period:       "Período Selecionado",
			Revenue:      result.Revenue,
			Orders:       int(result.Orders),
			Customers:    int(result.Customers),
			ProductsSold: int(result.ProductsSold),
		},
	}

	fmt.Printf("DEBUG: Returning daily data with %d items\n", len(dailyData))
	return c.JSON(http.StatusOK, dailyData)
}

// GetTopProducts godoc
// @Summary Get top products report
// @Description Get top-selling products with sales metrics
// @Tags reports
// @Accept json
// @Produce json
// @Param limit query int false "Limit results" default(10)
// @Param period query string false "Period (daily, weekly, monthly)" default(monthly)
// @Param start_date query string false "Start date (YYYY-MM-DD)"
// @Param end_date query string false "End date (YYYY-MM-DD)"
// @Success 200 {array} TopProductItem
// @Failure 500 {object} map[string]string
// @Router /reports/top-products [get]
// @Security BearerAuth
func (h *AnalyticsHandler) GetTopProducts(c echo.Context) error {
	// Get tenant ID from context
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit <= 0 {
		limit = 10
	}

	// Parse date parameters
	startDateStr := c.QueryParam("start_date")
	endDateStr := c.QueryParam("end_date")

	// Default to last 30 days if no dates provided
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	if startDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = parsed
		}
	}
	if endDateStr != "" {
		if parsed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = parsed
		}
	}

	type ProductSales struct {
		ProductID    string  `gorm:"column:product_id"`
		ProductName  string  `gorm:"column:product_name"`
		SalesCount   int     `gorm:"column:sales_count"`
		TotalRevenue float64 `gorm:"column:total_revenue"`
	}

	var topProducts []ProductSales

	// Query para produtos mais vendidos no período
	err := h.db.Raw(`
		SELECT 
			p.id as product_id,
			p.name as product_name,
			COALESCE(SUM(oi.quantity), 0) as sales_count,
			COALESCE(SUM(oi.quantity * CAST(oi.unit_price AS DECIMAL)), 0) as total_revenue
		FROM products p
		INNER JOIN order_items oi ON oi.product_id = p.id
		INNER JOIN orders o ON o.id = oi.order_id
		WHERE o.tenant_id = ? 
			AND o.created_at >= ? 
			AND o.created_at <= ?
			AND o.status NOT IN ('cancelled', 'refunded')
		GROUP BY p.id, p.name
		ORDER BY sales_count DESC, total_revenue DESC
		LIMIT ?
	`, tenantID, startDate, endDate, limit).Scan(&topProducts).Error

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to fetch top products"})
	}

	// Convert to TopProductItem format
	result := make([]TopProductItem, len(topProducts))
	for i, product := range topProducts {
		result[i] = TopProductItem{
			ProductID:    product.ProductID,
			ProductName:  product.ProductName,
			SalesCount:   product.SalesCount,
			TotalRevenue: product.TotalRevenue,
		}
	}

	return c.JSON(http.StatusOK, result)
}

// ComparisonMetadata provides context about growth rate calculations
type ComparisonMetadata struct {
	HasSufficientData bool   `json:"has_sufficient_data"`
	ComparisonPeriod  string `json:"comparison_period"`
	DataPoints        int    `json:"data_points"`
	Message           string `json:"message"`
}

// OrderStatsResponse represents order statistics for the sales dashboard
type OrderStatsResponse struct {
	TotalOrders     int                `json:"total_orders"`
	PendingOrders   int                `json:"pending_orders"`
	DeliveredToday  int                `json:"delivered_today"`
	Revenue         float64            `json:"revenue"`
	GrowthRates     map[string]float64 `json:"growth_rates"`
	Comparison      ComparisonMetadata `json:"comparison"`
	StatusBreakdown map[string]int     `json:"status_breakdown"`
	RecentOrders    []OrderSummary     `json:"recent_orders"`
}

type OrderSummary struct {
	ID            string  `json:"id"`
	OrderNumber   string  `json:"order_number"`
	CustomerName  string  `json:"customer_name"`
	Status        string  `json:"status"`
	PaymentStatus string  `json:"payment_status"`
	TotalAmount   float64 `json:"total_amount"`
	ItemsCount    int     `json:"items_count"`
	CreatedAt     string  `json:"created_at"`
}

// GetOrderStats godoc
// @Summary Get order statistics for sales dashboard
// @Description Get comprehensive order statistics and metrics for the sales dashboard
// @Tags analytics
// @Accept json
// @Produce json
// @Success 200 {object} OrderStatsResponse
// @Failure 500 {object} map[string]string
// @Router /analytics/orders [get]
// @Security BearerAuth
func (h *AnalyticsHandler) GetOrderStats(c echo.Context) error {
	// Get tenant ID from context
	tenantID := c.Get("tenant_id")
	if tenantID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Total orders this month
	var totalOrders int64
	h.db.Table("orders").
		Where("tenant_id = ?", tenantID).
		Where("created_at >= ?", startOfMonth).
		Count(&totalOrders)

	// Pending orders
	var pendingOrders int64
	h.db.Table("orders").
		Where("tenant_id = ?", tenantID).
		Where("status = ?", "pending").
		Count(&pendingOrders)

	// Delivered today
	var deliveredToday int64
	h.db.Table("orders").
		Where("tenant_id = ?", tenantID).
		Where("status = ?", "delivered").
		Where("updated_at >= ?", startOfToday).
		Count(&deliveredToday)

	// Revenue this month
	var revenue float64
	h.db.Table("orders").
		Where("tenant_id = ?", tenantID).
		Where("created_at >= ?", startOfMonth).
		Where("status NOT IN ?", []string{"cancelled", "refunded"}).
		Select("COALESCE(SUM(CAST(total_amount AS DECIMAL)), 0)").
		Row().Scan(&revenue)

	// Calculate growth rates (compare with previous month)
	lastMonth := startOfMonth.AddDate(0, -1, 0)
	var lastMonthOrders int64
	var lastMonthRevenue float64

	h.db.Table("orders").
		Where("tenant_id = ?", tenantID).
		Where("created_at >= ? AND created_at < ?", lastMonth, startOfMonth).
		Count(&lastMonthOrders)

	h.db.Table("orders").
		Where("tenant_id = ?", tenantID).
		Where("created_at >= ? AND created_at < ?", lastMonth, startOfMonth).
		Where("status NOT IN ?", []string{"cancelled", "refunded"}).
		Select("COALESCE(SUM(CAST(total_amount AS DECIMAL)), 0)").
		Row().Scan(&lastMonthRevenue)

	// Check if we have sufficient data for meaningful comparison
	var totalOrdersAllTime int64
	h.db.Table("orders").
		Where("tenant_id = ?", tenantID).
		Count(&totalOrdersAllTime)

	// Count number of months with data
	var monthsWithData int64
	h.db.Table("orders").
		Select("COUNT(DISTINCT DATE_TRUNC('month', created_at))").
		Where("tenant_id = ?", tenantID).
		Row().Scan(&monthsWithData)

	hasSufficientData := monthsWithData >= 2 && totalOrdersAllTime >= 5
	comparisonPeriod := lastMonth.Format("January 2006")

	var comparisonMessage string
	if !hasSufficientData {
		if totalOrdersAllTime < 5 {
			comparisonMessage = "Dados insuficientes para comparação. Realize mais pedidos para ver tendências."
		} else {
			comparisonMessage = "Aguarde mais alguns dias para ver comparações com períodos anteriores."
		}
	} else {
		comparisonMessage = fmt.Sprintf("Comparando com %s", comparisonPeriod)
	}

	growthRates := map[string]float64{
		"orders":  0,
		"revenue": 0,
	}

	if hasSufficientData {
		if lastMonthOrders > 0 {
			growthRates["orders"] = ((float64(totalOrders) - float64(lastMonthOrders)) / float64(lastMonthOrders)) * 100
		}
		if lastMonthRevenue > 0 {
			growthRates["revenue"] = ((revenue - lastMonthRevenue) / lastMonthRevenue) * 100
		}
	}

	comparison := ComparisonMetadata{
		HasSufficientData: hasSufficientData,
		ComparisonPeriod:  comparisonPeriod,
		DataPoints:        int(totalOrdersAllTime),
		Message:           comparisonMessage,
	}

	// Status breakdown
	type StatusCount struct {
		Status string `gorm:"column:status"`
		Count  int    `gorm:"column:count"`
	}

	var statusCounts []StatusCount
	h.db.Table("orders").
		Select("status, COUNT(*) as count").
		Where("tenant_id = ?", tenantID).
		Group("status").
		Find(&statusCounts)

	statusBreakdown := make(map[string]int)
	for _, sc := range statusCounts {
		statusBreakdown[sc.Status] = sc.Count
	}

	// Recent orders
	type OrderWithCustomer struct {
		ID            string    `gorm:"column:id"`
		OrderNumber   string    `gorm:"column:order_number"`
		Status        string    `gorm:"column:status"`
		PaymentStatus string    `gorm:"column:payment_status"`
		TotalAmount   float64   `gorm:"column:total_amount"`
		CreatedAt     time.Time `gorm:"column:created_at"`
		CustomerName  string    `gorm:"column:customer_name"`
	}

	var recentOrdersData []OrderWithCustomer
	h.db.Table("orders").
		Select(`
			orders.id, 
			orders.order_number, 
			orders.status, 
			orders.payment_status, 
			CAST(orders.total_amount AS DECIMAL) as total_amount, 
			orders.created_at,
			customers.name as customer_name
		`).
		Joins("JOIN customers ON customers.id = orders.customer_id").
		Where("orders.tenant_id = ?", tenantID).
		Order("orders.created_at DESC").
		Limit(10).
		Find(&recentOrdersData)

	recentOrders := make([]OrderSummary, len(recentOrdersData))
	for i, order := range recentOrdersData {
		// Count items in this order
		var itemsCount int64
		h.db.Table("order_items").
			Where("order_id = ?", order.ID).
			Select("COALESCE(SUM(quantity), 0)").
			Row().Scan(&itemsCount)

		recentOrders[i] = OrderSummary{
			ID:            order.ID,
			OrderNumber:   order.OrderNumber,
			CustomerName:  order.CustomerName,
			Status:        order.Status,
			PaymentStatus: order.PaymentStatus,
			TotalAmount:   order.TotalAmount,
			ItemsCount:    int(itemsCount),
			CreatedAt:     order.CreatedAt.Format("2006-01-02 15:04:05"),
		}
	}

	response := OrderStatsResponse{
		TotalOrders:     int(totalOrders),
		PendingOrders:   int(pendingOrders),
		DeliveredToday:  int(deliveredToday),
		Revenue:         revenue,
		GrowthRates:     growthRates,
		Comparison:      comparison,
		StatusBreakdown: statusBreakdown,
		RecentOrders:    recentOrders,
	}

	return c.JSON(http.StatusOK, response)
}
