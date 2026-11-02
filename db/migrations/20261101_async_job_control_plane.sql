-- Expand the existing async-job placeholder into a queryable, cancellable job
-- control plane. Existing rows remain valid and receive safe defaults.
BEGIN;

ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS actor_user_id VARCHAR(64)
  REFERENCES sys_users(id);
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS progress INTEGER NOT NULL DEFAULT 0;
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS result_metadata VARCHAR(4000) NOT NULL DEFAULT '{}';
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS inference_request_id VARCHAR(64);
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS reservation_wallet_id VARCHAR(64)
  REFERENCES user_wallets(id);
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS reservation_promotional_credits NUMERIC(24,8) NOT NULL DEFAULT 0;
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS reservation_paid_credits NUMERIC(24,8) NOT NULL DEFAULT 0;
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS billing_status VARCHAR(64) NOT NULL DEFAULT 'unreserved';
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS provider_job_id VARCHAR(255);
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS provider_status VARCHAR(128);
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS next_provider_poll_at TIMESTAMP;
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS error_code VARCHAR(255);
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS error_message VARCHAR(2000);
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS attempt_count INTEGER NOT NULL DEFAULT 0;
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS max_attempts INTEGER NOT NULL DEFAULT 3;
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS cancel_requested BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS started_at TIMESTAMP;
ALTER TABLE user_async_jobs ADD COLUMN IF NOT EXISTS completed_at TIMESTAMP;

CREATE UNIQUE INDEX IF NOT EXISTS uq_async_jobs_project_idempotency
  ON user_async_jobs(project_id, idempotency_key);
CREATE INDEX IF NOT EXISTS idx_async_jobs_project_created
  ON user_async_jobs(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_async_jobs_worker_queue
  ON user_async_jobs(status, created_at)
  WHERE status = 'queued';

COMMIT;
