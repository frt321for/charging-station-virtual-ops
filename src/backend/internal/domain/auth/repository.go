package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"charging-ops/backend/internal/platform/database"

	"github.com/jackc/pgx/v5"
)

// Repository persists users, sessions, permissions, and auth audit records.
type Repository struct {
	database *database.Client
}

// NewRepository creates an auth repository.
func NewRepository(database *database.Client) *Repository {
	return &Repository{database: database}
}

// FindUserByUsername returns a user plus roles, permissions, and site grants.
func (r *Repository) FindUserByUsername(ctx context.Context, username string) (userRecord, error) {
	var record userRecord
	err := r.database.Pool().QueryRow(ctx, `
		SELECT id::text, username, display_name, password_hash, status
		FROM users
		WHERE deleted_at IS NULL AND lower(username) = lower($1)
	`, username).Scan(
		&record.UserID,
		&record.Username,
		&record.DisplayName,
		&record.PasswordHash,
		&record.Status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return userRecord{}, ErrInvalidCredentials
	}
	if err != nil {
		return userRecord{}, fmt.Errorf("query user credentials: %w", err)
	}
	if err := r.hydratePrincipal(ctx, &record.Principal); err != nil {
		return userRecord{}, err
	}
	return record, nil
}

// CreateSession stores a hashed session token.
func (r *Repository) CreateSession(ctx context.Context, userID string, session sessionRecord) error {
	_, err := r.database.Pool().Exec(ctx, `
		INSERT INTO auth_sessions (user_id, token_hash, expires_at, user_agent, ip_address)
		VALUES ($1, $2, $3, $4, $5)
	`, userID, session.TokenHash, session.ExpiresAt, session.UserAgent, session.IPAddress)
	if err != nil {
		return fmt.Errorf("insert auth session: %w", err)
	}
	return nil
}

// FindSessionByHash returns an active authenticated principal.
func (r *Repository) FindSessionByHash(ctx context.Context, tokenHash string) (Principal, error) {
	var principal Principal
	err := r.database.Pool().QueryRow(ctx, `
		SELECT u.id::text, u.username, u.display_name, u.status, s.expires_at
		FROM auth_sessions s
		INNER JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > now()
		  AND u.deleted_at IS NULL
	`, tokenHash).Scan(
		&principal.UserID,
		&principal.Username,
		&principal.DisplayName,
		&principal.Status,
		&principal.ExpiresAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Principal{}, ErrUnauthorized
	}
	if err != nil {
		return Principal{}, fmt.Errorf("query auth session: %w", err)
	}
	if principal.Status != "active" {
		return Principal{}, ErrDisabledUser
	}
	if err := r.hydratePrincipal(ctx, &principal); err != nil {
		return Principal{}, err
	}
	return principal, nil
}

// RevokeSession marks one session as revoked.
func (r *Repository) RevokeSession(ctx context.Context, tokenHash string) error {
	_, err := r.database.Pool().Exec(ctx, `
		UPDATE auth_sessions
		SET revoked_at = COALESCE(revoked_at, now())
		WHERE token_hash = $1
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke auth session: %w", err)
	}
	return nil
}

// TouchSession records that a token was used.
func (r *Repository) TouchSession(ctx context.Context, tokenHash string) error {
	_, err := r.database.Pool().Exec(ctx, `
		UPDATE auth_sessions
		SET last_seen_at = now()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("touch auth session: %w", err)
	}
	return nil
}

// InsertAudit records one authentication or permission-sensitive action.
func (r *Repository) InsertAudit(ctx context.Context, record auditRecord) error {
	_, err := r.database.Pool().Exec(ctx, `
		INSERT INTO audit_logs (
			entity_type, entity_id, action, actor_user_id, actor_name,
			actor_role_code, ip_address, trace_id, payload
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		record.EntityType,
		uuidValue(record.EntityID),
		record.Action,
		uuidValue(record.ActorUserID),
		record.ActorName,
		record.ActorRoleCode,
		record.IPAddress,
		record.TraceID,
		record.Payload,
	)
	if err != nil {
		return fmt.Errorf("insert auth audit: %w", err)
	}
	return nil
}

// ListUsers returns users with their assigned roles and site boundaries.
func (r *Repository) ListUsers(ctx context.Context) ([]UserSummary, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT id::text, username, display_name, status, created_at, updated_at
		FROM users
		WHERE deleted_at IS NULL
		ORDER BY username
	`)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	users := make([]UserSummary, 0)
	for rows.Next() {
		var user UserSummary
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.DisplayName,
			&user.Status,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		principal := Principal{UserID: user.ID}
		if err := r.hydratePrincipal(ctx, &principal); err != nil {
			return nil, err
		}
		user.Roles = principal.Roles
		user.Permissions = principal.Permissions
		user.Sites = principal.Sites
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return users, nil
}

// ListRoles returns active and disabled roles with permission codes.
func (r *Repository) ListRoles(ctx context.Context) ([]RoleDetail, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT id::text, code, name, description, status, created_at, updated_at
		FROM roles
		WHERE deleted_at IS NULL
		ORDER BY code
	`)
	if err != nil {
		return nil, fmt.Errorf("query roles: %w", err)
	}
	defer rows.Close()

	roles := make([]RoleDetail, 0)
	for rows.Next() {
		var role RoleDetail
		if err := rows.Scan(
			&role.ID,
			&role.Code,
			&role.Name,
			&role.Description,
			&role.Status,
			&role.CreatedAt,
			&role.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		permissions, err := r.listRolePermissions(ctx, role.ID)
		if err != nil {
			return nil, err
		}
		role.Permissions = permissions
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate roles: %w", err)
	}
	return roles, nil
}

func (r *Repository) hydratePrincipal(ctx context.Context, principal *Principal) error {
	roles, err := r.listRoles(ctx, principal.UserID)
	if err != nil {
		return err
	}
	permissions, err := r.listPermissions(ctx, principal.UserID)
	if err != nil {
		return err
	}
	sites, err := r.listSites(ctx, principal.UserID)
	if err != nil {
		return err
	}
	principal.Roles = roles
	principal.Permissions = permissions
	principal.Sites = sites
	return nil
}

func (r *Repository) listRoles(ctx context.Context, userID string) ([]Role, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT r.code, r.name
		FROM user_roles ur
		INNER JOIN roles r ON r.id = ur.role_id
		WHERE ur.user_id::text = $1
		  AND r.deleted_at IS NULL
		  AND r.status = 'active'
		ORDER BY r.code
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query user roles: %w", err)
	}
	defer rows.Close()

	roles := make([]Role, 0)
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.Code, &role.Name); err != nil {
			return nil, fmt.Errorf("scan user role: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user roles: %w", err)
	}
	return roles, nil
}

func (r *Repository) listPermissions(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT DISTINCT p.code
		FROM user_roles ur
		INNER JOIN roles r ON r.id = ur.role_id
		INNER JOIN role_permissions rp ON rp.role_id = r.id
		INNER JOIN permissions p ON p.id = rp.permission_id
		WHERE ur.user_id::text = $1
		  AND r.deleted_at IS NULL
		  AND r.status = 'active'
		ORDER BY p.code
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query user permissions: %w", err)
	}
	defer rows.Close()

	permissions := make([]string, 0)
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, fmt.Errorf("scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate permissions: %w", err)
	}
	return permissions, nil
}

func (r *Repository) listSites(ctx context.Context, userID string) ([]AuthorizedSite, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT s.id::text, s.code, s.name, usa.access_level
		FROM user_site_access usa
		INNER JOIN sites s ON s.id = usa.site_id
		WHERE usa.user_id::text = $1
		  AND s.deleted_at IS NULL
		ORDER BY s.code
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("query user site access: %w", err)
	}
	defer rows.Close()

	sites := make([]AuthorizedSite, 0)
	for rows.Next() {
		var site AuthorizedSite
		if err := rows.Scan(&site.ID, &site.Code, &site.Name, &site.AccessLevel); err != nil {
			return nil, fmt.Errorf("scan user site access: %w", err)
		}
		sites = append(sites, site)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate user site access: %w", err)
	}
	return sites, nil
}

func (r *Repository) listRolePermissions(ctx context.Context, roleID string) ([]string, error) {
	rows, err := r.database.Pool().Query(ctx, `
		SELECT p.code
		FROM role_permissions rp
		INNER JOIN permissions p ON p.id = rp.permission_id
		WHERE rp.role_id::text = $1
		ORDER BY p.code
	`, roleID)
	if err != nil {
		return nil, fmt.Errorf("query role permissions: %w", err)
	}
	defer rows.Close()

	permissions := make([]string, 0)
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, fmt.Errorf("scan role permission: %w", err)
		}
		permissions = append(permissions, permission)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate role permissions: %w", err)
	}
	return permissions, nil
}

func uuidValue(value *string) *string {
	if value == nil || *value == "" {
		return nil
	}
	return value
}

func nullableStringValue(value sql.NullString) *string {
	if !value.Valid || value.String == "" {
		return nil
	}
	return &value.String
}
