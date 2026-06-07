-- Portable baseline schema for Model Market.
-- Keep this file conservative: no PostgreSQL extensions, jsonb, arrays, UUID
-- functions, generated columns, or database-specific enum types.
-- IDs are TEXT/VARCHAR values supplied by the application or seed data.

CREATE TABLE IF NOT EXISTS users (
  id VARCHAR(64) PRIMARY KEY,
  email VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  avatar_url VARCHAR(1024),
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS oauth_accounts (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  provider VARCHAR(64) NOT NULL,
  provider_account_id VARCHAR(255) NOT NULL,
  email VARCHAR(255),
  display_name VARCHAR(255),
  avatar_url VARCHAR(1024),
  last_login_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_oauth_accounts_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT uq_oauth_provider_account UNIQUE (provider, provider_account_id)
);

CREATE TABLE IF NOT EXISTS sessions (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE TABLE IF NOT EXISTS organizations (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  slug VARCHAR(255) NOT NULL UNIQUE,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS roles (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(128) NOT NULL UNIQUE,
  description TEXT
);

CREATE TABLE IF NOT EXISTS memberships (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  organization_id VARCHAR(64) NOT NULL,
  role VARCHAR(128) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_memberships_user FOREIGN KEY (user_id) REFERENCES users(id),
  CONSTRAINT fk_memberships_organization FOREIGN KEY (organization_id) REFERENCES organizations(id),
  CONSTRAINT uq_membership_user_org UNIQUE (user_id, organization_id)
);

CREATE TABLE IF NOT EXISTS projects (
  id VARCHAR(64) PRIMARY KEY,
  organization_id VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  slug VARCHAR(255) NOT NULL,
  environment VARCHAR(64) NOT NULL DEFAULT 'dev',
  retention_policy VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_projects_organization FOREIGN KEY (organization_id) REFERENCES organizations(id),
  CONSTRAINT uq_project_org_slug UNIQUE (organization_id, slug)
);

CREATE TABLE IF NOT EXISTS api_keys (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  prefix VARCHAR(32) NOT NULL,
  key_hash VARCHAR(255) NOT NULL UNIQUE,
  scopes VARCHAR(4000) NOT NULL DEFAULT '',
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  revoked_at TIMESTAMP,
  CONSTRAINT fk_api_keys_project FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS providers (
  id VARCHAR(64) PRIMARY KEY,
  slug VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  endpoint_url VARCHAR(1024),
  credential_ref VARCHAR(255),
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS provider_endpoints (
  id VARCHAR(64) PRIMARY KEY,
  provider_id VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  endpoint_url VARCHAR(1024) NOT NULL,
  region VARCHAR(128),
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_provider_endpoints_provider FOREIGN KEY (provider_id) REFERENCES providers(id)
);

CREATE TABLE IF NOT EXISTS capabilities (
  id VARCHAR(64) PRIMARY KEY,
  slug VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  description TEXT
);

CREATE TABLE IF NOT EXISTS models (
  id VARCHAR(64) PRIMARY KEY,
  provider_id VARCHAR(64) NOT NULL,
  slug VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  modality VARCHAR(64) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'public',
  context_window INTEGER NOT NULL DEFAULT 0,
  capabilities VARCHAR(4000) NOT NULL DEFAULT '{}',
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_models_provider FOREIGN KEY (provider_id) REFERENCES providers(id)
);

CREATE TABLE IF NOT EXISTS model_versions (
  id VARCHAR(64) PRIMARY KEY,
  model_id VARCHAR(64) NOT NULL,
  version VARCHAR(128) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_model_versions_model FOREIGN KEY (model_id) REFERENCES models(id),
  CONSTRAINT uq_model_version UNIQUE (model_id, version)
);

CREATE TABLE IF NOT EXISTS model_profiles (
  id VARCHAR(64) PRIMARY KEY,
  model_id VARCHAR(64) NOT NULL,
  slug VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'public',
  system_prompt VARCHAR(4000) NOT NULL DEFAULT '',
  default_parameters VARCHAR(4000) NOT NULL DEFAULT '{}',
  safety_settings VARCHAR(4000) NOT NULL DEFAULT '{}',
  config_version INTEGER NOT NULL DEFAULT 1,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_model_profiles_model FOREIGN KEY (model_id) REFERENCES models(id)
);

CREATE TABLE IF NOT EXISTS model_configurations (
  id VARCHAR(64) PRIMARY KEY,
  model_profile_id VARCHAR(64) NOT NULL,
  version INTEGER NOT NULL,
  config_data VARCHAR(4000) NOT NULL DEFAULT '{}',
  status VARCHAR(64) NOT NULL DEFAULT 'draft',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_model_configurations_profile FOREIGN KEY (model_profile_id) REFERENCES model_profiles(id),
  CONSTRAINT uq_model_configuration_version UNIQUE (model_profile_id, version)
);

CREATE TABLE IF NOT EXISTS pricing_plans (
  id VARCHAR(64) PRIMARY KEY,
  slug VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS price_rules (
  id VARCHAR(64) PRIMARY KEY,
  model_id VARCHAR(64),
  model_profile_id VARCHAR(64),
  input_token_price NUMERIC(18,8) NOT NULL DEFAULT 0,
  input_token_price_unit VARCHAR(64) NOT NULL DEFAULT '1k_tokens',
  output_token_price NUMERIC(18,8) NOT NULL DEFAULT 0,
  output_token_price_unit VARCHAR(64) NOT NULL DEFAULT '1k_tokens',
  provider_input_token_cost NUMERIC(18,8) NOT NULL DEFAULT 0,
  provider_input_token_cost_unit VARCHAR(64) NOT NULL DEFAULT '1k_tokens',
  provider_output_token_cost NUMERIC(18,8) NOT NULL DEFAULT 0,
  provider_output_token_cost_unit VARCHAR(64) NOT NULL DEFAULT '1k_tokens',
  currency VARCHAR(16) NOT NULL DEFAULT 'CREDIT',
  effective_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  CONSTRAINT fk_price_rules_model FOREIGN KEY (model_id) REFERENCES models(id),
  CONSTRAINT fk_price_rules_profile FOREIGN KEY (model_profile_id) REFERENCES model_profiles(id)
);

CREATE TABLE IF NOT EXISTS wallets (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL UNIQUE,
  paid_credits BIGINT NOT NULL DEFAULT 0,
  promotional_credits BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_wallets_project FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS ledger_transactions (
  id VARCHAR(64) PRIMARY KEY,
  wallet_id VARCHAR(64) NOT NULL,
  transaction_type VARCHAR(64) NOT NULL,
  amount BIGINT NOT NULL,
  credit_type VARCHAR(64) NOT NULL,
  status VARCHAR(64) NOT NULL,
  reason TEXT NOT NULL,
  idempotency_key VARCHAR(255) NOT NULL UNIQUE,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_ledger_wallet FOREIGN KEY (wallet_id) REFERENCES wallets(id)
);

CREATE TABLE IF NOT EXISTS payments (
  id VARCHAR(64) PRIMARY KEY,
  wallet_id VARCHAR(64) NOT NULL,
  provider VARCHAR(64) NOT NULL,
  provider_payment_id VARCHAR(255),
  amount_cents BIGINT NOT NULL,
  currency VARCHAR(32) NOT NULL,
  status VARCHAR(64) NOT NULL,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_payments_wallet FOREIGN KEY (wallet_id) REFERENCES wallets(id)
);

CREATE TABLE IF NOT EXISTS invoices (
  id VARCHAR(64) PRIMARY KEY,
  organization_id VARCHAR(64) NOT NULL,
  invoice_number VARCHAR(255) NOT NULL UNIQUE,
  status VARCHAR(64) NOT NULL,
  amount_cents BIGINT NOT NULL,
  currency VARCHAR(32) NOT NULL,
  issued_at TIMESTAMP,
  due_at TIMESTAMP,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  CONSTRAINT fk_invoices_organization FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

CREATE TABLE IF NOT EXISTS coupons (
  id VARCHAR(64) PRIMARY KEY,
  code VARCHAR(255) NOT NULL UNIQUE,
  credit_amount BIGINT NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  expires_at TIMESTAMP,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS conversations (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  title VARCHAR(255) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_conversations_project FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS conversation_branches (
  id VARCHAR(64) PRIMARY KEY,
  conversation_id VARCHAR(64) NOT NULL,
  parent_branch_id VARCHAR(64),
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_branches_conversation FOREIGN KEY (conversation_id) REFERENCES conversations(id),
  CONSTRAINT fk_branches_parent FOREIGN KEY (parent_branch_id) REFERENCES conversation_branches(id)
);

CREATE TABLE IF NOT EXISTS messages (
  id VARCHAR(64) PRIMARY KEY,
  conversation_id VARCHAR(64) NOT NULL,
  branch_id VARCHAR(64),
  role VARCHAR(64) NOT NULL,
  content TEXT NOT NULL,
  model_profile_id VARCHAR(64),
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_messages_conversation FOREIGN KEY (conversation_id) REFERENCES conversations(id),
  CONSTRAINT fk_messages_branch FOREIGN KEY (branch_id) REFERENCES conversation_branches(id),
  CONSTRAINT fk_messages_profile FOREIGN KEY (model_profile_id) REFERENCES model_profiles(id)
);

CREATE TABLE IF NOT EXISTS workspace_assets (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  conversation_id VARCHAR(64),
  asset_type VARCHAR(64) NOT NULL,
  storage_path VARCHAR(1024) NOT NULL,
  mime_type VARCHAR(255),
  size_bytes BIGINT,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_assets_project FOREIGN KEY (project_id) REFERENCES projects(id),
  CONSTRAINT fk_assets_conversation FOREIGN KEY (conversation_id) REFERENCES conversations(id)
);

CREATE TABLE IF NOT EXISTS message_attachments (
  id VARCHAR(64) PRIMARY KEY,
  message_id VARCHAR(64) NOT NULL,
  asset_id VARCHAR(64) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_attachments_message FOREIGN KEY (message_id) REFERENCES messages(id),
  CONSTRAINT fk_attachments_asset FOREIGN KEY (asset_id) REFERENCES workspace_assets(id)
);

CREATE TABLE IF NOT EXISTS file_extractions (
  id VARCHAR(64) PRIMARY KEY,
  asset_id VARCHAR(64) NOT NULL,
  extracted_text TEXT,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_file_extractions_asset FOREIGN KEY (asset_id) REFERENCES workspace_assets(id)
);

CREATE TABLE IF NOT EXISTS embedding_records (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  source_type VARCHAR(64) NOT NULL,
  source_id VARCHAR(64) NOT NULL,
  embedding_model VARCHAR(255) NOT NULL,
  vector_ref VARCHAR(1024),
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_embeddings_project FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS inference_requests (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  model_slug VARCHAR(255) NOT NULL,
  model_profile_id VARCHAR(64),
  provider_slug VARCHAR(255) NOT NULL,
  status VARCHAR(64) NOT NULL,
  input_units BIGINT NOT NULL DEFAULT 0,
  output_units BIGINT NOT NULL DEFAULT 0,
  customer_charge BIGINT NOT NULL DEFAULT 0,
  provider_cost BIGINT NOT NULL DEFAULT 0,
  margin BIGINT NOT NULL DEFAULT 0,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_inference_project FOREIGN KEY (project_id) REFERENCES projects(id),
  CONSTRAINT fk_inference_profile FOREIGN KEY (model_profile_id) REFERENCES model_profiles(id)
);

CREATE TABLE IF NOT EXISTS provider_attempts (
  id VARCHAR(64) PRIMARY KEY,
  inference_request_id VARCHAR(64) NOT NULL,
  provider_id VARCHAR(64) NOT NULL,
  status VARCHAR(64) NOT NULL,
  latency_ms BIGINT,
  provider_request_id VARCHAR(255),
  error_class VARCHAR(255),
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_attempts_inference FOREIGN KEY (inference_request_id) REFERENCES inference_requests(id),
  CONSTRAINT fk_attempts_provider FOREIGN KEY (provider_id) REFERENCES providers(id)
);

CREATE TABLE IF NOT EXISTS usage_events (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  inference_request_id VARCHAR(64),
  model_slug VARCHAR(255) NOT NULL,
  provider_slug VARCHAR(255) NOT NULL,
  event_type VARCHAR(64) NOT NULL,
  customer_charge BIGINT NOT NULL DEFAULT 0,
  provider_cost BIGINT NOT NULL DEFAULT 0,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_usage_project FOREIGN KEY (project_id) REFERENCES projects(id),
  CONSTRAINT fk_usage_inference FOREIGN KEY (inference_request_id) REFERENCES inference_requests(id)
);

CREATE TABLE IF NOT EXISTS async_jobs (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  job_type VARCHAR(64) NOT NULL,
  status VARCHAR(64) NOT NULL,
  model_slug VARCHAR(255),
  provider_slug VARCHAR(255),
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_jobs_project FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS routing_policies (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64),
  name VARCHAR(255) NOT NULL,
  policy_data VARCHAR(4000) NOT NULL DEFAULT '{}',
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_routing_project FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS budget_policies (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  limit_credits BIGINT NOT NULL,
  period VARCHAR(64) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_budget_project FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS webhook_endpoints (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  url VARCHAR(1024) NOT NULL,
  secret_ref VARCHAR(255),
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_webhooks_project FOREIGN KEY (project_id) REFERENCES projects(id)
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id VARCHAR(64) PRIMARY KEY,
  actor_user_id VARCHAR(64),
  organization_id VARCHAR(64),
  action VARCHAR(255) NOT NULL,
  target_type VARCHAR(255) NOT NULL,
  target_id VARCHAR(64),
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_audit_user FOREIGN KEY (actor_user_id) REFERENCES users(id),
  CONSTRAINT fk_audit_organization FOREIGN KEY (organization_id) REFERENCES organizations(id)
);

CREATE TABLE IF NOT EXISTS provider_settlements (
  id VARCHAR(64) PRIMARY KEY,
  provider_id VARCHAR(64) NOT NULL,
  period_start TIMESTAMP NOT NULL,
  period_end TIMESTAMP NOT NULL,
  amount_cents BIGINT NOT NULL,
  currency VARCHAR(32) NOT NULL,
  status VARCHAR(64) NOT NULL,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_settlements_provider FOREIGN KEY (provider_id) REFERENCES providers(id)
);

CREATE INDEX IF NOT EXISTS idx_api_keys_project ON api_keys(project_id);
CREATE INDEX IF NOT EXISTS idx_models_provider ON models(provider_id);
CREATE INDEX IF NOT EXISTS idx_model_profiles_model ON model_profiles(model_id);
CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_usage_project_created ON usage_events(project_id, created_at);
CREATE INDEX IF NOT EXISTS idx_inference_project_created ON inference_requests(project_id, created_at);
CREATE INDEX IF NOT EXISTS idx_assets_project_created ON workspace_assets(project_id, created_at);
CREATE INDEX IF NOT EXISTS idx_audit_org_created ON audit_logs(organization_id, created_at);
