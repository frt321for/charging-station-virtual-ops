package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "charging_ops_session"
	defaultTokenTTL   = 8 * time.Hour
)

type repository interface {
	FindUserByUsername(ctx context.Context, username string) (userRecord, error)
	CreateSession(ctx context.Context, userID string, session sessionRecord) error
	FindSessionByHash(ctx context.Context, tokenHash string) (Principal, error)
	RevokeSession(ctx context.Context, tokenHash string) error
	TouchSession(ctx context.Context, tokenHash string) error
	InsertAudit(ctx context.Context, record auditRecord) error
	ListUsers(ctx context.Context) ([]UserSummary, error)
	ListRoles(ctx context.Context) ([]RoleDetail, error)
}

// Service coordinates login, session validation, and permission checks.
type Service struct {
	repository repository
	tokenTTL   time.Duration
}

// NewService creates an auth service.
func NewService(repository repository) *Service {
	return &Service{repository: repository, tokenTTL: defaultTokenTTL}
}

// Login validates credentials and creates a short-lived session token.
func (s *Service) Login(ctx context.Context, params LoginParams) (LoginResult, error) {
	params = normalizeLoginParams(params)
	record, err := s.repository.FindUserByUsername(ctx, params.Username)
	if err != nil {
		return LoginResult{}, err
	}
	if record.Status != "active" {
		return LoginResult{}, ErrDisabledUser
	}
	if err := bcrypt.CompareHashAndPassword([]byte(record.PasswordHash), []byte(params.Password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, err := newToken()
	if err != nil {
		return LoginResult{}, err
	}
	expiresAt := time.Now().UTC().Add(s.tokenTTL)
	if err := s.repository.CreateSession(ctx, record.UserID, sessionRecord{
		TokenHash: hashToken(token),
		ExpiresAt: expiresAt,
		UserAgent: params.UserAgent,
		IPAddress: params.IPAddress,
	}); err != nil {
		return LoginResult{}, err
	}

	principal := record.Principal
	principal.ExpiresAt = expiresAt
	if err := s.audit(ctx, principal, "login", params.IPAddress, params.TraceID, json.RawMessage(`{}`)); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, ExpiresAt: expiresAt, User: principal}, nil
}

// AuthenticateToken resolves an API token into a principal.
func (s *Service) AuthenticateToken(ctx context.Context, token string) (Principal, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Principal{}, ErrUnauthorized
	}
	tokenHash := hashToken(token)
	principal, err := s.repository.FindSessionByHash(ctx, tokenHash)
	if err != nil {
		return Principal{}, err
	}
	if err := s.repository.TouchSession(ctx, tokenHash); err != nil {
		return Principal{}, err
	}
	return principal, nil
}

// Logout revokes the current token.
func (s *Service) Logout(ctx context.Context, token string, ipAddress string, traceID string) error {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return ErrUnauthorized
	}
	if err := s.repository.RevokeSession(ctx, hashToken(token)); err != nil {
		return err
	}
	return s.audit(ctx, principal, "logout", ipAddress, traceID, json.RawMessage(`{}`))
}

// ListUsers returns user authorization records.
func (s *Service) ListUsers(ctx context.Context) ([]UserSummary, error) {
	return s.repository.ListUsers(ctx)
}

// ListRoles returns role permission records.
func (s *Service) ListRoles(ctx context.Context) ([]RoleDetail, error) {
	return s.repository.ListRoles(ctx)
}

// Require protects a route with authentication and an optional permission.
func (s *Service) Require(permission string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, token, ok := s.authenticateRequest(w, r)
		if !ok {
			return
		}
		if !principal.HasPermission(permission) {
			s.writeAuthError(w, r, ErrForbidden)
			return
		}
		ctx := WithPrincipal(context.WithValue(r.Context(), tokenContextKey, token), principal)
		next(w, r.WithContext(ctx))
	}
}

// RequireSite protects a route and checks a route site's UUID or code.
func (s *Service) RequireSite(permission string, siteParam string, next http.HandlerFunc) http.HandlerFunc {
	return s.Require(permission, func(w http.ResponseWriter, r *http.Request) {
		principal, _ := PrincipalFromContext(r.Context())
		siteValue := r.PathValue(siteParam)
		if siteValue != "" && !principal.CanAccessSite(siteValue, siteValue) {
			s.writeAuthError(w, r, ErrForbidden)
			return
		}
		next(w, r)
	})
}

func (s *Service) authenticateRequest(w http.ResponseWriter, r *http.Request) (Principal, string, bool) {
	token := tokenFromRequest(r)
	principal, err := s.AuthenticateToken(r.Context(), token)
	if err != nil {
		s.writeAuthError(w, r, err)
		return Principal{}, "", false
	}
	return principal, token, true
}

func (s *Service) writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	writeAuthError(w, r, err)
}

func (s *Service) audit(
	ctx context.Context,
	principal Principal,
	action string,
	ipAddress string,
	traceID string,
	payload json.RawMessage,
) error {
	userID := principal.UserID
	return s.repository.InsertAudit(ctx, auditRecord{
		EntityType:    "auth_session",
		Action:        action,
		ActorUserID:   &userID,
		ActorName:     principal.DisplayName,
		ActorRoleCode: principal.PrimaryRoleCode(),
		IPAddress:     ipAddress,
		TraceID:       traceID,
		Payload:       payload,
	})
}

func normalizeLoginParams(params LoginParams) LoginParams {
	params.Username = strings.TrimSpace(params.Username)
	params.UserAgent = strings.TrimSpace(params.UserAgent)
	params.IPAddress = strings.TrimSpace(params.IPAddress)
	params.TraceID = strings.TrimSpace(params.TraceID)
	return params
}

func tokenFromRequest(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return strings.TrimSpace(header[7:])
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		return cookie.Value
	}
	return ""
}

func setSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func requestIP(r *http.Request) string {
	forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For"))
	if forwarded != "" {
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func newToken() (string, error) {
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", fmt.Errorf("generate auth token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes[:]), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
