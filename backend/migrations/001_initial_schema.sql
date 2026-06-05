create extension if not exists pgcrypto;

create table if not exists users (
	id text primary key default gen_random_uuid()::text,
	email text not null unique,
	name text not null,
 avatar_url text,
	status text not null default 'active',
	created_at timestamptz not null default now()
);

create table if not exists oauth_accounts (
	id text primary key default gen_random_uuid()::text,
	user_id text not null references users(id),
	provider text not null,
	provider_account_id text not null,
	email text,
	display_name text,
	avatar_url text,
	last_login_at timestamptz,
	created_at timestamptz not null default now(),
	unique(provider, provider_account_id)
);

create table if not exists sessions (
	id text primary key default gen_random_uuid()::text,
	user_id text not null references users(id),
	token_hash text not null unique,
	expires_at timestamptz not null,
	created_at timestamptz not null default now()
);

create table if not exists organizations (
	id text primary key default gen_random_uuid()::text,
	name text not null,
	slug text not null unique,
	created_at timestamptz not null default now()
);

create table if not exists memberships (
	id text primary key default gen_random_uuid()::text,
	user_id text not null references users(id),
	organization_id text not null references organizations(id),
	role text not null,
	created_at timestamptz not null default now(),
	unique(user_id, organization_id)
);

create table if not exists projects (
	id text primary key default gen_random_uuid()::text,
	organization_id text not null references organizations(id),
	name text not null,
	slug text not null,
	environment text not null default 'dev',
	retention_policy jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	unique(organization_id, slug)
);

create table if not exists api_keys (
	id text primary key default gen_random_uuid()::text,
	project_id text not null references projects(id),
	name text not null,
	prefix text not null,
	key_hash text not null unique,
	scopes text[] not null default '{}',
	status text not null default 'active',
	created_at timestamptz not null default now(),
	revoked_at timestamptz
);

create table if not exists providers (
	id text primary key default gen_random_uuid()::text,
	slug text not null unique,
	name text not null,
	status text not null default 'active',
	endpoint_url text,
	credential_ref text,
	metadata jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now()
);

create table if not exists models (
	id text primary key default gen_random_uuid()::text,
	provider_id text not null references providers(id),
	slug text not null unique,
	name text not null,
	modality text not null,
	status text not null default 'public',
	context_window integer not null default 0,
	capabilities jsonb not null default '{}'::jsonb,
	metadata jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now()
);

create table if not exists model_versions (
	id text primary key default gen_random_uuid()::text,
	model_id text not null references models(id),
	version text not null,
	status text not null default 'active',
	metadata jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now(),
	unique(model_id, version)
);

create table if not exists model_profiles (
	id text primary key default gen_random_uuid()::text,
	model_id text not null references models(id),
	slug text not null unique,
	name text not null,
	status text not null default 'public',
	system_prompt text not null default '',
	default_parameters jsonb not null default '{}'::jsonb,
	safety_settings jsonb not null default '{}'::jsonb,
	config_version integer not null default 1,
	created_at timestamptz not null default now()
);

create table if not exists price_rules (
	id text primary key default gen_random_uuid()::text,
	model_id text references models(id),
	model_profile_id text references model_profiles(id),
	unit_type text not null,
	customer_price_credits integer not null,
	provider_cost_credits integer not null default 0,
	currency text not null default 'CREDIT',
	effective_at timestamptz not null default now(),
	metadata jsonb not null default '{}'::jsonb
);

create table if not exists wallets (
	id text primary key default gen_random_uuid()::text,
	project_id text not null references projects(id) unique,
	paid_credits bigint not null default 0,
	promotional_credits bigint not null default 0,
	updated_at timestamptz not null default now()
);

create table if not exists ledger_transactions (
	id text primary key default gen_random_uuid()::text,
	wallet_id text not null references wallets(id),
	transaction_type text not null,
	amount bigint not null,
	credit_type text not null,
	status text not null,
	reason text not null,
	idempotency_key text not null unique,
	metadata jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now()
);

create table if not exists conversations (
	id text primary key default gen_random_uuid()::text,
	project_id text not null references projects(id),
	title text not null,
	status text not null default 'active',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now()
);

create table if not exists messages (
	id text primary key default gen_random_uuid()::text,
	conversation_id text not null references conversations(id),
	role text not null,
	content text not null,
	model_profile_id text references model_profiles(id),
	metadata jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now()
);

create table if not exists inference_requests (
	id text primary key default gen_random_uuid()::text,
	project_id text not null references projects(id),
	model_slug text not null,
	provider_slug text not null,
	status text not null,
	input_units bigint not null default 0,
	output_units bigint not null default 0,
	customer_charge bigint not null default 0,
	provider_cost bigint not null default 0,
	margin bigint not null default 0,
	created_at timestamptz not null default now()
);

create table if not exists usage_events (
	id text primary key default gen_random_uuid()::text,
	project_id text not null references projects(id),
	inference_request_id text references inference_requests(id),
	model_slug text not null,
	provider_slug text not null,
	event_type text not null,
	customer_charge bigint not null default 0,
	provider_cost bigint not null default 0,
	metadata jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now()
);

create table if not exists audit_logs (
	id text primary key default gen_random_uuid()::text,
	actor_user_id text references users(id),
	organization_id text references organizations(id),
	action text not null,
	target_type text not null,
	target_id text,
	metadata jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now()
);

create index if not exists idx_api_keys_project on api_keys(project_id);
create index if not exists idx_models_provider on models(provider_id);
create index if not exists idx_messages_conversation on messages(conversation_id, created_at);
create index if not exists idx_usage_project_created on usage_events(project_id, created_at desc);
create index if not exists idx_inference_project_created on inference_requests(project_id, created_at desc);
