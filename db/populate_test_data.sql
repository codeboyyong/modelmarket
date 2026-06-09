-- Portable deterministic test data for Model Market.
-- IDs are explicit so this file does not require database-specific UUID functions.
--see db/delete_all_data.sql
-- DELETE FROM sys_provider_settlements;
-- DELETE FROM sys_audit_logs;
-- DELETE FROM user_webhook_endpoints;
-- DELETE FROM user_budget_policies;
-- DELETE FROM user_routing_policies;
-- DELETE FROM user_async_jobs;
-- DELETE FROM user_usage_events;
-- DELETE FROM user_provider_attempts;
-- DELETE FROM user_embedding_records;
-- DELETE FROM user_file_extractions;
-- DELETE FROM user_message_attachments;
-- DELETE FROM user_workspace_assets;
-- DELETE FROM user_messages;
-- DELETE FROM user_inference_requests;
-- DELETE FROM user_conversation_branches;
-- DELETE FROM user_conversations;
-- DELETE FROM sys_coupons;
-- DELETE FROM user_invoices;
-- DELETE FROM user_payments;
-- DELETE FROM user_credit_purchases;
-- DELETE FROM user_ledger_transactions;
-- DELETE FROM user_wallets;
-- DELETE FROM sys_price_rules;
-- DELETE FROM sys_pricing_plans;
-- DELETE FROM sys_model_configurations;
-- DELETE FROM sys_model_profiles;
-- DELETE FROM sys_model_versions;
-- DELETE FROM sys_models;
-- DELETE FROM sys_capabilities;
-- DELETE FROM sys_provider_endpoints;
-- DELETE FROM sys_providers;
-- DELETE FROM user_api_keys;
-- DELETE FROM user_projects;
-- DELETE FROM user_companies;
-- DELETE FROM sys_memberships;
-- DELETE FROM sys_organizations;
-- DELETE FROM sys_sessions;
-- DELETE FROM sys_oauth_accounts;
-- DELETE FROM sys_users;
-- DELETE FROM sys_config;
-- DELETE FROM sys_roles;

INSERT INTO sys_config (conf_key, conf_value, description) VALUES
  ('usd_to_credit_ratio', '1', 'Number of paid credits granted per 1 USD for fake local purchases'),
  ('default_currency', 'USD', 'Default billing currency for local/demo purchases'),
  ('fake_payments_enabled', 'true', 'Allows local/demo purchases to succeed without a real payment provider'),
  ('model_catalog_cache_ttl_seconds', '300', 'Cache lifetime for model catalog responses'),
  ('pricing_cache_ttl_seconds', '300', 'Cache lifetime for pricing responses'),
  ('user_session_ttl_hours', '24', 'Default login session lifetime'),
  ('password_min_length', '8', 'Minimum password length for account password validation'),
  ('max_upload_file_mb', '200', 'Maximum supported upload size for workbench assets'),
  ('allowed_upload_mime_types', 'image/*,audio/*,video/*', 'Comma-separated MIME type allowlist for uploaded conversation assets'),
  ('s3_bucket_name', 'model-market-dev-assets', 'Default S3 bucket name for uploaded and generated workbench assets'),
  ('s3_asset_url_ttl_seconds', '3600', 'Signed asset download URL lifetime when private S3 URLs are enabled'),
  ('default_project_credit_limit', '0', 'Default project credit cap; 0 means no cap'),
  ('low_credit_warning_threshold', '100', 'Credit balance threshold used to warn users about low balance'),
  ('oauth_google_enabled', 'true', 'Enables Google login in the UI'),
  ('oauth_github_enabled', 'true', 'Enables GitHub login in the UI'),
  ('oauth_facebook_enabled', 'true', 'Enables Facebook login in the UI'),
  ('maintenance_mode', 'false', 'Disables write actions when enabled'),
  ('support_email', 'support@example.com', 'Default support email shown by contact surfaces');

INSERT INTO sys_roles (id, name, description) VALUES
  ('role-owner', 'owner', 'Organization owner'),
  ('role-admin', 'admin', 'Organization administrator'),
  ('role-developer', 'developer', 'Developer user'),
  ('role-readonly', 'readonly', 'Read-only user');

INSERT INTO sys_users (id, email, name, avatar_url, status, password_hash, user_type, company_id, ui_theme, language) VALUES
  ('user-admin', 'admin@example.com', 'Admin User', NULL, 'active', '0f694fc3267fd0e17454bbe6dd6b4e163a41519495539195fd1567b476a15d75', 'sys_admin', NULL, 'Light', 'EN'),
  ('user-developer', 'developer@example.com', 'Developer User', NULL, 'active', '0f694fc3267fd0e17454bbe6dd6b4e163a41519495539195fd1567b476a15d75', 'individual_consumer', NULL, 'Light', 'EN'),
  ('user-yong-zhao', 'yong_zhao@example.com', 'yong_zhao', NULL, 'active', '0f694fc3267fd0e17454bbe6dd6b4e163a41519495539195fd1567b476a15d75', 'individual_consumer', NULL, 'Light', 'EN'),
  ('user-corp-admin', 'corp-admin@example.com', 'Corporate Admin', NULL, 'active', '0f694fc3267fd0e17454bbe6dd6b4e163a41519495539195fd1567b476a15d75', 'corporate_admin', 'company-acme', 'Light', 'EN'),
  ('user-corp-designer', 'designer@acme.example', 'Acme Designer', NULL, 'active', '0f694fc3267fd0e17454bbe6dd6b4e163a41519495539195fd1567b476a15d75', 'corporate_member', 'company-acme', 'Light', 'EN'),
  ('user-corp-producer', 'producer@acme.example', 'Acme Producer', NULL, 'active', '0f694fc3267fd0e17454bbe6dd6b4e163a41519495539195fd1567b476a15d75', 'corporate_member', 'company-acme', 'Light', 'EN');

INSERT INTO user_companies (id, name, owner_user_id, status) VALUES
  ('company-acme', 'Acme Creative Studio', 'user-corp-admin', 'active');

INSERT INTO sys_oauth_accounts (id, user_id, provider, provider_account_id, email, display_name) VALUES
  ('oauth-google-admin', 'user-admin', 'google', 'google-admin-dev', 'admin@example.com', 'Admin User'),
  ('oauth-github-admin', 'user-admin', 'github', 'github-admin-dev', 'admin@example.com', 'Admin User'),
  ('oauth-facebook-dev', 'user-developer', 'facebook', 'facebook-dev-dev', 'developer@example.com', 'Developer User'),
  ('oauth-google-corp-admin', 'user-corp-admin', 'google', 'google-corp-admin-dev', 'corp-admin@example.com', 'Corporate Admin');

INSERT INTO sys_organizations (id, name, slug, status) VALUES
  ('org-demo', 'Demo Organization', 'demo-org', 'active'),
  ('org-acme', 'Acme Creative Studio', 'acme-creative-studio', 'active');

INSERT INTO sys_memberships (id, user_id, organization_id, role) VALUES
  ('membership-admin', 'user-admin', 'org-demo', 'owner'),
  ('membership-developer', 'user-developer', 'org-demo', 'developer'),
  ('membership-yong-zhao', 'user-yong-zhao', 'org-demo', 'developer'),
  ('membership-corp-admin', 'user-corp-admin', 'org-acme', 'owner'),
  ('membership-corp-designer', 'user-corp-designer', 'org-acme', 'developer'),
  ('membership-corp-producer', 'user-corp-producer', 'org-acme', 'developer');

INSERT INTO user_projects (id, organization_id, company_id, name, slug, environment, retention_policy) VALUES
  ('project-demo', 'org-demo', NULL, 'Demo Project', 'demo-project', 'dev', '{"conversation_days":30,"asset_days":30}'),
  ('project-image-studio', 'org-demo', NULL, 'Image Studio Demo', 'image-studio-demo', 'dev', '{"conversation_days":30,"asset_days":30}'),
  ('project-video-lab', 'org-demo', NULL, 'Video Lab Demo', 'video-lab-demo', 'dev', '{"conversation_days":30,"asset_days":30}'),
  ('project-acme-brand', 'org-acme', 'company-acme', 'Acme Brand Studio', 'acme-brand-studio', 'dev', '{"conversation_days":90,"asset_days":90}'),
  ('project-acme-video', 'org-acme', 'company-acme', 'Acme Video Team', 'acme-video-team', 'dev', '{"conversation_days":90,"asset_days":90}');

INSERT INTO user_api_keys (id, project_id, name, prefix, key_hash, scopes, status) VALUES
  ('api-key-demo', 'project-demo', 'Seeded development key', 'mk_seeded', 'seeded-key-hash-replace-before-real-use', 'models:read,chat:create', 'active'),
  ('api-key-image-studio', 'project-image-studio', 'Image Studio demo key', 'mk_imgdemo', 'image-studio-demo-key-hash', 'models:read,chat:create', 'active'),
  ('api-key-video-lab', 'project-video-lab', 'Video Lab demo key', 'mk_viddemo', 'video-lab-demo-key-hash', 'models:read,chat:create', 'active'),
  ('api-key-acme-brand', 'project-acme-brand', 'Acme Brand demo key', 'mk_acmebr', 'acme-brand-demo-key-hash', 'models:read,chat:create', 'active'),
  ('api-key-acme-video', 'project-acme-video', 'Acme Video demo key', 'mk_acmevd', 'acme-video-demo-key-hash', 'models:read,chat:create', 'active');

INSERT INTO sys_providers (id, slug, name, status, endpoint_url, credential_ref, metadata) VALUES
  ('provider-mock', 'mock-provider', 'Mock Provider', 'active', 'mock://provider', NULL, '{"mode":"dev","supports_streaming":true}'),
  ('provider-openrouter', 'openrouter', 'OpenRouter', 'active', 'https://openrouter.ai/api/v1', 'OPENROUTER_API_KEY', '{"source":"https://openrouter.ai/models","supports_multimodal":true}'),
  ('provider-openai-placeholder', 'openai-placeholder', 'OpenAI Placeholder', 'inactive', 'https://api.openai.com', 'OPENAI_API_KEY', '{"enabled_by_default":false}');

INSERT INTO sys_provider_endpoints (id, provider_id, name, endpoint_url, region, status, metadata) VALUES
  ('endpoint-mock-default', 'provider-mock', 'Mock Default Endpoint', 'mock://provider/default', 'local', 'active', '{}'),
  ('endpoint-openrouter-default', 'provider-openrouter', 'OpenRouter Default Endpoint', 'https://openrouter.ai/api/v1', 'global', 'active', '{"models_url":"https://openrouter.ai/api/v1/models","video_models_url":"https://openrouter.ai/api/v1/videos/models"}'),
  ('endpoint-openai-default', 'provider-openai-placeholder', 'OpenAI Default Endpoint', 'https://api.openai.com', 'global', 'inactive', '{}');

INSERT INTO sys_capabilities (id, slug, name, description) VALUES
  ('cap-chat', 'chat', 'Chat', 'Chat completion support'),
  ('cap-streaming', 'streaming', 'Streaming', 'Streaming response support'),
  ('cap-image-generation', 'image-generation', 'Image Generation', 'Image generation support'),
  ('cap-async-jobs', 'async-jobs', 'Async Jobs', 'Long-running async job support');

INSERT INTO sys_models (id, provider_id, slug, name, modality, status, context_window, capabilities, metadata) VALUES
  ('model-mock-chat', 'provider-mock', 'mock-chat', 'Mock Chat', 'chat', 'public', 8192, '{"chat":true,"streaming":true,"files":false}', '{"quality_tier":"dev"}'),
  ('model-mock-creative', 'provider-mock', 'mock-creative', 'Mock Creative', 'image', 'public', 0, '{"image_generation":true,"async_jobs":true}', '{"quality_tier":"dev"}'),
  ('model-openrouter-riverflow-v25-pro-free', 'provider-openrouter', 'sourceful/riverflow-v2.5-pro:free', 'Sourceful: Riverflow V2.5 Pro (free)', 'image', 'public', 8192, '{"image_generation":true,"image_input":true}', '{"openrouter_id":"sourceful/riverflow-v2.5-pro:free","canonical_slug":"sourceful/riverflow-v2.5-pro-20260605"}'),
  ('model-openrouter-riverflow-v25-fast-free', 'provider-openrouter', 'sourceful/riverflow-v2.5-fast:free', 'Sourceful: Riverflow V2.5 Fast (free)', 'image', 'public', 8192, '{"image_generation":true,"image_input":true}', '{"openrouter_id":"sourceful/riverflow-v2.5-fast:free","canonical_slug":"sourceful/riverflow-v2.5-fast-20260605"}'),
  ('model-openrouter-mai-image-25', 'provider-openrouter', 'microsoft/mai-image-2.5', 'Microsoft: MAI-Image-2.5', 'image', 'public', 4096, '{"image_generation":true,"image_input":true}', '{"openrouter_id":"microsoft/mai-image-2.5","canonical_slug":"microsoft/mai-image-2.5"}'),
  ('model-openrouter-grok-imagine-image-quality', 'provider-openrouter', 'x-ai/grok-imagine-image-quality', 'xAI: Grok Imagine Image Quality', 'image', 'public', 65536, '{"image_generation":true,"image_input":true}', '{"openrouter_id":"x-ai/grok-imagine-image-quality","canonical_slug":"x-ai/grok-imagine-image-quality-20260512"}'),
  ('model-openrouter-grok-imagine-video', 'provider-openrouter', 'x-ai/grok-imagine-video', 'xAI: Grok Imagine Video', 'video', 'public', 0, '{"video_generation":true,"image_input":true,"reference_input":true}', '{"openrouter_id":"x-ai/grok-imagine-video","canonical_slug":"x-ai/grok-imagine-video-20260512"}'),
  ('model-openrouter-kling-v30-pro', 'provider-openrouter', 'kwaivgi/kling-v3.0-pro', 'Kling: Video v3.0 Pro', 'video', 'public', 0, '{"video_generation":true,"image_input":true,"audio_output":true}', '{"openrouter_id":"kwaivgi/kling-v3.0-pro","canonical_slug":"kwaivgi/kling-v3.0-pro-20260429"}'),
  ('model-openrouter-kling-v30-std', 'provider-openrouter', 'kwaivgi/kling-v3.0-std', 'Kling: Video v3.0 Standard', 'video', 'public', 0, '{"video_generation":true,"image_input":true,"audio_output":true}', '{"openrouter_id":"kwaivgi/kling-v3.0-std","canonical_slug":"kwaivgi/kling-v3.0-std-20260429"}'),
  ('model-openrouter-veo-31-fast', 'provider-openrouter', 'google/veo-3.1-fast', 'Google: Veo 3.1 Fast', 'video', 'public', 0, '{"video_generation":true,"image_input":true,"audio_output":true}', '{"openrouter_id":"google/veo-3.1-fast","canonical_slug":"google/veo-3.1-fast-20260320"}'),
  ('model-openrouter-lyria-3-pro-preview', 'provider-openrouter', 'google/lyria-3-pro-preview', 'Google: Lyria 3 Pro Preview', 'audio', 'public', 1048576, '{"audio_generation":true,"music_generation":true,"image_input":true}', '{"openrouter_id":"google/lyria-3-pro-preview","canonical_slug":"google/lyria-3-pro-preview-20260330"}'),
  ('model-openrouter-lyria-3-clip-preview', 'provider-openrouter', 'google/lyria-3-clip-preview', 'Google: Lyria 3 Clip Preview', 'audio', 'public', 1048576, '{"audio_generation":true,"music_generation":true,"image_input":true}', '{"openrouter_id":"google/lyria-3-clip-preview","canonical_slug":"google/lyria-3-clip-preview-20260330"}'),
  ('model-openrouter-gpt-audio', 'provider-openrouter', 'openai/gpt-audio', 'OpenAI: GPT Audio', 'audio', 'public', 128000, '{"audio_generation":true,"audio_input":true,"chat":true}', '{"openrouter_id":"openai/gpt-audio","canonical_slug":"openai/gpt-audio"}'),
  ('model-openrouter-gpt-audio-mini', 'provider-openrouter', 'openai/gpt-audio-mini', 'OpenAI: GPT Audio Mini', 'audio', 'public', 128000, '{"audio_generation":true,"audio_input":true,"chat":true}', '{"openrouter_id":"openai/gpt-audio-mini","canonical_slug":"openai/gpt-audio-mini"}');

INSERT INTO sys_model_versions (id, model_id, version, status, metadata) VALUES
  ('model-version-mock-chat-dev', 'model-mock-chat', 'dev', 'active', '{}'),
  ('model-version-mock-creative-dev', 'model-mock-creative', 'dev', 'active', '{}'),
  ('model-version-openrouter-riverflow-v25-pro-free', 'model-openrouter-riverflow-v25-pro-free', '20260605', 'active', '{}'),
  ('model-version-openrouter-riverflow-v25-fast-free', 'model-openrouter-riverflow-v25-fast-free', '20260605', 'active', '{}'),
  ('model-version-openrouter-mai-image-25', 'model-openrouter-mai-image-25', 'current', 'active', '{}'),
  ('model-version-openrouter-grok-imagine-image-quality', 'model-openrouter-grok-imagine-image-quality', '20260512', 'active', '{}'),
  ('model-version-openrouter-grok-imagine-video', 'model-openrouter-grok-imagine-video', '20260512', 'active', '{}'),
  ('model-version-openrouter-kling-v30-pro', 'model-openrouter-kling-v30-pro', '20260429', 'active', '{}'),
  ('model-version-openrouter-kling-v30-std', 'model-openrouter-kling-v30-std', '20260429', 'active', '{}'),
  ('model-version-openrouter-veo-31-fast', 'model-openrouter-veo-31-fast', '20260320', 'active', '{}'),
  ('model-version-openrouter-lyria-3-pro-preview', 'model-openrouter-lyria-3-pro-preview', '20260330', 'active', '{}'),
  ('model-version-openrouter-lyria-3-clip-preview', 'model-openrouter-lyria-3-clip-preview', '20260330', 'active', '{}'),
  ('model-version-openrouter-gpt-audio', 'model-openrouter-gpt-audio', 'current', 'active', '{}'),
  ('model-version-openrouter-gpt-audio-mini', 'model-openrouter-gpt-audio-mini', 'current', 'active', '{}');

INSERT INTO sys_model_profiles (id, model_id, slug, name, status, system_prompt, default_parameters, safety_settings, config_version) VALUES
  ('profile-mock-chat-default', 'model-mock-chat', 'mock-chat-default', 'Mock Chat Default', 'public', 'You are a helpful mocked assistant for local development.', '{"temperature":0.2,"max_tokens":512}', '{"moderation":"mock"}', 1),
  ('profile-mock-creative-default', 'model-mock-creative', 'mock-creative-default', 'Mock Creative Default', 'public', 'Generate deterministic local development assets.', '{"size":"1024x1024"}', '{"moderation":"mock"}', 1),
  ('profile-openrouter-riverflow-v25-pro-free-default', 'model-openrouter-riverflow-v25-pro-free', 'sourceful-riverflow-v2-5-pro-free-default', 'Sourceful: Riverflow V2.5 Pro (free) Default', 'public', 'Generate high-quality images through OpenRouter.', '{"modalities":["image"],"size":"1024x1024"}', '{"moderation":"provider"}', 1),
  ('profile-openrouter-riverflow-v25-fast-free-default', 'model-openrouter-riverflow-v25-fast-free', 'sourceful-riverflow-v2-5-fast-free-default', 'Sourceful: Riverflow V2.5 Fast (free) Default', 'public', 'Generate fast images through OpenRouter.', '{"modalities":["image"],"size":"1024x1024"}', '{"moderation":"provider"}', 1),
  ('profile-openrouter-mai-image-25-default', 'model-openrouter-mai-image-25', 'microsoft-mai-image-2-5-default', 'Microsoft: MAI-Image-2.5 Default', 'public', 'Generate images through OpenRouter.', '{"modalities":["image"],"size":"1024x1024"}', '{"moderation":"provider"}', 1),
  ('profile-openrouter-grok-imagine-image-quality-default', 'model-openrouter-grok-imagine-image-quality', 'xai-grok-imagine-image-quality-default', 'xAI: Grok Imagine Image Quality Default', 'public', 'Generate high-fidelity images through OpenRouter.', '{"modalities":["image"],"size":"1024x1024"}', '{"moderation":"provider"}', 1),
  ('profile-openrouter-grok-imagine-video-default', 'model-openrouter-grok-imagine-video', 'xai-grok-imagine-video-default', 'xAI: Grok Imagine Video Default', 'public', 'Generate video through OpenRouter.', '{"duration_seconds":5,"resolution":"720p","aspect_ratio":"16:9"}', '{"moderation":"provider"}', 1),
  ('profile-openrouter-kling-v30-pro-default', 'model-openrouter-kling-v30-pro', 'kling-video-v3-0-pro-default', 'Kling: Video v3.0 Pro Default', 'public', 'Generate video through OpenRouter.', '{"duration_seconds":5,"resolution":"720p","aspect_ratio":"16:9"}', '{"moderation":"provider"}', 1),
  ('profile-openrouter-kling-v30-std-default', 'model-openrouter-kling-v30-std', 'kling-video-v3-0-standard-default', 'Kling: Video v3.0 Standard Default', 'public', 'Generate video through OpenRouter.', '{"duration_seconds":5,"resolution":"720p","aspect_ratio":"16:9"}', '{"moderation":"provider"}', 1),
  ('profile-openrouter-veo-31-fast-default', 'model-openrouter-veo-31-fast', 'google-veo-3-1-fast-default', 'Google: Veo 3.1 Fast Default', 'public', 'Generate video through OpenRouter.', '{"duration_seconds":4,"resolution":"720p","aspect_ratio":"16:9"}', '{"moderation":"provider"}', 1),
  ('profile-openrouter-lyria-3-pro-preview-default', 'model-openrouter-lyria-3-pro-preview', 'google-lyria-3-pro-preview-default', 'Google: Lyria 3 Pro Preview Default', 'public', 'Generate music and audio through OpenRouter.', '{"modalities":["text","audio"],"duration":"full_song"}', '{"moderation":"provider"}', 1),
  ('profile-openrouter-lyria-3-clip-preview-default', 'model-openrouter-lyria-3-clip-preview', 'google-lyria-3-clip-preview-default', 'Google: Lyria 3 Clip Preview Default', 'public', 'Generate short music clips through OpenRouter.', '{"modalities":["text","audio"],"duration_seconds":30}', '{"moderation":"provider"}', 1),
  ('profile-openrouter-gpt-audio-default', 'model-openrouter-gpt-audio', 'openai-gpt-audio-default', 'OpenAI: GPT Audio Default', 'public', 'Generate and process audio through OpenRouter.', '{"modalities":["text","audio"],"voice":"alloy"}', '{"moderation":"provider"}', 1),
  ('profile-openrouter-gpt-audio-mini-default', 'model-openrouter-gpt-audio-mini', 'openai-gpt-audio-mini-default', 'OpenAI: GPT Audio Mini Default', 'public', 'Generate and process audio through OpenRouter.', '{"modalities":["text","audio"],"voice":"alloy"}', '{"moderation":"provider"}', 1);

INSERT INTO sys_model_configurations (id, model_profile_id, version, config_data, status) VALUES
  ('config-mock-chat-v1', 'profile-mock-chat-default', 1, '{"system_prompt":"You are a helpful mocked assistant for local development.","temperature":0.2,"max_tokens":512}', 'published'),
  ('config-mock-creative-v1', 'profile-mock-creative-default', 1, '{"system_prompt":"Generate deterministic local development assets.","size":"1024x1024"}', 'published'),
  ('config-openrouter-riverflow-v25-pro-free-v1', 'profile-openrouter-riverflow-v25-pro-free-default', 1, '{"model":"sourceful/riverflow-v2.5-pro:free","modalities":["image"],"size":"1024x1024"}', 'published'),
  ('config-openrouter-riverflow-v25-fast-free-v1', 'profile-openrouter-riverflow-v25-fast-free-default', 1, '{"model":"sourceful/riverflow-v2.5-fast:free","modalities":["image"],"size":"1024x1024"}', 'published'),
  ('config-openrouter-mai-image-25-v1', 'profile-openrouter-mai-image-25-default', 1, '{"model":"microsoft/mai-image-2.5","modalities":["image"],"size":"1024x1024"}', 'published'),
  ('config-openrouter-grok-imagine-image-quality-v1', 'profile-openrouter-grok-imagine-image-quality-default', 1, '{"model":"x-ai/grok-imagine-image-quality","modalities":["image"],"size":"1024x1024"}', 'published'),
  ('config-openrouter-grok-imagine-video-v1', 'profile-openrouter-grok-imagine-video-default', 1, '{"model":"x-ai/grok-imagine-video","duration_seconds":5,"resolution":"720p","aspect_ratio":"16:9"}', 'published'),
  ('config-openrouter-kling-v30-pro-v1', 'profile-openrouter-kling-v30-pro-default', 1, '{"model":"kwaivgi/kling-v3.0-pro","duration_seconds":5,"resolution":"720p","aspect_ratio":"16:9"}', 'published'),
  ('config-openrouter-kling-v30-std-v1', 'profile-openrouter-kling-v30-std-default', 1, '{"model":"kwaivgi/kling-v3.0-std","duration_seconds":5,"resolution":"720p","aspect_ratio":"16:9"}', 'published'),
  ('config-openrouter-veo-31-fast-v1', 'profile-openrouter-veo-31-fast-default', 1, '{"model":"google/veo-3.1-fast","duration_seconds":4,"resolution":"720p","aspect_ratio":"16:9"}', 'published'),
  ('config-openrouter-lyria-3-pro-preview-v1', 'profile-openrouter-lyria-3-pro-preview-default', 1, '{"model":"google/lyria-3-pro-preview","modalities":["text","audio"],"duration":"full_song"}', 'published'),
  ('config-openrouter-lyria-3-clip-preview-v1', 'profile-openrouter-lyria-3-clip-preview-default', 1, '{"model":"google/lyria-3-clip-preview","modalities":["text","audio"],"duration_seconds":30}', 'published'),
  ('config-openrouter-gpt-audio-v1', 'profile-openrouter-gpt-audio-default', 1, '{"model":"openai/gpt-audio","modalities":["text","audio"],"voice":"alloy"}', 'published'),
  ('config-openrouter-gpt-audio-mini-v1', 'profile-openrouter-gpt-audio-mini-default', 1, '{"model":"openai/gpt-audio-mini","modalities":["text","audio"],"voice":"alloy"}', 'published');

INSERT INTO sys_pricing_plans (id, slug, name, status, metadata) VALUES
  ('plan-developer', 'developer', 'Developer', 'active', '{"default":true}');

INSERT INTO sys_price_rules (
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
  ('price-mock-creative-image', 'model-mock-creative', 'profile-mock-creative-default', 0, 'request', 10, 'image', 0, 'request', 0, 'image', 'CREDIT', '{}'),
  ('price-openrouter-riverflow-v25-pro-free-image', 'model-openrouter-riverflow-v25-pro-free', 'profile-openrouter-riverflow-v25-pro-free-default', 0, 'request', 0, 'image', 0, 'request', 0, 'image', 'CREDIT', '{"source":"openrouter","pricing":"free"}'),
  ('price-openrouter-riverflow-v25-fast-free-image', 'model-openrouter-riverflow-v25-fast-free', 'profile-openrouter-riverflow-v25-fast-free-default', 0, 'request', 0, 'image', 0, 'request', 0, 'image', 'CREDIT', '{"source":"openrouter","pricing":"free"}'),
  ('price-openrouter-mai-image-25-image', 'model-openrouter-mai-image-25', 'profile-openrouter-mai-image-25-default', 0.005, '1k_tokens', 10, 'image', 0.005, '1k_tokens', 0, 'image', 'CREDIT', '{"source":"openrouter","prompt_price":"0.000005"}'),
  ('price-openrouter-grok-imagine-image-quality-image', 'model-openrouter-grok-imagine-image-quality', 'profile-openrouter-grok-imagine-image-quality-default', 0, 'request', 10, 'image', 0, 'request', 0.01, 'image', 'CREDIT', '{"source":"openrouter","image_price_usd":"0.01"}'),
  ('price-openrouter-grok-imagine-video-second', 'model-openrouter-grok-imagine-video', 'profile-openrouter-grok-imagine-video-default', 0, 'request', 70, 'second_720p', 0, 'request', 0.07, 'second_720p', 'CREDIT', '{"source":"openrouter","cents_per_video_output_second_720p":"7"}'),
  ('price-openrouter-kling-v30-pro-second', 'model-openrouter-kling-v30-pro', 'profile-openrouter-kling-v30-pro-default', 0, 'request', 112, 'second_720p', 0, 'request', 0.112, 'second_720p', 'CREDIT', '{"source":"openrouter","duration_seconds":"0.112"}'),
  ('price-openrouter-kling-v30-std-second', 'model-openrouter-kling-v30-std', 'profile-openrouter-kling-v30-std-default', 0, 'request', 84, 'second_720p', 0, 'request', 0.084, 'second_720p', 'CREDIT', '{"source":"openrouter","duration_seconds":"0.084"}'),
  ('price-openrouter-veo-31-fast-second', 'model-openrouter-veo-31-fast', 'profile-openrouter-veo-31-fast-default', 0, 'request', 100, 'second_720p', 0, 'request', 0.10, 'second_720p', 'CREDIT', '{"source":"openrouter","duration_seconds_with_audio_720p":"0.10"}'),
  ('price-openrouter-lyria-3-pro-preview-song', 'model-openrouter-lyria-3-pro-preview', 'profile-openrouter-lyria-3-pro-preview-default', 0, 'request', 80, 'song', 0, 'request', 0.08, 'song', 'CREDIT', '{"source":"openrouter","song_price_usd":"0.08"}'),
  ('price-openrouter-lyria-3-clip-preview-clip', 'model-openrouter-lyria-3-clip-preview', 'profile-openrouter-lyria-3-clip-preview-default', 0, 'request', 40, 'clip', 0, 'request', 0.04, 'clip', 'CREDIT', '{"source":"openrouter","clip_price_usd":"0.04"}'),
  ('price-openrouter-gpt-audio-audio', 'model-openrouter-gpt-audio', 'profile-openrouter-gpt-audio-default', 0.0025, '1k_tokens', 0.01, '1k_tokens', 0.0025, '1k_tokens', 0.01, '1k_tokens', 'CREDIT', '{"source":"openrouter","audio_price":"0.000032"}'),
  ('price-openrouter-gpt-audio-mini-audio', 'model-openrouter-gpt-audio-mini', 'profile-openrouter-gpt-audio-mini-default', 0.0006, '1k_tokens', 0.0024, '1k_tokens', 0.0006, '1k_tokens', 0.0024, '1k_tokens', 'CREDIT', '{"source":"openrouter","audio_price":"0.0000006"}');

INSERT INTO user_wallets (id, project_id, company_id, paid_credits, promotional_credits) VALUES
  ('wallet-demo', 'project-demo', NULL, 10000, 5000),
  ('wallet-image-studio', 'project-image-studio', NULL, 2500, 1500),
  ('wallet-video-lab', 'project-video-lab', NULL, 5000, 2000),
  ('wallet-acme-company', NULL, 'company-acme', 125000, 25000);

INSERT INTO user_ledger_transactions (id, wallet_id, transaction_type, amount, credit_type, status, reason, idempotency_key, metadata) VALUES
  ('ledger-dev-paid', 'wallet-demo', 'grant', 10000, 'paid', 'posted', 'dev seed paid credits', 'dev-seed-paid-credit-grant', '{}'),
  ('ledger-dev-promo', 'wallet-demo', 'grant', 5000, 'promotional', 'posted', 'dev seed promotional credits', 'dev-seed-promo-credit-grant', '{}');

INSERT INTO user_credit_purchases (id, user_id, credits, amount_cents, currency, status, metadata, created_at) VALUES
  ('purchase-yong-2026-06-06', 'user-yong-zhao', 5000, 5000, 'USD', 'posted', '{"payment_method":"visa ending 4242"}', '2026-06-06 10:12:00'),
  ('purchase-yong-2026-05-22', 'user-yong-zhao', 10000, 10000, 'USD', 'posted', '{"payment_method":"visa ending 4242"}', '2026-05-22 09:30:00'),
  ('purchase-yong-2026-04-26', 'user-yong-zhao', 5000, 5000, 'USD', 'posted', '{"payment_method":"visa ending 4242"}', '2026-04-26 14:45:00'),
  ('purchase-yong-2026-03-29', 'user-yong-zhao', 20000, 20000, 'USD', 'posted', '{"payment_method":"bank transfer"}', '2026-03-29 16:20:00');

INSERT INTO sys_coupons (id, code, credit_amount, status, metadata) VALUES
  ('coupon-dev-welcome', 'DEV-WELCOME', 1000, 'active', '{"description":"Development welcome coupon"}');

INSERT INTO user_conversations (id, project_id, title, status) VALUES
  ('conversation-demo', 'project-demo', 'Seeded demo conversation', 'active'),
  ('conversation-image-product', 'project-image-studio', 'Product hero image concepts', 'active'),
  ('conversation-image-avatar', 'project-image-studio', 'Avatar style exploration', 'active'),
  ('conversation-video-launch', 'project-video-lab', 'Launch teaser storyboard', 'active'),
  ('conversation-video-social', 'project-video-lab', 'Short social clip variants', 'active'),
  ('conversation-acme-campaign', 'project-acme-brand', 'Corporate campaign concepts', 'active'),
  ('conversation-acme-training-video', 'project-acme-video', 'Training video drafts', 'active');

INSERT INTO user_conversation_branches (id, conversation_id, parent_branch_id, name) VALUES
  ('branch-demo-main', 'conversation-demo', NULL, 'Main'),
  ('branch-image-product-main', 'conversation-image-product', NULL, 'Main'),
  ('branch-image-product-dark', 'conversation-image-product', 'branch-image-product-main', 'Dark premium version'),
  ('branch-image-product-minimal', 'conversation-image-product', 'branch-image-product-main', 'Minimal white-background version'),
  ('branch-image-avatar-main', 'conversation-image-avatar', NULL, 'Main'),
  ('branch-video-launch-main', 'conversation-video-launch', NULL, 'Main'),
  ('branch-video-social-main', 'conversation-video-social', NULL, 'Main'),
  ('branch-acme-campaign-main', 'conversation-acme-campaign', NULL, 'Main'),
  ('branch-acme-training-main', 'conversation-acme-training-video', NULL, 'Main');

INSERT INTO user_messages (id, conversation_id, branch_id, role, content, model_profile_id, inference_request_id, customer_charge, provider_cost, metadata) VALUES
  ('message-demo-user', 'conversation-demo', 'branch-demo-main', 'user', 'What can I do in this model market?', NULL, NULL, 0, 0, '{}'),
  ('message-demo-assistant', 'conversation-demo', 'branch-demo-main', 'assistant', 'You can browse models, use the workbench, create API keys, and consume credits through mocked provider calls.', 'profile-mock-chat-default', NULL, 1, 0, '{"provider":"mock-provider"}'),
  ('message-image-product-user', 'conversation-image-product', 'branch-image-product-main', 'user', 'Create a clean product hero image for a compact AI hardware device on a white desk.', NULL, NULL, 0, 0, '{}'),
  ('message-image-product-assistant', 'conversation-image-product', 'branch-image-product-main', 'assistant', 'Generated a bright studio concept with soft shadows, brushed metal details, and a minimal background suitable for a landing page hero.', 'profile-openrouter-mai-image-25-default', NULL, 10, 0, '{"provider":"openrouter","artifact":"asset-image-product-hero"}'),
  ('message-image-product-dark-user', 'conversation-image-product', 'branch-image-product-dark', 'user', 'Fork this concept into a darker premium version with a graphite desk and sharper rim light.', NULL, NULL, 0, 0, '{}'),
  ('message-image-product-dark-assistant', 'conversation-image-product', 'branch-image-product-dark', 'assistant', 'Created a darker direction with a graphite surface, narrow rim lighting, and a more cinematic composition for premium positioning.', 'profile-openrouter-grok-imagine-image-quality-default', NULL, 10, 0, '{"provider":"openrouter","branch":"dark-premium"}'),
  ('message-image-product-minimal-user', 'conversation-image-product', 'branch-image-product-minimal', 'user', 'Fork this into a minimal white-background version with more empty space for headline text.', NULL, NULL, 0, 0, '{}'),
  ('message-image-product-minimal-assistant', 'conversation-image-product', 'branch-image-product-minimal', 'assistant', 'Created a minimal white-background branch with increased negative space, softer shadows, and a centered device angle for marketing copy.', 'profile-openrouter-mai-image-25-default', NULL, 10, 0, '{"provider":"openrouter","branch":"minimal-white"}'),
  ('message-image-avatar-user', 'conversation-image-avatar', 'branch-image-avatar-main', 'user', 'Explore three avatar styles for an engineering team profile page.', NULL, NULL, 0, 0, '{}'),
  ('message-image-avatar-assistant', 'conversation-image-avatar', 'branch-image-avatar-main', 'assistant', 'Created style directions: editorial monochrome portraits, warm illustrated badges, and crisp 3D profile icons.', 'profile-openrouter-grok-imagine-image-quality-default', NULL, 10, 0, '{"provider":"openrouter","artifact":"asset-image-avatar-board"}'),
  ('message-video-launch-user', 'conversation-video-launch', 'branch-video-launch-main', 'user', 'Draft a 10 second launch teaser showing model routing from prompt to final video.', NULL, NULL, 0, 0, '{}'),
  ('message-video-launch-assistant', 'conversation-video-launch', 'branch-video-launch-main', 'assistant', 'Prepared a storyboard with three shots: prompt entry, routing visualization, and generated clip preview with synchronized captions.', 'profile-openrouter-veo-31-fast-default', NULL, 100, 10, '{"provider":"openrouter","artifact":"asset-video-launch-storyboard"}'),
  ('message-video-social-user', 'conversation-video-social', 'branch-video-social-main', 'user', 'Make short social variants for image, audio, and video generation features.', NULL, NULL, 0, 0, '{}'),
  ('message-video-social-assistant', 'conversation-video-social', 'branch-video-social-main', 'assistant', 'Created three vertical clip concepts with fast cuts, concise feature callouts, and model cards animated into the frame.', 'profile-openrouter-grok-imagine-video-default', NULL, 70, 7, '{"provider":"openrouter","artifact":"asset-video-social-variants"}'),
  ('message-acme-campaign-user', 'conversation-acme-campaign', 'branch-acme-campaign-main', 'user', 'Create a polished image concept for Acme corporate brand refresh.', NULL, NULL, 0, 0, '{"actor_user_id":"user-corp-designer"}'),
  ('message-acme-campaign-assistant', 'conversation-acme-campaign', 'branch-acme-campaign-main', 'assistant', 'Generated a brand refresh concept with a clean product scene, bright typography space, and a professional enterprise tone.', 'profile-openrouter-mai-image-25-default', NULL, 10, 0, '{"provider":"openrouter","actor_user_id":"user-corp-designer"}'),
  ('message-acme-training-user', 'conversation-acme-training-video', 'branch-acme-training-main', 'user', 'Draft a short onboarding video for sales enablement training.', NULL, NULL, 0, 0, '{"actor_user_id":"user-corp-producer"}'),
  ('message-acme-training-assistant', 'conversation-acme-training-video', 'branch-acme-training-main', 'assistant', 'Prepared a 12 second training video structure with intro title, workflow demonstration, and a closing action slide.', 'profile-openrouter-grok-imagine-video-default', NULL, 70, 7, '{"provider":"openrouter","actor_user_id":"user-corp-producer"}');

INSERT INTO user_workspace_assets (
  id, project_id, conversation_id, asset_type, storage_path, storage_provider, bucket_name, object_key, download_url,
  mime_type, size_bytes, inference_request_id, customer_charge, provider_cost, metadata
) VALUES
  ('asset-demo-text', 'project-demo', 'conversation-demo', 'upload', 's3://model-market-dev-assets/demo-project/uploads/demo.txt', 's3', 'model-market-dev-assets', 'demo-project/uploads/demo.txt', 'https://model-market-dev-assets.s3.amazonaws.com/demo-project/uploads/demo.txt', 'text/plain', 128, NULL, 0, 0, '{"mock":true}'),
  ('asset-image-product-hero', 'project-image-studio', 'conversation-image-product', 'image', 's3://model-market-dev-assets/project-image-studio/generated/product-hero.png', 's3', 'model-market-dev-assets', 'project-image-studio/generated/product-hero.png', 'https://model-market-dev-assets.s3.amazonaws.com/project-image-studio/generated/product-hero.png', 'image/png', 2048000, NULL, 10, 0, '{"prompt":"compact AI hardware device on a white desk"}'),
  ('asset-image-avatar-board', 'project-image-studio', 'conversation-image-avatar', 'image', 's3://model-market-dev-assets/project-image-studio/generated/avatar-style-board.png', 's3', 'model-market-dev-assets', 'project-image-studio/generated/avatar-style-board.png', 'https://model-market-dev-assets.s3.amazonaws.com/project-image-studio/generated/avatar-style-board.png', 'image/png', 1536000, NULL, 10, 0, '{"prompt":"engineering team avatar style exploration"}'),
  ('asset-video-launch-storyboard', 'project-video-lab', 'conversation-video-launch', 'storyboard', 's3://model-market-dev-assets/project-video-lab/generated/launch-teaser-storyboard.json', 's3', 'model-market-dev-assets', 'project-video-lab/generated/launch-teaser-storyboard.json', 'https://model-market-dev-assets.s3.amazonaws.com/project-video-lab/generated/launch-teaser-storyboard.json', 'application/json', 8192, NULL, 100, 10, '{"duration_seconds":10,"shots":3}'),
  ('asset-video-social-variants', 'project-video-lab', 'conversation-video-social', 'video', 's3://model-market-dev-assets/project-video-lab/generated/social-variants.mp4', 's3', 'model-market-dev-assets', 'project-video-lab/generated/social-variants.mp4', 'https://model-market-dev-assets.s3.amazonaws.com/project-video-lab/generated/social-variants.mp4', 'video/mp4', 7340032, NULL, 70, 7, '{"variants":3,"aspect_ratio":"9:16"}'),
  ('asset-acme-campaign-image', 'project-acme-brand', 'conversation-acme-campaign', 'image', 's3://model-market-dev-assets/project-acme-brand/generated/campaign-concept.png', 's3', 'model-market-dev-assets', 'project-acme-brand/generated/campaign-concept.png', 'https://model-market-dev-assets.s3.amazonaws.com/project-acme-brand/generated/campaign-concept.png', 'image/png', 1887436, NULL, 10, 0, '{"actor_user_id":"user-corp-designer","prompt":"corporate brand refresh"}'),
  ('asset-acme-training-video', 'project-acme-video', 'conversation-acme-training-video', 'video', 's3://model-market-dev-assets/project-acme-video/generated/training-draft.mp4', 's3', 'model-market-dev-assets', 'project-acme-video/generated/training-draft.mp4', 'https://model-market-dev-assets.s3.amazonaws.com/project-acme-video/generated/training-draft.mp4', 'video/mp4', 8388608, NULL, 70, 7, '{"actor_user_id":"user-corp-producer","duration_seconds":12}');

INSERT INTO user_message_attachments (id, message_id, asset_id) VALUES
  ('attachment-demo-text', 'message-demo-user', 'asset-demo-text');

INSERT INTO user_file_extractions (id, asset_id, extracted_text, metadata) VALUES
  ('extraction-demo-text', 'asset-demo-text', 'This is mocked extracted text for local development.', '{}');

INSERT INTO user_embedding_records (id, project_id, source_type, source_id, embedding_model, vector_ref, metadata) VALUES
  ('embedding-demo-text', 'project-demo', 'asset', 'asset-demo-text', 'mock-embedding', 'mock://vectors/embedding-demo-text', '{}');

INSERT INTO user_inference_requests (id, project_id, actor_user_id, model_slug, model_profile_id, provider_slug, status, input_units, output_units, customer_charge, provider_cost, margin, metadata) VALUES
  ('inference-demo', 'project-demo', 'user-admin', 'mock-chat', 'profile-mock-chat-default', 'mock-provider', 'succeeded', 12, 24, 1, 0, 1, '{"mock":true}');

INSERT INTO user_provider_attempts (id, inference_request_id, provider_id, status, latency_ms, provider_request_id, error_class, metadata) VALUES
  ('attempt-demo', 'inference-demo', 'provider-mock', 'succeeded', 42, 'mock-request-1', NULL, '{}');

INSERT INTO user_usage_events (id, project_id, actor_user_id, inference_request_id, model_slug, provider_slug, event_type, input_tokens, output_tokens, customer_charge, provider_cost, metadata, created_at) VALUES
  ('usage-demo', 'project-demo', 'user-admin', 'inference-demo', 'mock-chat', 'mock-provider', 'chat_completion', 12, 24, 1, 0, '{"mock":true}', '2026-06-08 08:00:00'),
  ('usage-image-product-main', 'project-image-studio', 'user-developer', NULL, 'microsoft/mai-image-2.5', 'openrouter', 'image_generation', 520, 980, 10, 0, '{"conversation_id":"conversation-image-product","branch_id":"branch-image-product-main"}', '2026-06-05 12:00:00'),
  ('usage-image-product-dark', 'project-image-studio', 'user-developer', NULL, 'x-ai/grok-imagine-image-quality', 'openrouter', 'image_generation', 610, 1120, 10, 0, '{"conversation_id":"conversation-image-product","branch_id":"branch-image-product-dark"}', '2026-06-05 12:15:00'),
  ('usage-image-avatar', 'project-image-studio', 'user-developer', NULL, 'x-ai/grok-imagine-image-quality', 'openrouter', 'image_generation', 430, 760, 10, 0, '{"conversation_id":"conversation-image-avatar"}', '2026-05-30 10:00:00'),
  ('usage-video-launch', 'project-video-lab', 'user-developer', NULL, 'google/veo-3.1-fast', 'openrouter', 'video_generation', 900, 1820, 100, 10, '{"conversation_id":"conversation-video-launch"}', '2026-05-28 15:00:00'),
  ('usage-video-social', 'project-video-lab', 'user-developer', NULL, 'x-ai/grok-imagine-video', 'openrouter', 'video_generation', 840, 1640, 70, 7, '{"conversation_id":"conversation-video-social"}', '2026-05-27 15:00:00'),
  ('usage-acme-campaign-image', 'project-acme-brand', 'user-corp-designer', NULL, 'microsoft/mai-image-2.5', 'openrouter', 'image_generation', 510, 920, 10, 0, '{"company_id":"company-acme","conversation_id":"conversation-acme-campaign"}', '2026-06-07 09:30:00'),
  ('usage-acme-training-video', 'project-acme-video', 'user-corp-producer', NULL, 'x-ai/grok-imagine-video', 'openrouter', 'video_generation', 870, 1710, 70, 7, '{"company_id":"company-acme","conversation_id":"conversation-acme-training-video"}', '2026-06-07 11:20:00'),
  ('usage-yong-001', 'project-demo', 'user-yong-zhao', NULL, 'x-ai/grok-imagine-video', 'openrouter', 'video_generation', 860, 1690, 70, 7, '{"demo":"credit_usage"}', '2026-06-08 09:10:00'),
  ('usage-yong-002', 'project-demo', 'user-yong-zhao', NULL, 'microsoft/mai-image-2.5', 'openrouter', 'image_generation', 540, 980, 10, 0, '{"demo":"credit_usage"}', '2026-06-08 10:05:00'),
  ('usage-yong-003', 'project-demo', 'user-yong-zhao', NULL, 'openai/gpt-audio', 'openrouter', 'audio_generation', 460, 820, 12, 2, '{"demo":"credit_usage"}', '2026-06-07 14:30:00'),
  ('usage-yong-004', 'project-demo', 'user-yong-zhao', NULL, 'x-ai/grok-imagine-video', 'openrouter', 'video_generation', 920, 1840, 70, 7, '{"demo":"credit_usage"}', '2026-06-06 16:15:00'),
  ('usage-yong-005', 'project-demo', 'user-yong-zhao', NULL, 'microsoft/mai-image-2.5', 'openrouter', 'image_generation', 590, 1040, 10, 0, '{"demo":"credit_usage"}', '2026-06-05 13:45:00'),
  ('usage-yong-006', 'project-demo', 'user-yong-zhao', NULL, 'openai/gpt-audio', 'openrouter', 'audio_generation', 500, 910, 12, 2, '{"demo":"credit_usage"}', '2026-06-04 11:05:00'),
  ('usage-yong-007', 'project-demo', 'user-yong-zhao', NULL, 'x-ai/grok-imagine-video', 'openrouter', 'video_generation', 880, 1740, 70, 7, '{"demo":"credit_usage"}', '2026-06-03 18:25:00'),
  ('usage-yong-008', 'project-demo', 'user-yong-zhao', NULL, 'microsoft/mai-image-2.5', 'openrouter', 'image_generation', 620, 1160, 10, 0, '{"demo":"credit_usage"}', '2026-06-02 09:40:00'),
  ('usage-yong-009', 'project-demo', 'user-yong-zhao', NULL, 'x-ai/grok-imagine-video', 'openrouter', 'video_generation', 940, 1860, 70, 7, '{"demo":"credit_usage"}', '2026-06-01 17:10:00'),
  ('usage-yong-010', 'project-demo', 'user-yong-zhao', NULL, 'openai/gpt-audio', 'openrouter', 'audio_generation', 420, 760, 12, 2, '{"demo":"credit_usage"}', '2026-05-29 12:30:00'),
  ('usage-yong-011', 'project-demo', 'user-yong-zhao', NULL, 'microsoft/mai-image-2.5', 'openrouter', 'image_generation', 560, 1010, 10, 0, '{"demo":"credit_usage"}', '2026-05-26 10:20:00'),
  ('usage-yong-012', 'project-demo', 'user-yong-zhao', NULL, 'x-ai/grok-imagine-video', 'openrouter', 'video_generation', 790, 1510, 70, 7, '{"demo":"credit_usage"}', '2026-05-23 16:05:00'),
  ('usage-yong-013', 'project-demo', 'user-yong-zhao', NULL, 'openai/gpt-audio', 'openrouter', 'audio_generation', 470, 850, 12, 2, '{"demo":"credit_usage"}', '2026-05-20 11:55:00'),
  ('usage-yong-014', 'project-demo', 'user-yong-zhao', NULL, 'microsoft/mai-image-2.5', 'openrouter', 'image_generation', 640, 1190, 10, 0, '{"demo":"credit_usage"}', '2026-05-17 15:25:00'),
  ('usage-yong-015', 'project-demo', 'user-yong-zhao', NULL, 'x-ai/grok-imagine-video', 'openrouter', 'video_generation', 1010, 1960, 70, 7, '{"demo":"credit_usage"}', '2026-05-14 18:10:00'),
  ('usage-yong-016', 'project-demo', 'user-yong-zhao', NULL, 'openai/gpt-audio', 'openrouter', 'audio_generation', 390, 700, 12, 2, '{"demo":"credit_usage"}', '2026-05-08 13:00:00'),
  ('usage-yong-017', 'project-demo', 'user-yong-zhao', NULL, 'microsoft/mai-image-2.5', 'openrouter', 'image_generation', 610, 1110, 10, 0, '{"demo":"credit_usage"}', '2026-04-28 09:35:00'),
  ('usage-yong-018', 'project-demo', 'user-yong-zhao', NULL, 'x-ai/grok-imagine-video', 'openrouter', 'video_generation', 930, 1810, 70, 7, '{"demo":"credit_usage"}', '2026-04-18 17:45:00'),
  ('usage-yong-019', 'project-demo', 'user-yong-zhao', NULL, 'openai/gpt-audio', 'openrouter', 'audio_generation', 440, 790, 12, 2, '{"demo":"credit_usage"}', '2026-03-31 12:15:00'),
  ('usage-yong-020', 'project-demo', 'user-yong-zhao', NULL, 'microsoft/mai-image-2.5', 'openrouter', 'image_generation', 570, 1060, 10, 0, '{"demo":"credit_usage"}', '2026-03-20 10:50:00');

INSERT INTO user_async_jobs (id, project_id, job_type, status, model_slug, provider_slug, metadata) VALUES
  ('job-demo-image', 'project-demo', 'image_generation', 'completed', 'mock-creative', 'mock-provider', '{"mock":true}');

INSERT INTO user_routing_policies (id, project_id, name, policy_data, status) VALUES
  ('routing-demo-default', 'project-demo', 'Default dev routing', '{"mode":"fixed","provider":"mock-provider"}', 'active');

INSERT INTO user_budget_policies (id, project_id, name, limit_credits, period, status) VALUES
  ('budget-demo-monthly', 'project-demo', 'Monthly dev budget', 100000, 'month', 'active');

INSERT INTO user_webhook_endpoints (id, project_id, url, secret_ref, status) VALUES
  ('webhook-demo', 'project-demo', 'https://example.com/webhooks/model-market', 'WEBHOOK_DEMO_SECRET', 'inactive');

INSERT INTO sys_audit_logs (id, actor_user_id, organization_id, action, target_type, target_id, metadata) VALUES
  ('audit-demo-seed', 'user-admin', 'org-demo', 'seed.loaded', 'project', 'project-demo', '{"source":"populate_test_data.sql"}');

INSERT INTO sys_provider_settlements (id, provider_id, period_start, period_end, amount_cents, currency, status, metadata) VALUES
  ('settlement-mock-june', 'provider-mock', '2026-06-01 00:00:00', '2026-06-30 23:59:59', 0, 'USD', 'draft', '{"mock":true}');
