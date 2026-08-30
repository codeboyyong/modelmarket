package httpapi

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestGoogleOAuthStartUsesStateNonceAndPKCE(t *testing.T) {
	app, _, cleanup := testApp(t)
	defer cleanup()
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "secret")
	t.Setenv("GOOGLE_REDIRECT_URI", "http://localhost:8080/api/v1/auth/oauth/google/callback")
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/google/start", nil))
	if recorder.Code != http.StatusFound {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	location, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	query := location.Query()
	if location.Scheme != "https" || query.Get("client_id") != "client-id" || query.Get("scope") != "openid email profile" || query.Get("code_challenge_method") != "S256" || query.Get("state") == "" || query.Get("nonce") == "" || query.Get("code_challenge") == "" {
		t.Fatalf("unexpected authorization URL: %s", location)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 3 {
		t.Fatalf("cookies=%d", len(cookies))
	}
	for _, cookie := range cookies {
		if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.MaxAge != 600 {
			t.Fatalf("unsafe cookie: %+v", cookie)
		}
	}
}

func TestDevelopmentLoginEndpointsAreDisabledOutsideDevMode(t *testing.T) {
	app, _, cleanup := testApp(t)
	defer cleanup()
	app.Config.DevMode = false
	for _, path := range []string{"/api/v1/auth/dev-login", "/api/v1/auth/social/dev"} {
		recorder := httptest.NewRecorder()
		app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`)))
		if recorder.Code != http.StatusNotFound {
			t.Fatalf("path=%s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestVerifyGoogleIDTokenChecksSignatureAndClaims(t *testing.T) {
	app, _, cleanup := testApp(t)
	defer cleanup()
	privateKey, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		t.Fatal(err)
	}
	kid := "test-key"
	jwks, _ := json.Marshal(response{"keys": []response{{"kid": kid, "kty": "RSA", "alg": "RS256", "n": base64.RawURLEncoding.EncodeToString(privateKey.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(privateKey.E)).Bytes())}}})
	app.Client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != googleJWKSEndpoint {
			t.Fatalf("unexpected URL: %s", req.URL)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(string(jwks))), Header: http.Header{"Content-Type": []string{"application/json"}}, Request: req}, nil
	})}
	token := signedGoogleTestToken(t, privateKey, kid, googleIDClaims{Issuer: "https://accounts.google.com", Subject: "google-user-1", Audience: "client-id", Email: "Person@Example.com", EmailVerified: true, Name: "Person", Nonce: "nonce-1", IssuedAt: time.Now().Add(-time.Minute).Unix(), ExpiresAt: time.Now().Add(time.Hour).Unix()})
	claims, err := app.verifyGoogleIDToken(context.Background(), token, "client-id", "nonce-1")
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "google-user-1" || claims.Email != "person@example.com" {
		t.Fatalf("claims=%+v", claims)
	}
	if _, err := app.verifyGoogleIDToken(context.Background(), token, "client-id", "wrong-nonce"); err == nil {
		t.Fatal("expected nonce rejection")
	}
}

func TestOAuthLoginExchangeConsumesOneTimeCode(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	code := "oauth_test_code"
	mock.ExpectBegin()
	mock.ExpectQuery("select c.user_id,u.email").WithArgs(sessionTokenHash(code)).WillReturnRows(sqlmock.NewRows([]string{"user_id", "email"}).AddRow("user-1", "person@example.com"))
	mock.ExpectExec("delete from sys_oauth_login_codes").WithArgs(sessionTokenHash(code)).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("select u.id, u.name, u.user_type").WithArgs("person@example.com").WillReturnRows(sqlmock.NewRows([]string{"id", "name", "user_type", "company_id", "company_name", "org_id", "project_id"}).AddRow("user-1", "Person", "individual_consumer", "", "", "org-1", "project-1"))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/oauth/exchange", strings.NewReader(`{"code":"oauth_test_code"}`))
	app.Routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"provider":"google"`) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLinkGoogleIdentityLinksVerifiedEmailToExistingUser(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	claims := googleIDClaims{Subject: "google-subject", Email: "person@example.com", EmailVerified: true, Name: "Person", Picture: "https://example.com/avatar.png"}
	mock.ExpectBegin()
	mock.ExpectQuery("select user_id from sys_oauth_accounts").WithArgs("google-subject").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("select id from sys_users").WithArgs("person@example.com").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-1"))
	mock.ExpectExec("insert into sys_oauth_accounts").WithArgs(sqlmock.AnyArg(), "user-1", "google-subject", "person@example.com", "Person", "https://example.com/avatar.png").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	userID, err := app.linkGoogleIdentity(context.Background(), claims)
	if err != nil {
		t.Fatal(err)
	}
	if userID != "user-1" {
		t.Fatalf("userID=%s", userID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func signedGoogleTestToken(t *testing.T, key *rsa.PrivateKey, kid string, claims googleIDClaims) string {
	t.Helper()
	headerRaw, _ := json.Marshal(response{"alg": "RS256", "kid": kid, "typ": "JWT"})
	claimsRaw, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(headerRaw) + "." + base64.RawURLEncoding.EncodeToString(claimsRaw)
	digest := sha256.Sum256([]byte(unsigned))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(signature)
}
