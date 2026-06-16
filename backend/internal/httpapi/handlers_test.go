package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"model-market/backend/internal/config"
)

type fakeRedis struct {
	err error
}

func (f fakeRedis) Ping(_ context.Context) error {
	return f.err
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testApp(t *testing.T) (*App, sqlmock.Sqlmock, func()) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	app := &App{
		Config: config.Config{DevMode: true},
		DB:     db,
		Redis:  fakeRedis{},
		Logger: slog.New(slog.NewTextHandler(os.Stdout, nil)),
	}
	cleanup := func() {
		db.Close()
	}
	return app, mock, cleanup
}

func TestHealth(t *testing.T) {
	app, _, cleanup := testApp(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "status", "ok")
}

func TestReady(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	mock.ExpectPing()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "status", "ready")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestReadyAllowsRedisDisabled(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	app.Redis = nil

	mock.ExpectPing()

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "status", "ready")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDevLogin(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	mock.ExpectQuery("select u.id, u.name, u.user_type").
		WithArgs("admin@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_type", "coalesce", "coalesce", "id", "id"}).
			AddRow("user-1", "Admin User", "sys_admin", "", "", "org-1", "project-1"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/dev-login", bytes.NewBufferString(`{"email":"admin@example.com"}`))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "project_id", "project-1")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPasswordLogin(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	mock.ExpectQuery("select email, coalesce").
		WithArgs("admin@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"email", "coalesce"}).AddRow("admin@example.com", hashPassword("dev-password")))
	mock.ExpectQuery("select u.id, u.name, u.user_type").
		WithArgs("admin@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_type", "coalesce", "coalesce", "id", "id"}).
			AddRow("user-1", "Admin User", "sys_admin", "", "", "org-1", "project-1"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin@example.com","password":"dev-password"}`))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "project_id", "project-1")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestDevSocialLogin(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	mock.ExpectQuery("select u.id, u.name, u.user_type").
		WithArgs("developer@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_type", "coalesce", "coalesce", "id", "id"}).
			AddRow("user-2", "Developer User", "individual_consumer", "", "", "org-1", "project-1"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/social/dev", bytes.NewBufferString(`{"provider":"facebook"}`))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "provider", "facebook")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestSignup(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	mock.ExpectExec("insert into sys_users").
		WithArgs(sqlmock.AnyArg(), "new@example.com", "New User", hashPassword("dev-password"), "individual_consumer", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into sys_memberships").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "org-demo").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery("select u.id, u.name, u.user_type").
		WithArgs("new@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_type", "coalesce", "coalesce", "id", "id"}).
			AddRow("user-new", "New User", "individual_consumer", "", "", "org-1", "project-1"))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/signup", bytes.NewBufferString(`{"email":"new@example.com","name":"New User","password":"dev-password"}`))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "project_id", "project-1")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestModels(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	mock.ExpectQuery("select m.id, m.slug, m.name").
		WillReturnRows(sqlmock.NewRows([]string{"id", "slug", "name", "name", "modality", "status", "coalesce", "coalesce"}).
			AddRow("model-1", "mock-chat", "Mock Chat", "Mock Provider", "chat", "public", "mock-chat-default", "Mock Chat Default"))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/models", nil)
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body map[string][]map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body["models"]) != 1 || body["models"][0]["slug"] != "mock-chat" {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAPIKey(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	mock.ExpectExec("insert into user_api_keys").
		WithArgs(sqlmock.AnyArg(), "project-1", "Test key", sqlmock.AnyArg(), sqlmock.AnyArg(), "models:read,chat:create").
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/api-keys", bytes.NewBufferString(`{"project_id":"project-1","name":"Test key"}`))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["api_key"] == "" || body["prefix"] == "" {
		t.Fatalf("missing api key fields: %v", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestChatCompletionsRequiresAPIKey(t *testing.T) {
	app, _, cleanup := testApp(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", bytes.NewBufferString(`{"model":"mock-chat","messages":[]}`))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestChatCompletions(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	apiKey := "mk_test"
	mock.ExpectQuery("select project_id from user_api_keys").
		WithArgs(hashAPIKey(apiKey)).
		WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow("project-1"))
	mock.ExpectQuery("select r.id, r.route_group, m.slug").
		WithArgs("default", "mock-chat").
		WillReturnRows(sqlmock.NewRows([]string{"id", "route_group", "slug", "coalesce", "upstream_model_id", "id", "slug", "id", "name", "channel_type", "coalesce", "coalesce", "priority", "weight", "coalesce"}).
			AddRow("route-mock-chat-primary", "default", "mock-chat", "profile-mock-chat-default", "mock-chat", "provider-mock", "mock-provider", "channel-mock-primary", "Mock Primary Channel", "openai_compatible", "mock://provider/default", "", int64(100), int64(100), int64(42)))
	mock.ExpectExec("insert into user_inference_requests").
		WithArgs(sqlmock.AnyArg(), "project-1", "mock-chat", "profile-mock-chat-default", "route-mock-chat-primary", "channel-mock-primary", "mock-provider", 1, sqlmock.AnyArg(), int64(1), int64(0), int64(1), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_provider_attempts").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "provider-mock", "channel-mock-primary", "route-mock-chat-primary", int64(42), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", bytes.NewBufferString(`{"model":"mock-chat","messages":[{"role":"user","content":"hello"}]}`))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if id, ok := body["id"].(string); !ok || !strings.HasPrefix(id, "req_") {
		t.Fatalf("unexpected request id: %v", body["id"])
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestChatCompletionsCallsGemini(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	t.Setenv("TEST_GEMINI_API_KEY", "test-gemini-key")
	app.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://gemini.test/models/gemini-test:generateContent" {
			t.Fatalf("unexpected upstream url: %s", r.URL.String())
		}
		if got := r.Header.Get("x-goog-api-key"); got != "test-gemini-key" {
			t.Fatalf("unexpected gemini api key header: %q", got)
		}
		var body struct {
			Contents []struct {
				Role  string `json:"role"`
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body.Contents) != 1 || body.Contents[0].Role != "user" || body.Contents[0].Parts[0].Text != "hello gemini" {
			t.Fatalf("unexpected gemini payload: %+v", body)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"real gemini response"}]},"finishReason":"STOP"}],"usageMetadata":{"promptTokenCount":7,"candidatesTokenCount":5,"totalTokenCount":12}}`)),
			Request:    r,
		}, nil
	})}

	apiKey := "mk_test"
	mock.ExpectQuery("select project_id from user_api_keys").
		WithArgs(hashAPIKey(apiKey)).
		WillReturnRows(sqlmock.NewRows([]string{"project_id"}).AddRow("project-1"))
	mock.ExpectQuery("select r.id, r.route_group, m.slug").
		WithArgs("default", "gemini-2-5-flash-free-default").
		WillReturnRows(sqlmock.NewRows([]string{"id", "route_group", "slug", "coalesce", "upstream_model_id", "id", "slug", "id", "name", "channel_type", "coalesce", "coalesce", "priority", "weight", "coalesce"}).
			AddRow("route-gemini-test", "default", "gemini-2.5-flash", "profile-gemini-test", "gemini-test", "provider-google-gemini", "google-gemini", "channel-gemini-free", "Gemini Free Channel", "google_gemini", "https://gemini.test", "TEST_GEMINI_API_KEY", int64(95), int64(100), int64(140)))
	mock.ExpectExec("insert into user_inference_requests").
		WithArgs(sqlmock.AnyArg(), "project-1", "gemini-2.5-flash", "profile-gemini-test", "route-gemini-test", "channel-gemini-free", "google-gemini", 7, 5, int64(1), int64(0), int64(1), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_provider_attempts").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "provider-google-gemini", "channel-gemini-free", "route-gemini-test", sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", bytes.NewBufferString(`{"model":"gemini-2-5-flash-free-default","messages":[{"role":"user","content":"hello gemini"}]}`))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if len(body.Choices) != 1 || body.Choices[0].Message.Content != "real gemini response" {
		t.Fatalf("unexpected response body: %s", rec.Body.String())
	}
	if body.Usage.PromptTokens != 7 || body.Usage.CompletionTokens != 5 || body.Usage.TotalTokens != 12 {
		t.Fatalf("unexpected usage: %+v", body.Usage)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPurchaseCreditsUsesConfiguredUSDToCreditRatio(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	mock.ExpectQuery("select id").
		WithArgs("user-yong-zhao").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-yong-zhao"))
	mock.ExpectQuery("select conf_value from sys_config").
		WithArgs("usd_to_credit_ratio").
		WillReturnRows(sqlmock.NewRows([]string{"conf_value"}).AddRow("100"))
	mock.ExpectQuery("select conf_value from sys_config").
		WithArgs("payment_provider_mode").
		WillReturnRows(sqlmock.NewRows([]string{"conf_value"}).AddRow("mock"))
	mock.ExpectQuery("select conf_value from sys_config").
		WithArgs("payment_mock_enabled").
		WillReturnRows(sqlmock.NewRows([]string{"conf_value"}).AddRow("true"))
	mock.ExpectQuery("select w.id").
		WithArgs("user-yong-zhao").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("select w.id").
		WithArgs("user-yong-zhao").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("wallet-demo"))
	mock.ExpectBegin()
	mock.ExpectExec("insert into user_credit_purchases").
		WithArgs(sqlmock.AnyArg(), "user-yong-zhao", int64(10000), int64(10000), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_payments").
		WithArgs(sqlmock.AnyArg(), "wallet-demo", "credit_card", sqlmock.AnyArg(), int64(10000), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_ledger_transactions").
		WithArgs(sqlmock.AnyArg(), "wallet-demo", int64(10000), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("update user_wallets").
		WithArgs(int64(10000), "wallet-demo").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/credits/purchase", bytes.NewBufferString(`{"user_id":"user-yong-zhao","amount_cents":10000,"payment_method":"credit_card"}`))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "credits", float64(10000))
	assertJSONField(t, rec.Body.Bytes(), "amount_cents", float64(10000))
	assertJSONField(t, rec.Body.Bytes(), "payment_provider_mode", "mock")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestPurchaseCreditsStripeModeCreatesCheckoutSession(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_local")
	app.Client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.String() != "https://api.stripe.com/v1/checkout/sessions" {
			t.Fatalf("unexpected stripe request: %s %s", r.Method, r.URL.String())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test_local" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		form, err := url.ParseQuery(string(raw))
		if err != nil {
			t.Fatal(err)
		}
		if form.Get("line_items[0][price_data][unit_amount]") != "2500" {
			t.Fatalf("unexpected stripe amount form: %s", string(raw))
		}
		if form.Get("metadata[credits]") != "2500" {
			t.Fatalf("unexpected stripe credits form: %s", string(raw))
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"cs_test_123","url":"https://checkout.stripe.test/session","payment_status":"unpaid"}`)),
			Request:    r,
		}, nil
	})}

	mock.ExpectQuery("select id").
		WithArgs("user-yong-zhao").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-yong-zhao"))
	mock.ExpectQuery("select conf_value from sys_config").
		WithArgs("usd_to_credit_ratio").
		WillReturnRows(sqlmock.NewRows([]string{"conf_value"}).AddRow("100"))
	mock.ExpectQuery("select conf_value from sys_config").
		WithArgs("payment_provider_mode").
		WillReturnRows(sqlmock.NewRows([]string{"conf_value"}).AddRow("stripe"))
	mock.ExpectQuery("select w.id").
		WithArgs("user-yong-zhao").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("select w.id").
		WithArgs("user-yong-zhao").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("wallet-demo"))
	mock.ExpectBegin()
	mock.ExpectExec("insert into user_credit_purchases").
		WithArgs(sqlmock.AnyArg(), "user-yong-zhao", int64(2500), int64(2500), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_payments").
		WithArgs(sqlmock.AnyArg(), "wallet-demo", "cs_test_123", int64(2500), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/credits/purchase", bytes.NewBufferString(`{"user_id":"user-yong-zhao","amount_cents":2500,"payment_method":"credit_card"}`))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "credits", float64(2500))
	assertJSONField(t, rec.Body.Bytes(), "amount_cents", float64(2500))
	assertJSONField(t, rec.Body.Bytes(), "payment_provider_mode", "stripe")
	assertJSONField(t, rec.Body.Bytes(), "checkout_url", "https://checkout.stripe.test/session")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestStripeWebhookPostsCredits(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	t.Setenv("STRIPE_WEBHOOK_SECRET", "whsec_local")

	payload := []byte(`{"id":"evt_123","type":"checkout.session.completed","data":{"object":{"id":"cs_test_123","payment_status":"paid","payment_intent":"pi_123","metadata":{"purchase_id":"purchase_123","wallet_id":"wallet-demo","user_id":"user-yong-zhao","credits":"2500","amount_cents":"2500"}}}}`)
	mock.ExpectQuery("select id from user_ledger_transactions").
		WithArgs("stripe_checkout_cs_test_123").
		WillReturnError(sql.ErrNoRows)
	mock.ExpectBegin()
	mock.ExpectExec("update user_credit_purchases").
		WithArgs("purchase_123", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("update user_payments").
		WithArgs("cs_test_123", "pi_123", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_ledger_transactions").
		WithArgs(sqlmock.AnyArg(), "wallet-demo", int64(2500), "stripe_checkout_cs_test_123", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("update user_wallets").
		WithArgs(int64(2500), "wallet-demo").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/stripe/webhook", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", stripeTestSignature(payload, "whsec_local"))
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "status", "posted")
	assertJSONField(t, rec.Body.Bytes(), "purchase_id", "purchase_123")
	assertJSONField(t, rec.Body.Bytes(), "credits", float64(2500))
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAPIKeys(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("select id, name, prefix, status").
		WithArgs("project-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "prefix", "status", "created_at", "revoked_at"}).
			AddRow("key-1", "Test key", "mk_123", "active", now, sql.NullTime{}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/api-keys?project_id=project-1", nil)
	rec := httptest.NewRecorder()

	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func stripeTestSignature(payload []byte, secret string) string {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(payload)
	return "t=" + timestamp + ",v1=" + hex.EncodeToString(mac.Sum(nil))
}

func assertJSONField(t *testing.T, raw []byte, key string, expected any) {
	t.Helper()
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatal(err)
	}
	if body[key] != expected {
		t.Fatalf("%s = %v, want %v; body=%s", key, body[key], expected, raw)
	}
}
