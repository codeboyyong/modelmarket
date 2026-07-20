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

func TestSettleProjectChargeUsesPromotionalCreditsFirst(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectBegin()
	mock.ExpectQuery("select id, paid_credits, promotional_credits").WithArgs("project-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "paid_credits", "promotional_credits"}).AddRow("wallet-1", int64(10), int64(3)))
	mock.ExpectExec("update user_wallets set promotional_credits").WithArgs(int64(3), int64(2), "wallet-1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_ledger_transactions").WithArgs(sqlmock.AnyArg(), "wallet-1", int64(-3), "usage_promo_req_1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_ledger_transactions").WithArgs(sqlmock.AnyArg(), "wallet-1", int64(-2), "usage_paid_req_1").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	if err := app.settleProjectCharge(context.Background(), "project-1", "req_1", 5); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
