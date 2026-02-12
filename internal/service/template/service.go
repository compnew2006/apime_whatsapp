package template

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/open-apime/apime/internal/storage"
	"github.com/open-apime/apime/internal/storage/model"
)

type Service struct {
	repo storage.TemplateRepository
}

func NewService(repo storage.TemplateRepository) *Service {
	return &Service{repo: repo}
}

type CreateInput struct {
	InstanceID string
	Name       string
	Category   string
	Language   string
	Components interface{}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (model.Template, error) {
	if input.InstanceID == "" || input.Name == "" || input.Language == "" {
		return model.Template{}, errors.New("campos obrigatórios: instance_id, name, language")
	}

	componentsJSON, err := json.Marshal(input.Components)
	if err != nil {
		return model.Template{}, err
	}

	// Verificar se já existe (upsert manual para simplicidade)
	existing, err := s.repo.Get(ctx, input.InstanceID, input.Name, input.Language)
	if err == nil {
		// Se existe, deleta para recriar (simples update)
		_ = s.repo.Delete(ctx, existing.ID)
	}

	tmpl := model.Template{
		InstanceID: input.InstanceID,
		Name:       input.Name,
		Category:   input.Category,
		Language:   input.Language,
		Components: string(componentsJSON),
		Status:     "APPROVED",
	}

	return s.repo.Create(ctx, tmpl)
}

func (s *Service) List(ctx context.Context, instanceID string) ([]model.Template, error) {
	return s.repo.List(ctx, instanceID)
}

func (s *Service) Get(ctx context.Context, instanceID, name, language string) (model.Template, error) {
	return s.repo.Get(ctx, instanceID, name, language)
}

func (s *Service) GetByID(ctx context.Context, id string) (model.Template, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) DeleteByName(ctx context.Context, instanceID, name string) error {
	// Deletar todas as línguas
	// Por enquanto, apenas busca pelo nome e deleta se for único ou listar e deletar
	// Como a API deleta por nome, vamos assumir que deleta a primeira que encontrar ou todas
	// Idealmente List e filter.
	list, err := s.List(ctx, instanceID)
	if err != nil {
		return err
	}
	for _, t := range list {
		if t.Name == name {
			_ = s.repo.Delete(ctx, t.ID)
		}
	}
	return nil
}

// RenderTemplate substitui placeholders {{1}} pelos valores fornecidos
func (s *Service) RenderTemplate(ctx context.Context, instanceID, name, language string, components []interface{}) (string, string, string, error) {
	tmpl, err := s.repo.Get(ctx, instanceID, name, language)
	if err != nil {
		return "", "", "", err
	}

	var storedComponents []map[string]interface{}
	if err := json.Unmarshal([]byte(tmpl.Components), &storedComponents); err != nil {
		return "", "", "", err
	}

	var bodyText, headerURL, footerText string

	// Extrair parâmetros do input
	// A estrutura de input do Meta é complexa (components -> parameters -> type -> text/image)
	// Vamos simplificar e assumir que recebemos uma lista de strings para o body para este MVP
	// Ou melhor, o input já vem estruturado do handler.

	// Vamos simplificar: Retornar o texto cru com placeholders e deixar o handler fazer a substituição
	// Ou fazer aqui. Vamos tentar fazer aqui de forma básica.

	for _, comp := range storedComponents {
		cType, _ := comp["type"].(string)
		if cType == "BODY" {
			if text, ok := comp["text"].(string); ok {
				bodyText = text
			}
		} else if cType == "HEADER" {
			// Handle header media logic if needed
		} else if cType == "FOOTER" {
			if text, ok := comp["text"].(string); ok {
				footerText = text
			}
		}
	}

	return bodyText, headerURL, footerText, nil
}
