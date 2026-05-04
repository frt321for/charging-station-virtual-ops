package auth

import (
	"context"
	"crypto/subtle"
	"time"
)

type contextKey string

const principalContextKey contextKey = "authPrincipal"

const (
	PermissionSitesAny        = "sites:any"
	PermissionSitesRead       = "sites:read"
	PermissionSessionsRead    = "sessions:read"
	PermissionSessionsWrite   = "sessions:write"
	PermissionCommandsWrite   = "commands:write"
	PermissionSimulator       = "simulator:control"
	PermissionLoadControlRead = "load_control:read"
	PermissionLoadControl     = "load_control:write"
	PermissionBillingRead     = "billing:read"
	PermissionBillingGenerate = "billing:generate"
	PermissionFinanceReview   = "finance:review"
	PermissionMaintenanceRead = "maintenance:read"
	PermissionMaintenance     = "maintenance:write"
	PermissionAuditRead       = "audit:read"
	PermissionConfigManage    = "config:manage"
	PermissionUsersManage     = "users:manage"
	PermissionAIRead          = "ai:read"
)

// Role describes one assigned role.
type Role struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// AuthorizedSite describes one site boundary granted to a user.
type AuthorizedSite struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	AccessLevel string `json:"accessLevel"`
}

// UserSummary is returned by user administration APIs.
type UserSummary struct {
	ID          string           `json:"id"`
	Username    string           `json:"username"`
	DisplayName string           `json:"displayName"`
	Status      string           `json:"status"`
	Roles       []Role           `json:"roles"`
	Permissions []string         `json:"permissions"`
	Sites       []AuthorizedSite `json:"sites"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

// RoleDetail describes a role and its granted permissions.
type RoleDetail struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Principal is the authenticated user attached to request context.
type Principal struct {
	UserID      string           `json:"userId"`
	Username    string           `json:"username"`
	DisplayName string           `json:"displayName"`
	Status      string           `json:"status"`
	Roles       []Role           `json:"roles"`
	Permissions []string         `json:"permissions"`
	Sites       []AuthorizedSite `json:"sites"`
	ExpiresAt   time.Time        `json:"expiresAt"`
}

// HasPermission checks an exact permission code.
func (p Principal) HasPermission(permission string) bool {
	if permission == "" {
		return true
	}
	for _, granted := range p.Permissions {
		if subtle.ConstantTimeCompare([]byte(granted), []byte(permission)) == 1 {
			return true
		}
	}
	return false
}

// HasAnySiteAccess returns true when the principal is not site-limited.
func (p Principal) HasAnySiteAccess() bool {
	return p.HasPermission(PermissionSitesAny)
}

// PrimaryRoleCode returns the first assigned role code for audit records.
func (p Principal) PrimaryRoleCode() string {
	if len(p.Roles) == 0 {
		return ""
	}
	return p.Roles[0].Code
}

// CanAccessSite checks a site UUID or code against the principal boundary.
func (p Principal) CanAccessSite(siteID string, siteCode string) bool {
	if p.HasAnySiteAccess() {
		return true
	}
	for _, site := range p.Sites {
		if equalString(site.ID, siteID) || equalString(site.Code, siteCode) {
			return true
		}
	}
	return false
}

// WithPrincipal returns a context carrying the authenticated principal.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

// PrincipalFromContext returns the authenticated principal when present.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey).(Principal)
	return principal, ok
}

// CanAccessSite returns true for internal calls or allowed authenticated users.
func CanAccessSite(ctx context.Context, siteID string, siteCode string) bool {
	principal, ok := PrincipalFromContext(ctx)
	if !ok {
		return true
	}
	return principal.CanAccessSite(siteID, siteCode)
}

// LoginParams describes a user login attempt.
type LoginParams struct {
	Username  string
	Password  string
	UserAgent string
	IPAddress string
	TraceID   string
}

// LoginResult is returned after a successful login.
type LoginResult struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	User      Principal `json:"user"`
}

type userRecord struct {
	Principal
	PasswordHash string
}

type sessionRecord struct {
	TokenHash string
	ExpiresAt time.Time
	UserAgent string
	IPAddress string
}

type auditRecord struct {
	EntityType    string
	EntityID      *string
	Action        string
	ActorUserID   *string
	ActorName     string
	ActorRoleCode string
	IPAddress     string
	TraceID       string
	Payload       []byte
}

func equalString(left string, right string) bool {
	if left == "" || right == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
