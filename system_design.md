# System Design: Model Market and Unified AI Gateway

## 1. Purpose

This document describes the end-to-end system design for a model marketplace, unified model API gateway, and user-facing AI workbench.

The product will let users discover models, buy credits, use models through the website or API, persist conversation context, generate multimodal assets, and track usage and billing. The system should run locally in dev mode with repository-owned mock data and should deploy as containerized frontend and backend services.

The MVP is an API bridge and web application. It does not train models, store model weights, or run GPU inference infrastructure. Model execution happens through third-party provider APIs or customer/self-hosted endpoints connected through provider adapters.

## 2. Technology Stack

### Frontend

- Runtime/build stack: Node.js + TypeScript.
- Application type: web application serving public pages, dashboard, catalog, docs, workbench, admin UI, and provider portal.
- Recommended framework: Next.js or another TypeScript-first web framework.
- Responsibilities:
  - Public website.
  - Model catalog and pricing pages.
  - Workbench UI.
  - Developer dashboard.
  - Metadata manager UI.
  - Admin and provider portal.
  - Authentication UI.
  - Billing and usage views.

### Backend

- Language: Go.
- Application type: API server plus background workers.
- Responsibilities:
  - Authentication and session APIs.
  - OAuth login for Google and Facebook.
  - User, organization, project, and API key management.
  - Model catalog APIs.
  - Unified model API gateway.
  - Provider adapters.
  - Context manager.
  - Billing, wallet, credit ledger, and pricing engine.
  - Payment integration and webhooks.
  - Usage metering.
  - Async job processing.
  - Admin and provider APIs.

### Data and Infrastructure

- PostgreSQL: primary database for persisted data and metadata.
- Redis: cache, sessions, locks, queues, rate limits, and provider health cache.
- Object storage: uploaded files and generated assets. Local dev can use MinIO or a mounted volume.
- pgvector or vector database: embeddings and retrieval context.
- PostgreSQL full-text search for MVP catalog/docs search; dedicated search service can be added later.
- Payment provider: Stripe or equivalent.
- External model providers: OpenAI, Anthropic, Google, xAI, Replicate, custom/self-hosted providers.

## 3. E2E Architecture Diagram

```text
                                USERS
          --------------------------------------------------
          | Developers | Teams | Enterprise | Model Providers |
          --------------------------------------------------
                              |
                              v
+-----------------------------------------------------------------------+
|                            FRONTEND CONTAINER                         |
|-----------------------------------------------------------------------|
| Node.js + TypeScript Web App                                          |
| Public Site | Model Catalog | Pricing | Docs | Workbench              |
| Dashboard   | Playground    | Admin   | Metadata Manager | Provider  |
+-----------------------------------------------------------------------+
                              |
                         HTTPS / API
                              |
                              v
+-----------------------------------------------------------------------+
|                            BACKEND CONTAINER                          |
|-----------------------------------------------------------------------|
| Go API Server and Workers                                             |
| Auth & OAuth         | User / Org / Project Management                |
| Model Catalog API    | Workbench / Chat API                           |
| Metadata APIs        | API Gateway / Router                           |
| Provider Adapters    | Billing / Credit                               |
| Usage Metering       | Payment Webhooks                               |
| Async Job Workers    | Admin APIs | Provider Management                |
+-----------------------------------------------------------------------+
        |             |              |              |             |
        v             v              v              v             v
+-------------+  +----------+  +-------------+  +-----------+  +-----------+
| PostgreSQL  |  |  Redis   |  | Object      |  | Vector /  |  | Search /  |
| Primary DB  |  | Cache /  |  | Storage     |  | pgvector  |  | Full Text |
| Metadata    |  | Queue    |  | Files/Media |  | Memory    |  | Catalog   |
+-------------+  +----------+  +-------------+  +-----------+  +-----------+
        |
        v
+-----------------------------------------------------------------------+
|                       STORED DATA / METADATA                          |
|-----------------------------------------------------------------------|
| Users, Orgs, Projects, API Keys, Roles                                |
| OAuth Accounts, Sessions, Login Events                                |
| Models, Providers, Capabilities, Pricing Rules                        |
| Model Metadata Only; No Model Weights or Training Data                |
| Wallets, Credits, Ledger, Payments, Invoices                          |
| Conversations, Messages, Context Summaries, Branches                  |
| Uploaded Files, Generated Images/Videos/Audio                         |
| Usage Events, Provider Attempts, Costs, Margin                        |
| Audit Logs, Webhooks, Admin Actions                                   |
+-----------------------------------------------------------------------+

                              |
                              v
+-----------------------------------------------------------------------+
|                        EXTERNAL INTEGRATIONS                          |
|-----------------------------------------------------------------------|
| Model APIs: OpenAI | Anthropic | Google | xAI | Replicate | Custom     |
| OAuth: Google | Facebook                                           |
| Payments: Stripe or Equivalent                                       |
| Email / Notification Provider                                         |
| Optional Analytics / Monitoring / Status Page                         |
+-----------------------------------------------------------------------+
```

## 4. Product Surface

### 4.1 Public Website

- Marketing landing page.
- Model catalog.
- Model detail pages.
- Pricing pages.
- Documentation pages.
- Enterprise contact flow.
- Provider partnership pages.

### 4.2 Workbench

The workbench is the user-facing creation and testing environment. It combines chat, multimodal generation, comparison, and developer playground features.

```text
Workbench
├── Chat
│   ├── Talk with LLMs
│   ├── Switch models
│   ├── Upload files
│   ├── Continue saved conversations
│   └── Compare responses
│
├── Image
│   ├── Text-to-image
│   ├── Image edit
│   ├── Reference image generation
│   └── Asset history
│
├── Video
│   ├── Text-to-video
│   ├── Image-to-video
│   ├── Start/end frame video
│   └── Async job tracking
│
├── Audio
│   ├── Text-to-speech
│   ├── Speech-to-text
│   ├── Voice generation
│   └── Audio asset history
│
├── Compare
│   ├── Same prompt across models
│   ├── Cost comparison
│   ├── Latency comparison
│   └── Output quality comparison
│
└── Playground
    ├── API-style testing
    ├── Raw parameters
    ├── Request/response inspector
    └── Copy SDK/curl examples
```

### 4.3 Dashboard

- Project management.
- API key management.
- Credit balance and top-up.
- Usage charts.
- Billing history.
- Invoices and receipts.
- Conversation and asset library.
- Team and role management.
- Budget and routing policies.

### 4.4 Admin Console

- User and organization management.
- Model/provider management.
- Price rule management.
- Credit adjustments and refunds.
- Usage inspection.
- Provider health.
- Incident banners.
- Audit logs.

### 4.5 Metadata Manager

The metadata manager is an admin UI for maintaining the data that powers the marketplace and API bridge. It manages model/provider metadata only; it does not manage model weights or GPU execution.

- Model metadata:
  - Name.
  - Slug.
  - Provider.
  - Model version.
  - Lifecycle status.
  - Modality.
  - Capabilities.
  - Context limits.
  - File limits.
  - Supported parameters.
  - Safety settings.
  - Documentation metadata.
- Model configuration metadata:
  - Profile name.
  - Underlying provider model.
  - Default system prompt.
  - Instruction template.
  - Default parameters.
  - Response format defaults.
  - Tool/function defaults.
  - Safety defaults.
  - Multimodal generation defaults.
  - Provider-specific request fields.
  - Allowed project/user overrides.
  - Published configuration version.
  - Rollback target.
- Provider metadata:
  - Provider name.
  - Endpoint configuration.
  - Auth credential reference.
  - Region.
  - Rate limits.
  - Health status.
  - Data policy.
  - SLA metadata.
- Pricing metadata:
  - Customer price rules.
  - Provider cost rules.
  - Unit types.
  - Effective dates.
  - Volume tiers.
  - Promotion overrides.
- Routing metadata:
  - Model aliases.
  - Provider priorities.
  - Fallback rules.
  - Region rules.
  - Enterprise allowlists.
- Dev metadata:
  - Mock models.
  - Mock providers.
  - Mock price rules.
  - Mock provider health.
  - Mock usage examples.

Model profiles let the platform customize how a provider model is called without hosting the model. At request time, the backend resolves the selected profile into provider-ready parameters, prompts, safety settings, and routing metadata.

### 4.6 Provider Portal

- Provider profile.
- Model submission.
- Endpoint configuration.
- Usage analytics.
- Revenue and settlement reporting.
- Health and error reports.

## 5. Core Runtime Flows

### 5.1 Website Chat Flow

```text
User selects model in Workbench
        |
        v
Frontend sends message, files, parameters, and conversation ID
        |
        v
Backend authenticates session and loads user/project policy
        |
        v
Backend loads wallet balance and estimates cost
        |
        v
Context manager builds provider-ready context
        |
        v
Billing reserves credits
        |
        v
Router chooses model/provider endpoint
        |
        v
Provider adapter sends normalized request
        |
        v
Provider streams or returns response
        |
        v
Backend stores message, usage, cost, provider attempt, and route
        |
        v
Billing captures actual credits or refunds reservation
        |
        v
Frontend displays response, model, provider, cost, and usage
```

### 5.2 API Gateway Flow

```text
External app calls /api/v1/chat/completions with API key
        |
        v
Backend authenticates API key and project
        |
        v
Backend enforces model, provider, budget, and rate policies
        |
        v
Pricing engine estimates charge
        |
        v
Billing reserves credits if required
        |
        v
Router dispatches request to provider adapter
        |
        v
Provider response is normalized into platform schema
        |
        v
Usage metering records customer charge and provider cost
        |
        v
Billing captures or releases credits
        |
        v
Response returns to external app
```

### 5.3 Async Generation Flow

```text
User requests video/image/audio generation
        |
        v
Backend creates async job and reserves estimated credits
        |
        v
Worker dispatches provider request
        |
        v
Provider returns queued/running/completed/failed status
        |
        v
Worker polls or receives webhook
        |
        v
Generated asset is stored in object storage
        |
        v
Asset metadata, prompt, model, provider, and usage are stored
        |
        v
Billing captures actual charge or refunds reservation
        |
        v
Frontend shows final asset and job history
```

### 5.4 Model Switch Context Flow

```text
User switches from Model A to Model B
        |
        v
Backend loads canonical conversation history
        |
        v
Context manager checks Model B capability and context window
        |
        v
Context manager selects relevant messages, summaries, files, and assets
        |
        v
Backend creates a new branch if needed
        |
        v
Prompt is transformed into Model B provider format
        |
        v
Router sends request to Model B
```

### 5.5 Model Profile Resolution Flow

```text
User/API request selects model or model profile
        |
        v
Backend loads provider model metadata
        |
        v
Backend loads admin-published model profile
        |
        v
Backend applies organization policy and project preset
        |
        v
Backend merges request-level allowed overrides
        |
        v
Backend validates parameters against provider capabilities
        |
        v
Backend records resolved profile/config version
        |
        v
Provider adapter sends prompt, system message, parameters, and safety settings
```

### 5.6 Credit Purchase Flow

```text
User starts top-up
        |
        v
Frontend creates payment session through backend
        |
        v
Payment provider processes payment
        |
        v
Payment webhook notifies backend
        |
        v
Backend verifies webhook signature and idempotency
        |
        v
Credit ledger posts paid credit transaction
        |
        v
Wallet balance updates
        |
        v
User can consume credits
```

## 6. Backend Service Design

The MVP can run as one Go backend image with multiple logical services. Production should allow API server, worker, scheduler, and webhook dispatcher to scale separately.

### 6.1 Logical Services

- Auth service:
  - Email login.
  - Google OAuth.
  - Facebook OAuth.
  - Session/token management.
  - MFA hooks for future enterprise support.
- User/org/project service:
  - User profile.
  - Organization membership.
  - Role-based access control.
  - Project settings.
- API key service:
  - API key creation.
  - Hashing.
  - Scopes.
  - Expiration.
  - Rotation.
- Catalog service:
  - Models.
  - Providers.
  - Capabilities.
  - Prices.
  - Docs metadata.
- Workbench service:
  - Conversations.
  - Messages.
  - Branches.
  - Assets.
  - File upload metadata.
- Context service:
  - Conversation packing.
  - Summaries.
  - File extraction.
  - Retrieval context.
  - Model compatibility checks.
  - Model profile prompt/configuration resolution.
- Router service:
  - Fixed model routing.
  - Fallback.
  - Provider health routing.
  - Policy enforcement.
- Provider adapter service:
  - Provider-specific request transformation.
  - Provider-specific response normalization.
  - Error normalization.
  - Provider usage capture.
  - No model weight storage.
  - No GPU worker operation for MVP.
- Billing service:
  - Wallets.
  - Ledger.
  - Reservations.
  - Captures.
  - Refunds.
  - Adjustments.
- Pricing service:
  - Customer price rules.
  - Provider cost rules.
  - Cost estimates.
  - Margin calculation.
- Payment service:
  - Payment sessions.
  - Webhooks.
  - Receipts.
  - Refunds.
- Usage service:
  - Request records.
  - Provider attempts.
  - Token/unit counts.
  - Latency.
  - Customer charge.
  - Provider cost.
- Job service:
  - Async jobs.
  - Worker queue.
  - Retry.
  - Polling.
  - Webhook completion.
- Admin service:
  - Model/provider operations.
  - Manual credit adjustment.
  - User support.
  - Audit inspection.
- Metadata service:
  - Model metadata management.
  - Model profile management.
  - Model configuration versioning.
  - Provider metadata management.
  - Capability metadata management.
  - Pricing metadata management.
  - Routing metadata management.
  - Mock metadata management.

## 7. Database Design Scope

PostgreSQL is the source of truth for business data and metadata. Large files live in object storage, while PostgreSQL stores metadata and storage references.

The database stores model metadata, not model data. It must not store foundation model weights, training datasets, provider proprietary internals, or GPU execution state.

The database may store model configuration metadata used to call provider models, including default system prompts, instruction templates, provider parameters, safety settings, response formats, and versioned model profiles.

### 7.1 Main Entity Groups

- Identity:
  - users
  - oauth_accounts
  - sessions
  - login_events
  - mfa_factors
- Organization:
  - organizations
  - memberships
  - roles
  - invitations
  - projects
  - project_settings
- API access:
  - api_keys
  - api_key_scopes
  - rate_limits
  - budget_policies
  - routing_policies
- Catalog:
  - providers
  - provider_endpoints
  - models
  - model_versions
  - model_aliases
  - model_capabilities
  - model_parameters
  - model_profiles
  - model_configurations
  - model_configuration_versions
  - model_pricing_metadata
  - model_documentation_metadata
- Pricing and billing:
  - wallets
  - credit_balances
  - ledger_transactions
  - credit_reservations
  - payments
  - invoices
  - invoice_items
  - coupons
  - price_rules
  - provider_cost_rules
  - provider_settlements
- Usage:
  - inference_requests
  - provider_attempts
  - usage_events
  - async_jobs
  - job_events
- Workbench:
  - conversations
  - conversation_branches
  - messages
  - message_attachments
  - context_summaries
  - prompt_presets
  - workspace_assets
- Files and retrieval:
  - uploaded_files
  - file_extractions
  - file_chunks
  - embedding_records
  - retrieval_events
- Operations:
  - audit_logs
  - webhook_endpoints
  - webhook_deliveries
  - notifications
  - provider_health_events
  - admin_actions

### 7.2 Required Metadata

- Every model response should record model, model version, provider, route, request ID, token/unit usage, customer charge, provider cost, margin, latency, status, and error class.
- Every model response should record selected model profile, resolved model configuration version, and relevant request-time overrides when applicable.
- Every generated asset should record prompt, input references, model, provider, parameters, storage path, MIME type, size, moderation state, billing event, and owner project.
- Every conversation message should record role, content, attachments, selected model, provider, branch, timestamps, token counts, and safety state.
- Every billing mutation should record source, idempotency key, amount, currency/credit unit, reason, related request/job/payment, actor, and immutable posting timestamp.
- Every OAuth account should record provider, provider account ID, linked user, email, display name, avatar URL where allowed, and last login metadata.
- No database table should store model weights or training data. The system stores provider endpoint references and metadata needed to call external models.
- Model configuration tables may store prompts, parameters, safety defaults, routing preferences, and provider request transformation rules, but not model weights.

## 8. Dev Mode and Mock Data

The repo should include mocked data and metadata so developers can run the full product locally without real provider credentials, payment credentials, or production data.

### 8.1 Required Dev Assets

- Database migrations.
- Seed scripts.
- Mock users.
- Mock organizations and projects.
- Mock API keys.
- Mock wallets and credit balances.
- Mock providers and models.
- Mock model capabilities and price rules.
- Mock conversations and messages.
- Mock uploaded file metadata.
- Mock generated image/video/audio metadata.
- Mock usage events.
- Mock invoices and coupons.
- Mock provider health states.
- Mock routing policies.

### 8.2 Mock Adapters

Mock provider adapters should return deterministic responses for:

- Chat completion.
- Streaming chat.
- Image generation.
- Video generation.
- Audio generation.
- Embeddings.
- File Q&A.
- Provider failure.
- Provider timeout.
- Provider rate limit.
- Fallback success.

### 8.3 Mock Payments

Mock payment flows should simulate:

- Successful top-up.
- Failed payment.
- Refund.
- Invoice creation.
- Coupon redemption.
- Credit expiration.
- Insufficient credit path.

## 9. Container Design

### 9.1 Local Docker Compose

Local development should start the following:

```text
docker-compose
├── frontend
├── backend
├── postgres
├── redis
└── object-storage or local asset volume
```

Expected local startup behavior:

1. PostgreSQL starts.
2. Redis starts.
3. Object storage or asset volume is available.
4. Backend runs migrations.
5. Backend loads mock seed data when dev mode is enabled.
6. Backend exposes API and health endpoints.
7. Frontend starts and points to backend API URL.
8. Developer can log in with seeded users or mock OAuth flow.
9. Developer can use catalog, workbench, billing, and API key flows locally.

### 9.2 Production Containers

- Frontend and backend should be separately deployable.
- Backend image should support API, worker, scheduler, and webhook dispatcher modes.
- Containers should expose health and readiness endpoints.
- Containers should run without baked-in secrets.
- Logs should be emitted to stdout/stderr.
- Static files and uploads should not depend on container-local storage in production.
- Database migrations should be controlled and auditable.

## 10. Authentication Design

Initial authentication methods:

- Email login.
- Google OAuth login.
- Facebook OAuth login.

Future authentication methods:

- Passwordless login.
- GitHub OAuth.
- Enterprise SSO/SAML.
- SCIM provisioning.
- MFA enforcement.

OAuth requirements:

- OAuth accounts can be linked to existing users by verified email or explicit account-linking flow.
- A user may link multiple OAuth providers to one account.
- OAuth login should create a user and default organization on first successful signup.
- Login events should be stored for audit and security review.
- OAuth provider secrets must be injected by environment variables or secret manager.

## 11. API Surface

### 11.1 Public API

- `POST /api/v1/chat/completions`
- `POST /api/v1/images/generations`
- `POST /api/v1/images/edits`
- `POST /api/v1/videos/generations`
- `POST /api/v1/audio/speech`
- `POST /api/v1/audio/transcriptions`
- `POST /api/v1/embeddings`
- `GET /api/v1/models`
- `GET /api/v1/models/{id}`
- `GET /api/v1/jobs/{id}`
- `POST /api/v1/webhooks/test`

### 11.2 Dashboard API

- Auth and OAuth callback endpoints.
- User/org/project endpoints.
- API key endpoints.
- Wallet and payment endpoints.
- Usage endpoints.
- Conversation and asset endpoints.
- File upload endpoints.
- Workbench execution endpoints.

### 11.3 Admin API

- User search.
- Organization management.
- Model/provider management.
- Metadata manager endpoints for models, providers, capabilities, price rules, route policies, and dev mock records.
- Price rule management.
- Credit adjustment.
- Usage inspection.
- Provider health.
- Audit log search.

## 12. Billing Design

Billing must keep customer credits separate from provider costs.

For each request or async job, the backend should record:

- Customer price rule.
- Provider cost rule.
- Input units.
- Output units.
- Customer charge.
- Provider cost.
- Margin.
- Credit reservation.
- Final capture/refund.
- Provider attempt details.

Rules:

- Customer is charged once for final successful fallback result.
- Failed provider attempts are still recorded as internal cost events.
- Streaming calls reserve estimated credits and settle after completion.
- Async jobs reserve credits at creation and settle on completion/cancellation/failure.
- Ledger entries are immutable after posting.
- Manual adjustments require audit logs.

## 13. Open Implementation Decisions

- Frontend framework: Next.js is recommended but not yet final.
- Go web framework: standard library, Gin, Echo, Fiber, or another framework.
- Go database access: sqlc, GORM, Ent, or raw SQL.
- Queue implementation: Redis queue for MVP or dedicated queue later.
- Object storage: MinIO/local volume for dev, S3-compatible storage for production.
- Payment provider: Stripe is recommended but not yet final.
- First real model providers and launch model list.
- Whether the MVP includes only chat or also image generation in the first release.

## 14. Performance Testing Design

Performance testing should be included before launch and repeated for major gateway, billing, routing, database, and workbench changes.

### 14.1 Test Scenarios

- Public catalog browsing and search.
- User login and session validation.
- API key authentication.
- OpenAI-compatible chat completion.
- Streaming chat response.
- Workbench chat with persisted conversation history.
- Large conversation context resolution.
- File upload and metadata persistence.
- Async image/video/audio job creation and completion.
- Provider fallback under simulated provider errors.
- Billing credit reservation, capture, release, and refund.
- Payment webhook burst handling.
- Dashboard usage queries over recent and historical usage.
- Admin metadata manager model/profile/price updates.

### 14.2 Metrics

- Requests per second.
- Concurrent users.
- P50/P95/P99 latency.
- Time to first streamed token/chunk.
- Error rate.
- Database query latency.
- Redis latency.
- Queue depth and job wait time.
- Provider adapter overhead.
- Billing settlement time.
- Cost-estimation latency.
- Memory and CPU usage per container.

### 14.3 Tooling Direction

- Use a scriptable load-test tool such as k6, Locust, or JMeter.
- Keep representative performance test scripts in the repository.
- Provide a small smoke-performance suite for CI or pre-release checks.
- Provide a larger load-test suite for staging.
- Use mocked providers for stable baseline tests and real providers only for controlled external integration tests.

## 15. Security Scanning Design

Security scanning should be part of CI and production release gates.

### 15.1 Required Scan Categories

- Dependency vulnerability scanning for Go and Node.js packages.
- Container image scanning for frontend and backend images.
- Static application security testing for backend and frontend code.
- Secret scanning for source code, environment examples, and committed assets.
- Infrastructure/configuration scanning for Dockerfiles, Docker Compose, and deployment manifests.
- License compliance checks.

### 15.2 Runtime Security Test Areas

- Email login and OAuth login.
- Google and Facebook OAuth callback handling.
- Session handling.
- API key creation, hashing, display, rotation, and revocation.
- Authorization checks for users, projects, organizations, admin UI, and metadata manager UI.
- File upload validation and malware-risk handling.
- Payment webhook signature validation and replay protection.
- Provider credential storage and access.
- SSRF prevention for provider endpoints and uploaded-file processing.
- Rate limit and abuse control.
- IDOR prevention for conversations, assets, invoices, API keys, and projects.
- XSS prevention in chat messages, generated content previews, and admin metadata fields.

### 15.3 Release Gates

- High or critical dependency/container vulnerabilities should block production release unless explicitly waived.
- Secret scan findings should block release until resolved.
- Security-relevant admin and billing changes should include audit log coverage.
- Metadata manager changes should include authorization tests.
- Payment and ledger changes should include idempotency and replay tests.

## 16. Rate Limiting and DDoS Defense Design

Rate limiting and abuse protection should happen before expensive work such as provider dispatch, file processing, async job creation, or payment actions.

### 16.1 Rate Limit Dimensions

- IP address.
- User ID.
- Organization ID.
- Project ID.
- API key ID.
- Endpoint.
- Model.
- Provider.
- Route policy.
- Plan/tier.
- Payment risk state.
- Admin role.

### 16.2 Limit Types

- Requests per minute.
- Tokens per minute.
- Images/videos/audio jobs per minute.
- Concurrent requests.
- Concurrent streaming responses.
- Async jobs queued/running.
- File upload size and count.
- Daily/monthly spend budget.
- Login/signup attempts.
- Payment/top-up attempts.
- Metadata manager write operations.

### 16.3 Implementation Direction

- Use Redis-backed counters and token buckets for API and dashboard rate limits.
- Apply early request validation before provider calls.
- Emit standard rate-limit headers for public API clients.
- Store rate-limit events for support and abuse analysis.
- Support per-plan and per-customer override rules.
- Support emergency global throttles for provider outages or active attacks.
- Use CDN/WAF/load-balancer DDoS protections in production.
- Add bot protection to signup, login, top-up, referral, upload, and unauthenticated scraping surfaces.

### 16.4 Abuse Response

- Temporary throttling.
- API key suspension.
- Account review hold.
- Signup or login challenge.
- File upload block.
- Payment risk review.
- Admin unblock and appeal workflow.

## 17. Logging, Analytics, and Reporting Design

### 17.1 Logging

All backend services should emit structured logs with correlation IDs.

Required log dimensions where allowed:

- Request ID.
- Correlation ID.
- User ID.
- Organization ID.
- Project ID.
- API key ID.
- Endpoint.
- Model.
- Provider.
- Route.
- Status.
- Error class.
- Latency.
- Customer charge.
- Provider cost.
- Worker/job ID.

Never log:

- Raw API keys.
- OAuth access/refresh tokens.
- Provider API keys.
- Payment card data.
- Full private file contents.
- Secrets or signing keys.

### 17.2 Audit Logging

Audit logs are required for:

- Login and OAuth account linking.
- API key create/revoke/rotate.
- Organization membership and role changes.
- Credit adjustments.
- Refunds.
- Payment webhook processing.
- Price rule changes.
- Model profile/configuration changes.
- Provider endpoint and credential changes.
- Routing policy changes.
- Admin impersonation or support actions.
- Data export and deletion actions.

### 17.3 Product Analytics

Track:

- Signup funnel.
- Login activity.
- First model selection.
- First workbench message.
- First API key creation.
- First API request.
- First top-up.
- Model usage by modality.
- Workbench usage.
- Playground usage.
- Retention.
- Conversion.
- Churn.

### 17.4 Business Reporting

Reports should include:

- Revenue.
- Credits sold.
- Credits consumed.
- Paid vs promotional credit usage.
- Gross margin.
- Provider costs.
- Refunds.
- Failed payments.
- Coupon usage.
- Usage by customer/project/model/provider.
- Provider settlement.
- Enterprise invoice exports.

### 17.5 Operational Reporting

Reports should include:

- API traffic.
- Latency percentiles.
- Error rates.
- Provider health.
- Fallback rate.
- Queue depth.
- Async job completion time.
- Cache hit rate.
- Rate-limit events.
- Abuse events.
- Incident impact.

### 17.6 Export and Retention

- Reports should support dashboard views and CSV/JSON export.
- Enterprise users should be able to export usage, billing, and audit reports.
- Analytics retention should respect organization policy and privacy settings.
- Logs and analytics should be separable from raw user content where practical.

## 18. Data Retention, Backup, and Disaster Recovery Design

### 18.1 Retention Categories

- User-visible content:
  - Conversations.
  - Messages.
  - Uploaded files.
  - Generated images/videos/audio.
  - Workspace assets.
- Operational records:
  - Inference requests.
  - Usage events.
  - Provider attempts.
  - Async job events.
  - Webhook deliveries.
  - Rate-limit and abuse events.
- Business records:
  - Wallet balances.
  - Ledger transactions.
  - Payments.
  - Invoices.
  - Tax records.
  - Provider settlements.
- Governance records:
  - Audit logs.
  - Admin actions.
  - Metadata manager changes.
  - Security events.
  - Policy changes.
- Analytics records:
  - Product analytics.
  - Business analytics.
  - Operational analytics.

### 18.2 Retention Rules

- Conversations, files, and generated assets should follow user/project/organization retention settings.
- Enterprise organizations may disable persistence or set shorter retention windows where supported.
- Billing, tax, ledger, invoice, provider settlement, and audit records should follow legal/accounting retention rules.
- Provider request/response content should not be retained when zero data retention mode is enabled, except for minimal billing, abuse, and audit metadata required to operate the platform.
- Deletion workflows should remove user-visible content from active storage and mark records according to compliance policy.
- Backup deletion should follow retention-safe deletion rules and legal constraints.

### 18.3 Backup Strategy

- PostgreSQL:
  - Scheduled full backups.
  - Point-in-time recovery for production.
  - Migration-safe backup before high-risk schema changes.
  - Periodic restore tests.
- Object storage:
  - Durable/versioned storage for uploads, generated assets, invoices, and exports.
  - Lifecycle policies for expired assets.
  - Restore tests for selected assets.
- Redis and queues:
  - Redis should not be the only durable source for billing-critical or job-critical state.
  - Async jobs should be recoverable from PostgreSQL job records.
  - Queue replay/rebuild procedures should exist.
- Secrets:
  - Secrets should be stored in a secret manager or encrypted store.
  - Backup access should be restricted and audited.

### 18.4 Disaster Recovery

Define RPO and RTO targets before production launch.

Initial targets:

- MVP RPO: 24 hours or better for non-billing data; much lower for billing data where feasible.
- MVP RTO: same business day for non-critical services; faster for API and billing services where feasible.
- Enterprise targets should be stricter and contract-driven.

Runbooks should cover:

- PostgreSQL outage or data loss.
- Object storage outage or accidental asset deletion.
- Redis outage.
- Queue backlog or stuck workers.
- Payment webhook outage.
- Provider-wide outage.
- Bad metadata/profile/pricing publish.
- Bad database migration.
- Security incident requiring credential rotation.

### 18.5 Restore Testing

- Restore tests should be scheduled and documented.
- Tests should verify database restore, object metadata consistency, selected asset restore, and billing ledger integrity.
- Restore test failures should create operational action items.

## 19. Support, Notifications, and Customer Operations Design

### 19.1 Notification Events

Notifications should be generated for:

- Email verification.
- Login or OAuth account linking.
- API key creation, rotation, or revocation.
- Low balance.
- Successful top-up.
- Failed payment.
- Invoice available.
- Budget threshold reached.
- Async job completed or failed.
- Provider/model outage.
- Incident update.
- Account suspension or review.
- Enterprise policy change.

### 19.2 Notification Channels

- Email.
- In-app notifications.
- Webhook callback.
- Future channels such as Slack, SMS, or enterprise notification integrations.

### 19.3 Notification Requirements

- Notifications should be idempotent.
- Delivery attempts should be stored.
- Retries should use backoff.
- Users and organizations should be able to configure notification preferences.
- Sensitive notifications should avoid leaking secrets, private prompts, payment details, or raw generated content.

### 19.4 Support Operations

Admin/support tooling should allow authorized users to inspect:

- User and organization state.
- Project settings.
- API keys by ID and status, never raw keys.
- Wallet and ledger state.
- Payment and invoice state.
- Usage events.
- Provider attempts.
- Async jobs.
- Rate-limit decisions.
- Audit logs.
- Metadata manager publish history.

Support workflows should cover:

- Missing credits.
- Failed payments.
- Refunds.
- Usage disputes.
- Provider outage impact.
- API key compromise.
- Account unblock.
- File or generated asset issue.
- Enterprise support escalation.

## 20. CI/CD, Release, and Configuration Design

### 20.1 Environments

- Local/dev.
- Test.
- Staging.
- Production.

### 20.2 CI Checks

CI should run:

- Frontend typecheck.
- Frontend build.
- Backend tests.
- Backend build.
- Database migration validation.
- API contract tests.
- Integration tests with mocked providers.
- Dependency vulnerability scans.
- Container scans.
- Secret scans.
- Selected performance smoke tests.

### 20.3 Release Flow

Release flow should:

- Build frontend and backend container images.
- Tag images by version/commit.
- Run required CI gates.
- Publish images to a registry.
- Deploy to staging.
- Run smoke tests.
- Require production approval where appropriate.
- Deploy to production.
- Verify health checks.
- Support rollback.

### 20.4 Feature Flags and Configuration

Feature flags should support:

- New model/provider rollout.
- Workbench features.
- Payment features.
- Metadata manager features.
- Pricing/routing changes.
- Enterprise-only features.
- Beta customer access.

Configuration changes should be:

- Audited.
- Versioned.
- Validated before publish.
- Rollback-capable.
- Scoped by environment.

High-risk configuration includes:

- Price rules.
- Provider cost rules.
- Model profiles.
- Provider endpoints.
- Routing policies.
- Safety defaults.
- Payment settings.

## 21. API Lifecycle and SDK Design

### 21.1 API Versioning

- Public APIs should be versioned, starting with `/api/v1`.
- Breaking changes should require a new version or formal deprecation window.
- API errors should remain stable and documented.
- Every public API response should include or expose a request ID for support.

### 21.2 Contracts and Documentation

- Maintain API contract definitions for public API, dashboard API, and admin API.
- Keep docs aligned with contract tests.
- Document authentication, request/response schemas, error codes, rate limits, idempotency, streaming, async jobs, webhooks, and billing behavior.
- Publish model-specific capability notes.

### 21.3 SDKs

- SDKs should track public API versions.
- SDKs should include examples for chat, streaming, model list, async jobs, files/assets, billing usage, and webhooks.
- SDK releases should include changelog entries.
- Deprecated SDK versions should have support windows.

### 21.4 Webhook Lifecycle

- Webhooks should use signed payloads.
- Webhooks should include event IDs and timestamps.
- Webhooks should be idempotent.
- Delivery attempts should be stored.
- Failed deliveries should retry with backoff.
- Users should be able to replay or test webhook events.
