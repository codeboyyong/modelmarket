# Implementation Plan

## 1. Goal

Build the platform in four phases, starting with a runnable core system and adding production, operational, and enterprise capabilities after the core product works end to end.

The MVP should prove the primary business loop:

1. User signs in.
2. User receives or buys credits.
3. User selects a model.
4. User chats or calls the API.
5. Backend routes to a mocked or real provider.
6. Usage is metered.
7. Credits are charged correctly.
8. Conversation/history/assets are persisted.
9. Admin can manage model/provider/pricing metadata.

## 2. Phase 1: Foundation and Local Dev

### Objective

Create the runnable skeleton: frontend container, Go backend container, PostgreSQL, Redis, local storage, migrations, seed/mock data, auth baseline, and health checks.

### Scope

- Repository structure:
  - `frontend/`
  - `backend/`
  - `deploy/`
  - `docs/`
  - `scripts/`
  - `mock-data/`
- Docker:
  - frontend Dockerfile
  - backend Dockerfile
  - `docker-compose.yml`
  - PostgreSQL service
  - Redis service
  - local object storage or mounted asset volume
- Backend foundation:
  - Go project setup
  - config loading
  - logging
  - health/readiness endpoints
  - database connection
  - Redis connection
  - migration runner
  - basic service/module layout
- Frontend foundation:
  - Node.js + TypeScript app setup
  - routing/layout shell
  - API client setup
  - auth pages shell
  - dashboard shell
  - workbench shell
  - admin shell
- Database foundation:
  - users
  - oauth_accounts
  - sessions
  - organizations
  - memberships
  - projects
  - api_keys
  - providers
  - models
  - model_versions
  - model_profiles
  - price_rules
  - wallets
  - ledger_transactions
  - conversations
  - messages
  - usage_events
  - audit_logs
- Dev mode:
  - repeatable seed/reset scripts
  - mock users
  - mock organizations/projects
  - mock models/providers
  - mock prices
  - mock wallets/credits
  - mock conversations/messages
  - mock usage events
- Auth baseline:
  - email login or dev login
  - Google OAuth placeholder/config
  - Facebook OAuth placeholder/config
  - session handling
  - basic RBAC primitives
- API key baseline:
  - create API key
  - hash API key
  - list/revoke API key
  - authenticate API request

### Deliverables

- `docker-compose up` starts the system.
- Frontend opens locally.
- Backend health endpoint works.
- Database migrations run.
- Mock seed data loads.
- Developer can log in with seeded user.
- Developer can see dashboard shell, catalog shell, workbench shell, and admin shell.
- API key can be created and authenticated.

### Exit Criteria

- Fresh clone can run locally with one documented command.
- No real provider or payment credentials required.
- All core tables have migrations.
- Seed/reset is repeatable.
- Basic CI runs backend tests, frontend typecheck/build, and migration checks.

## 3. Phase 2: Core Product Loop

### Objective

Implement the core user experience and money loop: catalog, model profiles, workbench chat, provider adapter, usage metering, credit wallet, and admin metadata manager.

### Scope

- Model catalog:
  - model list
  - model detail
  - provider metadata
  - capability display
  - pricing display
  - search/filter basics
- Metadata manager UI:
  - manage providers
  - manage provider endpoints
  - manage models/model versions
  - manage capabilities
  - manage model profiles
  - manage default system prompts
  - manage default parameters
  - manage price rules
  - publish/disable metadata
  - audit metadata changes
- Model profile/config resolution:
  - platform defaults
  - provider model defaults
  - admin model profile
  - project preset
  - request override
  - validation against model capability
  - store resolved config version on request
- Workbench chat:
  - select model/profile
  - send chat message
  - stream or display response
  - save conversation
  - save messages
  - switch model/profile
  - continue context across model switch
  - basic context truncation
  - show model/provider/cost/usage
- Mock provider adapter:
  - chat completion
  - streaming response
  - provider error
  - timeout
  - rate limit
  - fallback success
- First real provider adapter:
  - implement one real LLM provider behind feature flag
  - support API key credential config
  - normalize request/response/errors
- API gateway:
  - `POST /api/v1/chat/completions`
  - `GET /api/v1/models`
  - API key authentication
  - project policy loading
  - model/profile lookup
  - provider dispatch
  - response normalization
- Billing core:
  - credit wallet
  - paid/promotional credit distinction
  - trial/demo credit grant
  - estimate cost
  - reserve credits
  - capture credits
  - release/refund credits
  - immutable ledger entries
  - usage event creation
  - customer charge vs provider cost fields
- Dashboard:
  - credit balance
  - usage list
  - recent requests
  - API keys
  - conversations
- Basic rate limits:
  - API key request limit
  - login limit
  - file size placeholder
- Basic logging:
  - request ID
  - correlation ID
  - user/org/project IDs
  - model/provider
  - latency/status/error

### Deliverables

- User can select a model/profile and chat in Workbench.
- Conversation is saved and can be resumed.
- User can switch models/profiles and keep usable context.
- API client can call `/api/v1/chat/completions`.
- Credits are reserved/captured/refunded correctly for mock calls.
- Admin can manage model/provider/profile/price metadata.
- One real provider can be enabled through config/feature flag.

### Exit Criteria

- End-to-end core loop works with mock provider.
- End-to-end core loop works with one real provider in controlled dev/staging config.
- Billing ledger remains correct for success, failure, timeout, and fallback scenarios.
- Metadata manager can update a model profile and the new config affects subsequent calls.
- Core integration tests cover auth, API keys, chat, routing, billing, and metadata profile resolution.

## 4. Phase 3: Payments, Files, Async Jobs, and Production Readiness

### Objective

Make the system production-capable for an initial launch: payment top-up, file upload, async jobs, provider fallback, observability, security scanning, performance tests, backup, and deployment workflow.

### Scope

- Payments:
  - payment provider integration
  - checkout/top-up flow
  - payment webhook verification
  - paid credit posting
  - failed payment handling
  - refund flow
  - invoice/receipt metadata
- File and asset foundation:
  - upload files
  - object storage integration
  - file metadata table
  - file size/type validation
  - attach files to conversation
  - basic text/PDF extraction placeholder
- Context and retrieval:
  - extracted text chunks
  - embeddings placeholder or pgvector integration
  - retrieval-aware context packing
  - pinned messages
  - context summaries
- Async jobs:
  - async job table
  - worker process
  - job status
  - retry/cancel
  - webhook delivery foundation
  - mock image/video/audio job
- More provider support:
  - second real LLM provider
  - basic fallback routing
  - provider health checks
  - provider attempt records
- Workbench expansion:
  - file upload in chat
  - image generation UI with mock adapter
  - async job status UI
  - asset library basics
- Observability:
  - structured logs
  - metrics endpoint or collector integration
  - provider health dashboard
  - queue depth visibility
  - billing anomaly logs
- Rate limiting and abuse:
  - Redis-backed rate limits
  - per API key/project limits
  - upload limits
  - basic account/API key suspension
- Security:
  - dependency scanning
  - container scanning
  - secret scanning
  - static security checks
  - file upload safety checks
  - webhook replay protection
- Performance:
  - smoke load tests
  - chat API test
  - streaming test
  - billing path test
  - dashboard query test
- Backup/restore:
  - PostgreSQL backup docs
  - restore test script/runbook
  - object storage retention/versioning plan
- CI/CD:
  - build/test workflow
  - container image build
  - staging deployment workflow
  - production release checklist

### Deliverables

- User can buy credits.
- Payment webhook adds paid credits idempotently.
- User can upload a file and use it in a chat flow.
- Async mock generation jobs work and persist assets.
- Fallback between providers works.
- Logs/metrics/rate limits/security scans are active.
- Staging deployment is reproducible.
- Backup and restore runbook exists and has been tested once.

### Exit Criteria

- Payment top-up and refund paths pass integration tests.
- Provider fallback charges the customer once and records internal failed attempts.
- File upload validation blocks unsafe/unsupported files.
- Performance smoke tests have baseline numbers.
- High/critical security scan findings are resolved or formally waived.
- Staging can be deployed from CI/CD.

## 5. Phase 4: Scale, Enterprise, Provider Portal, and Advanced Operations

### Objective

Expand beyond MVP into enterprise readiness, provider self-service, advanced reporting, advanced routing, SDKs, stronger governance, and scale hardening.

### Scope

- Enterprise:
  - SSO/SAML
  - SCIM
  - organization-wide policies
  - data retention controls
  - zero data retention mode
  - IP allowlists
  - enterprise invoices
  - audit exports
  - custom price contracts
- Provider portal:
  - provider login
  - model submission
  - endpoint metadata
  - provider usage analytics
  - provider settlement reports
  - provider health/error reports
- Advanced routing:
  - cheapest route
  - fastest route
  - quality-weighted route
  - region-aware route
  - enterprise allowlist/denylist
  - global emergency provider disable
- Advanced model/workbench:
  - model comparison UI
  - image generation with real provider
  - audio/video integrations as providers become available
  - prompt presets
  - shared team libraries
  - conversation branches UI
- Analytics and reporting:
  - business dashboards
  - margin reports
  - model performance reports
  - provider settlement reports
  - CSV/JSON exports
  - enterprise usage reports
- SDKs and API lifecycle:
  - TypeScript SDK
  - Python SDK
  - Go SDK
  - API contract generation
  - webhook replay/test UI
  - public changelog
- Security and compliance:
  - MFA
  - advanced audit logs
  - SOC 2 readiness work
  - DPA workflow
  - admin approval workflows
  - stronger abuse/fraud rules
- Scale and resilience:
  - split backend API/worker/scheduler/webhook services
  - analytics store migration if needed
  - search service if PostgreSQL search is insufficient
  - advanced caching
  - DR drills
  - performance load tests
  - incident response process

### Deliverables

- Enterprise workspace controls are usable.
- Provider portal supports basic self-service.
- Advanced routing policies are configurable.
- SDKs are available for primary developer languages.
- Reporting supports business, operations, customer, and provider needs.
- System can scale core services independently.

### Exit Criteria

- Enterprise pilot can onboard with SSO, policies, audit exports, and invoice workflow.
- Provider can view model usage and settlement report.
- Advanced routing has test coverage and production controls.
- SDKs pass contract tests.
- DR drill and load test have documented results.

## 6. Cross-Phase Engineering Rules

- Do not store model weights or training data.
- Use provider/model metadata and model profiles to configure external model calls.
- Keep billing ledger immutable after posting.
- Keep raw API keys, OAuth tokens, provider credentials, and payment secrets out of logs.
- Use idempotency keys for billing, payments, webhooks, and async job completion.
- Keep migrations versioned and tested.
- Keep mock data in the repo and resettable.
- Add tests with every billing, auth, routing, and provider adapter change.
- Prefer feature flags for risky provider, pricing, routing, and metadata-manager changes.
- Keep README updates for after the first runnable implementation milestone.

## 7. Suggested Immediate Next Tasks

1. Scaffold `frontend/` with Node.js + TypeScript.
2. Scaffold `backend/` with Go.
3. Add `docker-compose.yml` for frontend, backend, PostgreSQL, Redis, and local storage.
4. Add initial database migrations.
5. Add seed/reset scripts and mock data.
6. Add backend health endpoint.
7. Add frontend app shell.
8. Add basic CI for build/typecheck/tests/migrations.
