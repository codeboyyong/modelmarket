package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"
)

const maxExtractedFileBytes = 10 * 1024 * 1024

var errAsyncJobCancelled = errors.New("async_job_cancelled")
var errAsyncJobPending = errors.New("async_job_provider_pending")

type claimedAsyncJob struct {
	ID                            string
	ProjectID                     string
	ActorUserID                   string
	JobType                       string
	ModelSlug                     string
	ProviderSlug                  string
	Metadata                      string
	AttemptCount                  int
	MaxAttempts                   int
	InferenceRequestID            string
	ReservationWalletID           string
	ReservationPromotionalCredits float64
	ReservationPaidCredits        float64
	BillingStatus                 string
	PreviousStatus                string
	ProviderJobID                 string
	ProviderStatus                string
	StartedAt                     sql.NullTime
}

type generationJobPayload struct {
	Prompt         string   `json:"prompt"`
	Parameters     response `json:"parameters"`
	ConversationID string   `json:"conversation_id"`
	BranchID       string   `json:"branch_id"`
}

// RunAsyncWorker polls PostgreSQL for durable jobs until the context is
// cancelled. Multiple worker processes can run concurrently because claims use
// row locks with SKIP LOCKED.
func (a *App) RunAsyncWorker(ctx context.Context, pollInterval time.Duration) error {
	if pollInterval <= 0 {
		pollInterval = 2 * time.Second
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		processed, err := a.ProcessNextAsyncJob(ctx)
		if err != nil && !errors.Is(err, context.Canceled) && a.Logger != nil {
			a.Logger.Error("async_job_worker_error", "error", err)
		}
		if processed {
			continue
		}
		webhookProcessed, webhookErr := a.ProcessNextWebhookDelivery(ctx)
		if webhookErr != nil && !errors.Is(webhookErr, context.Canceled) && a.Logger != nil {
			a.Logger.Error("webhook_worker_error", "error", webhookErr)
		}
		if webhookProcessed {
			continue
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

// ProcessNextAsyncJob claims and executes one supported job. It returns false
// when the queue has no executable work.
func (a *App) ProcessNextAsyncJob(ctx context.Context) (bool, error) {
	job, err := a.claimAsyncJob(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		if errors.Is(err, errAsyncJobPending) {
			return true, nil
		}
		return false, err
	}
	if a.Logger != nil {
		a.Logger.Info("async_job_started", "job_id", job.ID, "job_type", job.JobType, "attempt", job.AttemptCount)
	}

	var result response
	completedByExecutor := false
	switch job.JobType {
	case "file_extraction":
		result, err = a.executeFileExtractionJob(ctx, job)
	case "image_generation", "video_generation", "audio_generation":
		result, err = a.executeGenerationJob(ctx, job)
		completedByExecutor = err == nil
	default:
		err = fmt.Errorf("unsupported_async_job_type: %s", job.JobType)
	}
	if err != nil {
		if errors.Is(err, errAsyncJobCancelled) {
			if cancelErr := a.markAsyncJobCancelled(context.WithoutCancel(ctx), job.ID); cancelErr != nil {
				return true, cancelErr
			}
			return true, nil
		}
		if failErr := a.failAsyncJob(context.WithoutCancel(ctx), job, err); failErr != nil {
			return true, fmt.Errorf("execute job: %v; persist failure: %w", err, failErr)
		}
		return true, nil
	}
	if completedByExecutor {
		return true, nil
	}
	if err := a.completeAsyncJob(ctx, job, result); err != nil {
		return true, err
	}
	if a.Logger != nil {
		a.Logger.Info("async_job_completed", "job_id", job.ID, "job_type", job.JobType)
	}
	return true, nil
}

func (a *App) executeGenerationJob(ctx context.Context, job claimedAsyncJob) (result response, runErr error) {
	var payload generationJobPayload
	if json.Unmarshal([]byte(job.Metadata), &payload) != nil || strings.TrimSpace(payload.Prompt) == "" || job.ModelSlug == "" {
		return nil, errors.New("invalid_generation_job_payload")
	}
	if payload.Parameters == nil {
		payload.Parameters = response{}
	}
	route, err := a.selectModelRoute(ctx, job.ModelSlug, "default")
	if err != nil {
		return nil, fmt.Errorf("no_route_for_model: %w", err)
	}
	expectedModality := strings.TrimSuffix(job.JobType, "_generation")
	if route.ModelModality != expectedModality {
		return nil, fmt.Errorf("generation_job_modality_mismatch: expected %s got %s", expectedModality, route.ModelModality)
	}
	pricingRules, pricingRatio, err := a.requestPricing(ctx, route, payload.Parameters)
	if err != nil {
		return nil, err
	}
	messages := []chatMessage{{Role: "user", Content: payload.Prompt}}
	estimate, err := chargeForUsage(route, payload.Parameters, estimateChatUsage(route, payload.Parameters, messages), pricingRules, pricingRatio)
	if err != nil {
		return nil, err
	}

	requestID := job.InferenceRequestID
	reservation := creditReservation{
		WalletID: job.ReservationWalletID, PromotionalCredits: job.ReservationPromotionalCredits, PaidCredits: job.ReservationPaidCredits,
	}
	if job.BillingStatus != "reserved" {
		requestID = "req_" + randomHex(12)
		reservation, err = a.reserveProjectCredits(ctx, job.ProjectID, requestID, estimate.CustomerCharge, func(tx *sql.Tx, reserved creditReservation) error {
			updateResult, updateErr := tx.ExecContext(ctx, `
				update user_async_jobs
				set inference_request_id = $2, reservation_wallet_id = nullif($3, ''),
					reservation_promotional_credits = $4, reservation_paid_credits = $5,
					billing_status = 'reserved', updated_at = current_timestamp
				where id = $1 and status = 'running'`,
				job.ID, requestID, reserved.WalletID, reserved.PromotionalCredits, reserved.PaidCredits)
			if updateErr != nil {
				return updateErr
			}
			if affected, _ := updateResult.RowsAffected(); affected != 1 {
				return errors.New("async_job_state_changed")
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	reservationCaptured := false
	keepReservation := false
	defer func() {
		if !reservationCaptured && !keepReservation {
			releaseErr := a.releaseProjectCredits(context.WithoutCancel(ctx), requestID, reservation, func(tx *sql.Tx) error {
				_, releaseErr := tx.ExecContext(context.WithoutCancel(ctx), `
					update user_async_jobs
					set billing_status = 'released', reservation_wallet_id = null,
						reservation_promotional_credits = 0, reservation_paid_credits = 0,
						updated_at = current_timestamp
					where id = $1`, job.ID)
				return releaseErr
			})
			if releaseErr != nil {
				result = nil
				runErr = fmt.Errorf("release_async_reservation: %w", releaseErr)
			}
		}
	}()

	if cancelled, checkErr := a.asyncJobCancellationRequested(ctx, job.ID); checkErr != nil {
		return nil, checkErr
	} else if cancelled {
		return nil, errAsyncJobCancelled
	}
	var upstream upstreamChatResult
	if route.ChannelType == "async_http" {
		var pending bool
		upstream, pending, err = a.runAsyncProviderStep(ctx, route, payload, job)
		if err != nil {
			return nil, err
		}
		if pending {
			keepReservation = true
			return nil, errAsyncJobPending
		}
	} else {
		upstream, err = a.runChatUpstream(ctx, route, messages, payload.Parameters)
	}
	if err != nil {
		return nil, err
	}
	if cancelled, checkErr := a.asyncJobCancellationRequested(ctx, job.ID); checkErr != nil {
		return nil, checkErr
	} else if cancelled {
		return nil, errAsyncJobCancelled
	}
	charge, err := chargeForUsage(route, payload.Parameters, upstream, pricingRules, pricingRatio)
	if err != nil {
		return nil, err
	}
	requestMetadata, _ := json.Marshal(response{
		"async_job_id": job.ID, "requested_model": job.ModelSlug,
		"upstream_model_id": route.UpstreamModelID, "route_group": route.RouteGroup,
		"parameters": payload.Parameters, "pricing": charge.Metadata,
	})
	result = response{}
	artifacts := []response{}
	assistantContent := upstream.Content
	err = a.captureProjectCredits(ctx, requestID, reservation, charge.CustomerCharge, func(tx *sql.Tx) error {
		_, persistErr := tx.ExecContext(ctx, `
			insert into user_inference_requests(id, project_id, actor_user_id, model_slug, model_profile_id, route_id, channel_id, provider_slug, status, input_units, output_units, customer_charge, provider_cost, margin, metadata)
			values($1,$2,nullif($3,''),$4,nullif($5,''),$6,$7,$8,'succeeded',$9,$10,$11,$12,$13,$14)`,
			requestID, job.ProjectID, job.ActorUserID, route.ModelSlug, route.ModelProfileID,
			route.ID, route.ChannelID, route.ProviderSlug, upstream.PromptTokens, upstream.CompletionTokens,
			charge.CustomerCharge, charge.ProviderCost, charge.CustomerCharge-charge.ProviderCost, string(requestMetadata))
		if persistErr != nil {
			return persistErr
		}
		if route.ChannelType == "google_gemini" {
			assistantContent, artifacts = a.rehostExternalFilesInContent(ctx, job.ProjectID, payload.ConversationID, payload.BranchID, requestID, upstream.Content, tx)
			if route.ModelModality == "image" && len(artifacts) == 0 {
				return errors.New("generated_image_storage_failed")
			}
		} else if strings.HasPrefix(route.BaseURL, "mock://") || route.ProviderSlug == "mock-provider" {
			artifacts, persistErr = a.createMockGeneratedArtifacts(ctx, job.ProjectID, payload.ConversationID, payload.BranchID, requestID, route, payload.Parameters, payload.Prompt, charge, tx)
			if persistErr != nil {
				return persistErr
			}
		}
		_, persistErr = tx.ExecContext(ctx, `
			insert into user_provider_attempts(id, inference_request_id, provider_id, channel_id, route_id, status, latency_ms, provider_request_id, error_class, metadata)
			values($1,$2,$3,$4,$5,'succeeded',$6,$7,null,$8)`,
			"attempt_"+randomHex(12), requestID, route.ProviderID, route.ChannelID, route.ID,
			upstream.LatencyMS, upstream.ProviderRequestID, fmt.Sprintf(`{"async_job_id":%q}`, job.ID))
		if persistErr != nil {
			return persistErr
		}
		if persistErr = a.recordUsageEvent(ctx, job.ProjectID, requestID, route, upstream, charge, requestMetadata, tx); persistErr != nil {
			return persistErr
		}
		if payload.ConversationID != "" {
			if persistErr = a.saveConversationTurn(ctx, job.ProjectID, payload.ConversationID, payload.BranchID, requestID, job.ModelSlug, payload.Prompt, assistantContent, charge.CustomerCharge, charge.ProviderCost, tx); persistErr != nil {
				return persistErr
			}
		}
		result = response{
			"request_id": requestID, "artifacts": artifacts,
			"usage": response{"prompt_tokens": upstream.PromptTokens, "completion_tokens": upstream.CompletionTokens, "customer_charge": charge.CustomerCharge, "provider_cost": charge.ProviderCost},
		}
		resultRaw, _ := json.Marshal(result)
		jobResult, persistErr := tx.ExecContext(ctx, `
			update user_async_jobs
			set status = 'completed', progress = 100, result_metadata = $2,
				billing_status = 'captured', completed_at = current_timestamp,
				updated_at = current_timestamp
			where id = $1 and status = 'running' and cancel_requested = false`, job.ID, truncateString(string(resultRaw), 3900))
		if persistErr != nil {
			return persistErr
		}
		if affected, _ := jobResult.RowsAffected(); affected != 1 {
			return errAsyncJobCancelled
		}
		return enqueueProjectWebhooks(ctx, tx, job.ProjectID, "async_job.completed", response{
			"id": job.ID, "project_id": job.ProjectID, "job_type": job.JobType,
			"status": "completed", "result": result,
		})
	})
	if err != nil {
		return nil, err
	}
	reservationCaptured = true
	return result, nil
}

func (a *App) asyncJobCancellationRequested(ctx context.Context, jobID string) (bool, error) {
	var cancelled bool
	err := a.DB.QueryRowContext(ctx, `select cancel_requested from user_async_jobs where id = $1`, jobID).Scan(&cancelled)
	return cancelled, err
}

func (a *App) markAsyncJobCancelled(ctx context.Context, jobID string) error {
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var projectID, jobType string
	if err = tx.QueryRowContext(ctx, `select project_id, job_type from user_async_jobs where id = $1 for update`, jobID).Scan(&projectID, &jobType); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `
		update user_async_jobs
		set status = 'cancelled', completed_at = current_timestamp, updated_at = current_timestamp
		where id = $1 and status in ('queued','running')`, jobID)
	if err != nil {
		return err
	}
	if err = enqueueProjectWebhooks(ctx, tx, projectID, "async_job.cancelled", response{
		"id": jobID, "project_id": projectID, "job_type": jobType, "status": "cancelled",
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *App) claimAsyncJob(ctx context.Context) (claimedAsyncJob, error) {
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return claimedAsyncJob{}, err
	}
	defer tx.Rollback()

	// Recover work abandoned by a terminated worker. Exhausted jobs become
	// terminal; retryable jobs return to the queue.
	if _, err = tx.ExecContext(ctx, `
		update user_async_jobs
		set status = case
			when provider_job_id is not null and provider_job_id <> '' then 'provider_pending'
			when attempt_count < max_attempts then 'queued' else 'failed' end,
			error_code = 'worker_timeout', error_message = 'worker lease expired',
			completed_at = case when attempt_count >= max_attempts then current_timestamp else null end,
			updated_at = current_timestamp
		where status = 'running' and updated_at < current_timestamp - interval '10 minutes'`); err != nil {
		return claimedAsyncJob{}, err
	}

	var job claimedAsyncJob
	err = tx.QueryRowContext(ctx, `
		select id, project_id, coalesce(actor_user_id, ''), job_type,
			coalesce(model_slug, ''), coalesce(provider_slug, ''), metadata,
			attempt_count, max_attempts, coalesce(inference_request_id, ''),
			coalesce(reservation_wallet_id, ''), reservation_promotional_credits,
			reservation_paid_credits, billing_status, status,
			coalesce(provider_job_id, ''), coalesce(provider_status, ''), started_at
		from user_async_jobs
		where ((status = 'queued' and cancel_requested = false) or status = 'provider_pending')
			and (status = 'queued' or next_provider_poll_at is null or next_provider_poll_at <= current_timestamp)
			and job_type in ('file_extraction','image_generation','video_generation','audio_generation')
		order by created_at asc
		for update skip locked
		limit 1`).Scan(
		&job.ID, &job.ProjectID, &job.ActorUserID, &job.JobType,
		&job.ModelSlug, &job.ProviderSlug, &job.Metadata,
		&job.AttemptCount, &job.MaxAttempts, &job.InferenceRequestID,
		&job.ReservationWalletID, &job.ReservationPromotionalCredits,
		&job.ReservationPaidCredits, &job.BillingStatus, &job.PreviousStatus,
		&job.ProviderJobID, &job.ProviderStatus, &job.StartedAt,
	)
	if err != nil {
		return claimedAsyncJob{}, err
	}
	if job.PreviousStatus == "queued" {
		job.AttemptCount++
	}
	if _, err = tx.ExecContext(ctx, `
		update user_async_jobs
		set status = 'running', progress = 5, attempt_count = $2,
			started_at = coalesce(started_at, current_timestamp), updated_at = current_timestamp,
			error_code = null, error_message = null
		where id = $1`, job.ID, job.AttemptCount); err != nil {
		return claimedAsyncJob{}, err
	}
	return job, tx.Commit()
}

func (a *App) executeFileExtractionJob(ctx context.Context, job claimedAsyncJob) (response, error) {
	var metadata struct {
		AssetID string `json:"asset_id"`
	}
	if json.Unmarshal([]byte(job.Metadata), &metadata) != nil || strings.TrimSpace(metadata.AssetID) == "" {
		return nil, errors.New("file_extraction_requires_asset_id")
	}
	var projectID, objectKey, mimeType string
	err := a.DB.QueryRowContext(ctx, `
		select project_id, coalesce(object_key, ''), coalesce(mime_type, '')
		from user_workbench_assets where id = $1`, metadata.AssetID).Scan(&projectID, &objectKey, &mimeType)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, errors.New("asset_not_found")
	}
	if err != nil {
		return nil, err
	}
	if projectID != job.ProjectID {
		return nil, errors.New("asset_project_mismatch")
	}
	if !extractableTextMimeType(mimeType) {
		return nil, fmt.Errorf("unsupported_extraction_mime_type: %s", mimeType)
	}
	if objectKey == "" {
		return nil, errors.New("asset_object_key_missing")
	}
	body, err := a.readStoredObject(ctx, objectKey)
	if err != nil {
		return nil, fmt.Errorf("read asset: %w", err)
	}
	if len(body) == 0 || len(body) > maxExtractedFileBytes {
		return nil, errors.New("asset_text_size_out_of_range")
	}
	if !utf8.Valid(body) {
		return nil, errors.New("asset_text_is_not_utf8")
	}
	return response{
		"asset_id":   metadata.AssetID,
		"text":       string(body),
		"mime_type":  mimeType,
		"size_bytes": len(body),
	}, nil
}

func extractableTextMimeType(value string) bool {
	value = strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	return strings.HasPrefix(value, "text/") || value == "application/json" || value == "application/xml"
}

func (a *App) completeAsyncJob(ctx context.Context, job claimedAsyncJob, result response) error {
	assetID, _ := result["asset_id"].(string)
	extractedText, _ := result["text"].(string)
	delete(result, "text")
	chunks := []fileChunk{}
	if job.JobType == "file_extraction" {
		chunks = chunkExtractedText(extractedText)
		result["chunk_count"] = len(chunks)
		result["embedding_model"] = localEmbeddingModel
	}
	resultRaw, _ := json.Marshal(result)
	extractionMetadata, _ := json.Marshal(response{
		"async_job_id": job.ID,
		"mime_type":    result["mime_type"],
		"size_bytes":   result["size_bytes"],
	})

	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var cancelRequested bool
	if err = tx.QueryRowContext(ctx, `select cancel_requested from user_async_jobs where id = $1 for update`, job.ID).Scan(&cancelRequested); err != nil {
		return err
	}
	if cancelRequested {
		_, err = tx.ExecContext(ctx, `update user_async_jobs set status = 'cancelled', completed_at = current_timestamp, updated_at = current_timestamp where id = $1`, job.ID)
		if err != nil {
			return err
		}
		if err = enqueueProjectWebhooks(ctx, tx, job.ProjectID, "async_job.cancelled", response{
			"id": job.ID, "project_id": job.ProjectID, "job_type": job.JobType, "status": "cancelled",
		}); err != nil {
			return err
		}
		return tx.Commit()
	}
	if job.JobType == "file_extraction" {
		extractionID := "extract_" + randomHex(12)
		_, err = tx.ExecContext(ctx, `
			insert into user_file_extractions(id, asset_id, extracted_text, metadata)
			values($1,$2,$3,$4)`, extractionID, assetID, extractedText, string(extractionMetadata))
		if err != nil {
			return err
		}
		for _, chunk := range chunks {
			chunkID := "chunk_" + randomHex(12)
			if _, err = tx.ExecContext(ctx, `
				insert into user_file_chunks(id, extraction_id, asset_id, project_id, chunk_index, content, char_count, token_estimate, embedding_model, embedding_data, metadata)
				values($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'{}')`,
				chunkID, extractionID, assetID, job.ProjectID, chunk.Index, chunk.Content,
				chunk.CharCount, chunk.TokenEstimate, localEmbeddingModel, encodeEmbedding(chunk.Embedding)); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `
				insert into user_embedding_records(id, project_id, source_type, source_id, embedding_model, vector_ref, metadata)
				values($1,$2,'file_chunk',$3,$4,$5,$6)`,
				"embedding_"+randomHex(12), job.ProjectID, chunkID, localEmbeddingModel, "inline:user_file_chunks", fmt.Sprintf(`{"asset_id":%q,"chunk_index":%d}`, assetID, chunk.Index)); err != nil {
				return err
			}
		}
	}
	resultExec, err := tx.ExecContext(ctx, `
		update user_async_jobs
		set status = 'completed', progress = 100, result_metadata = $2,
			completed_at = current_timestamp, updated_at = current_timestamp
		where id = $1 and status = 'running'`, job.ID, truncateString(string(resultRaw), 3900))
	if err != nil {
		return err
	}
	if affected, _ := resultExec.RowsAffected(); affected != 1 {
		return errors.New("async_job_state_changed")
	}
	if err = enqueueProjectWebhooks(ctx, tx, job.ProjectID, "async_job.completed", response{
		"id": job.ID, "project_id": job.ProjectID, "job_type": job.JobType,
		"status": "completed", "result": result,
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (a *App) failAsyncJob(ctx context.Context, job claimedAsyncJob, jobErr error) error {
	status := "queued"
	progress := 0
	var completedAt any
	if job.AttemptCount >= job.MaxAttempts || permanentAsyncJobError(jobErr) {
		status = "failed"
		completedAt = time.Now().UTC()
	}
	if status != "failed" {
		_, err := a.DB.ExecContext(ctx, `
			update user_async_jobs
			set status = $2, progress = $3, error_code = $4, error_message = $5,
				completed_at = $6, updated_at = current_timestamp
			where id = $1 and status = 'running'`,
			job.ID, status, progress, asyncJobErrorCode(jobErr), truncateString(jobErr.Error(), 1900), completedAt)
		if a.Logger != nil {
			a.Logger.Log(ctx, slog.LevelWarn, "async_job_failed", "job_id", job.ID, "status", status, "error", jobErr)
		}
		return err
	}
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `
		update user_async_jobs
		set status = $2, progress = $3, error_code = $4, error_message = $5,
			completed_at = $6, updated_at = current_timestamp
		where id = $1 and status = 'running'`,
		job.ID, status, progress, asyncJobErrorCode(jobErr), truncateString(jobErr.Error(), 1900), completedAt)
	if err == nil {
		err = enqueueProjectWebhooks(ctx, tx, job.ProjectID, "async_job.failed", response{
			"id": job.ID, "project_id": job.ProjectID, "job_type": job.JobType,
			"status": "failed", "error_code": asyncJobErrorCode(jobErr), "error_message": truncateString(jobErr.Error(), 1900),
		})
	}
	if a.Logger != nil {
		a.Logger.Log(ctx, slog.LevelWarn, "async_job_failed", "job_id", job.ID, "status", status, "error", jobErr)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

func permanentAsyncJobError(err error) bool {
	message := err.Error()
	for _, prefix := range []string{
		"file_extraction_requires_asset_id", "asset_not_found", "asset_project_mismatch",
		"unsupported_extraction_mime_type", "asset_object_key_missing",
		"asset_text_size_out_of_range", "asset_text_is_not_utf8", "unsupported_async_job_type",
		"invalid_generation_job_payload", "no_route_for_model", "generation_job_modality_mismatch",
		"insufficient_credits", "no_active_price_rule", "unsupported_billing_unit",
		"invalid_async_provider_settings", "invalid_async_provider_poll_limits",
		"async_provider_missing_job_id", "async_provider_missing_output",
		"async_provider_failed", "async_provider_poll_timeout",
	} {
		if strings.HasPrefix(message, prefix) {
			return true
		}
	}
	return false
}

func asyncJobErrorCode(err error) string {
	code := strings.SplitN(err.Error(), ":", 2)[0]
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "_")
	if code == "" {
		return "async_job_failed"
	}
	return truncateString(code, 255)
}
