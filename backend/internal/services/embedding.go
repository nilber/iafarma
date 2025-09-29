package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"github.com/sashabaranov/go-openai"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gorm.io/gorm"
)

// basicAuth implements credentials.PerRPCCredentials for basic authentication
type basicAuth struct {
	username string
	password string
}

func (b *basicAuth) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + b.password,
	}, nil
}

func (b *basicAuth) RequireTransportSecurity() bool {
	return false
}

type EmbeddingService struct {
	openaiClient *openai.Client
	qdrantClient qdrant.CollectionsClient
	conn         *grpc.ClientConn
}

type ProductEmbedding struct {
	ID   string                 `json:"id"`
	Text string                 `json:"text"`
	Meta map[string]interface{} `json:"meta"`
}

// GetVectorCount returns the number of vectors for a specific tenant
func (s *EmbeddingService) GetVectorCount(tenantID string) (int, error) {
	ctx := context.Background()
	collectionName := s.GetProductCollectionName(tenantID)

	// Get collection info
	info, err := s.qdrantClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: collectionName,
	})
	if err != nil {
		// Collection doesn't exist or other error
		return 0, nil // Return 0 instead of error for non-existent collections
	}

	if info.Result == nil || info.Result.PointsCount == nil {
		return 0, nil
	}

	return int(*info.Result.PointsCount), nil
}

// GetAllVectorIDs returns all vector IDs for a specific tenant
func (s *EmbeddingService) GetAllVectorIDs(tenantID string) ([]string, error) {
	ctx := context.Background()
	collectionName := s.GetProductCollectionName(tenantID)

	pointsClient := qdrant.NewPointsClient(s.conn)
	var allIDs []string
	var offset *qdrant.PointId

	fmt.Printf("🔍 GetAllVectorIDs: Starting for tenant %s, collection %s\n", tenantID, collectionName)

	for {
		scrollRequest := &qdrant.ScrollPoints{
			CollectionName: collectionName,
			Limit:          &[]uint32{1000}[0], // Increased limit to 1000
			WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: false}},
			WithVectors:    &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: false}},
		}

		if offset != nil {
			scrollRequest.Offset = offset
		}

		response, err := pointsClient.Scroll(ctx, scrollRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to scroll points: %v", err)
		}

		if len(response.Result) == 0 {
			break
		}

		fmt.Printf("🔍 GetAllVectorIDs: Retrieved batch of %d vectors (total so far: %d)\n", len(response.Result), len(allIDs))

		// Extract IDs from response
		for _, point := range response.Result {
			if point.Id != nil {
				switch id := point.Id.PointIdOptions.(type) {
				case *qdrant.PointId_Uuid:
					allIDs = append(allIDs, id.Uuid)
				case *qdrant.PointId_Num:
					allIDs = append(allIDs, fmt.Sprintf("%d", id.Num))
				}
			}
		}

		// Update offset for next iteration
		if len(response.Result) > 0 {
			lastPoint := response.Result[len(response.Result)-1]
			offset = lastPoint.Id
		} else {
			break
		}

		// If we got less than requested, we're done
		if len(response.Result) < 1000 {
			break
		}
	}

	fmt.Printf("🔍 GetAllVectorIDs: Completed for tenant %s, found %d total vectors\n", tenantID, len(allIDs))
	return allIDs, nil
}

type VectorSearchResult struct {
	Metadata  map[string]interface{} `json:"metadata"`
	TenantID  string                 `json:"tenant_id"`
	ProductID string                 `json:"product_id"`
	CreatedAt time.Time              `json:"created_at"`
}

func NewEmbeddingService(openaiAPIKey string, qdrantURL string, qdrantPassword string) (*EmbeddingService, error) {
	// Configurar cliente OpenAI
	openaiClient := openai.NewClient(openaiAPIKey)

	// Configurar opções de conexão com Qdrant
	var dialOpts []grpc.DialOption

	// Adicionar credenciais se senha fornecida
	if qdrantPassword != "" {
		// Usar credenciais básicas para autenticação com Qdrant
		dialOpts = append(dialOpts, grpc.WithPerRPCCredentials(&basicAuth{
			username: "qdrant",
			password: qdrantPassword,
		}))
		log.Printf("🔐 Using authentication for Qdrant connection")
	}

	// Adicionar configuração de segurança
	dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// Configurar conexão com Qdrant - tentar URL fornecida primeiro
	conn, err := grpc.Dial(qdrantURL, dialOpts...)
	if err != nil {
		// Se falhar e a URL contém porta 6334, tentar 6333
		if strings.Contains(qdrantURL, ":6334") {
			fallbackURL := strings.Replace(qdrantURL, ":6334", ":6333", 1)
			log.Printf("⚠️ Failed to connect to %s, trying fallback %s", qdrantURL, fallbackURL)
			conn, err = grpc.Dial(fallbackURL, dialOpts...)
			if err == nil {
				log.Printf("✅ Successfully connected to Qdrant on fallback URL: %s", fallbackURL)
				qdrantURL = fallbackURL // Update for logging
			}
		}

		if err != nil {
			return nil, fmt.Errorf("failed to connect to Qdrant: %v", err)
		}
	}

	qdrantClient := qdrant.NewCollectionsClient(conn)

	service := &EmbeddingService{
		openaiClient: openaiClient,
		qdrantClient: qdrantClient,
		conn:         conn,
	}

	// Collections são criadas dinamicamente por tenant quando necessário
	log.Printf("✅ Embedding service initialized successfully")

	return service, nil
}

// GetProductCollectionName retorna o nome da collection de produtos para um tenant
func (s *EmbeddingService) GetProductCollectionName(tenantID string) string {
	return fmt.Sprintf("products_tenant_%s", tenantID)
}

// GetConversationCollectionName retorna o nome da collection de conversas para um tenant e customer
func (s *EmbeddingService) GetConversationCollectionName(tenantID, customerID string) string {
	return fmt.Sprintf("conversations_tenant_%s_customer_%s", tenantID, customerID)
}

func (s *EmbeddingService) Close() {
	if s.conn != nil {
		s.conn.Close()
	}
}

// CheckQdrantHealth verifica se a conexão com o Qdrant está funcionando
func (s *EmbeddingService) CheckQdrantHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Tentar listar as collections como um health check básico
	_, err := s.qdrantClient.List(ctx, &qdrant.ListCollectionsRequest{})
	if err != nil {
		return fmt.Errorf("Qdrant connection failed: %v", err)
	}

	return nil
}

func (s *EmbeddingService) GenerateEmbedding(text string) ([]float32, error) {
	ctx := context.Background()

	req := openai.EmbeddingRequest{
		Input: []string{text},
		Model: openai.SmallEmbedding3, // text-embedding-3-small
	}

	resp, err := s.openaiClient.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %v", err)
	}

	if len(resp.Data) == 0 {
		return nil, fmt.Errorf("no embedding generated")
	}

	return resp.Data[0].Embedding, nil
}

func (s *EmbeddingService) StoreProductEmbedding(productID, tenantID, text string, metadata map[string]interface{}) error {
	ctx := context.Background()
	collectionName := s.GetProductCollectionName(tenantID)

	// Garantir que a collection existe
	err := s.ensureProductCollection(collectionName)
	if err != nil {
		return fmt.Errorf("failed to ensure product collection: %v", err)
	}

	// Gerar embedding
	embedding, err := s.GenerateEmbedding(text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %v", err)
	}

	// Adicionar metadados extras
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["product_id"] = productID
	metadata["tenant_id"] = tenantID
	metadata["text"] = text
	metadata["created_at"] = time.Now().Unix()

	// Converter metadata para Qdrant payload
	payload, err := s.createPayload(metadata)
	if err != nil {
		return fmt.Errorf("failed to create payload: %v", err)
	}

	// Armazenar no Qdrant
	pointsClient := qdrant.NewPointsClient(s.conn)
	_, err = pointsClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points: []*qdrant.PointStruct{
			{
				Id: &qdrant.PointId{
					PointIdOptions: &qdrant.PointId_Uuid{
						Uuid: productID,
					},
				},
				Vectors: &qdrant.Vectors{
					VectorsOptions: &qdrant.Vectors_Vector{
						Vector: &qdrant.Vector{
							Data: embedding,
						},
					},
				},
				Payload: payload,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to store embedding in Qdrant: %v", err)
	}

	log.Printf("Product embedding stored successfully for product %s in tenant %s", productID, tenantID)
	return nil
}

func (s *EmbeddingService) SearchSimilarProducts(query string, tenantID string, limit uint64) ([]*ProductSearchResult, error) {
	ctx := context.Background()
	collectionName := s.GetProductCollectionName(tenantID)

	// Gerar embedding da query
	queryEmbedding, err := s.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %v", err)
	}

	// Buscar pontos similares no Qdrant
	pointsClient := qdrant.NewPointsClient(s.conn)

	// Como agora cada tenant tem sua própria collection, não precisamos filtrar por tenant_id
	// mas manteremos o filtro como segurança adicional
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "tenant_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: tenantID,
							},
						},
					},
				},
			},
		},
	}

	searchResp, err := pointsClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: collectionName,
		Vector:         queryEmbedding,
		Filter:         filter,
		Limit:          limit,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search in Qdrant: %v", err)
	}

	// Converter resultados
	results := make([]*ProductSearchResult, 0, len(searchResp.Result))
	for _, point := range searchResp.Result {
		result := &ProductSearchResult{
			Score: point.Score,
		}

		// Extrair dados do payload
		if payload := point.Payload; payload != nil {
			if productID, ok := payload["product_id"]; ok {
				if productIDStr, ok := productID.Kind.(*qdrant.Value_StringValue); ok {
					result.ProductID = productIDStr.StringValue
				}
			}
			if text, ok := payload["text"]; ok {
				if textStr, ok := text.Kind.(*qdrant.Value_StringValue); ok {
					result.Text = textStr.StringValue
				}
			}

			// Converter payload completo para metadata
			metadataBytes, _ := json.Marshal(payload)
			json.Unmarshal(metadataBytes, &result.Metadata)
		}

		results = append(results, result)
	}

	return results, nil
}

func (s *EmbeddingService) DeleteProductEmbedding(productID, tenantID string) error {
	ctx := context.Background()
	collectionName := s.GetProductCollectionName(tenantID)

	fmt.Printf("🗑️ Deleting vector %s from collection %s\n", productID, collectionName)

	pointsClient := qdrant.NewPointsClient(s.conn)
	_, err := pointsClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: collectionName,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{
						{
							PointIdOptions: &qdrant.PointId_Uuid{
								Uuid: productID,
							},
						},
					},
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to delete embedding from Qdrant: %v", err)
	}

	log.Printf("Product embedding deleted successfully for product %s", productID)
	return nil
}

func (s *EmbeddingService) createPayload(metadata map[string]interface{}) (map[string]*qdrant.Value, error) {
	payload := make(map[string]*qdrant.Value)

	for key, value := range metadata {
		var qdrantValue *qdrant.Value

		switch v := value.(type) {
		case string:
			qdrantValue = &qdrant.Value{
				Kind: &qdrant.Value_StringValue{StringValue: v},
			}
		case int:
			qdrantValue = &qdrant.Value{
				Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(v)},
			}
		case int64:
			qdrantValue = &qdrant.Value{
				Kind: &qdrant.Value_IntegerValue{IntegerValue: v},
			}
		case float64:
			qdrantValue = &qdrant.Value{
				Kind: &qdrant.Value_DoubleValue{DoubleValue: v},
			}
		case bool:
			qdrantValue = &qdrant.Value{
				Kind: &qdrant.Value_BoolValue{BoolValue: v},
			}
		default:
			// Para tipos complexos, converter para string JSON
			jsonBytes, err := json.Marshal(v)
			if err != nil {
				continue
			}
			qdrantValue = &qdrant.Value{
				Kind: &qdrant.Value_StringValue{StringValue: string(jsonBytes)},
			}
		}

		payload[key] = qdrantValue
	}

	return payload, nil
}

type ProductSearchResult struct {
	ProductID string                 `json:"product_id"`
	Score     float32                `json:"score"`
	Text      string                 `json:"text"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// ConversationEntry representa uma entrada de conversa para armazenar no Qdrant
type ConversationEntry struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	CustomerID string                 `json:"customer_id"`
	Message    string                 `json:"message"`
	Response   string                 `json:"response"`
	Timestamp  string                 `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// StoreConversation armazena uma conversa no Qdrant
func (s *EmbeddingService) StoreConversation(tenantID, customerID string, entry ConversationEntry) error {
	collectionName := s.GetConversationCollectionName(tenantID, customerID)

	// Criar collection se não existir
	err := s.ensureConversationCollection(collectionName)
	if err != nil {
		return fmt.Errorf("failed to ensure conversation collection: %w", err)
	}

	// Gerar embedding para a mensagem + resposta
	text := fmt.Sprintf("Mensagem: %s\nResposta: %s", entry.Message, entry.Response)
	embedding, err := s.GenerateEmbedding(text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Preparar metadata
	metadata := map[string]interface{}{
		"tenant_id":   entry.TenantID,
		"customer_id": entry.CustomerID,
		"message":     entry.Message,
		"response":    entry.Response,
		"timestamp":   entry.Timestamp,
		"created_at":  time.Now().Unix(),
	}

	// Adicionar metadata extra se fornecida
	for key, value := range entry.Metadata {
		metadata[key] = value
	}

	// Converter metadata para Qdrant payload
	payload, err := s.createPayload(metadata)
	if err != nil {
		return fmt.Errorf("failed to create payload: %w", err)
	}

	// Armazenar no Qdrant
	ctx := context.Background()
	pointsClient := qdrant.NewPointsClient(s.conn)
	_, err = pointsClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points: []*qdrant.PointStruct{
			{
				Id: &qdrant.PointId{
					PointIdOptions: &qdrant.PointId_Uuid{
						Uuid: entry.ID,
					},
				},
				Vectors: &qdrant.Vectors{
					VectorsOptions: &qdrant.Vectors_Vector{
						Vector: &qdrant.Vector{
							Data: embedding,
						},
					},
				},
				Payload: payload,
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to store conversation in Qdrant: %w", err)
	}

	log.Printf("Conversation stored successfully for tenant %s, customer %s", tenantID, customerID)
	return nil
}

// SearchConversations busca conversas similares para um tenant e customer
func (s *EmbeddingService) SearchConversations(tenantID, customerID, query string, limit int) ([]ConversationSearchResult, error) {
	collectionName := s.GetConversationCollectionName(tenantID, customerID)

	// Gerar embedding para a query
	embedding, err := s.GenerateEmbedding(query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Filtro por tenant e customer
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "tenant_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: tenantID,
							},
						},
					},
				},
			},
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "customer_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: customerID,
							},
						},
					},
				},
			},
		},
	}

	// Buscar no Qdrant
	ctx := context.Background()
	pointsClient := qdrant.NewPointsClient(s.conn)
	searchResult, err := pointsClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: collectionName,
		Vector:         embedding,
		Filter:         filter,
		Limit:          uint64(limit),
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to search conversations: %w", err)
	}

	var results []ConversationSearchResult
	for _, point := range searchResult.Result {
		result := ConversationSearchResult{
			ID:       point.Id.GetUuid(),
			Score:    point.Score,
			Metadata: make(map[string]interface{}),
		}

		// Extrair campos do payload
		if point.Payload != nil {
			for key, value := range point.Payload {
				switch key {
				case "message":
					if v := value.GetStringValue(); v != "" {
						result.Message = v
					}
				case "response":
					if v := value.GetStringValue(); v != "" {
						result.Response = v
					}
				case "timestamp":
					if v := value.GetStringValue(); v != "" {
						result.Timestamp = v
					}
				default:
					result.Metadata[key] = value
				}
			}
		}

		results = append(results, result)
	}

	return results, nil
}

// SearchConversationsWithMaxAge busca conversas similares para um tenant e customer filtrando por idade máxima
func (s *EmbeddingService) SearchConversationsWithMaxAge(tenantID, customerID, query string, limit int, maxAgeHours int) ([]ConversationSearchResult, error) {
	// Primeiro buscar todas as conversas similares sem filtro de idade
	allResults, err := s.SearchConversations(tenantID, customerID, query, limit*2) // Buscar mais para compensar a filtragem
	if err != nil {
		return nil, fmt.Errorf("failed to search conversations: %w", err)
	}

	// Calcular timestamp limite (conversas mais antigas que maxAgeHours não serão retornadas)
	cutoffTime := time.Now().Add(-time.Duration(maxAgeHours) * time.Hour)

	// Filtrar conversas por idade na aplicação
	var filteredResults []ConversationSearchResult
	for _, result := range allResults {
		if result.Timestamp == "" {
			continue // Pular conversas sem timestamp
		}

		// Parsear timestamp da conversa
		conversationTime, err := time.Parse(time.RFC3339, result.Timestamp)
		if err != nil {
			log.Printf("⚠️ Failed to parse conversation timestamp %s: %v", result.Timestamp, err)
			continue // Pular conversas com timestamp inválido
		}

		// Verificar se a conversa é mais recente que o cutoff
		if conversationTime.After(cutoffTime) {
			filteredResults = append(filteredResults, result)
		}

		// Parar quando atingir o limite desejado
		if len(filteredResults) >= limit {
			break
		}
	}

	log.Printf("SearchConversationsWithMaxAge: Found %d conversations newer than %s (filtered from %d total) for tenant %s, customer %s",
		len(filteredResults), cutoffTime.Format(time.RFC3339), len(allResults), tenantID, customerID)

	return filteredResults, nil
}

// CleanupOldConversations remove conversas antigas do Qdrant com base na idade máxima
func (s *EmbeddingService) CleanupOldConversations(tenantID, customerID string, maxAgeHours int) (int, error) {
	collectionName := s.GetConversationCollectionName(tenantID, customerID)

	// Calcular timestamp limite (conversas mais antigas que maxAgeHours serão removidas)
	cutoffTime := time.Now().Add(-time.Duration(maxAgeHours) * time.Hour)

	// Primeiro, vamos buscar todas as conversas do cliente para filtrar por idade
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "tenant_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: tenantID,
							},
						},
					},
				},
			},
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "customer_id",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{
								Keyword: customerID,
							},
						},
					},
				},
			},
		},
	}

	ctx := context.Background()
	pointsClient := qdrant.NewPointsClient(s.conn)

	// Buscar todos os pontos que correspondem ao filtro
	scrollResult, err := pointsClient.Scroll(ctx, &qdrant.ScrollPoints{
		CollectionName: collectionName,
		Filter:         filter,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		Limit:          func() *uint32 { limit := uint32(1000); return &limit }(), // Buscar até 1000 pontos por vez
	})

	if err != nil {
		return 0, fmt.Errorf("failed to scroll conversations for cleanup: %w", err)
	}

	// Filtrar pontos antigos para remoção
	var pointsToDelete []string
	for _, point := range scrollResult.Result {
		if point.Payload != nil {
			if timestampValue, exists := point.Payload["timestamp"]; exists {
				timestampStr := timestampValue.GetStringValue()
				if timestampStr != "" {
					conversationTime, err := time.Parse(time.RFC3339, timestampStr)
					if err != nil {
						log.Printf("⚠️ Failed to parse conversation timestamp %s during cleanup: %v", timestampStr, err)
						continue
					}

					// Se a conversa é mais antiga que o cutoff, marcar para remoção
					if conversationTime.Before(cutoffTime) {
						pointsToDelete = append(pointsToDelete, point.Id.GetUuid())
					}
				}
			}
		}
	}

	// Remover pontos antigos se houver algum
	deletedCount := 0
	if len(pointsToDelete) > 0 {
		deleteResult, err := pointsClient.Delete(ctx, &qdrant.DeletePoints{
			CollectionName: collectionName,
			Points: &qdrant.PointsSelector{
				PointsSelectorOneOf: &qdrant.PointsSelector_Points{
					Points: &qdrant.PointsIdsList{
						Ids: convertStringIDsToPointIds(pointsToDelete),
					},
				},
			},
		})

		if err != nil {
			return 0, fmt.Errorf("failed to delete old conversations: %w", err)
		}

		deletedCount = len(pointsToDelete)
		log.Printf("✅ Cleaned up %d old conversations (older than %s) for tenant %s, customer %s. Operation result: %v",
			deletedCount, cutoffTime.Format(time.RFC3339), tenantID, customerID, deleteResult.Result)
	} else {
		log.Printf("📭 No old conversations found to cleanup for tenant %s, customer %s", tenantID, customerID)
	}

	return deletedCount, nil
}

// convertStringIDsToPointIds converte slice de strings para slice de PointId
func convertStringIDsToPointIds(stringIDs []string) []*qdrant.PointId {
	pointIds := make([]*qdrant.PointId, len(stringIDs))
	for i, id := range stringIDs {
		pointIds[i] = &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Uuid{
				Uuid: id,
			},
		}
	}
	return pointIds
}

// ensureConversationCollection garante que a collection de conversas existe
func (s *EmbeddingService) ensureConversationCollection(collectionName string) error {
	ctx := context.Background()

	// Verificar se a collection já existe
	_, err := s.qdrantClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: collectionName,
	})

	if err == nil {
		return nil // Collection já existe
	}

	// Criar a collection
	_, err = s.qdrantClient.Create(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     1536, // OpenAI embedding dimension
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create conversation collection %s: %w", collectionName, err)
	}

	log.Printf("Created conversation collection: %s", collectionName)
	return nil
}

// ensureProductCollection garante que a collection de produtos existe para um tenant
func (s *EmbeddingService) ensureProductCollection(collectionName string) error {
	ctx := context.Background()

	// Verificar se a collection já existe
	_, err := s.qdrantClient.Get(ctx, &qdrant.GetCollectionInfoRequest{
		CollectionName: collectionName,
	})

	if err == nil {
		return nil // Collection já existe
	}

	// Criar a collection
	_, err = s.qdrantClient.Create(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     1536, // OpenAI embedding dimension
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create product collection %s: %w", collectionName, err)
	}

	log.Printf("Created product collection: %s", collectionName)
	return nil
}

type ConversationSearchResult struct {
	ID        string                 `json:"id"`
	Score     float32                `json:"score"`
	Message   string                 `json:"message"`
	Response  string                 `json:"response"`
	Timestamp string                 `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// SyncAllProductsFromDB sincroniza todos os produtos do PostgreSQL para o Qdrant usando processamento em lote
// Remove todos os embeddings existentes e recria com dados atuais do banco
func (s *EmbeddingService) SyncAllProductsFromDB(db *gorm.DB) error {
	log.Printf("🔄 Starting batch product sync from PostgreSQL to Qdrant...")

	// Buscar todos os produtos com seus tenants, incluindo embedding_hash
	var products []struct {
		ID            string `gorm:"column:id"`
		TenantID      string `gorm:"column:tenant_id"`
		Name          string `gorm:"column:name"`
		Description   string `gorm:"column:description"`
		Brand         string `gorm:"column:brand"`
		Tags          string `gorm:"column:tags"`
		Price         string `gorm:"column:price"`
		SalePrice     string `gorm:"column:sale_price"`
		SKU           string `gorm:"column:sku"`
		Weight        string `gorm:"column:weight"`
		EmbeddingHash string `gorm:"column:embedding_hash"`
	}

	err := db.Table("products").
		Find(&products).Error

	if err != nil {
		return fmt.Errorf("failed to fetch products from database: %w", err)
	}

	log.Printf("📦 Found %d products to sync", len(products))

	if len(products) == 0 {
		log.Printf("ℹ️ No products found to sync")
		return nil
	}

	// Agrupar produtos por tenant para otimizar as operações
	tenantProducts := make(map[string][]BatchProductData)
	var totalSkipped int

	for _, product := range products {
		// Construir texto para embedding
		var textParts []string
		if product.Name != "" {
			textParts = append(textParts, "Nome: "+product.Name)
		}
		if product.Description != "" {
			textParts = append(textParts, "Descrição: "+product.Description)
		}
		if product.Brand != "" {
			textParts = append(textParts, "Marca: "+product.Brand)
		}
		if product.Tags != "" {
			textParts = append(textParts, "Tags: "+product.Tags)
		}

		productText := strings.Join(textParts, ". ")

		// Calculate current hash
		currentHash := s.calculateContentHash(productText)

		// Skip if hash matches and product already has embedding
		if product.EmbeddingHash != "" && product.EmbeddingHash == currentHash {
			totalSkipped++
			continue
		}

		// Preparar metadados
		metadata := map[string]interface{}{
			"name":        product.Name,
			"description": product.Description,
			"brand":       product.Brand,
			"tags":        product.Tags,
			"price":       product.Price,
			"sale_price":  product.SalePrice,
			"sku":         product.SKU,
			"weight":      product.Weight,
			"sync_source": "postgresql",
			"synced_at":   time.Now().Unix(),
		}

		batchProduct := BatchProductData{
			ID:       product.ID,
			TenantID: product.TenantID,
			Text:     productText,
			Metadata: metadata,
		}

		tenantProducts[product.TenantID] = append(tenantProducts[product.TenantID], batchProduct)
	}

	log.Printf("📊 Sync analysis: %d products need processing, %d skipped (hash unchanged)",
		len(products)-totalSkipped, totalSkipped)

	// Processar cada tenant em lote
	totalSynced := 0
	totalErrors := 0
	batchSize := 500 // Optimized batch size for startup sync

	for tenantID, tenantProductList := range tenantProducts {
		log.Printf("🏢 Processing tenant: %s (%d products)", tenantID, len(tenantProductList))

		// Sincronizar produtos em lote
		err := s.StoreBatchProductEmbeddings(tenantID, tenantProductList, batchSize)
		if err != nil {
			log.Printf("❌ Failed to sync products for tenant %s: %v", tenantID, err)
			totalErrors += len(tenantProductList)
		} else {
			log.Printf("✅ Successfully synced %d products for tenant %s", len(tenantProductList), tenantID)
			totalSynced += len(tenantProductList)

			// Update hashes in database after successful sync
			s.updateProductHashesAfterSync(db, tenantProductList)
		}
	}

	log.Printf("✅ Batch product sync completed: %d synced, %d errors", totalSynced, totalErrors)
	return nil
}

// recreateProductCollection remove e recria uma collection de produtos
func (s *EmbeddingService) recreateProductCollection(collectionName string) error {
	ctx := context.Background()

	log.Printf("🗑️ Recreating product collection: %s", collectionName)

	// Tentar deletar a collection se existir
	_, err := s.qdrantClient.Delete(ctx, &qdrant.DeleteCollection{
		CollectionName: collectionName,
	})
	if err != nil {
		log.Printf("⚠️ Collection %s may not exist (delete failed): %v", collectionName, err)
		// Continuar mesmo se a deleção falhar - pode ser que a collection não exista
	}

	// Criar nova collection
	_, err = s.qdrantClient.Create(ctx, &qdrant.CreateCollection{
		CollectionName: collectionName,
		VectorsConfig: &qdrant.VectorsConfig{
			Config: &qdrant.VectorsConfig_Params{
				Params: &qdrant.VectorParams{
					Size:     1536, // OpenAI embedding dimension
					Distance: qdrant.Distance_Cosine,
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to create collection %s: %w", collectionName, err)
	}

	log.Printf("✅ Collection %s recreated successfully", collectionName)
	return nil
}

// BatchProductData representa os dados de um produto para processamento em lote
type BatchProductData struct {
	ID       string
	TenantID string
	Text     string
	Metadata map[string]interface{}
}

// StoreBatchProductEmbeddings armazena múltiplos produtos em lote no Qdrant
func (s *EmbeddingService) StoreBatchProductEmbeddings(tenantID string, products []BatchProductData, batchSize int) error {
	log.Printf("🔄 StoreBatchProductEmbeddings called with %d products, batchSize %d, tenant %s",
		len(products), batchSize, tenantID)

	if len(products) == 0 {
		log.Printf("ℹ️ StoreBatchProductEmbeddings: No products to process")
		return nil
	}

	collectionName := s.GetProductCollectionName(tenantID)
	ctx := context.Background()

	log.Printf("🔄 Collection name: %s", collectionName)

	// Garantir que a collection existe
	log.Printf("🔄 Ensuring product collection exists...")
	err := s.ensureProductCollection(collectionName)
	if err != nil {
		log.Printf("❌ Failed to ensure product collection: %v", err)
		return fmt.Errorf("failed to ensure product collection: %v", err)
	}
	log.Printf("✅ Product collection ensured successfully")

	log.Printf("📦 Processing %d products in batches of %d for tenant %s", len(products), batchSize, tenantID)

	// Processar em lotes
	for i := 0; i < len(products); i += batchSize {
		end := i + batchSize
		if end > len(products) {
			end = len(products)
		}

		batch := products[i:end]
		log.Printf("🔄 Processing batch %d-%d (%d products)", i+1, end, len(batch))

		err := s.processBatch(ctx, collectionName, batch)
		if err != nil {
			log.Printf("❌ Failed to process batch %d-%d: %v", i, end-1, err)
			return fmt.Errorf("failed to process batch %d-%d: %w", i, end-1, err)
		}

		log.Printf("✅ Processed batch %d-%d (%d products)", i+1, end, len(batch))
	}

	log.Printf("✅ StoreBatchProductEmbeddings completed successfully for tenant %s", tenantID)
	return nil
}

// StoreBatchProductEmbeddingsWithRecreate armazena múltiplos produtos em lote no Qdrant
// recriando a collection primeiro para garantir dados limpos
func (s *EmbeddingService) StoreBatchProductEmbeddingsWithRecreate(tenantID string, products []BatchProductData, batchSize int) error {
	if len(products) == 0 {
		return nil
	}

	collectionName := s.GetProductCollectionName(tenantID)
	ctx := context.Background()

	// Recriar a collection para garantir dados limpos
	err := s.recreateProductCollection(collectionName)
	if err != nil {
		return fmt.Errorf("failed to recreate product collection: %v", err)
	}

	log.Printf("📦 Processing %d products in batches of %d for tenant %s (with collection recreate)", len(products), batchSize, tenantID)

	// Processar em lotes
	for i := 0; i < len(products); i += batchSize {
		end := i + batchSize
		if end > len(products) {
			end = len(products)
		}

		batch := products[i:end]
		err := s.processBatch(ctx, collectionName, batch)
		if err != nil {
			return fmt.Errorf("failed to process batch %d-%d: %w", i, end-1, err)
		}

		log.Printf("✅ Processed batch %d-%d (%d products)", i+1, end, len(batch))
	}

	return nil
}

// processBatch processa um lote de produtos
func (s *EmbeddingService) processBatch(ctx context.Context, collectionName string, products []BatchProductData) error {
	log.Printf("🔄 processBatch called with %d products for collection %s", len(products), collectionName)

	// Preparar textos para gerar embeddings em lote
	texts := make([]string, len(products))
	for i, product := range products {
		texts[i] = product.Text
		log.Printf("📝 Product %d: ID=%s, Text preview: '%.50s...'", i+1, product.ID, product.Text)
	}

	// Gerar embeddings em lote usando OpenAI
	log.Printf("🔄 Generating batch embeddings for %d texts...", len(texts))
	embeddings, err := s.GenerateBatchEmbeddings(texts)
	if err != nil {
		log.Printf("❌ Failed to generate batch embeddings: %v", err)
		return fmt.Errorf("failed to generate batch embeddings: %w", err)
	}
	log.Printf("✅ Successfully generated %d embeddings", len(embeddings))

	if len(embeddings) != len(products) {
		log.Printf("❌ Embedding count mismatch: got %d embeddings for %d products", len(embeddings), len(products))
		return fmt.Errorf("embedding count mismatch: got %d embeddings for %d products", len(embeddings), len(products))
	}

	log.Printf("🔄 Preparing %d points for Qdrant insertion...", len(products))
	// Preparar pontos para inserção em lote no Qdrant
	points := make([]*qdrant.PointStruct, len(products))
	for i, product := range products {
		// Adicionar metadados extras
		metadata := make(map[string]interface{})
		for k, v := range product.Metadata {
			metadata[k] = v
		}
		metadata["product_id"] = product.ID
		metadata["tenant_id"] = product.TenantID
		metadata["text"] = product.Text
		metadata["created_at"] = time.Now().Unix()

		// Converter metadata para Qdrant payload
		payload, err := s.createPayload(metadata)
		if err != nil {
			return fmt.Errorf("failed to create payload for product %s: %w", product.ID, err)
		}

		points[i] = &qdrant.PointStruct{
			Id: &qdrant.PointId{
				PointIdOptions: &qdrant.PointId_Uuid{
					Uuid: product.ID,
				},
			},
			Vectors: &qdrant.Vectors{
				VectorsOptions: &qdrant.Vectors_Vector{
					Vector: &qdrant.Vector{
						Data: embeddings[i],
					},
				},
			},
			Payload: payload,
		}
	}

	log.Printf("🔄 Inserting %d points to Qdrant collection %s...", len(points), collectionName)
	// Inserir pontos em lote no Qdrant
	pointsClient := qdrant.NewPointsClient(s.conn)
	_, err = pointsClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points:         points,
	})

	if err != nil {
		log.Printf("❌ Failed to upsert batch to Qdrant: %v", err)
		return fmt.Errorf("failed to upsert batch to Qdrant: %w", err)
	}

	log.Printf("✅ Successfully inserted %d points to Qdrant collection %s", len(points), collectionName)
	return nil
}

// GenerateBatchEmbeddings gera embeddings para múltiplos textos em uma única chamada
func (s *EmbeddingService) GenerateBatchEmbeddings(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	ctx := context.Background()

	req := openai.EmbeddingRequest{
		Input: texts,
		Model: openai.SmallEmbedding3, // text-embedding-3-small
	}

	resp, err := s.openaiClient.CreateEmbeddings(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate batch embeddings: %v", err)
	}

	if len(resp.Data) != len(texts) {
		return nil, fmt.Errorf("embedding count mismatch: requested %d, got %d", len(texts), len(resp.Data))
	}

	embeddings := make([][]float32, len(resp.Data))
	for i, data := range resp.Data {
		embeddings[i] = data.Embedding
	}

	return embeddings, nil
}

// calculateContentHash generates a hash of the product content for caching
func (s *EmbeddingService) calculateContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])[:16] // Use first 16 chars
}

// updateProductHashesAfterSync updates embedding hashes for products after successful sync
func (s *EmbeddingService) updateProductHashesAfterSync(db *gorm.DB, products []BatchProductData) {
	for _, product := range products {
		hash := s.calculateContentHash(product.Text)
		if err := db.Model(&struct {
			ID string `gorm:"column:id"`
		}{}).Table("products").Where("id = ?", product.ID).Update("embedding_hash", hash).Error; err != nil {
			log.Printf("Failed to update embedding hash for product %s: %v", product.ID, err)
		}
	}
	log.Printf("✅ Updated embedding hashes for %d products", len(products))
}
