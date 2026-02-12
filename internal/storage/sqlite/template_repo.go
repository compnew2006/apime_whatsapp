package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"

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
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := r.db.Conn.ExecContext(ctx, query,
		tmpl.ID, tmpl.InstanceID, tmpl.Name, tmpl.Category, tmpl.Language, tmpl.Components, tmpl.Status,
		tmpl.CreatedAt.Format(time.RFC3339), tmpl.UpdatedAt.Format(time.RFC3339),
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
		WHERE instance_id = ? AND name = ? AND language = ?
	`

	var tmpl model.Template
	var createdAt, updatedAt string

	err := r.db.Conn.QueryRowContext(ctx, query, instanceID, name, language).Scan(
		&tmpl.ID, &tmpl.InstanceID, &tmpl.Name, &tmpl.Category, &tmpl.Language, &tmpl.Components, &tmpl.Status,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return model.Template{}, mapError(err)
	}

	tmpl.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	tmpl.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return tmpl, nil
}

func (r *templateRepo) GetByID(ctx context.Context, id string) (model.Template, error) {
	query := `
		SELECT id, instance_id, name, category, language, components, status, created_at, updated_at
		FROM templates
		WHERE id = ?
	`

	var tmpl model.Template
	var createdAt, updatedAt string

	err := r.db.Conn.QueryRowContext(ctx, query, id).Scan(
		&tmpl.ID, &tmpl.InstanceID, &tmpl.Name, &tmpl.Category, &tmpl.Language, &tmpl.Components, &tmpl.Status,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return model.Template{}, mapError(err)
	}

	tmpl.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	tmpl.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return tmpl, nil
}

func (r *templateRepo) List(ctx context.Context, instanceID string) ([]model.Template, error) {
	query := `
		SELECT id, instance_id, name, category, language, components, status, created_at, updated_at
		FROM templates
		WHERE instance_id = ?
		ORDER BY created_at DESC
	`

	rows, err := r.db.Conn.QueryContext(ctx, query, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var templates []model.Template
	for rows.Next() {
		var tmpl model.Template
		var createdAt, updatedAt string

		if err := rows.Scan(
			&tmpl.ID, &tmpl.InstanceID, &tmpl.Name, &tmpl.Category, &tmpl.Language, &tmpl.Components, &tmpl.Status,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}

		tmpl.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		tmpl.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		templates = append(templates, tmpl)
	}

	return templates, rows.Err()
}

func (r *templateRepo) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM templates WHERE id = ?`
	result, err := r.db.Conn.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return mapError(sql.ErrNoRows)
	}
	return nil
}
