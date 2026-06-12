package httpapi

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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
		WillReturnRows(sqlmock.NewRows([]string{"id", "route_group", "slug", "coalesce", "upstream_model_id", "id", "slug", "id", "name", "priority", "weight", "coalesce"}).
			AddRow("route-mock-chat-primary", "default", "mock-chat", "profile-mock-chat-default", "mock-chat", "provider-mock", "mock-provider", "channel-mock-primary", "Mock Primary Channel", int64(100), int64(100), int64(42)))
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
