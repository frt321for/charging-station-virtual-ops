package health

import (
	"context"
	"net/http"
	"time"

	"charging-ops/backend/internal/common/response"
)

// Checker verifies a dependency is reachable.
type Checker interface {
	Check(ctx context.Context) error
}

// Dependencies lists infrastructure clients used by health checks.
type Dependencies struct {
	Database Checker
	Cache    Checker
	Events   Checker
}

// Handler serves health endpoints.
type Handler struct {
	dependencies Dependencies
}

// NewHandler creates a health handler.
func NewHandler(dependencies Dependencies) *Handler {
	return &Handler{dependencies: dependencies}
}

// Check returns dependency health for local and browser verification.
func (h *Handler) Check(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	dependencies := map[string]string{
		"database": h.check(ctx, h.dependencies.Database),
		"cache":    h.check(ctx, h.dependencies.Cache),
		"events":   h.check(ctx, h.dependencies.Events),
	}

	status := "ok"
	for _, value := range dependencies {
		if value != "ok" {
			status = "degraded"
			break
		}
	}

	response.OK(w, r, map[string]any{
		"status":       status,
		"dependencies": dependencies,
	})
}

func (h *Handler) check(ctx context.Context, checker Checker) string {
	if checker == nil {
		return "missing"
	}
	if err := checker.Check(ctx); err != nil {
		return "error"
	}
	return "ok"
}
