package handlers

import (
	"fmt"
	"net/http"

	"iafarma/pkg/models"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// SendEmailRequest represents the request to send order email
type SendEmailRequest struct {
	OrderID   string `json:"order_id" validate:"required"`
	Recipient string `json:"recipient" validate:"required,email"`
	Subject   string `json:"subject" validate:"required"`
	Message   string `json:"message"`
}

// SendEmail godoc
// @Summary Send order details via email
// @Description Send order details to a customer via email
// @Tags orders
// @Accept json
// @Produce json
// @Param request body SendEmailRequest true "Email request data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /orders/send-email [post]
// @Security BearerAuth
func (h *OrderHandler) SendEmail(c echo.Context) error {
	// Get tenant ID from context
	tenantID, ok := c.Get("tenant_id").(uuid.UUID)
	if !ok {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "Invalid tenant"})
	}

	// Parse request
	var req SendEmailRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid request data"})
	}

	// Validate request
	if req.OrderID == "" || req.Recipient == "" || req.Subject == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "OrderID, recipient and subject are required"})
	}

	// Parse order ID
	orderUUID, err := uuid.Parse(req.OrderID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "Invalid order ID"})
	}

	// Get the order
	order, err := h.orderRepo.GetByID(tenantID, orderUUID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "Order not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "Failed to get order"})
	}

	// TODO: Implementar integração com Amazon SES
	// Por enquanto, vamos simular o envio do email
	emailBody := h.generateEmailBody(order, req.Message)

	// Log the email sending attempt (for debugging)
	fmt.Printf("Sending email to %s with subject: %s\n", req.Recipient, req.Subject)
	fmt.Printf("Email body preview: %s...\n", emailBody[:100])

	// Simulate successful email sending
	// In a real implementation, this would integrate with Amazon SES

	return c.JSON(http.StatusOK, map[string]string{
		"message":      "Email sent successfully",
		"recipient":    req.Recipient,
		"order_number": order.OrderNumber,
	})
}

// generateEmailBody generates HTML email body for order details
func (h *OrderHandler) generateEmailBody(order *models.Order, customMessage string) string {
	html := fmt.Sprintf(`
		<html>
		<head>
			<style>
				body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
				.container { max-width: 600px; margin: 0 auto; padding: 20px; }
				.header { background-color: #f8f9fa; padding: 20px; text-align: center; border-radius: 8px 8px 0 0; }
				.content { background-color: #ffffff; padding: 20px; border: 1px solid #e9ecef; }
				.footer { background-color: #f8f9fa; padding: 15px; text-align: center; border-radius: 0 0 8px 8px; font-size: 12px; color: #6c757d; }
				.order-info { margin: 20px 0; }
				.order-info th, .order-info td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
				.order-info th { background-color: #f8f9fa; }
				.total { font-weight: bold; font-size: 1.2em; color: #28a745; }
			</style>
		</head>
		<body>
			<div class="container">
				<div class="header">
					<h1>Detalhes do Pedido #%s</h1>
				</div>
				<div class="content">
					%s
					
					<div class="order-info">
						<h3>Informações do Pedido</h3>
						<p><strong>Número:</strong> %s</p>
						<p><strong>Status:</strong> %s</p>
						<p><strong>Data:</strong> %s</p>
					</div>
					
					<div class="order-info">
						<h3>Dados do Cliente</h3>
						<p><strong>Nome:</strong> %s</p>
						<p><strong>Email:</strong> %s</p>
						<p><strong>Telefone:</strong> %s</p>
					</div>
					
					<div class="order-info">
						<h3>Resumo Financeiro</h3>
						<table style="width: 100%%; border-collapse: collapse;">
							<tr><td>Subtotal:</td><td>R$ %s</td></tr>
							<tr><td>Desconto:</td><td>R$ %s</td></tr>
							<tr><td>Frete:</td><td>R$ %s</td></tr>
							<tr><td>Taxa:</td><td>R$ %s</td></tr>
							<tr class="total"><td><strong>Total:</strong></td><td><strong>R$ %s</strong></td></tr>
						</table>
					</div>
				</div>
				<div class="footer">
					<p>Este é um email automático. Não responda a esta mensagem.</p>
				</div>
			</div>
		</body>
		</html>
	`,
		order.OrderNumber,
		customMessage,
		order.OrderNumber,
		order.Status,
		order.CreatedAt.Format("02/01/2006 15:04"),
		h.safeString(order.CustomerName),
		h.safeString(order.CustomerEmail),
		h.safeString(order.CustomerPhone),
		order.Subtotal,
		order.DiscountAmount,
		order.ShippingAmount,
		order.TaxAmount,
		order.TotalAmount,
	)

	return html
}

// safeString safely dereferences a string pointer
func (h *OrderHandler) safeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
