package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/sashabaranov/go-openai"
)

// ProductImageAnalysisRequest represents a request to analyze an image for products
type ProductImageAnalysisRequest struct {
	ImageURL string `json:"image_url"`
}

// ProductImageAnalysisResponse represents the AI response for product analysis
type ProductImageAnalysisResponse struct {
	Products []DetectedProduct `json:"products"`
}

// DetectedProduct represents a product detected in an image
type DetectedProduct struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Price       string   `json:"price"`
	Tags        []string `json:"tags"`
}

// ProductImageAnalysisService handles AI analysis of product images
type ProductImageAnalysisService struct {
	client *openai.Client
}

// NewProductImageAnalysisService creates a new product image analysis service
func NewProductImageAnalysisService(apiKey string) *ProductImageAnalysisService {
	client := openai.NewClient(apiKey)
	return &ProductImageAnalysisService{
		client: client,
	}
}

// AnalyzeProductImage analyzes an image to extract product information
func (s *ProductImageAnalysisService) AnalyzeProductImage(ctx context.Context, imageURL string) (*ProductImageAnalysisResponse, error) {
	prompt := `
Você é um assistente especializado em análise de imagens de produtos para e-commerce. 
Analise a imagem fornecida e extraia informações dos produtos visíveis.

Para cada produto identificado, forneça:
1. Nome claro e descritivo
2. Descrição detalhada e atrativa (2-3 frases)
3. Preço (se visível na imagem, caso contrário deixe vazio)
4. Tags relevantes para categorização (máximo 5 tags por produto)

REGRAS IMPORTANTES:
- Identifique TODOS os produtos visíveis na imagem
- Se for um cardápio, analise cada item listado
- Se for uma foto de produtos, identifique cada produto individual
- Responda APENAS com um JSON válido, sem texto adicional antes ou depois
- Use o formato JSON exato especificado abaixo
- Se não conseguir identificar produtos, retorne: {"products": []}
- Sempre use aspas duplas no JSON
- Sempre termine com chave fechada

FORMATO OBRIGATÓRIO (resposta completa):
{
  "products": [
    {
      "name": "Nome do Produto",
      "description": "Descrição detalhada e atrativa do produto",
      "price": "25.90",
      "tags": ["tag1", "tag2", "tag3"]
    }
  ]
}
`

	resp, err := s.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleUser,
					MultiContent: []openai.ChatMessagePart{
						{
							Type: openai.ChatMessagePartTypeText,
							Text: prompt,
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
			MaxTokens:   8000, // Increased to handle longer responses with many products
			Temperature: 0.3,  // Lower temperature for more consistent results
		},
	)

	if err != nil {
		fmt.Printf("OpenAI API call failed: %v\n", err)
		return nil, fmt.Errorf("failed to analyze image with AI: %w", err)
	}

	fmt.Printf("OpenAI Response received. Choices count: %d\n", len(resp.Choices))

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no response from AI")
	}

	content := resp.Choices[0].Message.Content
	fmt.Printf("Raw AI Response Content (length=%d): %s\n", len(content), content)

	content = strings.TrimSpace(content)
	fmt.Printf("Trimmed AI Response Content (length=%d): %s\n", len(content), content)

	if len(content) == 0 {
		return nil, fmt.Errorf("empty response from AI")
	}

	// Try to extract JSON from the response
	var result ProductImageAnalysisResponse
	if err := json.Unmarshal([]byte(content), &result); err != nil {
		fmt.Printf("Direct JSON unmarshal failed: %v\n", err)

		// If direct unmarshal fails, try to find JSON in the response
		startIdx := strings.Index(content, "{")
		endIdx := strings.LastIndex(content, "}") + 1

		fmt.Printf("JSON extraction indices: start=%d, end=%d\n", startIdx, endIdx)

		if startIdx >= 0 && endIdx > startIdx {
			jsonContent := content[startIdx:endIdx]
			fmt.Printf("Extracted JSON (length=%d): %s\n", len(jsonContent), jsonContent)

			// Try to fix incomplete JSON by ensuring it ends properly
			if !strings.HasSuffix(strings.TrimSpace(jsonContent), "}") {
				fmt.Printf("JSON appears incomplete, attempting to fix...\n")

				// Try to find the last complete product entry
				lastProductStart := strings.LastIndex(jsonContent, `"name":`)
				if lastProductStart > 0 {
					// Find the opening brace of the last product
					lastProductBrace := strings.LastIndex(jsonContent[:lastProductStart], "{")
					if lastProductBrace > 0 {
						// Truncate to the product before the incomplete one
						jsonContent = jsonContent[:lastProductBrace] + "]}"
						fmt.Printf("Fixed JSON (length=%d): %s\n", len(jsonContent), jsonContent)
					}
				}
			}

			if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
				fmt.Printf("Extracted JSON unmarshal failed: %v\n", err)

				// As a last resort, try to parse individual products from the content
				products := extractProductsFromPartialJSON(content)
				if len(products) > 0 {
					fmt.Printf("Recovered %d products from partial JSON\n", len(products))
					return &ProductImageAnalysisResponse{Products: products}, nil
				}

				return nil, fmt.Errorf("failed to parse AI response as JSON: %w\nOriginal content: %s", err, content)
			}
		} else {
			fmt.Printf("No JSON boundaries found in response\n")
			// Try to return empty result instead of error if no JSON found
			return &ProductImageAnalysisResponse{Products: []DetectedProduct{}}, nil
		}
	}

	// Validate the result
	fmt.Printf("Successfully parsed AI response. Products found: %d\n", len(result.Products))

	// Log each product for debugging
	for i, product := range result.Products {
		fmt.Printf("Product %d: Name='%s', Price='%s', Tags=%v\n",
			i+1, product.Name, product.Price, product.Tags)
	}

	return &result, nil
}

// GetImageAnalysisCreditCost returns the credit cost for image analysis
func (s *ProductImageAnalysisService) GetImageAnalysisCreditCost() int {
	return 1000 // 1000 credits per image analysis
}

// extractProductsFromPartialJSON attempts to extract product information from partial/malformed JSON
func extractProductsFromPartialJSON(content string) []DetectedProduct {
	var products []DetectedProduct

	// Find all product entries using regex
	nameRegex := regexp.MustCompile(`"name":\s*"([^"]+)"`)
	descRegex := regexp.MustCompile(`"description":\s*"([^"]+)"`)
	priceRegex := regexp.MustCompile(`"price":\s*"([^"]*)"`)
	tagsRegex := regexp.MustCompile(`"tags":\s*\[([^\]]*)\]`)

	names := nameRegex.FindAllStringSubmatch(content, -1)
	descriptions := descRegex.FindAllStringSubmatch(content, -1)
	prices := priceRegex.FindAllStringSubmatch(content, -1)
	tags := tagsRegex.FindAllStringSubmatch(content, -1)

	// Take the minimum count to avoid index out of bounds
	count := len(names)
	if len(descriptions) < count {
		count = len(descriptions)
	}
	if len(prices) < count {
		count = len(prices)
	}
	if len(tags) < count {
		count = len(tags)
	}

	for i := 0; i < count; i++ {
		product := DetectedProduct{
			Name:        names[i][1],
			Description: descriptions[i][1],
			Price:       prices[i][1],
		}

		// Parse tags
		if i < len(tags) {
			tagString := tags[i][1]
			tagItems := strings.Split(tagString, ",")
			for _, tag := range tagItems {
				tag = strings.Trim(strings.TrimSpace(tag), `"`)
				if tag != "" {
					product.Tags = append(product.Tags, tag)
				}
			}
		}

		products = append(products, product)
	}

	return products
}
