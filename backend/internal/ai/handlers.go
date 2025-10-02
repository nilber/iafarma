package ai

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"iafarma/internal/zapplus"
	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// formatCurrency formats a price string to Brazilian currency format
func formatCurrency(priceStr string) string {
	if priceStr == "" {
		return "0,00"
	}

	// Parse the price as float
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return priceStr // Return original if parsing fails
	}

	// Format to 2 decimal places with Brazilian formatting
	formatted := fmt.Sprintf("%.2f", price)

	// Replace dot with comma for decimal separator
	formatted = strings.ReplaceAll(formatted, ".", ",")

	// Add thousand separators if needed (for values >= 1000)
	parts := strings.Split(formatted, ",")
	integerPart := parts[0]
	decimalPart := parts[1]

	// Add dots for thousands
	if len(integerPart) > 3 {
		var result []rune
		for i, digit := range integerPart {
			if i > 0 && (len(integerPart)-i)%3 == 0 {
				result = append(result, '.')
			}
			result = append(result, digit)
		}
		integerPart = string(result)
	}

	return integerPart + "," + decimalPart
}

// getEffectivePrice returns the sale price if available, otherwise the regular price
func getEffectivePrice(product *models.Product) string {
	if product.SalePrice != "" && product.SalePrice != "0" && product.SalePrice != "0.00" {
		return product.SalePrice
	}
	return product.Price
}

func (s *AIService) handleConsultarItens(tenantID uuid.UUID, customerPhone string, args map[string]interface{}) (string, error) {
	query := ""
	if q, ok := args["query"].(string); ok {
		query = q
	}

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Str("query", query).
		Interface("args", args).
		Msg("🔍 DEBUG: handleConsultarItens called")

	promocional := false
	if p, ok := args["promocional"].(bool); ok {
		promocional = p
	}

	limite := 10
	if l, ok := args["limite"].(float64); ok {
		limite = int(l)
	}

	// Se é uma query genérica para mostrar produtos, não aplicar limite
	isGenericProductQuery := query == "" ||
		query == "produtos" ||
		query == "catálogo" ||
		query == "todos" ||
		query == "mostrar produtos" ||
		query == "ver produtos" ||
		query == "listar produtos" ||
		query == "produtos disponíveis" ||
		query == "cardápio" ||
		query == "menu" ||
		strings.Contains(strings.ToLower(query), "mostrar") && strings.Contains(strings.ToLower(query), "produto") ||
		strings.Contains(strings.ToLower(query), "ver") && strings.Contains(strings.ToLower(query), "produto") ||
		strings.Contains(strings.ToLower(query), "listar") && strings.Contains(strings.ToLower(query), "produto")

	if isGenericProductQuery {
		limite = 0 // Remove o limite para consultas genéricas
		log.Info().
			Bool("is_generic_query", isGenericProductQuery).
			Int("limite", limite).
			Msg("🔍 DEBUG: Generic product query detected")
	}

	// Novos filtros avançados
	marca := ""
	if m, ok := args["marca"].(string); ok {
		marca = m
	}

	tags := ""
	if t, ok := args["tags"].(string); ok {
		tags = t
	}

	var precoMin, precoMax float64
	if pm, ok := args["preco_min"].(float64); ok {
		precoMin = pm
	}
	if pm, ok := args["preco_max"].(float64); ok {
		precoMax = pm
	}

	// Parâmetro de ordenação
	ordenarPor := ""
	if op, ok := args["ordenar_por"].(string); ok {
		ordenarPor = op
	}

	// Conversão para formato interno do SortBy
	var sortBy string
	switch ordenarPor {
	case "preco_menor":
		sortBy = "price_asc"
	case "preco_maior":
		sortBy = "price_desc"
	case "nome":
		sortBy = "name_asc"
	case "relevancia":
		sortBy = "relevance"
	default:
		sortBy = "relevance" // padrão
	}

	var products []models.Product
	var err error

	if promocional {
		products, err = s.productService.GetPromotionalProducts(tenantID)
	} else {
		// 🎯 PRIORIDADE: Se há ordenação por preço, usar busca SQL tradicional para garantir ordenação correta
		if sortBy == "price_asc" || sortBy == "price_desc" {
			log.Info().Msgf("🔍 Price Sort Priority: Using SQL search for price ordering '%s'", sortBy)
			filters := ProductSearchFilters{
				Query:    query,
				Brand:    marca,
				Tags:     tags,
				MinPrice: precoMin,
				MaxPrice: precoMax,
				SortBy:   sortBy,
				Limit:    limite,
			}
			products, err = s.productService.SearchProductsAdvanced(tenantID, filters)
		} else if query != "" && s.embeddingService != nil {
			// 🎯 RAG/EMBEDDING para busca semântica (quando não há ordenação por preço)
			log.Info().Msgf("🔍 RAG Priority: Using semantic search for query='%s'", query)

			ragResults, ragErr := s.embeddingService.SearchSimilarProducts(query, tenantID.String(), limite)
			if ragErr == nil && len(ragResults) > 0 {
				log.Info().Msgf("🔍 RAG Success: Found %d products via semantic search", len(ragResults))

				// Converter ResultSet do RAG para []models.Product
				productIDs := make([]uuid.UUID, 0, len(ragResults))
				for _, result := range ragResults {
					if productID, parseErr := uuid.Parse(result.ID); parseErr == nil {
						productIDs = append(productIDs, productID)
					}
				}

				// Buscar produtos completos usando os IDs encontrados pelo RAG
				if len(productIDs) > 0 {
					ragProducts := make([]models.Product, 0, len(productIDs))
					failedProducts := 0
					for _, productID := range productIDs {
						if product, getErr := s.productService.GetProductByID(tenantID, productID); getErr == nil {
							ragProducts = append(ragProducts, *product)
						} else {
							failedProducts++
							log.Warn().
								Err(getErr).
								Str("product_id", productID.String()).
								Str("tenant_id", tenantID.String()).
								Msg("🚨 RAG Product Not Found: Product ID from RAG doesn't exist in database")
						}
					}

					if failedProducts > 0 {
						log.Warn().
							Int("total_rag_results", len(productIDs)).
							Int("failed_products", failedProducts).
							Int("successful_products", len(ragProducts)).
							Msg("🔍 RAG Sync Issue: Some RAG results not found in database")
					}

					if len(ragProducts) > 0 {
						products = ragProducts
						log.Info().Msgf("🔍 RAG Complete: Successfully retrieved %d products", len(products))
					}
				}
			} else {
				log.Warn().Err(ragErr).Msgf("🔍 RAG Failed: %v, falling back to database search", ragErr)
			}
		}

		// FALLBACK: Se RAG não funcionou ou não retornou resultados, usar busca no banco
		if len(products) == 0 {
			log.Info().Msg("🔍 Database Fallback: Using traditional database search")
			filters := ProductSearchFilters{
				Query:    query,
				Brand:    marca,
				Tags:     tags,
				MinPrice: precoMin,
				MaxPrice: precoMax,
				Limit:    limite,
				SortBy:   sortBy,
			}
			log.Info().
				Interface("filters", filters).
				Str("tenant_id", tenantID.String()).
				Msg("🔍 DEBUG: SearchProductsAdvanced filters")

			products, err = s.productService.SearchProductsAdvanced(tenantID, filters)

			log.Info().
				Int("products_found", len(products)).
				Err(err).
				Msg("🔍 DEBUG: SearchProductsAdvanced result")
		}
	}

	if err != nil {
		return "❌ Erro ao buscar produtos. Tente novamente.", err
	}

	// Se não há filtros específicos (só query vazia ou genérica) e é uma consulta genérica,
	// mostrar catálogo completo organizado por categorias
	if isGenericProductQuery && marca == "" && tags == "" && precoMin == 0 && precoMax == 0 {
		log.Info().
			Int("products_count", len(products)).
			Bool("is_generic_query", isGenericProductQuery).
			Msg("🔍 DEBUG: Calling formatProductsByCategoryComplete")
		return s.formatProductsByCategoryComplete(tenantID, customerPhone, products)
	}

	if len(products) == 0 {
		// Tentar sugestões alternativas baseadas nos produtos do tenant
		suggestions := s.generateDynamicSearchSuggestions(tenantID, query, marca, tags)

		filterDesc := ""
		if query != "" {
			filterDesc += fmt.Sprintf(" com '%s'", query)
		}
		if marca != "" {
			filterDesc += fmt.Sprintf(" da marca '%s'", marca)
		}
		if tags != "" {
			filterDesc += fmt.Sprintf(" na categoria '%s'", tags)
		}
		if precoMin > 0 || precoMax > 0 {
			if precoMin > 0 && precoMax > 0 {
				filterDesc += fmt.Sprintf(" entre R$ %.2f e R$ %.2f", precoMin, precoMax)
			} else if precoMin > 0 {
				filterDesc += fmt.Sprintf(" acima de R$ %.2f", precoMin)
			} else {
				filterDesc += fmt.Sprintf(" abaixo de R$ %.2f", precoMax)
			}
		}

		response := fmt.Sprintf("❌ Nenhum produto encontrado%s.", filterDesc)

		if len(suggestions) > 0 {
			response += "\n\n💡 **Você quis dizer:**\n"
			for _, suggestion := range suggestions {
				response += fmt.Sprintf("• %s\n", suggestion)
			}
			response += "\n📝 Tente um desses termos ou use 'produtos' para ver nosso catálogo completo."
		} else {
			response += "\n\n� **Dica:** Tente termos mais específicos ou use 'produtos' para ver nosso catálogo."
		}

		return response, nil
	}

	// Armazenar produtos na memória com numeração sequencial
	productRefs := s.memoryManager.StoreProductList(tenantID, customerPhone, products)

	result := "🛍️ **Produtos disponíveis:**\n\n"

	// Mostrar filtros aplicados se houver
	if query != "" || marca != "" || tags != "" || precoMin > 0 || precoMax > 0 {
		result += "🔍 **Filtros aplicados:** "
		filters := []string{}
		if query != "" {
			filters = append(filters, fmt.Sprintf("Busca: '%s'", query))
		}
		if marca != "" {
			filters = append(filters, fmt.Sprintf("Marca: %s", marca))
		}
		if tags != "" {
			filters = append(filters, fmt.Sprintf("Categoria: %s", tags))
		}
		if precoMin > 0 && precoMax > 0 {
			filters = append(filters, fmt.Sprintf("Preço: R$ %.2f - R$ %.2f", precoMin, precoMax))
		} else if precoMin > 0 {
			filters = append(filters, fmt.Sprintf("Preço mín: R$ %.2f", precoMin))
		} else if precoMax > 0 {
			filters = append(filters, fmt.Sprintf("Preço máx: R$ %.2f", precoMax))
		}
		result += strings.Join(filters, ", ") + "\n\n"
	}

	for _, productRef := range productRefs {
		if productRef.SequentialID > limite {
			break
		}

		var price string
		if productRef.SalePrice != "" && productRef.SalePrice != "0" {
			price = fmt.Sprintf("~~R$ %s~~ **R$ %s**", formatCurrency(productRef.Price), formatCurrency(productRef.SalePrice))
		} else {
			price = fmt.Sprintf("**R$ %s**", formatCurrency(productRef.Price))
		}

		result += fmt.Sprintf("%d. **%s**\n", productRef.SequentialID, productRef.Name)
		result += fmt.Sprintf("   💰 %s\n", price)
		if productRef.Description != "" {
			desc := productRef.Description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			result += fmt.Sprintf("   📝 %s\n", desc)
		}
		result += "\n"
	}

	result += "💡 Para ver detalhes, diga: 'produto [número]' ou 'produto [nome]'\n"
	result += "🛒 Para adicionar ao carrinho: 'adicionar [número] quantidade [X]'"

	return result, nil
}

func (s *AIService) handleMostrarOpcoesCategoria(tenantID uuid.UUID, customerPhone string, args map[string]interface{}) (string, error) {
	categoria, ok := args["categoria"].(string)
	if !ok {
		return "❌ Categoria é obrigatória.", nil
	}

	limite := 10
	if l, ok := args["limite"].(float64); ok {
		limite = int(l)
	}

	log.Info().
		Str("categoria", categoria).
		Int("limite", limite).
		Msg("🍕 Mostrando opções de categoria")

	// Buscar produtos da categoria
	products, err := s.productService.SearchProducts(tenantID, categoria, limite)
	if err != nil {
		return "❌ Erro ao buscar produtos.", err
	}

	if len(products) == 0 {
		return fmt.Sprintf("❌ Não encontrei produtos para '%s'. Tente outro termo ou veja nosso catálogo completo.", categoria), nil
	}

	// 🔑 CRUCIAL: Armazenar produtos na memória sequencial
	productRefs := s.memoryManager.StoreProductList(tenantID, customerPhone, products)

	// Gerar resposta personalizada baseada na categoria
	titulo := fmt.Sprintf("🛍️ Opções de %s", categoria)

	result := fmt.Sprintf("%s que temos:\n\n", titulo)

	for _, productRef := range productRefs {
		var price string
		if productRef.SalePrice != "" && productRef.SalePrice != "0" {
			price = fmt.Sprintf("~~R$ %s~~ **R$ %s**", formatCurrency(productRef.Price), formatCurrency(productRef.SalePrice))
		} else {
			price = fmt.Sprintf("**R$ %s**", formatCurrency(productRef.Price))
		}

		result += fmt.Sprintf("%d. **%s**\n", productRef.SequentialID, productRef.Name)
		result += fmt.Sprintf("   💰 %s\n", price)

		// Adicionar descrição se disponível
		if productRef.Description != "" {
			desc := productRef.Description
			if len(desc) > 80 {
				desc = desc[:80] + "..."
			}
			result += fmt.Sprintf("   %s\n", desc)
		}
		result += "\n"
	}

	result += "Qual dessas opções você gostaria de pedir?"

	return result, nil
}

func (s *AIService) handleDetalharItem(tenantID uuid.UUID, customerPhone string, args map[string]interface{}) (string, error) {
	identifier, ok := args["identifier"].(string)
	if !ok {
		return "❌ Identificador do produto é obrigatório (número ou nome).", nil
	}

	var product *models.Product
	var err error

	// Tentar converter para número sequencial primeiro
	if sequentialID, parseErr := strconv.Atoi(identifier); parseErr == nil {
		// É um número - buscar na memória
		productRef := s.memoryManager.GetProductBySequentialID(tenantID, customerPhone, sequentialID)
		if productRef != nil {
			product, err = s.productService.GetProductByID(tenantID, productRef.ProductID)
		} else {
			return "❌ Produto não encontrado. Use 'produtos' para ver a lista atualizada.", nil
		}
	} else {
		// Não é número - tentar buscar por nome na memória
		productRef := s.memoryManager.GetProductByName(tenantID, customerPhone, identifier)
		if productRef != nil {
			product, err = s.productService.GetProductByID(tenantID, productRef.ProductID)
		} else {
			// Tentar como UUID se não encontrou na memória
			if productID, uuidErr := uuid.Parse(identifier); uuidErr == nil {
				product, err = s.productService.GetProductByID(tenantID, productID)
			} else {
				return "❌ Produto não encontrado. Use 'produtos' para ver a lista atualizada.", nil
			}
		}
	}

	if err != nil || product == nil {
		return "❌ Produto não encontrado.", err
	}

	result := "🔍 **Detalhes do Produto**\n\n"
	result += fmt.Sprintf("📦 **Nome:** %s\n", product.Name)

	if product.SalePrice != "" && product.SalePrice != "0" {
		result += fmt.Sprintf("💰 **Preço:** ~~R$ %s~~ **R$ %s** (PROMOÇÃO! 🎉)\n", formatCurrency(product.Price), formatCurrency(product.SalePrice))
	} else {
		result += fmt.Sprintf("💰 **Preço:** R$ %s\n", formatCurrency(product.Price))
	}

	if product.Description != "" {
		result += fmt.Sprintf("📝 **Descrição:** %s\n", product.Description)
	}

	if product.SKU != "" {
		result += fmt.Sprintf("🏷️ **SKU:** %s\n", product.SKU)
	}

	if product.StockQuantity > 0 {
		result += fmt.Sprintf("📊 **Estoque:** %d unidades disponíveis\n", product.StockQuantity)
	} else {
		result += "⚠️ **Estoque:** Produto indisponível\n"
	}

	if product.Brand != "" {
		result += fmt.Sprintf("🏭 **Marca:** %s\n", product.Brand)
	}

	if product.Weight != "" {
		result += fmt.Sprintf("⚖️ **Peso:** %s\n", product.Weight)
	}

	result += "\n🛒 Para adicionar ao carrinho, diga: 'adicionar ao carrinho quantidade [X]'"

	return result, nil
}

// addToCartWithFallback tenta múltiplas estratégias para adicionar produto ao carrinho
func (s *AIService) addToCartWithFallback(tenantID, customerID uuid.UUID, customerPhone, identifier string, quantidade int) (string, error) {
	log.Info().
		Str("identifier", identifier).
		Int("quantidade", quantidade).
		Msg("🛒 Tentando adicionar ao carrinho com fallback")

	// Estratégia 1: Tentar por número sequencial (mais comum)
	if sequentialID, parseErr := strconv.Atoi(identifier); parseErr == nil {
		productRef := s.memoryManager.GetProductBySequentialID(tenantID, customerPhone, sequentialID)
		if productRef != nil {
			if result, err := s.tryAddProductToCart(tenantID, customerID, productRef.ProductID, quantidade); err == nil {
				log.Info().Msg("✅ Sucesso: Produto adicionado por número sequencial")
				return result, nil
			}
		}
		log.Info().Msg("❌ Falha: Produto não encontrado por número sequencial")
	}

	// Estratégia 2: Tentar por UUID
	if productID, uuidErr := uuid.Parse(identifier); uuidErr == nil {
		if result, err := s.tryAddProductToCart(tenantID, customerID, productID, quantidade); err == nil {
			log.Info().Msg("✅ Sucesso: Produto adicionado por UUID")
			return result, nil
		}
		log.Info().Msg("❌ Falha: Produto não encontrado por UUID")
	}

	// Estratégia 3: Tentar por nome (busca fuzzy)
	if result, err := s.tryAddProductByName(tenantID, customerID, customerPhone, identifier, quantidade); err == nil {
		log.Info().Msg("✅ Sucesso: Produto adicionado por nome")
		return result, nil
	}
	log.Info().Msg("❌ Falha: Produto não encontrado por nome")

	// Estratégia 4: Tentar buscar no contexto da conversa recente
	if result, err := s.tryAddFromRecentContext(tenantID, customerID, customerPhone, identifier, quantidade); err == nil {
		log.Info().Msg("✅ Sucesso: Produto adicionado do contexto recente")
		return result, nil
	}
	log.Info().Msg("❌ Falha: Produto não encontrado no contexto recente")

	// Se todas as estratégias falharam, retornar mensagem amigável
	return "❌ Não consegui identificar esse produto. 🔍\n\n💡 **Dicas:**\n• Use 'produtos' para ver nossa lista\n• Use 'buscar [nome]' para encontrar um item específico\n• Ou digite o nome do produto que você quer", nil
}

// tryAddProductToCart tenta adicionar um produto específico ao carrinho
func (s *AIService) tryAddProductToCart(tenantID, customerID, productID uuid.UUID, quantidade int) (string, error) {
	product, err := s.productService.GetProductByID(tenantID, productID)
	if err != nil || product == nil {
		return "", fmt.Errorf("produto não encontrado")
	}

	if product.StockQuantity < quantidade {
		return fmt.Sprintf("❌ Estoque insuficiente para **%s**. Disponível: %d unidades.", product.Name, product.StockQuantity), nil
	}

	cart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
	if err != nil {
		return "", fmt.Errorf("erro ao acessar carrinho")
	}

	err = s.cartService.AddItemToCart(cart.ID, tenantID, product.ID, quantidade)
	if err != nil {
		return "", fmt.Errorf("erro ao adicionar item ao carrinho")
	}

	adicional := "\n\nVocê pode continuar comprando ou digite 'finalizar' para fechar o pedido."

	return fmt.Sprintf("✅ **%s** adicionado ao carrinho!\n🔢 Quantidade: %d\n💰 Valor: R$ %s",
		product.Name, quantidade, formatCurrency(getEffectivePrice(product))) + adicional, nil
}

// tryAddProductByName tenta adicionar produto pelo nome
func (s *AIService) tryAddProductByName(tenantID, customerID uuid.UUID, customerPhone, nomeProduto string, quantidade int) (string, error) {
	products, err := s.productService.SearchProducts(tenantID, nomeProduto, 5)
	if err != nil || len(products) == 0 {
		return "", fmt.Errorf("nenhum produto encontrado")
	}

	// Se encontrou exatamente 1 produto, adicionar
	if len(products) == 1 {
		return s.tryAddProductToCart(tenantID, customerID, products[0].ID, quantidade)
	}

	// Se encontrou múltiplos, verificar se há match exato
	nomeLower := strings.ToLower(nomeProduto)
	for _, product := range products {
		if strings.ToLower(product.Name) == nomeLower {
			return s.tryAddProductToCart(tenantID, customerID, product.ID, quantidade)
		}
	}

	return "", fmt.Errorf("múltiplos produtos encontrados")
}

// tryAddFromRecentContext tenta encontrar produto no contexto da conversa recente
func (s *AIService) tryAddFromRecentContext(tenantID, customerID uuid.UUID, customerPhone, identifier string, quantidade int) (string, error) {
	// Buscar nas mensagens recentes por produtos mencionados
	conversationHistory := s.memoryManager.GetConversationHistory(tenantID, customerPhone)
	if len(conversationHistory) == 0 {
		return "", fmt.Errorf("sem contexto recente")
	}

	// Procurar nas últimas 10 mensagens
	searchLimit := 10
	if len(conversationHistory) < searchLimit {
		searchLimit = len(conversationHistory)
	}

	identifierLower := strings.ToLower(identifier)

	// Reverter a busca (mensagens mais recentes primeiro)
	for i := len(conversationHistory) - 1; i >= len(conversationHistory)-searchLimit && i >= 0; i-- {
		message := conversationHistory[i]
		if message.Role == "assistant" {
			messageLower := strings.ToLower(message.Content)

			// Procurar por números seguidos de produtos mencionados
			if strings.Contains(messageLower, identifierLower) {
				// Tentar extrair produtos mencionados na mensagem
				if products, err := s.extractProductsFromMessage(tenantID, message.Content); err == nil && len(products) > 0 {
					// Se o identificador é um número, usar como índice
					if idx, parseErr := strconv.Atoi(identifier); parseErr == nil && idx > 0 && idx <= len(products) {
						return s.tryAddProductToCart(tenantID, customerID, products[idx-1].ID, quantidade)
					}
				}
			}
		}
	}

	return "", fmt.Errorf("produto não encontrado no contexto")
}

// extractProductsFromMessage extrai produtos mencionados em uma mensagem
func (s *AIService) extractProductsFromMessage(tenantID uuid.UUID, message string) ([]*models.Product, error) {
	var products []*models.Product

	// Buscar por padrões como "1. Nome do Produto" ou "1⁠. Nome:"
	// Usar character class simples para evitar problemas com Unicode
	re := regexp.MustCompile(`^\d+[\.\s]+\s*([^:]+):`)

	lines := strings.Split(message, "\n")
	for _, line := range lines {
		// Padrão: número seguido de ponto e nome
		if re.MatchString(line) {
			matches := re.FindStringSubmatch(line)
			if len(matches) > 1 {
				productName := strings.TrimSpace(matches[1])
				if foundProducts, err := s.productService.SearchProducts(tenantID, productName, 1); err == nil && len(foundProducts) > 0 {
					products = append(products, &foundProducts[0])
				}
			}
		}
	}

	return products, nil
}

func (s *AIService) handleAdicionarAoCarrinho(tenantID, customerID uuid.UUID, customerPhone string, args map[string]interface{}) (string, error) {
	identifier, ok := args["identifier"].(string)
	if !ok {
		return "❌ Identificador do produto é obrigatório (número ou ID).", nil
	}

	quantidadeFloat, ok := args["quantidade"].(float64)
	if !ok {
		return "❌ Quantidade é obrigatória.", nil
	}
	quantidade := int(quantidadeFloat)

	if quantidade <= 0 {
		return "❌ Quantidade deve ser maior que zero.", nil
	}

	// Usar o sistema de fallback
	return s.addToCartWithFallback(tenantID, customerID, customerPhone, identifier, quantidade)
}

func (s *AIService) handleAdicionarProdutoPorNome(tenantID, customerID uuid.UUID, customerPhone string, args map[string]interface{}) (string, error) {
	nomeProduto, ok := args["nome_produto"].(string)
	if !ok {
		return "❌ Nome do produto é obrigatório.", nil
	}

	quantidadeFloat, ok := args["quantidade"].(float64)
	if !ok {
		return "❌ Quantidade é obrigatória.", nil
	}
	quantidade := int(quantidadeFloat)

	if quantidade <= 0 {
		return "❌ Quantidade deve ser maior que zero.", nil
	}

	// Buscar produtos pelo nome com busca mais flexível
	products, err := s.productService.SearchProducts(tenantID, nomeProduto, 10)
	if err != nil {
		return "❌ Erro ao buscar produtos.", err
	}

	if len(products) == 0 {
		// return fmt.Sprintf("❌ Nenhum produto encontrado com '%s'. \n\n💡 **Dica:** Tente termos mais específicos ou use 'produtos' para ver nosso catálogo.", nomeProduto), nil
		return fmt.Sprintf("🔍 *Ops! Não encontramos '%s'*\n\n💡 Que tal tentar:\n• Termos mais curtos ou específicos\n• Digitar 'produtos' para ver tudo o que temos\n• Procurar por categoria\n\nEstamos juntos nessa! 💪", nomeProduto), nil
	}

	// Se encontrou apenas um produto, adicionar diretamente
	if len(products) == 1 {
		product := &products[0]

		if product.StockQuantity < quantidade {
			return fmt.Sprintf("❌ Estoque insuficiente para **%s**. Disponível: %d unidades.", product.Name, product.StockQuantity), nil
		}

		// Obter ou criar carrinho ativo
		cart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
		if err != nil {
			return "❌ Erro ao acessar carrinho.", err
		}

		// Adicionar item ao carrinho
		err = s.cartService.AddItemToCart(cart.ID, tenantID, product.ID, quantidade)
		if err != nil {
			return "❌ Erro ao adicionar item ao carrinho.", err
		}

		adicional := "\n\nVocê pode continuar comprando ou digite 'finalizar' para fechar o pedido."
		return fmt.Sprintf("✅ **%s** adicionado ao carrinho!\n🔢 Quantidade: %d\n💰 Valor: R$ %s",
			product.Name, quantidade, formatCurrency(getEffectivePrice(product))) + adicional, nil
	}

	// Se encontrou múltiplos produtos, mostrar opções
	productRefs := s.memoryManager.StoreProductList(tenantID, customerPhone, products)

	result := fmt.Sprintf("🔍 Encontrei %d produtos similares a '%s':\n\n", len(products), nomeProduto)
	for _, productRef := range productRefs {
		// Usar nosso padrão de formatação
		var priceStr string
		if productRef.SalePrice != "" && productRef.SalePrice != "0" {
			priceStr = fmt.Sprintf("~~R$ %s~~ **R$ %s**", formatCurrency(productRef.Price), formatCurrency(productRef.SalePrice))
		} else {
			priceStr = fmt.Sprintf("**R$ %s**", formatCurrency(productRef.Price))
		}

		result += fmt.Sprintf("%d. **%s**\n   💰 %s\n", productRef.SequentialID, productRef.Name, priceStr)
	}

	result += "\n📝 Para adicionar ao carrinho basta informar o numero do item ou nome do produto.\n"
	// Don't add instruction message here - let AI handle it contextually
	return result, nil
}

func (s *AIService) handleRemoverDoCarrinho(tenantID, customerID uuid.UUID, args map[string]interface{}) (string, error) {
	// Try to get item number first (new approach)
	if itemNumberFloat, ok := args["item_number"].(float64); ok {
		itemNumber := int(itemNumberFloat)

		cartItem, err := s.getCartItemByNumber(tenantID, customerID, itemNumber)
		if err != nil {
			return "❌ Item não encontrado no carrinho. Use 'ver carrinho' para conferir os números.", nil
		}

		// Get cart
		cart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
		if err != nil {
			return "❌ Erro ao acessar carrinho.", err
		}

		// Remove item by ID
		err = s.cartService.RemoveItemFromCart(cart.ID, tenantID, cartItem.ID)
		if err != nil {
			return "❌ Erro ao remover item do carrinho.", err
		}

		return fmt.Sprintf("✅ **%s** removido do carrinho com sucesso!", getItemName(*cartItem)), nil
	}

	// Fallback to old ID-based approach for compatibility
	itemIDStr, ok := args["item_id"].(string)
	if !ok {
		return "❌ Número do item é obrigatório.", nil
	}

	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		return "❌ Número do item inválido.", nil
	}

	// Obter carrinho ativo
	cart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao acessar carrinho.", err
	}

	// Remover item do carrinho
	err = s.cartService.RemoveItemFromCart(cart.ID, tenantID, itemID)
	if err != nil {
		return "❌ Erro ao remover item do carrinho.", err
	}

	return "✅ Item removido do carrinho com sucesso!", nil
}

// handleVerCarrinhoWithOptions permite controlar se mostra as instruções de gerenciamento
func (s *AIService) handleVerCarrinhoWithOptions(tenantID, customerID uuid.UUID, showManagementInstructions bool) (string, error) {
	// Obter carrinho com itens
	cart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao acessar carrinho.", err
	}

	cartWithItems, err := s.cartService.GetCartWithItems(cart.ID, tenantID)
	if err != nil {
		return "❌ Erro ao carregar itens do carrinho.", err
	}

	if len(cartWithItems.Items) == 0 {
		return "🛒 Seu carrinho está vazio!\n\n💡 Explore nossos produtos e adicione alguns itens.", nil
	}

	result := "🛒 **Seu Carrinho:**\n\n"
	total := 0.0

	for i, item := range cartWithItems.Items {
		itemPrice, _ := strconv.ParseFloat(item.Price, 64)
		itemTotal := itemPrice * float64(item.Quantity)
		total += itemTotal

		result += fmt.Sprintf("%d. **%s**\n", i+1, getItemName(item))
		result += fmt.Sprintf("   💰 R$ %s x %d = R$ %s\n\n", formatCurrency(item.Price), item.Quantity, formatCurrency(fmt.Sprintf("%.2f", itemTotal)))
	}

	result += fmt.Sprintf("💳 **Total: R$ %s**", formatCurrency(fmt.Sprintf("%.2f", total)))

	// Adicionar instruções de gerenciamento apenas quando solicitado
	if showManagementInstructions {
		result += "\n\n🛍️ Quando quiser finalizar, é só avisar!\n"
		result += "📝 Para alterar quantidade: 'alterar item [número] para [quantidade]'\n"
		result += "🗑️ Para remover um item: 'remover item [número]'\n"
		result += "🧹 Para limpar carrinho: 'limpar carrinho'"
	}

	return result, nil
}

func (s *AIService) handleLimparCarrinho(tenantID, customerID uuid.UUID) (string, error) {
	// Obter carrinho ativo
	cart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao acessar carrinho.", err
	}

	// Limpar carrinho
	err = s.cartService.ClearCart(cart.ID, tenantID)
	if err != nil {
		return "❌ Erro ao limpar carrinho.", err
	}

	return "🧹 ✅ Carrinho limpo com sucesso!\n\n🛍️ Agora você pode adicionar novos produtos.", nil
}

func (s *AIService) handleAtualizarQuantidade(tenantID, customerID uuid.UUID, args map[string]interface{}) (string, error) {
	quantidadeFloat, ok := args["quantidade"].(float64)
	if !ok {
		return "❌ Quantidade é obrigatória.", nil
	}
	quantidade := int(quantidadeFloat)

	if quantidade <= 0 {
		return "❌ Quantidade deve ser maior que zero.", nil
	}

	// Try to get item number first (new approach)
	if itemNumberFloat, ok := args["item_number"].(float64); ok {
		itemNumber := int(itemNumberFloat)

		cartItem, err := s.getCartItemByNumber(tenantID, customerID, itemNumber)
		if err != nil {
			return "❌ Item não encontrado no carrinho. Use 'ver carrinho' para conferir os números.", nil
		}

		// Get cart
		cart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
		if err != nil {
			return "❌ Erro ao acessar carrinho.", err
		}

		// Update quantity
		err = s.cartService.UpdateCartItemQuantity(cart.ID, tenantID, cartItem.ID, quantidade)
		if err != nil {
			return "❌ Erro ao atualizar quantidade do item.", err
		}

		return fmt.Sprintf("✅ **%s** - quantidade atualizada para %d unidades!", getItemName(*cartItem), quantidade), nil
	}

	// Fallback to old ID-based approach for compatibility
	itemIDStr, ok := args["item_id"].(string)
	if !ok {
		return "❌ Número do item é obrigatório.", nil
	}

	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		return "❌ Número do item inválido.", nil
	}

	// Obter carrinho ativo
	cart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao acessar carrinho.", err
	}

	// Atualizar quantidade do item
	err = s.cartService.UpdateCartItemQuantity(cart.ID, tenantID, itemID, quantidade)
	if err != nil {
		return "❌ Erro ao atualizar quantidade do item.", err
	}

	return fmt.Sprintf("✅ Quantidade atualizada para %d unidades!\n\n🛒 Use 'ver carrinho' para conferir.", quantidade), nil
}

func (s *AIService) performFinalCheckout(tenantID, customerID uuid.UUID, customerPhone string) (string, error) {
	// Obter carrinho
	cart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao acessar carrinho.", err
	}

	cartWithItems, err := s.cartService.GetCartWithItems(cart.ID, tenantID)
	if err != nil {
		return "❌ Erro ao carregar carrinho.", err
	}

	if len(cartWithItems.Items) == 0 {
		return "❌ Carrinho vazio! Adicione alguns produtos antes de finalizar.", nil
	}

	// 🚚 VALIDAR SE FAZEMOS ENTREGA NO ENDEREÇO DO CLIENTE ANTES DE CRIAR O PEDIDO
	addresses, err := s.addressService.GetAddressesByCustomer(tenantID, customerID)
	if err != nil || len(addresses) == 0 {
		return "❌ Nenhum endereço encontrado. Por favor, cadastre um endereço de entrega.\n\n🏠 **Para cadastrar, informe seu endereço completo:**\n\n💡 **Exemplo:** Rua das Flores, 123, Centro, Brasília, DF, CEP 70000-000, Complemento (se houver)", err
	}

	// Encontrar o endereço padrão ou usar o primeiro
	var deliveryAddress *models.Address
	for _, addr := range addresses {
		if addr.IsDefault {
			deliveryAddress = &addr
			break
		}
	}
	if deliveryAddress == nil {
		deliveryAddress = &addresses[0]
	}

	// Validar se fazemos entrega neste endereço
	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_id", customerID.String()).
		Str("address", fmt.Sprintf("%s, %s, %s, %s, %s", deliveryAddress.Street, deliveryAddress.Number, deliveryAddress.Neighborhood, deliveryAddress.City, deliveryAddress.State)).
		Msg("🚚 Validando entrega antes do checkout final")

	deliveryResult, err := s.deliveryService.ValidateDeliveryAddress(
		tenantID,
		deliveryAddress.Street,
		deliveryAddress.Number,
		deliveryAddress.Neighborhood,
		deliveryAddress.City,
		deliveryAddress.State,
	)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao validar endereço de entrega")
		return "❌ Erro ao validar endereço de entrega. Tente novamente ou entre em contato conosco.", err
	}

	// Se não fazemos entrega neste endereço, oferecer opção de cadastrar novo
	if !deliveryResult.CanDeliver {
		addressText := formatAddressForDisplay(*deliveryAddress)

		var reason string
		switch deliveryResult.Reason {
		case "area_not_served":
			reason = "Esta região não está em nossa área de atendimento."
		case "outside_radius":
			reason = "Este endereço está fora da nossa área de entrega."
		case "no_store_location":
			reason = "Nossa loja ainda não tem localização configurada."
		default:
			reason = "Não conseguimos atender este endereço no momento."
		}

		return fmt.Sprintf("🚫 **Não fazemos entrega neste endereço:**\n\n📍 **Endereço atual:**\n%s\n\n⚠️ **Motivo:** %s\n\n🏠 **Opções:**\n1️⃣ **Cadastrar novo endereço:** Informe um endereço completo onde fazemos entrega\n2️⃣ **Gerenciar endereços:** Digite 'meus endereços' para ver/alterar\n3️⃣ **Verificar área:** Digite 'fazem entrega em [local]?' para verificar outras regiões\n\n💡 **Para continuar, informe um novo endereço de entrega.**",
			addressText, reason), nil
	}

	// Endereço validado - prosseguir com a criação do pedido
	log.Info().
		Str("delivery_reason", deliveryResult.Reason).
		Str("distance", deliveryResult.Distance).
		Bool("can_deliver", deliveryResult.CanDeliver).
		Msg("✅ Endereço de entrega validado com sucesso")

	// Buscar conversation ID armazenado para esta sessão
	conversationID := s.getConversationID(tenantID, customerPhone)

	// Criar pedido com status pendente incluindo o endereço de entrega e conversation ID
	var order *models.Order

	if conversationID != uuid.Nil {
		log.Info().Str("conversation_id", conversationID.String()).Msg("🔗 Criando pedido com conversation ID")
		order, err = s.orderService.CreateOrderFromCartWithConversation(tenantID, cart.ID, conversationID, deliveryAddress)
	} else {
		log.Warn().Msg("⚠️ Nenhum conversation ID encontrado, criando pedido sem conversation ID")
		order, err = s.orderService.CreateOrderFromCartWithAddress(tenantID, cart.ID, deliveryAddress)
	}
	if err != nil {
		log.Error().Err(err).Msg("Erro ao criar pedido no checkout final")
		return "❌ Erro ao criar pedido.", err
	}

	log.Info().
		Str("order_number", order.OrderNumber).
		Str("total", order.TotalAmount).
		Str("delivery_address", fmt.Sprintf("%s, %s, %s, %s", deliveryAddress.Street, deliveryAddress.Number, deliveryAddress.Neighborhood, deliveryAddress.City)).
		Msg("Pedido criado com sucesso no checkout final")

	// 🧹 LIMPEZA COMPLETA APÓS PEDIDO CRIADO
	s.cleanupAfterOrderCreation(tenantID, customerID, customerPhone, cart.ID)

	// Send alert notification if configured
	if s.alertService != nil {
		if err := s.alertService.SendOrderAlert(tenantID, order, customerPhone); err != nil {
			log.Error().Err(err).Msg("Erro ao enviar alerta do pedido")
		}
	}

	return fmt.Sprintf("🎉 **Pedido registrado com sucesso!**\n\n📋 **Número do Pedido:** %s\n💰 **Total:** R$ %s\n📦 **Status:** Pendente\n\n✅ **Seu pedido foi registrado em nosso sistema!**\n\n👥 Um de nossos operadores irá revisar e confirmar seu pedido em breve.\n📞 Você será contatado para confirmar os detalhes da entrega e pagamento.\n\n🔍 Acompanhe seu pedido pelo número: **%s**",
		order.OrderNumber,
		formatCurrency(order.TotalAmount),
		order.OrderNumber), nil
}

// cleanupAfterOrderCreation limpa carrinho, memória e dados do RAG após pedido criado
func (s *AIService) cleanupAfterOrderCreation(tenantID, customerID uuid.UUID, customerPhone string, cartID uuid.UUID) {
	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_id", customerID.String()).
		Str("customer_phone", customerPhone).
		Msg("🧹 Iniciando limpeza completa após criação do pedido")

	// 1. Limpar carrinho (já processado, mas garantir que está limpo)
	if err := s.cartService.ClearCart(cartID, tenantID); err != nil {
		log.Warn().Err(err).Msg("❌ Falha ao limpar carrinho após pedido")
	} else {
		log.Info().Msg("✅ Carrinho limpo")
	}

	// 2. Limpar memória da conversa atual (produtos listados, contexto temporário)
	s.memoryManager.ClearMemory(tenantID, customerPhone)
	log.Info().Msg("✅ Memória da conversa limpa")

	// 3. Não limpar o RAG (histórico de conversas) - manter para contexto futuro
	// O RAG deve ser mantido para aprender com as interações do cliente

	log.Info().Msg("🧹 ✅ Limpeza completa finalizada - sistema pronto para novo pedido")
}

func (s *AIService) handleCheckout(tenantID, customerID uuid.UUID, customerPhone string) (string, error) {
	// PRIMEIRO: Sempre mostrar o carrinho para o cliente conferir (sem instruções de gerenciamento)
	cartMessage, err := s.handleVerCarrinhoWithOptions(tenantID, customerID, false)
	if err != nil {
		return "❌ Erro ao verificar carrinho.", err
	}

	// Se carrinho está vazio, não prosseguir
	if strings.Contains(cartMessage, "carrinho está vazio") {
		return "❌ Carrinho vazio! Adicione alguns produtos antes de finalizar.", nil
	}

	// 🚨 CORREÇÃO: SEMPRE retornar o resumo do carrinho primeiro
	// O cliente deve ver todos os itens antes de confirmar o pedido

	// Verificar se cliente tem dados necessários
	customer, err := s.customerService.GetCustomerByID(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao verificar dados do cliente.", err
	}

	if customer.Name == "" {
		return fmt.Sprintf("%s\n\n📝 Para seguir com seu pedido, precisamos completar o seu cadastro.\n\n🙋‍♂️ **Por favor, me informe seu nome completo:**", cartMessage), nil
	}

	// 💳 VALIDAÇÃO OBRIGATÓRIA: Verificar se forma de pagamento foi selecionada (ANTES do endereço)
	// Primeiro obter o carrinho ativo
	activeCart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao verificar carrinho.", err
	}

	// Agora buscar com os itens
	cart, err := s.cartService.GetCartWithItems(activeCart.ID, tenantID)
	if err != nil {
		return "❌ Erro ao verificar carrinho.", err
	}

	if cart.PaymentMethodID == nil {
		// Buscar formas de pagamento disponíveis
		paymentOptions, err := s.orderService.GetPaymentOptions(tenantID)
		if err != nil || len(paymentOptions) == 0 {
			// Se não há formas de pagamento cadastradas, continuar sem bloquear
			log.Warn().Str("tenant_id", tenantID.String()).Msg("Nenhuma forma de pagamento cadastrada para o tenant")
		} else {
			// Mostrar opções de pagamento e bloquear até escolher
			result := fmt.Sprintf("%s\n\n💳 **Escolha a forma de pagamento:**\n\n", cartMessage)
			for i, option := range paymentOptions {
				result += fmt.Sprintf("%d. %s\n", i+1, option.Name)
			}
			result += "\n💬 **Como você quer pagar?** Me diga o número ou nome da forma de pagamento.\n"
			result += "\n💡 **Exemplo:** 'quero pagar com PIX' ou 'número 1'"

			return result, nil
		}
	}

	// Verificar se tem endereços
	addresses, err := s.addressService.GetAddressesByCustomer(tenantID, customerID)
	if err != nil || len(addresses) == 0 {
		return fmt.Sprintf("%s\n\n📝 Para finalizar o pedido, precisamos do seu endereço de entrega.\n\n🏠 **Por favor, me informe seu endereço completo:**\n\n💡 **Exemplo:** Rua das Flores, 123, Centro, Brasília, DF, CEP 70000-000, Complemento (se houver)", cartMessage), nil
	}

	// Se tem endereços, verificar se há múltiplos endereços
	if len(addresses) > 1 {
		// Verificar se já há um endereço padrão definido
		var defaultAddress *models.Address
		for _, addr := range addresses {
			if addr.IsDefault {
				defaultAddress = &addr
				break
			}
		}

		// Se já há um endereço padrão, mostrar para confirmação antes de finalizar
		if defaultAddress != nil {
			addressText := formatAddressForDisplay(*defaultAddress)
			return fmt.Sprintf("%s\n\n📦 **Confirme o endereço de entrega:**\n\n📍 **Endereço padrão:**\n%s\n\n✅ **Este endereço está correto para a entrega?**\n\n💬 Responda:\n🟢 **'sim'** ou **'confirmar'** - para finalizar o pedido\n🔄 **'não'** ou **'alterar'** - para escolher outro endereço\n📝 **'editar endereço'** - para modificar este endereço", cartMessage, addressText), nil
		}

		// Se não há endereço padrão, mostrar lista para seleção
		addressesText := formatAddressesForSelection(addresses)
		return fmt.Sprintf("%s\n\n%s\n\n✅ **Qual endereço deseja usar para esta entrega?**\n\n💬 Responda com o número do endereço ou 'confirmar' para usar o padrão.", cartMessage, addressesText), nil
	}

	// Se tem apenas um endereço, mostrar para confirmação
	if len(addresses) == 1 {
		defaultAddress := addresses[0]

		// Verificar se o endereço está completo
		if !isAddressComplete(defaultAddress) {
			addressText := formatAddressForDisplay(defaultAddress)
			return fmt.Sprintf("%s\n\n📋 **Endereço cadastrado:**\n%s\n\n⚠️ **Endereço incompleto!** Algumas informações estão faltando.\n\n🏠 **Por favor, informe seu endereço completo:**\n\n💡 **Exemplo:** Rua das Flores, 123, Centro, Brasília, DF, CEP 70000-000, Complemento (se houver)", cartMessage, addressText), nil
		}

		// 🚨 CORREÇÃO: Mostrar carrinho junto com o endereço para confirmação antes de finalizar
		addressText := formatAddressForDisplay(defaultAddress)
		return fmt.Sprintf("%s\n\n📦 **Confirme o endereço de entrega:**\n\n📍 **Endereço cadastrado:**\n%s\n\n✅ **Este endereço está correto para a entrega?**\n\n💬 Responda:\n🟢 **'sim'** ou **'confirmar'** - para finalizar o pedido\n🔄 **'não'** ou **'alterar'** - para cadastrar outro endereço\n📝 **'editar endereço'** - para modificar este endereço", cartMessage, addressText), nil
	}

	return "❌ Erro inesperado no checkout.", nil
}

func (s *AIService) handleCancelarPedido(tenantID uuid.UUID, args map[string]interface{}) (string, error) {
	orderIDStr, ok := args["order_id"].(string)
	if !ok {
		return "❌ ID do pedido é obrigatório.", nil
	}

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("order_id_str", orderIDStr).
		Msg("🔍 Tentando cancelar pedido")

	// Tentar encontrar o pedido por diferentes métodos
	var orderID uuid.UUID
	var err error

	// Primeiro tentar como UUID direto (compatibilidade com código antigo)
	orderID, err = uuid.Parse(orderIDStr)
	if err != nil {
		log.Info().
			Str("order_id_str", orderIDStr).
			Msg("🔍 Não é UUID, tentando buscar na memória")

		// Se não for UUID, tentar encontrar na memória por número sequencial
		customerID, exists := args["customer_id"]
		if exists {
			customerIDStr, ok := customerID.(string)
			if ok {
				log.Info().
					Str("customer_id", customerIDStr).
					Msg("🔍 Buscando na memória do cliente")

				// Buscar na memória os pedidos armazenados
				ordersListData, found := s.memoryManager.GetTempData(tenantID, customerIDStr, "orders_list")
				if found {
					log.Info().Msg("🔍 Dados encontrados na memória")
					ordersList, ok := ordersListData.([]map[string]interface{})
					if ok {
						log.Info().
							Int("orders_count", len(ordersList)).
							Msg("🔍 Lista de pedidos recuperada")

						// Tentar encontrar por número sequencial
						sequentialNum := orderIDStr
						for _, orderData := range ordersList {
							if sequential, seqOk := orderData["sequential"].(int); seqOk {
								if fmt.Sprintf("%d", sequential) == sequentialNum {
									if idStr, idOk := orderData["id"].(string); idOk {
										if parsedUUID, parseUUIDErr := uuid.Parse(idStr); parseUUIDErr == nil {
											orderID = parsedUUID
											log.Info().
												Str("found_uuid", orderID.String()).
												Msg("✅ Pedido encontrado por número sequencial")
											goto foundOrder
										}
									}
								}
							}
						}

						// Tentar encontrar por código do pedido
						for _, orderData := range ordersList {
							if orderNumber, numOk := orderData["number"].(string); numOk {
								if orderNumber == orderIDStr {
									if idStr, idOk := orderData["id"].(string); idOk {
										if parsedUUID, parseUUIDErr := uuid.Parse(idStr); parseUUIDErr == nil {
											orderID = parsedUUID
											log.Info().
												Str("found_uuid", orderID.String()).
												Msg("✅ Pedido encontrado por código")
											goto foundOrder
										}
									}
								}
							}
						}
					}
				} else {
					log.Warn().Msg("❌ Dados não encontrados na memória")
				}
			}
		}

		// Se chegou até aqui, não encontrou o pedido
		log.Warn().
			Str("order_id_str", orderIDStr).
			Msg("❌ Pedido não encontrado em nenhum método")
		return "❌ Pedido não encontrado. Use 'histórico de pedidos' primeiro e depois 'cancelar pedido [número]'.", nil
	}

foundOrder:
	log.Info().
		Str("order_uuid", orderID.String()).
		Msg("🔄 Tentando cancelar pedido")

	err = s.orderService.CancelOrder(tenantID, orderID)
	if err != nil {
		log.Error().
			Err(err).
			Str("order_uuid", orderID.String()).
			Msg("❌ Erro ao cancelar pedido")
		return "❌ Erro ao cancelar pedido.", err
	}

	log.Info().
		Str("order_uuid", orderID.String()).
		Msg("✅ Pedido cancelado com sucesso")

	return "✅ Pedido cancelado com sucesso!\n\n🛍️ Você pode fazer um novo pedido quando quiser.", nil
}

func (s *AIService) handleHistoricoPedidos(tenantID, customerID uuid.UUID) (string, error) {
	orders, err := s.orderService.GetOrdersByCustomer(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao buscar histórico de pedidos.", err
	}

	if len(orders) == 0 {
		return "📝 Você ainda não tem pedidos.\n\n🛍️ Que tal fazer seu primeiro pedido?", nil
	}

	result := "📋 **Histórico de Pedidos:**\n\n"

	// Preparar dados para memória sequencial
	memoryData := make(map[string]interface{})
	ordersList := make([]map[string]interface{}, 0)

	for i, order := range orders {
		if i >= 10 { // Limitar a 10 pedidos mais recentes
			break
		}

		statusEmoji := getStatusEmoji(order.Status)
		result += fmt.Sprintf("%d. **Pedido %s** %s\n", i+1, order.OrderNumber, statusEmoji)
		result += fmt.Sprintf("   💰 Total: R$ %s\n", formatCurrency(order.TotalAmount))
		result += fmt.Sprintf("   📅 Data: %s\n", order.CreatedAt.Format("02/01/2006"))

		// Adicionar status sempre
		statusText := "Pendente"
		switch order.Status {
		case "pending":
			statusText = "Pendente"
		case "confirmed":
			statusText = "Confirmado"
		case "processing":
			statusText = "Processando"
		case "shipped":
			statusText = "Enviado"
		case "delivered":
			statusText = "Entregue"
		case "cancelled":
			statusText = "Cancelado"
		default:
			statusText = order.Status
		}
		result += fmt.Sprintf("   📦 Status: %s\n", statusText)

		// Buscar quantidade de itens do pedido se disponível
		// Por agora, adicionaremos informação genérica
		result += "   📋 Itens: Ver detalhes do pedido\n"

		// Adicionar informação de cancelamento apenas para pedidos pendentes
		if order.Status == "pending" {
			result += "   ⚠️ Pode ser cancelado\n"
		}
		result += "\n"

		// Adicionar à lista para memória sequencial
		orderData := map[string]interface{}{
			"id":         order.ID.String(),
			"number":     order.OrderNumber,
			"sequential": i + 1,
			"status":     order.Status,
			"total":      formatCurrency(order.TotalAmount),
			"created_at": order.CreatedAt.Format("02/01/2006"),
		}
		ordersList = append(ordersList, orderData)
	}

	// Armazenar na memória para permitir cancelamento por número
	memoryData["orders_list"] = ordersList
	memoryData["context"] = "historic_orders"
	s.memoryManager.StoreTempData(tenantID, customerID.String(), memoryData)

	result += "💡 Para cancelar um pedido pendente, use: 'cancelar pedido [número]' ou 'cancelar pedido [código]'"

	return result, nil
}

// handleSelecionarFormaPagamento registra a forma de pagamento escolhida pelo cliente
func (s *AIService) handleSelecionarFormaPagamento(tenantID, customerID uuid.UUID, args map[string]interface{}) (string, error) {
	var paymentMethodID uuid.UUID
	var paymentMethodName string

	// Tentar obter ID ou nome do método de pagamento
	if paymentMethodIDStr, ok := args["payment_method_id"].(string); ok && paymentMethodIDStr != "" {
		// Tentar como UUID primeiro
		if id, err := uuid.Parse(paymentMethodIDStr); err == nil {
			paymentMethodID = id
		} else {
			// Se não for UUID, tratar como nome
			paymentMethodName = paymentMethodIDStr
		}
	}

	// Se foi passado nome, buscar ID pelo nome
	if paymentMethodName != "" || (paymentMethodID == uuid.Nil) {
		if paymentMethodName == "" {
			paymentMethodName, _ = args["payment_method_name"].(string)
		}

		// Buscar forma de pagamento pelo nome
		paymentOptions, err := s.orderService.GetPaymentOptions(tenantID)
		if err != nil {
			return "❌ Erro ao buscar formas de pagamento.", err
		}

		// Buscar por nome (case insensitive)
		found := false
		searchName := strings.ToLower(strings.TrimSpace(paymentMethodName))
		for _, option := range paymentOptions {
			if strings.ToLower(option.Name) == searchName || strings.Contains(strings.ToLower(option.Name), searchName) {
				paymentMethodID, _ = uuid.Parse(option.ID)
				paymentMethodName = option.Name
				found = true
				break
			}
		}

		if !found {
			return fmt.Sprintf("❌ Forma de pagamento '%s' não encontrada. Formas disponíveis: %v",
				paymentMethodName,
				func() []string {
					names := make([]string, len(paymentOptions))
					for i, opt := range paymentOptions {
						names[i] = opt.Name
					}
					return names
				}()), nil
		}
	}

	if paymentMethodID == uuid.Nil {
		return "❌ Erro: Forma de pagamento não informada.", fmt.Errorf("payment_method not provided")
	}

	// Buscar o carrinho ativo do cliente
	activeCart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao acessar carrinho.", err
	}

	cart, err := s.cartService.GetCartWithItems(activeCart.ID, tenantID)
	if err != nil {
		return "❌ Você ainda não tem produtos no carrinho. Adicione produtos antes de selecionar o pagamento.", err
	}

	// Atualizar o método de pagamento no carrinho
	err = s.cartService.UpdateCartPaymentMethod(cart.ID, tenantID, paymentMethodID)
	if err != nil {
		return "❌ Erro ao registrar a forma de pagamento. Tente novamente.", err
	}

	// Verificar se precisa de troco
	needsChange, _ := args["needs_change"].(bool)
	changeForAmount, _ := args["change_for_amount"].(string)
	observations, _ := args["observations"].(string)

	// Se precisa de troco, atualizar observações
	if needsChange && changeForAmount != "" {
		err = s.cartService.UpdateCartObservations(cart.ID, tenantID, observations, changeForAmount)
		if err != nil {
			log.Warn().Err(err).Msg("Erro ao salvar observações de troco")
		}
	}

	// Buscar informações do método de pagamento para confirmar (se ainda não temos o nome)
	if paymentMethodName == "" {
		paymentOptions2, err := s.orderService.GetPaymentOptions(tenantID)
		if err != nil {
			return "✅ Forma de pagamento registrada com sucesso!", nil
		}

		for _, option := range paymentOptions2 {
			if option.ID == paymentMethodID.String() {
				paymentMethodName = option.Name
				break
			}
		}
	}

	if paymentMethodName == "" {
		paymentMethodName = "Forma de pagamento selecionada"
	}

	result := fmt.Sprintf("✅ **Forma de pagamento registrada:**\n\n💳 %s\n", paymentMethodName)

	if needsChange && changeForAmount != "" {
		result += fmt.Sprintf("\n💵 Troco para: R$ %s\n", changeForAmount)
	}

	if observations != "" {
		result += fmt.Sprintf("\n📝 Observação: %s\n", observations)
	}

	result += "\n✨ Agora você pode finalizar seu pedido! Digite 'finalizar pedido' ou 'checkout' quando estiver pronto."

	return result, nil
}

// handleTrocarFormaPagamento permite alterar a forma de pagamento já selecionada
func (s *AIService) handleTrocarFormaPagamento(tenantID, customerID uuid.UUID, args map[string]interface{}) (string, error) {
	// Reutiliza a mesma lógica de seleção
	return s.handleSelecionarFormaPagamento(tenantID, customerID, args)
}

func (s *AIService) handleAtualizarCadastro(tenantID, customerID uuid.UUID, customerPhone string, args map[string]interface{}) (string, error) {
	var updates CustomerUpdateData
	var updatedFields []string

	log.Info().
		Interface("args", args).
		Msg("🔍 DEBUG: handleAtualizarCadastro received args")

	if nome, ok := args["nome"].(string); ok && nome != "" {
		updates.Name = nome
		updatedFields = append(updatedFields, "nome")
		log.Info().
			Str("nome", nome).
			Msg("🏷️ DEBUG: Name parameter received")
	}

	if email, ok := args["email"].(string); ok && email != "" {
		updates.Email = email
		updatedFields = append(updatedFields, "email")
	}

	if endereco, ok := args["endereco"].(string); ok && endereco != "" {
		// Verificar se é uma confirmação de endereço
		lowerEndereco := strings.ToLower(strings.TrimSpace(endereco))

		// Detectar solicitação de novo endereço
		if strings.Contains(lowerEndereco, "novo") && strings.Contains(lowerEndereco, "endereço") ||
			strings.Contains(lowerEndereco, "novo") && strings.Contains(lowerEndereco, "endereco") ||
			lowerEndereco == "novo endereço" || lowerEndereco == "novo endereco" {
			return "🏠 **Para cadastrar um novo endereço, informe o endereço completo:**\n\n💡 **Exemplo:** Rua das Flores, 123, Centro, Brasília, DF, CEP 70000-000, Complemento (se houver)\n\n📍 **Ou use 'gerenciar endereços' para ver opções.**", nil
		}

		// Detectar confirmações simples
		if lowerEndereco == "sim" || lowerEndereco == "confirmo" || lowerEndereco == "ok" || lowerEndereco == "confirmar" || lowerEndereco == "confirma" ||
			lowerEndereco == "finalizar" || lowerEndereco == "fechar" || lowerEndereco == "concluir" || lowerEndereco == "prosseguir" || lowerEndereco == "continuar" {
			// Cliente confirmou o endereço existente, prosseguir com checkout final
			response := "✅ **Endereço confirmado!**\n\n🎯 Prosseguindo com o pedido...\n\n"

			checkoutResult, checkoutErr := s.performFinalCheckout(tenantID, customerID, customerPhone)
			if checkoutErr == nil {
				response += checkoutResult
				return response, nil
			} else {
				response += "❌ Erro ao finalizar pedido: " + checkoutErr.Error()
				return response, checkoutErr
			}
		}

		// Detectar seleção de endereço por número (ex: "usar endereço 2", "endereço 1", "2")
		// Mas não deve capturar endereços completos com rua, número, etc.
		// Verifica se é um endereço completo primeiro
		isCompleteAddress := strings.Contains(lowerEndereco, ",") ||
			strings.Contains(lowerEndereco, "rua") ||
			strings.Contains(lowerEndereco, "avenida") ||
			strings.Contains(lowerEndereco, " av ") ||
			strings.Contains(lowerEndereco, "alameda") ||
			strings.Contains(lowerEndereco, "cep") ||
			len(strings.Fields(endereco)) > 3 // Endereço completo tem muitas palavras

		if !isCompleteAddress {
			addressNumberPattern := regexp.MustCompile(`(?i)^(?:usar\s+)?(?:endere[çc]o\s+)?(\d+)$`)
			if matches := addressNumberPattern.FindStringSubmatch(endereco); len(matches) > 1 {
				addressNum, err := strconv.Atoi(matches[1])
				if err == nil && addressNum > 0 {
					// Buscar endereços do cliente
					addresses, err := s.addressService.GetAddressesByCustomer(tenantID, customerID)
					if err != nil {
						return "❌ Erro ao buscar endereços.", err
					}

					if addressNum <= len(addresses) {
						selectedAddress := addresses[addressNum-1]

						// Definir este endereço como padrão
						err := s.addressService.SetDefaultAddress(tenantID, customerID, selectedAddress.ID)
						if err != nil {
							return "❌ Erro ao definir endereço padrão.", err
						}

						response := fmt.Sprintf("✅ **Endereço %d selecionado como padrão!**\n\n📋 **Endereço de entrega:**\n%s\n\n🎯 Prosseguindo com o pedido...\n\n",
							addressNum, formatAddressForDisplay(selectedAddress))

						checkoutResult, checkoutErr := s.performFinalCheckout(tenantID, customerID, customerPhone)
						if checkoutErr == nil {
							response += checkoutResult
							return response, nil
						} else {
							response += "❌ Erro ao finalizar pedido: " + checkoutErr.Error()
							return response, checkoutErr
						}
					} else {
						return fmt.Sprintf("❌ Endereço %d não encontrado. Você tem apenas %d endereços cadastrados.", addressNum, len(addresses)), nil
					}
				}
			}
		}

		// Detectar comando para apagar endereço
		if strings.Contains(lowerEndereco, "apagar") || strings.Contains(lowerEndereco, "deletar") || strings.Contains(lowerEndereco, "delete") || strings.Contains(lowerEndereco, "remover") {
			// Detectar se quer apagar todos
			if strings.Contains(lowerEndereco, "todos") || strings.Contains(lowerEndereco, "tudo") {
				err := s.addressService.DeleteAllAddresses(tenantID, customerID)
				if err != nil {
					return "❌ Erro ao deletar endereços.", err
				}
				return "✅ **Todos os endereços foram deletados!**\n\n🏠 **Para adicionar um novo endereço, informe:**\n� **Exemplo:** Rua das Flores, 123, Centro, Brasília, DF, CEP 70000-000", nil
			}

			// Detectar número específico para deletar
			deleteNumberPattern := regexp.MustCompile(`(?i)(?:apagar|deletar|delete|remover)\s+(?:endere[çc]o\s+)?(\d+)`)
			if matches := deleteNumberPattern.FindStringSubmatch(endereco); len(matches) > 1 {
				addressNum, err := strconv.Atoi(matches[1])
				if err == nil && addressNum > 0 {
					addresses, err := s.addressService.GetAddressesByCustomer(tenantID, customerID)
					if err != nil {
						return "❌ Erro ao buscar endereços.", err
					}

					if addressNum <= len(addresses) {
						addressToDelete := addresses[addressNum-1]
						err := s.addressService.DeleteAddress(tenantID, customerID, addressToDelete.ID)
						if err != nil {
							return "❌ Erro ao deletar endereço.", err
						}
						return fmt.Sprintf("✅ **Endereço %d deletado com sucesso!**\n\n🗑️ **Endereço removido:**\n%s",
							addressNum, formatAddressForDisplay(addressToDelete)), nil
					} else {
						return fmt.Sprintf("❌ Endereço %d não encontrado. Você tem apenas %d endereços cadastrados.", addressNum, len(addresses)), nil
					}
				}
			}

			return "🗑️ **Para remover um endereço específico:** 'deletar endereço 1', 'apagar endereço 2'\n🗑️ **Para remover todos:** 'deletar todos'\n💡 **Ou informe um novo endereço completo.**", nil
		}

		// Detectar "novo endereço" - solicita o endereço
		if strings.Contains(lowerEndereco, "novo") && strings.Contains(lowerEndereco, "endere") {
			return "🏠 **Por favor, informe seu novo endereço completo:**\n\n💡 **Exemplo:** Rua das Flores, 123, Centro, Brasília, DF, CEP 70000-000, Complemento (se houver)", nil
		}

		// É um novo endereço, usar IA para extrair campos
		log.Info().
			Str("address_text", endereco).
			Msg("🧠 Processing address with AI parsing")

		// Usar IA para extrair campos do endereço
		ctx := context.Background()
		parsedAddress, err := s.parseAddressWithAI(ctx, endereco)
		if err != nil {
			log.Error().
				Err(err).
				Str("address_text", endereco).
				Msg("❌ Failed to parse address with AI, using fallback")

			// Fallback para parsing simples se IA falhar
			updates.Address = endereco
			updatedFields = append(updatedFields, "endereço")
		} else {
			// Sucesso com IA - criar endereço diretamente
			log.Info().
				Interface("parsed_address", parsedAddress).
				Msg("✅ Address parsed successfully with AI")

			// Remover padrão de todos os endereços existentes
			existingAddresses, err := s.addressService.GetAddressesByCustomer(tenantID, customerID)
			if err == nil && len(existingAddresses) > 0 {
				// Remove o padrão de todos os endereços existentes
				for _, existingAddr := range existingAddresses {
					if existingAddr.IsDefault {
						s.addressService.SetDefaultAddress(tenantID, customerID, uuid.Nil) // Remove default
						break
					}
				}
			}

			// 🏙️ VALIDAÇÃO DE CIDADE DESABILITADA TEMPORARIAMENTE
			// A validação será implementada depois para evitar dependências circulares
			if parsedAddress.City == "" {
				// Se não tem cidade, é obrigatório informar
				return "❌ **Cidade obrigatória!**\n\n🏙️ Por favor, informe a cidade no seu endereço.\n\n📝 Exemplo: 'Avenida Hugo Musso, 1333, Praia da Costa, Vila Velha, ES'", nil
			}

			// Criar novo endereço com campos estruturados da IA
			address := &models.Address{
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

			err = s.addressService.CreateAddress(tenantID, address)
			if err != nil {
				log.Error().
					Err(err).
					Interface("address", address).
					Msg("❌ Failed to create address from AI parsing")
				return "❌ Erro ao salvar endereço.", err
			}

			updatedFields = append(updatedFields, "endereço")
			log.Info().
				Str("address_id", address.ID.String()).
				Msg("✅ Address created successfully from AI parsing")
		}
	}

	if len(updatedFields) == 0 {
		return "❌ Nenhum dado válido para atualizar.", nil
	}

	// Buscar dados do cliente para personalização (ANTES da atualização para comparação)
	customer, err := s.customerService.GetCustomerByID(tenantID, customerID)
	if err != nil {
		// Se não conseguir buscar, usar mensagem padrão
		customer = nil
	}

	// Só atualizar perfil do cliente se houver campos que não sejam endereço processado pela IA
	if updates.Name != "" || updates.Email != "" || updates.Address != "" {
		err := s.customerService.UpdateCustomerProfile(tenantID, customerID, updates)
		if err != nil {
			return "❌ Erro ao atualizar cadastro.", err
		}
	}

	// Preparar primeiro nome do cliente para personalização
	customerName := ""

	// Se temos um nome novo nos updates, usar esse
	if updates.Name != "" {
		names := strings.Fields(strings.TrimSpace(updates.Name))
		if len(names) > 0 {
			customerName = names[0]
		} else {
			customerName = "Cliente"
		}
		log.Info().
			Str("updates_name", updates.Name).
			Str("customer_name", customerName).
			Msg("🏷️ Using name from updates")
	} else if customer != nil && customer.Name != "" {
		// Se não há nome novo, usar o nome existente
		names := strings.Fields(strings.TrimSpace(customer.Name))
		if len(names) > 0 {
			customerName = names[0]
		} else {
			customerName = "Cliente"
		}
		log.Info().
			Str("existing_name", customer.Name).
			Str("customer_name", customerName).
			Msg("🏷️ Using existing customer name")
	} else {
		customerName = "Cliente"
		log.Info().
			Msg("🏷️ Using default 'Cliente' name")
	}

	// Tentar finalizar pedido automaticamente se dados estão completos
	if len(updatedFields) > 0 {
		// Tentar finalizar pedido diretamente
		checkoutResult, checkoutErr := s.performFinalCheckout(tenantID, customerID, customerPhone)
		if checkoutErr == nil {
			// Se finalizou com sucesso, mostrar mensagem de sucesso personalizada
			response := fmt.Sprintf("✅ **%s**, seu cadastro foi atualizado com sucesso!\n\n", customerName)
			response += checkoutResult
			return response, nil
		} else {
			// Se ainda precisa de mais dados, criar mensagem personalizada
			response := fmt.Sprintf("✅ **%s**, seu cadastro foi atualizado com sucesso!\n\n", customerName)

			// Verificar se precisa de endereço especificamente
			if strings.Contains(checkoutErr.Error(), "endereço") ||
				strings.Contains(checkoutErr.Error(), "address") ||
				strings.Contains(checkoutErr.Error(), "entrega") {
				response += "📝 Para finalizar o pedido, precisamos do seu endereço de entrega.\n\n"
				response += "🏠 Por favor, me informe seu endereço completo:\n\n"
				response += "💡 **Exemplo:** Rua das Flores, 123, Centro, Brasília, DF, CEP 70000-000, Complemento (se houver)\n\n"
				return response, nil
			}

			// Se for outro tipo de erro, mostrar o carrinho
			cartResult, cartErr := s.handleCheckout(tenantID, customerID, customerPhone)
			if cartErr == nil {
				response += cartResult
			} else {
				response += "💡 Agora você pode tentar finalizar seu pedido novamente!"
			}
			return response, nil
		}
	}

	// Se chegou aqui, não houve campos para atualizar ou erro inesperado
	return fmt.Sprintf("✅ **%s**, seu cadastro foi atualizado com sucesso!", customerName), nil
}

func (s *AIService) handleGerenciarEnderecos(tenantID, customerID uuid.UUID, args map[string]interface{}) (string, error) {
	acao, ok := args["acao"].(string)
	if !ok {
		return "❌ Ação não especificada.", nil
	}

	addresses, err := s.addressService.GetAddressesByCustomer(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao buscar endereços.", err
	}

	switch acao {
	case "listar":
		if len(addresses) == 0 {
			return "📍 **Nenhum endereço cadastrado.**\n\n🏠 **Para adicionar um endereço, informe:**\n\n💡 **Exemplo:** Rua das Flores, 123, Centro, Brasília, DF, CEP 70000-000", nil
		}

		addressesText := formatAddressesForSelection(addresses)
		return fmt.Sprintf("%s\n\n💡 **Para usar um endereço específico, diga:** 'usar endereço 2' ou 'endereço 1'\n🏠 **Para adicionar novo endereço, apenas informe o endereço completo.**\n🗑️ **Para deletar:** 'deletar endereço 2' ou 'deletar todos'\n\n**Exemplo de endereço completo:** Rua das Flores, 123, Centro, Brasília, DF, CEP 70000-000, Apto 101", addressesText), nil

	case "selecionar":
		numeroEndereco, ok := args["numero_endereco"].(float64)
		if !ok {
			return "❌ Número do endereço não especificado.", nil
		}

		addressNum := int(numeroEndereco)
		if addressNum < 1 || addressNum > len(addresses) {
			return fmt.Sprintf("❌ Endereço %d não encontrado. Você tem apenas %d endereços cadastrados.", addressNum, len(addresses)), nil
		}

		selectedAddress := addresses[addressNum-1]

		// Definir este endereço como padrão
		err := s.addressService.SetDefaultAddress(tenantID, customerID, selectedAddress.ID)
		if err != nil {
			return "❌ Erro ao definir endereço padrão.", err
		}

		return fmt.Sprintf("✅ **Endereço %d selecionado como padrão!**\n\n📋 **Endereço de entrega:**\n%s\n\n🛒 **Agora você pode finalizar seu pedido.**",
			addressNum, formatAddressForDisplay(selectedAddress)), nil

	case "deletar":
		if len(addresses) == 0 {
			return "📍 **Nenhum endereço para deletar.**", nil
		}

		numeroEndereco, ok := args["numero_endereco"].(float64)
		if !ok {
			return "❌ Número do endereço não especificado.", nil
		}

		addressNum := int(numeroEndereco)
		if addressNum < 1 || addressNum > len(addresses) {
			return fmt.Sprintf("❌ Endereço %d não encontrado. Você tem apenas %d endereços cadastrados.", addressNum, len(addresses)), nil
		}

		addressToDelete := addresses[addressNum-1]
		err := s.addressService.DeleteAddress(tenantID, customerID, addressToDelete.ID)
		if err != nil {
			return "❌ Erro ao deletar endereço.", err
		}

		return fmt.Sprintf("✅ **Endereço %d deletado com sucesso!**\n\n🗑️ **Endereço removido:**\n%s",
			addressNum, formatAddressForDisplay(addressToDelete)), nil

	case "deletar_todos":
		if len(addresses) == 0 {
			return "📍 **Nenhum endereço para deletar.**", nil
		}

		err := s.addressService.DeleteAllAddresses(tenantID, customerID)
		if err != nil {
			return "❌ Erro ao deletar endereços.", err
		}

		return fmt.Sprintf("✅ **Todos os %d endereços foram deletados!**\n\n🏠 **Para adicionar um novo endereço, informe:**\n💡 **Exemplo:** Rua das Flores, 123, Centro, Brasília, DF, CEP 70000-000", len(addresses)), nil

	default:
		return "❌ Ação não reconhecida. Use 'listar', 'selecionar', 'deletar' ou 'deletar_todos'.", nil
	}
}

func (s *AIService) handleCadastrarEndereco(tenantID, customerID uuid.UUID, args map[string]interface{}) (string, error) {
	// Parse endereço completo se fornecido
	enderecoCompleto, hasCompleto := args["endereco_completo"].(string)

	// Criar o endereço usando os campos individuais ou parseando o endereço completo
	address := &models.Address{
		CustomerID: customerID,
		Country:    "BR", // Default
	}

	// Se foi fornecido endereço completo, tenta parsear
	if hasCompleto && strings.TrimSpace(enderecoCompleto) != "" {
		// Parse do endereço completo
		parts := strings.Split(enderecoCompleto, ",")
		if len(parts) >= 4 {
			// Formato esperado: "Rua, Número, Bairro, Cidade, Estado, CEP"
			address.Street = strings.TrimSpace(parts[0])
			address.Number = strings.TrimSpace(parts[1])
			address.Neighborhood = strings.TrimSpace(parts[2])
			cityName := strings.TrimSpace(parts[3])

			// Inicialmente aceita a cidade digitada
			address.City = cityName

			if len(parts) >= 5 {
				address.State = strings.TrimSpace(parts[4])

				// Validar se a cidade existe no banco de dados
				if s.municipioService != nil {
					exists, municipio, err := s.municipioService.ValidarCidade(cityName, address.State)
					if err == nil && exists && municipio != nil {
						// Usar o nome oficial da cidade do banco
						address.City = municipio.NomeCidade
					}
					// Se não encontrou, mantém o que o usuário digitou
				}
			}
			if len(parts) >= 6 {
				cep := strings.TrimSpace(parts[5])
				// Remove "CEP" prefix if present
				cep = strings.ReplaceAll(cep, "CEP", "")
				cep = strings.ReplaceAll(cep, ":", "")
				cep = strings.TrimSpace(cep)
				// Limpa o CEP deixando apenas números
				address.ZipCode = cleanZipCode(cep)
			}
			// Se há mais partes, assume que é complemento
			if len(parts) >= 7 {
				complement := strings.TrimSpace(parts[6])
				address.Complement = complement
			}
		} else {
			return "❌ **Formato de endereço inválido.**\n\n💡 **Use o formato:** Rua, Número, Bairro, Cidade, Estado, CEP\n\n📋 **Exemplo:** Av Hugo Musso, 2380, Itapua, Vila Velha, ES, 29101789", nil
		}
	} else {
		// Usar campos individuais
		if rua, ok := args["rua"].(string); ok {
			address.Street = rua
		}
		if numero, ok := args["numero"].(string); ok {
			address.Number = numero
		}
		if bairro, ok := args["bairro"].(string); ok {
			address.Neighborhood = bairro
		}
		if cidade, ok := args["cidade"].(string); ok {
			address.City = cidade
		}
		if estado, ok := args["estado"].(string); ok {
			address.State = estado
		}
		if cep, ok := args["cep"].(string); ok {
			address.ZipCode = cleanZipCode(cep)
		}
	}

	// Validações básicas
	if address.Street == "" {
		return "❌ **Rua é obrigatória.**\n\n💡 **Informe o endereço completo:** Rua, Número, Bairro, Cidade, Estado, CEP, Complemento (se houver)", nil
	}
	if address.City == "" {
		return "❌ **Cidade é obrigatória.**\n\n💡 **Informe o endereço completo:** Rua, Número, Bairro, Cidade, Estado, CEP, Complemento (se houver)", nil
	}
	if address.State == "" {
		return "❌ **Estado é obrigatório.**\n\n💡 **Informe o endereço completo:** Rua, Número, Bairro, Cidade, Estado, CEP, Complemento (se houver)", nil
	}
	if address.ZipCode == "" {
		return "❌ **CEP é obrigatório.**\n\n💡 **Informe o endereço completo:** Rua, Número, Bairro, Cidade, Estado, CEP, Complemento (se houver)", nil
	}

	// Verificar se o cliente já tem endereços
	existingAddresses, err := s.addressService.GetAddressesByCustomer(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao verificar endereços existentes.", err
	}

	// Se é o primeiro endereço, marcar como padrão
	if len(existingAddresses) == 0 {
		address.IsDefault = true
	}

	// Criar o endereço
	err = s.addressService.CreateAddress(tenantID, address)
	if err != nil {
		return "❌ Erro ao cadastrar endereço.", err
	}

	// Buscar endereços atualizados para mostrar posição
	updatedAddresses, _ := s.addressService.GetAddressesByCustomer(tenantID, customerID)
	addressPosition := len(updatedAddresses)

	defaultText := ""
	if address.IsDefault {
		defaultText = " (padrão)"
	}

	return fmt.Sprintf("✅ **Endereço cadastrado com sucesso!**%s\n\n📍 **Endereço %d:**\n%s\n\n🛒 **Agora você pode finalizar seu pedido ou gerenciar seus endereços.**",
		defaultText, addressPosition, formatAddressForDisplay(*address)), nil
}

// Funções auxiliares
func cleanZipCode(zipcode string) string {
	// Remove todos os caracteres não numéricos
	cleaned := regexp.MustCompile(`\D`).ReplaceAllString(zipcode, "")

	// Limita a 8 dígitos (formato brasileiro)
	if len(cleaned) > 8 {
		cleaned = cleaned[:8]
	}

	return cleaned
}

func getItemName(item models.CartItem) string {
	if item.ProductName != nil && *item.ProductName != "" {
		return *item.ProductName
	}
	if item.Product != nil {
		return item.Product.Name
	}
	return "Produto"
}

func getStatusEmoji(status string) string {
	switch status {
	case "pending":
		return "⏳"
	case "processing":
		return "⚙️"
	case "shipped":
		return "🚚"
	case "delivered":
		return "✅"
	case "cancelled":
		return "❌"
	default:
		return "📦"
	}
}

func (s *AIService) handleBuscarMultiplosProdutos(tenantID, customerID uuid.UUID, customerPhone string, args map[string]interface{}) (string, error) {
	produtosInterface, ok := args["produtos"].([]interface{})
	if !ok {
		return "❌ Lista de produtos é obrigatória.", nil
	}

	// Convert interface slice to string slice
	var produtos []string
	for _, p := range produtosInterface {
		if produto, ok := p.(string); ok {
			produtos = append(produtos, produto)
		}
	}

	if len(produtos) == 0 {
		return "❌ Pelo menos um produto deve ser especificado.", nil
	}

	quantidadeFloat := 1.0
	if q, ok := args["quantidade"].(float64); ok {
		quantidadeFloat = q
	}
	quantidade := int(quantidadeFloat)

	if quantidade <= 0 {
		return "❌ Quantidade deve ser maior que zero.", nil
	}

	// Clear existing product list to start fresh
	s.memoryManager.StoreProductList(tenantID, customerPhone, []models.Product{})

	result := fmt.Sprintf("🔍 Busquei %d produtos para você:\n\n", len(produtos))
	totalFound := 0

	for i, nomeProduto := range produtos {
		// Search for each product
		searchResults, err := s.productService.SearchProducts(tenantID, nomeProduto, 5)
		if err != nil {
			result += fmt.Sprintf("%d. ❌ Erro ao buscar '%s'\n\n", i+1, nomeProduto)
			continue
		}

		if len(searchResults) == 0 {
			result += fmt.Sprintf("%d. ❌ Nenhum produto encontrado para '%s'\n\n", i+1, nomeProduto)
			continue
		}

		// Add products to memory with sequential numbering
		productRefs := s.memoryManager.AppendProductList(tenantID, customerPhone, searchResults)

		// Fix grammar: use "opção" for 1, "opções" for multiple
		opcaoOuOpcoes := "opção"
		if len(searchResults) > 1 {
			opcaoOuOpcoes = "opções"
		}

		result += fmt.Sprintf("%d. 🔍 **%s** (%d %s):\n", i+1, nomeProduto, len(searchResults), opcaoOuOpcoes)
		for _, ref := range productRefs {
			var price string
			if ref.SalePrice != "" && ref.SalePrice != "0" {
				price = fmt.Sprintf("~~R$ %s~~ **R$ %s**", formatCurrency(ref.Price), formatCurrency(ref.SalePrice))
			} else {
				price = fmt.Sprintf("**R$ %s**", formatCurrency(ref.Price))
			}
			result += fmt.Sprintf("   %d. **%s**\n", ref.SequentialID, ref.Name)
			result += fmt.Sprintf("      💰 %s\n", price)
		}
		result += "\n"
		totalFound += len(searchResults)
	}

	if totalFound > 0 {
		result += fmt.Sprintf("🛒 Para adicionar, diga: 'adicionar [número] quantidade %d'", quantidade)
	} else {
		result += "💡 Tente usar termos mais específicos ou consulte nosso catálogo com 'produtos'"
	}

	return result, nil
}

func (s *AIService) handleAdicionarMaisItemCarrinho(tenantID, customerID uuid.UUID, args map[string]interface{}) (string, error) {
	produtoNome, ok := args["produto_nome"].(string)
	if !ok {
		return "❌ Nome do produto é obrigatório.", nil
	}

	quantidadeFloat, ok := args["quantidade_adicional"].(float64)
	if !ok {
		return "❌ Quantidade adicional é obrigatória.", nil
	}
	quantidadeAdicional := int(quantidadeFloat)

	if quantidadeAdicional <= 0 {
		return "❌ Quantidade deve ser maior que zero.", nil
	}

	// Get current cart items
	cart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao acessar carrinho.", err
	}

	cartWithItems, err := s.cartService.GetCartWithItems(cart.ID, tenantID)
	if err != nil {
		return "❌ Erro ao carregar itens do carrinho.", err
	}

	if len(cartWithItems.Items) == 0 {
		return "🛒 Seu carrinho está vazio. Use 'produtos' para ver nosso catálogo.", nil
	}

	// Find matching item in cart (case insensitive partial match)
	var foundItem *models.CartItem

	produtoNomeLower := strings.ToLower(produtoNome)
	for _, item := range cartWithItems.Items {
		itemNameLower := strings.ToLower(getItemName(item))
		if strings.Contains(itemNameLower, produtoNomeLower) {
			foundItem = &item
			break
		}
	}

	if foundItem == nil {
		return fmt.Sprintf("❌ Não encontrei '%s' no seu carrinho.\n\n🛒 Use 'ver carrinho' para conferir os itens.", produtoNome), nil
	}

	// Update quantity (current + additional)
	novaQuantidade := foundItem.Quantity + quantidadeAdicional

	err = s.cartService.UpdateCartItemQuantity(cart.ID, tenantID, foundItem.ID, novaQuantidade)
	if err != nil {
		return "❌ Erro ao atualizar quantidade do item.", err
	}

	return fmt.Sprintf("✅ **%s** - adicionadas mais %d unidades!\n📦 Nova quantidade: %d unidades",
		getItemName(*foundItem), quantidadeAdicional, novaQuantidade), nil
}

func (s *AIService) handleAdicionarPorNumero(tenantID, customerID uuid.UUID, customerPhone string, args map[string]interface{}) (string, error) {
	numeroFloat, ok := args["numero"].(float64)
	if !ok {
		return "❌ Número do produto é obrigatório.", nil
	}
	numero := int(numeroFloat)

	quantidadeFloat := 1.0
	if q, ok := args["quantidade"].(float64); ok {
		quantidadeFloat = q
	}
	quantidade := int(quantidadeFloat)

	if quantidade <= 0 {
		return "❌ Quantidade deve ser maior que zero.", nil
	}

	// Get product from memory by sequential ID
	productRef := s.memoryManager.GetProductBySequentialID(tenantID, customerPhone, numero)
	if productRef == nil {
		return "❌ Produto não encontrado. Use 'produtos' para ver a lista atualizada.", nil
	}

	// Get full product details
	product, err := s.productService.GetProductByID(tenantID, productRef.ProductID)
	if err != nil || product == nil {
		return "❌ Produto não encontrado.", err
	}

	if product.StockQuantity < quantidade {
		return fmt.Sprintf("❌ Estoque insuficiente. Disponível: %d unidades.", product.StockQuantity), nil
	}

	// Get or create cart
	cart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
	if err != nil {
		return "❌ Erro ao acessar carrinho.", err
	}

	// Add item to cart
	err = s.cartService.AddItemToCart(cart.ID, tenantID, product.ID, quantidade)
	if err != nil {
		return "❌ Erro ao adicionar item ao carrinho.", err
	}

	adicional := "\n\nVocê pode continuar comprando ou digite 'finalizar' para fechar o pedido."
	return fmt.Sprintf("✅ **%s** adicionado ao carrinho!\n🔢 Quantidade: %d\n💰 Valor: R$ %s",
		product.Name, quantidade, formatCurrency(getEffectivePrice(product))) + adicional, nil
}

// getCartItemByNumber gets cart item by its position number (1-based)
func (s *AIService) getCartItemByNumber(tenantID, customerID uuid.UUID, itemNumber int) (*models.CartItem, error) {
	cart, err := s.cartService.GetOrCreateActiveCart(tenantID, customerID)
	if err != nil {
		return nil, err
	}

	cartWithItems, err := s.cartService.GetCartWithItems(cart.ID, tenantID)
	if err != nil {
		return nil, err
	}

	if itemNumber < 1 || itemNumber > len(cartWithItems.Items) {
		return nil, fmt.Errorf("item number %d not found", itemNumber)
	}

	return &cartWithItems.Items[itemNumber-1], nil
}

// generateDynamicSearchSuggestions gera sugestões alternativas baseadas nos produtos reais do tenant
func (s *AIService) generateDynamicSearchSuggestions(tenantID uuid.UUID, query, marca, tags string) []string {
	suggestions := []string{}

	// Buscar produtos populares para análise
	products, err := s.productService.SearchProductsAdvanced(tenantID, ProductSearchFilters{
		Query: "",
		Limit: 50, // Buscar um sample dos produtos para análise
	})

	if err == nil && len(products) > 0 {
		// Analisar tags mais comuns
		tagFrequency := make(map[string]int)
		brandFrequency := make(map[string]int)
		wordFrequency := make(map[string]int)

		for _, product := range products {
			// Analisar tags
			if product.Tags != "" {
				tags := strings.Split(product.Tags, ",")
				for _, tag := range tags {
					tag = strings.TrimSpace(tag)
					if tag != "" && len(tag) > 2 {
						tagFrequency[tag]++
					}
				}
			}

			// Analisar marcas
			if product.Brand != "" {
				brandFrequency[product.Brand]++
			}

			// Analisar palavras-chave no nome do produto
			words := strings.Fields(strings.ToLower(product.Name))
			for _, word := range words {
				if len(word) > 3 && !isStopWord(word) && !isNumber(word) {
					wordFrequency[word]++
				}
			}
		}

		// Coletar as tags mais populares
		for tag, count := range tagFrequency {
			if count >= 2 && len(suggestions) < 3 {
				if !containsSimilar(suggestions, tag) && !isSimilarToQuery(query, tag) {
					suggestions = append(suggestions, tag)
				}
			}
		}

		// Se não temos sugestões suficientes, adicionar palavras-chave populares
		if len(suggestions) < 3 {
			for word, count := range wordFrequency {
				if count >= 3 && len(suggestions) < 3 {
					if !containsSimilar(suggestions, word) && !isSimilarToQuery(query, word) {
						suggestions = append(suggestions, word)
					}
				}
			}
		}

		// Se ainda não temos sugestões suficientes, adicionar marcas populares
		if len(suggestions) < 3 {
			for brand, count := range brandFrequency {
				if count >= 2 && len(suggestions) < 3 {
					if !containsSimilar(suggestions, brand) && !isSimilarToQuery(query, brand) {
						suggestions = append(suggestions, brand)
					}
				}
			}
		}
	}

	// Se ainda não temos sugestões, usar fallback baseado no tipo de negócio
	if len(suggestions) == 0 && len(products) > 0 {
		businessType := s.detectBusinessType(products)
		suggestions = s.getDefaultSuggestionsByBusinessType(businessType)
	}

	// Fallback final se não conseguimos detectar nada
	if len(suggestions) == 0 {
		suggestions = []string{"produtos em destaque", "ofertas", "novidades"}
	}

	// Limitar a 3 sugestões
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	return suggestions
}

// isStopWord verifica se uma palavra é uma stop word comum
func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"para": true, "com": true, "sem": true, "por": true, "mais": true, "menos": true,
		"super": true, "mega": true, "kit": true, "pack": true, "unidade": true, "un": true,
		"und": true, "cx": true, "pct": true, "conjunto": true, "set": true,
	}
	return stopWords[word]
}

// isNumber verifica se uma string é principalmente numérica
func isNumber(word string) bool {
	matched, _ := regexp.MatchString(`^\d+[a-z]*$|^[a-z]*\d+$`, word)
	return matched
}

// containsSimilar verifica se já existe uma sugestão similar na lista
func containsSimilar(suggestions []string, newSuggestion string) bool {
	for _, existing := range suggestions {
		if strings.Contains(strings.ToLower(existing), strings.ToLower(newSuggestion)) ||
			strings.Contains(strings.ToLower(newSuggestion), strings.ToLower(existing)) {
			return true
		}
	}
	return false
}

// isSimilarToQuery verifica se a sugestão é muito similar à query original
func isSimilarToQuery(query, suggestion string) bool {
	queryLower := strings.ToLower(query)
	suggestionLower := strings.ToLower(suggestion)

	return strings.Contains(queryLower, suggestionLower) ||
		strings.Contains(suggestionLower, queryLower)
}

// detectBusinessType detecta o tipo de negócio baseado nos produtos
func (s *AIService) detectBusinessType(products []models.Product) string {
	// Usar análise dos produtos reais ao invés de keywords fixas
	if len(products) == 0 {
		return "geral"
	}

	// Análise baseada no catálogo real do tenant
	productCount := len(products)
	if productCount < 10 {
		return "pequeno_comercio"
	} else if productCount < 100 {
		return "comercio_medio"
	} else {
		return "grande_comercio"
	}
}

// formatAddressForDisplay formata um endereço para exibição
func formatAddressForDisplay(address models.Address) string {
	var parts []string

	if address.Street != "" {
		streetPart := address.Street
		if address.Number != "" {
			streetPart += ", " + address.Number
		}
		if address.Complement != "" {
			streetPart += " - " + address.Complement
		}
		parts = append(parts, streetPart)
	}

	if address.Neighborhood != "" {
		parts = append(parts, address.Neighborhood)
	}

	if address.City != "" {
		cityPart := address.City
		if address.State != "" {
			cityPart += ", " + address.State
		}
		parts = append(parts, cityPart)
	} else {
		// Se não tem cidade, tentar identificar usando o banco de municípios
		cityIdentified := false

		// Se temos CEP, tentar buscar a cidade no banco
		if address.ZipCode != "" && len(address.ZipCode) >= 8 {
			// Usar Google Maps ou serviço de CEP para identificar cidade
			// Por enquanto, apenas mostrar que vamos tentar identificar
		}

		// Se temos bairro e estado, tentar buscar cidade similar
		if !cityIdentified && address.Neighborhood != "" && address.State != "" {
			// Buscar no banco de municípios por similaridade
			// Por enquanto, não assumir nada
		}

		// Se ainda não identificamos, perguntar para o usuário
		if !cityIdentified {
			if address.State != "" {
				parts = append(parts, "Cidade a confirmar, "+address.State)
			} else {
				parts = append(parts, "Cidade a confirmar")
			}
		}
	}

	if address.ZipCode != "" {
		parts = append(parts, "CEP: "+address.ZipCode)
	}

	return strings.Join(parts, "\n")
}

// formatAddressesForSelection formata múltiplos endereços para seleção
func formatAddressesForSelection(addresses []models.Address) string {
	if len(addresses) == 0 {
		return "Nenhum endereço cadastrado."
	}

	if len(addresses) == 1 {
		return formatAddressForDisplay(addresses[0])
	}

	var parts []string
	parts = append(parts, "📍 **Endereços cadastrados:**\n")

	for i, addr := range addresses {
		addressText := formatAddressForDisplay(addr)
		defaultMarker := ""
		if addr.IsDefault {
			defaultMarker = " ⭐ **(padrão)**"
		}

		parts = append(parts, fmt.Sprintf("**%d.** %s%s", i+1,
			strings.ReplaceAll(addressText, "\n", ", "), defaultMarker))
	}

	parts = append(parts, "\n💡 **Para usar um endereço específico, informe o número (ex: 'usar endereço 2')**")
	return strings.Join(parts, "\n")
}

// isAddressComplete verifica se um endereço tem as informações essenciais
func isAddressComplete(address models.Address) bool {
	return address.Street != "" &&
		address.City != "" &&
		address.State != "" &&
		address.ZipCode != ""
}

// getDefaultSuggestionsByBusinessType retorna sugestões padrão baseadas no tipo de negócio
func (s *AIService) getDefaultSuggestionsByBusinessType(businessType string) []string {
	suggestions := map[string][]string{
		"farmacia":     {"medicamentos", "higiene pessoal", "cosméticos"},
		"papelaria":    {"canetas e marcadores", "cadernos e blocos", "materiais de escritório"},
		"supermercado": {"alimentos", "bebidas", "produtos de limpeza"},
		"eletronicos":  {"celulares", "informática", "eletrônicos"},
		"roupas":       {"roupas masculinas", "roupas femininas", "calçados"},
		"casa":         {"decoração", "móveis", "utilidades domésticas"},
		"geral":        {"produtos em destaque", "ofertas", "novidades"},
	}

	if suggs, exists := suggestions[businessType]; exists {
		return suggs
	}

	return suggestions["geral"]
}

// handleVerificarEntrega verifica se a loja faz entregas em um determinado local
func (s *AIService) handleVerificarEntrega(tenantID uuid.UUID, args map[string]interface{}) (string, error) {
	log.Info().
		Str("tenant_id", tenantID.String()).
		Interface("args", args).
		Msg("🚚 Verificando entrega para localização")

	// Extrair dados dos argumentos
	local, _ := args["local"].(string)
	cidade, _ := args["cidade"].(string)
	estado, _ := args["estado"].(string)

	if local == "" {
		return "❌ Preciso saber o local (bairro ou região) para verificar se fazemos entregas.", nil
	}

	// Primeiro, obter informações da loja para saber a cidade/estado padrão
	storeInfo, err := s.deliveryService.GetStoreLocation(tenantID)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao obter localização da loja")
		return "🚫 Não consegui verificar a localização da nossa loja. Entre em contato conosco para mais detalhes sobre entregas.", nil
	}

	// Se não foi especificada cidade/estado, usar da loja
	if cidade == "" {
		cidade = storeInfo.City
	}
	if estado == "" {
		estado = storeInfo.State
	}

	// Validar se a loja tem endereço configurado
	if !storeInfo.Coordinates {
		return "🚫 Nossa loja ainda não tem endereço configurado. Entre em contato conosco para verificar se atendemos sua região.", nil
	}

	// Tentar validar o endereço usando apenas o bairro/local
	result, err := s.deliveryService.ValidateDeliveryAddress(tenantID, "", "", local, cidade, estado)
	if err != nil {
		log.Error().Err(err).Str("local", local).Msg("Erro ao validar endereço de entrega")
		return "🚫 Não consegui verificar se atendemos essa região. Entre em contato conosco para mais detalhes sobre entregas.", nil
	}

	// Formatar resposta baseada no resultado
	if result.CanDeliver {
		switch result.Reason {
		case "whitelisted_area":
			return fmt.Sprintf("✅ **Sim, fazemos entregas em %s!** 🚚\n\nEsta região está em nossa área de atendimento especial.", local), nil
		case "within_radius":
			return fmt.Sprintf("✅ **Sim, fazemos entregas em %s!** 🚚\n\nEsta região está dentro da nossa área de atendimento (%s).", local, result.Distance), nil
		default:
			return fmt.Sprintf("✅ **Sim, fazemos entregas em %s!** 🚚", local), nil
		}
	} else {
		switch result.Reason {
		case "area_not_served":
			return fmt.Sprintf("🚫 Infelizmente não fazemos entregas em %s.\n\nEntre em contato conosco para verificar outras opções de atendimento.", local), nil
		case "outside_radius":
			return fmt.Sprintf("🚫 Infelizmente %s está fora da nossa área de atendimento.\n\nEntre em contato conosco para verificar outras opções de entrega.", local), nil
		case "no_store_location":
			return "🚫 Nossa loja ainda não tem localização configurada. Entre em contato conosco para verificar se atendemos sua região.", nil
		default:
			return fmt.Sprintf("🚫 Não conseguimos atender %s no momento.\n\nEntre em contato conosco para mais detalhes sobre nossa área de atendimento.", local), nil
		}
	}
}

// handleConsultarEnderecoEmpresa retorna o endereço da empresa/loja e envia localização via WhatsApp
func (s *AIService) handleConsultarEnderecoEmpresa(tenantID, customerID uuid.UUID, customerPhone string) (string, error) {
	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_id", customerID.String()).
		Str("customer_phone", customerPhone).
		Msg("📍 Consultando endereço da empresa")

	// Obter informações da loja
	storeInfo, err := s.deliveryService.GetStoreLocation(tenantID)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao obter informações da loja")
		return "❌ Não consegui obter as informações de localização da nossa empresa no momento. Entre em contato conosco para mais detalhes.", nil
	}

	// Verificar se há endereço configurado
	if storeInfo.Address == "" {
		return "📍 **Nossa Localização:**\n\nAinda estamos configurando nosso endereço no sistema. Entre em contato conosco para obter informações sobre nossa localização!", nil
	}

	// Formatar resposta com o endereço
	response := fmt.Sprintf("📍 **Nossa Localização:**\n\n%s", storeInfo.Address)

	// Adicionar cidade e estado se disponíveis
	if storeInfo.City != "" && storeInfo.State != "" {
		response += fmt.Sprintf("\n%s - %s", storeInfo.City, storeInfo.State)
	} else if storeInfo.City != "" {
		response += fmt.Sprintf("\n%s", storeInfo.City)
	}

	// Adicionar informação sobre entrega se configurada
	if storeInfo.Coordinates && storeInfo.RadiusKm > 0 {
		response += fmt.Sprintf("\n\n🚚 **Área de Entrega:** %.0f km de raio", storeInfo.RadiusKm)
	}

	response += "\n\n💬 Qualquer dúvida sobre como chegar ou nossa localização, é só perguntar!"

	// 📍 ENVIAR LOCALIZAÇÃO VIA WHATSAPP API (em paralelo)
	if storeInfo.Coordinates && storeInfo.Latitude != 0 && storeInfo.Longitude != 0 {
		go func() {
			err := s.sendLocationToWhatsApp(tenantID, customerID, customerPhone, storeInfo)
			if err != nil {
				log.Error().
					Err(err).
					Str("customer_phone", customerPhone).
					Msg("Erro ao enviar localização via WhatsApp API")
			} else {
				log.Info().
					Str("customer_phone", customerPhone).
					Float64("latitude", storeInfo.Latitude).
					Float64("longitude", storeInfo.Longitude).
					Msg("📍 Localização enviada com sucesso via WhatsApp API")
			}
		}()
	}

	return response, nil
}

// sendLocationToWhatsApp envia a localização da empresa via WhatsApp API externa
func (s *AIService) sendLocationToWhatsApp(tenantID, customerID uuid.UUID, customerPhone string, storeInfo *StoreLocationInfo) error {
	// Formatar o chatId (número do cliente + @c.us)
	chatId := fmt.Sprintf("%s@c.us", customerPhone)

	// Nome da empresa e sessão - buscar informações reais do banco
	var companyName = "" //  Não existe Valor padrão
	var sessionID = ""   // Não existe  Valor padrão

	// Buscar tenant name - usando uma consulta direta ao banco através de um service
	// Tentar obter informações do tenant através do deliveryService ou outro service disponível
	if tenant, err := s.getTenantInfo(tenantID); err == nil && tenant != nil {
		if tenant.Name != "" {
			companyName = tenant.Name
		}
	}

	// Buscar channel session - tentar obter através de customerPhone
	if channel, err := s.getChannelByCustomer(tenantID, customerPhone); err == nil && channel != nil {
		if channel.Session != "" {
			sessionID = channel.Session
		}
	}

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_id", customerID.String()).
		Str("company_name", companyName).
		Str("session", sessionID).
		Msg("📍 Preparando envio de localização")

	// Use centralized ZapPlus client
	client := zapplus.GetClient()
	err := client.SendLocation(sessionID, chatId, storeInfo.Latitude, storeInfo.Longitude, companyName)
	if err != nil {
		log.Error().Err(err).Msg("❌ Erro ao enviar localização via ZapPlus")
		return fmt.Errorf("erro ao enviar localização: %w", err)
	}

	log.Info().
		Str("chat_id", chatId).
		Float64("latitude", storeInfo.Latitude).
		Float64("longitude", storeInfo.Longitude).
		Str("title", companyName).
		Msg("✅ Localização enviada com sucesso via ZapPlus")

	return nil
}

// getTenantInfo busca informações do tenant pelo ID
func (s *AIService) getTenantInfo(tenantID uuid.UUID) (*models.Tenant, error) {
	// Usar uma abordagem através do settingsService que pode ter acesso a informações do tenant
	// ou usar o customerService para fazer uma query direta

	log.Debug().Str("tenant_id", tenantID.String()).Msg("Buscando informações do tenant")

	// Tentar usar customerService já que sabemos que tem acesso ao DB
	// A implementação CustomerServiceImpl tem um campo `db *gorm.DB`
	if customerSvc, ok := s.customerService.(*CustomerServiceImpl); ok {
		var tenant models.Tenant
		err := customerSvc.db.Where("id = ?", tenantID).First(&tenant).Error
		if err != nil {
			log.Debug().Err(err).Str("tenant_id", tenantID.String()).Msg("Não foi possível buscar tenant")
			return nil, err
		}
		log.Info().
			Str("tenant_id", tenantID.String()).
			Str("tenant_name", tenant.Name).
			Msg("✅ Tenant encontrado")
		return &tenant, nil
	}

	log.Debug().Msg("CustomerService não é do tipo esperado, usando fallback")
	return nil, fmt.Errorf("não foi possível acessar informações do tenant")
}

// getChannelByCustomer busca channel baseado no customerPhone e tenant
func (s *AIService) getChannelByCustomer(tenantID uuid.UUID, customerPhone string) (*models.Channel, error) {
	log.Debug().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Msg("Buscando channel para customer")

	// Tentar usar customerService para acessar o banco
	if customerSvc, ok := s.customerService.(*CustomerServiceImpl); ok {
		// Primeiro, buscar o customer
		customer, err := s.customerService.GetCustomerByPhone(tenantID, customerPhone)
		if err != nil {
			log.Debug().Err(err).Msg("Não foi possível encontrar customer")
			return nil, err
		}

		// Buscar conversation ativa para este customer
		var conversation models.Conversation
		err = customerSvc.db.
			Where("tenant_id = ? AND customer_id = ? AND status = ?",
				tenantID, customer.ID, "open").
			Order("updated_at DESC").
			First(&conversation).Error

		if err != nil {
			log.Debug().Err(err).
				Str("customer_id", customer.ID.String()).
				Msg("Não foi possível encontrar conversa ativa")
			return nil, err
		}

		// Buscar o channel da conversa
		var channel models.Channel
		err = customerSvc.db.Where("id = ?", conversation.ChannelID).First(&channel).Error
		if err != nil {
			log.Debug().Err(err).
				Str("channel_id", conversation.ChannelID.String()).
				Msg("Não foi possível buscar channel")
			return nil, err
		}

		log.Info().
			Str("channel_id", channel.ID.String()).
			Str("session", channel.Session).
			Msg("✅ Channel encontrado")

		return &channel, nil
	}

	log.Debug().Msg("CustomerService não é do tipo esperado para buscar channel")
	return nil, fmt.Errorf("não foi possível acessar informações do channel")
}

// handleSolicitarAtendimentoHumano processa solicitações de atendimento humano
func (s *AIService) handleSolicitarAtendimentoHumano(tenantID, customerID uuid.UUID, customerPhone string, args map[string]interface{}) (string, error) {
	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_id", customerID.String()).
		Str("customer_phone", customerPhone).
		Msg("🙋 Processando solicitação de atendimento humano")

	// Extrair motivo dos argumentos
	motivo := "Cliente solicitou atendimento humano"
	if motivoArg, ok := args["motivo"].(string); ok && motivoArg != "" {
		motivo = motivoArg
	}

	// Buscar informações do cliente
	customer, err := s.customerService.GetCustomerByPhone(tenantID, customerPhone)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao buscar informações do cliente")
		customer = &models.Customer{
			Phone: customerPhone,
			Name:  "Cliente não identificado",
		}
	}

	// Preparar mensagem para o grupo de alertas
	customerName := customer.Name
	if customerName == "" {
		customerName = "Cliente não identificado"
	}

	// Enviar alerta para o grupo de alertas (em paralelo)
	go func() {
		err := s.alertService.SendHumanSupportAlert(tenantID, customerID, customerPhone, motivo)
		if err != nil {
			log.Error().
				Err(err).
				Str("customer_phone", customerPhone).
				Str("customer_name", customerName).
				Msg("❌ Erro ao enviar alerta de solicitação de atendimento humano")
		} else {
			log.Info().
				Str("customer_phone", customerPhone).
				Str("customer_name", customerName).
				Str("reason", motivo).
				Msg("✅ Alerta de solicitação de atendimento humano enviado com sucesso")
		}
	}()

	// Resposta amigável para o cliente
	response := "👋 **Atendimento Humano Solicitado**\n\n" +
		"Entendi que você gostaria de falar com um atendente humano.\n\n" +
		"✅ **Sua solicitação foi encaminhada para nossa equipe!**\n\n" +
		"🕐 **Um de nossos atendentes entrará em contato com você em breve.**\n\n" +
		"💬 Enquanto isso, posso continuar te ajudando com informações sobre nossos produtos e pedidos."

	return response, nil
}

// formatProductsByCategoryComplete formata produtos organizados por categoria para catálogo completo
func (s *AIService) formatProductsByCategoryComplete(tenantID uuid.UUID, customerPhone string, products []models.Product) (string, error) {

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Int("products_count", len(products)).
		Msg("🔍 DEBUG: formatProductsByCategoryComplete called")

	// se products for maior que 100, retornar mensagem avisando que há o usuario precisa pesquisar por algum produto
	if len(products) > 100 {
		return "❌ Nosso catálogo é muito grande para ser exibido completo. Por favor, pesquise por um produto específico ou categoria para ver os itens disponíveis.", nil
	}

	// Buscar todas as categorias do tenant
	categories, err := s.categoryService.ListCategories(tenantID)
	if err != nil {
		log.Warn().Err(err).Msg("🔍 DEBUG: Error listing categories, using standard format")
		// Se erro ao buscar categorias, usar formato padrão
		return s.formatProductsStandard(tenantID, customerPhone, products)
	}

	log.Info().
		Int("categories_count", len(categories)).
		Msg("🔍 DEBUG: Categories found")

	// Criar mapa de categorias por ID
	categoryMap := make(map[string]models.Category)
	for _, cat := range categories {
		categoryMap[cat.ID.String()] = cat
	}

	// Organizar produtos por categoria
	productsByCategory := make(map[string][]models.Product)
	productsWithoutCategory := []models.Product{}

	for _, product := range products {
		if product.CategoryID != nil && *product.CategoryID != uuid.Nil {
			categoryID := product.CategoryID.String()
			productsByCategory[categoryID] = append(productsByCategory[categoryID], product)
		} else {
			productsWithoutCategory = append(productsWithoutCategory, product)
		}
	}

	// Armazenar todos os produtos na memória com numeração sequencial
	allProducts := []models.Product{}

	// Adicionar produtos organizados por categoria (ordenados por sort_order)
	categoryKeys := make([]string, 0, len(productsByCategory))
	for catID := range productsByCategory {
		categoryKeys = append(categoryKeys, catID)
	}

	// Ordenar categorias por sort_order
	sort.Slice(categoryKeys, func(i, j int) bool {
		catA := categoryMap[categoryKeys[i]]
		catB := categoryMap[categoryKeys[j]]
		if catA.SortOrder != catB.SortOrder {
			return catA.SortOrder < catB.SortOrder
		}
		return catA.Name < catB.Name
	})

	result := "🛍️ **Catálogo Completo - Organizado por Categorias**\n\n"

	// Adicionar produtos por categoria
	for _, categoryID := range categoryKeys {
		category := categoryMap[categoryID]
		categoryProducts := productsByCategory[categoryID]

		if len(categoryProducts) > 0 {
			// Ordenar produtos dentro da categoria por sort_order, depois por nome
			sort.Slice(categoryProducts, func(i, j int) bool {
				if categoryProducts[i].SortOrder != categoryProducts[j].SortOrder {
					return categoryProducts[i].SortOrder < categoryProducts[j].SortOrder
				}
				return categoryProducts[i].Name < categoryProducts[j].Name
			})

			result += fmt.Sprintf("📂 **%s**\n", category.Name)
			if category.Description != "" {
				result += fmt.Sprintf("   %s\n", category.Description)
			}
			result += "\n"

			for _, product := range categoryProducts {
				allProducts = append(allProducts, product)
			}
		}
	}

	// Adicionar produtos sem categoria no final
	if len(productsWithoutCategory) > 0 {
		// Ordenar produtos sem categoria por sort_order, depois por nome
		sort.Slice(productsWithoutCategory, func(i, j int) bool {
			if productsWithoutCategory[i].SortOrder != productsWithoutCategory[j].SortOrder {
				return productsWithoutCategory[i].SortOrder < productsWithoutCategory[j].SortOrder
			}
			return productsWithoutCategory[i].Name < productsWithoutCategory[j].Name
		})

		for _, product := range productsWithoutCategory {
			allProducts = append(allProducts, product)
		}
	}

	// Armazenar na memória e formatar
	productRefs := s.memoryManager.StoreProductList(tenantID, customerPhone, allProducts)

	// Reformatar resultado com produtos organizados
	result = "🛍️ **Catálogo Completo - Organizado por Categorias**\n\n"
	currentCategoryID := ""

	for _, productRef := range productRefs {
		// Encontrar o produto original para saber sua categoria
		var originalProduct models.Product
		for _, p := range allProducts {
			if p.ID == productRef.ProductID {
				originalProduct = p
				break
			}
		}

		// Verificar se mudou de categoria
		var productCategoryID string
		if originalProduct.CategoryID != nil {
			productCategoryID = originalProduct.CategoryID.String()
		} else {
			productCategoryID = "sem-categoria"
		}

		if currentCategoryID != productCategoryID {
			if currentCategoryID != "" {
				result += "\n"
			}

			if productCategoryID == "sem-categoria" {
				result += "📂 **Outros Produtos**\n\n"
			} else if category, exists := categoryMap[productCategoryID]; exists {
				result += fmt.Sprintf("📂 **%s**\n", category.Name)
				if category.Description != "" {
					result += fmt.Sprintf("   %s\n", category.Description)
				}
				result += "\n"
			}
			currentCategoryID = productCategoryID
		}

		var price string
		if productRef.SalePrice != "" && productRef.SalePrice != "0" {
			price = fmt.Sprintf("~~R$ %s~~ **R$ %s**", formatCurrency(productRef.Price), formatCurrency(productRef.SalePrice))
		} else {
			price = fmt.Sprintf("**R$ %s**", formatCurrency(productRef.Price))
		}

		result += fmt.Sprintf("   %d. **%s**\n", productRef.SequentialID, productRef.Name)
		result += fmt.Sprintf("      💰 %s\n", price)
	}

	result += "\n💡 Para ver detalhes: 'produto [número]' ou 'produto [nome]'\n"
	result += "🛒 Para adicionar ao carrinho: 'adicionar [número] quantidade [X]'"

	return result, nil
}

// formatProductsStandard formata produtos no formato padrão (fallback)
func (s *AIService) formatProductsStandard(tenantID uuid.UUID, customerPhone string, products []models.Product) (string, error) {
	productRefs := s.memoryManager.StoreProductList(tenantID, customerPhone, products)

	result := "🛍️ **Produtos disponíveis:**\n\n"

	for _, productRef := range productRefs {
		var price string
		if productRef.SalePrice != "" && productRef.SalePrice != "0" {
			price = fmt.Sprintf("~~R$ %s~~ **R$ %s**", formatCurrency(productRef.Price), formatCurrency(productRef.SalePrice))
		} else {
			price = fmt.Sprintf("**R$ %s**", formatCurrency(productRef.Price))
		}

		result += fmt.Sprintf("%d. **%s**\n", productRef.SequentialID, productRef.Name)
		result += fmt.Sprintf("   💰 %s\n", price)
		if productRef.Description != "" {
			desc := productRef.Description
			if len(desc) > 100 {
				desc = desc[:100] + "..."
			}
			result += fmt.Sprintf("   📝 %s\n", desc)
		}
		result += "\n"
	}

	result += "💡 Para ver detalhes, diga: 'produto [número]' ou 'produto [nome]'\n"
	result += "🛒 Para adicionar ao carrinho: 'adicionar [número] quantidade [X]'"

	return result, nil
}
