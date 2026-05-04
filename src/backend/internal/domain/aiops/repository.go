package aiops

import (
	"database/sql"
	"time"

	"charging-ops/backend/internal/platform/database"
)

// Repository reads bounded operational context for the AI assistant.
type Repository struct {
	database *database.Client
}

// NewRepository creates an AI operations repository.
func NewRepository(database *database.Client) *Repository {
	return &Repository{database: database}
}

func scanNullString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}

func nullableTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
