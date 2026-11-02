package httpapi

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func asyncJobRows() *sqlmock.Rows {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	return sqlmock.NewRows([]string{
		"id", "project_id", "actor_user_id", "job_type", "status", "progress",
		"model_slug", "provider_slug", "metadata", "result_metadata", "error_code",
		"error_message", "attempt_count", "max_attempts", "cancel_requested",
		"started_at", "completed_at", "created_at", "updated_at",
	}).AddRow(
		"job-1", "project-1", "", "file_extraction", "queued", 0,
		"", "", `{"asset_id":"asset-1"}`, `{}`, "", "",
		0, 3, false, sql.NullTime{}, sql.NullTime{}, now, now,
	)
}

func TestCreateAsyncJob(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("insert into user_async_jobs").
		WithArgs(sqlmock.AnyArg(), "project-1", nil, "file_extraction", nil, nil, `{"asset_id":"asset-1"}`, 3, "request-1").
		WillReturnRows(asyncJobRows())

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/async-jobs", bytes.NewBufferString(`{
		"project_id":"project-1","job_type":"file_extraction",
		"idempotency_key":"request-1","metadata":{"asset_id":"asset-1"}
	}`))
	app.Routes().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	for _, expected := range []string{`"id":"job-1"`, `"status":"queued"`, `"asset_id":"asset-1"`, `"max_attempts":3`} {
		if !bytes.Contains(recorder.Body.Bytes(), []byte(expected)) {
			t.Fatalf("response missing %s: %s", expected, recorder.Body.String())
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAsyncGenerationJob(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	rows := sqlmock.NewRows([]string{
		"id", "project_id", "actor_user_id", "job_type", "status", "progress",
		"model_slug", "provider_slug", "metadata", "result_metadata", "error_code",
		"error_message", "attempt_count", "max_attempts", "cancel_requested",
		"started_at", "completed_at", "created_at", "updated_at",
	}).AddRow(
		"job-image", "project-1", "", "image_generation", "queued", 0,
		"mock-image", "mock-provider", `{"prompt":"draw a fox"}`, `{}`, "", "",
		0, 3, false, sql.NullTime{}, sql.NullTime{}, now, now,
	)
	mock.ExpectQuery("insert into user_async_jobs").
		WithArgs(sqlmock.AnyArg(), "project-1", nil, "image_generation", "mock-image", "mock-provider", `{"prompt":"draw a fox"}`, 3, "image-request-1").
		WillReturnRows(rows)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/async-jobs", bytes.NewBufferString(`{
		"project_id":"project-1","job_type":"image_generation","model":"mock-image",
		"provider":"mock-provider","idempotency_key":"image-request-1","metadata":{"prompt":"draw a fox"}
	}`))
	app.Routes().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted || !bytes.Contains(recorder.Body.Bytes(), []byte(`"id":"job-image"`)) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestListAsyncJobs(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("from user_async_jobs where project_id = \\$1").
		WithArgs("project-1", 50).
		WillReturnRows(asyncJobRows())

	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/async-jobs?project_id=project-1", nil))
	if recorder.Code != http.StatusOK || !bytes.Contains(recorder.Body.Bytes(), []byte(`"id":"job-1"`)) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCancelAsyncJob(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectBegin()
	mock.ExpectQuery("update user_async_jobs").WithArgs("job-1").
		WillReturnRows(sqlmock.NewRows([]string{"status", "project_id", "job_type"}).AddRow("cancelled", "project-1", "file_extraction"))
	expectNoWebhookEndpoints(mock, "project-1", "async_job.cancelled")
	mock.ExpectCommit()

	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/async-jobs/job-1/cancel", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	assertJSONField(t, recorder.Body.Bytes(), "status", "cancelled")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestRetryAsyncJob(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("update user_async_jobs").WithArgs("job-1").
		WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("queued"))

	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/v1/async-jobs/job-1/retry", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	assertJSONField(t, recorder.Body.Bytes(), "status", "queued")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAsyncJobRejectsUnsupportedOrIncompleteJobs(t *testing.T) {
	app, _, cleanup := testApp(t)
	defer cleanup()
	for _, test := range []struct {
		name string
		body string
		code string
	}{
		{"incomplete media executor", `{"project_id":"project-1","job_type":"image_generation","metadata":{}}`, "generation_job_requires_model_and_prompt"},
		{"missing extraction asset", `{"project_id":"project-1","job_type":"file_extraction","metadata":{}}`, "file_extraction_requires_asset_id"},
	} {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/api/v1/async-jobs", bytes.NewBufferString(test.body))
			app.Routes().ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest || !bytes.Contains(recorder.Body.Bytes(), []byte(test.code)) {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestAuthorizeTenantAsyncJob(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("from user_async_jobs j join user_projects p").
		WithArgs("job-1", "user-1").
		WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/async-jobs/job-1", nil)
	if err := app.authorizeTenantRequest(request, authenticatedUser{ID: "user-1"}); err != nil {
		t.Fatal(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestAssetExtractionReturnsCompletedText(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	now := time.Date(2026, 11, 1, 13, 0, 0, 0, time.UTC)
	mock.ExpectQuery("from user_file_extractions").WithArgs("asset-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "extracted_text", "metadata", "created_at"}).
			AddRow("extract-1", "hello", `{"mime_type":"text/plain"}`, now))

	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/assets/asset-1/extraction", nil))
	if recorder.Code != http.StatusOK || !bytes.Contains(recorder.Body.Bytes(), []byte(`"text":"hello"`)) {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAssetExtractionReturnsPendingJobStatus(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectQuery("from user_file_extractions").WithArgs("asset-1").
		WillReturnRows(sqlmock.NewRows([]string{"id", "extracted_text", "metadata", "created_at"}))
	mock.ExpectQuery("from user_async_jobs").WithArgs(`%"asset_id":"asset-1"%`).
		WillReturnRows(sqlmock.NewRows([]string{"status", "error_message"}).AddRow("running", ""))

	recorder := httptest.NewRecorder()
	app.Routes().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/assets/asset-1/extraction", nil))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	assertJSONField(t, recorder.Body.Bytes(), "status", "running")
}
