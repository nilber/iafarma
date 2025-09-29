package models

// ProductImportRequest represents a product import request
type ProductImportRequest struct {
	Products []ProductImportItem `json:"products" validate:"required,dive"`
}

// ProductImportItem represents a single product for import
type ProductImportItem struct {
	Name              string `json:"name" validate:"required"`
	Description       string `json:"description"`
	Price             string `json:"price" validate:"required"`
	SalePrice         string `json:"sale_price"`
	SKU               string `json:"sku"`
	Barcode           string `json:"barcode"`
	Weight            string `json:"weight"`
	Dimensions        string `json:"dimensions"`
	Brand             string `json:"brand"`
	Tags              string `json:"tags"`
	StockQuantity     int    `json:"stock_quantity"`
	LowStockThreshold int    `json:"low_stock_threshold"`
	CategoryName      string `json:"category_name"` // Optional: name of category for lookup
}

// ProductImportResult represents the result of a product import
type ProductImportResult struct {
	TotalProcessed int                       `json:"total_processed"`
	Created        int                       `json:"created"`
	Updated        int                       `json:"updated"`
	Errors         int                       `json:"errors"`
	Results        []ProductImportItemResult `json:"results"`
}

// ProductImportItemResult represents the result of importing a single product
type ProductImportItemResult struct {
	RowNumber int     `json:"row_number"`
	Name      string  `json:"name"`
	SKU       string  `json:"sku"`
	Status    string  `json:"status"` // "created", "updated", "error"
	Message   string  `json:"message"`
	ProductID *string `json:"product_id,omitempty"`
	Error     string  `json:"error,omitempty"`
}

// ProductImageImportResult represents the result of importing products from an image
type ProductImageImportResult struct {
	Success      bool                        `json:"success"`
	CreatedCount int                         `json:"created_count"`
	CreditsUsed  int                         `json:"credits_used"`
	Products     []ProductImageImportProduct `json:"products,omitempty"`
	Errors       []string                    `json:"errors,omitempty"`
}

// ProductImageImportProduct represents a product detected in an image
type ProductImageImportProduct struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Price       string `json:"price,omitempty"`
	Tags        string `json:"tags,omitempty"`
	Status      string `json:"status"` // "success", "error"
}
