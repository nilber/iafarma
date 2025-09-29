package services

import (
	"context"
	"fmt"
	"iafarma/pkg/models"
	"math/rand"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TenantSettingsService struct {
	db *gorm.DB
}

func NewTenantSettingsService(db *gorm.DB) *TenantSettingsService {
	return &TenantSettingsService{db: db}
}

// GetSetting retrieves a specific setting for a tenant
func (s *TenantSettingsService) GetSetting(ctx context.Context, tenantID uuid.UUID, key string) (*models.TenantSetting, error) {
	var setting models.TenantSetting
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND setting_key = ? AND is_active = true", tenantID, key).
		First(&setting).Error

	if err != nil {
		return nil, err
	}

	return &setting, nil
}

// GetAllSettings retrieves all settings for a tenant
func (s *TenantSettingsService) GetAllSettings(ctx context.Context, tenantID uuid.UUID) ([]models.TenantSetting, error) {
	var settings []models.TenantSetting
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND is_active = true", tenantID).
		Find(&settings).Error

	return settings, err
}

// SetSetting creates or updates a setting for a tenant
func (s *TenantSettingsService) SetSetting(ctx context.Context, tenantID uuid.UUID, key string, value *string, settingType string) error {
	setting := models.TenantSetting{
		TenantID:     tenantID,
		SettingKey:   key,
		SettingValue: value,
		SettingType:  settingType,
		IsActive:     true,
	}

	return s.db.WithContext(ctx).
		Where("tenant_id = ? AND setting_key = ?", tenantID, key).
		Assign(setting).
		FirstOrCreate(&setting).Error
}

// GenerateAIProductExamples generates product examples based on tenant's actual products
func (s *TenantSettingsService) GenerateAIProductExamples(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	var products []models.Product
	err := s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("RANDOM()").
		Limit(6).
		Find(&products).Error

	if err != nil {
		return nil, err
	}

	if len(products) == 0 {
		// Fallback para exemplos genéricos se não houver produtos
		return []string{
			"quero 3 produtos",
			"adicionar 2 itens",
			"comprar 1 unidade",
		}, nil
	}

	var examples []string
	quantities := []int{1, 2, 3, 5}
	verbs := []string{"quero", "adicionar", "comprar", "preciso de"}

	for i, product := range products {
		if i >= 3 { // Máximo 3 exemplos
			break
		}

		quantity := quantities[rand.Intn(len(quantities))]
		verb := verbs[rand.Intn(len(verbs))]

		// Extrair palavra-chave principal do nome do produto
		productName := strings.ToLower(product.Name)
		words := strings.Fields(productName)
		mainWord := words[0]
		if len(words) > 1 {
			// Se tem mais palavras, pegar a mais significativa
			for _, word := range words {
				if len(word) > len(mainWord) && !isCommonWord(word) {
					mainWord = word
				}
			}
		}

		example := fmt.Sprintf("%s %d %s", verb, quantity, mainWord)
		examples = append(examples, example)
	}

	if len(examples) == 0 {
		examples = []string{
			"quero 3 produtos",
			"adicionar 2 itens",
			"comprar 1 unidade",
		}
	}

	return examples, nil
}

// GenerateWelcomeMessage generates a welcome message based on tenant's business type
func (s *TenantSettingsService) GenerateWelcomeMessage(ctx context.Context, tenantID uuid.UUID) (string, error) {
	examples, err := s.GenerateAIProductExamples(ctx, tenantID)
	if err != nil {
		return "", err
	}

	// Determinar tipo de negócio baseado nos produtos
	businessType := s.detectBusinessType(ctx, tenantID)

	var greeting string
	switch businessType {
	case "papelaria":
		greeting = "Oi! Bem-vindo à nossa papelaria! 📝 Como posso te ajudar hoje?"
	case "farmacia":
		greeting = "Olá! Bem-vindo à nossa farmácia! 💊 Em que posso ajudá-lo hoje?"
	case "cosmeticos":
		greeting = "Oi! Bem-vindo à nossa loja de cosméticos! 💄 Como posso te ajudar hoje?"
	case "alimentacao":
		greeting = "Olá! Bem-vindo ao nosso mercado! 🛒 Em que posso ajudá-lo hoje?"
	default:
		greeting = "Oi! Tudo certo? Como posso te ajudar hoje?"
	}

	message := fmt.Sprintf(`%s

Se já souber o item e a quantidade, é só mandar tudo junto, que eu adiciono direto ao carrinho. Exemplos:`, greeting)

	for _, example := range examples {
		message += fmt.Sprintf("\n• %s", example)
	}

	return message, nil
}

// detectBusinessType tries to detect business type based on products
func (s *TenantSettingsService) detectBusinessType(ctx context.Context, tenantID uuid.UUID) string {
	var products []models.Product
	s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Limit(20).
		Find(&products)

	if len(products) == 0 {
		return "general"
	}

	// Contar palavras-chave por categoria
	categories := map[string][]string{
		"papelaria":   {"caneta", "lápis", "caderno", "papel", "agenda", "borracha", "régua", "estojo", "marca", "texto"},
		"farmacia":    {"remédio", "medicamento", "vitamina", "sabonete", "shampoo", "creme", "protetor", "band", "aid"},
		"cosmeticos":  {"batom", "base", "rímel", "sombra", "blush", "perfume", "hidratante", "maquiagem", "esmalte"},
		"alimentacao": {"arroz", "feijão", "açúcar", "café", "leite", "pão", "carne", "frango", "verdura", "fruta"},
	}

	scores := make(map[string]int)

	for _, product := range products {
		productText := strings.ToLower(product.Name + " " + product.Description + " " + product.Tags)

		for category, keywords := range categories {
			for _, keyword := range keywords {
				if strings.Contains(productText, keyword) {
					scores[category]++
				}
			}
		}
	}

	// Encontrar categoria com maior score
	maxScore := 0
	detectedType := "general"
	for category, score := range scores {
		if score > maxScore {
			maxScore = score
			detectedType = category
		}
	}

	if maxScore == 0 {
		return "general"
	}

	return detectedType
}

// isCommonWord checks if a word is too common to be useful in examples
func isCommonWord(word string) bool {
	commonWords := []string{"de", "da", "do", "com", "para", "sem", "por", "em", "e", "ou", "kit", "pack", "unidade", "peça"}
	for _, common := range commonWords {
		if strings.EqualFold(word, common) {
			return true
		}
	}
	return false
}
