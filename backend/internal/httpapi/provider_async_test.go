package httpapi

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

const asyncProviderTestSettings = `{
  "async_submit_path":"v1/jobs",
  "async_status_path":"v1/jobs/{job_id}",
  "async_output_field":"result.url",
  "async_poll_interval_seconds":3,
  "async_max_poll_seconds":600
}`

func TestAsyncProviderSubmitPersistsPendingJob(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("select settings from sys_provider_channels").WithArgs("channel-async").
		WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(asyncProviderTestSettings))
	app.Client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.URL.Path != "/api/v1/jobs" || request.Header.Get("Idempotency-Key") != "model-market-job-1" {
			t.Fatalf("unexpected submit request: %s %s headers=%v", request.Method, request.URL, request.Header)
		}
		return &http.Response{StatusCode: 202, Body: io.NopCloser(strings.NewReader(`{"id":"provider-1","status":"queued","progress":20}`)), Header: http.Header{}, Request: request}, nil
	})}
	mock.ExpectExec("update user_async_jobs").
		WithArgs("job-1", "provider-1", "queued", 20, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	_, pending, err := app.runAsyncProviderStep(context.Background(), selectedModelRoute{
		ID: "route-1", ChannelID: "channel-async", BaseURL: "https://provider.test/api/", UpstreamModelID: "video-v1",
	}, generationJobPayload{Prompt: "ocean sunrise", Parameters: response{"duration_seconds": 5}}, claimedAsyncJob{ID: "job-1"})
	if err != nil || !pending {
		t.Fatalf("pending=%v err=%v", pending, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAsyncProviderPollReturnsCompletedOutput(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("select settings from sys_provider_channels").WithArgs("channel-async").
		WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(asyncProviderTestSettings))
	app.Client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/v1/jobs/provider-1" {
			t.Fatalf("unexpected poll request: %s %s", request.Method, request.URL)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{
			"id":"provider-1","status":"completed","result":{"url":"https://cdn.test/video.mp4"},
			"usage":{"prompt_tokens":7,"completion_tokens":3}
		}`)), Header: http.Header{"X-Request-Id": []string{"provider-request-1"}}, Request: request}, nil
	})}

	result, pending, err := app.runAsyncProviderStep(context.Background(), selectedModelRoute{
		ID: "route-1", ChannelID: "channel-async", BaseURL: "https://provider.test/api/", UpstreamModelID: "video-v1",
	}, generationJobPayload{Prompt: "ocean sunrise"}, claimedAsyncJob{
		ID: "job-1", ProviderJobID: "provider-1", StartedAt: sql.NullTime{Time: time.Now().Add(-time.Minute), Valid: true},
	})
	if err != nil || pending || result.Content != "https://cdn.test/video.mp4" || result.PromptTokens != 7 || result.CompletionTokens != 3 || result.ProviderRequestID != "provider-request-1" {
		t.Fatalf("result=%+v pending=%v err=%v", result, pending, err)
	}
}

func TestAsyncProviderPollTimeout(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("select settings from sys_provider_channels").WithArgs("channel-async").
		WillReturnRows(sqlmock.NewRows([]string{"settings"}).AddRow(asyncProviderTestSettings))
	_, _, err := app.runAsyncProviderStep(context.Background(), selectedModelRoute{ChannelID: "channel-async"}, generationJobPayload{}, claimedAsyncJob{
		ProviderJobID: "provider-1", StartedAt: sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
	})
	if err == nil || err.Error() != "async_provider_poll_timeout" {
		t.Fatalf("err=%v", err)
	}
}
