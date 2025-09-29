package webhook

// WebSocketNotifier defines the interface for WebSocket notifications
type WebSocketNotifier interface {
	BroadcastWebhookNotification(tenantID string, webhookType string, data interface{})
}
