-- Portable baseline schema for Model Market.
-- Keep this file conservative: no PostgreSQL extensions, jsonb, arrays, UUID
-- functions, generated columns, or database-specific enum types.
-- IDs are TEXT/VARCHAR values supplied by the application or seed data.

-- System table: application user identities used for login, ownership, and account display.
CREATE TABLE IF NOT EXISTS sys_users (
  id VARCHAR(64) PRIMARY KEY,
  email VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  avatar_url VARCHAR(1024),
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  password_hash VARCHAR(255),
  user_type VARCHAR(64) NOT NULL DEFAULT 'individual_consumer',
  company_id VARCHAR(64),
  ui_theme VARCHAR(32) NOT NULL DEFAULT 'Light',
  language VARCHAR(32) NOT NULL DEFAULT 'EN',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- User table: customer company accounts for corporate admins and members sharing credits.
CREATE TABLE IF NOT EXISTS user_companies (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  owner_user_id VARCHAR(64) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_companies_owner FOREIGN KEY (owner_user_id) REFERENCES sys_users(id)
);

-- System table: external OAuth identities linked to application users.
CREATE TABLE IF NOT EXISTS sys_oauth_accounts (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  provider VARCHAR(64) NOT NULL,
  provider_account_id VARCHAR(255) NOT NULL,
  email VARCHAR(255),
  display_name VARCHAR(255),
  avatar_url VARCHAR(1024),
  last_login_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_oauth_accounts_user FOREIGN KEY (user_id) REFERENCES sys_users(id),
  CONSTRAINT uq_sys_oauth_provider_account UNIQUE (provider, provider_account_id)
);

-- System table: login sessions and hashed access tokens for authenticated users.
CREATE TABLE IF NOT EXISTS sys_sessions (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  token_hash VARCHAR(255) NOT NULL UNIQUE,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_sessions_user FOREIGN KEY (user_id) REFERENCES sys_users(id)
);

-- System table: tenant organizations that own projects and billing context.
CREATE TABLE IF NOT EXISTS sys_organizations (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  slug VARCHAR(255) NOT NULL UNIQUE,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- System table: reusable role definitions for authorization and membership assignment.
CREATE TABLE IF NOT EXISTS sys_roles (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(128) NOT NULL UNIQUE,
  description TEXT
);

-- System table: links users to organizations with a role.
CREATE TABLE IF NOT EXISTS sys_memberships (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  organization_id VARCHAR(64) NOT NULL,
  role VARCHAR(128) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_memberships_user FOREIGN KEY (user_id) REFERENCES sys_users(id),
  CONSTRAINT fk_memberships_organization FOREIGN KEY (organization_id) REFERENCES sys_organizations(id),
  CONSTRAINT uq_sys_membership_user_org UNIQUE (user_id, organization_id)
);

-- User table: customer workspaces that group API keys, conversations, assets, and usage.
CREATE TABLE IF NOT EXISTS user_projects (
  id VARCHAR(64) PRIMARY KEY,
  organization_id VARCHAR(64) NOT NULL,
  company_id VARCHAR(64),
  name VARCHAR(255) NOT NULL,
  slug VARCHAR(255) NOT NULL,
  environment VARCHAR(64) NOT NULL DEFAULT 'dev',
  retention_policy VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_projects_organization FOREIGN KEY (organization_id) REFERENCES sys_organizations(id),
  CONSTRAINT fk_projects_company FOREIGN KEY (company_id) REFERENCES user_companies(id),
  CONSTRAINT uq_user_project_org_slug UNIQUE (organization_id, slug)
);

-- User table: project-scoped API credentials for calling the model gateway.
CREATE TABLE IF NOT EXISTS user_api_keys (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  prefix VARCHAR(32) NOT NULL,
  key_hash VARCHAR(255) NOT NULL UNIQUE,
  scopes VARCHAR(4000) NOT NULL DEFAULT '',
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  revoked_at TIMESTAMP,
  CONSTRAINT fk_api_keys_project FOREIGN KEY (project_id) REFERENCES user_projects(id)
);

-- System table: model provider catalog entries and provider-level metadata.
CREATE TABLE IF NOT EXISTS sys_providers (
  id VARCHAR(64) PRIMARY KEY,
  slug VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  endpoint_url VARCHAR(1024),
  credential_ref VARCHAR(255),
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- System table: provider API endpoint definitions, regions, and endpoint status.
CREATE TABLE IF NOT EXISTS sys_provider_endpoints (
  id VARCHAR(64) PRIMARY KEY,
  provider_id VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  endpoint_url VARCHAR(1024) NOT NULL,
  region VARCHAR(128),
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_provider_endpoints_provider FOREIGN KEY (provider_id) REFERENCES sys_providers(id)
);

-- System table: catalog of model capability labels used for discovery and filtering.
CREATE TABLE IF NOT EXISTS sys_capabilities (
  id VARCHAR(64) PRIMARY KEY,
  slug VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  description TEXT
);

-- System table: public model catalog records exposed in the marketplace.
CREATE TABLE IF NOT EXISTS sys_models (
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
  CONSTRAINT fk_models_provider FOREIGN KEY (provider_id) REFERENCES sys_providers(id)
);

-- System table: provider model version metadata for catalog lifecycle tracking.
CREATE TABLE IF NOT EXISTS sys_model_versions (
  id VARCHAR(64) PRIMARY KEY,
  model_id VARCHAR(64) NOT NULL,
  version VARCHAR(128) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_model_versions_model FOREIGN KEY (model_id) REFERENCES sys_models(id),
  CONSTRAINT uq_sys_model_version UNIQUE (model_id, version)
);

-- System table: model profiles with prompts, defaults, safety settings, and public routing metadata.
CREATE TABLE IF NOT EXISTS sys_model_profiles (
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
  CONSTRAINT fk_model_profiles_model FOREIGN KEY (model_id) REFERENCES sys_models(id)
);

-- System table: versioned configuration snapshots for model profiles.
CREATE TABLE IF NOT EXISTS sys_model_configurations (
  id VARCHAR(64) PRIMARY KEY,
  model_profile_id VARCHAR(64) NOT NULL,
  version INTEGER NOT NULL,
  config_data VARCHAR(4000) NOT NULL DEFAULT '{}',
  status VARCHAR(64) NOT NULL DEFAULT 'draft',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_model_configurations_profile FOREIGN KEY (model_profile_id) REFERENCES sys_model_profiles(id),
  CONSTRAINT uq_sys_model_configuration_version UNIQUE (model_profile_id, version)
);

-- System table: platform pricing plans and plan metadata.
CREATE TABLE IF NOT EXISTS sys_pricing_plans (
  id VARCHAR(64) PRIMARY KEY,
  slug VARCHAR(255) NOT NULL UNIQUE,
  name VARCHAR(255) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}'
);

-- System table: model and profile price rules plus provider cost data.
CREATE TABLE IF NOT EXISTS sys_price_rules (
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
  CONSTRAINT fk_price_rules_model FOREIGN KEY (model_id) REFERENCES sys_models(id),
  CONSTRAINT fk_price_rules_profile FOREIGN KEY (model_profile_id) REFERENCES sys_model_profiles(id)
);

-- User table: project credit balances for paid and promotional credits.
CREATE TABLE IF NOT EXISTS user_wallets (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) UNIQUE,
  company_id VARCHAR(64),
  paid_credits BIGINT NOT NULL DEFAULT 0,
  promotional_credits BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_wallets_project FOREIGN KEY (project_id) REFERENCES user_projects(id),
  CONSTRAINT fk_wallets_company FOREIGN KEY (company_id) REFERENCES user_companies(id)
);

-- User table: immutable credit balance changes for wallet auditability.
CREATE TABLE IF NOT EXISTS user_ledger_transactions (
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
  CONSTRAINT fk_ledger_wallet FOREIGN KEY (wallet_id) REFERENCES user_wallets(id)
);

-- User table: payment records that add or reconcile wallet credits.
CREATE TABLE IF NOT EXISTS user_payments (
  id VARCHAR(64) PRIMARY KEY,
  wallet_id VARCHAR(64) NOT NULL,
  provider VARCHAR(64) NOT NULL,
  provider_payment_id VARCHAR(255),
  amount_cents BIGINT NOT NULL,
  currency VARCHAR(32) NOT NULL,
  status VARCHAR(64) NOT NULL,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_payments_wallet FOREIGN KEY (wallet_id) REFERENCES user_wallets(id)
);

-- User table: user-facing credit purchase history for account billing views.
CREATE TABLE IF NOT EXISTS user_credit_purchases (
  id VARCHAR(64) PRIMARY KEY,
  user_id VARCHAR(64) NOT NULL,
  credits BIGINT NOT NULL,
  amount_cents BIGINT NOT NULL,
  currency VARCHAR(32) NOT NULL DEFAULT 'USD',
  status VARCHAR(64) NOT NULL DEFAULT 'posted',
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_credit_purchases_user FOREIGN KEY (user_id) REFERENCES sys_users(id)
);

-- User table: organization billing invoices and invoice metadata.
CREATE TABLE IF NOT EXISTS user_invoices (
  id VARCHAR(64) PRIMARY KEY,
  organization_id VARCHAR(64) NOT NULL,
  invoice_number VARCHAR(255) NOT NULL UNIQUE,
  status VARCHAR(64) NOT NULL,
  amount_cents BIGINT NOT NULL,
  currency VARCHAR(32) NOT NULL,
  issued_at TIMESTAMP,
  due_at TIMESTAMP,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  CONSTRAINT fk_invoices_organization FOREIGN KEY (organization_id) REFERENCES sys_organizations(id)
);

-- System table: platform-managed coupon codes for promotional credits.
CREATE TABLE IF NOT EXISTS sys_coupons (
  id VARCHAR(64) PRIMARY KEY,
  code VARCHAR(255) NOT NULL UNIQUE,
  credit_amount BIGINT NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  expires_at TIMESTAMP,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}'
);

-- User table: workbench conversation threads within a project.
CREATE TABLE IF NOT EXISTS user_conversations (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  title VARCHAR(255) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_conversations_project FOREIGN KEY (project_id) REFERENCES user_projects(id)
);

-- User table: alternate branches for a workbench conversation.
CREATE TABLE IF NOT EXISTS user_conversation_branches (
  id VARCHAR(64) PRIMARY KEY,
  conversation_id VARCHAR(64) NOT NULL,
  parent_branch_id VARCHAR(64),
  name VARCHAR(255) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_branches_conversation FOREIGN KEY (conversation_id) REFERENCES user_conversations(id),
  CONSTRAINT fk_branches_parent FOREIGN KEY (parent_branch_id) REFERENCES user_conversation_branches(id)
);

-- User table: prompt and assistant messages stored inside conversations.
CREATE TABLE IF NOT EXISTS user_messages (
  id VARCHAR(64) PRIMARY KEY,
  conversation_id VARCHAR(64) NOT NULL,
  branch_id VARCHAR(64),
  role VARCHAR(64) NOT NULL,
  content TEXT NOT NULL,
  model_profile_id VARCHAR(64),
  inference_request_id VARCHAR(64),
  customer_charge BIGINT NOT NULL DEFAULT 0,
  provider_cost BIGINT NOT NULL DEFAULT 0,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_messages_conversation FOREIGN KEY (conversation_id) REFERENCES user_conversations(id),
  CONSTRAINT fk_messages_branch FOREIGN KEY (branch_id) REFERENCES user_conversation_branches(id),
  CONSTRAINT fk_messages_profile FOREIGN KEY (model_profile_id) REFERENCES sys_model_profiles(id)
);

-- User table: uploaded and generated artifacts attached to projects or conversations.
CREATE TABLE IF NOT EXISTS user_workspace_assets (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  conversation_id VARCHAR(64),
  asset_type VARCHAR(64) NOT NULL,
  storage_path VARCHAR(1024) NOT NULL,
  storage_provider VARCHAR(64) NOT NULL DEFAULT 's3',
  bucket_name VARCHAR(255),
  object_key VARCHAR(1024),
  download_url VARCHAR(2048),
  mime_type VARCHAR(255),
  size_bytes BIGINT,
  inference_request_id VARCHAR(64),
  customer_charge BIGINT NOT NULL DEFAULT 0,
  provider_cost BIGINT NOT NULL DEFAULT 0,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_assets_project FOREIGN KEY (project_id) REFERENCES user_projects(id),
  CONSTRAINT fk_assets_conversation FOREIGN KEY (conversation_id) REFERENCES user_conversations(id)
);

-- User table: links messages to uploaded or generated workspace assets.
CREATE TABLE IF NOT EXISTS user_message_attachments (
  id VARCHAR(64) PRIMARY KEY,
  message_id VARCHAR(64) NOT NULL,
  asset_id VARCHAR(64) NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_attachments_message FOREIGN KEY (message_id) REFERENCES user_messages(id),
  CONSTRAINT fk_attachments_asset FOREIGN KEY (asset_id) REFERENCES user_workspace_assets(id)
);

-- User table: extracted text and metadata derived from workspace assets.
CREATE TABLE IF NOT EXISTS user_file_extractions (
  id VARCHAR(64) PRIMARY KEY,
  asset_id VARCHAR(64) NOT NULL,
  extracted_text TEXT,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_file_extractions_asset FOREIGN KEY (asset_id) REFERENCES user_workspace_assets(id)
);

-- User table: embedding references created from project content.
CREATE TABLE IF NOT EXISTS user_embedding_records (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  source_type VARCHAR(64) NOT NULL,
  source_id VARCHAR(64) NOT NULL,
  embedding_model VARCHAR(255) NOT NULL,
  vector_ref VARCHAR(1024),
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_embeddings_project FOREIGN KEY (project_id) REFERENCES user_projects(id)
);

-- User table: model gateway request history with charges, costs, and routing result.
CREATE TABLE IF NOT EXISTS user_inference_requests (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  actor_user_id VARCHAR(64),
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
  CONSTRAINT fk_inference_project FOREIGN KEY (project_id) REFERENCES user_projects(id),
  CONSTRAINT fk_inference_actor FOREIGN KEY (actor_user_id) REFERENCES sys_users(id),
  CONSTRAINT fk_inference_profile FOREIGN KEY (model_profile_id) REFERENCES sys_model_profiles(id)
);

-- User table: provider-level attempts for an inference request, including latency and errors.
CREATE TABLE IF NOT EXISTS user_provider_attempts (
  id VARCHAR(64) PRIMARY KEY,
  inference_request_id VARCHAR(64) NOT NULL,
  provider_id VARCHAR(64) NOT NULL,
  status VARCHAR(64) NOT NULL,
  latency_ms BIGINT,
  provider_request_id VARCHAR(255),
  error_class VARCHAR(255),
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_attempts_inference FOREIGN KEY (inference_request_id) REFERENCES user_inference_requests(id),
  CONSTRAINT fk_attempts_provider FOREIGN KEY (provider_id) REFERENCES sys_providers(id)
);

-- User table: metered usage events used for credit reporting and analytics.
CREATE TABLE IF NOT EXISTS user_usage_events (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  actor_user_id VARCHAR(64),
  inference_request_id VARCHAR(64),
  model_slug VARCHAR(255) NOT NULL,
  provider_slug VARCHAR(255) NOT NULL,
  event_type VARCHAR(64) NOT NULL,
  input_tokens BIGINT NOT NULL DEFAULT 0,
  output_tokens BIGINT NOT NULL DEFAULT 0,
  customer_charge BIGINT NOT NULL DEFAULT 0,
  provider_cost BIGINT NOT NULL DEFAULT 0,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_usage_project FOREIGN KEY (project_id) REFERENCES user_projects(id),
  CONSTRAINT fk_usage_actor FOREIGN KEY (actor_user_id) REFERENCES sys_users(id),
  CONSTRAINT fk_usage_inference FOREIGN KEY (inference_request_id) REFERENCES user_inference_requests(id)
);

-- User table: asynchronous generation job records for image, audio, video, and file workflows.
CREATE TABLE IF NOT EXISTS user_async_jobs (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  job_type VARCHAR(64) NOT NULL,
  status VARCHAR(64) NOT NULL,
  model_slug VARCHAR(255),
  provider_slug VARCHAR(255),
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_jobs_project FOREIGN KEY (project_id) REFERENCES user_projects(id)
);

-- User table: project-specific routing policy overrides for model gateway behavior.
CREATE TABLE IF NOT EXISTS user_routing_policies (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64),
  name VARCHAR(255) NOT NULL,
  policy_data VARCHAR(4000) NOT NULL DEFAULT '{}',
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_routing_project FOREIGN KEY (project_id) REFERENCES user_projects(id)
);

-- User table: project spending limits and budget guardrails.
CREATE TABLE IF NOT EXISTS user_budget_policies (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  name VARCHAR(255) NOT NULL,
  limit_credits BIGINT NOT NULL,
  period VARCHAR(64) NOT NULL,
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_budget_project FOREIGN KEY (project_id) REFERENCES user_projects(id)
);

-- User table: project webhook targets for async job and usage notifications.
CREATE TABLE IF NOT EXISTS user_webhook_endpoints (
  id VARCHAR(64) PRIMARY KEY,
  project_id VARCHAR(64) NOT NULL,
  url VARCHAR(1024) NOT NULL,
  secret_ref VARCHAR(255),
  status VARCHAR(64) NOT NULL DEFAULT 'active',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_webhooks_project FOREIGN KEY (project_id) REFERENCES user_projects(id)
);

-- System table: platform audit trail for administrative and organization-level actions.
CREATE TABLE IF NOT EXISTS sys_audit_logs (
  id VARCHAR(64) PRIMARY KEY,
  actor_user_id VARCHAR(64),
  organization_id VARCHAR(64),
  action VARCHAR(255) NOT NULL,
  target_type VARCHAR(255) NOT NULL,
  target_id VARCHAR(64),
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_audit_user FOREIGN KEY (actor_user_id) REFERENCES sys_users(id),
  CONSTRAINT fk_audit_organization FOREIGN KEY (organization_id) REFERENCES sys_organizations(id)
);

-- System table: provider settlement records for platform cost reconciliation.
CREATE TABLE IF NOT EXISTS sys_provider_settlements (
  id VARCHAR(64) PRIMARY KEY,
  provider_id VARCHAR(64) NOT NULL,
  period_start TIMESTAMP NOT NULL,
  period_end TIMESTAMP NOT NULL,
  amount_cents BIGINT NOT NULL,
  currency VARCHAR(32) NOT NULL,
  status VARCHAR(64) NOT NULL,
  metadata VARCHAR(4000) NOT NULL DEFAULT '{}',
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_settlements_provider FOREIGN KEY (provider_id) REFERENCES sys_providers(id)
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_messages_inference') THEN
    ALTER TABLE user_messages
      ADD CONSTRAINT fk_messages_inference FOREIGN KEY (inference_request_id) REFERENCES user_inference_requests(id);
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_assets_inference') THEN
    ALTER TABLE user_workspace_assets
      ADD CONSTRAINT fk_assets_inference FOREIGN KEY (inference_request_id) REFERENCES user_inference_requests(id);
  END IF;
END $$;

ALTER TABLE sys_users ADD COLUMN IF NOT EXISTS user_type VARCHAR(64) NOT NULL DEFAULT 'individual_consumer';
ALTER TABLE sys_users ADD COLUMN IF NOT EXISTS company_id VARCHAR(64);
ALTER TABLE sys_users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);
ALTER TABLE user_projects ADD COLUMN IF NOT EXISTS company_id VARCHAR(64);
ALTER TABLE user_wallets ADD COLUMN IF NOT EXISTS company_id VARCHAR(64);
ALTER TABLE user_wallets ALTER COLUMN project_id DROP NOT NULL;
ALTER TABLE user_inference_requests ADD COLUMN IF NOT EXISTS actor_user_id VARCHAR(64);
ALTER TABLE user_usage_events ADD COLUMN IF NOT EXISTS actor_user_id VARCHAR(64);
ALTER TABLE user_usage_events ADD COLUMN IF NOT EXISTS input_tokens BIGINT NOT NULL DEFAULT 0;
ALTER TABLE user_usage_events ADD COLUMN IF NOT EXISTS output_tokens BIGINT NOT NULL DEFAULT 0;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_projects_company') THEN
    ALTER TABLE user_projects
      ADD CONSTRAINT fk_projects_company FOREIGN KEY (company_id) REFERENCES user_companies(id);
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_wallets_company') THEN
    ALTER TABLE user_wallets
      ADD CONSTRAINT fk_wallets_company FOREIGN KEY (company_id) REFERENCES user_companies(id);
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_inference_actor') THEN
    ALTER TABLE user_inference_requests
      ADD CONSTRAINT fk_inference_actor FOREIGN KEY (actor_user_id) REFERENCES sys_users(id);
  END IF;
END $$;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints WHERE constraint_name = 'fk_usage_actor') THEN
    ALTER TABLE user_usage_events
      ADD CONSTRAINT fk_usage_actor FOREIGN KEY (actor_user_id) REFERENCES sys_users(id);
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_user_api_keys_project ON user_api_keys(project_id);
CREATE INDEX IF NOT EXISTS idx_sys_users_company ON sys_users(company_id);
CREATE INDEX IF NOT EXISTS idx_user_companies_owner ON user_companies(owner_user_id);
CREATE INDEX IF NOT EXISTS idx_user_projects_company ON user_projects(company_id);
CREATE INDEX IF NOT EXISTS idx_user_wallets_company ON user_wallets(company_id);
CREATE INDEX IF NOT EXISTS idx_user_credit_purchases_user_created ON user_credit_purchases(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_sys_models_provider ON sys_models(provider_id);
CREATE INDEX IF NOT EXISTS idx_sys_model_profiles_model ON sys_model_profiles(model_id);
CREATE INDEX IF NOT EXISTS idx_user_messages_conversation ON user_messages(conversation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_user_messages_inference ON user_messages(inference_request_id);
CREATE INDEX IF NOT EXISTS idx_user_usage_project_created ON user_usage_events(project_id, created_at);
CREATE INDEX IF NOT EXISTS idx_user_usage_actor_created ON user_usage_events(actor_user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_user_inference_project_created ON user_inference_requests(project_id, created_at);
CREATE INDEX IF NOT EXISTS idx_user_inference_actor_created ON user_inference_requests(actor_user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_user_assets_project_created ON user_workspace_assets(project_id, created_at);
CREATE INDEX IF NOT EXISTS idx_user_assets_inference ON user_workspace_assets(inference_request_id);
CREATE INDEX IF NOT EXISTS idx_sys_audit_org_created ON sys_audit_logs(organization_id, created_at);
