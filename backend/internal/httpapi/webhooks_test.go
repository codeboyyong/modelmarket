package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"model-market/backend/internal/config"
)

func TestValidateWebhookURL(t *testing.T) {
	for _, value := range []string{"https://hooks.example.com/model-market", "https://203.0.113.8/events"} {
		if err := validateWebhookURL(value, false); err != nil {
			t.Fatalf("expected %q to be valid: %v", value, err)
		}
	}
	for _, value := range []string{"http://hooks.example.com/events", "https://127.0.0.1/events", "https://10.0.0.5/events", "https://user:pass@example.com/events"} {
		if err := validateWebhookURL(value, false); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
	if err := validateWebhookURL("http://localhost:9999/events", true); err != nil {
		t.Fatalf("development localhost should be allowed: %v", err)
	}
}

func TestNormalizeWebhookEvents(t *testing.T) {
	events, err := normalizeWebhookEvents([]string{"async_job.failed", "async_job.completed", "async_job.failed"})
	if err != nil || strings.Join(events, ",") != "async_job.completed,async_job.failed" {
		t.Fatalf("events=%v err=%v", events, err)
	}
	if _, err := normalizeWebhookEvents([]string{"unknown.event"}); err == nil {
		t.Fatal("unknown event must be rejected")
	}
}

func TestCreateWebhookEndpoint(t *testing.T) {
	t.Setenv("WEBHOOK_TEST_SECRET", "secret-value")
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectExec("insert into user_webhook_endpoints").
		WithArgs(sqlmock.AnyArg(), "project-1", "http://example.test/hooks", "WEBHOOK_TEST_SECRET", "async_job.completed,async_job.failed").
		WillReturnResult(sqlmock.NewResult(0, 1))

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks", strings.NewReader(`{
		"project_id":"project-1","url":"http://example.test/hooks","secret_ref":"WEBHOOK_TEST_SECRET",
		"events":["async_job.failed","async_job.completed"]
	}`))
	app.Routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated || !strings.Contains(recorder.Body.String(), `"status":"active"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDeliverWebhookSignsPayload(t *testing.T) {
	t.Setenv("WEBHOOK_TEST_SECRET", "top-secret")
	payload := `{"id":"job-1","status":"completed"}`
	var received bool
	app := &App{
		Config: config.Config{DevMode: true},
		Client: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			received = true
			body, _ := io.ReadAll(request.Body)
			if string(body) != payload || request.Header.Get("X-Model-Market-Event") != "async_job.completed" {
				t.Fatalf("unexpected request body=%s headers=%v", body, request.Header)
			}
			parts := strings.Split(request.Header.Get("X-Model-Market-Signature"), ",")
			if len(parts) != 2 || !strings.HasPrefix(parts[0], "t=") || !strings.HasPrefix(parts[1], "v1=") {
				t.Fatalf("invalid signature header: %q", request.Header.Get("X-Model-Market-Signature"))
			}
			timestamp := strings.TrimPrefix(parts[0], "t=")
			mac := hmac.New(sha256.New, []byte("top-secret"))
			_, _ = mac.Write([]byte(timestamp + "." + payload))
			expected := hex.EncodeToString(mac.Sum(nil))
			if strings.TrimPrefix(parts[1], "v1=") != expected {
				t.Fatal("signature mismatch")
			}
			return &http.Response{StatusCode: http.StatusNoContent, Body: io.NopCloser(strings.NewReader("accepted")), Header: http.Header{}, Request: request}, nil
		})},
	}
	status, body, err := app.deliverWebhook(context.Background(), claimedWebhookDelivery{
		ID: "delivery-1", EventType: "async_job.completed", Payload: payload,
		URL: "http://example.test/hooks", SecretRef: "WEBHOOK_TEST_SECRET",
	})
	if err != nil || status != http.StatusNoContent || body != "accepted" || !received {
		t.Fatalf("status=%d body=%q received=%v err=%v", status, body, received, err)
	}
}

func TestProcessNextWebhookDeliveryMarksSuccess(t *testing.T) {
	t.Setenv("WEBHOOK_TEST_SECRET", "secret")
	app, mock, cleanup := testApp(t)
	defer cleanup()
	app.Client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok")), Header: http.Header{}, Request: request}, nil
	})}
	mock.ExpectBegin()
	mock.ExpectExec("webhook worker lease expired").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("from user_webhook_deliveries d").WillReturnRows(sqlmock.NewRows([]string{
		"id", "endpoint_id", "project_id", "event_type", "payload", "attempt_count", "max_attempts", "url", "secret_ref",
	}).AddRow("delivery-1", "webhook-1", "project-1", "async_job.completed", `{"id":"job-1"}`, 0, 6, "http://example.test/hooks", "WEBHOOK_TEST_SECRET"))
	mock.ExpectExec("update user_webhook_deliveries").WithArgs("delivery-1", 1).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("update user_webhook_deliveries").
		WithArgs("delivery-1", http.StatusOK, "ok").
		WillReturnResult(sqlmock.NewResult(0, 1))

	processed, err := app.ProcessNextWebhookDelivery(context.Background())
	if err != nil || !processed {
		t.Fatalf("processed=%v err=%v", processed, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessNextWebhookDeliverySchedulesRetry(t *testing.T) {
	t.Setenv("WEBHOOK_TEST_SECRET", "secret")
	app, mock, cleanup := testApp(t)
	defer cleanup()
	app.Client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusServiceUnavailable, Body: io.NopCloser(strings.NewReader("temporarily down")), Header: http.Header{}, Request: request}, nil
	})}
	mock.ExpectBegin()
	mock.ExpectExec("webhook worker lease expired").WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("from user_webhook_deliveries d").WillReturnRows(sqlmock.NewRows([]string{
		"id", "endpoint_id", "project_id", "event_type", "payload", "attempt_count", "max_attempts", "url", "secret_ref",
	}).AddRow("delivery-1", "webhook-1", "project-1", "async_job.failed", `{"id":"job-1"}`, 0, 6, "http://example.test/hooks", "WEBHOOK_TEST_SECRET"))
	mock.ExpectExec("update user_webhook_deliveries").WithArgs("delivery-1", 1).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("update user_webhook_deliveries").
		WithArgs("delivery-1", "retrying", http.StatusServiceUnavailable, "temporarily down", "webhook returned 503", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	processed, err := app.ProcessNextWebhookDelivery(context.Background())
	if err != nil || !processed {
		t.Fatalf("processed=%v err=%v", processed, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReplayWebhookDelivery(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectExec("update user_webhook_deliveries").WithArgs("delivery-1").WillReturnResult(sqlmock.NewResult(0, 1))
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/webhook-deliveries/delivery-1/replay", nil))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	assertJSONField(t, recorder.Body.Bytes(), "status", "pending")
}

func TestEnqueueProjectWebhooksCreatesOneDeliveryPerSubscriber(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectBegin()
	mock.ExpectQuery("select id from user_webhook_endpoints").
		WithArgs("project-1", "%,async_job.completed,%").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("webhook-1").AddRow("webhook-2"))
	mock.ExpectExec("insert into user_webhook_deliveries").WithArgs(sqlmock.AnyArg(), "webhook-1", "project-1", "async_job.completed", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_webhook_deliveries").WithArgs(sqlmock.AnyArg(), "webhook-2", "project-1", "async_job.completed", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	tx, err := app.DB.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = enqueueProjectWebhooks(context.Background(), tx, "project-1", "async_job.completed", response{"id": "job-1"}); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestWebhookRetryDelayIsBounded(t *testing.T) {
	if webhookRetryDelay(1) != 5*time.Second || webhookRetryDelay(20) != time.Hour {
		t.Fatalf("unexpected retry delays: first=%s max=%s", webhookRetryDelay(1), webhookRetryDelay(20))
	}
}
