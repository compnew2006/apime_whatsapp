package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/open-apime/apime/internal/storage/model"
)

type templateRepo struct {
	db *DB
}

func NewTemplateRepository(db *DB) *templateRepo {
	return &templateRepo{db: db}
}

func (r *templateRepo) Create(ctx context.Context, tmpl model.Template) (model.Template, error) {
	if tmpl.ID == "" {
		tmpl.ID = uuid.New().String()
	}
	now := time.Now()
	tmpl.CreatedAt = now
	tmpl.UpdatedAt = now
	if tmpl.Status == "" {
		tmpl.Status = "APPROVED"
	}

	query := `
		INSERT INTO templates (id, instance_id, name, category, language, components, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, instance_id, name, category, language, components, status, created_at, updated_at
	`

	err := r.db.Pool.QueryRow(ctx, query,
		tmpl.ID, tmpl.InstanceID, tmpl.Name, tmpl.Category, tmpl.Language, tmpl.Components, tmpl.Status,
		tmpl.CreatedAt, tmpl.UpdatedAt,
	).Scan(
		&tmpl.ID, &tmpl.InstanceID, &tmpl.Name, &tmpl.Category, &tmpl.Language, &tmpl.Components, &tmpl.Status,
		&tmpl.CreatedAt, &tmpl.UpdatedAt,
	)

	if err != nil {
		return model.Template{}, err
	}

	return tmpl, nil
}

func (r *templateRepo) Get(ctx context.Context, instanceID, name, language string) (model.Template, error) {
	query := `
		SELECT id, instance_id, name, category, language, components, status, created_at, updated_at
		FROM templates
		WHERE instance_id = $1 AND name = $2 AND language = $3
	`

	var tmpl model.Template

	err := r.db.Pool.QueryRow(ctx, query, instanceID, name, language).Scan(
		&tmpl.ID, &tmpl.InstanceID, &tmpl.Name, &tmpl.Category, &tmpl.Language, &tmpl.Components, &tmpl.Status,
		&tmpl.CreatedAt, &tmpl.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return model.Template{}, ErrNotFound
	}
	if err != nil {
		return model.Template{}, err
	}

	return tmpl, nil
}

func (r *templateRepo) GetByID(ctx context.Context, id string) (model.Template, error) {
	query := `
		SELECT id, instance_id, name, category, language, components, status, created_at, updated_at
		FROM templates
		WHERE id = $1
	`

	var tmpl model.Template

	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&tmpl.ID, &tmpl.InstanceID, &tmpl.Name, &tmpl.Category, &tmpl.Language, &tmpl.Components, &tmpl.Status,
		&tmpl.CreatedAt, &tmpl.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return model.Template{}, ErrNotFound
	}
	if err != nil {
		return model.Template{}, err
	}

	return tmpl, nil
}

func (r *templateRepo) List(ctx context.Context, instanceID string) ([]model.Template, error) {
	query := `
		SELECT id, instance_id, name, category, language, components, status, created_at, updated_at
		FROM templates
		WHERE instance_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []model.Template
	for rows.Next() {
		var tmpl model.Template
		if err := rows.Scan(
			&tmpl.ID, &tmpl.InstanceID, &tmpl.Name, &tmpl.Category, &tmpl.Language, &tmpl.Components, &tmpl.Status,
			&tmpl.CreatedAt, &tmpl.UpdatedAt,
		); err != nil {
			return nil, err
		}

		templates = append(templates, tmpl)
	}

	return templates, rows.Err()
}

func (r *templateRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM templates WHERE id = $1`
	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
