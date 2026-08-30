package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

func (a *App) updateConversation(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	var req struct {
		Title string `json:"title"`
	}
	if id == "" || json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.Title) == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_conversation_update"})
		return
	}
	req.Title = strings.TrimSpace(req.Title)
	result, err := a.DB.ExecContext(r.Context(), `update user_conversations set title = $1, updated_at = current_timestamp where id = $2`, req.Title, id)
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		writeJSON(w, 404, response{"error": "conversation_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, response{"conversation": response{"id": id, "title": req.Title}})
}

func (a *App) deleteConversation(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	if id == "" {
		writeJSON(w, 400, response{"error": "missing_conversation_id"})
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `select object_key from user_workbench_assets where conversation_id = $1 and coalesce(object_key, '') <> ''`, id)
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	objectKeys := []string{}
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			rows.Close()
			writeJSON(w, 500, response{"error": err.Error()})
			return
		}
		objectKeys = append(objectKeys, key)
	}
	rows.Close()
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	statements := []string{
		`delete from user_message_attachments where message_id in (select id from user_messages where conversation_id = $1) or asset_id in (select id from user_workbench_assets where conversation_id = $1)`,
		`delete from user_file_extractions where asset_id in (select id from user_workbench_assets where conversation_id = $1)`,
		`delete from user_workbench_assets where conversation_id = $1`,
		`delete from user_messages where conversation_id = $1`,
		`delete from user_conversation_branches where conversation_id = $1`,
		`delete from user_conversations where id = $1`,
	}
	for _, statement := range statements {
		result, err := tx.ExecContext(r.Context(), statement, id)
		if err != nil {
			writeJSON(w, 500, response{"error": err.Error()})
			return
		}
		if strings.Contains(statement, "delete from user_conversations") {
			count, _ := result.RowsAffected()
			if count == 0 {
				writeJSON(w, 404, response{"error": "conversation_not_found"})
				return
			}
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	for _, key := range objectKeys {
		_ = a.deleteStoredObject(r.Context(), key)
	}
	writeJSON(w, http.StatusOK, response{"status": "deleted", "id": id, "assets_deleted": len(objectKeys)})
}

func (a *App) exportConversation(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	var title, status, projectID string
	var createdAt, updatedAt time.Time
	if err := a.DB.QueryRowContext(r.Context(), `select project_id, title, status, created_at, updated_at from user_conversations where id = $1`, id).Scan(&projectID, &title, &status, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, 404, response{"error": "conversation_not_found"})
		} else {
			writeJSON(w, 500, response{"error": err.Error()})
		}
		return
	}
	branches, err := a.exportRows(r, `select id, coalesce(parent_branch_id, ''), name, created_at from user_conversation_branches where conversation_id = $1 order by created_at`, id, func(rows *sql.Rows) (response, error) {
		var branchID, parentID, name string
		var created time.Time
		err := rows.Scan(&branchID, &parentID, &name, &created)
		return response{"id": branchID, "parent_branch_id": parentID, "name": name, "created_at": created}, err
	})
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	messages, err := a.exportRows(r, `select id, coalesce(branch_id, ''), role, content, coalesce(model_profile_id, ''), customer_charge, provider_cost, created_at from user_messages where conversation_id = $1 order by created_at, id`, id, func(rows *sql.Rows) (response, error) {
		var messageID, branchID, role, content, profile string
		var charge, cost int64
		var created time.Time
		err := rows.Scan(&messageID, &branchID, &role, &content, &profile, &charge, &cost, &created)
		return response{"id": messageID, "branch_id": branchID, "role": role, "content": content, "model_profile_id": profile, "customer_charge": charge, "provider_cost": cost, "created_at": created}, err
	})
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	assets, err := a.exportRows(r, `select id, coalesce(branch_id, ''), asset_type, asset_origin, coalesce(download_url, ''), coalesce(mime_type, ''), coalesce(size_bytes, 0), created_at from user_workbench_assets where conversation_id = $1 order by created_at`, id, func(rows *sql.Rows) (response, error) {
		var assetID, branchID, assetType, origin, url, mime string
		var size int64
		var created time.Time
		err := rows.Scan(&assetID, &branchID, &assetType, &origin, &url, &mime, &size, &created)
		return response{"id": assetID, "branch_id": branchID, "asset_type": assetType, "asset_origin": origin, "download_url": url, "mime_type": mime, "size_bytes": size, "created_at": created}, err
	})
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="conversation-`+sanitizeFilename(id)+`.json"`)
	writeJSON(w, http.StatusOK, response{"version": 1, "exported_at": time.Now().UTC(), "conversation": response{"id": id, "project_id": projectID, "title": title, "status": status, "created_at": createdAt, "updated_at": updatedAt}, "branches": branches, "messages": messages, "assets": assets})
}

func (a *App) exportRows(r *http.Request, query string, id string, scan func(*sql.Rows) (response, error)) ([]response, error) {
	rows, err := a.DB.QueryContext(r.Context(), query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (a *App) deleteAsset(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	var objectKey string
	if err := a.DB.QueryRowContext(r.Context(), `select coalesce(object_key, '') from user_workbench_assets where id = $1`, id).Scan(&objectKey); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, 404, response{"error": "asset_not_found"})
		} else {
			writeJSON(w, 500, response{"error": err.Error()})
		}
		return
	}
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	for _, statement := range []string{`delete from user_message_attachments where asset_id = $1`, `delete from user_file_extractions where asset_id = $1`, `delete from user_workbench_assets where id = $1`} {
		if _, err := tx.ExecContext(r.Context(), statement, id); err != nil {
			writeJSON(w, 500, response{"error": err.Error()})
			return
		}
	}
	if err := tx.Commit(); err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	if objectKey != "" {
		_ = a.deleteStoredObject(r.Context(), objectKey)
	}
	writeJSON(w, http.StatusOK, response{"status": "deleted", "id": id})
}
