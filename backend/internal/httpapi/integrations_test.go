package httpapi

import (
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestFacebookOAuthStart(t *testing.T) {
	app, _, cleanup := testApp(t)
	defer cleanup()
	t.Setenv("FACEBOOK_CLIENT_ID", "facebook-client")
	t.Setenv("FACEBOOK_CLIENT_SECRET", "facebook-secret")
	t.Setenv("FACEBOOK_REDIRECT_URI", "http://localhost:8080/api/v1/auth/oauth/facebook/callback")
	t.Setenv("FACEBOOK_GRAPH_API_VERSION", "v23.0")
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/facebook/start", nil))
	location, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil || recorder.Code != http.StatusFound || location.Host != "www.facebook.com" || location.Query().Get("state") == "" || location.Query().Get("scope") != "email,public_profile" {
		t.Fatalf("status=%d location=%s", recorder.Code, location)
	}
	if cookies := recorder.Result().Cookies(); len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode {
		t.Fatalf("unsafe state cookie: %+v", cookies)
	}
}

func TestFacebookOAuthCallbackLinksVerifiedIdentity(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	t.Setenv("FACEBOOK_CLIENT_ID", "facebook-client")
	t.Setenv("FACEBOOK_CLIENT_SECRET", "facebook-secret")
	t.Setenv("FACEBOOK_REDIRECT_URI", "http://localhost:8080/api/v1/auth/oauth/facebook/callback")
	t.Setenv("FACEBOOK_GRAPH_API_VERSION", "v23.0")
	app.Config.PublicURL = "http://localhost:3000"
	app.Client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body := `{"access_token":"user-token"}`
		switch {
		case strings.Contains(req.URL.Path, "/debug_token"):
			body = `{"data":{"app_id":"facebook-client","is_valid":true,"user_id":"fb-123","expires_at":4102444800}}`
		case strings.HasSuffix(req.URL.Path, "/me"):
			body = `{"id":"fb-123","email":"person@example.com","name":"Person","picture":{"data":{"url":"https://example.com/avatar.jpg"}}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{"Content-Type": {"application/json"}}, Request: req}, nil
	})}
	mock.ExpectBegin()
	mock.ExpectQuery("select user_id from sys_oauth_accounts").WithArgs("fb-123").WillReturnError(sql.ErrNoRows)
	mock.ExpectQuery("select id from sys_users").WithArgs("person@example.com").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-1"))
	mock.ExpectExec("insert into sys_oauth_accounts").WithArgs(sqlmock.AnyArg(), "user-1", "fb-123", "person@example.com", "Person", "https://example.com/avatar.jpg").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectExec("insert into sys_oauth_login_codes").WithArgs(sqlmock.AnyArg(), "user-1", sqlmock.AnyArg(), sqlmock.AnyArg()).WillReturnResult(sqlmock.NewResult(0, 1))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/auth/oauth/facebook/callback?code=code&state=state", nil)
	request.AddCookie(&http.Cookie{Name: facebookStateCookie, Value: "state"})
	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusFound || !strings.Contains(recorder.Header().Get("Location"), "oauth_provider=facebook") || !strings.Contains(recorder.Header().Get("Location"), "oauth_code=") {
		t.Fatalf("status=%d location=%s", recorder.Code, recorder.Header().Get("Location"))
	}
}

func TestCreateStripeRefundUsesProviderAndIdempotency(t *testing.T) {
	app, _, cleanup := testApp(t)
	defer cleanup()
	t.Setenv("STRIPE_SECRET_KEY", "sk_test_secret")
	app.Client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != "https://api.stripe.com/v1/refunds" || req.Header.Get("Authorization") != "Bearer sk_test_secret" || req.Header.Get("Idempotency-Key") != "mm-refund-payment-1-0-500" {
			t.Fatalf("unexpected Stripe request")
		}
		raw, _ := io.ReadAll(req.Body)
		if !strings.Contains(string(raw), "payment_intent=pi_123") || !strings.Contains(string(raw), "amount=500") {
			t.Fatalf("form=%s", raw)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"id":"re_123","status":"succeeded"}`)), Header: http.Header{}, Request: req}, nil
	})}
	id, err := app.createStripeRefund(context.Background(), "pi_123", "payment-1", 0, 500)
	if err != nil || id != "re_123" {
		t.Fatalf("id=%s err=%v", id, err)
	}
}

type fakeObjectStore struct {
	putKey  string
	putBody []byte
}

func (f *fakeObjectStore) Put(_ context.Context, key string, body []byte, _ string) error {
	f.putKey = key
	f.putBody = body
	return nil
}
func (f *fakeObjectStore) Delete(context.Context, string) error { return nil }
func (f *fakeObjectStore) PresignPut(_ context.Context, key, _ string, _ time.Duration) (string, error) {
	return "https://s3.test/put/" + key, nil
}
func (f *fakeObjectStore) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	return "https://s3.test/get/" + key, nil
}

func TestS3ObjectStoreRouting(t *testing.T) {
	store := &fakeObjectStore{}
	app := &App{ObjectStore: store}
	app.Config.ObjectStorageProvider = "s3"
	app.Config.AssetBucket = "bucket"
	app.Config.S3PresignMinutes = 15
	upload, err := app.objectUploadURL(context.Background(), "project/file.png", "image/png")
	download, getErr := app.objectDownloadURL(context.Background(), "bucket", "project/file.png")
	size, putErr := app.writeGeneratedObject(context.Background(), "project/generated.svg", []byte("svg"), "image/svg+xml")
	if err != nil || getErr != nil || putErr != nil || upload != "https://s3.test/put/project/file.png" || download != "https://s3.test/get/project/file.png" || size != 3 || store.putKey != "project/generated.svg" {
		t.Fatalf("upload=%s download=%s size=%d", upload, download, size)
	}
}
