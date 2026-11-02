package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type asyncProviderSettings struct {
	SubmitPath          string   `json:"async_submit_path"`
	StatusPath          string   `json:"async_status_path"`
	IDField             string   `json:"async_id_field"`
	StatusField         string   `json:"async_status_field"`
	OutputField         string   `json:"async_output_field"`
	PromptTokensField   string   `json:"async_prompt_tokens_field"`
	OutputTokensField   string   `json:"async_output_tokens_field"`
	SucceededValues     []string `json:"async_succeeded_values"`
	FailedValues        []string `json:"async_failed_values"`
	PollIntervalSeconds int      `json:"async_poll_interval_seconds"`
	MaxPollSeconds      int      `json:"async_max_poll_seconds"`
}

func (a *App) runAsyncProviderStep(ctx context.Context, route selectedModelRoute, payload generationJobPayload, job claimedAsyncJob) (upstreamChatResult, bool, error) {
	settings, err := a.asyncProviderSettings(ctx, route.ChannelID)
	if err != nil {
		return upstreamChatResult{}, false, err
	}
	if job.StartedAt.Valid && time.Since(job.StartedAt.Time) > time.Duration(settings.MaxPollSeconds)*time.Second {
		return upstreamChatResult{}, false, errors.New("async_provider_poll_timeout")
	}
	endpoint := settings.SubmitPath
	method := http.MethodPost
	var body any = response{"model": route.UpstreamModelID, "prompt": payload.Prompt, "parameters": payload.Parameters}
	if job.ProviderJobID != "" {
		endpoint = strings.ReplaceAll(settings.StatusPath, "{job_id}", url.PathEscape(job.ProviderJobID))
		method = http.MethodGet
		body = nil
	}
	decoded, latency, requestID, err := a.callAsyncProvider(ctx, route, method, endpoint, body, job.ID)
	if err != nil {
		return upstreamChatResult{}, false, err
	}
	providerJobID := job.ProviderJobID
	if providerJobID == "" {
		providerJobID = stringField(decoded, settings.IDField)
		if providerJobID == "" {
			return upstreamChatResult{}, false, errors.New("async_provider_missing_job_id")
		}
	}
	providerStatus := strings.ToLower(stringField(decoded, settings.StatusField))
	if providerStatus == "" {
		providerStatus = "pending"
	}
	if stringInSet(providerStatus, settings.FailedValues) {
		return upstreamChatResult{}, false, fmt.Errorf("async_provider_failed: %s", providerStatus)
	}
	if !stringInSet(providerStatus, settings.SucceededValues) {
		progress := int(numberField(decoded, "progress"))
		if progress < 5 {
			progress = 10
		}
		if progress > 95 {
			progress = 95
		}
		_, err = a.DB.ExecContext(ctx, `
			update user_async_jobs
			set status = 'provider_pending', provider_job_id = $2, provider_status = $3,
				progress = $4, next_provider_poll_at = $5, updated_at = current_timestamp
			where id = $1 and status = 'running'`,
			job.ID, providerJobID, providerStatus, progress, time.Now().UTC().Add(time.Duration(settings.PollIntervalSeconds)*time.Second))
		return upstreamChatResult{}, true, err
	}
	content := stringField(decoded, settings.OutputField)
	if content == "" {
		return upstreamChatResult{}, false, errors.New("async_provider_missing_output")
	}
	return upstreamChatResult{
		Content: content, PromptTokens: int(numberField(decoded, settings.PromptTokensField)),
		CompletionTokens: int(numberField(decoded, settings.OutputTokensField)),
		LatencyMS:        latency, ProviderRequestID: firstNonEmpty(requestID, providerJobID),
	}, false, nil
}

func (a *App) asyncProviderSettings(ctx context.Context, channelID string) (asyncProviderSettings, error) {
	var raw string
	if err := a.DB.QueryRowContext(ctx, `select settings from sys_provider_channels where id = $1`, channelID).Scan(&raw); err != nil {
		return asyncProviderSettings{}, err
	}
	settings := asyncProviderSettings{
		IDField: "id", StatusField: "status", OutputField: "output",
		PromptTokensField: "usage.prompt_tokens", OutputTokensField: "usage.completion_tokens",
		SucceededValues: []string{"completed", "succeeded"}, FailedValues: []string{"failed", "cancelled", "canceled"},
		PollIntervalSeconds: 2, MaxPollSeconds: 900,
	}
	if json.Unmarshal([]byte(raw), &settings) != nil || settings.SubmitPath == "" || settings.StatusPath == "" {
		return asyncProviderSettings{}, errors.New("invalid_async_provider_settings")
	}
	if settings.PollIntervalSeconds < 1 || settings.PollIntervalSeconds > 60 || settings.MaxPollSeconds < 10 || settings.MaxPollSeconds > 86400 {
		return asyncProviderSettings{}, errors.New("invalid_async_provider_poll_limits")
	}
	return settings, nil
}

func (a *App) callAsyncProvider(ctx context.Context, route selectedModelRoute, method, endpoint string, payload any, idempotencyKey string) (map[string]any, int64, string, error) {
	base, err := url.Parse(strings.TrimRight(route.BaseURL, "/") + "/")
	if err != nil {
		return nil, 0, "", err
	}
	reference, err := url.Parse(strings.TrimLeft(endpoint, "/"))
	if err != nil {
		return nil, 0, "", err
	}
	target := base.ResolveReference(reference).String()
	var body io.Reader
	if payload != nil {
		raw, marshalErr := json.Marshal(payload)
		if marshalErr != nil {
			return nil, 0, "", marshalErr
		}
		body = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, 0, "", err
	}
	credentialRef := strings.TrimSpace(route.CredentialRef)
	if credentialRef != "" {
		credential, _, credentialErr := a.providerCredential(credentialRef)
		if credentialErr != nil {
			return nil, 0, "", credentialErr
		}
		req.Header.Set("Authorization", "Bearer "+credential)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "model-market-"+idempotencyKey)
	client := a.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	started := time.Now()
	resp, err := client.Do(req)
	latency := time.Since(started).Milliseconds()
	if err != nil {
		return nil, latency, "", err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, latency, "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, latency, resp.Header.Get("x-request-id"), fmt.Errorf("async_provider_http_%d", resp.StatusCode)
	}
	decoded := map[string]any{}
	if json.Unmarshal(raw, &decoded) != nil {
		return nil, latency, resp.Header.Get("x-request-id"), errors.New("invalid_async_provider_response")
	}
	return decoded, latency, resp.Header.Get("x-request-id"), nil
}

func nestedField(value map[string]any, path string) any {
	if path == "" {
		return nil
	}
	var current any = value
	for _, part := range strings.Split(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current = object[part]
	}
	return current
}

func stringField(value map[string]any, path string) string {
	result, _ := nestedField(value, path).(string)
	return result
}

func numberField(value map[string]any, path string) float64 {
	result, _ := nestedField(value, path).(float64)
	return result
}

func stringInSet(value string, options []string) bool {
	for _, option := range options {
		if value == strings.ToLower(option) {
			return true
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
