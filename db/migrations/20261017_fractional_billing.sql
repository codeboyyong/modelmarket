-- Apply once to an existing PostgreSQL database before deploying fractional
-- billing. No balances are rescaled, and no tables or data are deleted.
BEGIN;

ALTER TABLE user_wallets
  ALTER COLUMN paid_credits TYPE NUMERIC(24,8),
  ALTER COLUMN promotional_credits TYPE NUMERIC(24,8);
ALTER TABLE user_ledger_transactions ALTER COLUMN amount TYPE NUMERIC(24,8);
ALTER TABLE user_inference_requests
  ALTER COLUMN customer_charge TYPE NUMERIC(24,8),
  ALTER COLUMN provider_cost TYPE NUMERIC(24,8),
  ALTER COLUMN margin TYPE NUMERIC(24,8);
ALTER TABLE user_messages
  ALTER COLUMN customer_charge TYPE NUMERIC(24,8),
  ALTER COLUMN provider_cost TYPE NUMERIC(24,8);
ALTER TABLE user_workbench_assets
  ALTER COLUMN customer_charge TYPE NUMERIC(24,8),
  ALTER COLUMN provider_cost TYPE NUMERIC(24,8);
ALTER TABLE user_usage_events
  ALTER COLUMN customer_charge TYPE NUMERIC(24,8),
  ALTER COLUMN provider_cost TYPE NUMERIC(24,8);
ALTER TABLE user_api_keys ADD COLUMN IF NOT EXISTS created_by_user_id VARCHAR(64)
  REFERENCES sys_users(id);

COMMIT;
