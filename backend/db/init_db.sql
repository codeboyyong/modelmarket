create table if not exists schema_migrations (
	version varchar(255) primary key,
	applied_at timestamp not null default current_timestamp
);

create table if not exists users (
	id varchar(64) primary key,
	email varchar(255) not null unique,
	name varchar(255) not null,
	avatar_url varchar(1024),
	status varchar(32) not null default 'active',
	created_at timestamp not null default current_timestamp
);

create table if not exists oauth_accounts (
	id varchar(64) primary key,
	user_id varchar(64) not null references users(id),
	provider varchar(64) not null,
	provider_account_id varchar(255) not null,
	email varchar(255),
	display_name varchar(255),
	avatar_url varchar(1024),
	last_login_at timestamp,
	created_at timestamp not null default current_timestamp,
	unique(provider, provider_account_id)
);

create table if not exists sessions (
	id varchar(64) primary key,
	user_id varchar(64) not null references users(id),
	token_hash varchar(255) not null unique,
	expires_at timestamp not null,
	created_at timestamp not null default current_timestamp
);

create table if not exists organizations (
	id varchar(64) primary key,
	name varchar(255) not null,
	slug varchar(255) not null unique,
	created_at timestamp not null default current_timestamp
);

create table if not exists memberships (
	id varchar(64) primary key,
	user_id varchar(64) not null references users(id),
	organization_id varchar(64) not null references organizations(id),
	role varchar(64) not null,
	created_at timestamp not null default current_timestamp,
	unique(user_id, organization_id)
);

create table if not exists projects (
	id varchar(64) primary key,
	organization_id varchar(64) not null references organizations(id),
	name varchar(255) not null,
	slug varchar(255) not null,
	environment varchar(32) not null default 'dev',
	retention_policy text not null default '{}',
	created_at timestamp not null default current_timestamp,
	unique(organization_id, slug)
);

create table if not exists api_keys (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id),
	name varchar(255) not null,
	prefix varchar(64) not null,
	key_hash varchar(255) not null unique,
	scopes text not null default '',
	status varchar(32) not null default 'active',
	created_at timestamp not null default current_timestamp,
	revoked_at timestamp
);

create table if not exists providers (
	id varchar(64) primary key,
	slug varchar(255) not null unique,
	name varchar(255) not null,
	status varchar(32) not null default 'active',
	endpoint_url varchar(1024),
	credential_ref varchar(255),
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists models (
	id varchar(64) primary key,
	provider_id varchar(64) not null references providers(id),
	slug varchar(255) not null unique,
	name varchar(255) not null,
	modality varchar(64) not null,
	status varchar(32) not null default 'public',
	context_window integer not null default 0,
	capabilities text not null default '{}',
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists model_versions (
	id varchar(64) primary key,
	model_id varchar(64) not null references models(id),
	version varchar(255) not null,
	status varchar(32) not null default 'active',
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp,
	unique(model_id, version)
);

create table if not exists model_profiles (
	id varchar(64) primary key,
	model_id varchar(64) not null references models(id),
	slug varchar(255) not null unique,
	name varchar(255) not null,
	status varchar(32) not null default 'public',
	system_prompt text not null default '',
	default_parameters text not null default '{}',
	safety_settings text not null default '{}',
	config_version integer not null default 1,
	created_at timestamp not null default current_timestamp
);

create table if not exists price_rules (
	id varchar(64) primary key,
	model_id varchar(64) references models(id),
	model_profile_id varchar(64) references model_profiles(id),
	unit_type varchar(64) not null,
	customer_price_credits integer not null,
	provider_cost_credits integer not null default 0,
	currency varchar(16) not null default 'CREDIT',
	effective_at timestamp not null default current_timestamp,
	metadata text not null default '{}'
);

create table if not exists wallets (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id) unique,
	paid_credits bigint not null default 0,
	promotional_credits bigint not null default 0,
	updated_at timestamp not null default current_timestamp
);

create table if not exists ledger_transactions (
	id varchar(64) primary key,
	wallet_id varchar(64) not null references wallets(id),
	transaction_type varchar(64) not null,
	amount bigint not null,
	credit_type varchar(64) not null,
	status varchar(32) not null,
	reason varchar(255) not null,
	idempotency_key varchar(255) not null unique,
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists conversations (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id),
	title varchar(255) not null,
	status varchar(32) not null default 'active',
	created_at timestamp not null default current_timestamp,
	updated_at timestamp not null default current_timestamp
);

create table if not exists messages (
	id varchar(64) primary key,
	conversation_id varchar(64) not null references conversations(id),
	role varchar(64) not null,
	content text not null,
	model_profile_id varchar(64) references model_profiles(id),
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists conversation_branches (
	id varchar(64) primary key,
	conversation_id varchar(64) not null references conversations(id),
	parent_branch_id varchar(64),
	name varchar(255) not null,
	created_at timestamp not null default current_timestamp
);

create table if not exists message_attachments (
	id varchar(64) primary key,
	message_id varchar(64) not null references messages(id),
	asset_id varchar(64),
	file_id varchar(64),
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists context_summaries (
	id varchar(64) primary key,
	conversation_id varchar(64) not null references conversations(id),
	model_profile_id varchar(64) references model_profiles(id),
	summary text not null,
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists uploaded_files (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id),
	original_name varchar(255) not null,
	mime_type varchar(255) not null,
	size_bytes bigint not null default 0,
	storage_path varchar(1024) not null,
	file_hash varchar(255),
	status varchar(32) not null default 'ready',
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists file_extractions (
	id varchar(64) primary key,
	file_id varchar(64) not null references uploaded_files(id),
	extraction_type varchar(64) not null,
	content text not null,
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists embedding_records (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id),
	source_type varchar(64) not null,
	source_id varchar(64) not null,
	embedding_model varchar(255) not null,
	vector_ref varchar(1024),
	content text,
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists workspace_assets (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id),
	conversation_id varchar(64) references conversations(id),
	asset_type varchar(64) not null,
	mime_type varchar(255),
	storage_path varchar(1024) not null,
	prompt text,
	model_profile_id varchar(64) references model_profiles(id),
	status varchar(32) not null default 'ready',
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists inference_requests (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id),
	model_slug varchar(255) not null,
	provider_slug varchar(255) not null,
	status varchar(32) not null,
	input_units bigint not null default 0,
	output_units bigint not null default 0,
	customer_charge bigint not null default 0,
	provider_cost bigint not null default 0,
	margin bigint not null default 0,
	created_at timestamp not null default current_timestamp
);

create table if not exists provider_attempts (
	id varchar(64) primary key,
	inference_request_id varchar(64) not null references inference_requests(id),
	provider_id varchar(64) references providers(id),
	provider_request_id varchar(255),
	status varchar(32) not null,
	latency_ms integer not null default 0,
	provider_cost bigint not null default 0,
	error_class varchar(255),
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists async_jobs (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id),
	conversation_id varchar(64) references conversations(id),
	model_profile_id varchar(64) references model_profiles(id),
	job_type varchar(64) not null,
	status varchar(32) not null,
	input_metadata text not null default '{}',
	result_asset_id varchar(64),
	created_at timestamp not null default current_timestamp,
	updated_at timestamp not null default current_timestamp
);

create table if not exists job_events (
	id varchar(64) primary key,
	async_job_id varchar(64) not null references async_jobs(id),
	event_type varchar(64) not null,
	message text,
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists usage_events (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id),
	inference_request_id varchar(64) references inference_requests(id),
	model_slug varchar(255) not null,
	provider_slug varchar(255) not null,
	event_type varchar(64) not null,
	customer_charge bigint not null default 0,
	provider_cost bigint not null default 0,
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists payments (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id),
	provider varchar(64) not null,
	provider_payment_id varchar(255),
	status varchar(32) not null,
	amount_cents bigint not null,
	currency varchar(16) not null,
	credits_granted bigint not null default 0,
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists invoices (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id),
	status varchar(32) not null,
	currency varchar(16) not null,
	total_cents bigint not null default 0,
	invoice_number varchar(255),
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists invoice_items (
	id varchar(64) primary key,
	invoice_id varchar(64) not null references invoices(id),
	description varchar(1024) not null,
	quantity bigint not null default 1,
	unit_amount_cents bigint not null default 0,
	total_cents bigint not null default 0,
	metadata text not null default '{}'
);

create table if not exists coupons (
	id varchar(64) primary key,
	code varchar(255) not null unique,
	status varchar(32) not null,
	credit_amount bigint not null default 0,
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists routing_policies (
	id varchar(64) primary key,
	project_id varchar(64) references projects(id),
	name varchar(255) not null,
	status varchar(32) not null default 'active',
	policy text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists budget_policies (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id),
	name varchar(255) not null,
	monthly_credit_limit bigint,
	status varchar(32) not null default 'active',
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists webhook_endpoints (
	id varchar(64) primary key,
	project_id varchar(64) not null references projects(id),
	url varchar(1024) not null,
	secret_ref varchar(255),
	status varchar(32) not null default 'active',
	created_at timestamp not null default current_timestamp
);

create table if not exists webhook_deliveries (
	id varchar(64) primary key,
	webhook_endpoint_id varchar(64) not null references webhook_endpoints(id),
	event_type varchar(255) not null,
	status varchar(32) not null,
	attempt_count integer not null default 0,
	last_error text,
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists provider_settlements (
	id varchar(64) primary key,
	provider_id varchar(64) not null references providers(id),
	status varchar(32) not null,
	amount_cents bigint not null default 0,
	currency varchar(16) not null default 'USD',
	period_start timestamp,
	period_end timestamp,
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists notifications (
	id varchar(64) primary key,
	project_id varchar(64) references projects(id),
	user_id varchar(64) references users(id),
	notification_type varchar(255) not null,
	channel varchar(64) not null,
	status varchar(32) not null,
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists provider_health_events (
	id varchar(64) primary key,
	provider_id varchar(64) not null references providers(id),
	status varchar(32) not null,
	latency_ms integer,
	error_rate numeric(10,4),
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create table if not exists audit_logs (
	id varchar(64) primary key,
	actor_user_id varchar(64) references users(id),
	organization_id varchar(64) references organizations(id),
	action varchar(255) not null,
	target_type varchar(255) not null,
	target_id varchar(64),
	metadata text not null default '{}',
	created_at timestamp not null default current_timestamp
);

create index if not exists idx_api_keys_project on api_keys(project_id);
create index if not exists idx_models_provider on models(provider_id);
create index if not exists idx_messages_conversation on messages(conversation_id, created_at);
create index if not exists idx_usage_project_created on usage_events(project_id, created_at);
create index if not exists idx_inference_project_created on inference_requests(project_id, created_at);
create index if not exists idx_uploaded_files_project on uploaded_files(project_id, created_at);
create index if not exists idx_workspace_assets_project on workspace_assets(project_id, created_at);
create index if not exists idx_async_jobs_project on async_jobs(project_id, created_at);
create index if not exists idx_provider_attempts_request on provider_attempts(inference_request_id);
create index if not exists idx_audit_logs_created on audit_logs(created_at);
