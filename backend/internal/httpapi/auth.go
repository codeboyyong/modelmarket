package httpapi

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

type authContextKey struct{}

type authenticatedUser struct {
	ID       string
	Email    string
	UserType string
}

func currentUser(ctx context.Context) (authenticatedUser, bool) {
	user, ok := ctx.Value(authContextKey{}).(authenticatedUser)
	return user, ok
}

func sessionTokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (a *App) createSession(ctx context.Context, userID string) (string, time.Time, error) {
	raw := "sess_" + randomHex(32)
	expiresAt := time.Now().UTC().Add(24 * time.Hour)
	_, err := a.DB.ExecContext(ctx, `insert into sys_sessions(id, user_id, token_hash, expires_at) values($1,$2,$3,$4)`, "session_"+randomHex(12), userID, sessionTokenHash(raw), expiresAt)
	return raw, expiresAt, err
}

func (a *App) authenticated(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.Config.DevMode || isPublicRoute(r) {
			next.ServeHTTP(w, r)
			return
		}
		raw := bearerToken(r)
		if raw == "" {
			writeJSON(w, http.StatusUnauthorized, response{"error": "authentication_required"})
			return
		}
		var user authenticatedUser
		err := a.DB.QueryRowContext(r.Context(), `select u.id, u.email, u.user_type from sys_sessions s join sys_users u on u.id = s.user_id where s.token_hash = $1 and s.expires_at > current_timestamp`, sessionTokenHash(raw)).Scan(&user.ID, &user.Email, &user.UserType)
		if err != nil {
			if err == sql.ErrNoRows {
				writeJSON(w, http.StatusUnauthorized, response{"error": "invalid_or_expired_session"})
				return
			}
			writeJSON(w, http.StatusInternalServerError, response{"error": "session_lookup_failed"})
			return
		}
		r = r.WithContext(context.WithValue(r.Context(), authContextKey{}, user))
		if err := a.authorizeTenantRequest(r, user); err != nil {
			writeJSON(w, http.StatusForbidden, response{"error": "access_denied"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) authorizeTenantRequest(r *http.Request, user authenticatedUser) error {
	if strings.HasPrefix(r.URL.Path, "/api/v1/admin/") && user.UserType != "sys_admin" {
		return sql.ErrNoRows
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	organizationID := strings.TrimSpace(r.URL.Query().Get("organization_id"))
	requestedUserID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if r.Body != nil && r.Method != http.MethodGet && !strings.HasPrefix(r.URL.Path, "/api/v1/mock-s3/") {
		raw, err := io.ReadAll(io.LimitReader(r.Body, 2<<20))
		if err != nil {
			return err
		}
		r.Body.Close()
		r.Body = io.NopCloser(strings.NewReader(string(raw)))
		var body struct {
			ProjectID      string `json:"project_id"`
			OrganizationID string `json:"organization_id"`
			UserID         string `json:"user_id"`
		}
		if json.Unmarshal(raw, &body) == nil {
			if projectID == "" {
				projectID = strings.TrimSpace(body.ProjectID)
			}
			if organizationID == "" {
				organizationID = strings.TrimSpace(body.OrganizationID)
			}
			if requestedUserID == "" {
				requestedUserID = strings.TrimSpace(body.UserID)
			}
		}
	}
	if requestedUserID != "" && requestedUserID != user.ID && user.UserType != "sys_admin" {
		return sql.ErrNoRows
	}
	if projectID != "" {
		var found int
		return a.DB.QueryRowContext(r.Context(), `select 1 from user_projects p join sys_memberships m on m.organization_id = p.organization_id where p.id = $1 and m.user_id = $2 and p.status = 'active' limit 1`, projectID, user.ID).Scan(&found)
	}
	if organizationID != "" {
		var role string
		if err := a.DB.QueryRowContext(r.Context(), `select role from sys_memberships where organization_id = $1 and user_id = $2 limit 1`, organizationID, user.ID).Scan(&role); err != nil {
			return err
		}
		if r.Method != http.MethodGet && (strings.HasPrefix(r.URL.Path, "/api/v1/organization/members") || strings.HasPrefix(r.URL.Path, "/api/v1/organization/invitations")) {
			if role != "owner" && role != "admin" {
				return sql.ErrNoRows
			}
		}
		return nil
	}
	path := r.URL.Path
	if strings.HasPrefix(path, "/api/v1/conversations/") {
		id := strings.TrimPrefix(path, "/api/v1/conversations/")
		id = strings.TrimSuffix(id, "/export")
		var found int
		return a.DB.QueryRowContext(r.Context(), `select 1 from user_conversations c join user_projects p on p.id = c.project_id join sys_memberships m on m.organization_id = p.organization_id where c.id = $1 and m.user_id = $2 limit 1`, id, user.ID).Scan(&found)
	}
	if strings.HasPrefix(path, "/api/v1/assets/") && path != "/api/v1/assets/upload-intent" {
		id := strings.TrimPrefix(path, "/api/v1/assets/")
		var found int
		return a.DB.QueryRowContext(r.Context(), `select 1 from user_workbench_assets a join user_projects p on p.id = a.project_id join sys_memberships m on m.organization_id = p.organization_id where a.id = $1 and m.user_id = $2 limit 1`, id, user.ID).Scan(&found)
	}
	if strings.HasPrefix(path, "/api/v1/api-keys/") {
		id := strings.TrimPrefix(path, "/api/v1/api-keys/")
		var found int
		return a.DB.QueryRowContext(r.Context(), `select 1 from user_api_keys k join user_projects p on p.id = k.project_id join sys_memberships m on m.organization_id = p.organization_id where k.id = $1 and m.user_id = $2 limit 1`, id, user.ID).Scan(&found)
	}
	return nil
}

func bearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(header), "bearer ") {
		return ""
	}
	return strings.TrimSpace(header[7:])
}

func isPublicRoute(r *http.Request) bool {
	path := r.URL.Path
	if path == "/healthz" || path == "/readyz" || path == "/api/v1/models" || path == "/api/v1/pricing" {
		return true
	}
	if strings.HasPrefix(path, "/api/v1/auth/") || path == "/api/v1/payments/stripe/webhook" || path == "/api/v1/chat/completions" {
		return true
	}
	return false
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	raw := bearerToken(r)
	if raw != "" {
		_, _ = a.DB.ExecContext(r.Context(), `delete from sys_sessions where token_hash = $1`, sessionTokenHash(raw))
	}
	writeJSON(w, http.StatusOK, response{"status": "logged_out"})
}
