-- delete from model_configurations;
-- delete from message_attachments;
-- delete from audit_logs;
-- delete from file_extractions;
-- delete from provider_endpoints;
-- delete from messages;
-- delete from workspace_assets;
-- delete from user_interaction_history;
-- delete from provider_health_events;
-- delete from notifications;
-- delete from provider_settlements;
-- delete from webhook_deliveries;
-- delete from webhook_endpoints;
-- delete from budget_policies;
-- delete from routing_policies;
-- delete from coupons;
-- delete from invoice_items;
-- delete from invoices;
-- delete from payment_transactions;
-- delete from payment_methods;
-- delete from payments;
-- delete from usage_events;
-- delete from job_events;
-- delete from async_jobs;
-- delete from provider_attempts;
-- delete from inference_requests;
-- delete from embedding_records;
-- delete from uploaded_files;
-- delete from context_summaries;
-- delete from conversation_branches;
-- delete from conversations;
-- delete from ledger_transactions;
-- delete from wallets;
-- delete from price_rules;
-- delete from model_profiles;
-- delete from model_versions;
-- delete from models;
-- delete from providers;
-- delete from api_keys;
-- delete from sessions;
-- delete from oauth_accounts;
-- delete from memberships;
-- delete from projects;
-- delete from organizations;
-- delete from users; 

insert into users(id, email, name, status) values
	('user_admin_example_com', 'admin@example.com', 'Admin User', 'active'),
	('user_developer_example_com', 'developer@example.com', 'Developer User', 'active');

insert into organizations(id, name, slug) values
	('org_demo-org', 'Demo Organization', 'demo-org');

insert into memberships(id, user_id, organization_id, role) values
	('membership_admin_example_com', 'user_admin_example_com', 'org_demo-org', 'owner'),
	('membership_developer_example_com', 'user_developer_example_com', 'org_demo-org', 'developer');

insert into projects(id, organization_id, name, slug, environment, retention_policy) values
	('project_demo-project', 'org_demo-org', 'Demo Project', 'demo-project', 'dev', '{}');

insert into api_keys(id, project_id, name, prefix, key_hash, scopes, status) values
	('api_key_dev_test', 'project_demo-project', 'Dev test API key', 'mk_dev_tes', '8f10d494a55d6262aded7008470955c7b6175fa6afc719e837f3634c0e4d19aa', 'models:read,chat:create', 'active');

insert into wallets(id, project_id, paid_credits, promotional_credits) values
	('wallet_demo-project', 'project_demo-project', 10000, 5000);

insert into ledger_transactions(id, wallet_id, transaction_type, amount, credit_type, status, reason, idempotency_key, metadata) values
	('ledger_dev_test_credit_grant', 'wallet_demo-project', 'grant', 5000, 'promotional', 'posted', 'dev test credits', 'dev-test-credit-grant', '{}');

insert into providers(id, slug, name, status, endpoint_url, credential_ref, metadata) values
	('provider_mock-provider', 'mock-provider', 'Mock Provider', 'active', 'mock://provider', null, '{"mode":"dev","supports_streaming":true}'),
	('provider_openai-placeholder', 'openai-placeholder', 'OpenAI Placeholder', 'active', 'https://api.openai.com', 'OPENAI_API_KEY', '{"credential_ref":"OPENAI_API_KEY","enabled_by_default":false}');

insert into models(id, provider_id, slug, name, modality, status, context_window, capabilities, metadata) values
	('model_mock-chat', 'provider_mock-provider', 'mock-chat', 'Mock Chat', 'chat', 'public', 8192, '{"chat":true,"streaming":true,"files":false}', '{}'),
	('model_mock-creative', 'provider_mock-provider', 'mock-creative', 'Mock Creative', 'image', 'public', 0, '{"image_generation":true,"async_jobs":true}', '{}');

insert into model_versions(id, model_id, version, status, metadata) values
	('model_version_mock-chat_dev', 'model_mock-chat', 'dev', 'active', '{}'),
	('model_version_mock-creative_dev', 'model_mock-creative', 'dev', 'active', '{}');

insert into model_profiles(id, model_id, slug, name, status, system_prompt, default_parameters, safety_settings, config_version) values
	('profile_mock-chat-default', 'model_mock-chat', 'mock-chat-default', 'Mock Chat Default', 'public', 'You are a helpful mocked assistant for local development.', '{"temperature":0.2,"max_tokens":512}', '{}', 1),
	('profile_mock-creative-default', 'model_mock-creative', 'mock-creative-default', 'Mock Creative Default', 'public', 'Generate deterministic local development assets.', '{"size":"1024x1024"}', '{}', 1);

insert into price_rules(
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
) values
	('price_mock-chat-default', 'model_mock-chat', 'profile_mock-chat-default', 0.10, '1k_tokens', 0.60, '1k_tokens', 0, '1k_tokens', 0, '1k_tokens', 'CREDIT', '{}'),
	('price_mock-creative-default', 'model_mock-creative', 'profile_mock-creative-default', 0, 'prompt', 10.00, 'image_1024', 0, 'prompt', 0, 'image_1024', 'CREDIT', '{}');

insert into conversations(id, project_id, title, status) values
	('conversation_seed_demo', 'project_demo-project', 'Dev test demo conversation', 'active');

insert into messages(id, conversation_id, role, content, model_profile_id, metadata) values
	('message_seed_1', 'conversation_seed_demo', 'user', 'What can I do in this model market?', null, '{}'),
	('message_seed_2', 'conversation_seed_demo', 'assistant', 'You can browse models, use the workbench, create API keys, and consume credits through mocked provider calls.', 'profile_mock-chat-default', '{}');

insert into usage_events(id, project_id, inference_request_id, model_slug, provider_slug, event_type, customer_charge, provider_cost, metadata) values
	('usage_dev_test_1', 'project_demo-project', null, 'mock-chat-default', 'mock-provider', 'dev_test_demo_usage', 1, 0, '{"source":"populate_test_data.sql"}');

insert into routing_policies(id, project_id, name, status, policy) values
	('routing_policy_demo_default', 'project_demo-project', 'Demo default routing', 'active', '{"mode":"fixed","provider":"mock-provider"}');

insert into budget_policies(id, project_id, name, monthly_credit_limit, status, metadata) values
	('budget_policy_demo_monthly', 'project_demo-project', 'Demo monthly budget', 100000, 'active', '{}');

insert into coupons(id, code, status, credit_amount, metadata) values
	('coupon_demo_credits', 'DEV-CREDITS', 'active', 1000, '{"source":"populate_test_data.sql"}');

insert into payment_methods(id, organization_id, project_id, user_id, provider, provider_payment_method_id, method_type, display_name, last_four, exp_month, exp_year, billing_email, status, is_default, metadata) values
	('payment_method_demo_card', 'org_demo-org', 'project_demo-project', 'user_admin_example_com', 'mock-payment', 'pm_mock_demo_card', 'card', 'Mock Visa ending 4242', '4242', 12, 2030, 'admin@example.com', 'active', 1, '{"source":"populate_test_data.sql"}');

insert into payments(id, project_id, provider, provider_payment_id, status, amount_cents, currency, credits_granted, metadata) values
	('payment_demo_topup', 'project_demo-project', 'mock-payment', 'pay_mock_demo_topup', 'succeeded', 10000, 'USD', 10000, '{"source":"populate_test_data.sql"}');

insert into payment_transactions(id, payment_id, payment_method_id, project_id, transaction_type, provider, provider_transaction_id, status, amount_cents, currency, credits_delta, idempotency_key, metadata) values
	('payment_txn_demo_topup', 'payment_demo_topup', 'payment_method_demo_card', 'project_demo-project', 'top_up', 'mock-payment', 'txn_mock_demo_topup', 'succeeded', 10000, 'USD', 10000, 'payment-txn-demo-topup', '{"source":"populate_test_data.sql"}');

insert into user_interaction_history(id, user_id, organization_id, project_id, session_id, interaction_type, surface, target_type, target_id, event_name, metadata) values
	('interaction_dev_test_login', 'user_admin_example_com', 'org_demo-org', 'project_demo-project', null, 'auth', 'dashboard', 'user', 'user_admin_example_com', 'dev_login', '{"source":"populate_test_data.sql"}'),
	('interaction_dev_test_catalog', 'user_admin_example_com', 'org_demo-org', 'project_demo-project', null, 'view', 'catalog', 'model', 'model_mock-chat', 'view_model_catalog', '{"source":"populate_test_data.sql"}'),
	('interaction_dev_test_workbench', 'user_admin_example_com', 'org_demo-org', 'project_demo-project', null, 'message', 'workbench', 'conversation', 'conversation_seed_demo', 'send_chat_message', '{"source":"populate_test_data.sql"}');
