package services

import (
	"bytes"
	"fmt"
	"html/template"
	"iafarma/internal/repo"
	"iafarma/pkg/models"
	"net/smtp"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EmailService handles email operations
type EmailService struct {
	// SMTP configuration
	smtpHost     string
	smtpPort     string
	smtpUser     string
	smtpPassword string
	fromEmail    string

	// AWS SES configuration
	sesClient *ses.SES
	useSES    bool

	notificationRepo *repo.NotificationRepository
}

// NewEmailService creates a new email service
func NewEmailService(db *gorm.DB) (*EmailService, error) {
	emailService := &EmailService{
		notificationRepo: repo.NewNotificationRepository(db),
	}

	// Check for AWS SES configuration first
	awsRegion := os.Getenv("AWS_REGION")
	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	sesFromEmail := os.Getenv("SES_FROM_EMAIL")

	if awsRegion != "" && awsAccessKey != "" && awsSecretKey != "" && sesFromEmail != "" {
		// Initialize AWS session
		sess, err := session.NewSession(&aws.Config{
			Region:      aws.String(awsRegion),
			Credentials: credentials.NewStaticCredentials(awsAccessKey, awsSecretKey, ""),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create AWS session: %w", err)
		}

		// Create SES client - let AWS SDK determine the endpoint
		emailService.sesClient = ses.New(sess)
		emailService.fromEmail = sesFromEmail
		emailService.useSES = true

		// Log SES configuration for debugging
		fmt.Printf("✅ AWS SES configured for region: %s, from: %s\n", awsRegion, sesFromEmail)
		return emailService, nil
	}

	// Fallback to SMTP configuration
	smtpHost := os.Getenv("SMTP_HOST")
	smtpPort := os.Getenv("SMTP_PORT")
	smtpUser := os.Getenv("SMTP_USER")
	smtpPassword := os.Getenv("SMTP_PASSWORD")
	fromEmail := os.Getenv("FROM_EMAIL")

	if smtpHost == "" || smtpPort == "" || smtpUser == "" || smtpPassword == "" || fromEmail == "" {
		return nil, fmt.Errorf("email service not configured. Set either AWS SES credentials (AWS_REGION, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, SES_FROM_EMAIL) or SMTP credentials (SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_PASSWORD, FROM_EMAIL)")
	}

	emailService.smtpHost = smtpHost
	emailService.smtpPort = smtpPort
	emailService.smtpUser = smtpUser
	emailService.smtpPassword = smtpPassword
	emailService.fromEmail = fromEmail
	emailService.useSES = false

	return emailService, nil
}

// SendEmail sends an email using SES or SMTP
func (s *EmailService) SendEmail(to []string, subject, body string) error {
	if s.useSES {
		return s.sendEmailWithSES(to, subject, body)
	}
	return s.sendEmailWithSMTP(to, subject, body)
}

// sendEmailWithSES sends email using Amazon SES
func (s *EmailService) sendEmailWithSES(to []string, subject, body string) error {
	if s.sesClient == nil {
		return fmt.Errorf("SES client not configured")
	}

	// Convert to slice to AWS string pointers
	var toAddresses []*string
	for _, addr := range to {
		toAddresses = append(toAddresses, aws.String(addr))
	}

	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			ToAddresses: toAddresses,
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String("UTF-8"),
					Data:    aws.String(body),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String("UTF-8"),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(s.fromEmail),
	}

	_, err := s.sesClient.SendEmail(input)
	if err != nil {
		// Add more detailed error logging for SES issues
		fmt.Printf("❌ SES Error Details - Region: %s, From: %s, To: %v, Error: %v\n",
			*s.sesClient.Config.Region, s.fromEmail, to, err)
		return fmt.Errorf("failed to send email via SES: %w", err)
	}

	fmt.Printf("✅ Email sent successfully via SES to: %v\n", to)
	return nil
}

// sendEmailWithSMTP sends email using SMTP
func (s *EmailService) sendEmailWithSMTP(to []string, subject, body string) error {
	if s.smtpHost == "" {
		return fmt.Errorf("SMTP service not configured")
	}

	// Create message
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.fromEmail, to[0], subject, body)

	// Set up authentication
	auth := smtp.PlainAuth("", s.smtpUser, s.smtpPassword, s.smtpHost)

	// Send email
	err := smtp.SendMail(s.smtpHost+":"+s.smtpPort, auth, s.fromEmail, to, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email via SMTP: %w", err)
	}

	return nil
}

// LogNotification logs a notification attempt
func (s *EmailService) LogNotification(tenantID uuid.UUID, notificationType, recipient, subject, body, status, errorMessage string) error {
	log := &models.NotificationLog{
		BaseTenantModel: models.BaseTenantModel{
			TenantID: tenantID,
		},
		Type:         notificationType,
		Recipient:    recipient,
		Subject:      subject,
		Body:         body,
		Status:       status,
		ErrorMessage: errorMessage,
	}

	if status == "sent" {
		now := time.Now()
		log.SentAt = &now
	}

	return s.notificationRepo.CreateLog(log)
}

// SendDailySalesReport sends daily sales report to tenant
func (s *EmailService) SendDailySalesReport(tenantID uuid.UUID, tenantName, tenantEmail string, reportData map[string]interface{}) error {
	// Check if already sent today
	today := time.Now().Format("2006-01-02")
	if s.notificationRepo.IsNotificationSentToday(tenantID, "daily_sales_report", today) {
		return fmt.Errorf("daily sales report already sent today")
	}

	// Render email template
	subject := fmt.Sprintf("Relatório Diário de Vendas - %s", tenantName)
	body, err := s.renderSalesReportTemplate(reportData)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	// Send email
	err = s.SendEmail([]string{tenantEmail}, subject, body)
	if err != nil {
		// Log failed notification
		s.LogNotification(tenantID, "daily_sales_report", tenantEmail, subject, body, "failed", err.Error())
		return err
	}

	// Log successful notification
	s.LogNotification(tenantID, "daily_sales_report", tenantEmail, subject, body, "sent", "")

	// Mark as sent for today
	s.notificationRepo.MarkNotificationSent(tenantID, "daily_sales_report", today)

	return nil
}

// SendLowStockAlert sends low stock alert to tenant
func (s *EmailService) SendLowStockAlert(tenantID uuid.UUID, tenantName, tenantEmail string, lowStockProducts []map[string]interface{}) error {
	// Check if already sent today
	today := time.Now().Format("2006-01-02")
	if s.notificationRepo.IsNotificationSentToday(tenantID, "low_stock_alert", today) {
		return fmt.Errorf("low stock alert already sent today")
	}

	// Render email template
	subject := fmt.Sprintf("Alerta de Estoque Baixo - %s", tenantName)
	body, err := s.renderLowStockTemplate(lowStockProducts)
	if err != nil {
		return fmt.Errorf("failed to render email template: %w", err)
	}

	// Send email
	err = s.SendEmail([]string{tenantEmail}, subject, body)
	if err != nil {
		// Log failed notification
		s.LogNotification(tenantID, "low_stock_alert", tenantEmail, subject, body, "failed", err.Error())
		return err
	}

	// Log successful notification
	s.LogNotification(tenantID, "low_stock_alert", tenantEmail, subject, body, "sent", "")

	// Mark as sent for today
	s.notificationRepo.MarkNotificationSent(tenantID, "low_stock_alert", today)

	return nil
}

// renderSalesReportTemplate renders the sales report email template
func (s *EmailService) renderSalesReportTemplate(data map[string]interface{}) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .header { background-color: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .summary { background-color: #f9f9f9; padding: 15px; border-radius: 5px; margin: 20px 0; }
        .metric { display: inline-block; margin: 10px; text-align: center; }
        .metric-value { font-size: 24px; font-weight: bold; color: #4CAF50; }
        .metric-label { font-size: 14px; color: #666; }
        .footer { background-color: #f1f1f1; padding: 20px; text-align: center; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="header">
        <h1>📊 Relatório Diário de Vendas</h1>
        <p>{{.Date}}</p>
    </div>
    
    <div class="content">
        <div class="summary">
            <h2>Resumo do Dia</h2>
            <div class="metric">
                <div class="metric-value">{{.TotalOrders}}</div>
                <div class="metric-label">Pedidos</div>
            </div>
            <div class="metric">
                <div class="metric-value">R$ {{.TotalRevenue}}</div>
                <div class="metric-label">Faturamento</div>
            </div>
            <div class="metric">
                <div class="metric-value">{{.TotalProducts}}</div>
                <div class="metric-label">Produtos Vendidos</div>
            </div>
        </div>
        
        {{if .TopProducts}}
        <h3>🏆 Produtos Mais Vendidos</h3>
        <ul>
        {{range .TopProducts}}
            <li>{{.Name}} - {{.Quantity}} unidades vendidas</li>
        {{end}}
        </ul>
        {{end}}
    </div>
    
    <div class="footer">
        <p>Este é um relatório automático do seu sistema de vendas.</p>
    </div>
</body>
</html>
`

	t, err := template.New("sales_report").Parse(tmpl)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// renderLowStockTemplate renders the low stock alert email template
func (s *EmailService) renderLowStockTemplate(products []map[string]interface{}) (string, error) {
	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .header { background-color: #FF9800; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; }
        .product { background-color: #fff3cd; border: 1px solid #ffeaa7; padding: 10px; margin: 10px 0; border-radius: 5px; }
        .product-name { font-weight: bold; color: #e17055; }
        .stock-info { color: #666; }
        .footer { background-color: #f1f1f1; padding: 20px; text-align: center; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="header">
        <h1>⚠️ Alerta de Estoque Baixo</h1>
        <p>{{.Date}}</p>
    </div>
    
    <div class="content">
        <p>Os seguintes produtos estão com estoque baixo e precisam de reposição:</p>
        
        {{range .Products}}
        <div class="product">
            <div class="product-name">{{.Name}}</div>
            <div class="stock-info">
                Estoque atual: {{.CurrentStock}} unidades<br>
                Limite mínimo: {{.MinStock}} unidades
            </div>
        </div>
        {{end}}
        
        <p><strong>Recomendação:</strong> Faça a reposição destes produtos o quanto antes para evitar rupturas de estoque.</p>
    </div>
    
    <div class="footer">
        <p>Este é um alerta automático do seu sistema de gestão de estoque.</p>
    </div>
</body>
</html>
`

	t, err := template.New("low_stock").Parse(tmpl)
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"Date":     time.Now().Format("02/01/2006"),
		"Products": products,
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// SendPasswordResetEmail sends a password reset email to the user
func (s *EmailService) SendPasswordResetEmail(email, userName, resetToken string) error {
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", frontendURL, resetToken)

	subject := "Redefinição de Senha - Vendas Zap"
	body := s.renderPasswordResetTemplate(userName, resetURL)

	return s.SendEmail([]string{email}, subject, body)
}

// renderPasswordResetTemplate renders the password reset email template
func (s *EmailService) renderPasswordResetTemplate(userName, resetURL string) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Redefinição de Senha</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #4CAF50; color: white; text-align: center; padding: 20px; border-radius: 8px 8px 0 0; }
        .content { background-color: #f9f9f9; padding: 30px; border-radius: 0 0 8px 8px; }
        .button { display: inline-block; padding: 12px 24px; background-color: #4CAF50; color: white; text-decoration: none; border-radius: 4px; margin: 20px 0; }
        .footer { text-align: center; margin-top: 20px; font-size: 12px; color: #666; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Redefinição de Senha</h1>
        </div>
        <div class="content">
            <p>Olá %s,</p>
            <p>Recebemos uma solicitação para redefinir a senha da sua conta no Vendas Zap.</p>
            <p>Para criar uma nova senha, clique no botão abaixo:</p>
            <a href="%s" class="button">Redefinir Senha</a>
            <p><strong>Este link expira em 1 hora.</strong></p>
            <p>Se você não solicitou esta redefinição, pode ignorar este email com segurança. Sua senha atual permanecerá inalterada.</p>
            <p>Se o botão não funcionar, copie e cole o seguinte link no seu navegador:</p>
            <p style="word-break: break-all; color: #666;">%s</p>
        </div>
        <div class="footer">
            <p>Este é um email automático. Por favor, não responda.</p>
            <p>&copy; 2025 Vendas Zap. Todos os direitos reservados.</p>
        </div>
    </div>
</body>
</html>
`, userName, resetURL, resetURL)
}
