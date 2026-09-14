package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode"
)

func (a *App) requireSystemAdmin(ctx context.Context, userID string) error {
	if strings.TrimSpace(userID) == "" {
		return sql.ErrNoRows
	}
	var userType, status string
	err := a.DB.QueryRowContext(ctx, `select user_type, status from sys_users where id = $1`, userID).Scan(&userType, &status)
	if err != nil || userType != "sys_admin" || status != "active" {
		return sql.ErrNoRows
	}
	return nil
}

func validateAccountPassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	var letter, number, special bool
	for _, value := range password {
		letter = letter || unicode.IsLetter(value)
		number = number || unicode.IsNumber(value)
		special = special || (!unicode.IsLetter(value) && !unicode.IsNumber(value))
	}
	if !letter || !number || !special {
		return errors.New("password must include a letter, number, and special character")
	}
	return nil
}

func (a *App) adminUsers(w http.ResponseWriter, r *http.Request) {
	if err := a.requireSystemAdmin(r.Context(), r.URL.Query().Get("user_id")); err != nil {
		writeJSON(w, http.StatusForbidden, response{"error": "admin_required"})
		return
	}
	query := "%" + strings.ToLower(strings.TrimSpace(r.URL.Query().Get("query"))) + "%"
	rows, err := a.DB.QueryContext(r.Context(), `
		select id, email, name, user_type, status, created_at
		from sys_users
		where lower(email) like $1 or lower(name) like $1
		order by created_at desc, email`, query)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "users_load_failed"})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, email, name, userType, status string
		var createdAt any
		if err := rows.Scan(&id, &email, &name, &userType, &status, &createdAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": "users_load_failed"})
			return
		}
		items = append(items, response{"id": id, "email": email, "name": name, "user_type": userType, "status": status, "created_at": createdAt})
	}
	writeJSON(w, http.StatusOK, response{"users": items})
}

func (a *App) createAdminUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
		UserType string `json:"user_type"`
	}
	if decodeJSON(r, &req) != nil || a.requireSystemAdmin(r.Context(), req.UserID) != nil {
		writeJSON(w, http.StatusForbidden, response{"error": "admin_required"})
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" && strings.Contains(req.Email, "@") {
		req.Name = strings.Split(req.Email, "@")[0]
	}
	if req.UserType != "sys_admin" && req.UserType != "individual_consumer" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_user_type"})
		return
	}
	if !strings.Contains(req.Email, "@") || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_user"})
		return
	}
	if err := validateAccountPassword(req.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "weak_password", "message": err.Error()})
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "user_create_failed"})
		return
	}
	defer tx.Rollback()
	userID := "user_" + randomHex(12)
	if _, err = tx.ExecContext(r.Context(), `insert into sys_users(id,email,name,status,password_hash,user_type,ui_theme,language) values($1,$2,$3,'active',$4,$5,'Light','EN')`, userID, req.Email, req.Name, a.passwordHash(req.Password), req.UserType); err != nil {
		writeJSON(w, http.StatusConflict, response{"error": "user_exists_or_invalid"})
		return
	}
	var organizationID string
	if err = tx.QueryRowContext(r.Context(), `select id from sys_organizations where status = 'active' order by created_at limit 1`).Scan(&organizationID); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "organization_not_found"})
		return
	}
	role := "developer"
	if req.UserType == "sys_admin" {
		role = "owner"
	}
	if _, err = tx.ExecContext(r.Context(), `insert into sys_memberships(id,user_id,organization_id,role) values($1,$2,$3,$4)`, "membership_"+randomHex(12), userID, organizationID, role); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "user_create_failed"})
		return
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "user_create_failed"})
		return
	}
	writeJSON(w, http.StatusCreated, response{"user": response{"id": userID, "email": req.Email, "name": req.Name, "user_type": req.UserType, "status": "active"}})
}

func (a *App) updateAdminUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID string `json:"user_id"`
		Status string `json:"status"`
	}
	if decodeJSON(r, &req) != nil || a.requireSystemAdmin(r.Context(), req.UserID) != nil {
		writeJSON(w, http.StatusForbidden, response{"error": "admin_required"})
		return
	}
	targetID := r.PathValue("id")
	if targetID == req.UserID {
		writeJSON(w, http.StatusBadRequest, response{"error": "cannot_change_own_status"})
		return
	}
	if req.Status != "active" && req.Status != "suspended" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_status"})
		return
	}
	result, err := a.DB.ExecContext(r.Context(), `update sys_users set status = $1 where id = $2`, req.Status, targetID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "user_update_failed"})
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		writeJSON(w, http.StatusNotFound, response{"error": "user_not_found"})
		return
	}
	if req.Status == "suspended" {
		_, _ = a.DB.ExecContext(r.Context(), `delete from sys_sessions where user_id = $1`, targetID)
	}
	writeJSON(w, http.StatusOK, response{"status": req.Status})
}

func (a *App) resetAdminUserPassword(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		Password string `json:"password"`
	}
	if decodeJSON(r, &req) != nil || a.requireSystemAdmin(r.Context(), req.UserID) != nil {
		writeJSON(w, http.StatusForbidden, response{"error": "admin_required"})
		return
	}
	if err := validateAccountPassword(req.Password); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "weak_password", "message": err.Error()})
		return
	}
	targetID := r.PathValue("id")
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "password_reset_failed"})
		return
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(r.Context(), `update sys_users set password_hash = $1 where id = $2`, a.passwordHash(req.Password), targetID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "password_reset_failed"})
		return
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		writeJSON(w, http.StatusNotFound, response{"error": "user_not_found"})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `delete from sys_sessions where user_id = $1`, targetID); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "password_reset_failed"})
		return
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": "password_reset_failed"})
		return
	}
	writeJSON(w, http.StatusOK, response{"status": "password_reset"})
}

func decodeJSON(r *http.Request, target any) error {
	if r.Body == nil {
		return errors.New("missing body")
	}
	return json.NewDecoder(r.Body).Decode(target)
}
