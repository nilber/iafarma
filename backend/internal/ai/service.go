package ai

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"iafarma/pkg/models"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
)

type AIService struct {
	client           *openai.Client
	cartService      CartServiceInterface
	orderService     OrderServiceInterface
	productService   ProductServiceInterface
	customerService  CustomerServiceInterface
	messageService   MessageServiceInterface
	addressService   AddressServiceInterface
	settingsService  TenantSettingsServiceInterface
	municipioService MunicipioServiceInterface
	categoryService  CategoryServiceInterface
	memoryManager    *MemoryManager
	errorHandler     *ErrorHandler
	alertService     AlertServiceInterface
	deliveryService  DeliveryServiceInterface
	embeddingService EmbeddingServiceInterface
	s3Client         *s3.S3
	s3Bucket         string
	s3BaseURL        string
	// Map temporário para armazenar conversationID por sessão
	conversationContext sync.Map
	// Armazenar resultados de funções da última execução
	lastFunctionResults  []ToolExecutionResult
	functionResultsMutex sync.RWMutex
}

// Interfaces para injeção de dependência
type CartServiceInterface interface {
	GetOrCreateActiveCart(tenantID, customerID uuid.UUID) (*models.Cart, error)
	AddItemToCart(cartID, tenantID uuid.UUID, productID uuid.UUID, quantity int) error
	RemoveItemFromCart(cartID, tenantID, itemID uuid.UUID) error
	ClearCart(cartID, tenantID uuid.UUID) error
	UpdateCartItemQuantity(cartID, tenantID, itemID uuid.UUID, quantity int) error
	GetCartWithItems(cartID, tenantID uuid.UUID) (*models.Cart, error)
}

type OrderServiceInterface interface {
	CreateOrderFromCart(tenantID, cartID uuid.UUID) (*models.Order, error)
	CreateOrderFromCartWithAddress(tenantID, cartID uuid.UUID, deliveryAddress *models.Address) (*models.Order, error)
	CreateOrderFromCartWithConversation(tenantID, cartID, conversationID uuid.UUID, deliveryAddress *models.Address) (*models.Order, error)
	GetOrdersByCustomer(tenantID, customerID uuid.UUID) ([]models.Order, error)
	CancelOrder(tenantID, orderID uuid.UUID) error
	GetPaymentOptions(tenantID uuid.UUID) ([]PaymentOption, error)
}

type ProductServiceInterface interface {
	SearchProducts(tenantID uuid.UUID, query string, limit int) ([]models.Product, error)
	SearchProductsAdvanced(tenantID uuid.UUID, filters ProductSearchFilters) ([]models.Product, error)
	GetProductByID(tenantID, productID uuid.UUID) (*models.Product, error)
	GetPromotionalProducts(tenantID uuid.UUID) ([]models.Product, error)
	GetAllTenants() ([]models.Tenant, error)
	GetTenantByID(tenantID uuid.UUID) (*models.Tenant, error)
	GetProductsByTenantID(tenantID uuid.UUID) ([]models.Product, error)
}

// ProductSearchFilters represents advanced search filters
type ProductSearchFilters struct {
	Query    string
	Brand    string
	Tags     string
	MinPrice float64
	MaxPrice float64
	Limit    int
	SortBy   string // "price_asc", "price_desc", "name_asc", "name_desc", "relevance"
}

type CustomerServiceInterface interface {
	UpdateCustomerProfile(tenantID, customerID uuid.UUID, data CustomerUpdateData) error
	GetCustomerByPhone(tenantID uuid.UUID, phone string) (*models.Customer, error)
	GetCustomerByID(tenantID, customerID uuid.UUID) (*models.Customer, error)
}

type MessageServiceInterface interface {
	UpdateMessageContent(messageID, content string) error
	UpdateMessageContentAndMediaURL(messageID, content, mediaURL string) error
	GetMessageByID(messageID string) (*models.Message, error)
}

type AddressServiceInterface interface {
	GetAddressesByCustomer(tenantID, customerID uuid.UUID) ([]models.Address, error)
	CreateAddress(tenantID uuid.UUID, address *models.Address) error
	SetDefaultAddress(tenantID, customerID, addressID uuid.UUID) error
	DeleteAddress(tenantID, customerID, addressID uuid.UUID) error
	DeleteAllAddresses(tenantID, customerID uuid.UUID) error
}

type TenantSettingsServiceInterface interface {
	GetSetting(ctx context.Context, tenantID uuid.UUID, key string) (*models.TenantSetting, error)
	GenerateAIProductExamples(ctx context.Context, tenantID uuid.UUID) ([]string, error)
	GenerateWelcomeMessage(ctx context.Context, tenantID uuid.UUID) (string, error)
	GetWelcomeMessage(ctx context.Context, tenantID uuid.UUID) (string, error)
	GenerateFullSystemPrompt(ctx context.Context, tenantID uuid.UUID) (string, error)
	SetSetting(ctx context.Context, tenantID uuid.UUID, key string, value *string, settingType string) error
}

type MunicipioServiceInterface interface {
	ValidarCidade(nomeCidade, uf string) (bool, *models.MunicipioBrasileiro, error)
	BuscarCidadesComSimilaridade(nomeCidade, uf string, limite int) ([]models.MunicipioBrasileiro, error)
}

type CategoryServiceInterface interface {
	ListCategories(tenantID uuid.UUID) ([]models.Category, error)
	GetCategoryByID(tenantID, id uuid.UUID) (*models.Category, error)
}

type AlertServiceInterface interface {
	SendOrderAlert(tenantID uuid.UUID, order *models.Order, customerPhone string) error
	SendHumanSupportAlert(tenantID uuid.UUID, customerID uuid.UUID, customerPhone string, reason string) error
}

type DeliveryServiceInterface interface {
	ValidateDeliveryAddress(tenantID uuid.UUID, street, number, neighborhood, city, state string) (*DeliveryValidationResult, error)
	GetStoreLocation(tenantID uuid.UUID) (*StoreLocationInfo, error)
}

type EmbeddingServiceInterface interface {
	SearchSimilarProducts(query, tenantID string, limit int) ([]ProductSearchResult, error)
	SearchConversations(tenantID, customerID, query string, limit int) ([]ConversationSearchResult, error)
	SearchConversationsWithMaxAge(tenantID, customerID, query string, limit int, maxAgeHours int) ([]ConversationSearchResult, error)
	StoreConversation(tenantID, customerID string, entry ConversationEntry) error
	CleanupOldConversations(tenantID, customerID string, maxAgeHours int) (int, error)
}

type StorageServiceInterface interface {
	UploadAudioFile(mediaURL, tenantID, customerID, messageID string) (string, error)
	UploadImageFile(mediaURL, tenantID, customerID, messageID string) (string, error)
}

// RAG Types
type ProductSearchResult struct {
	ID       string                 `json:"id"`
	Text     string                 `json:"text"`
	Score    float32                `json:"score"`
	Metadata map[string]interface{} `json:"metadata"`
}

type ConversationSearchResult struct {
	ID        string                 `json:"id"`
	Score     float32                `json:"score"`
	Message   string                 `json:"message"`
	Response  string                 `json:"response"`
	Timestamp string                 `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type ConversationEntry struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	CustomerID string                 `json:"customer_id"`
	Message    string                 `json:"message"`
	Response   string                 `json:"response"`
	Timestamp  string                 `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type DeliveryValidationResult struct {
	CanDeliver bool   `json:"can_deliver"`
	Reason     string `json:"reason"`
	ZoneType   string `json:"zone_type,omitempty"`
	Distance   string `json:"distance,omitempty"`
}

type StoreLocationInfo struct {
	Address     string  `json:"address"`
	City        string  `json:"city"`
	State       string  `json:"state"`
	Coordinates bool    `json:"coordinates_configured"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
	RadiusKm    float64 `json:"delivery_radius_km"`
}

type PaymentOption struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Instructions string `json:"instructions"`
}

type BusinessInfo struct {
	Type        string   `json:"type"`        // Tipo de negócio (ex: "papelaria", "farmácia", "loja de roupas")
	Description string   `json:"description"` // Descrição do negócio
	Examples    []string `json:"examples"`    // Exemplos específicos do negócio
	Categories  []string `json:"categories"`  // Principais categorias de produtos
}

type CustomerUpdateData struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Address string `json:"address"`
}

// Estrutura para o resultado do parsing de endereço via IA
type AIAddressParsing struct {
	Street       string `json:"rua"`
	Number       string `json:"numero"`
	Complement   string `json:"complemento"`
	Neighborhood string `json:"bairro"`
	City         string `json:"cidade"`
	State        string `json:"estado"`
	ZipCode      string `json:"cep"`
}

// Tool function definitions for OpenAI
type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

func NewAIService(apiKey string, cartService CartServiceInterface, orderService OrderServiceInterface,
	productService ProductServiceInterface, customerService CustomerServiceInterface,
	addressService AddressServiceInterface, settingsService TenantSettingsServiceInterface,
	municipioService MunicipioServiceInterface, categoryService CategoryServiceInterface, alertService AlertServiceInterface,
	deliveryService DeliveryServiceInterface) *AIService {

	// Create custom HTTP client with TLS configuration for macOS compatibility
	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false, // Keep security but use system certs
			},
		},
	}

	// Create OpenAI client with custom HTTP client
	config := openai.DefaultConfig(apiKey)
	config.HTTPClient = httpClient
	client := openai.NewClientWithConfig(config)

	// Initialize S3 client
	var s3Client *s3.S3
	var s3Bucket, s3BaseURL string

	endpoint := os.Getenv("S3_ENDPOINT")
	accessKey := os.Getenv("S3_ACCESS_KEY")
	secretKey := os.Getenv("S3_SECRET_KEY")
	bucket := os.Getenv("S3_BUCKET")

	if accessKey != "" && secretKey != "" && bucket != "" {
		// Create AWS session with proper endpoint configuration
		config := &aws.Config{
			Region: aws.String("us-east-1"),
			Credentials: credentials.NewStaticCredentials(
				accessKey,
				secretKey,
				"",
			),
		}

		// Only set custom endpoint if provided
		if endpoint != "" {
			config.Endpoint = aws.String(endpoint)
			config.S3ForcePathStyle = aws.Bool(true)
		}

		sess, err := session.NewSession(config)
		if err == nil {
			s3Client = s3.New(sess)
			s3Bucket = bucket
			s3BaseURL = fmt.Sprintf("https://%s", bucket)
			log.Info().Str("bucket", bucket).Msg("S3 storage initialized successfully")
		} else {
			log.Error().Err(err).Msg("Failed to initialize S3 storage")
		}
	} else {
		log.Warn().Msg("S3 configuration missing, storage features disabled")
	}

	return &AIService{
		client:           client,
		cartService:      cartService,
		orderService:     orderService,
		productService:   productService,
		customerService:  customerService,
		addressService:   addressService,
		settingsService:  settingsService,
		municipioService: municipioService,
		categoryService:  categoryService,
		memoryManager:    NewMemoryManager(),
		alertService:     alertService,
		deliveryService:  deliveryService,
		s3Client:         s3Client,
		s3Bucket:         s3Bucket,
		s3BaseURL:        s3BaseURL,
	}
}

// 🧠 RAG (Retrieval-Augmented Generation) Functions

// normalizeProductQuery normaliza termos equivalentes para garantir consistência na busca RAG
func (s *AIService) normalizeProductQuery(query string) string {
	queryLower := strings.ToLower(strings.TrimSpace(query))

	// Mapeamento de termos equivalentes para busca de produtos
	productTerms := map[string]string{
		"cardápio":             "produtos",
		"cardapio":             "produtos",
		"menu":                 "produtos",
		"catálogo":             "produtos",
		"catalogo":             "produtos",
		"lista de produtos":    "produtos",
		"produtos disponíveis": "produtos",
		"mostrar produtos":     "produtos",
		"ver produtos":         "produtos",
		"listar produtos":      "produtos",
		"o que vocês têm":      "produtos",
		"o que voces tem":      "produtos",
		"que produtos tem":     "produtos",
		"quais produtos":       "produtos",
	}

	// Verificar se a query corresponde a algum termo de produto
	if normalized, exists := productTerms[queryLower]; exists {
		log.Debug().
			Str("original_query", query).
			Str("normalized_query", normalized).
			Msg("🔄 Query normalizada para busca RAG consistente")
		return normalized
	}

	// Se contém palavras-chave de produto, normalizar para "produtos"
	if strings.Contains(queryLower, "produto") ||
		strings.Contains(queryLower, "cardápio") ||
		strings.Contains(queryLower, "cardapio") ||
		strings.Contains(queryLower, "menu") ||
		strings.Contains(queryLower, "catálogo") ||
		strings.Contains(queryLower, "catalogo") {
		log.Debug().
			Str("original_query", query).
			Str("normalized_query", "produtos").
			Msg("🔄 Query com palavra-chave de produto normalizada")
		return "produtos"
	}

	return query
}

// getRAGProductContext busca produtos similares usando RAG para enriquecer o contexto da IA
func (s *AIService) getRAGProductContext(ctx context.Context, tenantID uuid.UUID, query string) string {
	log.Debug().
		Str("tenant_id", tenantID.String()).
		Str("query", query).
		Msg("🔍 Searching for similar products using RAG")

	if s.embeddingService == nil {
		log.Warn().Msg("⚠️ Embedding service not available, skipping product RAG")
		return ""
	}

	// Normalizar a query para garantir consistência
	normalizedQuery := s.normalizeProductQuery(query)

	// Buscar produtos similares - converter UUID para string e usar parâmetros corretos
	similarProducts, err := s.embeddingService.SearchSimilarProducts(normalizedQuery, tenantID.String(), 20)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID.String()).
			Msg("❌ Error searching similar products")
		return ""
	}

	if len(similarProducts) == 0 {
		log.Debug().Msg("📭 No similar products found")
		return ""
	}

	// Formatar contexto dos produtos usando os campos disponíveis
	var productContext strings.Builder
	productContext.WriteString("🛍️ PRODUTOS RELEVANTES ENCONTRADOS:\n")

	for i, product := range similarProducts {
		productContext.WriteString(fmt.Sprintf("\n%d. **Produto ID: %s**", i+1, product.ID))

		if product.Text != "" {
			productContext.WriteString(fmt.Sprintf("\n   📝 %s", product.Text))
		}

		productContext.WriteString(fmt.Sprintf("\n   🎯 Relevância: %.3f", product.Score))

		// Extrair informações do metadata, se disponível
		if product.Metadata != nil {
			if name, ok := product.Metadata["name"].(string); ok && name != "" {
				productContext.WriteString(fmt.Sprintf("\n   🏷️ Nome: %s", name))
			}
			if price, ok := product.Metadata["price"].(float64); ok && price > 0 {
				productContext.WriteString(fmt.Sprintf("\n   💰 R$ %.2f", price))
			}
			if category, ok := product.Metadata["category"].(string); ok && category != "" {
				productContext.WriteString(fmt.Sprintf("\n   📂 Categoria: %s", category))
			}
			if stock, ok := product.Metadata["stock"].(float64); ok && stock >= 0 {
				productContext.WriteString(fmt.Sprintf("\n   📦 Estoque: %.0f unidades", stock))
			}
		}

		productContext.WriteString("\n")
	}

	productContext.WriteString("\n💡 Use essas informações para responder sobre produtos de forma precisa e detalhada.")

	log.Info().
		Int("products_found", len(similarProducts)).
		Msg("✅ RAG product context generated successfully")

	return productContext.String()
}

// 🏠 parseAddressWithAI usa GPT para extrair campos estruturados de um endereço em texto livre
func (s *AIService) parseAddressWithAI(ctx context.Context, addressText string) (*AIAddressParsing, error) {
	log.Info().
		Str("address_text", addressText).
		Msg("🧠 Parsing address with AI")

	// Configurar o prompt do sistema para parsing de endereço
	systemPrompt := `Você é um especialista em endereços brasileiros. Sua tarefa é extrair campos estruturados de um endereço em texto livre.

REGRAS IMPORTANTES:
1. Extraia APENAS as informações que estão claramente presentes no texto
2. NÃO invente ou suponha informações que não estão explícitas
3. Para campos não encontrados, retorne string vazia
4. CEP deve conter apenas números (remova hífens e espaços)
5. Estado deve ser a sigla de 2 letras (ex: ES, SP, RJ)
6. Números de apartamento/complemento devem ir no campo "complemento"

EXEMPLOS:
- "Avenida Hugo Musso, número 1333, no bairro Praia da Costa, Vila Velha Espírito Santo. O CEP lá é 29-101-280, no apartamento 300"
  → rua: "Avenida Hugo Musso", numero: "1333", bairro: "Praia da Costa", cidade: "Vila Velha", estado: "ES", cep: "29101280", complemento: "apartamento 300"

Extraia os campos do endereço fornecido usando a função disponível.`

	// Criar mensagens para o GPT
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: systemPrompt,
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: fmt.Sprintf("Extraia os campos do seguinte endereço: %s", addressText),
		},
	}

	// Definir function calling para extrair os campos do endereço
	tools := []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "extrair_endereco",
				Description: "Extrai campos estruturados de um endereço brasileiro em texto livre",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"rua": map[string]interface{}{
							"type":        "string",
							"description": "Nome da rua, avenida, alameda, etc. (ex: 'Avenida Hugo Musso')",
						},
						"numero": map[string]interface{}{
							"type":        "string",
							"description": "Número do endereço (ex: '1333')",
						},
						"complemento": map[string]interface{}{
							"type":        "string",
							"description": "Apartamento, casa, bloco, andar, etc. (ex: 'apartamento 300', 'casa 2')",
						},
						"bairro": map[string]interface{}{
							"type":        "string",
							"description": "Nome do bairro ou distrito (ex: 'Praia da Costa')",
						},
						"cidade": map[string]interface{}{
							"type":        "string",
							"description": "Nome da cidade (ex: 'Vila Velha')",
						},
						"estado": map[string]interface{}{
							"type":        "string",
							"description": "Sigla do estado com 2 letras maiúsculas (ex: 'ES', 'SP', 'RJ')",
						},
						"cep": map[string]interface{}{
							"type":        "string",
							"description": "CEP com apenas números, sem hífens (ex: '29101280')",
						},
					},
					"required": []string{"rua", "numero", "bairro", "cidade", "estado", "cep", "complemento"},
				},
			},
		},
	}

	// Fazer a chamada para o GPT
	req := openai.ChatCompletionRequest{
		Model:       openai.GPT4TurboPreview,
		Messages:    messages,
		Tools:       tools,
		ToolChoice:  "auto",
		Temperature: 0.1, // Baixa temperatura para respostas consistentes
		MaxTokens:   500,
	}

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("erro ao chamar GPT para parsing de endereço: %w", err)
	}

	if len(resp.Choices) == 0 || len(resp.Choices[0].Message.ToolCalls) == 0 {
		return nil, fmt.Errorf("GPT não retornou function call para parsing de endereço")
	}

	// Processar a resposta do function call
	toolCall := resp.Choices[0].Message.ToolCalls[0]
	if toolCall.Function.Name != "extrair_endereco" {
		return nil, fmt.Errorf("GPT retornou function call inesperada: %s", toolCall.Function.Name)
	}

	// Parse do JSON retornado
	var parsedAddress AIAddressParsing
	err = json.Unmarshal([]byte(toolCall.Function.Arguments), &parsedAddress)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer parse do JSON do endereço: %w", err)
	}

	log.Info().
		Interface("parsed_address", parsedAddress).
		Msg("✅ Address parsed successfully with AI")

	return &parsedAddress, nil
}

// getRAGConversationContext busca conversas similares para fornecer contexto histórico
// 🚫 DESABILITADO: Função desabilitada para evitar respostas repetitivas
// O histórico de conversa já é gerenciado adequadamente pelo memoryManager
/*
func (s *AIService) getRAGConversationContext(ctx context.Context, tenantID uuid.UUID, customerPhone, query string) string {
	log.Debug().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Str("query", query).
		Msg("🧠 Searching for similar conversations using RAG")

	if s.embeddingService == nil {
		log.Warn().Msg("⚠️ Embedding service not available, skipping conversation RAG")
		return ""
	}

	// Buscar conversas similares
	similarConversations, err := s.embeddingService.SearchConversations(tenantID.String(), customerPhone, query, 3)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID.String()).
			Str("customer_phone", customerPhone).
			Msg("❌ Error searching similar conversations")
		return ""
	}

	if len(similarConversations) == 0 {
		log.Debug().Msg("📭 No similar conversations found")
		return ""
	}

	// Formatar contexto das conversas
	var conversationContext strings.Builder
	conversationContext.WriteString("💬 CONVERSAS RELEVANTES ENCONTRADAS:\n")

	for i, conversation := range similarConversations {
		conversationContext.WriteString(fmt.Sprintf("\n%d. **Conversa do %s**", i+1, conversation.Timestamp))
		conversationContext.WriteString(fmt.Sprintf("\n   🎯 Relevância: %.3f", conversation.Score))

		if conversation.Message != "" {
			conversationContext.WriteString(fmt.Sprintf("\n   👤 Cliente: %s", conversation.Message))
		}

		if conversation.Response != "" {
			conversationContext.WriteString(fmt.Sprintf("\n   🤖 Resposta: %s", conversation.Response))
		}

		conversationContext.WriteString("\n")
	}

	conversationContext.WriteString("\n💡 Use esse histórico para manter contexto e continuidade na conversa.")

	log.Info().
		Int("conversations_found", len(similarConversations)).
		Msg("✅ RAG conversation context generated successfully")

	return conversationContext.String()
}
*/

// storeConversationInRAG armazena a conversa no sistema RAG para futuros contextos
// 🚫 DESABILITADO: Função desabilitada para evitar respostas repetitivas
// O histórico de conversa já é gerenciado adequadamente pelo memoryManager
/*
func (s *AIService) storeConversationInRAG(ctx context.Context, tenantID uuid.UUID, customerPhone, message, response string) {
	log.Debug().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Msg("💾 Storing conversation in RAG system")

	if s.embeddingService == nil {
		log.Warn().Msg("⚠️ Embedding service not available, skipping conversation storage")
		return
	}

	// Criar entrada de conversa - Gerar UUID válido para o ID
	conversationID := uuid.New().String()

	entry := ConversationEntry{
		ID:         conversationID,
		TenantID:   tenantID.String(),
		CustomerID: customerPhone,
		Message:    message,
		Response:   response,
		Timestamp:  time.Now().Format(time.RFC3339),
		Metadata: map[string]interface{}{
			"created_at": time.Now().Format(time.RFC3339),
			"source":     "whatsapp",
		},
	}

	// Armazenar conversa
	err := s.embeddingService.StoreConversation(tenantID.String(), customerPhone, entry)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID.String()).
			Str("customer_phone", customerPhone).
			Msg("❌ Error storing conversation in RAG")
		return
	}

	log.Info().
		Str("conversation_id", entry.ID).
		Msg("✅ Conversation stored in RAG successfully")
}
*/

// minInt function helper
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (s *AIService) ProcessMessage(ctx context.Context, tenantID uuid.UUID, customerPhone, message string) (string, error) {
	return s.ProcessMessageWithConversation(ctx, tenantID, customerPhone, message, uuid.Nil)
}

func (s *AIService) ProcessMessageWithConversation(ctx context.Context, tenantID uuid.UUID, customerPhone, message string, conversationID uuid.UUID) (string, error) {
	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Str("message", message).
		Str("conversation_id", conversationID.String()).
		Msg("AI ProcessMessage started")

	// Armazenar conversationID para uso nos handlers
	sessionKey := fmt.Sprintf("%s-%s", tenantID.String(), customerPhone)
	if conversationID != uuid.Nil {
		s.conversationContext.Store(sessionKey, conversationID)
		log.Info().Str("session_key", sessionKey).Str("conversation_id", conversationID.String()).Msg("📝 Stored conversation ID for session")
	}

	// Buscar ou criar cliente
	customer, err := s.customerService.GetCustomerByPhone(tenantID, customerPhone)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID.String()).
			Str("customer_phone", customerPhone).
			Msg("Failed to get customer by phone")
		return "", fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	log.Info().
		Str("customer_id", customer.ID.String()).
		Str("customer_name", customer.Name).
		Msg("Customer found/created successfully")

	// Obter histórico da conversa para manter contexto
	conversationHistory := s.memoryManager.GetConversationHistory(tenantID, customerPhone)

	// 🎯 NOVA LÓGICA: Verificar se é a primeira mensagem e se é uma saudação simples
	isFirstMessage := len(conversationHistory) == 0
	isSimpleGreeting := s.isSimpleGreeting(message)

	if isFirstMessage && isSimpleGreeting {
		log.Info().
			Str("tenant_id", tenantID.String()).
			Str("customer_phone", customerPhone).
			Msg("🎉 Detected first message with simple greeting - sending welcome message")

		// Verificar horários de funcionamento para saudação
		hoursInfo := s.getBusinessHoursInfo(ctx, tenantID)

		var welcomeMessage string
		if hoursInfo != "" && strings.Contains(hoursInfo, "🔴 A loja está FECHADA") {
			// Loja fechada - resposta específica
			log.Info().Msg("🕐 Store is closed - sending hours information in greeting")
			welcomeMessage = s.generateClosedStoreGreeting(hoursInfo)
		} else {
			// Loja aberta - mensagem normal de boas-vindas
			var err error
			welcomeMessage, err = s.settingsService.GetWelcomeMessage(ctx, tenantID)
			if err != nil {
				log.Error().Err(err).Msg("Failed to get welcome message, using fallback")
				welcomeMessage = "Oi! Como posso ajudar você hoje?"
			}
		}

		// Salvar a saudação e resposta no histórico
		s.memoryManager.AddToConversationHistory(tenantID, customerPhone, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleUser,
			Content: message,
		})
		s.memoryManager.AddToConversationHistory(tenantID, customerPhone, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: welcomeMessage,
		})

		return welcomeMessage, nil
	}

	// Preparar mensagens para o chat incluindo o contexto
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: s.getSystemPrompt(customer),
		},
	}

	// Adicionar histórico das últimas 3 interações para contexto
	if len(conversationHistory) > 0 {
		historyLimit := 6 // 3 pares de pergunta/resposta
		startIndex := 0
		if len(conversationHistory) > historyLimit {
			startIndex = len(conversationHistory) - historyLimit
		}

		for i := startIndex; i < len(conversationHistory); i++ {
			messages = append(messages, conversationHistory[i])
		}
	}

	// Adicionar mensagem atual
	userMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: message,
	}
	messages = append(messages, userMessage)

	// // 🧠 RAG Integration: Obter contexto relevante baseado na mensagem do usuário
	// log.Info().Msg("🔍 Generating RAG context for enhanced AI response")

	// // Buscar produtos similares usando RAG
	// productContext := s.getRAGProductContext(ctx, tenantID, message)

	// // 🚫 DESABILITADO: RAG de conversas causa respostas repetitivas
	// // O histórico de conversa já é gerenciado pelo memoryManager
	// // conversationContext := s.getRAGConversationContext(ctx, tenantID, customerPhone, message)

	// // Enriquecer o prompt do sistema com contexto RAG se disponível
	// if productContext != "" {
	// 	enhancedSystemPrompt := s.getSystemPrompt(customer)

	// 	if productContext != "" {
	// 		enhancedSystemPrompt += "\n\n" + productContext
	// 	}

	// 	// Atualizar a mensagem do sistema com contexto RAG
	// 	messages[0].Content = enhancedSystemPrompt

	// 	log.Info().
	// 		Bool("has_product_context", productContext != "").
	// 		Bool("has_conversation_context", false). // Desabilitado
	// 		Msg("✅ RAG context integrated into system prompt")
	// }

	// ABORDAGEM SIMPLIFICADA: UMA ÚNICA CHAMADA PARA A IA, SEM PATTERN MATCHING
	// Deixar a IA decidir quais ferramentas usar baseado em compreensão natural

	// Definir tools disponíveis
	tools := s.getAvailableTools()

	log.Info().
		Int("tools_count", len(tools)).
		Int("context_messages", len(messages)).
		Msg("Making SINGLE OpenAI API call - trusting AI to understand naturally")

	req := openai.ChatCompletionRequest{
		Model:               openai.GPT4oMini, // Modelo mais atual e inteligente
		Messages:            messages,
		Tools:               tools,
		ToolChoice:          "auto",
		MaxCompletionTokens: 8000,
	}

	// Fazer UMA ÚNICA chamada para OpenAI - deixar a IA ser inteligente
	resp, err := s.client.CreateChatCompletion(ctx, req)

	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID.String()).
			Msg("OpenAI API call failed")
		return "", fmt.Errorf("erro na chamada OpenAI: %w", err)
	}

	log.Info().
		Int("choices_count", len(resp.Choices)).
		Msg("OpenAI API call successful")

	choice := resp.Choices[0]
	var aiResponse string

	// Se há tool calls, executar (sem validações complexas)
	if len(choice.Message.ToolCalls) > 0 {
		log.Info().
			Int("tool_calls_count", len(choice.Message.ToolCalls)).
			Msg("🔧 AI chose to use tools - executing naturally")
		aiResponse, err = s.executeToolCalls(ctx, tenantID, customer.ID, customerPhone, message, choice.Message.ToolCalls)
		if err != nil {
			return "", err
		}
	} else {
		// Resposta direta da IA - ACEITAR sem validações complexas
		aiResponse = choice.Message.Content
		log.Info().
			Str("direct_response", aiResponse).
			Msg("💬 AI provided direct response - accepting naturally")
	}

	// Salvar a conversa no histórico para manter contexto
	s.memoryManager.AddToConversationHistory(tenantID, customerPhone, userMessage)
	s.memoryManager.AddToConversationHistory(tenantID, customerPhone, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: aiResponse,
	})

	// 💾 RAG Integration: Armazenar a conversa no sistema RAG para contexto futuro
	// 🚫 DESABILITADO: RAG de conversas causa respostas repetitivas
	// O histórico de conversa já é gerenciado pelo memoryManager adequadamente
	// log.Info().Msg("💾 Storing conversation in RAG for future context enhancement")
	// go s.storeConversationInRAG(ctx, tenantID, customerPhone, message, aiResponse)

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Str("response_preview", aiResponse[:minInt(100, len(aiResponse))]).
		Msg("✅ AI ProcessMessage completed successfully with RAG integration")

	return aiResponse, nil
}

// getConversationID recupera o conversation ID armazenado para a sessão
func (s *AIService) getConversationID(tenantID uuid.UUID, customerPhone string) uuid.UUID {
	sessionKey := fmt.Sprintf("%s-%s", tenantID.String(), customerPhone)
	if value, exists := s.conversationContext.Load(sessionKey); exists {
		if conversationID, ok := value.(uuid.UUID); ok {
			return conversationID
		}
	}
	return uuid.Nil
}

// clearConversationID remove o conversation ID da sessão
func (s *AIService) clearConversationID(tenantID uuid.UUID, customerPhone string) {
	sessionKey := fmt.Sprintf("%s-%s", tenantID.String(), customerPhone)
	s.conversationContext.Delete(sessionKey)
}

// ProcessImageMessage processa mensagens de imagem para análise de medicamentos
func (s *AIService) ProcessImageMessage(ctx context.Context, tenantID uuid.UUID, customerPhone, imageURL, messageID string) (string, error) {
	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Str("image_url", imageURL).
		Str("message_id", messageID).
		Msg("AI ProcessImageMessage started - analyzing image for medications")

	// Buscar ou criar cliente
	customer, err := s.customerService.GetCustomerByPhone(tenantID, customerPhone)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID.String()).
			Str("customer_phone", customerPhone).
			Msg("Failed to get customer by phone")
		return "", fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	log.Info().
		Str("customer_id", customer.ID.String()).
		Str("customer_name", customer.Name).
		Msg("Customer found for image analysis")

	// Verificar se S3 está disponível
	if s.s3Client == nil {
		log.Error().Msg("S3 storage not available - using original URL for image analysis")
		// Continuar com URL original se S3 não estiver disponível
	} else {
		// Upload da imagem para S3
		publicImageURL, err := s.uploadImageFileToS3(imageURL, tenantID.String(), customer.ID.String(), messageID)
		if err != nil {
			log.Error().
				Err(err).
				Str("original_image_url", imageURL).
				Msg("Failed to upload image to S3, using original URL")
		} else {
			log.Info().
				Str("public_image_url", publicImageURL).
				Str("message_id", messageID).
				Msg("Image successfully uploaded to S3")

			// Atualizar mensagem com URL do S3
			err = s.updateMessageWithS3URL(messageID, publicImageURL)
			if err != nil {
				log.Error().
					Err(err).
					Str("message_id", messageID).
					Str("s3_url", publicImageURL).
					Msg("Failed to update message with S3 URL")
			}

			// Usar URL do S3 para análise
			imageURL = publicImageURL
		}
	}

	// Adicionar mensagem de imagem ao histórico
	s.memoryManager.AddToConversationHistory(tenantID, customerPhone, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: "Enviei uma imagem de medicamentos/receita para análise",
	})

	// Analisar imagem com GPT-4 Vision
	log.Info().Msg("Analyzing image with GPT-4 Vision for medication detection")

	// Criar a requisição para análise visual
	req := openai.ChatCompletionRequest{
		Model:     openai.GPT4o,
		MaxTokens: 300,
		Messages: []openai.ChatCompletionMessage{
			{
				Role: openai.ChatMessageRoleUser,
				MultiContent: []openai.ChatMessagePart{
					{
						Type: openai.ChatMessagePartTypeText,
						Text: "Analise esta imagem e identifique se contém medicamentos, receita médica ou prescrição. Se encontrar nomes de medicamentos, liste-os no formato exato que aparecem na imagem (um por linha, apenas os nomes). Se não for uma imagem de medicamentos ou receita médica, responda apenas com 'NAO_MEDICAMENTO'.",
					},
					{
						Type: openai.ChatMessagePartTypeImageURL,
						ImageURL: &openai.ChatMessageImageURL{
							URL: imageURL,
						},
					},
				},
			},
		},
	}

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("Failed to analyze image with GPT-4 Vision")
		// Fallback para solicitar descrição do usuário
		response := `Não foi possível analisar a imagem automaticamente. 

Você pode me dizer quais medicamentos aparecem na imagem ou receita? 

Por exemplo:
• "Dipirona 500mg"
• "Paracetamol 750mg"

Se preferir falar com um atendente humano, digite "sim".`

		s.memoryManager.AddToConversationHistory(tenantID, customerPhone, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: response,
		})
		return response, nil
	}

	if len(resp.Choices) == 0 {
		response := `Não foi possível analisar a imagem. Você pode me dizer quais medicamentos aparecem na receita?

Se preferir falar com um atendente humano, digite "sim".`
		s.memoryManager.AddToConversationHistory(tenantID, customerPhone, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: response,
		})
		return response, nil
	}

	aiAnalysis := resp.Choices[0].Message.Content
	log.Info().Str("ai_analysis", aiAnalysis).Msg("GPT-4 Vision analysis result")

	// Verificar se não é uma imagem de medicamento
	if strings.Contains(strings.ToUpper(aiAnalysis), "NAO_MEDICAMENTO") {
		response := `Não foi possível identificar medicamentos nesta imagem.

Se você tem uma receita médica ou lista de medicamentos, você pode:
• Tentar enviar outra foto mais clara
• Digitar os nomes dos medicamentos que precisa
• Solicitar ajuda humana digitando "sim"`

		s.memoryManager.AddToConversationHistory(tenantID, customerPhone, openai.ChatCompletionMessage{
			Role:    openai.ChatMessageRoleAssistant,
			Content: response,
		})
		return response, nil
	}

	// Se medicamentos foram encontrados, buscar produtos
	log.Info().Msg("Medications detected, searching for products")

	// Extrair nomes de medicamentos da análise da IA
	medications := s.extractMedicationNames(aiAnalysis)
	log.Info().Strs("medications", medications).Msg("Extracted medication names from AI analysis")

	// Buscar produtos na base de dados para cada medicamento
	var foundProducts []models.Product
	var medicationResults []string
	productCounter := 1

	for _, medication := range medications {
		products, err := s.productService.SearchProducts(tenantID, medication, 3)
		if err != nil {
			log.Error().Err(err).Str("medication", medication).Msg("Failed to search products for medication")
			continue
		}

		if len(products) > 0 {
			foundProducts = append(foundProducts, products...)
			medicationResults = append(medicationResults, fmt.Sprintf("*%s:*", medication))
			for _, product := range products {
				price := "Consulte"
				if product.Price != "" {
					price = fmt.Sprintf("R$ %s", product.Price)
				}
				medicationResults = append(medicationResults, fmt.Sprintf("%d. %s - %s", productCounter, product.Name, price))
				productCounter++
			}
			medicationResults = append(medicationResults, "")
		}
	}

	// Construir resposta com produtos encontrados
	response := fmt.Sprintf(`📸 Analisei sua imagem e identifiquei medicamentos!

%s

💊 *Produtos encontrados em nosso catálogo:*

`, aiAnalysis)

	if len(foundProducts) > 0 {
		// Armazenar lista de produtos na memória para permitir pedidos por número
		productRefs := s.memoryManager.StoreProductList(tenantID, customerPhone, foundProducts)

		// Reconstruir a resposta usando as referências numeradas oficiais
		medicationResults = []string{}
		currentMedication := ""

		for _, ref := range productRefs {
			// Encontrar qual medicamento este produto pertence
			productMedication := ""
			for _, medication := range medications {
				if strings.Contains(strings.ToLower(ref.Name), strings.ToLower(medication)) {
					productMedication = medication
					break
				}
			}

			// Se mudou de medicamento, adicionar cabeçalho
			if productMedication != currentMedication && productMedication != "" {
				if currentMedication != "" {
					medicationResults = append(medicationResults, "")
				}
				medicationResults = append(medicationResults, fmt.Sprintf("*%s:*", productMedication))
				currentMedication = productMedication
			}

			price := "Consulte"
			if ref.Price != "" {
				price = fmt.Sprintf("R$ %s", ref.Price)
			}
			medicationResults = append(medicationResults, fmt.Sprintf("%d. %s - %s", ref.SequentialID, ref.Name, price))
		}

		response += strings.Join(medicationResults, "\n")
		response += `

💡 Para ver detalhes, diga: "produto [número]" ou "produto [nome]"
🛒 Para adicionar ao carrinho: "adicionar [número] quantidade [X]"

Exemplo: "adicionar 1 quantidade 2" ou "quero 2 unidades do produto 1"`
	} else {
		response += `Não encontrei esses medicamentos específicos em nosso catálogo no momento.

Para continuar, você pode:
• Verificar se o nome está correto e tentar novamente
• Pedir produtos similares
• Pesquisar manualmente por outros termos

Digite o nome de algum medicamento que você procura para eu buscar em nosso catálogo.`
	}

	s.memoryManager.AddToConversationHistory(tenantID, customerPhone, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleAssistant,
		Content: response,
	})

	return response, nil
}

// ProcessAudioMessage processa mensagens de áudio usando Whisper para transcrição e GPT para análise
func (s *AIService) ProcessAudioMessage(ctx context.Context, tenantID uuid.UUID, customerPhone, audioURL, messageID string) (string, error) {
	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Str("audio_url", audioURL).
		Str("message_id", messageID).
		Msg("AI ProcessAudioMessage started - transcribing and analyzing audio")

	// Buscar ou criar cliente
	customer, err := s.customerService.GetCustomerByPhone(tenantID, customerPhone)
	if err != nil {
		log.Error().
			Err(err).
			Str("tenant_id", tenantID.String()).
			Str("customer_phone", customerPhone).
			Msg("Failed to get customer by phone")
		return "", fmt.Errorf("erro ao buscar cliente: %w", err)
	}

	log.Info().
		Str("customer_id", customer.ID.String()).
		Str("customer_name", customer.Name).
		Msg("Customer found for audio analysis")

	// Verificar se S3 está disponível
	if s.s3Client == nil {
		log.Error().Msg("S3 storage not available - cannot process audio")
		return "", fmt.Errorf("serviço de storage S3 não disponível")
	}

	// Upload do arquivo de áudio para S3 (download, conversão e upload)
	publicAudioURL, err := s.uploadAudioFileToS3(audioURL, tenantID.String(), customer.ID.String(), messageID)
	if err != nil {
		log.Error().
			Err(err).
			Str("original_audio_url", audioURL).
			Msg("Failed to upload and convert audio to S3")
		return "", fmt.Errorf("erro ao processar arquivo de áudio: %w", err)
	}

	log.Info().
		Str("public_audio_url", publicAudioURL).
		Str("message_id", messageID).
		Msg("Audio successfully uploaded to S3 and ready for transcription")

	// Transcrever áudio usando OpenAI Whisper com URL pública do S3
	transcription, err := s.transcribeAudioFromURL(ctx, publicAudioURL)
	if err != nil {
		log.Error().
			Err(err).
			Str("public_audio_url", publicAudioURL).
			Msg("Failed to transcribe audio from S3")
		return "", fmt.Errorf("erro ao transcrever áudio: %w", err)
	}

	log.Info().
		Str("transcription", transcription).
		Str("message_id", messageID).
		Msg("Audio transcribed successfully - updating message content")

	// Atualizar a mensagem original com a transcrição e URL do S3
	err = s.updateMessageWithTranscriptionAndS3URL(messageID, transcription, publicAudioURL)
	if err != nil {
		log.Error().
			Err(err).
			Str("message_id", messageID).
			Str("transcription", transcription).
			Str("s3_url", publicAudioURL).
			Msg("Failed to update message with transcription and S3 URL")
		// Não falhar o processo inteiro por causa disso, apenas log
	}

	// Adicionar mensagem de áudio ao histórico
	s.memoryManager.AddToConversationHistory(tenantID, customerPhone, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: fmt.Sprintf("🎙️ [Áudio transcrito]: %s", transcription),
	})

	// Processar a transcrição como se fosse uma mensagem de texto normal
	// Isso irá usar todo o sistema RAG e busca de produtos
	response, err := s.ProcessMessage(ctx, tenantID, customerPhone, transcription)
	if err != nil {
		log.Error().
			Err(err).
			Str("transcription", transcription).
			Msg("Failed to process transcribed audio message")
		return "", fmt.Errorf("erro ao processar mensagem transcrita: %w", err)
	}

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("customer_phone", customerPhone).
		Str("transcription", transcription).
		Str("response_preview", response[:min(100, len(response))]).
		Msg("✅ AI ProcessAudioMessage completed successfully")

	return response, nil
}

// transcribeAudioFromURL usa OpenAI Whisper API para transcrever áudio de uma URL pública
func (s *AIService) transcribeAudioFromURL(ctx context.Context, audioURL string) (string, error) {
	log.Debug().
		Str("audio_url", audioURL).
		Msg("Starting audio transcription from S3 URL with OpenAI Whisper")

	// Download audio file from public URL
	resp, err := http.Get(audioURL)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar arquivo do S3 via HTTP: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("erro ao baixar arquivo do S3: status %d", resp.StatusCode)
	}

	// Criar request para Whisper API usando Reader
	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		Reader:   resp.Body,
		FilePath: "audio.mp3", // Adicionar extensão para identificação de formato
		Prompt:   "Transcreva este áudio em português brasileiro. O cliente está fazendo pedidos de produtos ou serviços.",
		Language: "pt",
	}

	// Chamar Whisper API
	respTranscription, err := s.client.CreateTranscription(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Str("audio_url", audioURL).
			Msg("OpenAI Whisper API call failed with HTTP stream")
		return "", fmt.Errorf("erro na API Whisper: %w", err)
	}

	log.Debug().
		Str("transcription", respTranscription.Text).
		Str("audio_url", audioURL).
		Msg("Audio transcription completed successfully from S3 via HTTP")

	return respTranscription.Text, nil
}

// uploadAudioFileToS3 baixa, converte e faz upload de arquivo de áudio para S3
func (s *AIService) uploadAudioFileToS3(mediaURL, tenantID, customerID, messageID string) (string, error) {
	log.Printf("Starting audio file upload process for message: %s", messageID)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "audio_conversion_")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download original file
	originalPath := filepath.Join(tempDir, "original")
	err = s.downloadFileToPath(mediaURL, originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to download audio file: %w", err)
	}

	// Convert to MP3
	convertedPath := filepath.Join(tempDir, "converted.mp3")
	err = s.convertAudioFileToMP3(originalPath, convertedPath)
	if err != nil {
		return "", fmt.Errorf("failed to convert audio file: %w", err)
	}

	// Generate S3 key with structure: tenant_id/conversations/customer_id/audio_messageID.mp3
	s3Key := fmt.Sprintf("%s/conversations/%s/audio_%s.mp3", tenantID, customerID, messageID)

	// Upload to S3
	publicURL, err := s.uploadFileToS3(convertedPath, s3Key, "audio/mp3")
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	log.Printf("Audio file successfully uploaded to S3: %s", publicURL)
	return publicURL, nil
}

// uploadImageFileToS3 faz upload de arquivo de imagem para S3
func (s *AIService) uploadImageFileToS3(mediaURL, tenantID, customerID, messageID string) (string, error) {
	log.Printf("Starting image file upload process for message: %s", messageID)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "image_upload_")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download original file
	originalPath := filepath.Join(tempDir, "original")
	err = s.downloadFileToPath(mediaURL, originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to download image file: %w", err)
	}

	// Detect content type
	file, err := os.Open(originalPath)
	if err != nil {
		return "", fmt.Errorf("failed to open downloaded file: %w", err)
	}
	defer file.Close()

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", fmt.Errorf("failed to read file for content type detection: %w", err)
	}
	file.Close()

	contentType := http.DetectContentType(buffer)
	if !strings.HasPrefix(contentType, "image/") {
		return "", fmt.Errorf("file is not an image: %s", contentType)
	}

	// Determine file extension
	var ext string
	switch contentType {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/gif":
		ext = ".gif"
	case "image/webp":
		ext = ".webp"
	default:
		ext = ".jpg" // Default fallback
	}

	// Generate S3 key with structure: tenant_id/conversations/customer_id/image_messageID.ext
	s3Key := fmt.Sprintf("%s/conversations/%s/image_%s%s", tenantID, customerID, messageID, ext)

	// Upload to S3
	publicURL, err := s.uploadFileToS3(originalPath, s3Key, contentType)
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	log.Printf("Image file successfully uploaded to S3: %s", publicURL)
	return publicURL, nil
}

// downloadFileToPath baixa um arquivo de uma URL para um caminho local
func (s *AIService) downloadFileToPath(url, filepath string) error {
	log.Printf("Downloading file from: %s", url)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Copy response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	log.Printf("File downloaded successfully to: %s", filepath)
	return nil
}

// convertAudioFileToMP3 converte arquivo de áudio para MP3 usando FFmpeg
func (s *AIService) convertAudioFileToMP3(inputPath, outputPath string) error {
	log.Printf("Converting audio file to MP3: %s -> %s", inputPath, outputPath)

	// FFmpeg command to convert to MP3
	cmd := exec.Command("ffmpeg",
		"-i", inputPath, // Input file
		"-acodec", "mp3", // Audio codec
		"-ab", "128k", // Audio bitrate
		"-ar", "44100", // Audio sample rate
		"-y",       // Overwrite output file
		outputPath, // Output file
	)

	// Capture output for debugging
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		log.Printf("FFmpeg error: %s", stderr.String())
		return fmt.Errorf("FFmpeg conversion failed: %w", err)
	}

	log.Printf("Audio conversion completed successfully")
	return nil
}

// uploadFileToS3 faz upload de um arquivo para S3 com acesso público
func (s *AIService) uploadFileToS3(filePath, s3Key, contentType string) (string, error) {
	log.Printf("Uploading file to S3: %s", s3Key)

	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info for size
	fileInfo, err := file.Stat()
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Upload to S3 without ACL (bucket should have public read policy)
	_, err = s.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:        aws.String(s.s3Bucket),
		Key:           aws.String(s3Key),
		Body:          file,
		ContentLength: aws.Int64(fileInfo.Size()),
		ContentType:   aws.String(contentType),
		// Removed ACL since bucket doesn't support it
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Build public URL
	publicURL := fmt.Sprintf("%s/%s", s.s3BaseURL, s3Key)

	log.Printf("File uploaded to S3 successfully: %s", publicURL)
	return publicURL, nil
}

// updateMessageWithTranscriptionAndS3URL atualiza uma mensagem existente concatenando a transcrição no content e atualizando a media_url
func (s *AIService) updateMessageWithTranscriptionAndS3URL(messageID, transcription, s3URL string) error {
	log.Info().
		Str("message_id", messageID).
		Str("transcription", transcription).
		Str("s3_url", s3URL).
		Msg("Updating message with audio transcription and S3 URL")

	// Buscar a mensagem existente
	message, err := s.messageService.GetMessageByID(messageID)
	if err != nil {
		return fmt.Errorf("failed to find message: %w", err)
	}

	// Preparar o novo conteúdo concatenando a transcrição
	var newContent string
	if message.Content != "" {
		// Se já há conteúdo, concatenar com quebra de linha e tag
		newContent = fmt.Sprintf("%s\n\n[transcrição]\n%s", message.Content, transcription)
	} else {
		// Se não há conteúdo, apenas adicionar a transcrição com tag
		newContent = fmt.Sprintf("[transcrição]\n%s", transcription)
	}

	// Atualizar a mensagem no banco com novo content e media_url do S3
	if err := s.messageService.UpdateMessageContentAndMediaURL(messageID, newContent, s3URL); err != nil {
		return fmt.Errorf("failed to update message content and media_url: %w", err)
	}

	log.Info().
		Str("message_id", messageID).
		Str("original_content", message.Content).
		Str("new_content", newContent).
		Str("original_media_url", message.MediaURL).
		Str("new_media_url", s3URL).
		Msg("✅ Message updated with transcription and S3 URL successfully")

	return nil
}

// updateMessageWithS3URL atualiza apenas a media_url de uma mensagem
func (s *AIService) updateMessageWithS3URL(messageID, s3URL string) error {
	log.Info().
		Str("message_id", messageID).
		Str("s3_url", s3URL).
		Msg("Updating message with S3 URL")

	// Buscar a mensagem existente
	message, err := s.messageService.GetMessageByID(messageID)
	if err != nil {
		return fmt.Errorf("failed to find message: %w", err)
	}

	// Atualizar apenas a media_url
	if err := s.messageService.UpdateMessageContentAndMediaURL(messageID, message.Content, s3URL); err != nil {
		return fmt.Errorf("failed to update message media_url: %w", err)
	}

	log.Info().
		Str("message_id", messageID).
		Str("original_media_url", message.MediaURL).
		Str("new_media_url", s3URL).
		Msg("✅ Message updated with S3 URL successfully")

	return nil
}

// updateMessageWithTranscription atualiza uma mensagem existente concatenando a transcrição no content
func (s *AIService) updateMessageWithTranscription(messageID, transcription string) error {
	log.Info().
		Str("message_id", messageID).
		Str("transcription", transcription).
		Msg("Updating message with audio transcription")

	// Buscar a mensagem existente
	message, err := s.messageService.GetMessageByID(messageID)
	if err != nil {
		return fmt.Errorf("failed to find message: %w", err)
	}

	// Preparar o novo conteúdo concatenando a transcrição
	var newContent string
	if message.Content != "" {
		// Se já há conteúdo, concatenar com quebra de linha e tag
		newContent = fmt.Sprintf("%s\n\n[transcrição]\n%s", message.Content, transcription)
	} else {
		// Se não há conteúdo, apenas adicionar a transcrição com tag
		newContent = fmt.Sprintf("[transcrição]\n%s", transcription)
	}

	// Atualizar a mensagem no banco
	if err := s.messageService.UpdateMessageContent(messageID, newContent); err != nil {
		return fmt.Errorf("failed to update message content: %w", err)
	}

	log.Info().
		Str("message_id", messageID).
		Str("original_content", message.Content).
		Str("new_content", newContent).
		Msg("✅ Message updated with transcription successfully")

	return nil
}

// transcribeAudio usa a OpenAI Whisper API para transcrever áudio
func (s *AIService) transcribeAudio(ctx context.Context, audioURL string) (string, error) {
	log.Debug().
		Str("audio_url", audioURL).
		Msg("Starting audio transcription with OpenAI Whisper")

	// Fazer download do arquivo de áudio
	audioData, err := s.downloadAudioFile(audioURL)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar arquivo de áudio: %w", err)
	}

	// Criar request para Whisper API
	req := openai.AudioRequest{
		Model:    openai.Whisper1,
		Reader:   audioData,
		Prompt:   "Transcreva este áudio em português brasileiro. O cliente está fazendo pedidos de produtos ou serviços.",
		Language: "pt",
	}

	// Chamar Whisper API
	resp, err := s.client.CreateTranscription(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Msg("OpenAI Whisper API call failed")
		return "", fmt.Errorf("erro na API Whisper: %w", err)
	}

	log.Debug().
		Str("transcription", resp.Text).
		Msg("Audio transcription completed")

	return resp.Text, nil
}

// downloadAudioFile faz download do arquivo de áudio da URL e converte se necessário
func (s *AIService) downloadAudioFile(audioURL string) (*strings.Reader, error) {
	log.Debug().
		Str("audio_url", audioURL).
		Msg("Downloading audio file")

	// Criar HTTP client
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: false,
			},
		},
	}

	// Fazer request
	resp, err := client.Get(audioURL)
	if err != nil {
		return nil, fmt.Errorf("erro ao fazer download do áudio: %w", err)
	}
	defer resp.Body.Close()

	// Verificar status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro HTTP ao baixar áudio: %d", resp.StatusCode)
	}

	// Ler conteúdo do arquivo original
	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler conteúdo do áudio: %w", err)
	}

	log.Debug().
		Int("audio_size_bytes", len(audioBytes)).
		Msg("Audio file downloaded successfully")

	// Detectar formato do arquivo pela URL ou Content-Type
	needsConversion := s.needsAudioConversion(audioURL, resp.Header.Get("Content-Type"))

	if needsConversion {
		log.Debug().
			Str("original_format", s.getAudioFormatFromURL(audioURL)).
			Msg("Audio needs conversion to compatible format")

		// Converter para MP3 usando FFmpeg
		convertedBytes, err := s.convertAudioToMP3(audioBytes)
		if err != nil {
			log.Warn().
				Err(err).
				Str("audio_url", audioURL).
				Msg("Failed to convert audio, trying original format anyway")

			// Se conversão falhar, tenta usar arquivo original mesmo assim
			return strings.NewReader(string(audioBytes)), nil
		}

		log.Debug().
			Int("converted_size_bytes", len(convertedBytes)).
			Msg("Audio converted to MP3 successfully")

		return strings.NewReader(string(convertedBytes)), nil
	}

	// Arquivo já está em formato suportado
	return strings.NewReader(string(audioBytes)), nil
}

// needsAudioConversion verifica se o arquivo precisa ser convertido
func (s *AIService) needsAudioConversion(audioURL, contentType string) bool {
	// Formatos suportados pelo Whisper
	supportedFormats := map[string]bool{
		"mp3":  true,
		"mp4":  true,
		"mpeg": true,
		"mpga": true,
		"m4a":  true,
		"wav":  true,
		"webm": true,
		"flac": true,
		"ogg":  true, // OGG é suportado, mas .oga às vezes não funciona
	}

	// Verificar pela extensão da URL
	format := s.getAudioFormatFromURL(audioURL)
	if format == "oga" || format == "opus" {
		return true // .oga e .opus precisam conversão
	}

	// Verificar se formato é suportado
	if !supportedFormats[format] {
		return true
	}

	// Verificar pelo Content-Type se disponível
	if contentType != "" {
		if strings.Contains(contentType, "ogg") && strings.Contains(audioURL, ".oga") {
			return true // OGA específico precisa conversão
		}
	}

	return false
}

// getAudioFormatFromURL extrai a extensão do arquivo da URL
func (s *AIService) getAudioFormatFromURL(audioURL string) string {
	// Encontrar última extensão na URL
	parts := strings.Split(audioURL, ".")
	if len(parts) < 2 {
		return "unknown"
	}

	// Pegar extensão e remover query parameters
	ext := parts[len(parts)-1]
	if questionIdx := strings.Index(ext, "?"); questionIdx != -1 {
		ext = ext[:questionIdx]
	}

	return strings.ToLower(ext)
}

// convertAudioToMP3 converte áudio para MP3 usando FFmpeg
func (s *AIService) convertAudioToMP3(audioBytes []byte) ([]byte, error) {
	// Criar arquivos temporários
	tempDir := os.TempDir()
	inputFile := filepath.Join(tempDir, fmt.Sprintf("audio_input_%d", time.Now().UnixNano()))
	outputFile := filepath.Join(tempDir, fmt.Sprintf("audio_output_%d.mp3", time.Now().UnixNano()))

	// Limpar arquivos temporários no final
	defer func() {
		os.Remove(inputFile)
		os.Remove(outputFile)
	}()

	// Escrever bytes do áudio original para arquivo temporário
	err := os.WriteFile(inputFile, audioBytes, 0644)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar arquivo temporário: %w", err)
	}

	// Executar FFmpeg para conversão
	cmd := exec.Command("ffmpeg",
		"-i", inputFile, // arquivo de entrada
		"-acodec", "mp3", // codec de áudio MP3
		"-ar", "16000", // sample rate otimizado para Whisper
		"-ac", "1", // mono channel
		"-b:a", "32k", // bitrate baixo para economia
		"-y",       // sobrescrever arquivo de saída
		outputFile, // arquivo de saída
	)

	// Capturar stderr para logs de erro
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	log.Debug().
		Str("command", cmd.String()).
		Msg("Executing FFmpeg conversion")

	// Executar comando
	err = cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("erro na conversão FFmpeg: %w, stderr: %s", err, stderr.String())
	}

	// Ler arquivo convertido
	convertedBytes, err := os.ReadFile(outputFile)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo convertido: %w", err)
	}

	if len(convertedBytes) == 0 {
		return nil, fmt.Errorf("arquivo convertido está vazio")
	}

	log.Debug().
		Int("original_size", len(audioBytes)).
		Int("converted_size", len(convertedBytes)).
		Msg("Audio conversion completed successfully")

	return convertedBytes, nil
}

func (s *AIService) getSystemPrompt(customer *models.Customer) string {
	ctx := context.Background()

	log.Info().
		Str("tenant_id", customer.TenantID.String()).
		Msg("Getting system prompt - checking for custom prompt")

	// Verificar se existe um prompt personalizado configurado
	customPromptSetting, err := s.settingsService.GetSetting(ctx, customer.TenantID, "ai_system_prompt_template")
	if err == nil && customPromptSetting.SettingValue != nil && *customPromptSetting.SettingValue != "" {
		// Usar prompt personalizado se configurado
		log.Info().Msg("Using custom prompt template")
		return s.processCustomPrompt(*customPromptSetting.SettingValue, customer)
	}

	log.Info().Msg("No custom prompt found, generating business-specific prompt")

	// Obter informações específicas do tenant baseadas nos produtos
	businessInfo, err := s.getBusinessInfo(ctx, customer.TenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", customer.TenantID.String()).Msg("Failed to get business info")
		businessInfo = &BusinessInfo{
			Type:        "loja",
			Description: "uma loja online",
			Examples:    []string{"quero 3 produtos", "adicionar 2 itens", "comprar 1 unidade"},
		}
	}

	log.Info().
		Str("business_type", businessInfo.Type).
		Str("business_desc", businessInfo.Description).
		Msg("Business info obtained successfully")

	// Construir string de exemplos
	var exampleText string
	for _, example := range businessInfo.Examples {
		exampleText += fmt.Sprintf("• \"%s\"\n", example)
	}

	// Buscar informações de horários de funcionamento
	log.Info().Str("tenant_id", customer.TenantID.String()).Msg("🕐 Chamando getBusinessHoursInfo")
	hoursInfo := s.getBusinessHoursInfo(ctx, customer.TenantID)
	log.Info().Str("hours_info", hoursInfo).Msg("🕐 Resultado getBusinessHoursInfo")

	// Incluir informações de horário no prompt
	hoursSection := ""
	if hoursInfo != "" {
		hoursSection = fmt.Sprintf("\n\nHORÁRIOS DE FUNCIONAMENTO:\n%s\n", hoursInfo)
	}

	// Buscar limitação de contexto personalizada
	contextLimitationSection := s.getContextLimitationSection(ctx, customer.TenantID)

	return fmt.Sprintf(`Você é um assistente de vendas inteligente para %s via WhatsApp.

CLIENTE: %s (ID: %s, Telefone: %s)

SOBRE O NEGÓCIO:
- Tipo: %s
- Especialidade: %s%s

SUAS CAPACIDADES:
🔍 Pesquisar produtos no catálogo
🛒 Gerenciar carrinho de compras  
📦 Processar pedidos e checkout
📍 Verificar entregas e endereços
� Atualizar dados do cliente

COMPORTAMENTO NATURAL:
- Seja conversacional e amigável
- Entenda a intenção do cliente, não apenas palavras específicas
- Use as ferramentas disponíveis quando apropriado
- Para produtos: sempre consulte nosso banco de dados primeiro
- Para checkout: SEMPRE use a função 'checkout' quando cliente quiser finalizar
- Se a loja estiver fechada, informe educadamente os horários de funcionamento
- Se estiver aberto, atenda normalmente e processe pedidos
- Se o cliente se apresentar com seu nome, use atualizarCadastro para salvar
- Para personalizar o atendimento, pergunte o nome do cliente se ainda não souber

🎯 REGRAS OBRIGATÓRIAS DE ORDENAÇÃO/PREÇOS:
🚨 SEMPRE use 'consultarItens' para perguntas sobre preços e ordenação:
- "produto mais caro" → use query: "[categoria/produto]" + ordenar_por: "preco_maior" + limite: 1
- "produto mais barato" → use query: "[categoria/produto]" + ordenar_por: "preco_menor" + limite: 1
- "menor preço" → use query: "[categoria/produto]" + ordenar_por: "preco_menor" + limite: 1
- "maior preço" → use query: "[categoria/produto]" + ordenar_por: "preco_maior" + limite: 1
- "nimesulida mais cara" → use query: "nimesulida" + ordenar_por: "preco_maior" + limite: 1
- "ordena por preço" → use query: "[categoria]" + ordenar_por apropriado
🎯 SEMPRE inclua o parâmetro 'query' com o produto/categoria mencionado pelo cliente
🎯 Para perguntas ESPECÍFICAS sobre UM produto (mais caro/barato), SEMPRE use limite: 1
🎯 Para comparações ou listas de produtos ordenados, use limite maior
🚨 NUNCA responda diretamente sobre preços sem usar a função
🚨 NUNCA invente ou estime preços baseado em memória

%s

REGRAS OBRIGATÓRIAS DE CHECKOUT:
🚨 NUNCA pergunte sobre endereço antes de usar a função 'checkout'
🚨 SEMPRE use 'checkout' para frases como: 'pronto', 'finalizar', 'fechar pedido', 'pode enviar'
🚨 A função 'checkout' verifica automaticamente os endereços cadastrados
🚨 NUNCA gere respostas sobre endereço sem usar as ferramentas

REGRAS DE CONFIRMAÇÃO DE ENDEREÇO - SUPER IMPORTANTE:
🎯 APÓS mostrar endereço, use 'finalizarPedido' para: 'sim', 'confirmar', 'confirmo', 'ok', 'isso', 'está certo', 'correto'
🎯 NUNCA repita pergunta de endereço - se cliente confirmar, FINALIZE IMEDIATAMENTE
🎯 CONTEXTO: Se sua última mensagem perguntou sobre endereço e cliente responde positivamente = USAR finalizarPedido
🎯 EXEMPLO: IA pergunta endereço → Cliente diz "sim" → OBRIGATÓRIO usar finalizarPedido
🚨 NUNCA use 'checkout' quando cliente está confirmando endereço - isso cria loop infinito!

DIFERENÇA ENTRE FUNÇÕES:
- 'checkout' = Primeira vez que cliente quer finalizar (ainda não mostrou endereço)
- 'finalizarPedido' = Cliente já viu endereço e confirmou (criar pedido efetivamente)

EXEMPLOS NATURAIS:
- "Oi, bom dia!" → Responda naturalmente e ofereça ajuda
- "Quero comprar sabonete" → Pesquise sabonetes no catálogo
- "Adicione 3 ao carrinho" → Adicione baseado no contexto da conversa
- "Finalizar pedido" → Use checkout para processar
- "Pronto, somente isso" → Use checkout para processar
- Cliente vê endereço e diz "sim" → Use finalizarPedido IMEDIATAMENTE
- "Vocês entregam em Vila Madalena?" → Verifique entrega para o endereço

REGRA SIMPLES: Confie na sua inteligência para entender o que o cliente quer e usar as ferramentas certas.`,
		businessInfo.Description,
		customer.Name, customer.ID, customer.Phone,
		businessInfo.Type,
		businessInfo.Description,
		hoursSection,
		contextLimitationSection,
	)
}

// processCustomPrompt processes a custom prompt template with customer variables
func (s *AIService) processCustomPrompt(template string, customer *models.Customer) string {
	// Replace placeholders in custom prompt
	processed := strings.ReplaceAll(template, "{{customer_name}}", customer.Name)
	processed = strings.ReplaceAll(processed, "{{customer_id}}", customer.ID.String())
	processed = strings.ReplaceAll(processed, "{{customer_phone}}", customer.Phone)

	// Generate dynamic examples if placeholder exists
	if strings.Contains(processed, "{{product_examples}}") {
		ctx := context.Background()
		examples, err := s.settingsService.GenerateAIProductExamples(ctx, customer.TenantID)
		if err != nil {
			examples = []string{"quero 3 produtos", "adicionar 2 itens", "comprar 1 unidade"}
		}

		var exampleText string
		for _, example := range examples {
			exampleText += fmt.Sprintf("• \"%s\"\n", example)
		}

		processed = strings.ReplaceAll(processed, "{{product_examples}}", exampleText)
	}

	return processed
}

// getContextLimitationSection retorna a seção de limitação de contexto personalizada ou padrão
func (s *AIService) getContextLimitationSection(ctx context.Context, tenantID uuid.UUID) string {
	// Buscar limitação de contexto personalizada
	customLimitationSetting, err := s.settingsService.GetSetting(ctx, tenantID, "ai_context_limitation_custom")
	if err == nil && customLimitationSetting.SettingValue != nil && *customLimitationSetting.SettingValue != "" {
		log.Info().Str("tenant_id", tenantID.String()).Msg("Using custom context limitation")
		return *customLimitationSetting.SettingValue
	}

	// Retornar limitação padrão se não há customização
	log.Info().Str("tenant_id", tenantID.String()).Msg("Using default context limitation")
	return `🚨 LIMITAÇÃO DE CONTEXTO - SUPER IMPORTANTE:
- Você é um ASSISTENTE DE VENDAS, não um assistente geral
- NUNCA responda perguntas sobre: política, notícias, medicina, direito, aposentadoria, educação, tecnologia geral, ou qualquer assunto não relacionado à nossa loja
- Para perguntas fora do contexto, responda: "Sou um assistente focado em vendas da nossa loja. Como posso ajudá-lo com nossos produtos ou serviços?"
- SEMPRE redirecione conversas para produtos, pedidos, entregas ou informações da loja
- Sua função é EXCLUSIVAMENTE ajudar com vendas e atendimento comercial`
}

// isSimpleGreeting checks if the message is a simple greeting without specific requests
func (s *AIService) isSimpleGreeting(message string) bool {
	normalizedMessage := strings.ToLower(strings.TrimSpace(message))

	// Padrões de saudações simples (sem solicitações específicas)
	greetingPatterns := []string{
		"oi", "olá", "hello", "hi",
		"bom dia", "boa tarde", "boa noite",
		"e aí", "eai", "opa", "hey",
		"tudo bem", "tudo certo", "como vai",
		"oi tudo bem", "ola tudo bem",
		"oi bom dia", "olá bom dia",
		"oi boa tarde", "olá boa tarde",
	}

	// Verificar se é uma saudação simples (sem solicitações de produtos)
	for _, pattern := range greetingPatterns {
		if normalizedMessage == pattern {
			return true
		}
	}

	// Se contém palavras de produtos/compras, não é saudação simples
	productWords := []string{
		"quero", "comprar", "preciso", "produto", "item",
		"adicionar", "carrinho", "pedido", "preço", "valor",
	}

	for _, word := range productWords {
		if strings.Contains(normalizedMessage, word) {
			return false
		}
	}

	return false
}

// getBusinessInfo pega informações do negócio do tenant configurado no super admin
func (s *AIService) getBusinessInfo(ctx context.Context, tenantID uuid.UUID) (*BusinessInfo, error) {
	// Buscar informações do tenant no banco
	tenant, err := s.getTenantInfo(tenantID)
	if err != nil {
		log.Error().Err(err).Str("tenant_id", tenantID.String()).Msg("Failed to get tenant info")
		return &BusinessInfo{
			Type:        "loja",
			Description: "uma loja online",
			Examples:    []string{"quero 3 produtos", "adicionar 2 itens", "comprar 1 unidade"},
			Categories:  []string{},
		}, nil
	}

	log.Info().
		Str("tenant_id", tenantID.String()).
		Str("business_category", tenant.BusinessCategory).
		Str("about", tenant.About).
		Msg("Using tenant business info")

	// Usar categoria configurada no super admin
	businessType := tenant.BusinessCategory
	if businessType == "" {
		businessType = "loja" // fallback padrão
	}

	// Usar descrição do campo "About" da loja
	businessDesc := tenant.About
	if businessDesc == "" {
		businessDesc = "uma " + businessType // fallback usando a categoria
	}

	// Gerar exemplos específicos do negócio baseados no tipo
	examples := s.generateExamplesByBusinessType(businessType)

	// Buscar categorias de produtos reais do tenant (mantém funcionalidade existente)
	products, err := s.productService.SearchProducts(tenantID, "", 50)
	var topCategories []string
	if err == nil && len(products) > 0 {
		categories := make(map[string]int)
		for _, product := range products {
			if product.Tags != "" {
				tags := strings.Split(strings.ToLower(product.Tags), ",")
				for _, tag := range tags {
					tag = strings.TrimSpace(tag)
					if tag != "" {
						categories[tag]++
					}
				}
			}
		}

		// Extrair principais categorias
		for category, count := range categories {
			if count >= 2 { // Só incluir categorias com pelo menos 2 produtos
				topCategories = append(topCategories, category)
			}
		}
	}

	return &BusinessInfo{
		Type:        businessType,
		Description: businessDesc,
		Examples:    examples,
		Categories:  topCategories,
	}, nil
}

// generateExamplesByBusinessType gera exemplos específicos por tipo de negócio
func (s *AIService) generateExamplesByBusinessType(businessType string) []string {
	switch strings.ToLower(businessType) {
	case "farmacia", "farmácia":
		return []string{
			"preciso de dipirona",
			"tem paracetamol?",
			"quero 2 caixas de dorflex",
			"medicamento para dor de cabeça",
		}
	case "hamburgeria":
		return []string{
			"quero um hambúrguer",
			"qual o combo mais barato?",
			"um x-bacon com batata",
			"hambúrguer artesanal",
		}
	case "pizzaria":
		return []string{
			"pizza marguerita grande",
			"quero uma calabresa",
			"pizza doce de chocolate",
			"qual o sabor mais pedido?",
		}
	case "acaiteria", "açaiteria":
		return []string{
			"açaí de 500ml",
			"tigela de açaí com granola",
			"açaí com leite condensado",
			"bowl de açaí",
		}
	case "restaurante":
		return []string{
			"prato feito",
			"qual o prato do dia?",
			"quero um almoço",
			"cardápio de hoje",
		}
	default:
		return []string{
			"quero 3 produtos",
			"adicionar 2 itens",
			"comprar 1 unidade",
			"ver catálogo",
		}
	}
}

// determineBusinessType analisa palavras-chave para determinar o tipo de negócio
func (s *AIService) determineBusinessType(keywords, categories map[string]int) (string, string) {
	// Sistema genérico - não usar exemplos fixos de produtos
	// Retornar tipo genérico baseado apenas nos dados reais do tenant
	return "loja", "uma loja"
}

// isCommonWordBusiness verifica se uma palavra é muito comum e deve ser ignorada
func isCommonWordBusiness(word string) bool {
	commonWords := map[string]bool{
		"de": true, "da": true, "do": true, "com": true, "para": true, "em": true,
		"um": true, "uma": true, "o": true, "a": true, "e": true, "ou": true,
		"kit": true, "pack": true, "conjunto": true, "unidade": true, "pacote": true,
		"ml": true, "mg": true, "gr": true, "kg": true, "lt": true,
		// Palavras de checkout/finalização que não são produtos
		"finalizar": true, "fechar": true, "fecha": true, "fechando": true,
		"pedido": true, "compra": true, "checkout": true,
		"confirmar": true, "confirma": true, "concluir": true, "conclui": true,
		"terminar": true, "termina": true, "acabar": true, "acaba": true,
		// Saudações e conversação que não são produtos
		"olá": true, "ola": true, "oi": true, "bom": true, "boa": true, "tarde": true, "dia": true,
		"noite": true, "gostaria": true, "quero": true, "preciso": true, "fazer": true, "ver": true,
		"produtos": true, "cardapio": true, "catálogo": true, "catalogo": true, "medicamentos": true, "remedios": true,
		"obrigado": true, "obrigada": true, "tchau": true, "até": true, "logo": true,
	}
	return commonWords[word]
}

func (s *AIService) getAvailableTools() []openai.Tool {
	return []openai.Tool{
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "consultarItens",
				Description: "🛍️ OBRIGATÓRIO E PRIORITÁRIO: Use SEMPRE PRIMEIRO quando cliente mencionar 'produtos', 'cardápio', 'menu', 'catálogo', 'itens', 'o que vocês têm', ou qualquer termo relacionado a ver produtos disponíveis. 🎯 OBRIGATÓRIO para ordenação/comparação: Use SEMPRE para perguntas sobre 'produto mais caro', 'mais barato', 'menor preço', 'maior preço', 'ordena por preço' - SEMPRE inclua o parâmetro 'query' com o produto/categoria mencionado pelo cliente (ex: query: 'nimesulida'), use 'ordenar_por' apropriado ('preco_maior' ou 'preco_menor') e 'limite: 1' para perguntas ESPECÍFICAS sobre UM produto. NUNCA invente produtos ou preços - use SEMPRE esta função para buscar produtos reais do banco de dados do tenant. Para consultas genéricas (cardápio, produtos), use query vazia ou 'produtos' para retornar catálogo completo organizado por categorias. NUNCA responda sobre produtos sem usar esta função primeiro.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Palavra-chave para buscar produtos (nome, descrição, SKU, marca, tags)",
						},
						"marca": map[string]interface{}{
							"type":        "string",
							"description": "Filtrar por marca específica (ex: 'Faber-Castell', 'BIC')",
						},
						"tags": map[string]interface{}{
							"type":        "string",
							"description": "Filtrar por tags/categoria (ex: 'caneta', 'papelaria', 'escolar')",
						},
						"preco_min": map[string]interface{}{
							"type":        "number",
							"description": "Preço mínimo para filtrar produtos",
						},
						"preco_max": map[string]interface{}{
							"type":        "number",
							"description": "Preço máximo para filtrar produtos",
						},
						"promocional": map[string]interface{}{
							"type":        "boolean",
							"description": "Se true, retorna apenas produtos em promoção",
						},
						"limite": map[string]interface{}{
							"type":        "integer",
							"description": "Número máximo de produtos para retornar (padrão: 10)",
						},
						"ordenar_por": map[string]interface{}{
							"type":        "string",
							"description": "Critério de ordenação: 'preco_menor' (mais barato primeiro), 'preco_maior' (mais caro primeiro), 'nome' (alfabética), 'relevancia' (padrão)",
						},
					},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "mostrarOpcoesCategoria",
				Description: "📂 Use quando cliente perguntar sobre categorias específicas ou tipos de produtos como 'que tipos de X vocês têm?', 'quais opções de Y?'. Esta função busca produtos reais do banco de dados e armazena na memória para permitir seleção por número. NUNCA responda diretamente sobre categorias - sempre use esta função para buscar produtos reais.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"categoria": map[string]interface{}{
							"type":        "string",
							"description": "Categoria ou tipo de produto mencionado pelo cliente (use exatamente as palavras que o cliente usou)",
						},
						"limite": map[string]interface{}{
							"type":        "integer",
							"description": "Número máximo de opções para mostrar (padrão: 5)",
						},
					},
					"required": []string{"categoria"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "detalharItem",
				Description: "Obtém detalhes completos de um produto específico pelo número sequencial, nome ou ID",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"identifier": map[string]interface{}{
							"type":        "string",
							"description": "Número sequencial (ex: '1'), nome parcial (ex: 'sabonete') ou ID do produto",
						},
					},
					"required": []string{"identifier"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "adicionarAoCarrinho",
				Description: "Adiciona um produto ao carrinho do cliente pelo número sequencial ou ID",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"identifier": map[string]interface{}{
							"type":        "string",
							"description": "Número sequencial (ex: '1') ou ID do produto",
						},
						"quantidade": map[string]interface{}{
							"type":        "integer",
							"description": "Quantidade do produto",
							"minimum":     1,
						},
					},
					"required": []string{"identifier", "quantidade"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "buscarMultiplosProdutos",
				Description: "🎯 USE SOMENTE quando usuário mencionar MÚLTIPLOS produtos ESPECÍFICOS na mesma mensagem com nomes exatos fornecidos pelo usuário. Ex: 'quero sabonete dove e shampoo head shoulders', 'preciso de caneta bic e papel A4'. NUNCA use para consultas genéricas como 'produtos', 'cardápio', 'menu', 'catálogo' - para isso use SEMPRE 'consultarItens'. NUNCA invente nomes de produtos - use APENAS os nomes que o usuário forneceu explicitamente.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"produtos": map[string]interface{}{
							"type":        "array",
							"description": "Lista de produtos mencionados pelo usuário",
							"items": map[string]interface{}{
								"type": "string",
							},
						},
						"quantidade": map[string]interface{}{
							"type":        "integer",
							"description": "Quantidade para cada produto (padrão: 1)",
							"minimum":     1,
						},
					},
					"required": []string{"produtos"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "adicionarProdutoPorNome",
				Description: "🚨 FUNÇÃO PRINCIPAL: Use SEMPRE que o usuário mencionar UM ÚNICO produto + quantidade. Ex: 'quero 3 sabonetes', 'adicione 2 shampoos', 'comprar 1 perfume'",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"nome_produto": map[string]interface{}{
							"type":        "string",
							"description": "Nome ou palavra-chave do produto mencionado pelo usuário (ex: 'sabonete', 'shampoo', 'perfume')",
						},
						"quantidade": map[string]interface{}{
							"type":        "integer",
							"description": "Quantidade solicitada pelo usuário",
							"minimum":     1,
						},
					},
					"required": []string{"nome_produto", "quantidade"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "removerDoCarrinho",
				Description: "Remove um item específico do carrinho pelo número do item",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"item_number": map[string]interface{}{
							"type":        "integer",
							"description": "Número do item no carrinho (1, 2, 3, etc.)",
						},
						"item_id": map[string]interface{}{
							"type":        "string",
							"description": "ID do item no carrinho (fallback para compatibilidade)",
						},
					},
					"required": []string{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "verCarrinho",
				Description: "Mostra todos os itens no carrinho atual do cliente",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "limparCarrinho",
				Description: "Remove todos os itens do carrinho",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "checkout",
				Description: "🛒 OBRIGATÓRIO: SEMPRE use esta função quando cliente quiser finalizar pedido ('pronto', 'finalizar', 'fechar pedido', 'pode enviar', 'quero finalizar'). Esta função verifica automaticamente endereços cadastrados e mostra para o cliente. NUNCA pergunte sobre endereço antes de usar esta função. Use SEMPRE que detectar intenção de finalização.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "finalizarPedido",
				Description: "🚀 FINALIZAR PEDIDO OBRIGATÓRIO: Use IMEDIATAMENTE quando cliente confirmar endereço após ver pergunta de endereço ('sim', 'confirmar', 'confirmo', 'isso mesmo', 'está certo', 'ok', 'pode enviar', 'correto'). NUNCA repetir pergunta de endereço - cliente confirmou = usar esta função AGORA. REGRA: se última mensagem da IA foi sobre endereço e cliente responde positivamente, SEMPRE use esta função. Cria pedido final e fecha venda.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "cancelarPedido",
				Description: "Cancela um pedido específico. Aceita número sequencial (1, 2, 3...) mostrado no histórico ou código do pedido (ex: ORD-123)",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"order_id": map[string]interface{}{
							"type":        "string",
							"description": "Número sequencial do pedido (1, 2, 3...) ou código do pedido mostrado no histórico",
						},
					},
					"required": []string{"order_id"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "historicoPedidos",
				Description: "Lista o histórico de pedidos do cliente",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "adicionarPorNumero",
				Description: "🔢 Use quando cliente responder apenas com um NÚMERO após ver uma lista de produtos. Ex: cliente vê lista e responde '3' para adicionar produto número 3",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"numero": map[string]interface{}{
							"type":        "integer",
							"description": "Número do produto na lista mostrada anteriormente",
						},
						"quantidade": map[string]interface{}{
							"type":        "integer",
							"description": "Quantidade a adicionar (padrão: 1)",
							"minimum":     1,
						},
					},
					"required": []string{"numero"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "adicionarMaisItemCarrinho",
				Description: "🎯 Use quando cliente quiser adicionar mais unidades de um produto que JÁ ESTÁ no carrinho. Ex: 'essa agenda vou querer mais 3', 'quero mais 2 desse marca texto'",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"produto_nome": map[string]interface{}{
							"type":        "string",
							"description": "Nome ou parte do nome do produto mencionado pelo cliente",
						},
						"quantidade_adicional": map[string]interface{}{
							"type":        "integer",
							"description": "Quantidade adicional que o cliente quer",
							"minimum":     1,
						},
					},
					"required": []string{"produto_nome", "quantidade_adicional"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "atualizarQuantidade",
				Description: "Atualiza a quantidade de um item específico no carrinho pelo número do item. Use quando o cliente quiser alterar a quantidade de um produto já adicionado",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"item_number": map[string]interface{}{
							"type":        "integer",
							"description": "Número do item no carrinho (1, 2, 3, etc.)",
						},
						"item_id": map[string]interface{}{
							"type":        "string",
							"description": "ID do item no carrinho (fallback para compatibilidade)",
						},
						"quantidade": map[string]interface{}{
							"type":        "integer",
							"description": "Nova quantidade do item",
							"minimum":     1,
						},
					},
					"required": []string{"quantidade"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "atualizarCadastro",
				Description: "Atualiza dados do cliente. Use quando: 1) Cliente fornecer informações pessoais (nome, email), 2) Cliente fornecer novo endereço COMPLETO (rua + número + bairro/cidade), 3) Cliente disser 'novo endereço', 4) Cliente quiser deletar endereços ('delete todos', 'apagar endereço'). SEMPRE use esta função para endereços completos que contenham rua, número e localização (ex: 'Av Hugo musso, 2380, Itapua, Vila Velha, ES'). NUNCA use para confirmações simples como 'sim', 'confirmo', 'ok', 'finalizar' - essas são para checkout.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"nome": map[string]interface{}{
							"type":        "string",
							"description": "Nome completo do cliente quando fornecido",
						},
						"email": map[string]interface{}{
							"type":        "string",
							"description": "Email do cliente quando fornecido",
						},
						"endereco": map[string]interface{}{
							"type":        "string",
							"description": "Endereço completo do cliente (rua, número, bairro, cidade, estado, CEP) quando cliente fornecer um endereço novo para cadastro",
						},
					},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "gerenciarEnderecos",
				Description: "Lista, seleciona, deleta ou gerencia endereços EXISTENTES do cliente. Use quando: 1) Cliente pedir para ver endereços ('meus endereços', 'ver endereços', 'vocês já tem meu endereço', 'vcs tem meu endereço'), 2) Cliente responder com NÚMERO ISOLADO para selecionar endereço (ex: '1', '2', '3'), 3) Cliente quiser deletar endereço específico, 4) Cliente pedir para listar endereços, 5) Cliente quiser mudar/alterar/trocar endereço ('muda o endereço', 'altera meu endereço', 'trocar endereço', 'mudar endereço'). NUNCA use esta função quando cliente fornecer endereço completo com rua+número+localização - isso é cadastro novo.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"acao": map[string]interface{}{
							"type":        "string",
							"description": "Ação a ser executada: 'listar' (mostrar endereços), 'selecionar' (escolher endereço por número), 'deletar' (remover endereço específico), 'deletar_todos' (remover todos os endereços)",
							"enum":        []string{"listar", "selecionar", "deletar", "deletar_todos"},
						},
						"numero_endereco": map[string]interface{}{
							"type":        "integer",
							"description": "Número do endereço para selecionar ou deletar (usado apenas com ações 'selecionar' e 'deletar')",
							"minimum":     1,
						},
					},
					"required": []string{"acao"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "cadastrarEndereco",
				Description: "Cadastra um novo endereço quando cliente fornecer endereço completo com rua, número, bairro, cidade, estado e CEP. Use quando cliente informar endereço completo após solicitar finalização de pedido. Ex: 'Rua das Flores, 123, Centro, Brasília, DF, CEP 70000-000'",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"endereco_completo": map[string]interface{}{
							"type":        "string",
							"description": "Endereço completo fornecido pelo cliente incluindo rua, número, bairro, cidade, estado e CEP",
						},
						"rua": map[string]interface{}{
							"type":        "string",
							"description": "Nome da rua ou avenida",
						},
						"numero": map[string]interface{}{
							"type":        "string",
							"description": "Número do endereço",
						},
						"bairro": map[string]interface{}{
							"type":        "string",
							"description": "Nome do bairro",
						},
						"cidade": map[string]interface{}{
							"type":        "string",
							"description": "Nome da cidade",
						},
						"estado": map[string]interface{}{
							"type":        "string",
							"description": "Sigla do estado (ex: SP, RJ, ES)",
						},
						"cep": map[string]interface{}{
							"type":        "string",
							"description": "CEP do endereço",
						},
					},
					"required": []string{"endereco_completo", "rua", "numero", "bairro", "cidade", "estado", "cep"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "verificarEntrega",
				Description: "Verifica se a loja faz entregas em um determinado bairro, região ou endereço. Use quando cliente perguntar sobre entrega ('vocês entregam?', 'fazem entrega?', 'atendem minha região?', etc.)",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"local": map[string]interface{}{
							"type":        "string",
							"description": "Local mencionado pelo cliente (bairro, região, endereço). Ex: 'Vila Madalena', 'Praia da Costa', 'Centro'",
						},
						"cidade": map[string]interface{}{
							"type":        "string",
							"description": "Cidade mencionada pelo cliente, se específica. Caso não tenha, deixe vazio para usar a cidade da loja",
						},
						"estado": map[string]interface{}{
							"type":        "string",
							"description": "Estado mencionado pelo cliente, se específico. Caso não tenha, deixe vazio para usar o estado da loja",
						},
					},
					"required": []string{"local"},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "consultarEnderecoEmpresa",
				Description: "Consulta o endereço da empresa/loja. Use quando cliente perguntar sobre localização da empresa: 'qual o endereço?', 'onde fica a empresa?', 'onde posso buscar?', 'endereço da loja', 'como chegar aí?', etc.",
				Parameters: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		},
		{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        "solicitarAtendimentoHumano",
				Description: "Solicita atendimento humano quando cliente demonstra necessidade de falar com uma pessoa. Use quando cliente mencionar: 'falar com atendente', 'quero falar com humano', 'preciso de ajuda humana', 'transferir para pessoa', 'atendimento pessoal', 'representante', 'operador', reclamações complexas, problemas que a IA não consegue resolver, ou qualquer situação que requeira intervenção humana.",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"motivo": map[string]interface{}{
							"type":        "string",
							"description": "Motivo específico da solicitação de atendimento humano (ex: 'Cliente solicitou falar com atendente', 'Problema complexo que requer ajuda humana', 'Reclamação que necessita atenção pessoal')",
						},
					},
					"required": []string{"motivo"},
				},
			},
		},
	}
}

// ToolExecutionResult represents the result of executing a single tool/function
type ToolExecutionResult struct {
	ToolName   string                 `json:"tool_name"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     string                 `json:"result"`
	Error      string                 `json:"error,omitempty"`
}

// ToolExecutionResults represents results from multiple tool executions
type ToolExecutionResults struct {
	CombinedResponse  string                `json:"combined_response"`
	IndividualResults []ToolExecutionResult `json:"individual_results"`
}

func (s *AIService) executeToolCalls(ctx context.Context, tenantID, customerID uuid.UUID, customerPhone, userMessage string, toolCalls []openai.ToolCall) (string, error) {
	return s.executeToolCallsWithResults(ctx, tenantID, customerID, customerPhone, userMessage, toolCalls)
}

func (s *AIService) executeToolCallsWithResults(ctx context.Context, tenantID, customerID uuid.UUID, customerPhone, userMessage string, toolCalls []openai.ToolCall) (string, error) {
	var results []string
	var individualResults []ToolExecutionResult

	// // 🚨 CORREÇÃO: Detectar se são ações conflitantes ou repetitivas
	// if len(toolCalls) > 1 {
	// 	// Analisar se há ações duplicadas ou conflitantes
	// 	toolNames := make([]string, len(toolCalls))
	// 	for i, toolCall := range toolCalls {
	// 		toolNames[i] = toolCall.Function.Name
	// 	}

	// 	// Se há múltiplas ações relacionadas ao carrinho, executar apenas a mais específica
	// 	if s.shouldFilterMultipleCartActions(toolNames, userMessage) {
	// 		log.Warn().
	// 			Strs("tool_names", toolNames).
	// 			Str("user_message", userMessage).
	// 			Msg("🚨 Detectadas múltiplas ações de carrinho conflitantes - filtrando para uma ação")

	// 		// Filtrar para manter apenas a ação mais específica
	// 		toolCalls = s.filterCartActions(toolCalls, userMessage)
	// 	}
	// }

	for _, toolCall := range toolCalls {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
			log.Error().
				Err(err).
				Str("tool_name", toolCall.Function.Name).
				Str("tenant_id", tenantID.String()).
				Msg("Failed to parse tool arguments")

			// Add error result to individual results
			individualResults = append(individualResults, ToolExecutionResult{
				ToolName:   toolCall.Function.Name,
				Parameters: args,
				Result:     "",
				Error:      err.Error(),
			})
			continue
		}

		result, err := s.executeTool(ctx, tenantID, customerID, customerPhone, toolCall.Function.Name, args)
		if err != nil {
			log.Error().
				Err(err).
				Str("tool_name", toolCall.Function.Name).
				Str("tenant_id", tenantID.String()).
				Msg("Tool execution failed")

			// Usar o ErrorHandler para processar o erro e gerar uma mensagem amigável
			friendlyMessage := s.errorHandler.LogAIError(tenantID, customerID, customerPhone, userMessage, toolCall.Function.Name, args, err)
			results = append(results, friendlyMessage)

			// Add error result to individual results
			individualResults = append(individualResults, ToolExecutionResult{
				ToolName:   toolCall.Function.Name,
				Parameters: args,
				Result:     friendlyMessage,
				Error:      err.Error(),
			})
		} else {
			results = append(results, result)

			// Add successful result to individual results
			individualResults = append(individualResults, ToolExecutionResult{
				ToolName:   toolCall.Function.Name,
				Parameters: args,
				Result:     result,
			})
		}
	}

	// Debug log to see results before conditional logic

	if len(results) == 1 {
		// Store individual results for logging
		s.functionResultsMutex.Lock()
		s.lastFunctionResults = individualResults
		s.functionResultsMutex.Unlock()
		return results[0], nil
	}

	// 🚨 CORREÇÃO: Se há múltiplos resultados válidos, combinar em uma resposta
	// Isso permite feedback de múltiplos itens adicionados em uma única operação
	if len(results) > 1 {
		// Verificar se são ações de adicionar produtos - detectar pela mensagem de resposta
		addActions := 0
		searchActions := 0

		for _, result := range results {
			// Verificar se a resposta contém indicadores de adição ao carrinho
			if strings.Contains(result, "adicionado ao carrinho!") ||
				strings.Contains(result, "✅") {
				addActions++
			}
			// Verificar se é uma busca de produto
			if strings.Contains(result, "🔍 Encontrei") ||
				strings.Contains(result, "Produtos disponíveis") {
				searchActions++
			}
		}

		// Se todas são adições bem-sucedidas, combinar
		if addActions == len(results) {
			// Combinar múltiplas adições em uma resposta coesa
			log.Info().
				Int("results_count", len(results)).
				Int("add_actions", addActions).
				Str("user_message", userMessage).
				Msg("🔍 DEBUG: All additions successful case - combining results")

			// Criar resposta consolidada que mostra todos os itens
			combinedMessage := "✅ Itens adicionados ao carrinho:\n\n"

			for i, result := range results {
				// Extrair apenas a parte essencial (produto + preço)
				lines := strings.Split(result, "\n")
				if len(lines) > 0 {
					// Pegar primeira linha (nome do produto) e ajustar numeração
					productLine := strings.Replace(lines[0], "✅", fmt.Sprintf("%d.", i+1), 1)
					combinedMessage += productLine + "\n"

					// Adicionar quantidade e valor se disponível
					for _, line := range lines[1:] {
						if strings.Contains(line, "Quantidade:") || strings.Contains(line, "Valor:") {
							combinedMessage += "   " + line + "\n"
						}
					}
				}
			}

			combinedMessage += "\nVocê pode continuar comprando ou digite 'finalizar' para fechar o pedido."

			// Store individual results for logging
			s.functionResultsMutex.Lock()
			s.lastFunctionResults = individualResults
			s.functionResultsMutex.Unlock()

			return combinedMessage, nil
		}

		// Se há uma mistura de add + busca, isso sugere que alguns itens não foram encontrados
		// Vamos combinar de forma mais inteligente
		if addActions > 0 && searchActions > 0 {
			log.Info().
				Int("results_count", len(results)).
				Int("add_actions", addActions).
				Int("search_actions", searchActions).
				Str("user_message", userMessage).
				Msg("� DEBUG: Mixed add + search case - combining results")

			combinedMessage := ""

			// Primeiro, mostrar os itens adicionados com sucesso
			for _, result := range results {
				if strings.Contains(result, "adicionado ao carrinho!") {
					combinedMessage += result + "\n\n"
				}
			}

			// Depois, mostrar produtos encontrados que podem ser adicionados
			for _, result := range results {
				if strings.Contains(result, "🔍 Encontrei") {
					// Limpar instruções duplicadas
					cleanResult := strings.ReplaceAll(result, "📝 Para adicionar ao carrinho basta informar o numero do item ou nome do produto.", "")
					combinedMessage += "🔍 **Outros produtos encontrados:**\n" + strings.TrimSpace(cleanResult) + "\n\n"
				}
			}

			combinedMessage += "💡 Para ver detalhes: 'produto [número]'\n"
			combinedMessage += "🛒 Para adicionar: 'adicionar [número] quantidade [X]'"

			// Store individual results for logging
			s.functionResultsMutex.Lock()
			s.lastFunctionResults = individualResults
			s.functionResultsMutex.Unlock()

			return combinedMessage, nil
		}

		// 🔍 DETECTAR MÚLTIPLAS CONSULTAS DE PRODUTOS
		// Verificar se são múltiplas chamadas para consultarItens ou busca de produtos
		consultaActionCount := 0
		validProductsFound := 0

		for _, result := range results {
			// Verificar se a resposta contém indicadores de busca de produtos
			isProductSearch := strings.Contains(result, "🛍️") ||
				strings.Contains(result, "Produtos disponíveis") ||
				strings.Contains(result, "mais barato é") ||
				strings.Contains(result, "mais caro é") ||
				strings.Contains(result, "Opções de") ||
				strings.Contains(result, "Busquei") ||
				strings.Contains(result, "Encontrei") ||
				strings.Contains(result, "produtos similares")

			// Verificar se encontrou produtos válidos (não mensagens de erro)
			hasValidProducts := isProductSearch &&
				!strings.Contains(result, "❌ Não encontrei") &&
				!strings.Contains(result, "Não foi possível encontrar") &&
				!strings.Contains(result, "Tente outro termo")

			if isProductSearch {
				consultaActionCount++
			}

			if hasValidProducts {
				validProductsFound++
			}
		}

		// Se há múltiplas consultas de produtos, combinar em uma resposta
		if consultaActionCount > 1 && validProductsFound >= 1 {
			log.Info().
				Int("results_count", len(results)).
				Int("consulta_count", consultaActionCount).
				Int("valid_products_found", validProductsFound).
				Str("user_message", userMessage).
				Msg("🔍 Combinando múltiplas consultas de produtos em uma resposta")

			// Usar nova função para combinar resultados com numeração sequencial
			combinedMessage := s.combineProductResultsWithSequentialNumbering(results)

			// Store individual results for logging
			s.functionResultsMutex.Lock()
			s.lastFunctionResults = individualResults
			s.functionResultsMutex.Unlock()

			return combinedMessage, nil
		}

		// Para outros casos, retornar apenas o primeiro para evitar confusão
		log.Warn().
			Int("results_count", len(results)).
			Str("user_message", userMessage).
			Msg("🚨 Múltiplos resultados não relacionados - retornando apenas o primeiro")

		// Store individual results for logging
		s.functionResultsMutex.Lock()
		s.lastFunctionResults = individualResults
		s.functionResultsMutex.Unlock()

		return results[0], nil
	}

	// Store individual results for logging
	s.functionResultsMutex.Lock()
	s.lastFunctionResults = individualResults
	s.functionResultsMutex.Unlock()

	return "❌ Não foi possível processar a solicitação.", nil
}

// getLastFunctionResults returns and clears the last function results
func (s *AIService) getLastFunctionResults() []ToolExecutionResult {
	s.functionResultsMutex.Lock()
	defer s.functionResultsMutex.Unlock()

	results := s.lastFunctionResults
	s.lastFunctionResults = nil // Clear after retrieving
	return results
}

// shouldFilterMultipleCartActions detecta se há ações conflitantes ou repetitivas de carrinho
func (s *AIService) shouldFilterMultipleCartActions(toolNames []string, userMessage string) bool {
	cartActions := []string{"adicionarAoCarrinho", "adicionarProdutoPorNome", "adicionarPorNumero", "atualizarQuantidade", "verCarrinho"}

	cartActionCount := 0
	for _, toolName := range toolNames {
		for _, cartAction := range cartActions {
			if toolName == cartAction {
				cartActionCount++
				break
			}
		}
	}

	lowerMessage := strings.ToLower(userMessage)

	// Detectar múltiplos itens legítimos - NÃO filtrar quando usuário pede múltiplos itens
	// Melhorar detecção de múltiplos itens
	multipleItemsIndicators := []string{
		" e ", " e uma", " e um", " e o", " e a",
		", ", " mais ", " também", " outro", " outra",
		"item 1 e", "coca e", "água e", "primeiro e", "segundo e",
	}

	hasMultipleItems := false
	for _, indicator := range multipleItemsIndicators {
		if strings.Contains(lowerMessage, indicator) {
			hasMultipleItems = true
			break
		}
	}

	// Padrões adicionais para detectar múltiplos itens
	// Ex: "coca 350" e "água", "item 1 e coca", etc.
	spaceAndPattern := strings.Contains(lowerMessage, " e ") ||
		strings.Contains(lowerMessage, " mais ") ||
		strings.Contains(lowerMessage, ", ")

	if spaceAndPattern {
		hasMultipleItems = true
	}

	// Se usuário claramente quer múltiplos itens diferentes, NÃO filtrar
	if hasMultipleItems && cartActionCount > 1 {
		log.Info().
			Str("user_message", userMessage).
			Strs("tool_names", toolNames).
			Msg("🛍️ Usuário solicitou múltiplos itens - permitindo execução de todas as ações")
		return false
	}

	// Detectar casos específicos como "adiciona mais uma coca" que não deveria gerar múltiplas ações
	if strings.Contains(lowerMessage, "adiciona") && !strings.Contains(lowerMessage, "finalizar") && !strings.Contains(lowerMessage, "carrinho") {
		// Se tem apenas uma solicitação de adicionar item SIMPLES, não deveria gerar múltiplas ações
		isSingleItemRequest := !hasMultipleItems &&
			!strings.Contains(lowerMessage, "e ") &&
			!strings.Contains(lowerMessage, ",")

		if isSingleItemRequest && cartActionCount > 1 {
			return true
		}
	}

	// Se há mais de uma ação de carrinho mas não são múltiplos itens legítimos, filtrar
	return cartActionCount > 1 && !hasMultipleItems
}

// filterCartActions filtra tool calls para manter apenas a ação mais apropriada
func (s *AIService) filterCartActions(toolCalls []openai.ToolCall, userMessage string) []openai.ToolCall {
	lowerMessage := strings.ToLower(userMessage)

	// Priorizar ações baseadas na mensagem do usuário
	if strings.Contains(lowerMessage, "adiciona") || strings.Contains(lowerMessage, "quero") {
		// Priorizar adicionar produtos
		for _, toolCall := range toolCalls {
			if toolCall.Function.Name == "adicionarProdutoPorNome" || toolCall.Function.Name == "adicionarAoCarrinho" {
				return []openai.ToolCall{toolCall}
			}
		}
	}

	if strings.Contains(lowerMessage, "ver") || strings.Contains(lowerMessage, "carrinho") {
		// Priorizar ver carrinho
		for _, toolCall := range toolCalls {
			if toolCall.Function.Name == "verCarrinho" {
				return []openai.ToolCall{toolCall}
			}
		}
	}

	// Se não conseguiu filtrar especificamente, retornar apenas o primeiro
	if len(toolCalls) > 0 {
		return []openai.ToolCall{toolCalls[0]}
	}

	return toolCalls
}

func joinResults(results []string) string {
	var joined string
	for i, result := range results {
		if i > 0 {
			joined += "\n\n"
		}
		joined += fmt.Sprintf("%d. %s", i+1, result)
	}
	return joined
}

func (s *AIService) executeTool(ctx context.Context, tenantID, customerID uuid.UUID, customerPhone string, toolName string, args map[string]interface{}) (string, error) {
	switch toolName {
	case "consultarItens":
		return s.handleConsultarItens(tenantID, customerPhone, args)
	case "mostrarOpcoesCategoria":
		return s.handleMostrarOpcoesCategoria(tenantID, customerPhone, args)
	case "detalharItem":
		return s.handleDetalharItem(tenantID, customerPhone, args)
	case "adicionarAoCarrinho":
		return s.handleAdicionarAoCarrinho(tenantID, customerID, customerPhone, args)
	case "buscarMultiplosProdutos":
		return s.handleBuscarMultiplosProdutos(tenantID, customerID, customerPhone, args)
	case "adicionarProdutoPorNome":
		return s.handleAdicionarProdutoPorNome(tenantID, customerID, customerPhone, args)
	case "adicionarPorNumero":
		return s.handleAdicionarPorNumero(tenantID, customerID, customerPhone, args)
	case "adicionarMaisItemCarrinho":
		return s.handleAdicionarMaisItemCarrinho(tenantID, customerID, args)
	case "atualizarQuantidade":
		return s.handleAtualizarQuantidade(tenantID, customerID, args)
	case "removerDoCarrinho":
		return s.handleRemoverDoCarrinho(tenantID, customerID, args)
	case "verCarrinho":
		return s.handleVerCarrinhoWithOptions(tenantID, customerID, true) // Full instructions for view cart
	case "limparCarrinho":
		return s.handleLimparCarrinho(tenantID, customerID)
	case "checkout":
		log.Info().Str("tool_name", "checkout").Msg("🎯 EXECUTING CHECKOUT FUNCTION")
		return s.handleCheckout(tenantID, customerID, customerPhone)
	case "finalizarPedido":
		log.Info().Str("tool_name", "finalizarPedido").Msg("🚀 EXECUTING FINALIZAR PEDIDO FUNCTION")
		return s.performFinalCheckout(tenantID, customerID, customerPhone)
	case "cancelarPedido":
		// Adicionar customer_id aos argumentos para permitir busca na memória
		args["customer_id"] = customerID.String()
		return s.handleCancelarPedido(tenantID, args)
	case "historicoPedidos":
		return s.handleHistoricoPedidos(tenantID, customerID)
	case "atualizarCadastro":
		log.Info().Str("tool_name", "atualizarCadastro").Interface("args", args).Msg("🔄 EXECUTING ATUALIZAR CADASTRO FUNCTION")
		return s.handleAtualizarCadastro(tenantID, customerID, customerPhone, args)
	case "gerenciarEnderecos":
		return s.handleGerenciarEnderecos(tenantID, customerID, args)
	case "cadastrarEndereco":
		return s.handleCadastrarEndereco(tenantID, customerID, args)
	case "verificarEntrega":
		return s.handleVerificarEntrega(tenantID, args)
	case "consultarEnderecoEmpresa":
		return s.handleConsultarEnderecoEmpresa(tenantID, customerID, customerPhone)
	case "solicitarAtendimentoHumano":
		return s.handleSolicitarAtendimentoHumano(tenantID, customerID, customerPhone, args)
	default:
		return "", fmt.Errorf("ferramenta não reconhecida: %s", toolName)
	}
}

// isProductMentionedWithoutConsultation detecta se produtos foram mencionados sem consulta ao banco
func (s *AIService) isProductMentionedWithoutConsultation(userMessage, aiResponse string) bool {
	userLower := strings.ToLower(userMessage)
	responseLower := strings.ToLower(aiResponse)

	log.Info().
		Str("user_message", userMessage).
		Str("ai_response", aiResponse).
		Msg("🔍 Checking for product mention without consultation")

	// Verificação 1: Se a resposta contém regra de escopo, significa que recusou sem consultar
	if strings.Contains(responseLower, "sou assistente de vendas") &&
		strings.Contains(responseLower, "só posso ajudar com nossos produtos") {
		log.Warn().Msg("🚨 Detected scope rule applied - likely refused without database consultation")
		return true
	}

	// Verificação 2: Se usuário menciona palavras que indicam solicitação de produto
	productRequestKeywords := []string{
		"quero", "preciso", "busco", "procuro", "tem", "vende", "vendem",
		"onde encontro", "vocês têm", "vocês tem", "qual", "me mostra",
		"buscar", "encontrar", "comprar", "adicionar",
	}

	hasProductRequest := false
	for _, keyword := range productRequestKeywords {
		if strings.Contains(userLower, keyword) {
			hasProductRequest = true
			log.Info().Str("keyword", keyword).Msg("🔍 Found product request keyword")
			break
		}
	}

	// Verificação 3: Se menciona substantivos específicos (nomes de produtos)
	words := strings.Fields(userLower)
	hasSpecificProduct := false
	for _, word := range words {
		cleanWord := strings.Trim(word, ".,!?;:")
		// Se a palavra tem mais de 3 caracteres e não é uma palavra comum
		if len(cleanWord) > 3 && !isCommonWordBusiness(cleanWord) {
			hasSpecificProduct = true
			log.Info().Str("product_word", cleanWord).Msg("🔍 Found potential product name")
			break
		}
	}

	// Verificação 4: Detectar se é uma consulta de endereço/localização
	addressKeywords := []string{
		"endereço", "endereco", "onde fica", "onde vocês ficam", "localização", "localizacao",
		"como chegar", "endereço da empresa", "endereço da loja", "onde posso buscar",
		"ponto de coleta", "onde retirar", "buscar", "onde fica a loja", "onde fica a empresa",
	}

	isAddressQuery := false
	for _, keyword := range addressKeywords {
		if strings.Contains(userLower, keyword) {
			isAddressQuery = true
			log.Info().Str("address_keyword", keyword).Msg("🏠 Detected address/location query")
			break
		}
	}

	// Se é consulta de endereço, NÃO forçar consulta de produto
	if isAddressQuery {
		log.Info().Msg("✅ Address query detected - skipping forced product consultation")
		return false
	}

	// Se há indicação de produto E resposta de escopo, forçar consulta
	if hasProductRequest || hasSpecificProduct {
		log.Warn().
			Bool("has_request", hasProductRequest).
			Bool("has_product", hasSpecificProduct).
			Msg("🚨 Product mention detected - forcing database consultation")
		return true
	}

	return false
}

// forceProductConsultation força uma nova análise consultando o banco de dados
func (s *AIService) forceProductConsultation(ctx context.Context, tenantID uuid.UUID, customer *models.Customer, customerPhone, userMessage string, originalMessages []openai.ChatCompletionMessage) (string, error) {
	log.Info().Msg("Forcing product consultation to prevent hallucination")

	// Criar um prompt específico que força a consulta
	forceConsultationPrompt := fmt.Sprintf(`O usuário disse: "%s"

IMPORTANTE: Esta mensagem menciona produtos. Você DEVE:
1. PRIMEIRO: Use a função "consultarItens" para pesquisar produtos no banco de dados
2. APENAS depois da consulta, responda baseado nos resultados reais
3. NÃO invente informações sobre produtos - use apenas dados do banco

Execute a consulta agora:`, userMessage)

	// Criar nova mensagem forçando consulta
	forceMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: forceConsultationPrompt,
	}

	// Preparar mensagens incluindo contexto original
	messages := append(originalMessages[:len(originalMessages)-1], forceMessage) // Remove última msg e adiciona nova

	// Definir tools disponíveis
	tools := s.getAvailableTools()

	// Fazer nova chamada para OpenAI
	resp, err := s.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:               openai.GPT5Nano,
		Messages:            messages,
		Tools:               tools,
		ToolChoice:          "auto",
		MaxCompletionTokens: 10000,
	})

	if err != nil {
		log.Error().Err(err).Msg("Failed to force product consultation")
		return "Desculpe, houve um erro ao consultar nossos produtos. Tente novamente.", nil
	}

	choice := resp.Choices[0]

	// Desta vez, deve haver tool calls
	if len(choice.Message.ToolCalls) > 0 {
		log.Info().Msg("Forced consultation successful - executing tool calls")
		return s.executeToolCalls(ctx, tenantID, customer.ID, customerPhone, userMessage, choice.Message.ToolCalls)
	} else {
		// Se mesmo assim não usar tools, dar uma resposta padrão
		log.Warn().Msg("Even forced consultation did not trigger tool calls")
		return "Para ajudar você melhor, que tipo específico de produto você está procurando? Use 'produtos' para ver nosso catálogo completo.", nil
	}
}

// isRefinementWithoutContext detecta se usuário está refinando busca anterior sem contexto
func (s *AIService) isRefinementWithoutContext(userMessage, aiResponse string, conversationHistory []openai.ChatCompletionMessage) bool {
	userLower := strings.ToLower(userMessage)

	log.Info().
		Str("user_message", userMessage).
		Int("history_length", len(conversationHistory)).
		Msg("🔍 Checking for refinement without context")

	// Verificar se é uma mensagem de características técnicas isoladas
	refinementPatterns := []string{
		// Medicamentos
		"20mg", "40mg", "mg", "comprimidos", "cápsulas", "ml", "gotas",
		// Papelaria
		"a4", "a3", "a5", "folhas", "páginas", "cm", "mm",
		// Cores
		"azul", "preto", "vermelho", "verde", "branco", "amarelo",
		// Quantidades
		"unidades", "caixas", "pacotes", "unids", "pçs",
		// Especificações
		"com capa", "sem pauta", "pautado", "liso", "quadriculado",
	}

	hasRefinementPattern := false
	matchedPattern := ""
	for _, pattern := range refinementPatterns {
		if strings.Contains(userLower, pattern) {
			hasRefinementPattern = true
			matchedPattern = pattern
			break
		}
	}

	log.Info().
		Bool("has_refinement_pattern", hasRefinementPattern).
		Str("matched_pattern", matchedPattern).
		Msg("🔍 Refinement pattern check")

	if !hasRefinementPattern {
		log.Info().Msg("❌ No refinement pattern found")
		return false
	}

	// Verificar se nas últimas mensagens houve consulta de produtos
	log.Info().Int("checking_history_messages", len(conversationHistory)).Msg("🔍 Analyzing conversation history for product lists")

	for i := len(conversationHistory) - 1; i >= 0 && i >= len(conversationHistory)-6; i-- {
		msg := conversationHistory[i]
		log.Info().
			Int("msg_index", i).
			Str("role", string(msg.Role)).
			Int("content_length", len(msg.Content)).
			Str("content_preview", msg.Content[:min(150, len(msg.Content))]).
			Msg("🔍 Checking history message")

		if msg.Role == openai.ChatMessageRoleAssistant {
			content := strings.ToLower(msg.Content)

			// Log what we're searching for
			log.Info().
				Bool("has_produtos_disponiveis", strings.Contains(content, "produtos disponíveis")).
				Bool("has_encontrei", strings.Contains(content, "encontrei")).
				Bool("has_money_emoji", strings.Contains(content, "💰")).
				Bool("has_adicionar_carrinho", strings.Contains(content, "para adicionar ao carrinho")).
				Str("full_content", msg.Content).
				Msg("🔍 History content analysis")

			// Se a IA mostrou produtos recentemente
			if strings.Contains(content, "produtos disponíveis") ||
				strings.Contains(content, "encontrei") ||
				strings.Contains(content, "💰") ||
				strings.Contains(content, "para adicionar ao carrinho") {
				log.Warn().
					Str("user_message", userMessage).
					Str("previous_response", msg.Content[:min(200, len(msg.Content))]).
					Msg("� DETECTED: Refinement pattern after product list - forcing contextual consultation")
				return true
			}
		}
	}

	log.Info().Msg("❌ No recent product list found in history - checking previous user message")

	// ABORDAGEM ALTERNATIVA: Verificar se a mensagem anterior do usuário mencionava produto
	log.Info().
		Int("total_history_length", len(conversationHistory)).
		Msg("🔍 Starting previous message analysis")

	// Primeiro, vamos logar todo o histórico para debug
	for i, msg := range conversationHistory {
		log.Info().
			Int("msg_index", i).
			Str("role", string(msg.Role)).
			Str("content", msg.Content[:min(100, len(msg.Content))]).
			Msg("🔍 History message content")
	}

	if len(conversationHistory) >= 2 {
		// Procurar a mensagem anterior do usuário (ignorando a atual)
		userMessageCount := 0
		for i := len(conversationHistory) - 1; i >= 0; i-- {
			if conversationHistory[i].Role == openai.ChatMessageRoleUser {
				userMessageCount++

				// Ignorar a primeira mensagem do usuário (que é a atual)
				if userMessageCount == 1 {
					log.Info().
						Str("current_message", conversationHistory[i].Content).
						Msg("🔍 Skipping current user message")
					continue
				}

				// Esta é a mensagem anterior do usuário
				previousMessage := strings.ToLower(conversationHistory[i].Content)
				log.Info().
					Str("previous_user_message", conversationHistory[i].Content).
					Str("current_message", userMessage).
					Msg("🔍 Checking previous user message for product context")

				// Lista de produtos/termos que indicam busca anterior
				productTerms := []string{
					"pantoprazol", "omeprazol", "paracetamol", "dipirona", "ibuprofeno",
					"caneta", "caderno", "papel", "régua", "lápis", "borracha",
					"quero", "preciso", "busco", "tem", "vende", "procuro",
				}

				for _, term := range productTerms {
					if strings.Contains(previousMessage, term) {
						log.Warn().
							Str("previous_search", conversationHistory[i].Content).
							Str("current_refinement", userMessage).
							Str("detected_term", term).
							Msg("🔥 CONTEXTUAL REFINEMENT DETECTED - forcing combined search")
						return true
					}
				}
				break // Só verificar a mensagem anterior do usuário
			}
		}
	}

	return false
}

// forceContextualConsultation força consulta combinando contexto anterior
func (s *AIService) forceContextualConsultation(ctx context.Context, tenantID uuid.UUID, customer *models.Customer, customerPhone, userMessage string, originalMessages []openai.ChatCompletionMessage) (string, error) {
	log.Info().Msg("Forcing contextual consultation to maintain search context")

	// Extrair produto da busca anterior
	previousProduct := s.extractPreviousProduct(originalMessages)

	// Criar prompt que força consulta contextual
	contextualPrompt := fmt.Sprintf(`O usuário disse: "%s"

CONTEXTO: O usuário estava buscando por "%s" anteriormente e agora está especificando características.

IMPORTANTE: Você DEVE:
1. COMBINAR automaticamente: "%s %s"
2. PRIMEIRO: Use a função "consultarItens" com a query combinada
3. APENAS depois da consulta, responda baseado nos resultados reais
4. NÃO trate como busca independente

Execute a consulta contextual agora:`, userMessage, previousProduct, previousProduct, userMessage)

	// Criar nova mensagem forçando consulta contextual
	forceMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: contextualPrompt,
	}

	// Adicionar à conversa e reenviar
	contextualMessages := append(originalMessages, forceMessage)

	req := openai.ChatCompletionRequest{
		Model:    openai.GPT4oMini,
		Messages: contextualMessages,
		Tools:    s.getAvailableTools(),
	}

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return "", fmt.Errorf("erro na consulta contextual forçada: %w", err)
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if len(choice.Message.ToolCalls) > 0 {
			// Executar tool calls da consulta contextual
			return s.executeToolCalls(ctx, tenantID, customer.ID, customerPhone, userMessage, choice.Message.ToolCalls)
		}
		return choice.Message.Content, nil
	}

	return "Para ajudar você melhor, que características específicas você procura? Use 'produtos' para ver nosso catálogo.", nil
}

// isDirectDeliveryQuestion detecta se é uma pergunta direta sobre entrega
func (s *AIService) isDirectDeliveryQuestion(userMessage string) bool {
	userLower := strings.ToLower(userMessage)

	// Primeiro, verificar se é claramente uma pergunta sobre produto
	productPatterns := []string{
		"tem ", "há ", "vocês tem ", "vcs tem ", "possui ", "vende ",
		"vendem ", "trabalha com ", "trabalham com ", "qual o preço",
		"quanto custa", "preço do ", "preço da ",
	}

	for _, pattern := range productPatterns {
		if strings.HasPrefix(userLower, pattern) || strings.Contains(userLower, pattern) {
			log.Info().
				Str("product_pattern", pattern).
				Str("user_message", userMessage).
				Msg("🛍️ Detected product question - not delivery")
			return false
		}
	}

	// Padrões que indicam pergunta direta sobre entrega
	directDeliveryPatterns := []string{
		"entrega em ",
		"entregam em ",
		"entregam no ",
		"entregam na ",
		"entrega no ",
		"entrega na ",
		"fazem entrega em ",
		"fazem entrega no ",
		"fazem entrega na ",
		"vocês entregam em ",
		"vocês entregam no ",
		"vocês entregam na ",
		"vcs entregam em ",
		"vcs entregam no ",
		"vcs entregam na ",
		"atendem em ",
		"atendem no ",
		"atendem na ",
	}

	for _, pattern := range directDeliveryPatterns {
		if strings.Contains(userLower, pattern) {
			log.Info().
				Str("pattern", pattern).
				Str("user_message", userMessage).
				Msg("🚚 Found direct delivery question pattern")
			return true
		}
	}

	return false
}

// isDeliveryContinuation detecta se usuário está continuando uma conversa sobre entregas
func (s *AIService) isDeliveryContinuation(userMessage string, conversationHistory []openai.ChatCompletionMessage) bool {
	userLower := strings.ToLower(userMessage)

	log.Info().
		Str("user_message", userMessage).
		Int("history_length", len(conversationHistory)).
		Msg("🚚 Checking for delivery continuation context")

	// Primeiro, verificar se é claramente uma pergunta sobre produto - se for, não é continuação de entrega
	productPatterns := []string{
		"tem ", "há ", "vocês tem ", "vcs tem ", "possui ", "vende ",
		"vendem ", "trabalha com ", "trabalham com ", "qual o preço",
		"quanto custa", "preço do ", "preço da ", "quero ", "preciso de ",
		"gostaria de ", "pode me mostrar ", "onde está ",
	}

	for _, pattern := range productPatterns {
		if strings.HasPrefix(userLower, pattern) || strings.Contains(userLower, pattern) {
			log.Info().
				Str("product_pattern", pattern).
				Str("user_message", userMessage).
				Msg("🛍️ Detected product question - not delivery continuation")
			return false
		}
	}

	// Verificar se é uma pergunta de continuação (e, também, ou similar) sobre locais
	continuationPatterns := []string{
		"e ", "e praia", "e vila", "e centro", "e bairro", "e cidade", "e região",
		"também ", "ou ", "que tal ", "como fica ", "e o ", "e a ",
	}

	isContinuation := false
	for _, pattern := range continuationPatterns {
		if strings.HasPrefix(userLower, pattern) {
			isContinuation = true
			log.Info().Str("continuation_pattern", pattern).Msg("🚚 Found continuation pattern")
			break
		}
	}

	// Ou perguntas diretas sobre áreas/bairros de entrega
	deliveryAreaPatterns := []string{
		"quais bairros", "que bairros", "onde entregam", "quais regiões", "que regiões",
		"área de entrega", "cobertura", "quais locais", "que locais",
	}

	isDeliveryAreaQuery := false
	for _, pattern := range deliveryAreaPatterns {
		if strings.Contains(userLower, pattern) {
			isDeliveryAreaQuery = true
			log.Info().Str("delivery_area_pattern", pattern).Msg("🚚 Found delivery area query pattern")
			break
		}
	}

	if !isContinuation && !isDeliveryAreaQuery {
		log.Info().Msg("❌ No delivery continuation pattern found")
		return false
	}

	// Verificar se nas últimas mensagens houve conversa sobre entregas
	log.Info().Int("checking_history_messages", len(conversationHistory)).Msg("🚚 Analyzing conversation history for delivery context")

	for i := len(conversationHistory) - 1; i >= 0 && i >= len(conversationHistory)-6; i-- {
		msg := conversationHistory[i]
		msgLower := strings.ToLower(msg.Content)

		log.Info().
			Int("msg_index", i).
			Str("role", string(msg.Role)).
			Int("content_length", len(msg.Content)).
			Str("content_preview", msg.Content[:min(100, len(msg.Content))]).
			Msg("🚚 Checking history message for delivery context")

		// Verificar se há contexto de entrega nas mensagens anteriores
		deliveryContext := strings.Contains(msgLower, "entrega") ||
			strings.Contains(msgLower, "atender") ||
			strings.Contains(msgLower, "não conseguimos") ||
			strings.Contains(msgLower, "área de atendimento") ||
			strings.Contains(msgLower, "🚫") ||
			strings.Contains(msgLower, "🚚")

		if deliveryContext {
			log.Info().
				Bool("has_delivery_context", true).
				Msg("🚚 Found delivery context in conversation history")
			return true
		}
	}

	log.Info().Msg("❌ No delivery context found in conversation history")
	return false
}

// processDeliveryRequest processa especificamente perguntas sobre entrega
func (s *AIService) processDeliveryRequest(ctx context.Context, tenantID uuid.UUID, customerPhone, userMessage string, messages []openai.ChatCompletionMessage) (string, error) {
	log.Info().
		Str("user_message", userMessage).
		Str("customer_phone", customerPhone).
		Str("tenant_id", tenantID.String()).
		Msg("🚚 Processing delivery request")

	// Verificar se é pergunta sobre áreas de entrega em geral
	userLower := strings.ToLower(userMessage)
	if strings.Contains(userLower, "quais bairros") ||
		strings.Contains(userLower, "que bairros") ||
		strings.Contains(userLower, "onde entregam") ||
		strings.Contains(userLower, "quais regiões") ||
		strings.Contains(userLower, "área de entrega") {

		log.Info().Msg("🚚 General delivery area question - providing general info")
		return "Para verificar se entregamos no seu endereço, me informe sua localização completa (rua, bairro, cidade) que eu consulto nossa área de cobertura! 🚚", nil
	}

	// Tentar extrair local da mensagem ou do contexto
	extractedLocation := s.extractLocationFromMessage(userMessage)
	if extractedLocation == "" {
		// Tentar extrair do contexto da conversa
		extractedLocation = s.extractLocationFromContext(messages)
	}

	if extractedLocation == "" {
		log.Info().Msg("🚚 No location found - asking for details")
		return "Para verificar a entrega, preciso que você me informe o endereço completo (rua, bairro, cidade). 📍", nil
	}

	log.Info().Str("extracted_location", extractedLocation).Msg("🚚 Using extracted location for delivery check")

	// Chamar verificação de entrega usando o mesmo padrão do handleVerificarEntrega
	result, err := s.deliveryService.ValidateDeliveryAddress(tenantID, "", "", extractedLocation, "", "")
	if err != nil {
		log.Error().Err(err).Str("location", extractedLocation).Msg("Failed to validate delivery address")
		return "Desculpe, houve um erro ao verificar a entrega. Tente novamente em alguns instantes.", err
	}

	// Formatar resposta baseada no resultado
	if result.CanDeliver {
		response := fmt.Sprintf("✅ Ótima notícia! Entregamos em %s", extractedLocation)
		if result.Distance != "" {
			response += fmt.Sprintf(" (%s de distância)", result.Distance)
		}
		response += "! Posso ajudar com algum produto?"
		return response, nil
	} else {
		response := fmt.Sprintf("🚫 Infelizmente não conseguimos entregar em %s", extractedLocation)
		if result.Reason != "" {
			response += fmt.Sprintf(" (%s)", result.Reason)
		}
		response += ". Verifique se o endereço está correto ou consulte outras áreas próximas."
		return response, nil
	}
}

// extractLocationFromMessage tenta extrair localização da mensagem atual
func (s *AIService) extractLocationFromMessage(message string) string {
	messageLower := strings.ToLower(message)

	// Padrões específicos de entrega (processados primeiro)
	deliveryPatterns := []string{
		"entrega em ", "entregam em ", "entregam no ", "entregam na ",
		"entrega no ", "entrega na ", "fazem entrega em ", "fazem entrega no ",
		"fazem entrega na ", "vocês entregam em ", "vocês entregam no ",
		"vocês entregam na ", "vcs entregam em ", "vcs entregam no ",
		"vcs entregam na ", "atendem em ", "atendem no ", "atendem na ",
	}

	for _, pattern := range deliveryPatterns {
		if idx := strings.Index(messageLower, pattern); idx >= 0 {
			// Extrair o que vem depois do pattern
			after := message[idx+len(pattern):]
			location := s.extractLocationPhrase(after)
			if location != "" {
				return location
			}
		}
	}

	// Padrões genéricos de localização
	locationPatterns := []string{
		"praia de ", "vila ", "centro de ", "bairro ", "cidade de ",
		"em ", "para ", "no ", "na ", "do ", "da ",
	}

	for _, pattern := range locationPatterns {
		if idx := strings.Index(messageLower, pattern); idx >= 0 {
			// Extrair o que vem depois do pattern
			after := message[idx+len(pattern):]
			// Extrair até o final da frase ou pontuação
			location := s.extractLocationPhrase(after)
			if location != "" {
				return location
			}
		}
	}

	// Se começar com "e " pode ser continuação - pegar tudo após "e "
	if strings.HasPrefix(messageLower, "e ") {
		after := message[2:] // Remove "e "
		location := s.extractLocationPhrase(after)
		if location != "" {
			return location
		}
	}

	return ""
}

// extractLocationPhrase extrai uma frase completa de localização
func (s *AIService) extractLocationPhrase(text string) string {
	// Limpar o texto
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	// Caracteres que indicam fim da localização
	stopChars := []string{"?", "!", ".", ",", ";", "(", ")"}

	// Encontrar onde parar
	endPos := len(text)
	for _, stopChar := range stopChars {
		if pos := strings.Index(text, stopChar); pos != -1 && pos < endPos {
			endPos = pos
		}
	}

	// Extrair até a posição encontrada
	location := strings.TrimSpace(text[:endPos])

	// Capitalizar adequadamente
	if location != "" {
		// Capitalizar primeira letra de cada palavra
		words := strings.Fields(location)
		for i, word := range words {
			if len(word) > 0 {
				words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
			}
		}
		return strings.Join(words, " ")
	}

	return ""
}

// extractLocationFromContext tenta extrair localização do contexto da conversa
func (s *AIService) extractLocationFromContext(messages []openai.ChatCompletionMessage) string {
	// Procurar nas últimas mensagens por endereços mencionados
	for i := len(messages) - 1; i >= 0 && i >= len(messages)-5; i-- {
		msg := messages[i]
		if extracted := s.extractLocationFromMessage(msg.Content); extracted != "" {
			return extracted
		}
	}
	return ""
}

// extractPreviousProduct extrai o produto buscado anteriormente do histórico
func (s *AIService) extractPreviousProduct(messages []openai.ChatCompletionMessage) string {
	// Procurar por tool calls anteriores de consultarItens
	for i := len(messages) - 1; i >= 0 && i >= len(messages)-10; i-- {
		msg := messages[i]
		if msg.Role == openai.ChatMessageRoleAssistant && len(msg.ToolCalls) > 0 {
			for _, toolCall := range msg.ToolCalls {
				if toolCall.Function.Name == "consultarItens" {
					// Extrair query do tool call
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err == nil {
						if query, ok := args["query"].(string); ok && query != "" {
							log.Info().Str("extracted_product", query).Msg("🔍 Extracted previous product from context")
							return query
						}
					}
				}
			}
		}

		// Alternativa: procurar por mensagens do usuário com produtos
		if msg.Role == openai.ChatMessageRoleUser {
			content := strings.ToLower(msg.Content)
			commonProducts := []string{
				"pantoprazol", "paracetamol", "dipirona", "ibuprofeno",
				"caneta", "lápis", "caderno", "papel", "régua",
			}
			for _, product := range commonProducts {
				if strings.Contains(content, product) {
					log.Info().Str("extracted_product", product).Msg("🔍 Extracted product from user message")
					return product
				}
			}
		}
	}

	return "produto"
}

// isAddressResponseWithoutTool detecta se a IA respondeu sobre endereço sem usar a ferramenta consultarEnderecoEmpresa
func (s *AIService) isAddressResponseWithoutTool(userMessage, aiResponse string) bool {
	userLower := strings.ToLower(userMessage)
	responseLower := strings.ToLower(aiResponse)

	// Verificar se usuário perguntou sobre endereço/localização
	addressKeywords := []string{
		"endereço", "endereco", "onde fica", "onde vocês ficam", "localização", "localizacao",
		"como chegar", "endereço da empresa", "endereço da loja", "onde posso buscar",
		"ponto de coleta", "onde retirar", "buscar", "onde fica a loja", "onde fica a empresa",
	}

	isAddressQuery := false
	for _, keyword := range addressKeywords {
		if strings.Contains(userLower, keyword) {
			isAddressQuery = true
			log.Info().Str("address_keyword", keyword).Msg("🏠 Detected address query in user message")
			break
		}
	}

	// Se não é consulta de endereço, não precisa forçar ferramenta
	if !isAddressQuery {
		return false
	}

	// Verificar se a resposta contém informações de endereço (indicando que respondeu diretamente)
	addressResponseKeywords := []string{
		"nosso endereço", "nossa localização", "localização:", "endereço:", "rodovia", "rua", "avenida",
		"ficamos em", "nossa loja fica", "estamos localizados", "você pode nos encontrar",
	}

	for _, keyword := range addressResponseKeywords {
		if strings.Contains(responseLower, keyword) {
			log.Warn().
				Str("response_keyword", keyword).
				Msg("🚨 Detected direct address response - should use consultarEnderecoEmpresa tool")
			return true
		}
	}

	return false
}

// forceAddressConsultation força o uso da ferramenta consultarEnderecoEmpresa
func (s *AIService) forceAddressConsultation(ctx context.Context, tenantID uuid.UUID, customer *models.Customer, customerPhone, userMessage string, originalMessages []openai.ChatCompletionMessage) (string, error) {
	log.Info().Msg("🏠 Forcing address consultation using consultarEnderecoEmpresa tool")

	// Criar um prompt específico que força o uso da ferramenta
	forceAddressPrompt := fmt.Sprintf(`O usuário perguntou sobre endereço/localização: "%s"

IMPORTANTE: Para consultas sobre endereço da empresa, você DEVE:
1. SEMPRE usar a função "consultarEnderecoEmpresa" 
2. NUNCA responder diretamente com informações de endereço
3. A ferramenta consultarEnderecoEmpresa irá fornecer informações completas e enviar localização via WhatsApp

Use a função consultarEnderecoEmpresa agora.`, userMessage)

	// Criar mensagens com o prompt forçado
	forceMessages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: s.getSystemPrompt(customer),
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: forceAddressPrompt,
		},
	}

	// Fazer nova chamada com força de tool
	req := openai.ChatCompletionRequest{
		Model:       "gpt-4o",
		Messages:    forceMessages,
		Tools:       s.getAvailableTools(),
		ToolChoice:  "auto",
		Temperature: 0.1,
		MaxTokens:   1000,
	}

	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("Error in forced address consultation")
		return "❌ Erro ao consultar informações de endereço. Tente novamente.", err
	}

	if len(resp.Choices) == 0 {
		return "❌ Erro ao processar consulta de endereço.", fmt.Errorf("no response choices")
	}

	choice := resp.Choices[0]

	// Se agora tem tool calls, executar
	if len(choice.Message.ToolCalls) > 0 {
		log.Info().Msg("✅ Address consultation now using tools - executing consultarEnderecoEmpresa")
		return s.executeToolCalls(ctx, tenantID, customer.ID, customerPhone, userMessage, choice.Message.ToolCalls)
	}

	// Se ainda não usa tools, retornar resposta direta (fallback)
	log.Warn().Msg("Address consultation still not using tools - returning direct response")
	return choice.Message.Content, nil
}

// getBusinessHoursInfo retorna informações sobre horários de funcionamento
func (s *AIService) getBusinessHoursInfo(ctx context.Context, tenantID uuid.UUID) string {
	log.Info().Str("tenant_id", tenantID.String()).Msg("🕐 Verificando horários de funcionamento")

	// Buscar configuração de horários
	setting, err := s.settingsService.GetSetting(ctx, tenantID, "business_hours")
	if err != nil {
		log.Debug().Err(err).Msg("Horários não configurados")
		return ""
	}

	if setting.SettingValue == nil {
		log.Debug().Msg("SettingValue é nil")
		return ""
	}

	log.Info().Str("setting_value", *setting.SettingValue).Msg("🕐 Horários encontrados")

	var businessHours BusinessHours
	if err := json.Unmarshal([]byte(*setting.SettingValue), &businessHours); err != nil {
		log.Error().Err(err).Msg("Erro ao decodificar horários")
		return ""
	}

	// Determinar timezone
	timezone := businessHours.Timezone
	if timezone == "" {
		timezone = "America/Sao_Paulo"
	}

	// Obter horário atual no timezone da loja
	location, err := time.LoadLocation(timezone)
	if err != nil {
		log.Error().Err(err).Msg("Erro ao carregar timezone")
		location = time.UTC
	}

	now := time.Now().In(location)

	// Verificar se está aberto agora
	isOpen, nextTime := s.isStoreOpen(businessHours, now)

	// Construir mensagem sobre horários
	var hoursInfo string
	if isOpen {
		hoursInfo = "🟢 A loja está ABERTA agora."
		if nextTime != "" {
			hoursInfo += fmt.Sprintf(" Fechamos às %s.", nextTime)
		}
	} else {
		hoursInfo = "🔴 A loja está FECHADA agora."
		if nextTime != "" {
			hoursInfo += fmt.Sprintf(" Abrimos %s.", nextTime)
		}
	}

	// Adicionar horários da semana
	hoursInfo += "\n\nHORÁRIOS DE FUNCIONAMENTO:\n"
	hoursInfo += s.formatWeeklyHours(businessHours)

	log.Info().Str("hours_info", hoursInfo).Msg("🕐 Informações de horário geradas")

	return hoursInfo
}

// isStoreOpen verifica se a loja está aberta no momento atual
func (s *AIService) isStoreOpen(businessHours BusinessHours, now time.Time) (bool, string) {
	currentDay := s.getCurrentDayKey(now.Weekday())
	currentTime := now.Format("15:04")

	dayHours := s.getDayHours(businessHours, currentDay)
	if !dayHours.Enabled {
		// Loja fechada hoje, buscar próximo dia aberto
		nextDay, nextOpen := s.getNextOpenDay(businessHours, now)
		if nextDay != "" {
			return false, fmt.Sprintf("%s às %s", nextDay, nextOpen)
		}
		return false, ""
	}

	// Verificar se está no horário de funcionamento
	if currentTime >= dayHours.Open && currentTime < dayHours.Close {
		return true, dayHours.Close
	}

	// Loja fechada hoje, buscar próximo horário de abertura
	nextDay, nextOpen := s.getNextOpenTime(businessHours, now)
	if nextDay != "" {
		return false, fmt.Sprintf("%s às %s", nextDay, nextOpen)
	}

	return false, ""
}

// getCurrentDayKey converte time.Weekday para string
func (s *AIService) getCurrentDayKey(weekday time.Weekday) string {
	days := map[time.Weekday]string{
		time.Monday:    "monday",
		time.Tuesday:   "tuesday",
		time.Wednesday: "wednesday",
		time.Thursday:  "thursday",
		time.Friday:    "friday",
		time.Saturday:  "saturday",
		time.Sunday:    "sunday",
	}
	return days[weekday]
}

// getDayHours retorna os horários de um dia específico
func (s *AIService) getDayHours(businessHours BusinessHours, day string) DayHours {
	switch day {
	case "monday":
		return businessHours.Monday
	case "tuesday":
		return businessHours.Tuesday
	case "wednesday":
		return businessHours.Wednesday
	case "thursday":
		return businessHours.Thursday
	case "friday":
		return businessHours.Friday
	case "saturday":
		return businessHours.Saturday
	case "sunday":
		return businessHours.Sunday
	default:
		return DayHours{Enabled: false}
	}
}

// getNextOpenDay encontra o próximo dia que a loja abre
func (s *AIService) getNextOpenDay(businessHours BusinessHours, now time.Time) (string, string) {
	days := []string{"monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday"}
	dayNames := map[string]string{
		"monday":    "segunda-feira",
		"tuesday":   "terça-feira",
		"wednesday": "quarta-feira",
		"thursday":  "quinta-feira",
		"friday":    "sexta-feira",
		"saturday":  "sábado",
		"sunday":    "domingo",
	}

	currentDayIndex := int(now.Weekday())
	if currentDayIndex == 0 { // Sunday = 0, mas queremos Sunday = 6
		currentDayIndex = 6
	} else {
		currentDayIndex-- // Monday = 1 -> 0, etc.
	}

	// Verificar próximos 7 dias
	for i := 1; i <= 7; i++ {
		nextDayIndex := (currentDayIndex + i) % 7
		nextDayKey := days[nextDayIndex]
		dayHours := s.getDayHours(businessHours, nextDayKey)

		if dayHours.Enabled {
			dayName := dayNames[nextDayKey]
			if i == 1 {
				dayName = "amanhã"
			}
			return dayName, dayHours.Open
		}
	}

	return "", ""
}

// getNextOpenTime encontra o próximo horário de abertura (incluindo hoje)
func (s *AIService) getNextOpenTime(businessHours BusinessHours, now time.Time) (string, string) {
	currentDay := s.getCurrentDayKey(now.Weekday())
	currentTime := now.Format("15:04")

	dayHours := s.getDayHours(businessHours, currentDay)

	// Se ainda não passou do horário de abertura hoje
	if dayHours.Enabled && currentTime < dayHours.Open {
		return "hoje", dayHours.Open
	}

	// Buscar próximo dia
	return s.getNextOpenDay(businessHours, now)
}

// formatWeeklyHours formata os horários da semana
func (s *AIService) formatWeeklyHours(businessHours BusinessHours) string {
	days := []struct {
		key   string
		name  string
		hours DayHours
	}{
		{"monday", "Segunda", businessHours.Monday},
		{"tuesday", "Terça", businessHours.Tuesday},
		{"wednesday", "Quarta", businessHours.Wednesday},
		{"thursday", "Quinta", businessHours.Thursday},
		{"friday", "Sexta", businessHours.Friday},
		{"saturday", "Sábado", businessHours.Saturday},
		{"sunday", "Domingo", businessHours.Sunday},
	}

	var result string
	for _, day := range days {
		if day.hours.Enabled {
			result += fmt.Sprintf("• %s: %s às %s\n", day.name, day.hours.Open, day.hours.Close)
		} else {
			result += fmt.Sprintf("• %s: Fechado\n", day.name)
		}
	}

	return result
}

// generateClosedStoreGreeting gera uma saudação amigável quando a loja está fechada
func (s *AIService) generateClosedStoreGreeting(hoursInfo string) string {
	// Extrair informação do próximo horário de abertura
	lines := strings.Split(hoursInfo, "\n")
	var nextOpenTime string

	for _, line := range lines {
		if strings.Contains(line, "🔴 A loja está FECHADA") && strings.Contains(line, "Abrimos") {
			// Extrair apenas a parte "Abrimos [informação]"
			parts := strings.Split(line, "Abrimos ")
			if len(parts) > 1 {
				nextOpenTime = strings.TrimSuffix(parts[1], ".")
			}
			break
		}
	}

	// Gerar saudação personalizada
	var greeting string

	if nextOpenTime != "" {
		greeting = fmt.Sprintf("Oi! 😊\n\nObrigado pelo contato! No momento estamos fechados, mas abrimos %s.\n\nQuando voltarmos, estarei aqui para ajudar você com tudo que precisar!", nextOpenTime)
	} else {
		greeting = "Oi! 😊\n\nObrigado pelo contato! No momento estamos fechados, mas assim que voltarmos, estarei aqui para ajudar você com tudo que precisar!"
	}

	// Adicionar horários completos de forma mais amigável
	if strings.Contains(hoursInfo, "HORÁRIOS DE FUNCIONAMENTO:") {
		parts := strings.Split(hoursInfo, "HORÁRIOS DE FUNCIONAMENTO:\n")
		if len(parts) > 1 {
			scheduleInfo := strings.TrimSpace(parts[1])
			greeting += "\n\n📅 **Nossos horários:**\n" + scheduleInfo
		}
	}

	return greeting
}

// min retorna o menor entre dois números
// extractMedicationNames extrai nomes de medicamentos da análise da IA
func (s *AIService) extractMedicationNames(aiAnalysis string) []string {
	var medications []string

	// Converter para texto limpo
	text := strings.TrimSpace(aiAnalysis)
	lines := strings.Split(text, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Primeiro, tentar detectar lista simples separada por vírgula
		if strings.Contains(line, ",") && !strings.Contains(strings.ToLower(line), "medicamento") && !strings.Contains(strings.ToLower(line), "receita") {
			parts := strings.Split(line, ",")
			for _, part := range parts {
				med := strings.TrimSpace(part)
				if len(med) > 2 {
					medications = append(medications, med)
				}
			}
			continue
		}

		// Procurar por padrões comuns de listas de medicamentos
		patterns := []string{
			`^\d+\.?\s*(.+)$`,  // "1. Medicamento" ou "1 Medicamento"
			`^[•\-\*]\s*(.+)$`, // "• Medicamento", "- Medicamento", "* Medicamento"
			`^(.+):?\s*$`,      // "Medicamento:" ou apenas "Medicamento"
		}

		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			if matches := re.FindStringSubmatch(line); len(matches) > 1 {
				medication := strings.TrimSpace(matches[1])
				medication = strings.TrimSuffix(medication, ":")
				medication = strings.TrimSpace(medication)

				// Filtrar linhas muito curtas ou que não parecem nomes de medicamentos
				if len(medication) > 2 && !strings.Contains(strings.ToLower(medication), "medicamento") &&
					!strings.Contains(strings.ToLower(medication), "produtos") && !strings.Contains(strings.ToLower(medication), "catálogo") &&
					!strings.Contains(strings.ToLower(medication), "receita") && !strings.Contains(strings.ToLower(medication), "imagem") {
					medications = append(medications, medication)
				}
				break
			}
		}
	}

	// Se não encontrou com padrões, tentar extrair palavras que parecem nomes de medicamentos
	if len(medications) == 0 {
		words := strings.Fields(text)
		for _, word := range words {
			word = strings.TrimSpace(word)
			word = strings.Trim(word, ".,;:")
			// Medicamentos geralmente têm mais de 3 caracteres e começam com maiúscula
			if len(word) > 3 && word[0] >= 'A' && word[0] <= 'Z' &&
				!strings.Contains(strings.ToLower(word), "medicamento") &&
				!strings.Contains(strings.ToLower(word), "receita") &&
				!strings.Contains(strings.ToLower(word), "imagem") {
				medications = append(medications, word)
			}
		}
	}

	// Remover duplicatas e normalizar nomes
	seen := make(map[string]bool)
	var unique []string
	for _, med := range medications {
		// Manter o formato original para melhor correspondência
		med = strings.TrimSpace(med)
		if !seen[med] && len(med) > 2 {
			seen[med] = true
			unique = append(unique, med)
		}
	}

	return unique
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// extractSystemPrompt extracts the system prompt from messages
func extractSystemPrompt(messages []openai.ChatCompletionMessage) string {
	for _, msg := range messages {
		if msg.Role == openai.ChatMessageRoleSystem {
			return msg.Content
		}
	}
	return ""
}

// extractUserPrompt extracts the latest user message
func extractUserPrompt(messages []openai.ChatCompletionMessage) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == openai.ChatMessageRoleUser {
			return messages[i].Content
		}
	}
	return ""
}

// Implementação dos handlers para cada tool será no próximo arquivo

// combineProductResultsWithSequentialNumbering combina resultados de múltiplas funções
// renumerando produtos sequencialmente para evitar conflitos
func (s *AIService) combineProductResultsWithSequentialNumbering(results []string) string {
	combinedMessage := "🔍 **Resultados da sua busca:**\n\n"
	globalProductNumber := 1

	for sectionIndex, result := range results {
		// Limpar o resultado removendo cabeçalhos e instruções duplicadas
		cleanResult := s.cleanProductResult(result)

		if cleanResult == "" {
			continue
		}

		// Verificar se é uma resposta humanizada (mais barato/caro), manter intacta
		if strings.Contains(result, "mais barato é") || strings.Contains(result, "mais caro é") {
			combinedMessage += cleanResult + "\n\n"
			continue
		}

		// Para listas de produtos, extrair e renumerar
		sectionTitle := fmt.Sprintf("**Seção %d:**\n", sectionIndex+1)

		// Detectar o tipo de busca para um título mais específico
		if strings.Contains(result, "🔍 Encontrei") {
			// Extrair o termo da busca do resultado
			lines := strings.Split(cleanResult, "\n")
			for _, line := range lines {
				if strings.Contains(line, "🔍 Encontrei") && strings.Contains(line, "similares a") {
					// Extrair o termo entre aspas
					if start := strings.Index(line, "'"); start != -1 {
						if end := strings.Index(line[start+1:], "'"); end != -1 {
							searchTerm := line[start+1 : start+1+end]
							sectionTitle = fmt.Sprintf("**Produtos similares a '%s':**\n", searchTerm)
							break
						}
					}
				}
			}
		} else if strings.Contains(result, "🔍 **Filtros aplicados:** Busca:") {
			// Extrair o termo da busca do filtro
			lines := strings.Split(cleanResult, "\n")
			for _, line := range lines {
				if strings.Contains(line, "🔍 **Filtros aplicados:** Busca:") {
					if start := strings.Index(line, "'"); start != -1 {
						if end := strings.Index(line[start+1:], "'"); end != -1 {
							searchTerm := line[start+1 : start+1+end]
							sectionTitle = fmt.Sprintf("**Busca por '%s':**\n", searchTerm)
							break
						}
					}
				}
			}
		}

		combinedMessage += sectionTitle

		// Renumerar produtos sequencialmente
		renumberedResult := s.renumberProducts(cleanResult, &globalProductNumber)
		combinedMessage += renumberedResult + "\n\n"
	}

	// Adicionar instruções finais
	combinedMessage += "💡 Para ver detalhes: 'produto [número]'\n"
	combinedMessage += "🛒 Para adicionar: 'adicionar [número] quantidade [X]'"

	return combinedMessage
}

// cleanProductResult remove cabeçalhos e instruções duplicadas de um resultado
func (s *AIService) cleanProductResult(result string) string {
	cleanResult := result

	// Remover cabeçalhos duplicados
	cleanResult = strings.ReplaceAll(cleanResult, "🛍️ **Produtos disponíveis:**", "")
	cleanResult = strings.ReplaceAll(cleanResult, "🛍️ Produtos disponíveis:", "")
	cleanResult = strings.ReplaceAll(cleanResult, "🛍️", "")

	// Remover instruções duplicadas que aparecem em cada seção
	cleanResult = strings.ReplaceAll(cleanResult, "💡 Para ver detalhes, diga: 'produto [número]' ou 'produto [nome]'", "")
	cleanResult = strings.ReplaceAll(cleanResult, "🛒 Para adicionar ao carrinho: 'adicionar [número] quantidade [X]'", "")
	cleanResult = strings.ReplaceAll(cleanResult, "💡 Para ver detalhes: 'produto [número]' ou 'produto [nome]'", "")
	cleanResult = strings.ReplaceAll(cleanResult, "🛒 Para adicionar: 'adicionar [número] quantidade [X]'", "")
	cleanResult = strings.ReplaceAll(cleanResult, "💡 Para ver detalhes: 'produto [número]'", "")
	cleanResult = strings.ReplaceAll(cleanResult, "🛒 Para adicionar: 'adicionar [número] quantidade [X]'", "")
	cleanResult = strings.ReplaceAll(cleanResult, "📝 Para adicionar ao carrinho basta informar o numero do item ou nome do produto.", "")

	return strings.TrimSpace(cleanResult)
}

// renumberProducts renumera produtos em um resultado usando numeração global sequencial
func (s *AIService) renumberProducts(result string, globalNumber *int) string {
	lines := strings.Split(result, "\n")
	var processedLines []string

	for _, line := range lines {
		// Detectar linhas que começam com número seguido de ponto
		if matched := regexp.MustCompile(`^\s*\d+\.\s*`).MatchString(line); matched {
			// Substituir o número pela numeração global
			newLine := regexp.MustCompile(`^\s*\d+\.\s*`).ReplaceAllString(line, fmt.Sprintf("%d. ", *globalNumber))
			processedLines = append(processedLines, newLine)
			*globalNumber++
		} else {
			processedLines = append(processedLines, line)
		}
	}

	return strings.Join(processedLines, "\n")
}
