package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func TestServiceLogin(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("Password2026!"), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	tests := []struct {
		name        string
		password    string
		status      string
		wantErr     error
		wantSession bool
	}{
		{name: "creates session for valid credentials", password: "Password2026!", status: "active", wantSession: true},
		{name: "rejects invalid password", password: "wrong", status: "active", wantErr: ErrInvalidCredentials},
		{name: "rejects disabled user", password: "Password2026!", status: "disabled", wantErr: ErrDisabledUser},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newFakeRepository()
			repository.user = userRecord{
				Principal: Principal{
					UserID:      "user-1",
					Username:    "station.manager",
					DisplayName: "站长",
					Status:      test.status,
					Roles:       []Role{{Code: "station_manager", Name: "站长"}},
					Permissions: []string{PermissionSitesRead},
				},
				PasswordHash: string(hash),
			}
			service := NewService(repository)

			result, err := service.Login(context.Background(), LoginParams{
				Username: "station.manager",
				Password: test.password,
			})

			if test.wantErr == nil && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if test.wantErr != nil && !errors.Is(err, test.wantErr) {
				t.Fatalf("expected %v, got %v", test.wantErr, err)
			}
			if repository.sessionCreated != test.wantSession {
				t.Fatalf("expected sessionCreated=%v, got %v", test.wantSession, repository.sessionCreated)
			}
			if test.wantSession && (result.Token == "" || result.User.UserID != "user-1") {
				t.Fatalf("expected token and user in login result")
			}
			if test.wantSession && repository.lastTokenHash == result.Token {
				t.Fatalf("expected stored token hash, got raw token")
			}
		})
	}
}

func TestServiceRequire(t *testing.T) {
	tests := []struct {
		name       string
		token      string
		principal  Principal
		permission string
		wantStatus int
		wantCode   int
	}{
		{name: "rejects missing token", permission: PermissionSitesRead, wantStatus: http.StatusUnauthorized, wantCode: 20001},
		{
			name:       "rejects missing permission",
			token:      "valid-token",
			permission: PermissionCommandsWrite,
			principal:  Principal{UserID: "user-1", Status: "active", Permissions: []string{PermissionSitesRead}},
			wantStatus: http.StatusForbidden,
			wantCode:   20002,
		},
		{
			name:       "allows matching permission",
			token:      "valid-token",
			permission: PermissionSitesRead,
			principal:  Principal{UserID: "user-1", Status: "active", Permissions: []string{PermissionSitesRead}},
			wantStatus: http.StatusNoContent,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := newFakeRepository()
			if test.token != "" {
				repository.sessions[hashToken(test.token)] = test.principal
			}
			service := NewService(repository)
			handler := service.Require(test.permission, func(w http.ResponseWriter, r *http.Request) {
				if _, ok := PrincipalFromContext(r.Context()); !ok {
					t.Fatalf("expected principal in context")
				}
				w.WriteHeader(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if test.token != "" {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			response := httptest.NewRecorder()

			handler(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("expected status %d, got %d", test.wantStatus, response.Code)
			}
			if test.wantCode != 0 {
				var envelope struct {
					Code int `json:"code"`
				}
				if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
					t.Fatalf("decode error envelope: %v", err)
				}
				if envelope.Code != test.wantCode {
					t.Fatalf("expected code %d, got %d", test.wantCode, envelope.Code)
				}
			}
		})
	}
}

type fakeRepository struct {
	user           userRecord
	sessions       map[string]Principal
	sessionCreated bool
	lastTokenHash  string
	revoked        bool
	touched        bool
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{sessions: make(map[string]Principal)}
}

func (f *fakeRepository) FindUserByUsername(ctx context.Context, username string) (userRecord, error) {
	if f.user.Username == "" || username != f.user.Username {
		return userRecord{}, ErrInvalidCredentials
	}
	return f.user, nil
}

func (f *fakeRepository) CreateSession(ctx context.Context, userID string, session sessionRecord) error {
	f.sessionCreated = true
	f.lastTokenHash = session.TokenHash
	principal := f.user.Principal
	principal.ExpiresAt = session.ExpiresAt
	f.sessions[session.TokenHash] = principal
	return nil
}

func (f *fakeRepository) FindSessionByHash(ctx context.Context, tokenHash string) (Principal, error) {
	principal, ok := f.sessions[tokenHash]
	if !ok {
		return Principal{}, ErrUnauthorized
	}
	if principal.ExpiresAt.IsZero() {
		principal.ExpiresAt = time.Now().UTC().Add(time.Hour)
	}
	return principal, nil
}

func (f *fakeRepository) RevokeSession(ctx context.Context, tokenHash string) error {
	f.revoked = true
	delete(f.sessions, tokenHash)
	return nil
}

func (f *fakeRepository) TouchSession(ctx context.Context, tokenHash string) error {
	f.touched = true
	return nil
}

func (f *fakeRepository) InsertAudit(ctx context.Context, record auditRecord) error {
	return nil
}

func (f *fakeRepository) ListUsers(ctx context.Context) ([]UserSummary, error) {
	return []UserSummary{}, nil
}

func (f *fakeRepository) ListRoles(ctx context.Context) ([]RoleDetail, error) {
	return []RoleDetail{}, nil
}
