package services

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"iafarma/internal/zapplus"
	"iafarma/pkg/models"

	"gorm.io/gorm"
)

// ChannelMonitorService handles monitoring of WhatsApp channels
type ChannelMonitorService struct {
	db             *gorm.DB
	emailService   *EmailService
	checkInterval  time.Duration
	mutex          sync.RWMutex
	isRunning      bool
	stopChan       chan struct{}
	failedChannels map[string]*FailedChannel
	zapClient      *zapplus.Client
}

// FailedChannel represents a channel that failed monitoring
type FailedChannel struct {
	ChannelID   string
	TenantID    string
	SessionName string
	LastError   string
	FirstFailed time.Time
}

// NewChannelMonitorService creates a new channel monitor service
func NewChannelMonitorService(db *gorm.DB, emailService *EmailService) *ChannelMonitorService {
	return &ChannelMonitorService{
		db:             db,
		emailService:   emailService,
		checkInterval:  1 * time.Minute,
		failedChannels: make(map[string]*FailedChannel),
		stopChan:       make(chan struct{}),
		zapClient:      zapplus.GetClient(),
	}
}

// Start begins the monitoring process
func (cms *ChannelMonitorService) Start(ctx context.Context) {
	cms.mutex.Lock()
	if cms.isRunning {
		cms.mutex.Unlock()
		return
	}
	cms.isRunning = true
	cms.mutex.Unlock()

	log.Println("📡 Iniciando monitoramento de canais WhatsApp...")

	go func() {
		ticker := time.NewTicker(cms.checkInterval)
		defer ticker.Stop()

		// Executa primeira verificação imediatamente
		cms.checkAllChannels(ctx)

		for {
			select {
			case <-ticker.C:
				cms.checkAllChannels(ctx)
			case <-cms.stopChan:
				log.Println("📡 Parando monitoramento de canais...")
				return
			case <-ctx.Done():
				log.Println("📡 Contexto cancelado, parando monitoramento...")
				return
			}
		}
	}()
}

// Stop stops the monitoring process
func (cms *ChannelMonitorService) Stop() {
	cms.mutex.Lock()
	defer cms.mutex.Unlock()

	if !cms.isRunning {
		return
	}

	cms.isRunning = false
	close(cms.stopChan)
}

// checkAllChannels verifies all connected channels
func (cms *ChannelMonitorService) checkAllChannels(ctx context.Context) {
	// Buscar apenas canais conectados
	var channels []models.Channel
	err := cms.db.Where("status = ? AND is_active = ?", "connected", true).Find(&channels).Error
	if err != nil {
		log.Printf("❌ Erro ao buscar canais conectados: %v", err)
		return
	}

	if len(channels) == 0 {
		// log.Println("ℹ️ Nenhum canal conectado encontrado para monitorar")
		return
	}

	// log.Printf("🔍 Verificando %d canais conectados...", len(channels))

	var wg sync.WaitGroup
	newFailedChannels := make(map[string]*FailedChannel)
	var mutex sync.Mutex

	for _, channel := range channels {
		wg.Add(1)
		go func(ch models.Channel) {
			defer wg.Done()

			if err := cms.checkChannelStatus(ctx, &ch); err != nil {
				mutex.Lock()
				key := fmt.Sprintf("%s-%s", ch.TenantID.String(), ch.ID.String())

				// Se é uma nova falha, adiciona na lista
				if _, exists := cms.failedChannels[key]; !exists {
					newFailedChannels[key] = &FailedChannel{
						ChannelID:   ch.ID.String(),
						TenantID:    ch.TenantID.String(),
						SessionName: ch.Session,
						LastError:   err.Error(),
						FirstFailed: time.Now(),
					}
				}
				mutex.Unlock()

				// Atualizar status do canal no banco
				cms.updateChannelStatus(&ch, "disconnected")
				log.Printf("❌ Canal %s (sessão: %s) falhou: %v", ch.Name, ch.Session, err)
			} // else {
			//log.Printf("✅ Canal %s (sessão: %s) está funcionando", ch.Name, ch.Session)
			// }
		}(channel)
	}

	wg.Wait()

	// Se há novas falhas, enviar email
	if len(newFailedChannels) > 0 {
		cms.mutex.Lock()
		// Adiciona as novas falhas à lista global
		for key, failed := range newFailedChannels {
			cms.failedChannels[key] = failed
		}
		cms.mutex.Unlock()

		// Enviar email de notificação
		cms.sendFailureNotification(newFailedChannels)
	}
}

// checkChannelStatus verifies a single channel status with ZapPlus
func (cms *ChannelMonitorService) checkChannelStatus(ctx context.Context, channel *models.Channel) error {
	sessionResp, err := cms.zapClient.GetSessionStatus(channel.Session)
	if err != nil {
		return fmt.Errorf("erro ao verificar status da sessão: %w", err)
	}

	// Verificar se o status é WORKING
	if sessionResp.Status != "WORKING" {
		return fmt.Errorf("status da sessão não é WORKING: %s", sessionResp.Status)
	}

	return nil
}

// updateChannelStatus updates the channel status in database
func (cms *ChannelMonitorService) updateChannelStatus(channel *models.Channel, status string) {
	err := cms.db.Model(channel).Update("status", status).Error
	if err != nil {
		log.Printf("❌ Erro ao atualizar status do canal %s: %v", channel.Name, err)
	}
}

// sendFailureNotification sends email notification about failed channels to super admins
func (cms *ChannelMonitorService) sendFailureNotification(failedChannels map[string]*FailedChannel) {
	if cms.emailService == nil {
		log.Println("⚠️ EmailService não configurado, não é possível enviar notificação")
		return
	}

	// Buscar todos os super admins
	var superAdmins []models.User
	err := cms.db.Where("role = ?", "system_admin").Find(&superAdmins).Error
	if err != nil {
		log.Printf("❌ Erro ao buscar super admins: %v", err)
		return
	}

	if len(superAdmins) == 0 {
		log.Println("⚠️ Nenhum super admin encontrado para envio de notificação")
		return
	}

	// Enviar email para cada super admin
	for _, admin := range superAdmins {
		go cms.sendSuperAdminFailureEmail(admin.Email, failedChannels)
	}
}

// sendSuperAdminFailureEmail sends failure notification to a super admin
func (cms *ChannelMonitorService) sendSuperAdminFailureEmail(adminEmail string, failedChannels map[string]*FailedChannel) {
	// Construir conteúdo do email
	subject := "🚨 Alerta: Canais WhatsApp Desconectados - IAFarma"

	var body bytes.Buffer
	body.WriteString(fmt.Sprintf(`
<html>
<body>
<h2>🚨 Alerta de Canais Desconectados</h2>
<p>Olá Super Admin,</p>
<p>Detectamos que %d canal(is) WhatsApp perderam a conexão no sistema:</p>
<ul>
`, len(failedChannels)))

	for _, failure := range failedChannels {
		body.WriteString(fmt.Sprintf(`
<li>
	<strong>Canal ID:</strong> %s<br>
	<strong>Sessão:</strong> %s<br>
	<strong>Erro:</strong> %s<br>
	<strong>Detectado em:</strong> %s
</li>
`, failure.ChannelID, failure.SessionName, failure.LastError, failure.FirstFailed.Format("02/01/2006 15:04:05")))
	}

	body.WriteString(`
</ul>
<p>Por favor, verifique a conexão dos canais no painel administrativo do sistema.</p>
<p><strong>Ação necessária:</strong> Verificar configuração dos canais WhatsApp no ZapPlus e reconectar se necessário.</p>
<p>Atenciosamente,<br>Sistema de Monitoramento IAFarma</p>
</body>
</html>
`)

	// Enviar email
	if err := cms.emailService.SendEmail([]string{adminEmail}, subject, body.String()); err != nil {
		log.Printf("❌ Erro ao enviar email de notificação para super admin %s: %v", adminEmail, err)
	} else {
		log.Printf("📧 Email de notificação enviado para super admin %s (%d canais falhados)", adminEmail, len(failedChannels))
	}
}

// GetMonitoringStatus returns current monitoring status
func (cms *ChannelMonitorService) GetMonitoringStatus() map[string]interface{} {
	cms.mutex.RLock()
	defer cms.mutex.RUnlock()

	return map[string]interface{}{
		"is_running":      cms.isRunning,
		"check_interval":  cms.checkInterval.String(),
		"failed_channels": len(cms.failedChannels),
		"last_check":      time.Now().Format("2006-01-02 15:04:05"),
	}
}

// ChannelReconnectionService handles monitoring channel reconnections
type ChannelReconnectionService struct {
	db                   *gorm.DB
	emailService         *EmailService
	channelMonitor       *ChannelMonitorService
	checkInterval        time.Duration
	mutex                sync.RWMutex
	isRunning            bool
	stopChan             chan struct{}
	disconnectedChannels map[string]*DisconnectedChannel
}

// DisconnectedChannel represents a channel that was disconnected
type DisconnectedChannel struct {
	ChannelID        string
	TenantID         string
	SessionName      string
	DisconnectedAt   time.Time
	LastNotification time.Time
}

// NewChannelReconnectionService creates a new channel reconnection monitor service
func NewChannelReconnectionService(db *gorm.DB, emailService *EmailService, channelMonitor *ChannelMonitorService) *ChannelReconnectionService {
	return &ChannelReconnectionService{
		db:                   db,
		emailService:         emailService,
		channelMonitor:       channelMonitor,
		checkInterval:        2 * time.Minute, // Verifica a cada 2 minutos
		disconnectedChannels: make(map[string]*DisconnectedChannel),
		stopChan:             make(chan struct{}),
	}
}

// Start begins the reconnection monitoring process
func (crs *ChannelReconnectionService) Start(ctx context.Context) {
	crs.mutex.Lock()
	if crs.isRunning {
		crs.mutex.Unlock()
		return
	}
	crs.isRunning = true
	crs.mutex.Unlock()

	log.Println("🔄 Iniciando monitoramento de reconexões de canais...")

	go func() {
		ticker := time.NewTicker(crs.checkInterval)
		defer ticker.Stop()

		// Primeira verificação para mapear canais já desconectados
		crs.mapDisconnectedChannels(ctx)

		for {
			select {
			case <-ticker.C:
				crs.checkForReconnections(ctx)
			case <-crs.stopChan:
				log.Println("🔄 Parando monitoramento de reconexões...")
				return
			case <-ctx.Done():
				log.Println("🔄 Contexto cancelado, parando monitoramento de reconexões...")
				return
			}
		}
	}()
}

// Stop stops the reconnection monitoring process
func (crs *ChannelReconnectionService) Stop() {
	crs.mutex.Lock()
	defer crs.mutex.Unlock()

	if !crs.isRunning {
		return
	}

	crs.isRunning = false
	close(crs.stopChan)
}

// mapDisconnectedChannels initially maps all currently disconnected channels
func (crs *ChannelReconnectionService) mapDisconnectedChannels(ctx context.Context) {
	var channels []models.Channel
	err := crs.db.Where("status != ? AND is_active = ?", "connected", true).Find(&channels).Error
	if err != nil {
		log.Printf("❌ Erro ao buscar canais desconectados: %v", err)
		return
	}

	crs.mutex.Lock()
	defer crs.mutex.Unlock()

	for _, channel := range channels {
		if _, exists := crs.disconnectedChannels[channel.ID.String()]; !exists {
			crs.disconnectedChannels[channel.ID.String()] = &DisconnectedChannel{
				ChannelID:      channel.ID.String(),
				TenantID:       channel.TenantID.String(),
				SessionName:    channel.Session,
				DisconnectedAt: time.Now(),
			}
		}
	}

	// log.Printf("🔄 Mapeados %d canais desconectados para monitoramento", len(crs.disconnectedChannels))
}

// checkForReconnections checks if any previously disconnected channels have reconnected
func (crs *ChannelReconnectionService) checkForReconnections(ctx context.Context) {
	crs.mutex.RLock()
	disconnectedCopy := make(map[string]*DisconnectedChannel)
	for k, v := range crs.disconnectedChannels {
		disconnectedCopy[k] = v
	}
	crs.mutex.RUnlock()

	if len(disconnectedCopy) == 0 {
		return
	}

	// log.Printf("🔄 Verificando reconexões para %d canais desconectados...", len(disconnectedCopy))

	var reconnected []string

	for channelID, disconnectedChannel := range disconnectedCopy {
		var channel models.Channel
		err := crs.db.Where("id = ?", channelID).First(&channel).Error
		if err != nil {
			log.Printf("❌ Erro ao buscar canal %s: %v", channelID, err)
			continue
		}

		// Verificar status atual do canal usando o mesmo método do monitor principal
		err = crs.channelMonitor.checkChannelStatus(ctx, &channel)
		if err == nil {
			// Canal reconectou!
			log.Printf("✅ Canal reconectado: %s (%s)", channel.Name, channel.Session)

			// Atualizar status no banco
			crs.channelMonitor.updateChannelStatus(&channel, "connected")

			// Enviar notificação de reconexão
			crs.sendReconnectionNotification(&channel, disconnectedChannel)

			reconnected = append(reconnected, channelID)
		} else {
			// Ainda desconectado, atualizar na lista se necessário
			crs.updateDisconnectedChannelStatus(&channel)
		}
	}

	// Remover canais reconectados da lista
	if len(reconnected) > 0 {
		crs.mutex.Lock()
		for _, channelID := range reconnected {
			delete(crs.disconnectedChannels, channelID)
		}
		crs.mutex.Unlock()

		log.Printf("🔄 %d canais foram reconectados e removidos da lista de monitoramento", len(reconnected))
	}

	// Adicionar novos canais desconectados
	crs.addNewlyDisconnectedChannels(ctx)
}

// updateDisconnectedChannelStatus updates the status of a disconnected channel
func (crs *ChannelReconnectionService) updateDisconnectedChannelStatus(channel *models.Channel) {
	if channel.Status == "connected" {
		return // Canal já está conectado
	}

	// Atualizar status no banco se necessário
	if channel.Status != "disconnected" && channel.Status != "failed" {
		crs.channelMonitor.updateChannelStatus(channel, "disconnected")
	}
}

// addNewlyDisconnectedChannels adds newly disconnected channels to monitoring
func (crs *ChannelReconnectionService) addNewlyDisconnectedChannels(ctx context.Context) {
	var channels []models.Channel
	err := crs.db.Where("status != ? AND is_active = ?", "connected", true).Find(&channels).Error
	if err != nil {
		log.Printf("❌ Erro ao buscar canais desconectados: %v", err)
		return
	}

	crs.mutex.Lock()
	defer crs.mutex.Unlock()

	newCount := 0
	for _, channel := range channels {
		if _, exists := crs.disconnectedChannels[channel.ID.String()]; !exists {
			crs.disconnectedChannels[channel.ID.String()] = &DisconnectedChannel{
				ChannelID:      channel.ID.String(),
				TenantID:       channel.TenantID.String(),
				SessionName:    channel.Session,
				DisconnectedAt: time.Now(),
			}
			newCount++
		}
	}

	if newCount > 0 {
		log.Printf("🔄 Adicionados %d novos canais desconectados ao monitoramento", newCount)
	}
}

// sendReconnectionNotification sends email notification about channel reconnection
func (crs *ChannelReconnectionService) sendReconnectionNotification(channel *models.Channel, disconnectedChannel *DisconnectedChannel) {
	if crs.emailService == nil {
		log.Println("⚠️ EmailService não configurado, não é possível enviar notificação de reconexão")
		return
	}

	// Buscar informações do tenant
	var tenant models.Tenant
	err := crs.db.Where("id = ?", channel.TenantID).First(&tenant).Error
	if err != nil {
		log.Printf("❌ Erro ao buscar tenant %s: %v", channel.TenantID, err)
		return
	}

	// Buscar admin do tenant
	var adminUser models.User
	err = crs.db.Where("tenant_id = ? AND role = ?", channel.TenantID, "tenant_admin").First(&adminUser).Error
	if err != nil {
		log.Printf("❌ Erro ao buscar admin do tenant %s: %v", tenant.Name, err)
		return
	}

	// Calcular tempo de desconexão
	downtime := time.Since(disconnectedChannel.DisconnectedAt)

	subject := fmt.Sprintf("✅ Canal WhatsApp Reconectado - %s", channel.Name)

	body := fmt.Sprintf(`
<html>
<body style="font-family: Arial, sans-serif; line-height: 1.6; color: #333;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <div style="background: linear-gradient(135deg, #22c55e, #16a34a); color: white; padding: 20px; border-radius: 8px; text-align: center; margin-bottom: 20px;">
            <h1 style="margin: 0; font-size: 24px;">✅ Canal Reconectado</h1>
            <p style="margin: 10px 0 0 0; opacity: 0.9;">Serviço WhatsApp foi reestabelecido</p>
        </div>
        
        <div style="background: #f8fafc; padding: 20px; border-radius: 8px; margin-bottom: 20px;">
            <h2 style="color: #16a34a; margin-top: 0;">Detalhes da Reconexão</h2>
            <table style="width: 100%; border-collapse: collapse;">
                <tr>
                    <td style="padding: 8px 0; font-weight: bold; width: 120px;">Canal:</td>
                    <td style="padding: 8px 0;">%s</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0; font-weight: bold;">Empresa:</td>
                    <td style="padding: 8px 0;">%s</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0; font-weight: bold;">Sessão:</td>
                    <td style="padding: 8px 0;">%s</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0; font-weight: bold;">Reconectado em:</td>
                    <td style="padding: 8px 0;">%s</td>
                </tr>
                <tr>
                    <td style="padding: 8px 0; font-weight: bold;">Tempo offline:</td>
                    <td style="padding: 8px 0;">%s</td>
                </tr>
            </table>
        </div>
        
        <div style="background: #eff6ff; border-left: 4px solid #3b82f6; padding: 15px; margin-bottom: 20px;">
            <p style="margin: 0; color: #1e40af;">
                <strong>ℹ️ Status atual:</strong> O canal está novamente operacional e pronto para receber mensagens.
            </p>
        </div>
        
        <div style="text-align: center; margin-top: 30px;">
            <p style="color: #6b7280; font-size: 14px;">
                Esta é uma notificação automática do sistema IAFarma.<br>
                Gerada em %s
            </p>
        </div>
    </div>
</body>
</html>`,
		channel.Name,
		tenant.Name,
		channel.Session,
		time.Now().Format("02/01/2006 às 15:04:05"),
		formatDuration(downtime),
		time.Now().Format("02/01/2006 às 15:04:05"),
	)

	err = crs.emailService.SendEmail([]string{adminUser.Email}, subject, body)
	if err != nil {
		log.Printf("❌ Erro ao enviar email de reconexão para %s: %v", adminUser.Email, err)
	} else {
		log.Printf("📧 Email de reconexão enviado para %s (%s)", adminUser.Email, tenant.Name)
	}
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return "menos de 1 minuto"
}

// GetReconnectionMonitoringStatus returns the current status of reconnection monitoring
func (crs *ChannelReconnectionService) GetReconnectionMonitoringStatus() map[string]interface{} {
	crs.mutex.RLock()
	defer crs.mutex.RUnlock()

	return map[string]interface{}{
		"is_running":            crs.isRunning,
		"check_interval":        crs.checkInterval.String(),
		"disconnected_channels": len(crs.disconnectedChannels),
		"last_check":            time.Now().Format("2006-01-02 15:04:05"),
	}
}
