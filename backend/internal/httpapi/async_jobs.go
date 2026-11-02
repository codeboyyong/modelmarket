package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var supportedAsyncJobTypes = map[string]bool{
	"image_generation": true,
	"video_generation": true,
	"audio_generation": true,
	"file_extraction":  true,
}

type asyncJobRecord struct {
	ID              string
	ProjectID       string
	ActorUserID     string
	JobType         string
	Status          string
	Progress        int
	ModelSlug       string
	ProviderSlug    string
	Metadata        string
	ResultMetadata  string
	ErrorCode       string
	ErrorMessage    string
	AttemptCount    int
	MaxAttempts     int
	CancelRequested bool
	StartedAt       sql.NullTime
	CompletedAt     sql.NullTime
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (j asyncJobRecord) response() response {
	var metadata, resultMetadata any
	if json.Unmarshal([]byte(j.Metadata), &metadata) != nil {
		metadata = response{}
	}
	if json.Unmarshal([]byte(j.ResultMetadata), &resultMetadata) != nil {
		resultMetadata = response{}
	}
	return response{
		"id": j.ID, "project_id": j.ProjectID, "actor_user_id": j.ActorUserID,
		"job_type": j.JobType, "status": j.Status, "progress": j.Progress,
		"model": j.ModelSlug, "provider": j.ProviderSlug, "metadata": metadata,
		"result": resultMetadata, "error_code": j.ErrorCode, "error_message": j.ErrorMessage,
		"attempt_count": j.AttemptCount, "max_attempts": j.MaxAttempts,
		"cancel_requested": j.CancelRequested, "started_at": nullTime(j.StartedAt),
		"completed_at": nullTime(j.CompletedAt), "created_at": j.CreatedAt, "updated_at": j.UpdatedAt,
	}
}

const asyncJobColumns = `
	id, project_id, coalesce(actor_user_id, ''), job_type, status, progress,
	coalesce(model_slug, ''), coalesce(provider_slug, ''), metadata, result_metadata,
	coalesce(error_code, ''), coalesce(error_message, ''), attempt_count, max_attempts,
	cancel_requested, started_at, completed_at, created_at, updated_at`

type rowScanner interface {
	Scan(...any) error
}

func scanAsyncJob(row rowScanner) (asyncJobRecord, error) {
	var job asyncJobRecord
	err := row.Scan(
		&job.ID, &job.ProjectID, &job.ActorUserID, &job.JobType, &job.Status, &job.Progress,
		&job.ModelSlug, &job.ProviderSlug, &job.Metadata, &job.ResultMetadata,
		&job.ErrorCode, &job.ErrorMessage, &job.AttemptCount, &job.MaxAttempts,
		&job.CancelRequested, &job.StartedAt, &job.CompletedAt, &job.CreatedAt, &job.UpdatedAt,
	)
	return job, err
}

func (a *App) asyncJobs(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_project_id"})
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			writeJSON(w, http.StatusBadRequest, response{"error": "invalid_limit"})
			return
		}
		limit = parsed
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	query := `select ` + asyncJobColumns + ` from user_async_jobs where project_id = $1`
	args := []any{projectID}
	if status != "" {
		query += ` and status = $2 order by created_at desc limit $3`
		args = append(args, status, limit)
	} else {
		query += ` order by created_at desc limit $2`
		args = append(args, limit)
	}
	rows, err := a.DB.QueryContext(r.Context(), query, args...)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		job, err := scanAsyncJob(rows)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, job.response())
	}
	writeJSON(w, http.StatusOK, response{"jobs": items})
}

func (a *App) createAsyncJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID      string         `json:"project_id"`
		JobType        string         `json:"job_type"`
		Model          string         `json:"model"`
		Provider       string         `json:"provider"`
		Metadata       map[string]any `json:"metadata"`
		MaxAttempts    int            `json:"max_attempts"`
		IdempotencyKey string         `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_json"})
		return
	}
	req.ProjectID = strings.TrimSpace(req.ProjectID)
	req.JobType = strings.TrimSpace(req.JobType)
	if req.ProjectID == "" || !supportedAsyncJobTypes[req.JobType] {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_async_job"})
		return
	}
	if req.MaxAttempts == 0 {
		req.MaxAttempts = 3
	}
	if req.MaxAttempts < 1 || req.MaxAttempts > 10 || len(req.IdempotencyKey) > 255 {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_async_job_options"})
		return
	}
	if req.Metadata == nil {
		req.Metadata = map[string]any{}
	}
	if req.JobType == "file_extraction" {
		assetID, _ := req.Metadata["asset_id"].(string)
		if strings.TrimSpace(assetID) == "" {
			writeJSON(w, http.StatusBadRequest, response{"error": "file_extraction_requires_asset_id"})
			return
		}
	}
	if req.JobType != "file_extraction" {
		prompt, _ := req.Metadata["prompt"].(string)
		if strings.TrimSpace(req.Model) == "" || strings.TrimSpace(prompt) == "" {
			writeJSON(w, http.StatusBadRequest, response{"error": "generation_job_requires_model_and_prompt"})
			return
		}
	}
	metadata, err := json.Marshal(req.Metadata)
	if err != nil || len(metadata) > 4000 {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_async_job_metadata"})
		return
	}
	var actorUserID any
	if user, ok := currentUser(r.Context()); ok {
		actorUserID = user.ID
	}
	var idempotencyKey any
	if value := strings.TrimSpace(req.IdempotencyKey); value != "" {
		idempotencyKey = value
	}
	jobID := "job_" + randomHex(12)
	job, err := scanAsyncJob(a.DB.QueryRowContext(r.Context(), `
		insert into user_async_jobs(
			id, project_id, actor_user_id, job_type, status, progress, model_slug,
			provider_slug, metadata, max_attempts, idempotency_key)
		values($1,$2,$3,$4,'queued',0,$5,$6,$7,$8,$9)
		on conflict(project_id, idempotency_key) do update set idempotency_key = excluded.idempotency_key
		returning `+asyncJobColumns,
		jobID, req.ProjectID, actorUserID, req.JobType, nullIfEmpty(strings.TrimSpace(req.Model)),
		nullIfEmpty(strings.TrimSpace(req.Provider)), string(metadata), req.MaxAttempts, idempotencyKey,
	))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, response{"job": job.response()})
}

func (a *App) asyncJob(w http.ResponseWriter, r *http.Request) {
	job, err := scanAsyncJob(a.DB.QueryRowContext(r.Context(), `select `+asyncJobColumns+` from user_async_jobs where id = $1`, r.PathValue("id")))
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, response{"error": "async_job_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{"job": job.response()})
}

func (a *App) cancelAsyncJob(w http.ResponseWriter, r *http.Request) {
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	var status, projectID, jobType string
	err = tx.QueryRowContext(r.Context(), `
		update user_async_jobs
		set cancel_requested = true,
			status = case when status = 'queued' then 'cancelled' else status end,
			completed_at = case when status = 'queued' then current_timestamp else completed_at end,
			updated_at = current_timestamp
		where id = $1 and status in ('queued','running','provider_pending')
		returning status, project_id, job_type`, r.PathValue("id")).Scan(&status, &projectID, &jobType)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusConflict, response{"error": "async_job_not_cancellable"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if status == "cancelled" {
		if err = enqueueProjectWebhooks(r.Context(), tx, projectID, "async_job.cancelled", response{
			"id": r.PathValue("id"), "project_id": projectID, "job_type": jobType, "status": "cancelled",
		}); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{"status": status})
}

func (a *App) retryAsyncJob(w http.ResponseWriter, r *http.Request) {
	var status string
	err := a.DB.QueryRowContext(r.Context(), `
		update user_async_jobs
		set status = 'queued', progress = 0, cancel_requested = false,
			error_code = null, error_message = null, started_at = null,
			completed_at = null, provider_job_id = null, provider_status = null,
			next_provider_poll_at = null, inference_request_id = null,
			reservation_wallet_id = null, reservation_promotional_credits = 0,
			reservation_paid_credits = 0, billing_status = 'unreserved',
			updated_at = current_timestamp
		where id = $1 and status = 'failed' and attempt_count < max_attempts
		returning status`, r.PathValue("id")).Scan(&status)
	writeAsyncJobTransition(w, err, status, "async_job_not_retryable")
}

func writeAsyncJobTransition(w http.ResponseWriter, err error, status, conflictCode string) {
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusConflict, response{"error": conflictCode})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{"status": status})
}

func (a *App) assetExtraction(w http.ResponseWriter, r *http.Request) {
	assetID := r.PathValue("id")
	var id, extractedText, metadata string
	var createdAt time.Time
	err := a.DB.QueryRowContext(r.Context(), `
		select id, coalesce(extracted_text, ''), metadata, created_at
		from user_file_extractions
		where asset_id = $1
		order by created_at desc limit 1`, assetID).Scan(&id, &extractedText, &metadata, &createdAt)
	if err == nil {
		var decodedMetadata any
		if json.Unmarshal([]byte(metadata), &decodedMetadata) != nil {
			decodedMetadata = response{}
		}
		writeJSON(w, http.StatusOK, response{"extraction": response{
			"id": id, "asset_id": assetID, "text": extractedText,
			"metadata": decodedMetadata, "created_at": createdAt,
		}})
		return
	}
	if !errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	var status, errorMessage string
	err = a.DB.QueryRowContext(r.Context(), `
		select status, coalesce(error_message, '')
		from user_async_jobs
		where job_type = 'file_extraction' and metadata like $1
		order by created_at desc limit 1`, `%"asset_id":"`+assetID+`"%`).Scan(&status, &errorMessage)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, response{"error": "extraction_not_found"})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, response{"asset_id": assetID, "status": status, "error_message": errorMessage})
}
