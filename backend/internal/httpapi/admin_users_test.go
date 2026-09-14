package httpapi

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestChangePasswordRequiresCurrentPassword(t *testing.T) {
	app, _, cleanup := testApp(t)
	defer cleanup()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewBufferString(`{"username":"user@example.com","password":"New-password1"}`))
	rec := httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "error", "missing_credentials")
}

func TestChangePasswordRejectsWrongCurrentPassword(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	mock.ExpectQuery("select email, coalesce").WithArgs("user@example.com").
		WillReturnRows(sqlmock.NewRows([]string{"email", "password_hash"}).AddRow("user@example.com", hashPassword("Current-password1")))
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/change-password", bytes.NewBufferString(`{"username":"user@example.com","current_password":"wrong","password":"New-password1"}`))
	rec := httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "error", "invalid_credentials")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAdminUsersRejectsNonAdmin(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	mock.ExpectQuery("select user_type, status from sys_users").WithArgs("user-1").
		WillReturnRows(sqlmock.NewRows([]string{"user_type", "status"}).AddRow("individual_consumer", "active"))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?user_id=user-1", nil)
	rec := httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	assertJSONField(t, rec.Body.Bytes(), "error", "admin_required")
}

func TestAdminUsersListsAccounts(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()

	mock.ExpectQuery("select user_type, status from sys_users").WithArgs("user-admin").
		WillReturnRows(sqlmock.NewRows([]string{"user_type", "status"}).AddRow("sys_admin", "active"))
	mock.ExpectQuery("select id, email, name, user_type, status, created_at").WithArgs("%%").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "name", "user_type", "status", "created_at"}).
			AddRow("user-1", "one@example.com", "One", "individual_consumer", "active", time.Now()))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/users?user_id=user-admin", nil)
	rec := httptest.NewRecorder()
	app.Routes().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"email":"one@example.com"`)) {
		t.Fatalf("expected user email in body: %s", rec.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAccountPassword(t *testing.T) {
	for _, password := range []string{"short", "onlyletters", "letters123", "12345678!"} {
		if validateAccountPassword(password) == nil {
			t.Fatalf("expected %q to be rejected", password)
		}
	}
	if err := validateAccountPassword("Secure-password1"); err != nil {
		t.Fatalf("strong password rejected: %v", err)
	}
}
