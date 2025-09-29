package ai

import (
	"fmt"
	"iafarma/pkg/models"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"gorm.io/gorm"
)

// ProductServiceImpl implementa ProductServiceInterface
type ProductServiceImpl struct {
	db *gorm.DB
}

func NewProductService(db *gorm.DB) ProductServiceInterface {
	return &ProductServiceImpl{db: db}
}

func (s *ProductServiceImpl) SearchProducts(tenantID uuid.UUID, query string, limit int) ([]models.Product, error) {
	return s.SearchProductsAdvanced(tenantID, ProductSearchFilters{
		Query: query,
		Limit: limit,
	})
}

// SearchProductsAdvanced searches products with advanced filters using PostgreSQL FTS
func (s *ProductServiceImpl) SearchProductsAdvanced(tenantID uuid.UUID, filters ProductSearchFilters) ([]models.Product, error) {
	var products []models.Product

	// FILTRO OBRIGATÓRIO: Apenas produtos com estoque > 0 (disponíveis para venda)
	dbQuery := s.db.Where("tenant_id = ? AND stock_quantity > 0", tenantID)

	// Full Text Search usando PostgreSQL FTS
	if filters.Query != "" {
		searchQuery := strings.TrimSpace(filters.Query)

		// Log para debug
		log.Info().Msgf("🔍 FTS Debug: searchQuery='%s'", searchQuery)

		// NOVA IMPLEMENTAÇÃO: Usar sistema de abreviações farmacêuticas
		expandedQuery := BuildAdvancedSearchQuery(searchQuery)
		log.Info().Msgf("🔍 Pharmacy Abbreviations: original='%s' -> expanded='%s'", searchQuery, expandedQuery)

		// Estratégia de busca melhorada: SEMPRE usar FTS primeiro quando disponível
		// 1. Usar a query expandida com abreviações
		tsqueryFormat := expandedQuery
		log.Info().Msgf("🔍 FTS Debug: tsqueryFormat='%s'", tsqueryFormat)

		// Testar FTS diretamente com operador AND/OR expandido
		ftsQuery := s.db.Where("tenant_id = ? AND stock_quantity > 0", tenantID).
			Where("search_text @@ to_tsquery('portuguese', ?)", tsqueryFormat)

		// Verificar se FTS encontrou resultados
		var ftsCount int64
		ftsQuery.Model(&models.Product{}).Count(&ftsCount)

		log.Info().Msgf("🔍 FTS Debug: ftsCount=%d", ftsCount)

		if ftsCount > 0 {
			// PRIORIDADE 1: Usar FTS se encontrou resultados - ordenar por relevância
			log.Info().Msg("🔍 FTS Debug: Using FTS query with abbreviations (PRIORITY)")
			dbQuery = ftsQuery

			// Aplicar ordenação conforme solicitado
			switch filters.SortBy {
			case "price_asc":
				dbQuery = dbQuery.Order("CAST(price AS DECIMAL) ASC")
			case "price_desc":
				dbQuery = dbQuery.Order("CAST(price AS DECIMAL) DESC")
			case "name_asc":
				dbQuery = dbQuery.Order("name ASC")
			case "name_desc":
				dbQuery = dbQuery.Order("name DESC")
			default: // "relevance" ou vazio - sem ts_rank por enquanto
				dbQuery = dbQuery.Order("stock_quantity DESC, name ASC")
			}
		} else {
			// PRIORIDADE 2: Fallback para busca simples AND se query expandida não funcionou
			simpleAndQuery := strings.ReplaceAll(searchQuery, " ", " & ")

			simpleFtsQuery := s.db.Where("tenant_id = ? AND stock_quantity > 0", tenantID).
				Where("search_text @@ to_tsquery('portuguese', ?)", simpleAndQuery)

			var simpleFtsCount int64
			simpleFtsQuery.Model(&models.Product{}).Count(&simpleFtsCount)

			log.Info().Msgf("🔍 FTS Debug: simpleFtsCount=%d", simpleFtsCount)

			if simpleFtsCount > 0 {
				log.Info().Msg("🔍 FTS Debug: Using simple FTS query (fallback)")
				dbQuery = simpleFtsQuery.Select("*, ts_rank(search_text, to_tsquery('portuguese', ?)) as rank", simpleAndQuery)

				// Aplicar ordenação conforme solicitado
				switch filters.SortBy {
				case "price_asc":
					dbQuery = dbQuery.Order("CAST(price AS DECIMAL) ASC")
				case "price_desc":
					dbQuery = dbQuery.Order("CAST(price AS DECIMAL) DESC")
				case "name_asc":
					dbQuery = dbQuery.Order("name ASC")
				case "name_desc":
					dbQuery = dbQuery.Order("name DESC")
				default: // "relevance" ou vazio
					dbQuery = dbQuery.Order("rank DESC, stock_quantity DESC, name ASC")
				}
			} else {
				// PRIORIDADE 3: Busca exata no nome se FTS não encontrou
				exactNameQuery := s.db.Where("tenant_id = ? AND stock_quantity > 0", tenantID).
					Where("LOWER(name) LIKE LOWER(?)", "%"+searchQuery+"%")

				var exactNameCount int64
				exactNameQuery.Model(&models.Product{}).Count(&exactNameCount)

				log.Info().Msgf("🔍 FTS Debug: exactNameCount=%d", exactNameCount)

				if exactNameCount > 0 {
					// Usar busca exata no nome como fallback
					log.Info().Msg("🔍 FTS Debug: Using exact name match (fallback)")
					dbQuery = exactNameQuery

					// Aplicar ordenação conforme solicitado
					switch filters.SortBy {
					case "price_asc":
						dbQuery = dbQuery.Order("CAST(price AS DECIMAL) ASC")
					case "price_desc":
						dbQuery = dbQuery.Order("CAST(price AS DECIMAL) DESC")
					case "name_asc":
						dbQuery = dbQuery.Order("name ASC")
					case "name_desc":
						dbQuery = dbQuery.Order("name DESC")
					default:
						dbQuery = dbQuery.Order("stock_quantity DESC, name ASC")
					}
				} else {
					// PRIORIDADE 4: Fallback para busca LIKE ampla com abreviações
					log.Info().Msg("🔍 FTS Debug: Using LIKE fallback with abbreviations")

					// Construir busca LIKE que inclui abreviações
					searchWords := regexp.MustCompile(`\s+`).Split(searchQuery, -1)
					var whereConditions []string
					var whereArgs []interface{}

					for _, word := range searchWords {
						cleanWord := strings.TrimSpace(word)
						if cleanWord == "" {
							continue
						}

						// Obter abreviações para a palavra
						abbreviations := GetAbbreviationsForWord(cleanWord)

						// Construir condição LIKE para a palavra e suas abreviações
						var wordConditions []string
						searchTerms := []string{cleanWord}
						if abbreviations != nil {
							searchTerms = append(searchTerms, abbreviations...)
						}

						for _, term := range searchTerms {
							termPattern := "%" + strings.ToLower(term) + "%"
							wordConditions = append(wordConditions,
								"(LOWER(name) LIKE ? OR LOWER(description) LIKE ? OR LOWER(brand) LIKE ? OR LOWER(tags) LIKE ? OR LOWER(sku) LIKE ?)")
							whereArgs = append(whereArgs, termPattern, termPattern, termPattern, termPattern, termPattern)
						}

						if len(wordConditions) > 0 {
							whereConditions = append(whereConditions, "("+strings.Join(wordConditions, " OR ")+")")
						}
					}

					if len(whereConditions) > 0 {
						dbQuery = dbQuery.Where(strings.Join(whereConditions, " AND "), whereArgs...)
					}

					// Aplicar ordenação conforme solicitado
					switch filters.SortBy {
					case "price_asc":
						dbQuery = dbQuery.Order("CAST(price AS DECIMAL) ASC")
					case "price_desc":
						dbQuery = dbQuery.Order("CAST(price AS DECIMAL) DESC")
					case "name_asc":
						dbQuery = dbQuery.Order("name ASC")
					case "name_desc":
						dbQuery = dbQuery.Order("name DESC")
					default:
						dbQuery = dbQuery.Order("stock_quantity DESC, name ASC")
					}
				}
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

	// Aplicar limite
	if filters.Limit > 0 {
		dbQuery = dbQuery.Limit(filters.Limit)
	}

	err := dbQuery.Find(&products).Error
	return products, err
}

func (s *ProductServiceImpl) GetPromotionalProducts(tenantID uuid.UUID) ([]models.Product, error) {
	var products []models.Product
	err := s.db.Where("tenant_id = ? AND stock_quantity > 0 AND sale_price IS NOT NULL AND sale_price != '' AND sale_price != '0'",
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

func (s *ProductServiceImpl) GetAllTenants() ([]models.Tenant, error) {
	var tenants []models.Tenant
	err := s.db.Find(&tenants).Error
	return tenants, err
}

func (s *ProductServiceImpl) GetTenantByID(tenantID uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	err := s.db.Where("id = ?", tenantID).First(&tenant).Error
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (s *ProductServiceImpl) GetProductsByTenantID(tenantID uuid.UUID) ([]models.Product, error) {
	var products []models.Product
	err := s.db.Where("tenant_id = ?", tenantID).Find(&products).Error
	return products, err
}

// CartServiceImpl implementa CartServiceInterface
type CartServiceImpl struct {
	db *gorm.DB
}

func NewCartService(db *gorm.DB) CartServiceInterface {
	return &CartServiceImpl{db: db}
}

func (s *CartServiceImpl) GetOrCreateActiveCart(tenantID, customerID uuid.UUID) (*models.Cart, error) {
	var cart models.Cart
	err := s.db.Where("tenant_id = ? AND customer_id = ? AND status = 'active'",
		tenantID, customerID).First(&cart).Error

	if err == gorm.ErrRecordNotFound {
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

func (s *CartServiceImpl) AddItemToCart(cartID, tenantID uuid.UUID, productID uuid.UUID, quantity int) error {
	var existingItem models.CartItem
	err := s.db.Where("cart_id = ? AND product_id = ?", cartID, productID).First(&existingItem).Error

	if err == gorm.ErrRecordNotFound {
		var product models.Product
		if err := s.db.Where("id = ? AND tenant_id = ?", productID, tenantID).First(&product).Error; err != nil {
			return err
		}

		item := models.CartItem{
			BaseTenantModel: models.BaseTenantModel{
				ID:       uuid.New(),
				TenantID: tenantID,
			},
			CartID:      cartID,
			ProductID:   &productID,
			Quantity:    quantity,
			Price:       getEffectivePrice(&product),
			ProductName: &product.Name,
			ProductSKU:  &product.SKU,
		}
		return s.db.Create(&item).Error
	} else if err != nil {
		return err
	} else {
		existingItem.Quantity += quantity
		return s.db.Save(&existingItem).Error
	}
}

func (s *CartServiceImpl) RemoveItemFromCart(cartID, tenantID, itemID uuid.UUID) error {
	return s.db.Where("cart_id = ? AND id = ?", cartID, itemID).Delete(&models.CartItem{}).Error
}

func (s *CartServiceImpl) ClearCart(cartID, tenantID uuid.UUID) error {
	return s.db.Where("cart_id = ?", cartID).Delete(&models.CartItem{}).Error
}

func (s *CartServiceImpl) UpdateCartItemQuantity(cartID, tenantID, itemID uuid.UUID, quantity int) error {
	return s.db.Model(&models.CartItem{}).
		Where("cart_id = ? AND id = ?", cartID, itemID).
		Update("quantity", quantity).Error
}

func (s *CartServiceImpl) GetCartWithItems(cartID, tenantID uuid.UUID) (*models.Cart, error) {
	var cart models.Cart
	err := s.db.Preload("Items").Preload("Items.Product").
		Where("id = ? AND tenant_id = ?", cartID, tenantID).First(&cart).Error
	return &cart, err
}

// OrderServiceImpl implementa OrderServiceInterface
type OrderServiceImpl struct {
	db *gorm.DB
}

func NewOrderService(db *gorm.DB) OrderServiceInterface {
	return &OrderServiceImpl{db: db}
}

func (s *OrderServiceImpl) CreateOrderFromCart(tenantID, cartID uuid.UUID) (*models.Order, error) {
	return s.CreateOrderFromCartWithAddress(tenantID, cartID, nil)
}

func (s *OrderServiceImpl) CreateOrderFromCartWithAddress(tenantID, cartID uuid.UUID, deliveryAddress *models.Address) (*models.Order, error) {
	return s.CreateOrderFromCartWithConversation(tenantID, cartID, uuid.Nil, deliveryAddress)
}

func (s *OrderServiceImpl) CreateOrderFromCartWithConversation(tenantID, cartID, conversationID uuid.UUID, deliveryAddress *models.Address) (*models.Order, error) {
	// Debug logging
	fmt.Printf("DEBUG CreateOrderFromCart - tenantID: %s, cartID: %s\n", tenantID, cartID)

	// Obter carrinho com itens e cliente
	var cart models.Cart
	err := s.db.Preload("Items").Preload("Items.Product").Preload("Items.Attributes").Preload("Customer").
		Where("id = ? AND tenant_id = ?", cartID, tenantID).First(&cart).Error
	if err != nil {
		fmt.Printf("DEBUG CreateOrderFromCart - Error loading cart: %v\n", err)
		return nil, err
	}

	fmt.Printf("DEBUG CreateOrderFromCart - Cart found with %d items\n", len(cart.Items))
	for i, item := range cart.Items {
		fmt.Printf("DEBUG CreateOrderFromCart - Item %d: ProductID=%s, Quantity=%d, Price=%s\n",
			i+1, item.ProductID, item.Quantity, item.Price)
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

	// Criar pedido com status pendente
	order := models.Order{
		BaseTenantModel: models.BaseTenantModel{
			ID:       uuid.New(),
			TenantID: tenantID,
		},
		CustomerID:        &cart.CustomerID,
		OrderNumber:       generateOrderNumber(),
		Status:            "pending",
		PaymentStatus:     "pending",
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

		fmt.Printf("DEBUG CreateOrderFromCart - Endereço de entrega copiado: %s, %s, %s, %s\n",
			deliveryAddress.Street, deliveryAddress.Number, deliveryAddress.Neighborhood, deliveryAddress.City)
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

	// Criar itens do pedido copiando do carrinho
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

		// Copiar atributos do item do carrinho para o item do pedido
		for _, cartAttr := range cartItem.Attributes {
			orderItemAttr := models.OrderItemAttribute{
				BaseTenantModel: models.BaseTenantModel{
					ID:       uuid.New(),
					TenantID: tenantID,
				},
				OrderItemID:   orderItem.ID,
				AttributeID:   cartAttr.AttributeID,
				OptionID:      cartAttr.OptionID,
				AttributeName: cartAttr.AttributeName,
				OptionName:    cartAttr.OptionName,
				OptionPrice:   cartAttr.OptionPrice,
			}

			err = tx.Create(&orderItemAttr).Error
			if err != nil {
				tx.Rollback()
				return nil, err
			}
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

func (s *OrderServiceImpl) GetPaymentOptions(tenantID uuid.UUID) ([]PaymentOption, error) {
	options := []PaymentOption{
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

// CustomerServiceImpl implementa CustomerServiceInterface
type CustomerServiceImpl struct {
	db *gorm.DB
}

func NewCustomerService(db *gorm.DB) CustomerServiceInterface {
	return &CustomerServiceImpl{db: db}
}

func (s *CustomerServiceImpl) GetCustomerByPhone(tenantID uuid.UUID, phone string) (*models.Customer, error) {
	var customer models.Customer
	err := s.db.Where("tenant_id = ? AND phone = ?", tenantID, phone).First(&customer).Error

	if err == gorm.ErrRecordNotFound {
		// Cliente não existe, criar novo
		customer = models.Customer{
			BaseTenantModel: models.BaseTenantModel{
				ID:       uuid.New(),
				TenantID: tenantID,
			},
			Phone:    phone,
			Name:     "", // Nome será preenchido depois
			IsActive: true,
		}

		err = s.db.Create(&customer).Error
		if err != nil {
			return nil, fmt.Errorf("erro ao criar cliente: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("erro ao buscar cliente: %w", err)
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

func (s *CustomerServiceImpl) UpdateCustomerProfile(tenantID, customerID uuid.UUID, data CustomerUpdateData) error {
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

	if data.Address != "" {
		// Parse the address into components
		parsedAddress := parseAddressText(data.Address)

		// Verificar se já existe um endereço para este cliente
		var existingAddress models.Address
		dbErr := s.db.Where("customer_id = ? AND tenant_id = ?", customerID, tenantID).
			Order("created_at DESC").First(&existingAddress).Error

		if dbErr == gorm.ErrRecordNotFound {
			// Remover padrão de todos os endereços existentes antes de criar novo
			s.db.Model(&models.Address{}).
				Where("customer_id = ? AND tenant_id = ?", customerID, tenantID).
				Update("is_default", false)

			// Criar novo endereço
			address := models.Address{
				BaseTenantModel: models.BaseTenantModel{
					ID:       uuid.New(),
					TenantID: tenantID,
				},
				CustomerID:   customerID,
				Street:       parsedAddress.Street,
				Number:       parsedAddress.Number,
				Complement:   parsedAddress.Complement,
				Neighborhood: parsedAddress.Neighborhood,
				City:         parsedAddress.City,
				State:        parsedAddress.State,
				ZipCode:      parsedAddress.ZipCode,
				Country:      "BR",
				IsDefault:    true,
			}
			return s.db.Create(&address).Error
		} else if dbErr != nil {
			return dbErr
		} else {
			// Remover padrão de todos os outros endereços antes de atualizar
			s.db.Model(&models.Address{}).
				Where("customer_id = ? AND tenant_id = ? AND id != ?", customerID, tenantID, existingAddress.ID).
				Update("is_default", false)

			// Atualizar endereço existente
			addressUpdates := map[string]interface{}{
				"street":       parsedAddress.Street,
				"number":       parsedAddress.Number,
				"complement":   parsedAddress.Complement,
				"neighborhood": parsedAddress.Neighborhood,
				"city":         parsedAddress.City,
				"state":        parsedAddress.State,
				"zipcode":      cleanZipCodeImpl(parsedAddress.ZipCode), // Usar 'zipcode' (nome da coluna no banco) e limpar
				"is_default":   true,
			}

			// Se algum campo ficou vazio após o parsing, manter o valor anterior
			if parsedAddress.Street == "" && existingAddress.Street != "" {
				addressUpdates["street"] = existingAddress.Street
			}
			if parsedAddress.Number == "" && existingAddress.Number != "" {
				addressUpdates["number"] = existingAddress.Number
			}
			if parsedAddress.Neighborhood == "" && existingAddress.Neighborhood != "" {
				addressUpdates["neighborhood"] = existingAddress.Neighborhood
			}
			if parsedAddress.City == "" && existingAddress.City != "" {
				addressUpdates["city"] = existingAddress.City
			}
			if parsedAddress.State == "" && existingAddress.State != "" {
				addressUpdates["state"] = existingAddress.State
			}
			if parsedAddress.ZipCode == "" && existingAddress.ZipCode != "" {
				addressUpdates["zipcode"] = existingAddress.ZipCode // Usar 'zipcode' (nome da coluna no banco)
			}

			return s.db.Model(&existingAddress).Updates(addressUpdates).Error
		}
	}

	return nil
}

// AddressServiceImpl implementa AddressServiceInterface
type AddressServiceImpl struct {
	db *gorm.DB
}

func NewAddressService(db *gorm.DB) AddressServiceInterface {
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

// Funções auxiliares
func generateOrderNumber() string {
	return fmt.Sprintf("PED%d", uuid.New().ID())
}

// Estrutura para armazenar o endereço parseado
type ParsedAddress struct {
	Street       string
	Number       string
	Complement   string
	Neighborhood string
	City         string
	State        string
	ZipCode      string
}

// parseAddressText faz o parsing inteligente do texto do endereço
func parseAddressText(addressText string) ParsedAddress {
	parsed := ParsedAddress{}

	// Normalize the input
	text := strings.TrimSpace(addressText)

	// Regex patterns for address components
	cepPattern := regexp.MustCompile(`\b(\d{5}[-]?\d{3})\b`)
	numberPattern := regexp.MustCompile(`\b(\d+[A-Za-z]?)\b`)
	statePattern := regexp.MustCompile(`\b([A-Z]{2})\b`)

	// Extract CEP/ZIP code - também remove a palavra "CEP" se presente
	if cepMatch := cepPattern.FindStringSubmatch(text); len(cepMatch) > 1 {
		parsed.ZipCode = cleanZipCodeImpl(cepMatch[1])
		text = cepPattern.ReplaceAllString(text, "")
		// Remove também a palavra "CEP" que pode ter ficado isolada
		text = regexp.MustCompile(`\b[Cc][Ee][Pp]\b`).ReplaceAllString(text, "")
	}

	// Extract state (2 letters uppercase)
	if stateMatch := statePattern.FindStringSubmatch(text); len(stateMatch) > 1 {
		parsed.State = stateMatch[1]
		text = statePattern.ReplaceAllString(text, "")
	}

	// Common street indicators
	streetIndicators := []string{"rua", "avenida", "av", "alameda", "travessa", "praça", "estrada", "rodovia"}

	// Common complement indicators
	complementIndicators := []string{"apto", "apt", "apartamento", "casa", "bloco", "bl", "condomínio", "cond", "torre", "andar", "sala", "lote", "qd", "quadra"}

	// Split by common separators
	parts := regexp.MustCompile(`[,;]`).Split(text, -1)

	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		lowerPart := strings.ToLower(part)

		// Skip parts that are just "CEP" or similar
		if lowerPart == "cep" || lowerPart == "cep:" {
			continue
		}

		// Check if this part contains street indicators or looks like street
		isStreetPart := false
		for _, indicator := range streetIndicators {
			if strings.Contains(lowerPart, indicator) {
				isStreetPart = true
				break
			}
		}

		// Check if this part contains complement indicators
		isComplementPart := false
		for _, indicator := range complementIndicators {
			if strings.Contains(lowerPart, indicator) {
				isComplementPart = true
				break
			}
		}

		// Extract number from this part if present
		if numberMatches := numberPattern.FindAllString(part, -1); len(numberMatches) > 0 {
			for _, numMatch := range numberMatches {
				if parsed.Number == "" && len(numMatch) <= 6 { // Reasonable number length
					parsed.Number = numMatch
					part = strings.ReplaceAll(part, numMatch, "")
					part = strings.TrimSpace(part)
				}
			}
		}

		// Assign parts based on position and content
		if isComplementPart {
			// This part contains complement indicators
			if parsed.Complement == "" {
				parsed.Complement = part
			}
		} else if i == 0 || isStreetPart {
			if parsed.Street == "" {
				parsed.Street = part
			}
		} else if i == len(parts)-1 && part != "" && len(part) > 1 {
			// Last part is likely city - but only if it's not empty and meaningful
			if parsed.City == "" {
				parsed.City = part
			}
		} else if strings.Contains(lowerPart, "centro") ||
			strings.Contains(lowerPart, "bairro") ||
			(i > 0 && parsed.Street != "" && parsed.City == "") {
			// Middle parts are likely neighborhood
			if parsed.Neighborhood == "" {
				parsed.Neighborhood = part
			}
		} else if parsed.City == "" && len(part) > 2 {
			// Only assign as city if it's meaningful (more than 2 characters)
			parsed.City = part
		}
	}

	// Fallback: if we have unassigned parts, try to guess
	if parsed.Street == "" && len(parts) > 0 {
		parsed.Street = strings.TrimSpace(parts[0])
	}

	// Default values for Brazilian addresses
	if parsed.State == "" {
		parsed.State = "BR" // Default fallback
	}

	// Tentar identificar cidade de forma mais inteligente
	if parsed.City == "" {
		// Se ainda não temos cidade e temos bairro, deixar vazio para que a IA ou Google Maps tente resolver
		if parsed.City == "" && parsed.Neighborhood != "" {
			// Não assumir que bairro é cidade - deixar para validação posterior
		}

		// Não usar fallback hardcoded - deixar vazio para que o sistema peça para informar
		if parsed.City == "" {
			parsed.City = "" // Deixar vazio para validação posterior
		}
	}

	if parsed.Street == "" {
		parsed.Street = addressText // Use original text as fallback
	}

	return parsed
}

// cleanZipCodeImpl removes all non-numeric characters and limits to 8 digits
func cleanZipCodeImpl(zipcode string) string {
	// Remove todos os caracteres não numéricos
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(zipcode, "")

	// Limita a 8 dígitos (formato brasileiro)
	if len(cleaned) > 8 {
		cleaned = cleaned[:8]
	}

	return cleaned
}

// MessageServiceImpl implementa MessageServiceInterface
type MessageServiceImpl struct {
	db *gorm.DB
}

func NewMessageService(db *gorm.DB) MessageServiceInterface {
	return &MessageServiceImpl{db: db}
}

func (s *MessageServiceImpl) GetMessageByID(messageID string) (*models.Message, error) {
	var message models.Message
	if err := s.db.First(&message, "id = ?", messageID).Error; err != nil {
		return nil, fmt.Errorf("failed to find message: %w", err)
	}
	return &message, nil
}

func (s *MessageServiceImpl) UpdateMessageContent(messageID, content string) error {
	if err := s.db.Model(&models.Message{}).Where("id = ?", messageID).Update("content", content).Error; err != nil {
		return fmt.Errorf("failed to update message content: %w", err)
	}
	return nil
}

func (s *MessageServiceImpl) UpdateMessageContentAndMediaURL(messageID, content, mediaURL string) error {
	updates := map[string]interface{}{
		"content":   content,
		"media_url": mediaURL,
	}
	if err := s.db.Model(&models.Message{}).Where("id = ?", messageID).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update message content and media_url: %w", err)
	}
	return nil
}
