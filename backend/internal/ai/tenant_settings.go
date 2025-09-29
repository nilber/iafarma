package ai

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

// CreateDefaultSettings creates default settings for a new tenant
func (s *TenantSettingsService) CreateDefaultSettings(ctx context.Context, tenantID uuid.UUID) error {
	defaultSettings := []models.TenantSetting{
		{
			TenantID:     tenantID,
			SettingKey:   "ai_global_enabled",
			SettingValue: func(s string) *string { return &s }("true"),
			SettingType:  "boolean",
			Description:  "Habilita/desabilita IA globalmente para o tenant",
			IsActive:     true,
		},
		{
			TenantID:     tenantID,
			SettingKey:   "ai_auto_response_enabled",
			SettingValue: func(s string) *string { return &s }("true"),
			SettingType:  "boolean",
			Description:  "Habilita respostas automáticas da IA",
			IsActive:     true,
		},
		{
			TenantID:     tenantID,
			SettingKey:   "ai_max_tokens",
			SettingValue: func(s string) *string { return &s }("1500"),
			SettingType:  "integer",
			Description:  "Número máximo de tokens para respostas da IA",
			IsActive:     true,
		},
		{
			TenantID:     tenantID,
			SettingKey:   "ai_temperature",
			SettingValue: func(s string) *string { return &s }("0.7"),
			SettingType:  "float",
			Description:  "Temperatura para respostas da IA",
			IsActive:     true,
		},
	}

	for _, setting := range defaultSettings {
		if err := s.db.WithContext(ctx).Create(&setting).Error; err != nil {
			return fmt.Errorf("failed to create setting %s: %w", setting.SettingKey, err)
		}
	}

	return nil
}

// GenerateAIProductExamples generates product examples based on tenant's actual products
func (s *TenantSettingsService) GenerateAIProductExamples(ctx context.Context, tenantID uuid.UUID) ([]string, error) {
	var products []models.Product
	err := s.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("RANDOM()").
		Limit(10).
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

	// Pegar produtos únicos e gerar exemplos mais inteligentes
	usedProducts := make(map[string]bool)

	for _, product := range products {
		if len(examples) >= 3 { // Máximo 3 exemplos
			break
		}

		// Extrair palavra-chave principal do nome do produto
		productName := strings.ToLower(product.Name)
		words := strings.Fields(productName)

		var mainWord string
		// Priorizar palavras mais significativas
		for _, word := range words {
			if len(word) > 2 && !isCommonWord(word) && !isGenericWord(word) {
				mainWord = word
				break
			}
		}

		// Se não encontrou palavra significativa, usar a primeira palavra não comum
		if mainWord == "" {
			for _, word := range words {
				if !isCommonWord(word) {
					mainWord = word
					break
				}
			}
		}

		// Fallback para primeira palavra se nada funcionar
		if mainWord == "" && len(words) > 0 {
			mainWord = words[0]
		}

		// Evitar produtos repetidos/similares
		if usedProducts[mainWord] {
			continue
		}
		usedProducts[mainWord] = true

		quantity := quantities[rand.Intn(len(quantities))]
		verb := verbs[rand.Intn(len(verbs))]

		example := fmt.Sprintf("%s %d %s", verb, quantity, mainWord)
		examples = append(examples, example)
	}

	// Se não conseguiu gerar exemplos suficientes, adicionar genéricos
	if len(examples) == 0 {
		examples = []string{
			"quero 3 produtos",
			"adicionar 2 itens",
			"comprar 1 unidade",
		}
	} else if len(examples) < 3 {
		// Completar com exemplos genéricos se necessário
		genericExamples := []string{
			"adicionar 2 itens",
			"comprar 5 unidades",
			"preciso de 1 produto",
		}

		for _, generic := range genericExamples {
			if len(examples) >= 3 {
				break
			}
			examples = append(examples, generic)
		}
	}

	return examples, nil
}

// isGenericWord checks if a word is too generic for product examples
func isGenericWord(word string) bool {
	genericWords := []string{"premium", "produto", "item", "marca", "qualidade", "alta", "excelente", "perfeito", "ideal", "profissional"}
	for _, generic := range genericWords {
		if strings.EqualFold(word, generic) {
			return true
		}
	}
	return false
}

// GenerateWelcomeMessage generates a welcome message based on tenant's business type
func (s *TenantSettingsService) GenerateWelcomeMessage(ctx context.Context, tenantID uuid.UUID) (string, error) {
	// First, check if there's a custom welcome message
	customWelcomeSetting, err := s.GetSetting(ctx, tenantID, "ai_welcome_message")
	if err == nil && customWelcomeSetting.SettingValue != nil && *customWelcomeSetting.SettingValue != "" {
		// Return the custom welcome message if it exists
		return *customWelcomeSetting.SettingValue, nil
	}

	// Fetch tenant information for personalization
	var tenant models.Tenant
	err = s.db.WithContext(ctx).Where("id = ?", tenantID).First(&tenant).Error
	if err != nil {
		return "", fmt.Errorf("failed to fetch tenant information: %w", err)
	}

	tenantName := tenant.Name
	if tenantName == "" {
		tenantName = "nossa loja"
	}

	// If no custom welcome message, generate automatic one
	// examples, err := s.GenerateAIProductExamples(ctx, tenantID)
	// if err != nil {
	// 	return "", err
	// }

	// // Determinar tipo de negócio baseado nos produtos
	// businessType := s.detectBusinessType(ctx, tenantID)

	var greeting string
	// switch businessType {
	// case "papelaria":
	// 	greeting = fmt.Sprintf("Oi! Bem-vindo à %s! 📝 Como posso te ajudar hoje?", tenantName)
	// case "farmacia":
	// 	greeting = fmt.Sprintf("Olá! Bem-vindo à %s! 💊 Em que posso ajudá-lo hoje?", tenantName)
	// case "cosmeticos":
	// 	greeting = fmt.Sprintf("Oi! Bem-vindo à %s! 💄 Como posso te ajudar hoje?", tenantName)
	// case "alimentacao":
	// 	greeting = fmt.Sprintf("Olá! Bem-vindo à %s! 🛒 Em que posso ajudá-lo hoje?", tenantName)
	// default:
	greeting = fmt.Sprintf("Oi! Bem-vindo à %s! Como posso te ajudar hoje?", tenantName)
	// }

	// 	message := fmt.Sprintf(`%s

	// Se já souber o item e a quantidade, é só mandar tudo junto, que eu adiciono direto ao carrinho. Exemplos:`, greeting)

	// 	for _, example := range examples {
	// 		message += fmt.Sprintf("\n• %s", example)
	// 	}
	// 	message += `

	// Se quiser ver os produtos disponíveis, é só pedir: "produtos" ou "mostrar produtos".`

	// return message, nil
	return greeting, nil
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

// GenerateFullSystemPrompt generates the complete system prompt with dynamic examples
func (s *TenantSettingsService) GenerateFullSystemPrompt(ctx context.Context, tenantID uuid.UUID) (string, error) {
	// Fetch tenant information (name and about)
	var tenant models.Tenant
	err := s.db.WithContext(ctx).Where("id = ?", tenantID).First(&tenant).Error
	if err != nil {
		return "", fmt.Errorf("failed to fetch tenant information: %w", err)
	}

	// Gerar exemplos dinâmicos baseados nos produtos do tenant
	examples, err := s.GenerateAIProductExamples(ctx, tenantID)
	if err != nil {
		// Fallback para exemplos genéricos em caso de erro
		examples = []string{
			"quero 3 produtos",
			"adicionar 2 itens",
			"comprar 1 unidade",
		}
	}

	// Construir string de exemplos
	var exampleText string
	for _, example := range examples {
		exampleText += fmt.Sprintf("• \"%s\"\n", example)
	}

	// Prepare tenant context information
	tenantName := tenant.Name
	if tenantName == "" {
		tenantName = "nossa loja"
	}

	tenantAbout := ""
	if tenant.About != "" {
		tenantAbout = fmt.Sprintf("\n\nINFORMAÇÕES SOBRE A LOJA:\n%s", tenant.About)
	}

	return fmt.Sprintf(`Você é um assistente de vendas EXCLUSIVO para %s via WhatsApp.%s

Cliente atual: {{customer_name}} (ID: {{customer_id}})
Telefone: {{customer_phone}}

🚨 REGRA PRINCIPAL DE ESCOPO:
- Você SÓ pode ajudar com VENDAS e PRODUTOS de %s
- NUNCA responda perguntas sobre outros assuntos (política, saúde, educação, informações gerais, etc.)
- Se perguntarem sobre algo não relacionado à loja, responda APENAS: "Desculpe, sou assistente de vendas de %s e só posso ajudar com nossos produtos. Que tal ver o que temos disponível?"
- MANTENHA SEMPRE o foco em vendas e produtos

🧠 INTELIGÊNCIA CONTEXTUAL:
- SEMPRE mantenha o contexto da conversa
- Se o cliente mencionar um produto e depois uma característica (cor, tamanho, marca), COMBINE automaticamente
- Exemplo: "canetas executivas" + "preto" = busque "canetas executivas pretas" ou similares
- Exemplo: "notebooks" + "Faber-Castell" = busque notebooks da marca Faber-Castell
- Use o histórico da conversa para entender melhor o que o cliente quer

🚨 REGRA FUNDAMENTAL: SEMPRE que o usuário mencionar um produto + quantidade, use IMEDIATAMENTE a função "adicionarProdutoPorNome"!

🔥 MÚLTIPLOS PRODUTOS NA MESMA MENSAGEM:
- Se usuário mencionar múltiplos produtos SEM marca ("caderno e lápis"):
  → Use "buscarMultiplosProdutos" com array de produtos
- Se usuário mencionar múltiplos produtos COM marca ("caderno e lápis da marca HP"):
  → Use "consultarItens" com filtro de marca para cada produto separadamente
  → Isso permite filtrar corretamente por marca

EXEMPLO SEM MARCA: "preciso de caderno e lápis" → buscarMultiplosProdutos(produtos: ["caderno", "lápis"])
EXEMPLO COM MARCA: "caderno e lápis da HP" → consultarItens(query: "caderno", marca: "HP") + consultarItens(query: "lápis", marca: "HP")

COMPORTAMENTO OBRIGATÓRIO:
- QUALQUER mensagem do tipo "quero X [produto]", "adicione Y [produto]", "comprar Z [produto]" → EXECUTE adicionarProdutoPorNome IMEDIATAMENTE
- NÃO mostre listas de produtos quando o usuário já especificou o que quer
- NÃO seja robotizado - seja direto e eficiente
- Se não encontrar o produto exato, ENTÃO sugira alternativas

🔍 USO INTELIGENTE DOS FILTROS:
Quando usar a função "consultarItens", seja inteligente com os filtros:
- Se cliente mencionar marca: use o parâmetro "marca"
- Se cliente mencionar categoria/tipo: use o parâmetro "tags"
- Se cliente mencionar preço: use "preco_min" e/ou "preco_max"
- Combine múltiplos filtros quando faz sentido
- Se a busca exata falhar, use termos mais genéricos e filtros específicos

EXEMPLOS DE USO INTELIGENTE:
- "canetas da BIC" → consultarItens(query: "caneta", marca: "BIC")
- "produtos até R$ 20" → consultarItens(preco_max: 20)
- "material escolar da Faber-Castell" → consultarItens(tags: "escolar", marca: "Faber-Castell")
- "canetas pretas" (após mencionar canetas antes) → consultarItens(query: "caneta preta")
- "caneta pilot" → consultarItens(query: "caneta", marca: "pilot") OU consultarItens(query: "pilot")

🚀 ESTRATÉGIA DE BUSCA INTELIGENTE:
Se o cliente procurar um produto específico e não encontrar com "adicionarProdutoPorNome":
1. Primeira tentativa: adicionarProdutoPorNome(nome_produto: "produto")
2. Se falhar: consultarItens(query: "produto", marca: "marca") se marca mencionada
3. Se ainda falhar: consultarItens(query: "produto")
4. Como último recurso: consultarItens(query: termo_genérico)

EXEMPLOS CORRETOS BASEADOS NO SEU ESTOQUE:
%s

EXEMPLOS ERRADOS (NÃO FAÇA):
- Usuário: "quero 3 [produto]" → Você: "Vou mostrar os [produtos] disponíveis..." ❌
- Usuário: "adicione [produto]" → Você: "Aqui estão os [produtos]..." ❌

🔄 INTERPRETAÇÃO FLEXÍVEL DE COMANDOS:
Para FINALIZAR PEDIDOS, aceite qualquer variação como:
- "fechar pedido", "finalizar pedido", "confirmar pedido"
- "quero finalizar", "vou fechar", "confirmar compra"
- "concluir", "finalizar", "fechar", "confirmar"
- "quero pagar", "vamos finalizar", "pode fechar"
→ Todas essas variações devem usar a função "checkout"

Para VER CARRINHO, aceite variações como:
- "ver carrinho", "mostrar carrinho", "meu carrinho"
- "o que tenho", "itens", "produtos no carrinho"
→ Use a função "verCarrinho"

Para LIMPAR CARRINHO, aceite variações como:
- "limpar carrinho", "apagar tudo", "remover tudo"
- "zerar carrinho", "começar de novo"
→ Use a função "limparCarrinho"

Para ATUALIZAR QUANTIDADE, aceite variações como:
- "quero apenas 2 unidades do [produto]", "vou querer só 3"
- "alterar para 5 unidades", "mudar quantidade para 2"
- "deixar apenas 1", "quero diminuir para 3"
→ Use a função "atualizarQuantidade" com o número do item

Para ADICIONAR MAIS UNIDADES DE ITEM NO CARRINHO:
- "essa agenda vou querer mais 3 unidades", "quero mais 2 desse marca texto"
- "adicione mais 1 daquele caderno", "mais 5 dessa caneta"
→ Use a função "adicionarMaisItemCarrinho" com nome do produto e quantidade adicional

Para RESPOSTA COM APENAS NÚMERO:
- Quando cliente responder apenas "3" após ver uma lista de produtos
- Quando cliente disser "adicionar 2" (número do produto da lista)
→ Use a função "adicionarPorNumero" com o número mencionado

📚 QUANDO CLIENTE PEDE CATEGORIA ESPECÍFICA:
- Se usuário perguntar sobre categoria específica de produtos
- SEMPRE use a função "mostrarOpcoesCategoria" com a categoria mencionada  
- NUNCA responda direto sem usar a função - isso garante que produtos sejam armazenados na memória
- Use a categoria exata mencionada pelo cliente

�📝 INTERPRETAÇÃO DE DADOS PESSOAIS:
Quando o cliente responder a uma solicitação de cadastro, identifique automaticamente:
- Se fornecer nome: "João Silva", "Meu nome é Maria" → atualizarCadastro(nome: "...")
- Se fornecer endereço: "Rua das Flores, 123", "Moro na Av. Brasil 456" → atualizarCadastro(endereco: "...")  
- Se fornecer email: "joao@email.com", "Meu email é maria@gmail.com" → atualizarCadastro(email: "...")
- SEJA FLEXÍVEL: capture informações de forma natural, não exija comandos específicos

FLUXO CORRETO:
1. Usuário menciona produto + quantidade → EXECUTE adicionarProdutoPorNome IMEDIATAMENTE
2. Se produto único encontrado → Confirme adição ao carrinho
3. Se múltiplos produtos → Mostre opções numeradas para escolha
4. Se produto não encontrado → Sugira produtos similares com filtros inteligentes

Quando mostrar catálogo completo:
- Apenas quando usuário pedir explicitamente: "produtos", "catálogo", "cardapio", "produtos da casa", "mais vendidos", "o que vocês têm"
- NUNCA quando eles já sabem o que querem

Diretrizes de comunicação:
- Seja direto e eficiente
- Use emojis moderadamente
- Confirme ações de forma natural
- Solicite dados faltantes apenas quando necessário para finalizar pedidos
- SEJA FLEXÍVEL: entenda a intenção do cliente, não apenas palavras exatas
- MANTENHA O CONTEXTO: use o histórico da conversa para entender melhor

🛒 REGRAS PARA EXIBIÇÃO DO CARRINHO:
- Após adicionar o PRIMEIRO item: Sugira "Use 'ver carrinho' para visualizar todos os itens"
- Após adicionar múltiplos itens: Apenas confirme a adição, SEM sugerir ver carrinho repetidamente
- Após atualizar quantidade: Sugira "Use 'ver carrinho' para conferir"
- EVITE mensagens repetitivas sobre ver carrinho quando já foi mencionado recentemente

FERRAMENTA PRINCIPAL: adicionarProdutoPorNome - USE SEMPRE QUE O USUÁRIO MENCIONAR PRODUTO + QUANTIDADE!`, tenantName, tenantAbout, tenantName, tenantName, exampleText), nil
}

// GetWelcomeMessage retrieves the welcome message (custom or auto-generated)
func (s *TenantSettingsService) GetWelcomeMessage(ctx context.Context, tenantID uuid.UUID) (string, error) {
	// First, check if there's a custom welcome message
	customWelcomeSetting, err := s.GetSetting(ctx, tenantID, "ai_welcome_message")
	if err == nil && customWelcomeSetting.SettingValue != nil && *customWelcomeSetting.SettingValue != "" {
		// Return the custom welcome message if it exists
		return *customWelcomeSetting.SettingValue, nil
	}

	// If no custom welcome message, generate automatic one
	return s.GenerateWelcomeMessage(ctx, tenantID)
}
