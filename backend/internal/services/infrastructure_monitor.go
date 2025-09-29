package services

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"gorm.io/gorm"
)

// InfrastructureMonitorService monitora a saúde da infraestrutura
type InfrastructureMonitorService struct {
	db               *gorm.DB
	embeddingService *EmbeddingService
	emailService     *EmailService
	isRunning        bool
	stopChannel      chan bool
	mutex            sync.Mutex
	emailSent        bool // Flag para evitar spam de emails
}

// InfrastructureStatus representa o status dos componentes da infraestrutura
type InfrastructureStatus struct {
	PostgreSQLHealthy bool
	PostgreSQLError   string
	QdrantHealthy     bool
	QdrantError       string
	Timestamp         time.Time
}

// NewInfrastructureMonitorService cria um novo serviço de monitoramento
func NewInfrastructureMonitorService(db *gorm.DB, embeddingService *EmbeddingService) (*InfrastructureMonitorService, error) {
	emailService, err := NewEmailService(db)
	if err != nil {
		log.Printf("Warning: Email service not available for infrastructure monitoring: %v", err)
	}

	return &InfrastructureMonitorService{
		db:               db,
		embeddingService: embeddingService,
		emailService:     emailService,
		isRunning:        false,
		stopChannel:      make(chan bool),
		emailSent:        false,
	}, nil
}

// Start inicia o monitoramento da infraestrutura
func (s *InfrastructureMonitorService) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.isRunning {
		return
	}

	s.isRunning = true
	log.Println("🔍 Starting infrastructure monitoring service...")

	// Iniciar o loop de monitoramento
	go s.monitoringLoop()
}

// Stop para o monitoramento da infraestrutura
func (s *InfrastructureMonitorService) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if !s.isRunning {
		return
	}

	log.Println("⏹️ Stopping infrastructure monitoring service...")
	s.stopChannel <- true
	s.isRunning = false
}

// monitoringLoop é o loop principal de monitoramento
func (s *InfrastructureMonitorService) monitoringLoop() {
	ticker := time.NewTicker(5 * time.Minute) // Monitorar a cada 5 minutos
	defer ticker.Stop()

	// Fazer primeira verificação imediatamente
	s.checkInfrastructure()

	for {
		select {
		case <-ticker.C:
			s.checkInfrastructure()
		case <-s.stopChannel:
			log.Println("🛑 Infrastructure monitoring loop stopped")
			return
		}
	}
}

// checkInfrastructure verifica o status de todos os componentes da infraestrutura
func (s *InfrastructureMonitorService) checkInfrastructure() {
	status := InfrastructureStatus{
		Timestamp: time.Now(),
	}

	// Verificar PostgreSQL
	status.PostgreSQLHealthy, status.PostgreSQLError = s.checkPostgreSQLHealth()

	// Verificar Qdrant (se disponível)
	if s.embeddingService != nil {
		status.QdrantHealthy, status.QdrantError = s.checkQdrantHealth()
	} else {
		status.QdrantHealthy = true // Se não estiver configurado, considerar como OK
		status.QdrantError = ""
	}

	// Log do status
	s.logInfrastructureStatus(status)

	// Verificar se precisa enviar email de alerta
	if s.shouldSendAlert(status) {
		s.sendInfrastructureAlert(status)
	} else if status.PostgreSQLHealthy && status.QdrantHealthy {
		// Se ambos estão saudáveis, resetar a flag de email
		s.mutex.Lock()
		s.emailSent = false
		s.mutex.Unlock()
	}
}

// checkPostgreSQLHealth verifica a saúde do PostgreSQL
func (s *InfrastructureMonitorService) checkPostgreSQLHealth() (bool, string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Tentar uma query simples para verificar a conexão
	var result int
	err := s.db.WithContext(ctx).Raw("SELECT 1").Scan(&result).Error
	if err != nil {
		return false, fmt.Sprintf("PostgreSQL connection failed: %v", err)
	}

	return true, ""
}

// checkQdrantHealth verifica a saúde do Qdrant
func (s *InfrastructureMonitorService) checkQdrantHealth() (bool, string) {
	if s.embeddingService == nil {
		return true, "" // Se não configurado, não é erro
	}

	err := s.embeddingService.CheckQdrantHealth()
	if err != nil {
		return false, fmt.Sprintf("Qdrant connection failed: %v", err)
	}

	return true, ""
}

// logInfrastructureStatus faz log do status da infraestrutura
func (s *InfrastructureMonitorService) logInfrastructureStatus(status InfrastructureStatus) {
	if status.PostgreSQLHealthy && status.QdrantHealthy {
		// Log apenas a cada 30 minutos quando tudo está OK para não poluir
		if time.Now().Minute()%30 == 0 {
			log.Printf("✅ Infrastructure health check: All systems operational")
		}
	} else {
		log.Printf("⚠️  Infrastructure health check problems detected:")
		if !status.PostgreSQLHealthy {
			log.Printf("   • PostgreSQL: %s", status.PostgreSQLError)
		}
		if !status.QdrantHealthy {
			log.Printf("   • Qdrant: %s", status.QdrantError)
		}
	}
}

// shouldSendAlert determina se um alerta por email deve ser enviado
func (s *InfrastructureMonitorService) shouldSendAlert(status InfrastructureStatus) bool {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Se já enviou email e há problemas, não enviar novamente
	if s.emailSent {
		return false
	}

	// Se não há serviço de email, não pode enviar
	if s.emailService == nil {
		return false
	}

	// Enviar apenas se houver problemas
	return !status.PostgreSQLHealthy || !status.QdrantHealthy
}

// sendInfrastructureAlert envia um alerta por email sobre problemas na infraestrutura
func (s *InfrastructureMonitorService) sendInfrastructureAlert(status InfrastructureStatus) {
	// Buscar email do super admin
	var superAdmin struct {
		Email string `json:"email"`
		Name  string `json:"name"`
	}

	err := s.db.Raw("SELECT email, name FROM users WHERE role = 'system_admin' LIMIT 1").Scan(&superAdmin).Error
	if err != nil || superAdmin.Email == "" {
		log.Printf("❌ Cannot send infrastructure alert: No super admin email found")
		return
	}

	// Preparar conteúdo do email
	subject := "🚨 Alerta de Infraestrutura - Sistema IAFarma"

	// Contar problemas
	var problemsCount int
	if !status.PostgreSQLHealthy {
		problemsCount++
	}
	if !status.QdrantHealthy {
		problemsCount++
	}

	// Gerar HTML do email
	body := s.renderInfrastructureAlertTemplate(status, problemsCount)

	// Tentar enviar o email
	err = s.emailService.SendEmail([]string{superAdmin.Email}, subject, body)
	if err != nil {
		log.Printf("❌ Failed to send infrastructure alert email: %v", err)
		return
	}

	// Marcar como enviado para evitar spam
	s.mutex.Lock()
	s.emailSent = true
	s.mutex.Unlock()

	log.Printf("📧 Infrastructure alert email sent to super admin: %s", superAdmin.Email)
	log.Printf("📊 Alert details: %d component(s) with problems", problemsCount)
}

// renderInfrastructureAlertTemplate renderiza o template HTML do alerta de infraestrutura
func (s *InfrastructureMonitorService) renderInfrastructureAlertTemplate(status InfrastructureStatus, problemsCount int) string {
	// Determinar cor do cabeçalho baseado na severidade
	headerColor := "#FF6B6B" // Vermelho para problemas críticos
	if problemsCount == 1 && !status.QdrantHealthy && status.PostgreSQLHealthy {
		headerColor = "#FF9800" // Laranja para apenas Qdrant (menos crítico)
	}

	// Gerar lista de componentes com status
	componentsHTML := ""

	// PostgreSQL
	postgresIcon := "✅"
	postgresStatus := "Operacional"
	postgresClass := "component-ok"
	if !status.PostgreSQLHealthy {
		postgresIcon = "❌"
		postgresStatus = "Com Problemas"
		postgresClass = "component-error"
	}

	componentsHTML += fmt.Sprintf(`
	<div class="component %s">
		<div class="component-header">
			<span class="component-icon">%s</span>
			<span class="component-name">PostgreSQL Database</span>
			<span class="component-status">%s</span>
		</div>
		%s
	</div>
	`, postgresClass, postgresIcon, postgresStatus, func() string {
		if !status.PostgreSQLHealthy {
			return fmt.Sprintf(`<div class="component-error-details">%s</div>`, status.PostgreSQLError)
		}
		return ""
	}())

	// Qdrant
	qdrantIcon := "✅"
	qdrantStatus := "Operacional"
	qdrantClass := "component-ok"
	if !status.QdrantHealthy {
		qdrantIcon = "❌"
		qdrantStatus = "Com Problemas"
		qdrantClass = "component-error"
	}

	componentsHTML += fmt.Sprintf(`
	<div class="component %s">
		<div class="component-header">
			<span class="component-icon">%s</span>
			<span class="component-name">Qdrant Vector Database</span>
			<span class="component-status">%s</span>
		</div>
		%s
	</div>
	`, qdrantClass, qdrantIcon, qdrantStatus, func() string {
		if !status.QdrantHealthy {
			return fmt.Sprintf(`<div class="component-error-details">%s</div>`, status.QdrantError)
		}
		return ""
	}())

	// Gerar ações recomendadas baseadas nos problemas
	actionsHTML := ""
	if !status.PostgreSQLHealthy {
		actionsHTML += `
		<li>🔍 <strong>PostgreSQL:</strong> Verificar se o serviço PostgreSQL está executando</li>
		<li>🌐 <strong>Conectividade:</strong> Confirmar se as credenciais e configurações de rede estão corretas</li>
		<li>💾 <strong>Recursos:</strong> Verificar espaço em disco e memória disponível</li>
		`
	}
	if !status.QdrantHealthy {
		actionsHTML += `
		<li>🚀 <strong>Qdrant:</strong> Verificar se o serviço Qdrant está executando na porta 6334</li>
		<li>🔐 <strong>Autenticação:</strong> Validar credenciais de acesso ao Qdrant</li>
		<li>📡 <strong>Rede:</strong> Verificar conectividade gRPC com o servidor Qdrant</li>
		`
	}

	// Template HTML completo
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; 
            line-height: 1.6; 
            color: #333; 
            margin: 0; 
            padding: 0; 
            background-color: #f8f9fa;
        }
        .container { 
            max-width: 600px; 
            margin: 0 auto; 
            background-color: #ffffff; 
            box-shadow: 0 4px 6px rgba(0,0,0,0.1); 
            border-radius: 8px; 
            overflow: hidden;
        }
        .header { 
            background: linear-gradient(135deg, %s, %s); 
            color: white; 
            padding: 30px 20px; 
            text-align: center; 
        }
        .header h1 { 
            margin: 0; 
            font-size: 24px; 
            font-weight: 600; 
        }
        .header p { 
            margin: 10px 0 0 0; 
            opacity: 0.9; 
            font-size: 16px;
        }
        .content { 
            padding: 30px; 
        }
        .alert-summary {
            background: linear-gradient(135deg, #fee2e2, #fef3c7);
            border: 1px solid #fbbf24;
            border-radius: 8px;
            padding: 20px;
            margin-bottom: 25px;
            text-align: center;
        }
        .alert-summary h2 {
            margin: 0 0 10px 0;
            color: #d97706;
            font-size: 20px;
        }
        .alert-count {
            font-size: 36px;
            font-weight: bold;
            color: #dc2626;
            margin: 5px 0;
        }
        .component {
            border: 1px solid #e5e7eb;
            border-radius: 8px;
            margin-bottom: 15px;
            overflow: hidden;
        }
        .component-ok {
            border-left: 4px solid #10b981;
            background-color: #f0fdf4;
        }
        .component-error {
            border-left: 4px solid #ef4444;
            background-color: #fef2f2;
        }
        .component-header {
            padding: 15px;
            display: flex;
            align-items: center;
            gap: 12px;
        }
        .component-icon {
            font-size: 18px;
        }
        .component-name {
            font-weight: 600;
            flex-grow: 1;
            color: #374151;
        }
        .component-status {
            font-size: 14px;
            font-weight: 500;
        }
        .component-ok .component-status {
            color: #059669;
        }
        .component-error .component-status {
            color: #dc2626;
        }
        .component-error-details {
            padding: 0 15px 15px 15px;
            color: #7f1d1d;
            background-color: rgba(239, 68, 68, 0.05);
            font-family: 'Monaco', 'Consolas', monospace;
            font-size: 13px;
            border-top: 1px solid rgba(239, 68, 68, 0.1);
        }
        .actions {
            background-color: #f8fafc;
            border-radius: 8px;
            padding: 20px;
            margin: 25px 0;
        }
        .actions h3 {
            margin: 0 0 15px 0;
            color: #1f2937;
            font-size: 18px;
        }
        .actions ul {
            margin: 0;
            padding-left: 20px;
        }
        .actions li {
            margin-bottom: 8px;
            color: #4b5563;
        }
        .footer { 
            background-color: #f9fafb; 
            padding: 20px; 
            text-align: center; 
            font-size: 13px; 
            color: #6b7280;
            border-top: 1px solid #e5e7eb;
        }
        .timestamp {
            font-size: 12px;
            color: #9ca3af;
            margin-top: 10px;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>� Alerta de Infraestrutura</h1>
            <p>Sistema de Monitoramento IAFarma</p>
        </div>
        
        <div class="content">
            <div class="alert-summary">
                <h2>Problemas Detectados</h2>
                <div class="alert-count">%d</div>
                <p>componente(s) com problemas identificados</p>
            </div>

            <h3 style="margin-bottom: 20px; color: #1f2937;">📊 Status dos Componentes</h3>
            %s

            <div class="actions">
                <h3>🔧 Ações Recomendadas</h3>
                <ul>
                    %s
                    <li>📋 <strong>Logs:</strong> Consultar logs do sistema para informações detalhadas</li>
                    <li>🔄 <strong>Monitoramento:</strong> O sistema continuará verificando a cada 5 minutos</li>
                </ul>
            </div>

            <p style="color: #6b7280; font-size: 14px; margin-top: 30px;">
                <strong>� Sobre este alerta:</strong> Este email foi enviado automaticamente pelo sistema de monitoramento. 
                Você não receberá novos alertas até que os problemas sejam resolvidos.
            </p>
        </div>
        
        <div class="footer">
            <p><strong>Sistema de Monitoramento Automático</strong><br>
            IAFarma - Infraestrutura</p>
            <div class="timestamp">Detectado em: %s</div>
        </div>
    </div>
</body>
</html>
	`,
		headerColor, func() string {
			// Gradiente mais escuro para o cabeçalho
			if headerColor == "#FF9800" {
				return "#F57C00"
			}
			return "#E53E3E"
		}(),
		problemsCount,
		componentsHTML,
		actionsHTML,
		status.Timestamp.Format("02/01/2006 às 15:04:05"),
	)
}
