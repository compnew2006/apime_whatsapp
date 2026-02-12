package handler

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/open-apime/apime/internal/pkg/response"
	messageSvc "github.com/open-apime/apime/internal/service/message"
)

type MetaHandler struct {
	service *messageSvc.Service
}

func NewMetaHandler(service *messageSvc.Service) *MetaHandler {
	return &MetaHandler{service: service}
}

func (h *MetaHandler) Register(r *gin.RouterGroup) {
	r.POST("/meta/:id/messages", h.sendMessage)
}

// MetaRequest representa a estrutura de envio de mensagem da Cloud API
type MetaRequest struct {
	MessagingProduct string          `json:"messaging_product"`
	To               string          `json:"to"`
	Type             string          `json:"type"`
	Text             *MetaText       `json:"text,omitempty"`
	Image            *MetaMedia      `json:"image,omitempty"`
	Video            *MetaMedia      `json:"video,omitempty"`
	Audio            *MetaMedia      `json:"audio,omitempty"`
	Document         *MetaMedia      `json:"document,omitempty"`
	Sticker          *MetaMedia      `json:"sticker,omitempty"`
	Location         *MetaLocation   `json:"location,omitempty"`
	Contacts         []*MetaContact  `json:"contacts,omitempty"`
	Interactive      *MetaInteractive `json:"interactive,omitempty"`
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
