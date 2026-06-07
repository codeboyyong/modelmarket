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
  ('oauth-github-admin', 'user-admin', 'github', 'github-admin-dev', 'admin@example.com', 'Admin User'),
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
  ('provider-openrouter', 'openrouter', 'OpenRouter', 'active', 'https://openrouter.ai/api/v1', 'OPENROUTER_API_KEY', '{"source":"https://openrouter.ai/models","supports_multimodal":true}'),
  ('provider-openai-placeholder', 'openai-placeholder', 'OpenAI Placeholder', 'inactive', 'https://api.openai.com', 'OPENAI_API_KEY', '{"enabled_by_default":false}');

INSERT INTO provider_endpoints (id, provider_id, name, endpoint_url, region, status, metadata) VALUES
  ('endpoint-mock-default', 'provider-mock', 'Mock Default Endpoint', 'mock://provider/default', 'local', 'active', '{}'),
  ('endpoint-openrouter-default', 'provider-openrouter', 'OpenRouter Default Endpoint', 'https://openrouter.ai/api/v1', 'global', 'active', '{"models_url":"https://openrouter.ai/api/v1/models","video_models_url":"https://openrouter.ai/api/v1/videos/models"}'),
  ('endpoint-openai-default', 'provider-openai-placeholder', 'OpenAI Default Endpoint', 'https://api.openai.com', 'global', 'inactive', '{}');

INSERT INTO capabilities (id, slug, name, description) VALUES
  ('cap-chat', 'chat', 'Chat', 'Chat completion support'),
  ('cap-streaming', 'streaming', 'Streaming', 'Streaming response support'),
  ('cap-image-generation', 'image-generation', 'Image Generation', 'Image generation support'),
  ('cap-async-jobs', 'async-jobs', 'Async Jobs', 'Long-running async job support');

INSERT INTO models (id, provider_id, slug, name, modality, status, context_window, capabilities, metadata) VALUES
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

INSERT INTO model_versions (id, model_id, version, status, metadata) VALUES
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

INSERT INTO model_profiles (id, model_id, slug, name, status, system_prompt, default_parameters, safety_settings, config_version) VALUES
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

INSERT INTO model_configurations (id, model_profile_id, version, config_data, status) VALUES
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
