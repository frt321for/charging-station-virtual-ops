package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"charging-ops/backend/internal/common/response"
)

type tokenContextKeyType string

const tokenContextKey tokenContextKeyType = "authToken"

// TokenFromContext returns the current bearer token stored by auth middleware.
func TokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value(tokenContextKey).(string)
	return token, ok && token != ""
}

// Handler serves authentication APIs.
type Handler struct {
	service *Service
}

// NewHandler creates an auth handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Login validates credentials and returns the authenticated user.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if err := decodeJSON(r, &request); err != nil {
		response.Error(w, r, http.StatusBadRequest, 10002, "请求格式错误", "invalid json body")
		return
	}

	result, err := h.service.Login(r.Context(), LoginParams{
		Username:  request.Username,
		Password:  request.Password,
		UserAgent: r.UserAgent(),
		IPAddress: requestIP(r),
		TraceID:   response.TraceID(r.Context()),
	})
	if err != nil {
		writeAuthError(w, r, err)
		return
	}

	setSessionCookie(w, result.Token, result.ExpiresAt)
	response.OK(w, r, result)
}

// Me returns the current authenticated user.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		writeAuthError(w, r, ErrUnauthorized)
		return
	}
	response.OK(w, r, principal)
}

// Logout revokes the current authenticated session.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token, _ := r.Context().Value(tokenContextKey).(string)
	if token == "" {
		token = tokenFromRequest(r)
	}
	if err := h.service.Logout(r.Context(), token, requestIP(r), response.TraceID(r.Context())); err != nil {
		writeAuthError(w, r, err)
		return
	}
	clearSessionCookie(w)
	response.OK(w, r, map[string]any{"status": "logged_out"})
}

// ListUsers returns users and their authorization boundaries.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		writeAuthError(w, r, err)
		return
	}
	response.OK(w, r, map[string]any{"list": users})
}

// ListRoles returns roles and their permissions.
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.service.ListRoles(r.Context())
	if err != nil {
		writeAuthError(w, r, err)
		return
	}
	response.OK(w, r, map[string]any{"list": roles})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrUnauthorized), errors.Is(err, ErrInvalidCredentials):
		response.Error(w, r, http.StatusUnauthorized, 20001, "未登录或登录已失效", "unauthorized")
	case errors.Is(err, ErrDisabledUser):
		response.Error(w, r, http.StatusForbidden, 20002, "用户已停用", "user disabled")
	case errors.Is(err, ErrForbidden):
		response.Error(w, r, http.StatusForbidden, 20002, "权限不足", "permission denied")
	default:
		response.Error(w, r, http.StatusInternalServerError, 50002, "认证处理失败", "auth operation failed")
	}
}
