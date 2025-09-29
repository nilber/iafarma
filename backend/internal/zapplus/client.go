package zapplus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Client representa o cliente para interagir com a API do ZapPlus
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// SendTextRequest representa a estrutura para envio de mensagem de texto
type SendTextRequest struct {
	ChatID                 string `json:"chatId"`
	Text                   string `json:"text"`
	LinkPreview            bool   `json:"linkPreview"`
	LinkPreviewHighQuality bool   `json:"linkPreviewHighQuality"`
	Session                string `json:"session"`
}

// SendTextResponse representa a resposta da API para envio de texto
type SendTextResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	ID      string `json:"id,omitempty"`
}

// ZapPlusTextResponse representa a resposta completa da API ZapPlus para envio de texto
type ZapPlusTextResponse struct {
	Data struct {
		ID struct {
			ID string `json:"id"`
		} `json:"id"`
	} `json:"_data"`
}

// SessionResponse representa a resposta da API de sessão
type SessionResponse struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Config struct {
		Metadata interface{} `json:"metadata"`
		Webhooks []struct {
			URL    string   `json:"url"`
			Events []string `json:"events"`
		} `json:"webhooks"`
	} `json:"config"`
	Me struct {
		ID       string `json:"id"`
		PushName string `json:"pushName"`
	} `json:"me"`
	Engine struct {
		Engine      string `json:"engine"`
		WWebVersion string `json:"WWebVersion"`
		State       string `json:"state"`
	} `json:"engine"`
}

// GroupResponse representa a resposta da API de grupos
type GroupResponse struct {
	Success bool `json:"success"`
	Data    []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
}

// Singleton instance
var instance *Client

// GetClient retorna a instância singleton do cliente ZapPlus
func GetClient() *Client {
	if instance == nil {
		baseURL := os.Getenv("ZAPPLUS_BASE_URL")
		if baseURL == "" {
			baseURL = "http://zap-plus.heltec.com.br:3000" // fallback
		}

		instance = &Client{
			baseURL: baseURL,
			httpClient: &http.Client{
				Timeout: 30 * time.Second,
			},
		}
	}
	return instance
}

// NewClient cria uma nova instância do cliente ZapPlus (para testes)
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendTextMessage envia uma mensagem de texto para um chat específico
func (c *Client) SendTextMessage(session, chatID, text string) error {
	return c.SendTextMessageWithPreview(session, chatID, text, true, false)
}

// SendTextMessageWithResponse envia uma mensagem de texto e retorna a resposta completa
func (c *Client) SendTextMessageWithResponse(session, chatID, text string) (*ZapPlusTextResponse, error) {
	request := SendTextRequest{
		ChatID:                 chatID,
		Text:                   text,
		LinkPreview:            true,
		LinkPreviewHighQuality: false,
		Session:                session,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/sendText", c.baseURL)
	time.Sleep(1 * time.Second) // evitar rate limit
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("ZapPlus API returned status %d", resp.StatusCode)
	}

	var response ZapPlusTextResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	log.Printf("📤 ZapPlus message sent to %s via session %s (ID: %s)", chatID, session, response.Data.ID.ID)
	return &response, nil
}

// SendTextMessageWithPreview envia uma mensagem de texto com configurações de preview
func (c *Client) SendTextMessageWithPreview(session, chatID, text string, linkPreview, linkPreviewHQ bool) error {
	request := SendTextRequest{
		ChatID:                 chatID,
		Text:                   text,
		LinkPreview:            linkPreview,
		LinkPreviewHighQuality: linkPreviewHQ,
		Session:                session,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/sendText", c.baseURL)
	time.Sleep(1 * time.Second) // evitar rate limit
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("ZapPlus API returned status %d", resp.StatusCode)
	}

	log.Printf("📤 ZapPlus message sent to %s via session %s", chatID, session)
	return nil
}

// SendGroupMessage envia uma mensagem para um grupo
func (c *Client) SendGroupMessage(session, groupID, text string) error {
	return c.SendTextMessage(session, groupID, text)
}

// GetSessionStatus verifica o status de uma sessão
func (c *Client) GetSessionStatus(session string) (*SessionResponse, error) {
	url := fmt.Sprintf("%s/api/sessions/%s", c.baseURL, session)
	// fmt.Println(url)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get session status: %w", err)
	}
	defer resp.Body.Close()

	var response SessionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// GetQRCode obtém o QR code para conectar uma sessão
func (c *Client) GetQRCode(session string) (string, error) {
	status, err := c.GetSessionStatus(session)
	if err != nil {
		return "", err
	}

	// Para sessões que não estão conectadas, precisamos usar o endpoint específico de QR code
	if status.Status != "WORKING" {
		// Tentar obter QR code através do endpoint específico
		qrResp, err := c.GetQRCodeImage(session)
		if err != nil {
			return "", fmt.Errorf("session not connected and QR code unavailable: %w", err)
		}
		qrResp.Body.Close()
		return "QR code available via image endpoint", nil
	}

	return "", fmt.Errorf("session already connected")
}

// GetQRCodeImage obtém o QR code como imagem para conectar uma sessão
func (c *Client) GetQRCodeImage(session string) (*http.Response, error) {
	url := fmt.Sprintf("%s/api/%s/auth/qr?format=image", c.baseURL, session)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get QR code image: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("ZapPlus API returned status %d for QR code", resp.StatusCode)
	}

	return resp, nil
}

// GetGroups lista os grupos disponíveis em uma sessão
func (c *Client) GetGroups(session string) (*GroupResponse, error) {
	url := fmt.Sprintf("%s/api/%s/groups", c.baseURL, session)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %w", err)
	}
	defer resp.Body.Close()

	var response GroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}

// AddParticipantToGroup adiciona um participante a um grupo
func (c *Client) AddParticipantToGroup(session, groupID, participantPhone string) error {
	type AddParticipantRequest struct {
		Participants []struct {
			ID string `json:"id"`
		} `json:"participants"`
	}
	participantPhone = strings.ReplaceAll(participantPhone, "+", "")
	request := AddParticipantRequest{
		Participants: []struct {
			ID string `json:"id"`
		}{
			{ID: fmt.Sprintf("%s@c.us", participantPhone)},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// encodedGroupID := url.QueryEscape(groupID)
	url := fmt.Sprintf("%s/api/%s/groups/%s/participants/add", c.baseURL, session, groupID)

	fmt.Println(string(jsonData))
	fmt.Println(url)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println(err)
		// print response
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))

		return fmt.Errorf("failed to add participant: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("ZapPlus API returned status %d", resp.StatusCode)
	}

	return nil
}

// RemoveParticipantFromGroup remove um participante de um grupo
func (c *Client) RemoveParticipantFromGroup(session, groupID, participantPhone string) error {
	type RemoveParticipantRequest struct {
		Participants []struct {
			ID string `json:"id"`
		} `json:"participants"`
	}

	participantPhone = strings.ReplaceAll(participantPhone, "+", "")
	request := RemoveParticipantRequest{
		Participants: []struct {
			ID string `json:"id"`
		}{
			{ID: fmt.Sprintf("%s@c.us", participantPhone)},
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// encodedGroupID := url.QueryEscape(groupID)
	url := fmt.Sprintf("%s/api/%s/groups/%s/participants/remove", c.baseURL, session, groupID)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to remove participant: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("ZapPlus API returned status %d", resp.StatusCode)
	}

	return nil
}

// SetGroupMessagesAdminOnly configura se apenas admins podem enviar mensagens no grupo
func (c *Client) SetGroupMessagesAdminOnly(session, groupID string, adminOnly bool) error {
	type AdminOnlyRequest struct {
		AdminsOnly bool `json:"adminsOnly"`
	}

	request := AdminOnlyRequest{
		AdminsOnly: adminOnly,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	encodedGroupID := url.QueryEscape(groupID)
	requestURL := fmt.Sprintf("%s/api/%s/groups/%s/settings/security/messages-admin-only", c.baseURL, session, encodedGroupID)

	req, err := http.NewRequest("PUT", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to set admin only: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("ZapPlus API returned status %d", resp.StatusCode)
	}

	return nil
}

// GetGroupInfo obtém informações sobre um grupo específico
func (c *Client) GetGroupInfo(session, groupID string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/api/%s/groups/%s", c.baseURL, session, groupID)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get group info: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
}

// FormatPhoneToWhatsApp formata um número de telefone para o formato WhatsApp
func FormatPhoneToWhatsApp(phone string) string {
	// Remove caracteres não numéricos
	cleaned := ""
	for _, char := range phone {
		if char >= '0' && char <= '9' {
			cleaned += string(char)
		}
	}

	// Adiciona o sufixo @c.us
	return fmt.Sprintf("%s@c.us", cleaned)
}

// IsValidSession verifica se uma sessão está válida e conectada
func (c *Client) IsValidSession(session string) bool {
	status, err := c.GetSessionStatus(session)
	if err != nil {
		log.Printf("❌ Error checking session %s: %v", session, err)
		return false
	}

	// Status WORKING significa que a sessão está conectada e funcionando
	return status.Status == "WORKING" && status.Engine.State == "CONNECTED"
}

// GroupParticipant representa um participante do grupo
type GroupParticipant struct {
	ID string `json:"id"`
}

// CreateGroupRequest representa uma requisição para criar grupo
type CreateGroupRequest struct {
	Name         string             `json:"name"`
	Participants []GroupParticipant `json:"participants"`
}

// CreateGroupResponse representa a resposta da criação de grupo
type CreateGroupResponse struct {
	Title        string                 `json:"title"`
	GID          GroupID                `json:"gid"`
	Participants map[string]interface{} `json:"participants"`
}

// GroupID representa o ID de um grupo
type GroupID struct {
	Server     string `json:"server"`
	User       string `json:"user"`
	Serialized string `json:"_serialized"`
}

// CreateGroup cria um novo grupo WhatsApp
func (c *Client) CreateGroup(session, name string, participants []string) (*CreateGroupResponse, error) {
	var groupParticipants []GroupParticipant
	for _, phone := range participants {
		groupParticipants = append(groupParticipants, GroupParticipant{
			ID: FormatPhoneToWhatsApp(phone),
		})
	}

	request := CreateGroupRequest{
		Name:         name,
		Participants: groupParticipants,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal create group request: %w", err)
	}

	url := fmt.Sprintf("%s/api/%s/groups", c.baseURL, session)

	fmt.Println("criando grupo zapplus...")
	fmt.Println(string(jsonData))
	fmt.Println(url)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create group: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("ZapPlus API returned status %d", resp.StatusCode)
	}

	var response CreateGroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode create group response: %w", err)
	}

	return &response, nil
}

// DeleteGroup deleta um grupo WhatsApp
func (c *Client) DeleteGroup(session, groupID string) error {
	url := fmt.Sprintf("%s/api/%s/groups/%s", c.baseURL, session, url.QueryEscape(groupID))

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete group: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("ZapPlus API returned status %d", resp.StatusCode)
	}

	return nil
}

// LocationRequest representa uma requisição para enviar localização
type LocationRequest struct {
	ChatID    string  `json:"chatId"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Title     string  `json:"title"`
	ReplyTo   *string `json:"reply_to,omitempty"`
	Session   string  `json:"session"`
}

// SendLocation envia uma localização via WhatsApp
func (c *Client) SendLocation(session, chatID string, latitude, longitude float64, title string) error {
	request := LocationRequest{
		ChatID:    chatID,
		Latitude:  latitude,
		Longitude: longitude,
		Title:     title,
		Session:   session,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal location request: %w", err)
	}

	url := fmt.Sprintf("%s/api/sendLocation", c.baseURL)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send location: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("ZapPlus API returned status %d", resp.StatusCode)
	}

	log.Printf("📍 ZapPlus location sent to %s via session %s", chatID, session)
	return nil
}

// MediaRequest representa uma requisição para enviar mídia (imagem, arquivo, áudio)
type MediaRequest struct {
	ChatID  string      `json:"chatId"`
	File    MediaFile   `json:"file"`
	Caption string      `json:"caption,omitempty"`
	ReplyTo interface{} `json:"reply_to"`
	Session string      `json:"session"`
}

// MediaFile representa o arquivo de mídia
type MediaFile struct {
	Mimetype string `json:"mimetype"`
	Filename string `json:"filename"`
	URL      string `json:"url"`
}

// SendImage envia uma imagem via WhatsApp
func (c *Client) SendImage(session, chatID, imageURL, caption string) error {
	response, err := c.SendImageWithResponse(session, chatID, imageURL, caption)
	if err != nil {
		return err
	}

	log.Printf("🖼️ ZapPlus image sent to %s via session %s", chatID, session)
	_ = response // ignora a resposta nesta versão legacy
	return nil
}

// SendImageWithResponse envia uma imagem via WhatsApp e retorna a resposta
func (c *Client) SendImageWithResponse(session, chatID, imageURL, caption string) (map[string]interface{}, error) {
	// Extrair mimetype e filename da URL
	mimetype := "image/jpeg" // default
	filename := "image.jpg"  // default

	// Tentar determinar o mimetype pela extensão da URL
	if strings.HasSuffix(strings.ToLower(imageURL), ".png") {
		mimetype = "image/png"
		filename = "image.png"
	} else if strings.HasSuffix(strings.ToLower(imageURL), ".gif") {
		mimetype = "image/gif"
		filename = "image.gif"
	} else if strings.HasSuffix(strings.ToLower(imageURL), ".webp") {
		mimetype = "image/webp"
		filename = "image.webp"
	}

	request := MediaRequest{
		ChatID: chatID,
		File: MediaFile{
			Mimetype: mimetype,
			Filename: filename,
			URL:      imageURL,
		},
		Caption: caption,
		ReplyTo: nil,
		Session: session,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal image request: %w", err)
	}

	// fmt.Println(string(jsonData))

	url := fmt.Sprintf("%s/api/sendImage", c.baseURL)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("ZapPlus API returned status %d", resp.StatusCode)
	}

	// Ler e parsear a resposta
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response, nil
}

// SendFile envia um arquivo via WhatsApp
func (c *Client) SendFile(session, chatID, fileURL, caption string) error {
	response, err := c.SendFileWithResponse(session, chatID, fileURL, caption)
	if err != nil {
		return err
	}

	log.Printf("📄 ZapPlus file sent to %s via session %s", chatID, session)
	_ = response // ignora a resposta nesta versão legacy
	return nil
}

// SendFileWithResponse envia um arquivo via WhatsApp e retorna a resposta
func (c *Client) SendFileWithResponse(session, chatID, fileURL, caption string) (map[string]interface{}, error) {
	// Determinar mimetype e filename para arquivo
	mimetype := "application/octet-stream" // default
	filename := "file"                     // default

	// Tentar determinar o tipo pela extensão
	if strings.HasSuffix(strings.ToLower(fileURL), ".pdf") {
		mimetype = "application/pdf"
		filename = "document.pdf"
	} else if strings.HasSuffix(strings.ToLower(fileURL), ".doc") || strings.HasSuffix(strings.ToLower(fileURL), ".docx") {
		mimetype = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		filename = "document.docx"
	} else if strings.HasSuffix(strings.ToLower(fileURL), ".xls") || strings.HasSuffix(strings.ToLower(fileURL), ".xlsx") {
		mimetype = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
		filename = "spreadsheet.xlsx"
	}

	request := MediaRequest{
		ChatID: chatID,
		File: MediaFile{
			Mimetype: mimetype,
			Filename: filename,
			URL:      fileURL,
		},
		Caption: caption,
		ReplyTo: nil,
		Session: session,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal file request: %w", err)
	}

	url := fmt.Sprintf("%s/api/sendFile", c.baseURL)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("ZapPlus API returned status %d", resp.StatusCode)
	}

	// Ler e parsear a resposta
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response, nil
}

// SendVoice envia um áudio via WhatsApp
func (c *Client) SendVoice(session, chatID, audioURL string) error {
	response, err := c.SendVoiceWithResponse(session, chatID, audioURL)
	if err != nil {
		return err
	}

	log.Printf("🎵 ZapPlus voice sent to %s via session %s", chatID, session)
	_ = response // ignora a resposta nesta versão legacy
	return nil
}

// SendVoiceWithResponse envia um áudio via WhatsApp e retorna a resposta
func (c *Client) SendVoiceWithResponse(session, chatID, audioURL string) (map[string]interface{}, error) {
	// Determinar mimetype e filename para áudio
	mimetype := "audio/mpeg" // default
	filename := "audio.mp3"  // default

	// Tentar determinar o tipo pela extensão
	if strings.HasSuffix(strings.ToLower(audioURL), ".wav") {
		mimetype = "audio/wav"
		filename = "audio.wav"
	} else if strings.HasSuffix(strings.ToLower(audioURL), ".ogg") {
		mimetype = "audio/ogg"
		filename = "audio.ogg"
	} else if strings.HasSuffix(strings.ToLower(audioURL), ".m4a") {
		mimetype = "audio/mp4"
		filename = "audio.m4a"
	}

	request := MediaRequest{
		ChatID: chatID,
		File: MediaFile{
			Mimetype: mimetype,
			Filename: filename,
			URL:      audioURL,
		},
		Caption: "",
		ReplyTo: nil,
		Session: session,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal voice request: %w", err)
	}

	url := fmt.Sprintf("%s/api/sendVoice", c.baseURL)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to send voice: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return nil, fmt.Errorf("ZapPlus API returned status %d", resp.StatusCode)
	}

	// Ler e parsear a resposta
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response, nil
}
