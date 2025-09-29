package zapplus

import (
	"fmt"
	"iafarma/pkg/models"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// NotificationService gerencia envio de notificações via ZapPlus
type NotificationService struct {
	db     *gorm.DB
	client *Client
}

// NewNotificationService cria uma nova instância do serviço de notificação
func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{
		db:     db,
		client: GetClient(),
	}
}

// SendShippingNotification envia notificação de pedido enviado
func (s *NotificationService) SendShippingNotification(tenantID uuid.UUID, order *models.Order) error {
	log.Printf("📨 Sending shipping notification for order %s", order.OrderNumber)

	// Buscar dados completos do pedido
	var fullOrder models.Order
	err := s.db.Preload("Customer").Preload("Conversation").
		Where("id = ? AND tenant_id = ?", order.ID, tenantID).
		First(&fullOrder).Error
	if err != nil {
		return fmt.Errorf("failed to load order data: %w", err)
	}

	// Verificar se há telefone do cliente
	if fullOrder.Customer == nil || fullOrder.Customer.Phone == "" {
		return fmt.Errorf("no customer phone found for order %s", order.OrderNumber)
	}

	// Preparar mensagem
	message := fmt.Sprintf("🚚 *Seu pedido foi enviado!*\n\n📦 Pedido: #%s\n📅 Data: %s\n\nEm breve você receberá as informações de rastreamento.\n\nObrigado pela preferência! 😊",
		order.OrderNumber,
		order.UpdatedAt.Format("02/01/2006 15:04"))

	// Enviar mensagem
	return s.SendDirectMessage(tenantID, fullOrder.Customer.Phone, message)
}

// SendDirectMessage envia mensagem direta para um cliente
func (s *NotificationService) SendDirectMessage(tenantID uuid.UUID, customerPhone, message string) error {
	session, err := s.findActiveSession(tenantID, customerPhone)
	if err != nil {
		return fmt.Errorf("failed to find active session: %w", err)
	}

	chatID := FormatPhoneToWhatsApp(customerPhone)
	err = s.client.SendTextMessage(session, chatID, message)
	if err != nil {
		log.Printf("❌ Failed to send WhatsApp message to %s: %v", customerPhone, err)
		return err
	}

	// Buscar conversa ativa para registrar a mensagem
	var conversation models.Conversation
	convErr := s.db.Where("conversations.tenant_id = ?", tenantID).
		Joins("JOIN customers ON conversations.customer_id = customers.id").
		Where("customers.phone = ?", customerPhone).
		Where("conversations.status IN (?)", []string{"active", "open"}).
		Preload("Customer").
		Order("conversations.created_at DESC").
		First(&conversation).Error

	if convErr == nil && conversation.Customer != nil {
		// Registrar mensagem na conversa
		outgoingMessage := models.Message{
			ConversationID: conversation.ID,
			CustomerID:     conversation.Customer.ID,
			Content:        message,
			Type:           "text",
			Direction:      "out",
			Status:         "sent",
			Source:         "whatsapp",
			ExternalID:     fmt.Sprintf("otp_%d", time.Now().Unix()),
		}
		outgoingMessage.TenantID = tenantID

		if msgErr := s.db.Create(&outgoingMessage).Error; msgErr != nil {
			log.Printf("⚠️ Failed to save outgoing message to database: %v", msgErr)
		} else {
			log.Printf("✅ Message saved to conversation %s for customer %s", conversation.ID, customerPhone)
		}

		// Atualizar última atividade da conversa
		updateErr := s.db.Model(&conversation).Updates(map[string]interface{}{
			"updated_at": time.Now(),
		}).Error
		if updateErr != nil {
			log.Printf("⚠️ Failed to update conversation timestamp: %v", updateErr)
		}
	} else {
		log.Printf("⚠️ Could not find active conversation for customer %s to save message: %v", customerPhone, convErr)
	}

	log.Printf("✅ WhatsApp message sent successfully to %s", customerPhone)
	return nil
}

// SendGroupAlert envia alerta para grupo configurado
func (s *NotificationService) SendGroupAlert(tenantID uuid.UUID, groupID, message, session string) error {
	if !s.client.IsValidSession(session) {
		return fmt.Errorf("session %s is not valid or connected", session)
	}

	return s.client.SendGroupMessage(session, groupID, message)
}

// SendOrderAlert envia alerta de novo pedido
func (s *NotificationService) SendOrderAlert(tenantID uuid.UUID, order *models.Order, customerPhone string) error {
	// Buscar alertas configurados
	var alerts []models.Alert
	err := s.db.Where("tenant_id = ? AND is_active = ? AND trigger_on = ?",
		tenantID, true, "order_created").
		Preload("Channel").
		Find(&alerts).Error
	if err != nil {
		return fmt.Errorf("failed to find alerts: %w", err)
	}

	if len(alerts) == 0 {
		log.Printf("No alerts configured for tenant %s", tenantID)
		return nil
	}

	// Buscar dados do cliente
	var customer models.Customer
	err = s.db.Where("id = ? AND tenant_id = ?", order.CustomerID, tenantID).First(&customer).Error
	if err != nil {
		return fmt.Errorf("failed to find customer: %w", err)
	}

	// Preparar mensagem do alerta
	message := s.formatOrderAlert(order, &customer, customerPhone)

	// Enviar para todos os grupos configurados
	var lastError error
	sentCount := 0

	for _, alert := range alerts {
		if alert.GroupID != "" && alert.Channel != nil && alert.Channel.Session != "" {
			err := s.SendGroupAlert(tenantID, alert.GroupID, message, alert.Channel.Session)
			if err != nil {
				log.Printf("❌ Failed to send alert to group %s: %v", alert.GroupName, err)
				lastError = err
				continue
			}
			log.Printf("✅ Alert sent to group %s", alert.GroupName)
			sentCount++
		}
	}

	if sentCount == 0 && lastError != nil {
		return lastError
	}

	return nil
}

// SendHumanSupportAlert envia alerta quando cliente solicita atendimento humano
func (s *NotificationService) SendHumanSupportAlert(tenantID uuid.UUID, customerID uuid.UUID, customerPhone, reason string) error {
	// Buscar alertas de suporte humano
	var alerts []models.Alert
	err := s.db.Where("tenant_id = ? AND is_active = ? AND trigger_on = ?",
		tenantID, true, "human_support_request").
		Preload("Channel").
		Find(&alerts).Error
	if err != nil {
		// Fallback: buscar qualquer alerta ativo
		err = s.db.Where("tenant_id = ? AND is_active = ?", tenantID, true).
			Preload("Channel").
			Find(&alerts).Error
		if err != nil {
			return fmt.Errorf("failed to find alerts: %w", err)
		}
	}

	if len(alerts) == 0 {
		log.Printf("No human support alerts configured for tenant %s", tenantID)
		return nil
	}

	// Buscar dados do cliente
	var customer models.Customer
	err = s.db.Where("id = ? AND tenant_id = ?", customerID, tenantID).First(&customer).Error
	if err != nil {
		return fmt.Errorf("failed to find customer: %w", err)
	}

	// Preparar mensagem
	message := s.formatHumanSupportAlert(&customer, customerPhone, reason)

	// Enviar para todos os grupos
	for _, alert := range alerts {
		if alert.GroupID != "" && alert.Channel != nil && alert.Channel.Session != "" {
			err := s.SendGroupAlert(tenantID, alert.GroupID, message, alert.Channel.Session)
			if err != nil {
				log.Printf("❌ Failed to send human support alert to group %s: %v", alert.GroupName, err)
				continue
			}
			log.Printf("✅ Human support alert sent to group %s", alert.GroupName)
		}
	}

	return nil
}

// findActiveSession encontra uma sessão ativa para o tenant/cliente
func (s *NotificationService) findActiveSession(tenantID uuid.UUID, customerPhone string) (string, error) {
	// Primeiro, tentar buscar conversa específica do cliente
	var conversation models.Conversation
	err := s.db.Where("conversations.tenant_id = ?", tenantID).
		Joins("JOIN customers ON conversations.customer_id = customers.id").
		Where("customers.phone = ?", customerPhone).
		Preload("Channel").
		Order("conversations.created_at DESC").
		First(&conversation).Error

	if err == nil && conversation.Channel != nil && conversation.Channel.Session != "" {
		log.Printf("📱 Using customer-specific session: %s", conversation.Channel.Session)
		return conversation.Channel.Session, nil
	}

	// Buscar um canal ativo do tenant para criar conversa
	log.Printf("⚠️ No specific conversation for customer %s, searching for active channel...", customerPhone)

	var channel models.Channel
	err = s.db.Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Order("created_at DESC").
		First(&channel).Error
	if err != nil {
		return "", fmt.Errorf("no active channel found for tenant %s", tenantID)
	}

	if channel.Session == "" {
		return "", fmt.Errorf("no active session found for tenant %s", tenantID)
	}

	// Tentar criar ou encontrar cliente
	var customer models.Customer
	customerErr := s.db.Where("tenant_id = ? AND phone = ?", tenantID, customerPhone).
		First(&customer).Error

	var shouldCreateConversation = false

	if customerErr == gorm.ErrRecordNotFound {
		// Criar novo cliente
		customer = models.Customer{
			Phone:    customerPhone,
			IsActive: true,
		}
		customer.TenantID = tenantID

		if err := s.db.Create(&customer).Error; err != nil {
			log.Printf("❌ Failed to create customer: %v", err)
			return channel.Session, nil // Retorna sessão padrão mesmo com erro
		} else {
			log.Printf("✅ Created new customer: %s", customerPhone)
			shouldCreateConversation = true
		}
	} else if customerErr != nil {
		log.Printf("❌ Error finding customer: %v", customerErr)
		return channel.Session, nil // Retorna sessão padrão em caso de erro
	} else {
		// Cliente existe, verificar se precisa criar conversa
		var existingConversation models.Conversation
		convErr := s.db.Where("tenant_id = ? AND customer_id = ?", tenantID, customer.ID).
			Where("status IN (?)", []string{"active", "open"}).
			First(&existingConversation).Error

		if convErr == gorm.ErrRecordNotFound {
			shouldCreateConversation = true
			log.Printf("📞 Customer %s exists but has no active conversation, creating new one", customerPhone)
		}
	}

	// Criar conversa se necessário
	if shouldCreateConversation && customer.ID != uuid.Nil {
		newConversation := models.Conversation{
			CustomerID: customer.ID,
			ChannelID:  channel.ID,
			Status:     "active",
			AIEnabled:  true,
		}
		newConversation.TenantID = tenantID

		if err := s.db.Create(&newConversation).Error; err != nil {
			log.Printf("⚠️ Failed to create conversation: %v", err)
		} else {
			log.Printf("✅ Created new conversation for customer %s (conversation_id: %s)", customerPhone, newConversation.ID)

			// Criar mensagem inicial para vincular à conversa
			initialMessage := models.Message{
				ConversationID: newConversation.ID,
				CustomerID:     customer.ID,
				Content:        fmt.Sprintf("Nova conversa iniciada para envio de código OTP para %s", customerPhone),
				Type:           "text",
				Direction:      "out",
				Status:         "sent",
				Source:         "system",
			}
			initialMessage.TenantID = tenantID

			if msgErr := s.db.Create(&initialMessage).Error; msgErr != nil {
				log.Printf("⚠️ Failed to create initial message: %v", msgErr)
			}
		}
	}

	log.Printf("📱 Using tenant default session: %s", channel.Session)
	return channel.Session, nil
}

// formatOrderAlert formata mensagem de alerta de pedido
func (s *NotificationService) formatOrderAlert(order *models.Order, customer *models.Customer, customerPhone string) string {
	return fmt.Sprintf(`🚨 *NOVO PEDIDO RECEBIDO* 🚨

📋 *Pedido:* %s
👤 *Cliente:* %s
📱 *Telefone:* %s
💰 *Valor Total:* R$ %s
📅 *Data:* %s

🔗 *Status:* %s

⚡ _Este pedido foi criado através do sistema de vendas automatizado._`,
		order.OrderNumber,
		customer.Name,
		customerPhone,
		order.TotalAmount,
		order.CreatedAt.Format("02/01/2006 15:04"),
		order.Status,
	)
}

// formatHumanSupportAlert formata mensagem de alerta de suporte humano
func (s *NotificationService) formatHumanSupportAlert(customer *models.Customer, customerPhone, reason string) string {
	return fmt.Sprintf(`🙋‍♂️ *SOLICITAÇÃO DE ATENDIMENTO HUMANO* 🙋‍♂️

👤 *Cliente:* %s
📱 *Telefone:* %s
💬 *Motivo:* %s
📅 *Data:* %s

🚨 *AÇÃO NECESSÁRIA:* Cliente precisa de atendimento personalizado

👋 Por favor, entre em contato com este cliente para fornecer suporte humano.

⚡ _Esta solicitação foi gerada automaticamente pelo sistema de IA._`,
		customer.Name,
		customerPhone,
		reason,
		time.Now().Format("02/01/2006 15:04"),
	)
}
