package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (a *App) promptPresets(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectID == "" {
		writeJSON(w, 400, response{"error": "missing_project_id"})
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `select id, name, model_slug, prompt, parameters, created_at, updated_at from user_prompt_presets where project_id = $1 order by name`, projectID)
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, name, model, prompt, rawParameters string
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&id, &name, &model, &prompt, &rawParameters, &createdAt, &updatedAt); err != nil {
			writeJSON(w, 500, response{"error": err.Error()})
			return
		}
		parameters := response{}
		_ = json.Unmarshal([]byte(rawParameters), &parameters)
		items = append(items, response{"id": id, "name": name, "model": model, "prompt": prompt, "parameters": parameters, "created_at": createdAt, "updated_at": updatedAt})
	}
	writeJSON(w, http.StatusOK, response{"presets": items})
}

func (a *App) createPromptPreset(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID  string   `json:"project_id"`
		Name       string   `json:"name"`
		Model      string   `json:"model"`
		Prompt     string   `json:"prompt"`
		Parameters response `json:"parameters"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.ProjectID) == "" || strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.Model) == "" {
		writeJSON(w, 400, response{"error": "invalid_prompt_preset"})
		return
	}
	raw, err := json.Marshal(req.Parameters)
	if err != nil || len(raw) > 4000 {
		writeJSON(w, 400, response{"error": "invalid_preset_parameters"})
		return
	}
	id := "preset_" + randomHex(10)
	_, err = a.DB.ExecContext(r.Context(), `insert into user_prompt_presets(id, project_id, name, model_slug, prompt, parameters) values($1,$2,$3,$4,$5,$6) on conflict (project_id, name) do update set model_slug = excluded.model_slug, prompt = excluded.prompt, parameters = excluded.parameters, updated_at = current_timestamp`, id, req.ProjectID, strings.TrimSpace(req.Name), req.Model, req.Prompt, string(raw))
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, response{"preset": response{"id": id, "name": strings.TrimSpace(req.Name), "model": req.Model, "prompt": req.Prompt, "parameters": req.Parameters}})
}

func (a *App) deletePromptPreset(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(r.PathValue("id"))
	result, err := a.DB.ExecContext(r.Context(), `delete from user_prompt_presets where id = $1`, id)
	if err != nil {
		writeJSON(w, 500, response{"error": err.Error()})
		return
	}
	count, _ := result.RowsAffected()
	if count == 0 {
		writeJSON(w, 404, response{"error": "prompt_preset_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, response{"status": "deleted", "id": id})
}
