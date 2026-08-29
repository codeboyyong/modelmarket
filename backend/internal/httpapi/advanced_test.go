package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestCreateOrganizationInvitation(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectExec("insert into user_organization_invitations").
		WithArgs(sqlmock.AnyArg(), "org-1", "person@example.com", "developer", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organization/invitations", bytes.NewBufferString(`{"organization_id":"org-1","email":"person@example.com","role":"developer"}`))
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusCreated || !strings.Contains(recorder.Body.String(), "invitation_token") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateOrganizationMemberRole(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectExec("update sys_memberships set role").WithArgs("analyst", "org-1", "user-1").WillReturnResult(sqlmock.NewResult(0, 1))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/organization/members", bytes.NewBufferString(`{"organization_id":"org-1","user_id":"user-1","role":"analyst"}`))
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListPayments(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("select id, provider").WithArgs("wallet-1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "provider", "provider_payment_id", "amount_cents", "refunded_amount_cents", "currency", "status", "created_at", "updated_at"}).
			AddRow("payment-1", "credit_card", "fake-1", int64(1000), int64(200), "USD", "partially_refunded", time.Now(), time.Now()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/payments?wallet_id=wallet-1", nil)
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "payment-1") {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRequestIDAndMetrics(t *testing.T) {
	app, _, cleanup := testApp(t)
	defer cleanup()
	health := httptest.NewRecorder()
	app.Routes().ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if !strings.HasPrefix(health.Header().Get("X-Request-ID"), "req_") {
		t.Fatalf("missing request id: %v", health.Header())
	}
	metrics := httptest.NewRecorder()
	app.Routes().ServeHTTP(metrics, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if metrics.Code != 200 || !strings.Contains(metrics.Body.String(), `model_market_http_requests_total{method="GET",path="/healthz"} 1`) {
		t.Fatalf("metrics=%s", metrics.Body.String())
	}
}

func TestLoginLocksAfterFiveFailures(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	for i := 0; i < 5; i++ {
		mock.ExpectQuery("select email, coalesce").WithArgs("person@example.com").WillReturnRows(sqlmock.NewRows([]string{"email", "password_hash"}).AddRow("person@example.com", hashPassword("right-password")))
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"person@example.com","password":"wrong-password"}`))
		recorder := httptest.NewRecorder()
		app.Routes().ServeHTTP(recorder, req)
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d status=%d", i+1, recorder.Code)
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"person@example.com","password":"wrong-password"}`))
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusTooManyRequests || recorder.Header().Get("Retry-After") == "" {
		t.Fatalf("status=%d body=%s headers=%v", recorder.Code, recorder.Body.String(), recorder.Header())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestUploadIntentRejectsUnsafeTypeAndLargeFile(t *testing.T) {
	app, _, cleanup := testApp(t)
	defer cleanup()
	for _, body := range []string{
		`{"project_id":"project-1","filename":"payload.svg","content_type":"image/svg+xml","size_bytes":10}`,
		`{"project_id":"project-1","filename":"large.png","content_type":"image/png","size_bytes":26214401}`,
	} {
		recorder := httptest.NewRecorder()
		app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/assets/upload-intent", bytes.NewBufferString(body)))
		if recorder.Code < 400 {
			t.Fatalf("body=%s status=%d", body, recorder.Code)
		}
	}
}

func TestAcceptOrganizationInvitation(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	token := "invite_secret"
	expires := time.Now().Add(time.Hour)
	mock.ExpectBegin()
	mock.ExpectQuery("select id, organization_id, email, role, expires_at").WithArgs(hashAPIKey(token)).WillReturnRows(sqlmock.NewRows([]string{"id", "organization_id", "email", "role", "expires_at"}).AddRow("invite-1", "org-1", "person@example.com", "developer", expires))
	mock.ExpectQuery("select email from sys_users").WithArgs("user-1").WillReturnRows(sqlmock.NewRows([]string{"email"}).AddRow("person@example.com"))
	mock.ExpectExec("insert into sys_memberships").WithArgs(sqlmock.AnyArg(), "user-1", "org-1", "developer").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("update user_organization_invitations").WithArgs("invite-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/organization/invitations/accept", bytes.NewBufferString(`{"token":"invite_secret","user_id":"user-1"}`)))
	if recorder.Code != 200 {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRefundPayment(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectBegin()
	mock.ExpectQuery("select wallet_id, amount_cents").WithArgs("payment-1").WillReturnRows(sqlmock.NewRows([]string{"wallet_id", "amount_cents", "refunded_amount_cents", "status"}).AddRow("wallet-1", int64(1000), int64(0), "succeeded"))
	mock.ExpectQuery("select conf_value from sys_config").WithArgs("usd_to_credit_ratio").WillReturnRows(sqlmock.NewRows([]string{"conf_value"}).AddRow("100"))
	mock.ExpectExec("update user_wallets set paid_credits").WithArgs(int64(500), "wallet-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("update user_payments set refunded_amount_cents").WithArgs(int64(500), "partially_refunded", "payment-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_ledger_transactions").WithArgs(sqlmock.AnyArg(), "wallet-1", int64(-500), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/payments/refund", bytes.NewBufferString(`{"payment_id":"payment-1","amount_cents":500}`)))
	if recorder.Code != 200 {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRenameConversation(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectExec("update user_conversations set title").WithArgs("New title", "conversation-1").WillReturnResult(sqlmock.NewResult(0, 1))
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodPatch, "/api/v1/conversations/conversation-1", bytes.NewBufferString(`{"title":"New title"}`)))
	if recorder.Code != 200 {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCreatePromptPreset(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectExec("insert into user_prompt_presets").WithArgs(sqlmock.AnyArg(), "project-1", "Portrait", "mock-image", "A portrait", `{"size":"1024x1024"}`).WillReturnResult(sqlmock.NewResult(0, 1))
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/prompt-presets", bytes.NewBufferString(`{"project_id":"project-1","name":"Portrait","model":"mock-image","prompt":"A portrait","parameters":{"size":"1024x1024"}}`)))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestDeleteAsset(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("select coalesce\\(object_key").WithArgs("asset-1").WillReturnRows(sqlmock.NewRows([]string{"object_key"}).AddRow(""))
	mock.ExpectBegin()
	mock.ExpectExec("delete from user_message_attachments").WithArgs("asset-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("delete from user_file_extractions").WithArgs("asset-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("delete from user_workbench_assets").WithArgs("asset-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodDelete, "/api/v1/assets/asset-1", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRequestLimit(t *testing.T) {
	app := &App{}
	if remaining, _, err := app.enforceRequestLimit("key-1", 2); err != nil || remaining != 1 {
		t.Fatalf("first request: remaining=%d err=%v", remaining, err)
	}
	if remaining, _, err := app.enforceRequestLimit("key-1", 2); err != nil || remaining != 0 {
		t.Fatalf("second request: remaining=%d err=%v", remaining, err)
	}
	if _, _, err := app.enforceRequestLimit("key-1", 2); err == nil || err.Error() != "rate_limit_exceeded" {
		t.Fatalf("third request err=%v", err)
	}
}

func TestChatMessageTextParts(t *testing.T) {
	message := chatMessage{Content: []any{
		map[string]any{"type": "text", "text": "hello"},
		map[string]any{"type": "image_url", "image_url": map[string]any{"url": "https://example.test/image.png"}},
		map[string]any{"type": "text", "text": "world"},
	}}
	if got := message.Text(); got != "hello\nworld" {
		t.Fatalf("Text()=%q", got)
	}
}

func TestWriteChatCompletionStream(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeChatCompletionStream(recorder, nil, "mock-chat", "hello", "req_1", upstreamChatResult{PromptTokens: 2, CompletionTokens: 1})
	if contentType := recorder.Header().Get("Content-Type"); contentType != "text/event-stream" {
		t.Fatalf("content type=%q", contentType)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `"object":"chat.completion.chunk"`) || !strings.Contains(body, "data: [DONE]") {
		t.Fatalf("unexpected stream: %s", body)
	}
}

func TestReserveProjectCreditsUsesPromotionalCreditsFirst(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectBegin()
	mock.ExpectQuery("select w.id, w.paid_credits, w.promotional_credits").WithArgs("project-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "paid_credits", "promotional_credits"}).AddRow("wallet-1", int64(10), int64(3)))
	mock.ExpectExec("update user_wallets set promotional_credits").WithArgs(int64(3), int64(2), "wallet-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_ledger_transactions").WithArgs(sqlmock.AnyArg(), "wallet-1", int64(-3), "reserve_promo_req_1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_ledger_transactions").WithArgs(sqlmock.AnyArg(), "wallet-1", int64(-2), "reserve_paid_req_1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	reservation, err := app.reserveProjectCredits(context.Background(), "project-1", "req_1", 5)
	if err != nil {
		t.Fatal(err)
	}
	if reservation.PromotionalCredits != 3 || reservation.PaidCredits != 2 {
		t.Fatalf("unexpected reservation: %+v", reservation)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReleaseProjectCreditsRestoresReservation(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectBegin()
	mock.ExpectExec("update user_wallets set promotional_credits").WithArgs(int64(3), int64(2), "wallet-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_ledger_transactions").WithArgs(sqlmock.AnyArg(), "wallet-1", int64(3), "release_promo_req_1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_ledger_transactions").WithArgs(sqlmock.AnyArg(), "wallet-1", int64(2), "release_paid_req_1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := app.releaseProjectCredits(context.Background(), "req_1", creditReservation{WalletID: "wallet-1", PromotionalCredits: 3, PaidCredits: 2}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCaptureProjectCreditsReleasesUnusedPaidCredits(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectBegin()
	mock.ExpectQuery("select paid_credits, promotional_credits").WithArgs("wallet-1").WillReturnRows(sqlmock.NewRows([]string{"paid_credits", "promotional_credits"}).AddRow(int64(10), int64(0)))
	mock.ExpectExec("update user_wallets set promotional_credits").WithArgs(int64(0), int64(3), "wallet-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_ledger_transactions").WithArgs(sqlmock.AnyArg(), "wallet-1", int64(3), "capture_release_paid_req_1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_ledger_transactions").WithArgs(sqlmock.AnyArg(), "wallet-1", "capture_req_1", sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	reservation := creditReservation{WalletID: "wallet-1", PaidCredits: 5}
	if err := app.captureProjectCredits(context.Background(), "req_1", reservation, 2); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCaptureProjectCreditsRejectsUnaffordableOverage(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectBegin()
	mock.ExpectQuery("select paid_credits, promotional_credits").WithArgs("wallet-1").WillReturnRows(sqlmock.NewRows([]string{"paid_credits", "promotional_credits"}).AddRow(int64(1), int64(0)))
	mock.ExpectRollback()
	err := app.captureProjectCredits(context.Background(), "req_1", creditReservation{WalletID: "wallet-1", PaidCredits: 2}, 4)
	if err == nil || err.Error() != "insufficient_credits" {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
