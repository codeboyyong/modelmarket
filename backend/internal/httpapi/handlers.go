package httpapi

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type response map[string]any

func (a *App) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, response{"status": "ok", "service": "model-market-backend"})
}

func (a *App) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.DB.PingContext(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, response{"status": "not_ready", "database": err.Error()})
		return
	}
	if a.Redis != nil {
		if err := a.Redis.Ping(ctx); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, response{"status": "not_ready", "redis": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, response{"status": "ready"})
}

func (a *App) devSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	counts := map[string]int64{}
	for _, table := range []string{"users", "organizations", "projects", "providers", "models", "model_profiles", "wallets", "conversations", "messages", "usage_events"} {
		var count int64
		if err := a.DB.QueryRowContext(ctx, fmt.Sprintf("select count(*) from %s", table)).Scan(&count); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		counts[table] = count
	}
	writeJSON(w, http.StatusOK, response{"dev_mode": a.Config.DevMode, "counts": counts})
}

func (a *App) devLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	if req.Email == "" {
		req.Email = "admin@example.com"
	}
	login, err := a.loginByEmail(r.Context(), req.Email)
	if err != nil {
		if a.Config.DevMode {
			writeJSON(w, http.StatusOK, devAuthResponse(req.Email, "Admin User"))
			return
		}
		writeJSON(w, http.StatusUnauthorized, response{"error": "dev_user_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, login)
}

func (a *App) passwordLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_credentials"})
		return
	}
	login, err := a.loginByIdentity(r.Context(), req.Username)
	if err != nil {
		if a.Config.DevMode {
			email := req.Username
			name := strings.Split(req.Username, "@")[0]
			if !strings.Contains(email, "@") {
				email = "admin@example.com"
				name = req.Username
			}
			writeJSON(w, http.StatusOK, devAuthResponse(email, name))
			return
		}
		writeJSON(w, http.StatusUnauthorized, response{"error": "invalid_credentials"})
		return
	}
	writeJSON(w, http.StatusOK, login)
}

func (a *App) devSocialLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider != "google" && provider != "github" && provider != "facebook" {
		writeJSON(w, http.StatusBadRequest, response{"error": "unsupported_provider"})
		return
	}
	email := map[string]string{
		"google":   "admin@example.com",
		"github":   "admin@example.com",
		"facebook": "developer@example.com",
	}[provider]
	login, err := a.loginByEmail(r.Context(), email)
	if err != nil {
		if a.Config.DevMode {
			login = devAuthResponse(email, map[string]string{
				"google":   "Admin User",
				"github":   "Admin User",
				"facebook": "Developer User",
			}[provider])
			login["provider"] = provider
			writeJSON(w, http.StatusOK, login)
			return
		}
		writeJSON(w, http.StatusUnauthorized, response{"error": "dev_user_not_found"})
		return
	}
	login["provider"] = provider
	writeJSON(w, http.StatusOK, login)
}

func (a *App) signup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	if req.Email == "" || req.Password == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_signup_fields"})
		return
	}
	if req.Name == "" {
		req.Name = strings.Split(req.Email, "@")[0]
	}
	userID := "user_" + randomHex(8)
	membershipID := "membership_" + randomHex(8)
	_, err := a.DB.ExecContext(r.Context(), `
		insert into users(id, email, name, avatar_url, status)
		values($1, $2, $3, null, 'active')`, userID, req.Email, req.Name)
	if err != nil {
		if a.Config.DevMode {
			writeJSON(w, http.StatusCreated, devAuthResponse(req.Email, req.Name))
			return
		}
		writeJSON(w, http.StatusConflict, response{"error": "user_exists_or_invalid"})
		return
	}
	_, err = a.DB.ExecContext(r.Context(), `
		insert into memberships(id, user_id, organization_id, role)
		values($1, $2, 'org-demo', 'developer')`, membershipID, userID)
	if err != nil {
		if a.Config.DevMode {
			writeJSON(w, http.StatusCreated, devAuthResponse(req.Email, req.Name))
			return
		}
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	login, err := a.loginByEmail(r.Context(), req.Email)
	if err != nil {
		if a.Config.DevMode {
			writeJSON(w, http.StatusCreated, devAuthResponse(req.Email, req.Name))
			return
		}
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, login)
}

func (a *App) loginByIdentity(ctx context.Context, identity string) (response, error) {
	var email string
	err := a.DB.QueryRowContext(ctx, `
		select email
		from users
		where lower(email) = lower($1) or lower(name) = lower($1)
		limit 1`, identity).Scan(&email)
	if err != nil {
		return nil, err
	}
	return a.loginByEmail(ctx, email)
}

func (a *App) loginByEmail(ctx context.Context, email string) (response, error) {
	var userID, orgID, projectID, name string
	err := a.DB.QueryRowContext(ctx, `
		select u.id, u.name, o.id, p.id
		from users u
		join memberships m on m.user_id = u.id
		join organizations o on o.id = m.organization_id
		join projects p on p.organization_id = o.id
		where u.email = $1
		order by p.created_at asc
		limit 1`, email).Scan(&userID, &name, &orgID, &projectID)
	if err != nil {
		return nil, err
	}
	return response{
		"user":            response{"id": userID, "email": email, "name": name},
		"organization_id": orgID,
		"project_id":      projectID,
		"session":         response{"access_token": "dev_" + randomHex(16), "token_type": "Bearer"},
	}, nil
}

func devAuthResponse(email, name string) response {
	if strings.TrimSpace(name) == "" {
		name = "Dev User"
	}
	userID := "user-dev-fallback"
	if strings.EqualFold(email, "admin@example.com") {
		userID = "user-admin"
	}
	return response{
		"user":            response{"id": userID, "email": email, "name": name},
		"organization_id": "org-demo",
		"project_id":      "project-demo",
		"session":         response{"access_token": "dev_" + randomHex(16), "token_type": "Bearer"},
		"dev_fallback":    true,
	}
}

func (a *App) models(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.QueryContext(r.Context(), `
		select m.id, m.slug, m.name, p.name, m.modality, m.status, coalesce(mp.slug, ''), coalesce(mp.name, '')
		from models m
		join providers p on p.id = m.provider_id
		left join model_profiles mp on mp.model_id = m.id and mp.status = 'public'
		order by p.name, m.name`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, slug, name, provider, modality, status, profileSlug, profileName string
		if err := rows.Scan(&id, &slug, &name, &provider, &modality, &status, &profileSlug, &profileName); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": id, "slug": slug, "name": name, "provider": provider, "modality": modality, "status": status, "profile_slug": profileSlug, "profile_name": profileName})
	}
	writeJSON(w, http.StatusOK, response{"models": items})
}

func (a *App) projects(w http.ResponseWriter, r *http.Request) {
	rows, err := a.DB.QueryContext(r.Context(), `
		select p.id, p.name, o.name, coalesce(w.paid_credits, 0), coalesce(w.promotional_credits, 0)
		from projects p
		join organizations o on o.id = p.organization_id
		left join wallets w on w.project_id = p.id
		order by p.created_at asc`)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, name, org string
		var paid, promo int64
		if err := rows.Scan(&id, &name, &org, &paid, &promo); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": id, "name": name, "organization": org, "paid_credits": paid, "promotional_credits": promo})
	}
	writeJSON(w, http.StatusOK, response{"projects": items})
}

func (a *App) createProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_project_name"})
		return
	}
	id := "project_" + randomHex(8)
	slug := slugify(name) + "-" + randomHex(3)
	_, err := a.DB.ExecContext(r.Context(), `
		insert into projects(id, organization_id, name, slug, environment, retention_policy)
		values($1, 'org-demo', $2, $3, 'dev', '{"conversation_days":30,"asset_days":30}')`, id, name, slug)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	_, _ = a.DB.ExecContext(r.Context(), `
		insert into wallets(id, project_id, paid_credits, promotional_credits)
		values($1, $2, 0, 1000)`, "wallet_"+randomHex(8), id)
	writeJSON(w, http.StatusCreated, response{"project": response{"id": id, "name": name, "organization": "Demo Organization", "paid_credits": 0, "promotional_credits": 1000}})
}

func (a *App) conversations(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_project_id"})
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `
		select c.id, c.title, c.status, c.created_at, c.updated_at, count(m.id)
		from conversations c
		left join messages m on m.conversation_id = c.id
		where c.project_id = $1
		group by c.id, c.title, c.status, c.created_at, c.updated_at
		order by c.updated_at desc`, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, title, status string
		var createdAt, updatedAt time.Time
		var messageCount int64
		if err := rows.Scan(&id, &title, &status, &createdAt, &updatedAt, &messageCount); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": id, "title": title, "status": status, "created_at": createdAt, "updated_at": updatedAt, "message_count": messageCount})
	}
	writeJSON(w, http.StatusOK, response{"conversations": items})
}

func (a *App) createConversation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string `json:"project_id"`
		Title     string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProjectID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_request"})
		return
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "New conversation"
	}
	id := "conversation_" + randomHex(8)
	_, err := a.DB.ExecContext(r.Context(), `
		insert into conversations(id, project_id, title, status)
		values($1, $2, $3, 'active')`, id, req.ProjectID, title)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	_, _ = a.DB.ExecContext(r.Context(), `
		insert into conversation_branches(id, conversation_id, parent_branch_id, name)
		values($1, $2, null, 'Main')`, "branch_"+randomHex(8), id)
	writeJSON(w, http.StatusCreated, response{"conversation": response{"id": id, "title": title, "status": "active", "message_count": 0}})
}

func (a *App) assets(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_project_id"})
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `
		select id, coalesce(conversation_id, ''), asset_type, storage_path, coalesce(mime_type, ''), coalesce(size_bytes, 0), metadata, created_at
		from workspace_assets
		where project_id = $1
		order by created_at desc`, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, conversationID, assetType, storagePath, mimeType, metadata string
		var sizeBytes int64
		var createdAt time.Time
		if err := rows.Scan(&id, &conversationID, &assetType, &storagePath, &mimeType, &sizeBytes, &metadata, &createdAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{"id": id, "conversation_id": conversationID, "asset_type": assetType, "storage_path": storagePath, "mime_type": mimeType, "size_bytes": sizeBytes, "metadata": metadata, "created_at": createdAt})
	}
	writeJSON(w, http.StatusOK, response{"assets": items})
}

func (a *App) apiKeys(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("project_id")
	rows, err := a.DB.QueryContext(r.Context(), `
		select id, name, prefix, status, created_at, revoked_at
		from api_keys
		where project_id = $1
		order by created_at desc`, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, name, prefix, status string
		var createdAt time.Time
		var revokedAt sql.NullTime
		if err := rows.Scan(&id, &name, &prefix, &status, &createdAt, &revokedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		item := response{"id": id, "name": name, "prefix": prefix, "status": status, "created_at": createdAt}
		if revokedAt.Valid {
			item["revoked_at"] = revokedAt.Time
		}
		items = append(items, item)
	}
	writeJSON(w, http.StatusOK, response{"api_keys": items})
}

func (a *App) createAPIKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string `json:"project_id"`
		Name      string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProjectID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_request"})
		return
	}
	if req.Name == "" {
		req.Name = "Development key"
	}
	raw := "mk_" + randomHex(24)
	prefix := raw[:10]
	hash := hashAPIKey(raw)
	id := "key_" + randomHex(12)
	_, err := a.DB.ExecContext(r.Context(), `
		insert into api_keys(id, project_id, name, prefix, key_hash, scopes, status)
		values($1, $2, $3, $4, $5, $6, 'active')`, id, req.ProjectID, req.Name, prefix, hash, "models:read,chat:create")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, response{"id": id, "api_key": raw, "prefix": prefix})
}

func (a *App) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/api-keys/")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_id"})
		return
	}
	_, err := a.DB.ExecContext(r.Context(), `update api_keys set status = 'revoked', revoked_at = current_timestamp where id = $1`, id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{"status": "revoked", "id": id})
}

func (a *App) chatCompletions(w http.ResponseWriter, r *http.Request) {
	projectID, err := a.authenticateAPIKey(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, response{"error": err.Error()})
		return
	}
	var req struct {
		Model    string `json:"model"`
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Model == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_request"})
		return
	}
	content := "Mock response from " + req.Model
	if len(req.Messages) > 0 {
		content = "Mock response to: " + req.Messages[len(req.Messages)-1].Content
	}
	requestID := "req_" + randomHex(12)
	_, err = a.DB.ExecContext(r.Context(), `
		insert into inference_requests(id, project_id, model_slug, provider_slug, status, input_units, output_units, customer_charge, provider_cost, margin)
		values($1, $2, $3, 'mock-provider', 'succeeded', $4, $5, 1, 0, 1)`, requestID, projectID, req.Model, len(req.Messages), len(content))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{
		"id":      requestID,
		"object":  "chat.completion",
		"model":   req.Model,
		"choices": []response{{"index": 0, "message": response{"role": "assistant", "content": content}, "finish_reason": "stop"}},
		"usage":   response{"prompt_tokens": len(req.Messages), "completion_tokens": len(content), "total_tokens": len(req.Messages) + len(content)},
	})
}

func (a *App) authenticateAPIKey(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", errors.New("missing_api_key")
	}
	hash := hashAPIKey(strings.TrimPrefix(header, "Bearer "))
	var projectID string
	err := a.DB.QueryRowContext(r.Context(), `select project_id from api_keys where key_hash = $1 and status = 'active'`, hash).Scan(&projectID)
	if err != nil {
		return "", errors.New("invalid_api_key")
	}
	return projectID, nil
}

func randomHex(bytesLen int) string {
	buf := make([]byte, bytesLen)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func hashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, ch := range value {
		if ch >= 'a' && ch <= 'z' || ch >= '0' && ch <= '9' {
			b.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "project"
	}
	return slug
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
