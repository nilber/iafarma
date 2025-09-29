package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// ProductGenerationRequest represents the request for product generation
type ProductGenerationRequest struct {
	ProductName string `json:"product_name" validate:"required"`
}

// ProductGenerationResponse represents the AI-generated product information
type ProductGenerationResponse struct {
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
	Suggestions struct {
		Brand    string   `json:"brand,omitempty"`
		Category string   `json:"category,omitempty"`
		Keywords []string `json:"keywords,omitempty"`
	} `json:"suggestions"`
}

// ProductAIService handles AI operations for products
type ProductAIService struct {
	client *openai.Client
}

// NewProductAIService creates a new product AI service
func NewProductAIService(apiKey string) *ProductAIService {
	client := openai.NewClient(apiKey)
	return &ProductAIService{
		client: client,
	}
}

// GenerateProductInfo generates product description and tags based on product name
func (s *ProductAIService) GenerateProductInfo(ctx context.Context, productName string) (*ProductGenerationResponse, error) {
	prompt := fmt.Sprintf(`
Você é um especialista em e-commerce brasileiro. Com base no nome do produto "%s", gere as seguintes informações em português brasileiro:

1. Uma descrição detalhada e atrativa do produto (2-3 parágrafos)
2. Tags relevantes para SEO e categorização (5-8 tags)
3. Sugestões de marca, categoria e palavras-chave relacionadas

IMPORTANTE: Responda APENAS com um JSON válido no formato abaixo, sem texto adicional:

{
  "description": "Descrição detalhada do produto aqui...",
  "tags": ["tag1", "tag2", "tag3", "tag4", "tag5"],
  "suggestions": {
    "brand": "Marca sugerida (opcional)",
    "category": "Categoria sugerida",
    "keywords": ["palavra1", "palavra2", "palavra3"]
  }
}

Nome do produto: %s`, productName, productName)

	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: prompt,
				},
			},
			MaxTokens:   1000,
			Temperature: 0.7,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to generate AI response: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	content := resp.Choices[0].Message.Content
	content = strings.TrimSpace(content)

	// Try to extract JSON from the response
	var result ProductGenerationResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// If direct unmarshal fails, try to find JSON in the response
		startIdx := strings.Index(content, "{")
		endIdx := strings.LastIndex(content, "}") + 1

		if startIdx >= 0 && endIdx > startIdx {
			jsonContent := content[startIdx:endIdx]
			if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
				return nil, fmt.Errorf("failed to parse AI response as JSON: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no valid JSON found in AI response")
		}
	}

	// Validate the response
	if result.Description == "" {
		return nil, fmt.Errorf("AI did not generate a description")
	}

	if len(result.Tags) == 0 {
		result.Tags = []string{productName}
	}

	return &result, nil
}

// GetUsageEstimate returns the estimated credit cost for generating product info
func (s *ProductAIService) GetUsageEstimate(productName string) int {
	// Estimate based on token usage - simple calculation
	// For product generation, we typically use 1 credit per generation
	return 1
}
