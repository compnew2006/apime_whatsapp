package storage

import (
	"context"

	"github.com/open-apime/apime/internal/storage/model"
)

type TemplateRepository interface {
	Create(ctx context.Context, tmpl model.Template) (model.Template, error)
	Get(ctx context.Context, instanceID, name, language string) (model.Template, error)
	GetByID(ctx context.Context, id string) (model.Template, error)
	List(ctx context.Context, instanceID string) ([]model.Template, error)
	Delete(ctx context.Context, id string) error
}
