package services

import (
	"fmt"
	"iafarma/internal/ai"
	"iafarma/pkg/models"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

type ProductServiceImpl struct {
	db *gorm.DB
}

func NewProductService(db *gorm.DB) ai.ProductServiceInterface {
	return &ProductServiceImpl{db: db}
}

func (s *ProductServiceImpl) SearchProducts(tenantID uuid.UUID, query string, limit int) ([]models.Product, error) {
	return s.SearchProductsAdvanced(tenantID, ai.ProductSearchFilters{
		Query: query,
		Limit: limit,
	})
}

// SearchProductsAdvanced searches products with advanced filters
func (s *ProductServiceImpl) SearchProductsAdvanced(tenantID uuid.UUID, filters ai.ProductSearchFilters) ([]models.Product, error) {
	var products []models.Product

	// FILTRO OBRIGATÓRIO: Apenas produtos com estoque > 0 (disponíveis para venda)
	dbQuery := s.db.Where("tenant_id = ? AND stock_quantity > 0", tenantID)

	// Full Text Search usando PostgreSQL FTS
	if filters.Query != "" {
		searchQuery := strings.TrimSpace(filters.Query)

		// Log para debug
		log.Info().Msgf("🔍 FTS Debug: searchQuery='%s'", searchQuery)

		// Estratégia de busca melhorada: SEMPRE usar FTS primeiro quando disponível
		// 1. Converter busca para formato AND - palavras separadas por &
		tsqueryFormat := strings.ReplaceAll(searchQuery, " ", " & ")
		log.Info().Msgf("🔍 FTS Debug: tsqueryFormat='%s'", tsqueryFormat)

		// Testar FTS diretamente com operador AND
		ftsQuery := s.db.Where("tenant_id = ? AND stock_quantity > 0", tenantID).
			Where("search_vector @@ to_tsquery('portuguese', ?)", tsqueryFormat)

		// Verificar se FTS encontrou resultados
		var ftsCount int64
		ftsQuery.Model(&models.Product{}).Count(&ftsCount)

		log.Info().Msgf("🔍 FTS Debug: ftsCount=%d", ftsCount)

		if ftsCount > 0 {
			// PRIORIDADE 1: Usar FTS se encontrou resultados - ordenar por relevância
			log.Info().Msg("🔍 FTS Debug: Using FTS query (PRIORITY)")
			dbQuery = ftsQuery.Select("*, ts_rank(search_vector, to_tsquery('portuguese', ?)) as rank", tsqueryFormat).
				Order("rank DESC, stock_quantity DESC, name ASC")
		} else {
			// PRIORIDADE 2: Busca exata no nome se FTS não encontrou
			exactNameQuery := s.db.Where("tenant_id = ? AND stock_quantity > 0", tenantID).
				Where("LOWER(name) LIKE LOWER(?)", "%"+searchQuery+"%")

			var exactNameCount int64
			exactNameQuery.Model(&models.Product{}).Count(&exactNameCount)

			log.Info().Msgf("🔍 FTS Debug: exactNameCount=%d", exactNameCount)

			if exactNameCount > 0 {
				// Usar busca exata no nome como fallback
				log.Info().Msg("🔍 FTS Debug: Using exact name match (fallback)")
				dbQuery = exactNameQuery.Order("stock_quantity DESC, name ASC")
			} else {
				// PRIORIDADE 3: Fallback para busca LIKE ampla
				log.Info().Msg("🔍 FTS Debug: Using LIKE fallback")
				searchPattern := "%" + strings.ToLower(searchQuery) + "%"
				dbQuery = dbQuery.Where(
					"LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(brand) LIKE ? OR LOWER(tags) LIKE ? OR LOWER(sku) LIKE ?",
					searchPattern, searchPattern, searchPattern, searchPattern, searchPattern,
				).Order("stock_quantity DESC, name ASC")
			}
		}
	}

	// Filtro por marca
	if filters.Brand != "" {
		dbQuery = dbQuery.Where("LOWER(brand) LIKE LOWER(?)", "%"+filters.Brand+"%")
	}

	// Filtro por tags
	if filters.Tags != "" {
		dbQuery = dbQuery.Where("LOWER(tags) LIKE LOWER(?)", "%"+filters.Tags+"%")
	}

	// Filtro por preço mínimo
	if filters.MinPrice > 0 {
		dbQuery = dbQuery.Where("CAST(price AS DECIMAL) >= ?", filters.MinPrice)
	}

	// Filtro por preço máximo
	if filters.MaxPrice > 0 {
		dbQuery = dbQuery.Where("CAST(price AS DECIMAL) <= ?", filters.MaxPrice)
	}

	// Aplicar ordenação padrão se não foi definida antes (para casos sem busca)
	if filters.Query == "" {
		dbQuery = dbQuery.Order("stock_quantity DESC, name ASC")
	}

	// Aplicar limite
	if filters.Limit > 0 {
		dbQuery = dbQuery.Limit(filters.Limit)
	}

	err := dbQuery.Find(&products).Error
	// Print SQL gerado para debug
	sql := dbQuery.Statement.SQL.String()
	log.Info().Msgf("🔍 FTS Debug: Generated SQL: %s", sql)
	return products, err
}

func (s *ProductServiceImpl) GetPromotionalProducts(tenantID uuid.UUID) ([]models.Product, error) {
	var products []models.Product

	err := s.db.Where("tenant_id = ? AND sale_price IS NOT NULL AND sale_price != '' AND sale_price != '0'",
		tenantID).Find(&products).Error
	return products, err
}

func (s *ProductServiceImpl) GetProductByID(tenantID, productID uuid.UUID) (*models.Product, error) {
	var product models.Product

	err := s.db.Where("tenant_id = ? AND id = ?", tenantID, productID).First(&product).Error
	if err != nil {
		return nil, err
	}

	return &product, nil
}

type CartServiceImpl struct {
	db *gorm.DB
}

func NewCartService(db *gorm.DB) ai.CartServiceInterface {
	return &CartServiceImpl{db: db}
}

func (s *CartServiceImpl) GetOrCreateActiveCart(tenantID, customerID uuid.UUID) (*models.Cart, error) {
	var cart models.Cart

	// Tentar encontrar carrinho ativo
	err := s.db.Where("tenant_id = ? AND customer_id = ? AND status = 'active'",
		tenantID, customerID).First(&cart).Error

	if err == gorm.ErrRecordNotFound {
		// Criar novo carrinho
		cart = models.Cart{
			BaseTenantModel: models.BaseTenantModel{
				ID:       uuid.New(),
				TenantID: tenantID,
			},
			CustomerID: customerID,
			Status:     "active",
		}
		err = s.db.Create(&cart).Error
	}

	return &cart, err
}

func (s *CartServiceImpl) AddItemToCart(cartID, tenantID, productID uuid.UUID, quantity int) error {
	// Verificar se item já existe no carrinho
	var existingItem models.CartItem
	err := s.db.Where("cart_id = ? AND product_id = ?", cartID, productID).First(&existingItem).Error

	if err == gorm.ErrRecordNotFound {
		// Obter dados do produto para histórico
		var product models.Product
		if err := s.db.Where("id = ? AND tenant_id = ?", productID, tenantID).First(&product).Error; err != nil {
			return err
		}

		// Criar novo item
		item := models.CartItem{
			BaseTenantModel: models.BaseTenantModel{
				ID:       uuid.New(),
				TenantID: tenantID,
			},
			CartID:      cartID,
			ProductID:   &productID,
			Quantity:    quantity,
			Price:       product.Price,
			ProductName: &product.Name,
			ProductSKU:  &product.SKU,
		}
		return s.db.Create(&item).Error
	} else if err != nil {
		return err
	} else {
		// Atualizar quantidade do item existente
		existingItem.Quantity += quantity
		return s.db.Save(&existingItem).Error
	}
}

func (s *CartServiceImpl) RemoveItemFromCart(cartID, tenantID, itemID uuid.UUID) error {
	return s.db.Where("cart_id = ? AND id = ?", cartID, itemID).Delete(&models.CartItem{}).Error
}

func (s *CartServiceImpl) GetCartWithItems(cartID, tenantID uuid.UUID) (*models.Cart, error) {
	var cart models.Cart

	err := s.db.Preload("Items").Preload("Items.Product").
		Where("id = ? AND tenant_id = ?", cartID, tenantID).First(&cart).Error

	return &cart, err
}

func (s *CartServiceImpl) ClearCart(cartID, tenantID uuid.UUID) error {
	return s.db.Where("cart_id = ?", cartID).Delete(&models.CartItem{}).Error
}

func (s *CartServiceImpl) UpdateCartItemQuantity(cartID, tenantID, itemID uuid.UUID, quantity int) error {
	return s.db.Model(&models.CartItem{}).
		Where("cart_id = ? AND id = ?", cartID, itemID).
		Update("quantity", quantity).Error
}

type OrderServiceImpl struct {
	db *gorm.DB
}

func NewOrderService(db *gorm.DB) ai.OrderServiceInterface {
	return &OrderServiceImpl{db: db}
}

func (s *OrderServiceImpl) CreateOrderFromCart(tenantID, cartID uuid.UUID) (*models.Order, error) {
	return s.CreateOrderFromCartWithAddress(tenantID, cartID, nil)
}

func (s *OrderServiceImpl) CreateOrderFromCartWithAddress(tenantID, cartID uuid.UUID, deliveryAddress *models.Address) (*models.Order, error) {
	return s.CreateOrderFromCartWithConversation(tenantID, cartID, uuid.Nil, deliveryAddress)
}

func (s *OrderServiceImpl) CreateOrderFromCartWithConversation(tenantID, cartID, conversationID uuid.UUID, deliveryAddress *models.Address) (*models.Order, error) {
	// Obter carrinho com itens e cliente
	var cart models.Cart
	err := s.db.Preload("Items").Preload("Items.Product").Preload("Customer").
		Where("id = ? AND tenant_id = ?", cartID, tenantID).First(&cart).Error
	if err != nil {
		return nil, err
	}

	if len(cart.Items) == 0 {
		return nil, fmt.Errorf("carrinho vazio")
	}

	// Calcular total e subtotal
	subtotal := 0.0
	for _, item := range cart.Items {
		price, _ := strconv.ParseFloat(item.Price, 64)
		subtotal += price * float64(item.Quantity)
	}

	// Criar pedido com status pendente (não requer pagamento imediato)
	order := models.Order{
		BaseTenantModel: models.BaseTenantModel{
			ID:       uuid.New(),
			TenantID: tenantID,
		},
		CustomerID:        &cart.CustomerID,
		OrderNumber:       generateOrderNumber(),
		Status:            "pending",
		PaymentStatus:     "pending", // Pagamento pendente - será processado depois
		FulfillmentStatus: "pending",
		TotalAmount:       fmt.Sprintf("%.2f", subtotal),
		Subtotal:          fmt.Sprintf("%.2f", subtotal),
		TaxAmount:         "0.00",
		ShippingAmount:    "0.00",
		DiscountAmount:    "0.00",
		Currency:          "BRL",
	}

	// Adicionar ConversationID se fornecido
	if conversationID != uuid.Nil {
		order.ConversationID = &conversationID
	}

	// Copiar dados históricos do cliente
	if cart.Customer != nil {
		order.CustomerName = &cart.Customer.Name
		order.CustomerEmail = &cart.Customer.Email
		order.CustomerPhone = &cart.Customer.Phone
		order.CustomerDocument = &cart.Customer.Document
	}

	// 🏠 Copiar dados do endereço de entrega se fornecido
	if deliveryAddress != nil {
		order.AddressID = &deliveryAddress.ID
		order.ShippingName = &deliveryAddress.Name
		order.ShippingStreet = &deliveryAddress.Street
		order.ShippingNumber = &deliveryAddress.Number
		order.ShippingComplement = &deliveryAddress.Complement
		order.ShippingNeighborhood = &deliveryAddress.Neighborhood
		order.ShippingCity = &deliveryAddress.City
		order.ShippingState = &deliveryAddress.State
		order.ShippingZipcode = &deliveryAddress.ZipCode
		order.ShippingCountry = &deliveryAddress.Country
	}

	// Iniciar transação para garantir consistência
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Criar pedido
	err = tx.Create(&order).Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Criar itens do pedido
	for _, cartItem := range cart.Items {
		itemPrice, _ := strconv.ParseFloat(cartItem.Price, 64)
		itemTotal := itemPrice * float64(cartItem.Quantity)

		orderItem := models.OrderItem{
			BaseTenantModel: models.BaseTenantModel{
				ID:       uuid.New(),
				TenantID: tenantID,
			},
			OrderID:   order.ID,
			ProductID: cartItem.ProductID,
			Quantity:  cartItem.Quantity,
			Price:     cartItem.Price,
			Total:     fmt.Sprintf("%.2f", itemTotal),
		}

		// Copiar dados históricos do produto
		if cartItem.Product != nil {
			orderItem.ProductName = &cartItem.Product.Name
			orderItem.ProductDescription = &cartItem.Product.Description
			orderItem.ProductSKU = &cartItem.Product.SKU
			orderItem.UnitPrice = &cartItem.Product.Price
		}

		err = tx.Create(&orderItem).Error
		if err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	// Marcar carrinho como processado
	err = tx.Model(&cart).Update("status", "processed").Error
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// Confirmar transação
	err = tx.Commit().Error
	if err != nil {
		return nil, err
	}

	// Recarregar o pedido com todos os dados
	err = s.db.Preload("Items").Where("id = ?", order.ID).First(&order).Error
	if err != nil {
		return nil, err
	}

	return &order, nil
}

func (s *OrderServiceImpl) CancelOrder(tenantID, orderID uuid.UUID) error {
	return s.db.Model(&models.Order{}).
		Where("id = ? AND tenant_id = ?", orderID, tenantID).
		Update("status", "cancelled").Error
}

func (s *OrderServiceImpl) GetOrdersByCustomer(tenantID, customerID uuid.UUID) ([]models.Order, error) {
	var orders []models.Order

	err := s.db.Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Order("created_at DESC").Find(&orders).Error

	return orders, err
}

func (s *OrderServiceImpl) GetPaymentOptions(tenantID uuid.UUID) ([]ai.PaymentOption, error) {
	// Por enquanto, retornar opções padrão
	// No futuro, pode vir do banco de dados por tenant
	options := []ai.PaymentOption{
		{
			ID:           "pix",
			Name:         "PIX",
			Instructions: "Envie PIX para a chave: vendas@empresa.com",
		},
		{
			ID:           "dinheiro",
			Name:         "Dinheiro",
			Instructions: "Pagamento na entrega em dinheiro",
		},
		{
			ID:           "cartao",
			Name:         "Cartão na Entrega",
			Instructions: "Pagamento com cartão na entrega",
		},
	}

	return options, nil
}

type CustomerServiceImpl struct {
	db *gorm.DB
}

func NewCustomerService(db *gorm.DB) ai.CustomerServiceInterface {
	return &CustomerServiceImpl{db: db}
}

func (s *CustomerServiceImpl) GetCustomerByPhone(tenantID uuid.UUID, phone string) (*models.Customer, error) {
	var customer models.Customer

	err := s.db.Where("tenant_id = ? AND phone = ?", tenantID, phone).First(&customer).Error
	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func (s *CustomerServiceImpl) GetCustomerByID(tenantID, customerID uuid.UUID) (*models.Customer, error) {
	var customer models.Customer

	err := s.db.Where("tenant_id = ? AND id = ?", tenantID, customerID).First(&customer).Error
	if err != nil {
		return nil, err
	}

	return &customer, nil
}

func (s *CustomerServiceImpl) UpdateCustomerProfile(tenantID, customerID uuid.UUID, data ai.CustomerUpdateData) error {
	updates := make(map[string]interface{})

	if data.Name != "" {
		updates["name"] = data.Name
	}
	if data.Email != "" {
		updates["email"] = data.Email
	}

	if len(updates) > 0 {
		err := s.db.Model(&models.Customer{}).
			Where("id = ? AND tenant_id = ?", customerID, tenantID).
			Updates(updates).Error
		if err != nil {
			return err
		}
	}

	// Se tem endereço, criar/atualizar registro de endereço
	if data.Address != "" {
		address := models.Address{
			BaseTenantModel: models.BaseTenantModel{
				ID:       uuid.New(),
				TenantID: tenantID,
			},
			CustomerID: customerID,
			Street:     data.Address,
			IsDefault:  true,
		}
		return s.db.Create(&address).Error
	}

	return nil
}

type AddressServiceImpl struct {
	db *gorm.DB
}

func NewAddressService(db *gorm.DB) ai.AddressServiceInterface {
	return &AddressServiceImpl{db: db}
}

func (s *AddressServiceImpl) GetAddressesByCustomer(tenantID, customerID uuid.UUID) ([]models.Address, error) {
	var addresses []models.Address

	err := s.db.Where("tenant_id = ? AND customer_id = ?", tenantID, customerID).
		Order("is_default DESC, created_at ASC").
		Find(&addresses).Error

	return addresses, err
}

func (s *AddressServiceImpl) CreateAddress(tenantID uuid.UUID, address *models.Address) error {
	address.TenantID = tenantID
	if address.ID == (uuid.UUID{}) {
		address.ID = uuid.New()
	}
	return s.db.Create(address).Error
}

func (s *AddressServiceImpl) SetDefaultAddress(tenantID, customerID, addressID uuid.UUID) error {
	// Primeiro, remover o padrão de todos os endereços do cliente
	err := s.db.Model(&models.Address{}).
		Where("customer_id = ? AND tenant_id = ?", customerID, tenantID).
		Update("is_default", false).Error
	if err != nil {
		return err
	}

	// Depois, definir o endereço especificado como padrão
	return s.db.Model(&models.Address{}).
		Where("id = ? AND customer_id = ? AND tenant_id = ?", addressID, customerID, tenantID).
		Update("is_default", true).Error
}

func (s *AddressServiceImpl) DeleteAddress(tenantID, customerID, addressID uuid.UUID) error {
	// Verificar se é o último endereço
	var count int64
	err := s.db.Model(&models.Address{}).
		Where("customer_id = ? AND tenant_id = ?", customerID, tenantID).
		Count(&count).Error
	if err != nil {
		return err
	}

	// Deletar o endereço
	err = s.db.Where("id = ? AND customer_id = ? AND tenant_id = ?", addressID, customerID, tenantID).
		Delete(&models.Address{}).Error
	if err != nil {
		return err
	}

	// Se deletou um endereço padrão e ainda há outros, definir o primeiro como padrão
	if count > 1 {
		var firstAddress models.Address
		err = s.db.Where("customer_id = ? AND tenant_id = ?", customerID, tenantID).
			Order("created_at ASC").First(&firstAddress).Error
		if err == nil {
			s.db.Model(&firstAddress).Update("is_default", true)
		}
	}

	return nil
}

func (s *AddressServiceImpl) DeleteAllAddresses(tenantID, customerID uuid.UUID) error {
	return s.db.Where("customer_id = ? AND tenant_id = ?", customerID, tenantID).
		Delete(&models.Address{}).Error
}

// GetAllTenants returns all tenants from database
func (s *ProductServiceImpl) GetAllTenants() ([]models.Tenant, error) {
	var tenants []models.Tenant
	err := s.db.Find(&tenants).Error
	return tenants, err
}

// GetTenantByID returns a specific tenant by ID
func (s *ProductServiceImpl) GetTenantByID(tenantID uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	err := s.db.Where("id = ?", tenantID).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

// GetProductsByTenantID returns all products for a specific tenant
func (s *ProductServiceImpl) GetProductsByTenantID(tenantID uuid.UUID) ([]models.Product, error) {
	var products []models.Product
	err := s.db.Where("tenant_id = ?", tenantID).Find(&products).Error
	return products, err
}

// Funções auxiliares
func generateOrderNumber() string {
	return fmt.Sprintf("PED%d", uuid.New().ID())
}
