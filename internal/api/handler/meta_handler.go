package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"crypto/sha256"
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
	"github.com/open-apime/apime/internal/pkg/response"
	instanceSvc "github.com/open-apime/apime/internal/service/instance"
	messageSvc "github.com/open-apime/apime/internal/service/message"
	templateSvc "github.com/open-apime/apime/internal/service/template"
	"github.com/open-apime/apime/internal/storage/media"
)

type MetaHandler struct {
	service         *messageSvc.Service
	instanceService *instanceSvc.Service
	templateService *templateSvc.Service
	mediaStorage    *media.Storage
}

func NewMetaHandler(service *messageSvc.Service, instanceService *instanceSvc.Service, templateService *templateSvc.Service, mediaStorage *media.Storage) *MetaHandler {
	return &MetaHandler{
		service:         service,
		instanceService: instanceService,
		templateService: templateService,
		mediaStorage:    mediaStorage,
	}
}

func (h *MetaHandler) Register(r *gin.RouterGroup) {
	// Rotas diretas (compatibilidade simples)
	r.GET("/meta/:id", h.getPhoneNumber)
	r.POST("/meta/:id/messages", h.sendMessage)

	// Rotas com versionamento (compatibilidade Meta Cloud API)
	// Ex: /v16.0/PHONE_ID/messages
	r.GET("/meta/:version/:id", h.getPhoneNumber)
	r.POST("/meta/:version/:id/messages", h.sendMessage)
	r.GET("/meta/:version/:id/business_profile", h.getBusinessAccount) // Placeholder logic uses phone ID as business ID
	r.GET("/meta/:version/:id/phone_numbers", h.getBusinessPhoneNumbers)
	r.POST("/meta/:version/:id/subscribed_apps", h.subscribeApp)
	r.POST("/meta/:version/:id/media", h.uploadMedia)
	r.GET("/meta/:version/:media_id", h.getMedia)
	r.GET("/meta/:version/:id/whatsapp_business_profile", h.getBusinessProfile)
	r.POST("/meta/:version/:id/whatsapp_business_profile", h.updateBusinessProfile)
	r.GET("/meta/:version/:id/message_templates", h.getTemplates)
	r.POST("/meta/:version/:id/message_templates", h.createTemplate)
	r.DELETE("/meta/:version/:id/message_templates", h.deleteTemplate)
}

func (h *MetaHandler) getPhoneNumber(c *gin.Context) {
	instanceID := c.Param("id")
	// Verificar token se necessário
	if c.GetString("authType") == "instance_token" {
		if c.GetString("instanceID") != instanceID {
			response.ErrorWithMessage(c, http.StatusForbidden, "token inválido para esta instância")
			return
		}
	}

	inst, err := h.instanceService.Get(c.Request.Context(), instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusNotFound, "instância não encontrada")
		return
	}

	// Formato da resposta Meta Get Phone Number
	c.JSON(http.StatusOK, gin.H{
		"verified_name":            inst.Name,
		"code_verification_status": "VERIFIED",
		"display_phone_number":     inst.WhatsAppJID, // JID cru ou formatado
		"quality_rating":           "GREEN",
		"account_mode":             "LIVE",
		"id":                       inst.ID,
	})
}

func (h *MetaHandler) getBusinessAccount(c *gin.Context) {
	// Mock response for Business Account
	// In Apime, the Instance ID or OwnerUserID acts as the business container
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"id":   id,
		"name": "Apime Business Account",
	})
}

func (h *MetaHandler) getBusinessPhoneNumbers(c *gin.Context) {
	// List instances as phone numbers
	// We use the authenticated user (from token) to list instances
	userID := c.GetString("userID")
	userRole := c.GetString("userRole")

	instances, err := h.instanceService.ListByUser(c.Request.Context(), userID, userRole)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	var data []gin.H
	for _, inst := range instances {
		data = append(data, gin.H{
			"verified_name":            inst.Name,
			"display_phone_number":     inst.WhatsAppJID,
			"id":                       inst.ID,
			"quality_rating":           "GREEN",
			"code_verification_status": "VERIFIED",
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": data,
	})
}

func (h *MetaHandler) subscribeApp(c *gin.Context) {
	// Mock success for webhooks subscription
	c.JSON(http.StatusOK, gin.H{
		"success": true,
	})
}

func (h *MetaHandler) uploadMedia(c *gin.Context) {
	instanceID := c.Param("id")
	// Validação de token já feita pelo middleware

	// Verificar permissão
	if c.GetString("authType") == "instance_token" {
		if c.GetString("instanceID") != instanceID {
			response.ErrorWithMessage(c, http.StatusForbidden, "token inválido para esta instância")
			return
		}
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "arquivo não fornecido")
		return
	}

	src, err := file.Open()
	if err != nil {
		response.ErrorWithMessage(c, http.StatusInternalServerError, "erro ao abrir arquivo")
		return
	}
	defer src.Close()

	fileData, err := io.ReadAll(src)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusInternalServerError, "erro ao ler arquivo")
		return
	}

	// Usar um ID temporário para o "messageID" já que não há mensagem associada ainda
	tempID := uuid.NewString()
	contentType := file.Header.Get("Content-Type")

	mediaID, err := h.mediaStorage.Save(c.Request.Context(), instanceID, tempID, fileData, contentType)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	// Codificar ID combinado (instanceID:mediaID)
	combined := fmt.Sprintf("%s:%s", instanceID, mediaID)
	metaMediaID := base64.StdEncoding.EncodeToString([]byte(combined))

	c.JSON(http.StatusOK, gin.H{
		"id": metaMediaID,
	})
}

func (h *MetaHandler) getMedia(c *gin.Context) {
	metaMediaID := c.Param("media_id")

	decodedBytes, err := base64.StdEncoding.DecodeString(metaMediaID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusBadRequest, "ID de mídia inválido")
		return
	}

	parts := splitID(string(decodedBytes))
	if len(parts) != 2 {
		response.ErrorWithMessage(c, http.StatusBadRequest, "ID de mídia malformado")
		return
	}

	instanceID := parts[0]
	mediaID := parts[1]

	// Verificar se mídia existe
	if !h.mediaStorage.Exists(instanceID, mediaID) {
		response.ErrorWithMessage(c, http.StatusNotFound, "mídia não encontrada")
		return
	}

	data, err := h.mediaStorage.Get(c.Request.Context(), instanceID, mediaID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	// Calcular hash SHA256
	hash := sha256.Sum256(data)
	sha256Hex := fmt.Sprintf("%x", hash)

	// Construir URL pública (assumindo rota /api/media do MediaHandler)
	// Host deve vir do request ou config
	host := c.Request.Host
	scheme := "http"
	if c.Request.TLS != nil {
		scheme = "https"
	}
	// Se estiver atrás de proxy, confiar no header X-Forwarded-Proto/Host se configurado,
	// mas por padrão Go/Gin usa Request.Host.
	publicURL := fmt.Sprintf("%s://%s/api/media/%s/%s", scheme, host, instanceID, mediaID)

	c.JSON(http.StatusOK, gin.H{
		"url":               publicURL,
		"mime_type":         http.DetectContentType(data), // Ou salvar/recuperar mime real se storage suportasse getMetadata
		"sha256":            sha256Hex,
		"file_size":         len(data),
		"id":                metaMediaID,
		"messaging_product": "whatsapp",
	})
}

func splitID(s string) []string {
	// Simple split by first colon
	for i := 0; i < len(s); i++ {
		if s[i] == ':' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func (h *MetaHandler) getBusinessProfile(c *gin.Context) {
	instanceID := c.Param("id")
	if c.GetString("authType") == "instance_token" {
		if c.GetString("instanceID") != instanceID {
			response.ErrorWithMessage(c, http.StatusForbidden, "token inválido para esta instância")
			return
		}
	}

	// Como não temos acesso direto ao sessionManager aqui, e messageService não expõe GetBusinessProfile,
	// teríamos que expandir o messageService ou instanceService.
	// Por simplicidade neste MVP, retornamos um perfil mockado baseado na instância.
	inst, err := h.instanceService.Get(c.Request.Context(), instanceID)
	if err != nil {
		response.ErrorWithMessage(c, http.StatusNotFound, "instância não encontrada")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": []gin.H{
			{
				"about":               "Available",
				"address":             "",
				"description":         "Powered by Apime",
				"email":               inst.OwnerEmail,
				"profile_picture_url": "https://placehold.co/200x200?text=" + inst.Name,
				"websites":            []string{},
				"vertical":            "OTHER",
				"messaging_product":   "whatsapp",
			},
		},
	})
}

func (h *MetaHandler) updateBusinessProfile(c *gin.Context) {
	// Mock update success
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (h *MetaHandler) getTemplates(c *gin.Context) {
	instanceID := c.Param("id")
	// Auth check skipped for brevity (middleware handles it mostly)

	templates, err := h.templateService.List(c.Request.Context(), instanceID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	var data []gin.H
	for _, t := range templates {
		var components []interface{}
		_ = json.Unmarshal([]byte(t.Components), &components)
		data = append(data, gin.H{
			"id":         t.ID,
			"name":       t.Name,
			"category":   t.Category,
			"language":   t.Language,
			"status":     t.Status,
			"components": components,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"paging": gin.H{
			"cursors": gin.H{
				"before": "",
				"after":  "",
			},
		},
	})
}

type CreateTemplateRequest struct {
	Name       string        `json:"name"`
	Category   string        `json:"category"`
	Language   string        `json:"language"`
	Components []interface{} `json:"components"`
}

func (h *MetaHandler) createTemplate(c *gin.Context) {
	instanceID := c.Param("id")
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	// Meta API accepts "en_US" but storage might prefer normalized or as is.
	// We store as is.

	tmpl, err := h.templateService.Create(c.Request.Context(), templateSvc.CreateInput{
		InstanceID: instanceID,
		Name:       req.Name,
		Category:   req.Category,
		Language:   req.Language,
		Components: req.Components,
	})
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       tmpl.ID,
		"status":   tmpl.Status,
		"category": tmpl.Category,
	})
}

func (h *MetaHandler) deleteTemplate(c *gin.Context) {
	instanceID := c.Param("id")
	name := c.Query("name")
	if name == "" {
		response.ErrorWithMessage(c, http.StatusBadRequest, "parametro 'name' é obrigatório")
		return
	}

	if err := h.templateService.DeleteByName(c.Request.Context(), instanceID, name); err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MetaRequest representa a estrutura de envio de mensagem da Cloud API
type MetaRequest struct {
	MessagingProduct string           `json:"messaging_product"`
	To               string           `json:"to"`
	Type             string           `json:"type"`
	Text             *MetaText        `json:"text,omitempty"`
	Image            *MetaMedia       `json:"image,omitempty"`
	Video            *MetaMedia       `json:"video,omitempty"`
	Audio            *MetaMedia       `json:"audio,omitempty"`
	Document         *MetaMedia       `json:"document,omitempty"`
	Sticker          *MetaMedia       `json:"sticker,omitempty"`
	Location         *MetaLocation    `json:"location,omitempty"`
	Contacts         []*MetaContact   `json:"contacts,omitempty"`
	Interactive      *MetaInteractive `json:"interactive,omitempty"`
	Template         *MetaTemplate    `json:"template,omitempty"`
}

type MetaTemplate struct {
	Name       string           `json:"name"`
	Language   MetaTemplateLang `json:"language"`
	Components []interface{}    `json:"components"`
}

type MetaTemplateLang struct {
	Code string `json:"code"`
}

type MetaText struct {
	Body       string `json:"body"`
	PreviewURL bool   `json:"preview_url"`
}

type MetaMedia struct {
	ID       string `json:"id,omitempty"`
	Link     string `json:"link,omitempty"`
	Caption  string `json:"caption,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type MetaLocation struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
	Name      string  `json:"name,omitempty"`
	Address   string  `json:"address,omitempty"`
}

type MetaContact struct {
	Name MetaContactName `json:"name"`
}

type MetaContactName struct {
	FormattedName string `json:"formatted_name"`
	FirstName     string `json:"first_name"`
}

type MetaInteractive struct {
	Type string `json:"type"`
	// Outros campos omitidos por simplicidade
}

func (h *MetaHandler) sendMessage(c *gin.Context) {
	instanceID := c.Param("id")
	// Verificar token se necessário (Assumindo middleware de auth já valida)
	// Mas o middleware Auth no router valida Bearer Token geral ou Instance Token.
	// Se for instance token, deve bater com o ID.
	if c.GetString("authType") == "instance_token" {
		if c.GetString("instanceID") != instanceID {
			response.ErrorWithMessage(c, http.StatusForbidden, "token inválido para esta instância")
			return
		}
	}

	var req MetaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	if req.MessagingProduct != "whatsapp" {
		response.ErrorWithMessage(c, http.StatusBadRequest, "messaging_product deve ser 'whatsapp'")
		return
	}

	input := messageSvc.SendInput{
		InstanceID: instanceID,
		To:         req.To,
		Type:       req.Type,
	}

	switch req.Type {
	case "text":
		if req.Text == nil {
			response.ErrorWithMessage(c, http.StatusBadRequest, "campo 'text' obrigatório")
			return
		}
		input.Text = req.Text.Body
	case "image", "video", "audio", "document":
		var media *MetaMedia
		switch req.Type {
		case "image":
			media = req.Image
		case "video":
			media = req.Video
		case "audio":
			media = req.Audio
		case "document":
			media = req.Document
		}

		if media == nil {
			response.ErrorWithMessage(c, http.StatusBadRequest, "campo de mídia obrigatório")
			return
		}

		// Suporte apenas a Link por enquanto
		if media.Link == "" {
			response.ErrorWithMessage(c, http.StatusBadRequest, "apenas 'link' é suportado para mídia no momento (upload de ID não implementado)")
			return
		}

		// Baixar a mídia do link
		data, contentType, err := h.downloadMedia(media.Link)
		if err != nil {
			response.ErrorWithMessage(c, http.StatusBadRequest, "falha ao baixar mídia do link: "+err.Error())
			return
		}

		input.MediaData = data
		input.MediaType = contentType // "image/jpeg", etc.
		input.Caption = media.Caption
		input.FileName = media.Filename

	case "template":
		if req.Template == nil {
			response.ErrorWithMessage(c, http.StatusBadRequest, "campo 'template' obrigatório")
			return
		}

		// Buscar template e renderizar
		bodyText, _, _, err := h.templateService.RenderTemplate(
			c.Request.Context(),
			instanceID,
			req.Template.Name,
			req.Template.Language.Code,
			nil, // TODO: Passar componentes/parâmetros para substituição correta
		)
		if err != nil {
			response.ErrorWithMessage(c, http.StatusBadRequest, "erro ao processar template: "+err.Error())
			return
		}

		input.Type = "text" // Converte para texto simples por enquanto
		input.Text = bodyText

	default:
		response.ErrorWithMessage(c, http.StatusBadRequest, "tipo de mensagem não suportado: "+req.Type)
		return
	}

	msg, err := h.service.Send(c.Request.Context(), input)
	if err != nil {
		if errors.Is(err, messageSvc.ErrInstanceNotConnected) {
			response.ErrorWithMessage(c, http.StatusBadRequest, "instância não conectada")
		} else {
			response.Error(c, http.StatusInternalServerError, err)
		}
		return
	}

	// Resposta no formato Meta
	c.JSON(http.StatusOK, gin.H{
		"messaging_product": "whatsapp",
		"contacts": []gin.H{
			{
				"input": req.To,
				"wa_id": msg.To, // O número normalizado
			},
		},
		"messages": []gin.H{
			{
				"id": msg.WhatsAppID,
			},
		},
	})
}

func (h *MetaHandler) downloadMedia(url string) ([]byte, string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", errors.New("status code diferente de 200")
	}

	// Limit to 20MB
	const maxFileSize = 20 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxFileSize))
	if err != nil {
		return nil, "", err
	}

	contentType := resp.Header.Get("Content-Type")
	return data, contentType, nil
}
