package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var webhookSecretRefPattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{2,127}$`)

var supportedWebhookEvents = map[string]bool{
	"async_job.completed": true,
	"async_job.failed":    true,
	"async_job.cancelled": true,
}

func (a *App) webhookEndpoints(w http.ResponseWriter, r *http.Request) {
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectID == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "missing_project_id"})
		return
	}
	rows, err := a.DB.QueryContext(r.Context(), `
		select id, url, coalesce(secret_ref, ''), events, status, created_at
		from user_webhook_endpoints where project_id = $1 order by created_at desc`, projectID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, endpointURL, secretRef, events, status string
		var createdAt time.Time
		if err := rows.Scan(&id, &endpointURL, &secretRef, &events, &status, &createdAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{
			"id": id, "url": endpointURL, "secret_ref": secretRef, "events": strings.Split(events, ","),
			"status": status, "configured": strings.TrimSpace(os.Getenv(secretRef)) != "", "created_at": createdAt,
		})
	}
	writeJSON(w, http.StatusOK, response{"webhooks": items})
}

func (a *App) createWebhookEndpoint(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ProjectID string   `json:"project_id"`
		URL       string   `json:"url"`
		SecretRef string   `json:"secret_ref"`
		Events    []string `json:"events"`
	}
	if json.NewDecoder(r.Body).Decode(&req) != nil || strings.TrimSpace(req.ProjectID) == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_webhook"})
		return
	}
	if err := validateWebhookURL(req.URL, a.Config.DevMode); err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": err.Error()})
		return
	}
	req.SecretRef = strings.TrimSpace(req.SecretRef)
	if !webhookSecretRefPattern.MatchString(req.SecretRef) {
		writeJSON(w, http.StatusBadRequest, response{"error": "invalid_webhook_secret_ref"})
		return
	}
	if strings.TrimSpace(os.Getenv(req.SecretRef)) == "" {
		writeJSON(w, http.StatusBadRequest, response{"error": "webhook_secret_not_configured"})
		return
	}
	events, err := normalizeWebhookEvents(req.Events)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, response{"error": err.Error()})
		return
	}
	id := "webhook_" + randomHex(12)
	_, err = a.DB.ExecContext(r.Context(), `
		insert into user_webhook_endpoints(id, project_id, url, secret_ref, events, status)
		values($1,$2,$3,$4,$5,'active')`, id, req.ProjectID, strings.TrimSpace(req.URL), req.SecretRef, strings.Join(events, ","))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, response{"webhook": response{
		"id": id, "project_id": req.ProjectID, "url": strings.TrimSpace(req.URL),
		"secret_ref": req.SecretRef, "events": events, "status": "active",
	}})
}

func normalizeWebhookEvents(values []string) ([]string, error) {
	if len(values) == 0 {
		return []string{"*"}, nil
	}
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "*" && !supportedWebhookEvents[value] {
			return nil, fmt.Errorf("unsupported_webhook_event: %s", value)
		}
		seen[value] = true
	}
	if seen["*"] {
		return []string{"*"}, nil
	}
	events := make([]string, 0, len(seen))
	for event := range seen {
		events = append(events, event)
	}
	sort.Strings(events)
	return events, nil
}

func validateWebhookURL(raw string, devMode bool) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.Fragment != "" {
		return errors.New("invalid_webhook_url")
	}
	if parsed.Scheme != "https" && !(devMode && parsed.Scheme == "http") {
		return errors.New("webhook_https_required")
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		if !devMode {
			return errors.New("webhook_private_address_not_allowed")
		}
		return nil
	}
	if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast()) && !devMode {
		return errors.New("webhook_private_address_not_allowed")
	}
	return nil
}

func (a *App) disableWebhookEndpoint(w http.ResponseWriter, r *http.Request) {
	tx, err := a.DB.BeginTx(r.Context(), nil)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(r.Context(), `update user_webhook_endpoints set status = 'disabled' where id = $1 and status <> 'disabled'`, r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		writeJSON(w, http.StatusNotFound, response{"error": "webhook_not_found"})
		return
	}
	if _, err = tx.ExecContext(r.Context(), `
		update user_webhook_deliveries
		set status = 'cancelled', error_message = 'webhook endpoint disabled', updated_at = current_timestamp
		where endpoint_id = $1 and status in ('pending','retrying')`, r.PathValue("id")); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if err = tx.Commit(); err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, response{"status": "disabled"})
}

func (a *App) webhookDeliveries(w http.ResponseWriter, r *http.Request) {
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
	rows, err := a.DB.QueryContext(r.Context(), `
		select id, endpoint_id, event_type, status, attempt_count, max_attempts,
			coalesce(response_status, 0), coalesce(response_body, ''), coalesce(error_message, ''),
			next_attempt_at, delivered_at, created_at, updated_at
		from user_webhook_deliveries where project_id = $1
		order by created_at desc limit $2`, projectID, limit)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	defer rows.Close()
	items := []response{}
	for rows.Next() {
		var id, endpointID, eventType, status, responseBody, errorMessage string
		var attempts, maxAttempts, responseStatus int
		var nextAttempt, createdAt, updatedAt time.Time
		var deliveredAt sql.NullTime
		if err := rows.Scan(&id, &endpointID, &eventType, &status, &attempts, &maxAttempts, &responseStatus, &responseBody, &errorMessage, &nextAttempt, &deliveredAt, &createdAt, &updatedAt); err != nil {
			writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
			return
		}
		items = append(items, response{
			"id": id, "endpoint_id": endpointID, "event_type": eventType, "status": status,
			"attempt_count": attempts, "max_attempts": maxAttempts, "response_status": responseStatus,
			"response_body": responseBody, "error_message": errorMessage, "next_attempt_at": nextAttempt,
			"delivered_at": nullTime(deliveredAt), "created_at": createdAt, "updated_at": updatedAt,
		})
	}
	writeJSON(w, http.StatusOK, response{"deliveries": items})
}

func (a *App) replayWebhookDelivery(w http.ResponseWriter, r *http.Request) {
	result, err := a.DB.ExecContext(r.Context(), `
		update user_webhook_deliveries
		set status = 'pending', attempt_count = 0, next_attempt_at = current_timestamp,
			response_status = null, response_body = null, error_message = null,
			delivered_at = null, updated_at = current_timestamp
		where id = $1 and status in ('delivered','failed','cancelled')`, r.PathValue("id"))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, response{"error": err.Error()})
		return
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		writeJSON(w, http.StatusConflict, response{"error": "webhook_delivery_not_replayable"})
		return
	}
	writeJSON(w, http.StatusAccepted, response{"status": "pending"})
}

func enqueueProjectWebhooks(ctx context.Context, tx *sql.Tx, projectID, eventType string, payload response) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if len(raw) > 3900 {
		return errors.New("webhook_payload_too_large")
	}
	rows, err := tx.QueryContext(ctx, `
		select id from user_webhook_endpoints
		where project_id = $1 and status = 'active'
			and (events = '*' or (',' || events || ',') like $2)`, projectID, "%,"+eventType+",%")
	if err != nil {
		return err
	}
	defer rows.Close()
	endpointIDs := []string{}
	for rows.Next() {
		var endpointID string
		if err := rows.Scan(&endpointID); err != nil {
			return err
		}
		endpointIDs = append(endpointIDs, endpointID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, endpointID := range endpointIDs {
		if _, err := tx.ExecContext(ctx, `
			insert into user_webhook_deliveries(id, endpoint_id, project_id, event_type, payload, status)
			values($1,$2,$3,$4,$5,'pending')`, "delivery_"+randomHex(12), endpointID, projectID, eventType, string(raw)); err != nil {
			return err
		}
	}
	return nil
}

type claimedWebhookDelivery struct {
	ID           string
	EndpointID   string
	ProjectID    string
	EventType    string
	Payload      string
	AttemptCount int
	MaxAttempts  int
	URL          string
	SecretRef    string
}

func (a *App) ProcessNextWebhookDelivery(ctx context.Context) (bool, error) {
	delivery, err := a.claimWebhookDelivery(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	statusCode, responseBody, deliverErr := a.deliverWebhook(ctx, delivery)
	if deliverErr == nil {
		_, err = a.DB.ExecContext(ctx, `
			update user_webhook_deliveries
			set status = 'delivered', response_status = $2, response_body = $3,
				error_message = null, delivered_at = current_timestamp, updated_at = current_timestamp
			where id = $1 and status = 'delivering'`, delivery.ID, statusCode, truncateString(responseBody, 1900))
		return true, err
	}
	status := "retrying"
	if delivery.AttemptCount >= delivery.MaxAttempts {
		status = "failed"
	}
	backoff := webhookRetryDelay(delivery.AttemptCount)
	_, err = a.DB.ExecContext(ctx, `
		update user_webhook_deliveries
		set status = $2, response_status = $3, response_body = $4, error_message = $5,
			next_attempt_at = $6, updated_at = current_timestamp
		where id = $1 and status = 'delivering'`, delivery.ID, status, nullIfZero(statusCode), truncateString(responseBody, 1900), truncateString(deliverErr.Error(), 1900), time.Now().UTC().Add(backoff))
	return true, err
}

func (a *App) claimWebhookDelivery(ctx context.Context) (claimedWebhookDelivery, error) {
	tx, err := a.DB.BeginTx(ctx, nil)
	if err != nil {
		return claimedWebhookDelivery{}, err
	}
	defer tx.Rollback()
	var delivery claimedWebhookDelivery
	if _, err = tx.ExecContext(ctx, `
		update user_webhook_deliveries
		set status = case when attempt_count < max_attempts then 'retrying' else 'failed' end,
			error_message = 'webhook worker lease expired', next_attempt_at = current_timestamp,
			updated_at = current_timestamp
		where status = 'delivering' and updated_at < current_timestamp - interval '10 minutes'`); err != nil {
		return claimedWebhookDelivery{}, err
	}
	err = tx.QueryRowContext(ctx, `
		select d.id, d.endpoint_id, d.project_id, d.event_type, d.payload,
			d.attempt_count, d.max_attempts, e.url, coalesce(e.secret_ref, '')
		from user_webhook_deliveries d
		join user_webhook_endpoints e on e.id = d.endpoint_id
		where d.status in ('pending','retrying') and d.next_attempt_at <= current_timestamp
			and e.status = 'active'
		order by d.next_attempt_at asc, d.created_at asc
		for update of d skip locked limit 1`).Scan(
		&delivery.ID, &delivery.EndpointID, &delivery.ProjectID, &delivery.EventType, &delivery.Payload,
		&delivery.AttemptCount, &delivery.MaxAttempts, &delivery.URL, &delivery.SecretRef,
	)
	if err != nil {
		return claimedWebhookDelivery{}, err
	}
	delivery.AttemptCount++
	if _, err = tx.ExecContext(ctx, `
		update user_webhook_deliveries
		set status = 'delivering', attempt_count = $2, updated_at = current_timestamp
		where id = $1`, delivery.ID, delivery.AttemptCount); err != nil {
		return claimedWebhookDelivery{}, err
	}
	return delivery, tx.Commit()
}

func (a *App) deliverWebhook(ctx context.Context, delivery claimedWebhookDelivery) (int, string, error) {
	if err := validateWebhookURL(delivery.URL, a.Config.DevMode); err != nil {
		return 0, "", err
	}
	secret := strings.TrimSpace(os.Getenv(delivery.SecretRef))
	if secret == "" {
		return 0, "", errors.New("webhook_secret_not_configured")
	}
	timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	signedPayload := timestamp + "." + delivery.Payload
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, delivery.URL, bytes.NewBufferString(delivery.Payload))
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Model-Market-Webhook/1.0")
	req.Header.Set("X-Model-Market-Delivery", delivery.ID)
	req.Header.Set("X-Model-Market-Event", delivery.EventType)
	req.Header.Set("X-Model-Market-Signature", "t="+timestamp+",v1="+signature)
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2001))
	responseBody := string(body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, responseBody, fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return resp.StatusCode, responseBody, nil
}

func webhookRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := 5 * time.Second * time.Duration(1<<min(attempt-1, 10))
	if delay > time.Hour {
		return time.Hour
	}
	return delay
}

func nullIfZero(value int) any {
	if value == 0 {
		return nil
	}
	return value
}
