package model

import "time"

type Template struct {
	ID         string    `json:"id"`
	InstanceID string    `json:"instanceId"`
	Name       string    `json:"name"`
	Category   string    `json:"category"`
	Language   string    `json:"language"`
	Components string    `json:"components"` // Stored as JSON string
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}
