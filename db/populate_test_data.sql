-- Portable deterministic test data for Model Market.
-- IDs are explicit so this file does not require database-specific UUID functions.

DELETE FROM provider_settlements;
DELETE FROM audit_logs;
DELETE FROM webhook_endpoints;
DELETE FROM budget_policies;
DELETE FROM routing_policies;
DELETE FROM async_jobs;
DELETE FROM usage_events;
DELETE FROM provider_attempts;
DELETE FROM inference_requests;
DELETE FROM embedding_records;
DELETE FROM file_extractions;
DELETE FROM message_attachments;
DELETE FROM workspace_assets;
DELETE FROM messages;
DELETE FROM conversation_branches;
DELETE FROM conversations;
DELETE FROM coupons;
DELETE FROM invoices;
DELETE FROM payments;
DELETE FROM ledger_transactions;
DELETE FROM wallets;
DELETE FROM price_rules;
DELETE FROM pricing_plans;
DELETE FROM model_configurations;
DELETE FROM model_profiles;
DELETE FROM model_versions;
DELETE FROM models;
DELETE FROM capabilities;
DELETE FROM provider_endpoints;
DELETE FROM providers;
DELETE FROM api_keys;
DELETE FROM projects;
DELETE FROM memberships;
DELETE FROM organizations;
DELETE FROM sessions;
DELETE FROM oauth_accounts;
DELETE FROM users;
DELETE FROM roles;

INSERT INTO roles (id, name, description) VALUES
  ('role-owner', 'owner', 'Organization owner'),
  ('role-admin', 'admin', 'Organization administrator'),
  ('role-developer', 'developer', 'Developer user'),
  ('role-readonly', 'readonly', 'Read-only user');

INSERT INTO users (id, email, name, avatar_url, status) VALUES
  ('user-admin', 'admin@example.com', 'Admin User', NULL, 'active'),
  ('user-developer', 'developer@example.com', 'Developer User', NULL, 'active');

INSERT INTO oauth_accounts (id, user_id, provider, provider_account_id, email, display_name) VALUES
  ('oauth-google-admin', 'user-admin', 'google', 'google-admin-dev', 'admin@example.com', 'Admin User'),
  ('oauth-facebook-dev', 'user-developer', 'facebook', 'facebook-dev-dev', 'developer@example.com', 'Developer User');

INSERT INTO organizations (id, name, slug, status) VALUES
  ('org-demo', 'Demo Organization', 'demo-org', 'active');

INSERT INTO memberships (id, user_id, organization_id, role) VALUES
  ('membership-admin', 'user-admin', 'org-demo', 'owner'),
  ('membership-developer', 'user-developer', 'org-demo', 'developer');

INSERT INTO projects (id, organization_id, name, slug, environment, retention_policy) VALUES
  ('project-demo', 'org-demo', 'Demo Project', 'demo-project', 'dev', '{"conversation_days":30,"asset_days":30}');

INSERT INTO api_keys (id, project_id, name, prefix, key_hash, scopes, status) VALUES
  ('api-key-demo', 'project-demo', 'Seeded development key', 'mk_seeded', 'seeded-key-hash-replace-before-real-use', 'models:read,chat:create', 'active');

INSERT INTO providers (id, slug, name, status, endpoint_url, credential_ref, metadata) VALUES
  ('provider-mock', 'mock-provider', 'Mock Provider', 'active', 'mock://provider', NULL, '{"mode":"dev","supports_streaming":true}'),
  ('provider-openai-placeholder', 'openai-placeholder', 'OpenAI Placeholder', 'inactive', 'https://api.openai.com', 'OPENAI_API_KEY', '{"enabled_by_default":false}');

INSERT INTO provider_endpoints (id, provider_id, name, endpoint_url, region, status, metadata) VALUES
  ('endpoint-mock-default', 'provider-mock', 'Mock Default Endpoint', 'mock://provider/default', 'local', 'active', '{}'),
  ('endpoint-openai-default', 'provider-openai-placeholder', 'OpenAI Default Endpoint', 'https://api.openai.com', 'global', 'inactive', '{}');

INSERT INTO capabilities (id, slug, name, description) VALUES
  ('cap-chat', 'chat', 'Chat', 'Chat completion support'),
  ('cap-streaming', 'streaming', 'Streaming', 'Streaming response support'),
  ('cap-image-generation', 'image-generation', 'Image Generation', 'Image generation support'),
  ('cap-async-jobs', 'async-jobs', 'Async Jobs', 'Long-running async job support');

INSERT INTO models (id, provider_id, slug, name, modality, status, context_window, capabilities, metadata) VALUES
  ('model-mock-chat', 'provider-mock', 'mock-chat', 'Mock Chat', 'chat', 'public', 8192, '{"chat":true,"streaming":true,"files":false}', '{"quality_tier":"dev"}'),
  ('model-mock-creative', 'provider-mock', 'mock-creative', 'Mock Creative', 'image', 'public', 0, '{"image_generation":true,"async_jobs":true}', '{"quality_tier":"dev"}');

INSERT INTO model_versions (id, model_id, version, status, metadata) VALUES
  ('model-version-mock-chat-dev', 'model-mock-chat', 'dev', 'active', '{}'),
  ('model-version-mock-creative-dev', 'model-mock-creative', 'dev', 'active', '{}');

INSERT INTO model_profiles (id, model_id, slug, name, status, system_prompt, default_parameters, safety_settings, config_version) VALUES
  ('profile-mock-chat-default', 'model-mock-chat', 'mock-chat-default', 'Mock Chat Default', 'public', 'You are a helpful mocked assistant for local development.', '{"temperature":0.2,"max_tokens":512}', '{"moderation":"mock"}', 1),
  ('profile-mock-creative-default', 'model-mock-creative', 'mock-creative-default', 'Mock Creative Default', 'public', 'Generate deterministic local development assets.', '{"size":"1024x1024"}', '{"moderation":"mock"}', 1);

INSERT INTO model_configurations (id, model_profile_id, version, config_data, status) VALUES
  ('config-mock-chat-v1', 'profile-mock-chat-default', 1, '{"system_prompt":"You are a helpful mocked assistant for local development.","temperature":0.2,"max_tokens":512}', 'published'),
  ('config-mock-creative-v1', 'profile-mock-creative-default', 1, '{"system_prompt":"Generate deterministic local development assets.","size":"1024x1024"}', 'published');

INSERT INTO pricing_plans (id, slug, name, status, metadata) VALUES
  ('plan-developer', 'developer', 'Developer', 'active', '{"default":true}');

INSERT INTO price_rules (
  id,
  model_id,
  model_profile_id,
  input_token_price,
  input_token_price_unit,
  output_token_price,
  output_token_price_unit,
  provider_input_token_cost,
  provider_input_token_cost_unit,
  provider_output_token_cost,
  provider_output_token_cost_unit,
  currency,
  metadata
) VALUES
  ('price-mock-chat-request', 'model-mock-chat', 'profile-mock-chat-default', 0.001, '1k_tokens', 0.002, '1k_tokens', 0, '1k_tokens', 0, '1k_tokens', 'CREDIT', '{}'),
  ('price-mock-creative-image', 'model-mock-creative', 'profile-mock-creative-default', 0, 'request', 10, 'image', 0, 'request', 0, 'image', 'CREDIT', '{}');

INSERT INTO wallets (id, project_id, paid_credits, promotional_credits) VALUES
  ('wallet-demo', 'project-demo', 10000, 5000);

INSERT INTO ledger_transactions (id, wallet_id, transaction_type, amount, credit_type, status, reason, idempotency_key, metadata) VALUES
  ('ledger-dev-paid', 'wallet-demo', 'grant', 10000, 'paid', 'posted', 'dev seed paid credits', 'dev-seed-paid-credit-grant', '{}'),
  ('ledger-dev-promo', 'wallet-demo', 'grant', 5000, 'promotional', 'posted', 'dev seed promotional credits', 'dev-seed-promo-credit-grant', '{}');

INSERT INTO coupons (id, code, credit_amount, status, metadata) VALUES
  ('coupon-dev-welcome', 'DEV-WELCOME', 1000, 'active', '{"description":"Development welcome coupon"}');

INSERT INTO conversations (id, project_id, title, status) VALUES
  ('conversation-demo', 'project-demo', 'Seeded demo conversation', 'active');

INSERT INTO conversation_branches (id, conversation_id, parent_branch_id, name) VALUES
  ('branch-demo-main', 'conversation-demo', NULL, 'Main');

INSERT INTO messages (id, conversation_id, branch_id, role, content, model_profile_id, metadata) VALUES
  ('message-demo-user', 'conversation-demo', 'branch-demo-main', 'user', 'What can I do in this model market?', NULL, '{}'),
  ('message-demo-assistant', 'conversation-demo', 'branch-demo-main', 'assistant', 'You can browse models, use the workbench, create API keys, and consume credits through mocked provider calls.', 'profile-mock-chat-default', '{"provider":"mock-provider"}');

INSERT INTO workspace_assets (id, project_id, conversation_id, asset_type, storage_path, mime_type, size_bytes, metadata) VALUES
  ('asset-demo-text', 'project-demo', 'conversation-demo', 'upload', 'mock://assets/demo.txt', 'text/plain', 128, '{"mock":true}');

INSERT INTO message_attachments (id, message_id, asset_id) VALUES
  ('attachment-demo-text', 'message-demo-user', 'asset-demo-text');

INSERT INTO file_extractions (id, asset_id, extracted_text, metadata) VALUES
  ('extraction-demo-text', 'asset-demo-text', 'This is mocked extracted text for local development.', '{}');

INSERT INTO embedding_records (id, project_id, source_type, source_id, embedding_model, vector_ref, metadata) VALUES
  ('embedding-demo-text', 'project-demo', 'asset', 'asset-demo-text', 'mock-embedding', 'mock://vectors/embedding-demo-text', '{}');

INSERT INTO inference_requests (id, project_id, model_slug, model_profile_id, provider_slug, status, input_units, output_units, customer_charge, provider_cost, margin, metadata) VALUES
  ('inference-demo', 'project-demo', 'mock-chat', 'profile-mock-chat-default', 'mock-provider', 'succeeded', 12, 24, 1, 0, 1, '{"mock":true}');

INSERT INTO provider_attempts (id, inference_request_id, provider_id, status, latency_ms, provider_request_id, error_class, metadata) VALUES
  ('attempt-demo', 'inference-demo', 'provider-mock', 'succeeded', 42, 'mock-request-1', NULL, '{}');

INSERT INTO usage_events (id, project_id, inference_request_id, model_slug, provider_slug, event_type, customer_charge, provider_cost, metadata) VALUES
  ('usage-demo', 'project-demo', 'inference-demo', 'mock-chat', 'mock-provider', 'chat_completion', 1, 0, '{"mock":true}');

INSERT INTO async_jobs (id, project_id, job_type, status, model_slug, provider_slug, metadata) VALUES
  ('job-demo-image', 'project-demo', 'image_generation', 'completed', 'mock-creative', 'mock-provider', '{"mock":true}');

INSERT INTO routing_policies (id, project_id, name, policy_data, status) VALUES
  ('routing-demo-default', 'project-demo', 'Default dev routing', '{"mode":"fixed","provider":"mock-provider"}', 'active');

INSERT INTO budget_policies (id, project_id, name, limit_credits, period, status) VALUES
  ('budget-demo-monthly', 'project-demo', 'Monthly dev budget', 100000, 'month', 'active');

INSERT INTO webhook_endpoints (id, project_id, url, secret_ref, status) VALUES
  ('webhook-demo', 'project-demo', 'https://example.com/webhooks/model-market', 'WEBHOOK_DEMO_SECRET', 'inactive');

INSERT INTO audit_logs (id, actor_user_id, organization_id, action, target_type, target_id, metadata) VALUES
  ('audit-demo-seed', 'user-admin', 'org-demo', 'seed.loaded', 'project', 'project-demo', '{"source":"populate_test_data.sql"}');

INSERT INTO provider_settlements (id, provider_id, period_start, period_end, amount_cents, currency, status, metadata) VALUES
  ('settlement-mock-june', 'provider-mock', '2026-06-01 00:00:00', '2026-06-30 23:59:59', 0, 'USD', 'draft', '{"mock":true}');
