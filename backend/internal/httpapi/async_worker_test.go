package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func expectAsyncJobClaim(mock sqlmock.Sqlmock, metadata string, attemptCount, maxAttempts int) {
	mock.ExpectBegin()
	mock.ExpectExec("update user_async_jobs").
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("select id, project_id, coalesce\\(actor_user_id").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "project_id", "actor_user_id", "job_type", "model_slug", "provider_slug",
			"metadata", "attempt_count", "max_attempts", "inference_request_id",
			"reservation_wallet_id", "reservation_promotional_credits", "reservation_paid_credits", "billing_status",
			"status", "provider_job_id", "provider_status", "started_at",
		}).AddRow(
			"job-1", "project-1", "", "file_extraction", "", "", metadata,
			attemptCount, maxAttempts, "", "", 0, 0, "unreserved", "queued", "", "", sql.NullTime{},
		))
	mock.ExpectExec("update user_async_jobs").
		WithArgs("job-1", attemptCount+1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
}

func expectNoWebhookEndpoints(mock sqlmock.Sqlmock, projectID, eventType string) {
	mock.ExpectQuery("select id from user_webhook_endpoints").
		WithArgs(projectID, "%,"+eventType+",%").
		WillReturnRows(sqlmock.NewRows([]string{"id"}))
}

func TestProcessNextAsyncJobExtractsText(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	app.Config.ObjectStorageProvider = "s3"
	app.ObjectStore = &fakeObjectStore{putBody: []byte("hello from the uploaded file")}
	expectAsyncJobClaim(mock, `{"asset_id":"asset-1"}`, 0, 3)
	mock.ExpectQuery("from user_workbench_assets where id = \\$1").
		WithArgs("asset-1").
		WillReturnRows(sqlmock.NewRows([]string{"project_id", "object_key", "mime_type"}).
			AddRow("project-1", "uploads/file.txt", "text/plain; charset=utf-8"))
	mock.ExpectBegin()
	mock.ExpectQuery("select cancel_requested from user_async_jobs").
		WithArgs("job-1").
		WillReturnRows(sqlmock.NewRows([]string{"cancel_requested"}).AddRow(false))
	mock.ExpectExec("insert into user_file_extractions").
		WithArgs(sqlmock.AnyArg(), "asset-1", "hello from the uploaded file", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_file_chunks").
		WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), "asset-1", "project-1", 0, "hello from the uploaded file", 28, 7, localEmbeddingModel, sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_embedding_records").
		WithArgs(sqlmock.AnyArg(), "project-1", sqlmock.AnyArg(), localEmbeddingModel, "inline:user_file_chunks", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("update user_async_jobs").
		WithArgs("job-1", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoWebhookEndpoints(mock, "project-1", "async_job.completed")
	mock.ExpectCommit()

	processed, err := app.ProcessNextAsyncJob(context.Background())
	if err != nil || !processed {
		t.Fatalf("processed=%v err=%v", processed, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessNextAsyncJobHonorsCancellationBeforePersisting(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	app.Config.ObjectStorageProvider = "s3"
	app.ObjectStore = &fakeObjectStore{putBody: []byte("discard this extraction")}
	expectAsyncJobClaim(mock, `{"asset_id":"asset-1"}`, 0, 3)
	mock.ExpectQuery("from user_workbench_assets where id = \\$1").
		WithArgs("asset-1").
		WillReturnRows(sqlmock.NewRows([]string{"project_id", "object_key", "mime_type"}).
			AddRow("project-1", "uploads/file.txt", "text/plain"))
	mock.ExpectBegin()
	mock.ExpectQuery("select cancel_requested from user_async_jobs").
		WithArgs("job-1").
		WillReturnRows(sqlmock.NewRows([]string{"cancel_requested"}).AddRow(true))
	mock.ExpectExec("update user_async_jobs set status = 'cancelled'").
		WithArgs("job-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoWebhookEndpoints(mock, "project-1", "async_job.cancelled")
	mock.ExpectCommit()

	processed, err := app.ProcessNextAsyncJob(context.Background())
	if err != nil || !processed {
		t.Fatalf("processed=%v err=%v", processed, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessNextAsyncJobPermanentlyFailsInvalidInput(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	expectAsyncJobClaim(mock, `{}`, 0, 3)
	mock.ExpectBegin()
	mock.ExpectExec("update user_async_jobs").
		WithArgs("job-1", "failed", 0, "file_extraction_requires_asset_id", "file_extraction_requires_asset_id", sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoWebhookEndpoints(mock, "project-1", "async_job.failed")
	mock.ExpectCommit()

	processed, err := app.ProcessNextAsyncJob(context.Background())
	if err != nil || !processed {
		t.Fatalf("processed=%v err=%v", processed, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

type failingObjectStore struct{ err error }

func (f failingObjectStore) Put(context.Context, string, []byte, string) error { return f.err }
func (f failingObjectStore) Get(context.Context, string) ([]byte, error)       { return nil, f.err }
func (f failingObjectStore) Delete(context.Context, string) error              { return f.err }
func (f failingObjectStore) PresignPut(context.Context, string, string, time.Duration) (string, error) {
	return "", f.err
}
func (f failingObjectStore) PresignGet(context.Context, string, time.Duration) (string, error) {
	return "", f.err
}

func TestProcessNextAsyncJobRequeuesTransientStorageFailure(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	app.Config.ObjectStorageProvider = "s3"
	app.ObjectStore = failingObjectStore{err: errors.New("storage temporarily unavailable")}
	expectAsyncJobClaim(mock, `{"asset_id":"asset-1"}`, 1, 3)
	mock.ExpectQuery("from user_workbench_assets where id = \\$1").
		WithArgs("asset-1").
		WillReturnRows(sqlmock.NewRows([]string{"project_id", "object_key", "mime_type"}).
			AddRow("project-1", "uploads/file.txt", "text/plain"))
	mock.ExpectExec("update user_async_jobs").
		WithArgs("job-1", "queued", 0, "read_asset", sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(0, 1))

	processed, err := app.ProcessNextAsyncJob(context.Background())
	if err != nil || !processed {
		t.Fatalf("processed=%v err=%v", processed, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestProcessNextAsyncJobReturnsIdleAfterLeaseRecovery(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	mock.ExpectBegin()
	mock.ExpectExec("worker lease expired").WillReturnResult(sqlmock.NewResult(0, 2))
	mock.ExpectQuery("for update skip locked").WillReturnRows(
		sqlmock.NewRows([]string{
			"id", "project_id", "actor_user_id", "job_type", "model_slug", "provider_slug",
			"metadata", "attempt_count", "max_attempts", "inference_request_id",
			"reservation_wallet_id", "reservation_promotional_credits", "reservation_paid_credits", "billing_status",
			"status", "provider_job_id", "provider_status", "started_at",
		}),
	)
	mock.ExpectRollback()

	processed, err := app.ProcessNextAsyncJob(context.Background())
	if err != nil || processed {
		t.Fatalf("processed=%v err=%v", processed, err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestExtractableTextMimeType(t *testing.T) {
	for _, value := range []string{"text/plain", "text/csv; charset=utf-8", "application/json", "application/xml"} {
		if !extractableTextMimeType(value) {
			t.Fatalf("expected %q to be extractable", value)
		}
	}
	for _, value := range []string{"application/pdf", "image/png", "application/octet-stream"} {
		if extractableTextMimeType(value) {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestExecuteAsyncMockImageGenerationCompletesAtomically(t *testing.T) {
	app, mock, cleanup := testApp(t)
	defer cleanup()
	app.Config.ObjectStorageProvider = "s3"
	app.Config.AssetBucket = "test-assets"
	app.Config.AppEnv = "test"
	app.ObjectStore = &fakeObjectStore{}

	mock.ExpectQuery("select r.id, r.route_group, m.slug").
		WithArgs("default", "mock-image").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "route_group", "slug", "modality", "profile_id", "upstream_model_id",
			"provider_id", "provider_slug", "channel_id", "channel_name", "channel_type",
			"base_url", "credential_ref", "priority", "weight", "response_time_ms",
		}).AddRow(
			"route-image", "default", "mock-image", "image", "profile-image", "mock-image-v1",
			"provider-mock", "mock-provider", "channel-mock", "Mock", "mock", "mock://image", "", 100, 100, 1,
		))
	mock.ExpectQuery("select pr.id, pr.pricing_variant").
		WithArgs("mock-image", "profile-image").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "pricing_variant", "price_type", "price_unit", "price", "provider_price", "price_metadata", "profile_matched",
		}).AddRow("price-image", "default", "output", "image", 0.0, 0.0, `{}`, true))
	mock.ExpectQuery("select conf_value from sys_config").WithArgs("usd_to_credit_ratio").
		WillReturnRows(sqlmock.NewRows([]string{"conf_value"}).AddRow("100"))

	// A zero-priced route still persists the billing/job state atomically.
	mock.ExpectBegin()
	mock.ExpectExec("update user_async_jobs").
		WithArgs("job-image", sqlmock.AnyArg(), "", 0.0, 0.0).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()
	mock.ExpectQuery("select cancel_requested from user_async_jobs").WithArgs("job-image").
		WillReturnRows(sqlmock.NewRows([]string{"cancel_requested"}).AddRow(false))
	mock.ExpectQuery("select cancel_requested from user_async_jobs").WithArgs("job-image").
		WillReturnRows(sqlmock.NewRows([]string{"cancel_requested"}).AddRow(false))

	mock.ExpectBegin()
	mock.ExpectExec("insert into user_inference_requests").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_workbench_assets").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_provider_attempts").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("insert into user_usage_events").WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectExec("update user_async_jobs").WillReturnResult(sqlmock.NewResult(0, 1))
	expectNoWebhookEndpoints(mock, "project-1", "async_job.completed")
	mock.ExpectCommit()

	result, err := app.executeGenerationJob(context.Background(), claimedAsyncJob{
		ID: "job-image", ProjectID: "project-1", JobType: "image_generation",
		ModelSlug: "mock-image", ProviderSlug: "mock-provider", Metadata: `{"prompt":"draw a fox","parameters":{"output_count":1}}`,
		MaxAttempts: 3, BillingStatus: "unreserved",
	})
	if err != nil {
		t.Fatal(err)
	}
	artifacts, ok := result["artifacts"].([]response)
	if !ok || len(artifacts) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
