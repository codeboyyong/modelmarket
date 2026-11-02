BEGIN;

ALTER TABLE user_webhook_endpoints
  ADD COLUMN IF NOT EXISTS events VARCHAR(2000) NOT NULL DEFAULT '*';

CREATE TABLE IF NOT EXISTS user_webhook_deliveries (
  id VARCHAR(64) PRIMARY KEY,
  endpoint_id VARCHAR(64) NOT NULL REFERENCES user_webhook_endpoints(id),
  project_id VARCHAR(64) NOT NULL REFERENCES user_projects(id),
  event_type VARCHAR(255) NOT NULL,
  payload VARCHAR(4000) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'pending',
  attempt_count INTEGER NOT NULL DEFAULT 0,
  max_attempts INTEGER NOT NULL DEFAULT 6,
  next_attempt_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  response_status INTEGER,
  response_body VARCHAR(2000),
  error_message VARCHAR(2000),
  delivered_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_webhook_delivery_queue
  ON user_webhook_deliveries(status, next_attempt_at, created_at);
CREATE INDEX IF NOT EXISTS idx_webhook_delivery_project
  ON user_webhook_deliveries(project_id, created_at DESC);

COMMIT;
