package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"golang.org/x/crypto/bcrypt"
)

func TestVerifyPasswordSupportsBcryptAndLegacyDevHashes(t *testing.T) {
	bcryptHash, err := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if !verifyPassword(string(bcryptHash), "correct") || verifyPassword(string(bcryptHash), "wrong") {
		t.Fatal("bcrypt verification failed")
	}
	if !verifyPassword(hashPassword("correct"), "correct") || verifyPassword(hashPassword("correct"), "wrong") {
		t.Fatal("legacy verification failed")
	}
}

func TestProductionSessionScopesProjectListing(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	app.Config.DevMode = false
	token := "sess_test"
	mock.ExpectQuery("select u.id, u.email, u.user_type").WithArgs(sessionTokenHash(token)).WillReturnRows(sqlmock.NewRows([]string{"id", "email", "user_type"}).AddRow("user-1", "one@example.com", "individual_consumer"))
	mock.ExpectQuery("select p.id, p.name").WithArgs("user-1").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "org_id", "org_name", "wallet_id", "paid", "promo", "used"}).AddRow("project-1", "One", "org-1", "Org", "wallet-1", int64(10), int64(2), int64(1)))
	req := httptest.NewRequest(http.MethodGet, "/api/v1/projects", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"project-1"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
