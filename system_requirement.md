# System Requirements: Model Market and Unified AI Gateway

## 1. Product Vision

Build a unified AI model marketplace and API gateway where developers, teams, and enterprises can discover, compare, pay for, and consume AI models through one account, one billing balance, and one API surface.

The platform is an API bridge and web application. It does not train models, host model weights, or require GPU inference infrastructure for the MVP. Model execution is performed by third-party providers or customer/self-hosted endpoints integrated through provider adapters.

The system should combine these product patterns:

- A broad model catalog and multimodal API marketplace like Atlas Cloud, covering LLM, image, video, audio, speech, embedding, reranking, and tool-capable models.
- A unified LLM routing layer like OpenRouter, with OpenAI-compatible APIs, provider fallback, model comparison, usage analytics, and credit-based consumption.
- Enterprise inference and deployment capabilities like SiliconFlow, including reserved capacity, private deployment, model acceleration, SLA support, and industry-specific solutions.

## 2. Target Users

- Individual developers who need fast API access to many models without managing multiple provider accounts.
- AI application builders who need reliable routing, fallback, cost control, and usage reporting.
- Startups and SaaS companies embedding AI into products.
- Marketing, creator, and design teams using image/video/audio generation.
- Enterprise teams requiring organization controls, compliance, private deployment, stable capacity, invoices, and support.
- Model providers who want distribution, billing, analytics, and demand aggregation.

## 3. Business Capabilities

### 3.1 Model Marketplace

- Public model catalog with search, filters, tags, categories, providers, modalities, capabilities, context length, latency, quality tier, input/output format, and pricing.
- Model detail pages with description, use cases, API examples, pricing table, limits, provider details, changelog, safety policy, sample outputs, and supported parameters.
- Model comparison by price, latency, quality, context size, throughput, modality, region, and availability.
- Featured, new, trending, cheapest, fastest, highest-quality, and enterprise-ready model collections.
- Provider marketplace for first-party, third-party, open-source, private, and self-hosted models.
- Optional model rankings and benchmark pages based on measured platform usage and curated evaluations.
- Admin-managed model configurations should allow the platform to define model profiles on top of provider models without hosting model weights.

### 3.2 Unified API Platform

- One API key can access all enabled models.
- OpenAI-compatible chat completions API for easy migration.
- Native APIs for model families and modalities where OpenAI compatibility is insufficient.
- Support for chat, text completion, structured output, function/tool calling, embeddings, reranking, moderation, image generation/editing, video generation/editing, audio generation, speech-to-text, text-to-speech, and multimodal input.
- Streaming and non-streaming responses.
- Async job API for long-running image, video, audio, and batch inference.
- Webhook callbacks for async completion, failure, moderation review, and billing events.
- SDKs for TypeScript, Python, Go, Java, and CLI usage.
- Request builder and interactive playground.

### 3.3 Chat and Creative Workspace

- Website users should be able to select a model from the catalog and immediately use it in a hosted chat/workspace interface.
- The chat interface should support text conversation with LLMs, reasoning models, coding models, agent-capable models, and multimodal models.
- Users should be able to switch models inside a conversation or start a new conversation with a selected model.
- When users switch models during a conversation, the system should preserve usable conversation context and continue the workflow with the new model when technically possible.
- Users should be able to compare multiple models side by side on the same prompt.
- The interface should show model name, provider, context window, estimated cost, current balance, generation status, and usage after each response.
- Users should be able to upload files and use supported models for document Q&A, summarization, extraction, analysis, and multimodal understanding.
- Supported upload types should include text documents, PDFs, images, audio files, video files, CSV/JSON data, and code files, subject to model capability and policy limits.
- Users should be able to generate images from text prompts, edit images, use reference images, and download or save generated assets.
- Users should be able to generate videos from text, image, start/end frames, references, or other supported model inputs.
- Users should be able to generate audio, transcribe audio, convert text to speech, and use voice input where supported.
- Long-running generations should appear as async jobs with progress, status, retry, cancel, history, and final asset download.
- Conversations and generated assets should be saved in a user library unless the user or organization disables retention.
- Users should be able to organize chats and assets by project, folder, tag, model, date, and modality.
- Users should be able to share conversations or assets with teammates based on organization permissions.
- The workspace should support prompt templates, presets, parameter controls, system prompts, seed/reference management, and negative prompts where supported.
- The workspace should allow users to select admin-defined model profiles and project-defined presets when permitted.
- The workspace should make model limitations clear, including unsupported file types, max file size, context limits, generation limits, safety restrictions, and expected cost.
- The workspace should provide safety review and moderation states for blocked, pending, or policy-sensitive outputs.

### 3.4 Persistent Context and History

- The system should persist user conversations, messages, model selections, uploaded files, generated assets, tool calls, job results, and relevant metadata.
- Conversation history should be stored independently from any single model provider so users can continue work across different models and providers.
- Each conversation should support a canonical history format that can be transformed into provider-specific request formats.
- The backend should track which model generated each message or asset, including provider, model version, route, parameters, usage, cost, latency, and request ID.
- Users should be able to resume conversations from the website, continue through the API when permitted, and view past generations in their project library.
- The context manager should build the best available prompt context for the selected model based on context window, modality support, file support, user preferences, organization retention policy, and cost limits.
- For long conversations, the system should support truncation, summarization, retrieval-augmented context, or user-selected pinned messages to keep the conversation usable across models with different context limits.
- Uploaded files should be persisted as project assets with extracted text, metadata, embeddings, thumbnails, transcripts, or previews where useful.
- Generated images, videos, and audio should be persisted as assets linked to the originating conversation, prompt, model, parameters, and billing event.
- The system should support conversation branches when a user switches models, regenerates a response, edits a prompt, or compares outputs.
- The system should support deletion, export, retention limits, and privacy controls for conversations, messages, files, and generated assets.
- Enterprise organizations should be able to disable persistence, set retention windows, require zero data retention for provider calls where supported, and restrict which data can be reused as context.
- The system should make context portability limitations visible when moving between models with different modalities, context lengths, tool support, safety policies, or file support.

### 3.5 Routing and Reliability

- Model aliases such as `provider/model`, `model-family/latest`, `fast`, `cheap`, and `best`.
- Provider fallback when one provider is unavailable, rate limited, too slow, or too expensive.
- Routing modes:
  - Fixed model.
  - Cheapest provider.
  - Lowest latency provider.
  - Highest availability provider.
  - Quality-weighted routing.
  - Region-specific routing.
  - Enterprise allowlist routing.
- Retry policy with idempotency keys.
- Circuit breakers for unhealthy providers.
- Provider health scoring based on latency, error rate, queue depth, success rate, and capacity.
- Graceful degradation for unavailable features.
- SLA reporting by user, organization, provider, and model.

### 3.6 Credit, Charging, and Payment

- Prepaid credit wallet for individuals and teams.
- Postpaid monthly invoicing for approved enterprise customers.
- Pay-as-you-go billing by token, image, video second, audio second, character, request, compute second, or custom provider unit.
- Customer-facing prices and backend provider costs must be tracked separately so the platform can calculate margin, detect loss-making routes, and reconcile provider invoices.
- Token pricing should support separate input token, output token, cached input token, reasoning token, image token, audio token, and provider-specific unit rates when applicable.
- Customer credits should be consumed according to the platform price rule in effect at request time, not directly according to the provider's raw invoice price.
- Real-time balance checks before request execution.
- Credit reservation for long-running jobs, with final settlement after completion.
- Refund or partial refund for failed, canceled, or provider-error jobs.
- Automatic top-up rules with thresholds and maximum monthly caps.
- Promotional credits, coupons, referral rewards, trial credits, and expiration policies.
- Payment methods:
  - Credit card.
  - ACH or bank transfer for enterprise.
  - Invoice payment.
  - Optional regional payment methods based on target markets.
- Tax, VAT, invoice, and receipt management.
- Revenue share settlement for model providers.
- Fraud detection for payment abuse, stolen cards, promotional credit abuse, and abnormal traffic.

### 3.7 Pricing and Packaging

- Public per-model usage pricing.
- Tiered usage discounts by volume.
- Developer, team, business, and enterprise plans.
- Subscription add-ons for priority support, advanced analytics, higher rate limits, private routing policies, and compliance features.
- Reserved capacity plans for predictable high-volume inference.
- Private deployment pricing for enterprise.
- Custom enterprise agreements with committed spend.
- Dynamic margin controls by provider, region, model, and customer segment.
- Pricing history and future effective-date scheduling.

### 3.8 User and Organization Management

- User sign-up/login with email, passwordless login, OAuth, and SSO/SAML for enterprise.
- Third-party account login should support Google and Facebook in the initial release, with the design allowing additional OAuth providers later.
- Organization workspaces.
- Role-based access control:
  - Owner.
  - Admin.
  - Billing admin.
  - Developer.
  - Analyst.
  - Read-only.
  - Provider admin.
- API key management with scoped permissions, expiration, rotation, environment labels, and rate limits.
- Project-level isolation for keys, budgets, routing policies, logs, and analytics.
- Team invitations and audit logs.
- Enterprise SCIM user provisioning.
- Account suspension, risk review, and compliance hold flows.

### 3.9 Developer Experience

- Dashboard onboarding flow:
  - Create account.
  - Add credits or start trial.
  - Generate API key.
  - Make first request.
  - View usage and cost.
- API docs with copyable examples.
- Model playground with side-by-side comparison.
- Request inspector showing token count, cost estimate, latency, provider route, cache status, and response metadata.
- Error code documentation and troubleshooting.
- SDK package registry publishing.
- Status page and incident history.
- Sandbox/test mode for integration testing.

### 3.10 Marketing and Growth

- SEO-friendly pages for models, providers, use cases, pricing, API documentation, and comparisons.
- Landing pages by persona:
  - Developers.
  - AI startups.
  - Enterprise AI teams.
  - Marketing/content teams.
  - Model providers.
- Use-case pages for chatbots, agents, coding, search, image generation, video generation, voice, education, ecommerce, and customer support.
- Referral program with trackable invitations and credit rewards.
- Promotion campaigns, coupons, top-up bonuses, and seasonal launch offers.
- Featured apps or customer showcase.
- Newsletter, changelog, announcements, and launch posts for new models.
- Partner and provider co-marketing pages.
- Analytics for acquisition funnel, activation, conversion, retention, and revenue.

### 3.11 Enterprise Capabilities

- Dedicated account management and support plans.
- Organization-wide budgets, quotas, and policy enforcement.
- Custom model/provider allowlists and denylists.
- Zero data retention option.
- Private networking, IP allowlists, and VPC peering where applicable.
- BYOC or private deployment for sensitive workloads.
- Reserved inference capacity and guaranteed throughput.
- Data residency and region routing.
- Compliance reporting, audit exports, DPA support, and security reviews.
- Enterprise invoices, contracts, procurement support, and purchase orders.

### 3.12 Model Provider Capabilities

- Provider onboarding workflow.
- Model submission, metadata management, pricing setup, capability declaration, API schema, sample prompts, and SLA declaration.
- Provider API credential vaulting.
- Provider settlement reports and payout tracking.
- Provider usage analytics by model, customer segment, geography, latency, error rate, and revenue.
- Provider quality monitoring and customer feedback.
- Provider terms, content policy, and data policy configuration.

## 4. System Layers

### 4.1 Frontend Layer

Primary surfaces:

- Public marketing site.
- Model marketplace.
- Pricing pages.
- Documentation portal.
- Hosted chat and creative workspace.
- Developer dashboard.
- Admin console.
- Metadata manager UI for administrators to manage model, provider, capability, pricing, routing, documentation, and mocked development metadata.
- Provider portal.
- Playground and request builder.
- Enterprise sales/contact flows.

Frontend requirements:

- Responsive web application.
- Strong search and filter UX for model discovery.
- Clear price display by unit and model.
- Authenticated dashboard for API keys, billing, usage, projects, settings, and logs.
- Real-time or near-real-time usage charts.
- Production-quality chat interface for text, file, image, video, and audio workflows.
- Side-by-side model testing with cost and latency display.
- Accessible UI with keyboard navigation and screen-reader support.
- Localization-ready copy and formatting.
- SEO support for public pages.

### 4.2 Backend Application Layer

Core services:

- API gateway service.
- Auth and identity service.
- Organization and project service.
- Model catalog service.
- Routing service.
- Provider adapter service.
- Usage metering service.
- Billing and credit ledger service.
- Payment service.
- Job orchestration service.
- Notification and webhook service.
- Admin service.
- Provider management service.
- Analytics service.
- Compliance and audit service.
- Metadata management service for model/provider/pricing/capability/routing metadata.

Backend requirements:

- Stateless API services where possible.
- Service-to-service authentication.
- Idempotent request handling for billing-sensitive operations.
- Centralized validation of model parameters and policy rules.
- Clear separation between external API, internal control plane, and admin operations.
- Horizontal scalability for high request volume.
- Background queues for async jobs, billing reconciliation, retries, webhooks, and reports.
- Strong observability across request, route, provider, cost, and user dimensions.

### 4.3 API Gateway and Inference Layer

Gateway capabilities:

- Authenticate API keys.
- Enforce project, organization, model, provider, region, and budget policies.
- Estimate cost before dispatch.
- Reserve credits when required.
- Normalize request payloads.
- Route to selected model/provider.
- Stream responses.
- Normalize provider responses into platform schema.
- Record usage and billing events.
- Apply retries and fallback rules.
- Emit logs, metrics, and traces.

Provider adapter capabilities:

- Adapter per provider or provider API family.
- Capability mapping for model parameters.
- Request/response normalization.
- Provider-specific auth.
- Provider-specific error mapping.
- Provider rate-limit handling.
- Provider health checks.
- Provider cost reconciliation.
- The system should not store model weights or operate GPU inference workers in the MVP.
- The system should store only provider/model metadata, request/response metadata, user data, conversation data, uploaded files, generated assets, usage data, and billing data required to operate the bridge.

### 4.4 Data Layer

Recommended data stores:

- PostgreSQL as the primary relational database for users, organizations, projects, model catalog, pricing, invoices, payments, credit ledger, provider metadata, and policies.
- Time-series or analytics database for usage events, latency, errors, token counts, and provider health.
- Object storage for generated assets, input uploads, logs requiring retention, invoices, exports, and documentation assets. For local development this can be a mounted volume or MinIO-compatible service.
- Redis or equivalent cache for sessions, rate limits, model metadata cache, provider health, balance cache, and short-lived locks.
- Queue system for async jobs, webhooks, settlements, reports, and provider retries.
- Search index for model catalog, docs, and marketplace search.
- Vector index or retrieval store for conversation memory, uploaded file chunks, embeddings, and project knowledge retrieval.
- Secret manager for provider keys, payment keys, webhook signing secrets, and encryption keys.

Minimum data and metadata to persist:

- Identity data: users, login identities, sessions, MFA factors, organization memberships, roles, invitations, SSO mappings, and audit events.
- Project data: organizations, projects, environments, API keys, API key scopes, project settings, budgets, rate limits, routing policies, and data retention policies.
- Model catalog data: providers, models, model versions, aliases, modalities, capabilities, supported parameters, context limits, file limits, safety settings, regions, endpoint availability, lifecycle status, launch/deprecation dates, and documentation metadata.
- Model configuration data: model profiles, default parameters, default system prompts, instruction templates, safety defaults, response format defaults, tool defaults, generation defaults, project overrides, and versioned configuration snapshots.
- Pricing data: customer price books, provider cost books, price rules, unit types, effective dates, volume tiers, enterprise overrides, coupons, promotions, exchange rates, and historical pricing snapshots.
- Billing data: wallets, paid credit balances, promotional credit balances, ledger transactions, credit reservations, captures, refunds, adjustments, expirations, payments, invoices, tax records, receipts, disputes, and provider settlements.
- Usage data: inference requests, async jobs, provider attempts, retries, fallback attempts, input units, output units, token counts, generated asset counts, latency, route, model, provider, status, errors, customer charge, provider cost, and margin.
- Conversation data: conversations, branches, messages, system prompts, user prompts, assistant responses, tool calls, selected models, model switches, pinned messages, context summaries, and share permissions.
- File and asset data: uploaded files, extracted text, file metadata, MIME type, size, hash, storage path, thumbnails, transcripts, generated images, generated videos, generated audio, prompt metadata, seed/reference data, moderation status, and download metadata.
- Retrieval data: file chunks, conversation memory chunks, embeddings, embedding model metadata, vector index references, retrieval scores, and source citations.
- Provider integration data: provider credentials references, endpoint configuration, rate limits, health state, SLA metadata, provider request IDs, provider-reported usage, provider errors, and reconciliation status.
- Operational data: logs requiring retention, metrics references, webhook endpoints, webhook delivery attempts, notifications, admin actions, incident banners, status events, and support metadata.

Data storage boundary:

- The system should not store foundation model weights, training datasets, provider proprietary model internals, or GPU execution state.
- The system should store model metadata only, such as model name, provider, version, capabilities, limits, pricing, endpoint configuration, status, and documentation metadata.
- The system may store model configuration metadata, such as default system prompts, default parameters, routing preferences, safety settings, request transformation rules, and model profile definitions, but not model weights.
- User content should be stored only as required for product features, such as conversation history, uploaded files, generated assets, usage records, and billing records, subject to user and organization retention policies.
- Provider responses should be stored as conversation messages or generated assets only when persistence is enabled for the user or organization.

Core relational entities:

- User.
- Organization.
- Membership.
- Role.
- Project.
- API key.
- Model.
- Model version.
- Model profile.
- Model configuration.
- Provider.
- Provider endpoint.
- Capability.
- Conversation.
- Conversation branch.
- Message.
- Message attachment.
- Workspace asset.
- Context summary.
- File extraction.
- Embedding record.
- Pricing plan.
- Price rule.
- Credit wallet.
- Ledger transaction.
- Usage event.
- Inference request.
- Async job.
- Payment.
- Invoice.
- Coupon.
- Referral.
- Routing policy.
- Budget policy.
- Audit log.
- Webhook endpoint.
- Provider settlement.

Database implementation requirements:

- Use migrations for every schema change.
- Use immutable ledger tables for posted billing entries.
- Use foreign keys for core ownership relationships where practical.
- Use soft deletion for user-visible conversations, files, generated assets, API keys, and projects unless compliance requires hard deletion.
- Use unique idempotency keys for payments, inference requests, ledger captures, refunds, and webhook events.
- Store raw provider secrets only in a secret manager or encrypted secret table; store references in the relational database.
- Separate large binary data from the database; store files/assets in object storage and keep metadata in PostgreSQL.
- Support export and deletion workflows for user data.
- Support indexes for common dashboard queries: recent usage, project spend, model usage, conversation history, asset library, billing history, and provider health.

Development and mock data requirements:

- The system should support a dev mode that can run end-to-end with mocked data, mocked metadata, mocked providers, mocked payments, and seeded users.
- Mocked data and metadata should live in the code repository so developers can start the system without access to production databases, real provider credentials, or payment credentials.
- Dev seed data should include sample users, organizations, projects, API keys, wallets, credit balances, models, providers, model capabilities, price rules, conversations, messages, uploaded file metadata, generated asset metadata, usage events, invoices, coupons, routing policies, and provider health states.
- Mock provider adapters should return deterministic text, image, video, audio, embedding, and error responses for testing chat, creative workspace, billing, retries, fallback, and async jobs.
- Mock payment flows should simulate successful top-up, failed payment, refund, invoice creation, coupon redemption, and credit expiration.
- Dev mode should clearly mark mock records and must not allow accidental use of mock payment/provider credentials in production.
- Seed scripts should be repeatable and should support resetting the local database to a known state.
- Mock data should cover common scenarios and edge cases such as low balance, insufficient credits, failed provider attempt, fallback success, long conversation context, file upload, async video job, refunded job, and enterprise organization policy.

### 4.5 User Management and Security Layer

- Secure authentication with MFA support.
- SSO/SAML for enterprise.
- Passwordless and OAuth options for developer onboarding.
- Initial OAuth providers should include Google and Facebook account login.
- API key hashing; never store raw API keys after creation.
- Fine-grained API key scopes.
- RBAC and project-level permissions.
- IP allowlists for dashboard and API keys.
- Audit logs for security, billing, policy, key, and admin events.
- Sensitive data encryption at rest and in transit.
- Data retention controls.
- Abuse detection and rate limiting.
- Content safety policy enforcement per model and customer.
- Compliance-ready logs and export controls.

### 4.6 Billing, Credit, and Ledger Layer

Ledger requirements:

- Double-entry or append-only ledger design.
- Every credit mutation must be traceable to payment, grant, refund, usage, adjustment, or expiration.
- Ledger transactions must be immutable after posting.
- Support pending, reserved, captured, refunded, expired, and adjusted credit states.
- Separate promotional credits from paid credits.
- Configurable credit expiration.
- Reconciliation between usage events, provider invoices, customer invoices, and internal ledger.
- Separate customer billing ledger from provider cost accounting while linking both to the same inference request or async job.
- Store the exact customer price rule, provider cost rule, exchange rate, unit counts, and timestamp used for each billed event.
- Support margin calculation per request, model, provider, project, organization, pricing plan, and time period.
- Support negative-margin alerts when provider costs exceed customer charges because of stale pricing, bad routing, retries, or provider unit conversion errors.
- Support manual billing adjustments with audit logs and approval workflow.

Usage metering requirements:

- Capture request ID, user, organization, project, API key, model, provider, route, input units, output units, billable units, cost, margin, latency, status, and error class.
- Support sync streaming requests and async jobs.
- Capture provider-reported usage when available.
- Estimate usage when provider usage is unavailable, with reconciliation correction.
- Prevent double charging on retries and fallback.
- For LLM calls, record prompt tokens, completion tokens, cached tokens, reasoning tokens, total tokens, provider-reported tokens, and platform-billable tokens where applicable.
- For multimodal calls, record billable dimensions such as image count, resolution, quality tier, video seconds, audio seconds, characters, file size, compute seconds, and custom provider units.
- For failed calls, record whether the provider charged the platform and whether the customer should be charged, refunded, or not billed.
- For fallback routes, charge the customer once for the successful final result while retaining internal cost records for failed provider attempts.
- For streaming calls, reserve estimated credits up front and settle actual usage after stream completion or interruption.
- For async jobs, reserve credits at job creation, adjust reservation when parameters change, capture on success, release on cancellation, and refund provider-error failures.

Pricing engine requirements:

- Maintain customer price books by model, provider, modality, unit type, region, tier, plan, promotion, and effective date.
- Maintain provider cost books by provider contract, model, endpoint, unit type, region, currency, and effective date.
- Support price overrides for enterprise contracts, volume tiers, coupons, free models, promotional campaigns, and internal test accounts.
- Support currency conversion and stable reporting currency.
- Support scheduled price changes without redeploying code.
- Expose cost estimates before request execution in dashboard, chat UI, playground, and API responses where possible.
- Preserve historical prices for auditability and accurate invoice reconstruction.

### 4.7 Model Integration Layer

Model onboarding requirements:

- Define model metadata, schema, capabilities, context limits, supported parameters, safety settings, and pricing.
- Define optional model profiles and configurations on top of provider models, including default prompts, default parameters, safety defaults, and request transformation rules.
- Configure provider endpoints and credentials.
- Run smoke tests, quality checks, latency tests, and billing tests before publishing.
- Support staged release: draft, internal, beta, public, deprecated, retired.
- Support model versioning and aliases.
- Support documentation and example generation for each model.

Model configuration requirements:

- The platform should support configurable model profiles that reference an underlying provider model.
- A model profile may define default system prompts, instruction templates, default temperature/top-p/max token settings, response format settings, tool/function defaults, safety options, multimodal defaults, and provider-specific parameters.
- Model profiles should be versioned so changes can be audited, tested, published, rolled back, and applied consistently.
- Admins should be able to mark profiles as public, internal, beta, enterprise-only, deprecated, or disabled.
- Projects and organizations may override allowed profile settings when policy permits.
- Request-time configuration resolution should merge settings in this order: platform defaults, provider model defaults, admin model profile, organization policy, project preset, user/request parameters.
- The backend should validate that merged configuration is supported by the selected provider model before dispatch.
- The backend should record the resolved model configuration version used for each inference request or async job.
- The system should distinguish model configuration from model hosting; configuration changes only affect request payloads, prompts, policies, and routing metadata.

Supported modality groups:

- LLM chat and completion.
- Reasoning models.
- Agent/tool-capable models.
- Embeddings.
- Rerankers.
- Moderation/classification.
- Image generation and editing.
- Video generation and editing.
- Speech-to-text.
- Text-to-speech.
- Music/audio generation.
- Multimodal input/output.

### 4.8 Operations and Observability Layer

- Centralized logs, metrics, and traces.
- Dashboard for provider health, model health, gateway traffic, revenue, cost, margin, errors, and incidents.
- Per-provider and per-model latency/error monitoring.
- Alerting for provider outage, billing anomaly, traffic spike, fraud pattern, queue backlog, failed payments, and credit ledger mismatch.
- Public status page.
- Incident management process and postmortems.
- Capacity planning reports.
- Synthetic tests against critical models and providers.
- Admin tooling for refunds, manual adjustments, suspensions, incident banners, and pricing changes.
- Admin metadata manager for editing and publishing model catalog metadata, provider endpoint metadata, capability definitions, price rules, route policies, mock records, and documentation metadata.

### 4.9 Rate Limiting, Abuse Protection, and DDoS Defense Layer

- Enforce rate limits by IP, user, organization, project, API key, endpoint, model, provider, and payment risk state.
- Support separate limits for public website traffic, authenticated dashboard traffic, public API traffic, file uploads, payment actions, OAuth callbacks, and admin endpoints.
- Support burst limits, sustained limits, concurrency limits, token-per-minute limits, request-per-minute limits, job queue limits, and daily/monthly budget limits.
- Support configurable rate-limit tiers by plan, customer, organization, project, API key, and enterprise contract.
- Return clear rate-limit headers and error responses for API clients.
- Protect expensive endpoints before provider dispatch by checking authentication, balance, policy, and request size early.
- Include DDoS protection at the edge through CDN/WAF/load balancer controls in production.
- Add bot protection and abuse controls for signup, login, top-up, referral, file upload, and unauthenticated catalog scraping.
- Detect abnormal traffic patterns such as credential stuffing, API key sharing, provider-cost abuse, prompt spam, file-upload abuse, webhook replay, and payment fraud.
- Support automated throttling, temporary bans, account review, API key suspension, and admin unblock workflows.
- Log rate-limit and abuse decisions for audit and support.

### 4.10 Logging, Analytics, and Reporting Layer

- Structured application logs should be emitted by frontend server, backend API, workers, scheduler, provider adapters, payment webhooks, and admin actions.
- Logs should include correlation IDs, request IDs, user/project/org IDs where allowed, API key ID, model, provider, route, status, latency, cost fields, and error class.
- Sensitive data such as raw API keys, payment data, OAuth tokens, provider secrets, and private file contents must not be logged.
- Support audit logs for login, OAuth linking, API key creation/revocation, billing changes, metadata manager changes, admin actions, refunds, price changes, provider credential changes, and policy changes.
- Product analytics should track acquisition, signup, activation, top-up, first API call, workbench usage, model usage, retention, conversion, and churn.
- Business analytics should track revenue, credits sold, credits consumed, gross margin, provider cost, refunds, failed payments, coupon usage, customer lifetime value, and provider settlement.
- Operational analytics should track traffic, latency, errors, provider health, queue depth, fallback rate, cache hit rate, rate-limit events, and incident impact.
- Reporting should include user/project usage reports, invoices, credit ledger exports, provider settlement reports, enterprise audit exports, cost/margin reports, and model performance reports.
- Reports should support dashboard views and CSV/JSON export where appropriate.
- Analytics and reporting data should respect retention policies, privacy settings, and enterprise logging controls.

### 4.11 Support, Notifications, and Customer Operations Layer

- Support user notifications for signup, login alerts, email verification, passwordless login, OAuth linking, API key creation, low balance, top-up success/failure, invoice availability, usage budget alerts, async job completion, provider incidents, and policy changes.
- Support notification channels including email, in-app notifications, webhook callbacks, and future SMS/Slack integrations.
- Provide support workflows for failed payments, refunds, missing credits, API key compromise, blocked accounts, usage disputes, provider failures, file/asset issues, and enterprise requests.
- Support admin support tools to inspect user account state, project usage, billing ledger, provider attempts, async jobs, rate-limit decisions, and audit logs.
- Support customer-facing help content, FAQ, contact forms, and issue escalation.
- Support incident banners and targeted customer notifications for provider outages, degraded models, billing incidents, or planned maintenance.
- Notifications should be idempotent, auditable, and retryable.
- Sensitive support actions should be permissioned and logged.

### 4.12 CI/CD, Release, and Configuration Management Layer

- The project should include CI workflows for frontend build, backend build, unit tests, integration tests, database migration checks, API contract tests, security scans, and selected performance smoke tests.
- The project should include release workflows for building and publishing frontend/backend container images.
- Environments should include local/dev, test, staging, and production.
- Deployment should support environment-specific configuration through environment variables and secrets.
- Feature flags should support controlled rollout of new providers, models, model profiles, pricing rules, workbench features, payment features, and admin metadata manager changes.
- Configuration changes for pricing, routing, model profiles, provider endpoints, and safety settings should be auditable and support publish/rollback.
- Database migrations should be versioned, tested, and reversible where practical.
- Production releases should include rollback guidance.
- CI/CD should prevent production deploys when required tests, security scans, migration checks, or release approvals fail.

### 4.13 API Lifecycle, Versioning, and SDK Layer

- Public APIs should be versioned and avoid breaking existing integrations.
- API changes should include compatibility review, documentation updates, contract tests, and changelog entries.
- Deprecation policies should be defined for models, endpoints, request fields, response fields, SDK versions, and provider routes.
- SDKs should be generated or maintained consistently with API contracts.
- API docs should include auth, errors, rate limits, billing behavior, idempotency, streaming behavior, async jobs, webhooks, examples, and model capability notes.
- Webhooks should support signed payloads, retry policies, delivery logs, replay protection, and test events.
- Public API errors should be stable, documented, and include request IDs for support.

### 4.14 Compliance, Privacy, and Governance Layer

- Terms of service, privacy policy, acceptable use policy, and model/provider-specific policy disclosures.
- Data processing agreement support for enterprise.
- User controls for request logging and retention.
- Zero data retention mode where supported.
- Region routing and data residency controls.
- Exportable audit logs.
- PII handling and redaction options.
- Compliance roadmap for SOC 2, ISO 27001, GDPR, and relevant regional requirements.
- Provider data policy transparency.
- Safety review workflow for high-risk usage.

### 4.15 Data Retention, Backup, and Disaster Recovery Layer

- Define retention policies for conversations, messages, uploaded files, generated assets, usage events, billing records, audit logs, operational logs, analytics events, invoices, and provider reconciliation data.
- Support user and organization-level retention settings where product and compliance requirements allow.
- Support zero data retention mode for provider calls where supported, while preserving required billing, abuse, and audit metadata.
- Support user-initiated deletion and export workflows for conversations, files, assets, and account data.
- Billing records, ledger entries, invoices, tax records, and audit logs should follow legal/accounting retention requirements and may outlive user-visible content.
- Back up PostgreSQL on a regular schedule with point-in-time recovery for production.
- Back up object storage or use versioned/durable object storage for uploaded files, generated assets, invoices, and exports.
- Back up or reconstruct Redis/queue state as appropriate; Redis should not be the only durable store for billing-critical or job-critical state.
- Test restore procedures regularly, including database restore, object metadata restore, and selected asset restore.
- Define recovery point objective and recovery time objective for MVP and enterprise tiers.
- Support disaster recovery runbooks for database loss, object storage loss, provider outage, payment webhook outage, queue backlog, and accidental metadata/pricing change.
- Keep backup access restricted, encrypted, audited, and separated from normal application credentials.
- Support retention-safe deletion from backups according to legal and technical constraints.

### 4.16 Deployment and Containerization Layer

The system should be deployable as containers with a simple local development setup and a production-ready path.

Required application containers:

- Frontend container:
  - Hosts the public website, model catalog UI, pricing pages, docs shell, dashboard, chat workspace, playground, admin console, and provider portal UI.
  - Should be built with Node.js and TypeScript.
  - Serves static assets and frontend routes.
  - Talks to the backend through configured API base URLs.
  - Supports environment-specific configuration without rebuilding for every deployment.
- Backend container:
  - Hosts the external API, dashboard APIs, auth/session APIs, model catalog APIs, routing/gateway logic, provider adapters, billing services, usage metering, async job workers, webhooks, and admin APIs.
  - Should be built with Go.
  - May run separate processes inside the same image for API server and worker in MVP, but production should allow them to scale independently.
  - Connects to database, cache, queue, object storage, payment provider, email provider, and upstream model providers through environment variables and secrets.

Required supporting services:

- PostgreSQL database for primary persisted data and metadata.
- Redis for cache, sessions, locks, queue backend, and rate limiting.
- Object storage for uploads and generated assets. Local development may use a mounted volume or MinIO.
- Optional search service for model/docs search. MVP may start with PostgreSQL full-text search.
- Optional vector database for embeddings and retrieval. MVP may use PostgreSQL with pgvector.
- Optional analytics/time-series store for high-volume usage metrics. MVP may start with PostgreSQL tables and later move hot analytics data to a specialized store.

Local development deployment:

- Provide a `docker-compose.yml` that starts frontend, backend, PostgreSQL, Redis, and local object storage or mounted asset volume.
- Provide seed data for initial admin user, sample providers, sample models, price rules, and demo credits.
- Provide repository-owned mock data and metadata for dev mode so the full product can be explored locally without real external services.
- Provide database migration commands that run automatically or through a documented command.
- Provide health checks for frontend, backend, database, Redis, and object storage.
- Provide environment variable examples for provider API keys, payment keys, JWT/session secrets, object storage settings, and public URLs.

Production deployment:

- Application images should be built from Dockerfiles and deployable to Kubernetes, ECS, Docker Compose, or another container platform.
- Frontend and backend should be independently scalable.
- Backend API server, worker, scheduler, and webhook dispatcher should be independently scalable once traffic grows.
- Database, Redis, object storage, and secret management should normally use managed services in production.
- Containers should expose health and readiness endpoints.
- Containers should run as non-root users where possible.
- Logs should be emitted to stdout/stderr for platform collection.
- Secrets should be injected by the deployment platform, not baked into images.
- Static asset and upload storage should survive container restarts.
- Database migrations should be controlled and auditable in production.

## 5. Core User Journeys

### 5.1 Developer First Request

1. User signs up.
2. User receives trial credits or purchases credits.
3. User creates a project.
4. User creates an API key.
5. User selects a model from the catalog.
6. User copies an SDK or curl example.
7. User makes a request.
8. Dashboard shows cost, latency, route, and usage.

### 5.2 Team Production Setup

1. Admin creates organization.
2. Admin invites developers and billing owners.
3. Admin creates projects for dev, staging, and production.
4. Admin configures budgets, rate limits, and model allowlists.
5. Developers rotate production API keys.
6. Team monitors spend, errors, and model performance.

### 5.3 Enterprise Procurement

1. Enterprise submits sales/contact form.
2. Sales/admin creates enterprise account.
3. Security review and DPA are completed.
4. SSO, invoices, private routing policies, and reserved capacity are configured.
5. Enterprise receives SLA reporting and support access.

### 5.4 Model Provider Onboarding

1. Provider applies or is invited.
2. Provider submits model metadata, endpoint details, and pricing.
3. Platform runs validation and billing tests.
4. Model is published to catalog.
5. Provider tracks usage, reliability, and revenue.
6. Platform settles revenue share.

## 6. Non-Functional Requirements

### 6.1 Availability and Reliability

- Public API target availability: 99.9% for MVP, 99.95%+ for enterprise tier.
- Dashboard target availability: 99.5% for MVP.
- Provider failures should not take down the gateway.
- Billing ledger must remain accurate under retries, partial failures, and async completion.

### 6.2 Performance

- API gateway authentication and routing overhead should be low enough to preserve provider latency advantage.
- Streaming responses should begin as soon as provider streams are available.
- Catalog pages should load quickly with cached metadata.
- Usage dashboard should support near-real-time updates for recent activity.
- Performance testing should be part of the release process for API gateway, chat streaming, async job handling, billing reservation/capture, catalog search, and dashboard usage queries.
- Load tests should cover concurrent API requests, concurrent workbench users, large conversation histories, file uploads, async generation jobs, provider fallback, and payment webhook bursts.
- Performance tests should report latency percentiles, throughput, error rate, queue depth, database query latency, Redis latency, provider adapter overhead, and billing settlement time.
- MVP performance targets should be defined before launch and tracked in CI or pre-release test runs.

### 6.3 Scalability

- Horizontal scaling for API traffic.
- Queue-based scaling for async creative generation.
- Partition or shard high-volume usage events.
- Cache model metadata and pricing but preserve strong consistency for billing-critical reads.

### 6.4 Security

- Secure API key generation and hashing.
- Least-privilege service credentials.
- Encryption in transit and at rest.
- Regular secret rotation.
- Abuse monitoring and automated throttling.
- Auditability for admin and billing actions.
- Security scanning should be part of CI and release workflows.
- Security checks should include dependency vulnerability scanning, container image scanning, static application security testing, secret scanning, infrastructure/configuration scanning, and license checks.
- Dynamic security testing should be performed for authentication, OAuth callbacks, API key flows, file uploads, payment webhooks, admin endpoints, and provider credential handling.
- The system should include protections against common web/API risks including injection, broken auth, insecure direct object references, SSRF, CSRF where applicable, XSS, unsafe file uploads, rate-limit bypass, and webhook replay.
- Admin and metadata manager actions should require authorization checks, audit logs, and high-risk action confirmation.
- Security scan failures above the configured severity threshold should block production release.

### 6.5 Maintainability

- Provider adapters must be modular.
- Pricing rules must be configurable without code deployment.
- Model catalog should support frequent launches and deprecations.
- API versioning must protect existing integrations.
- Strong automated tests for billing, routing, auth, and provider normalization.
- CI should run unit tests, integration tests, migration checks, API contract tests, security scans, and selected performance smoke tests.

## 7. MVP Scope

### 7.1 MVP Must Have

- Public landing page.
- Model catalog with model detail pages.
- Hosted chat interface for selected LLM models.
- User signup/login.
- Organization and project basics.
- API key creation and management.
- OpenAI-compatible chat completion endpoint.
- At least three provider integrations.
- Fixed-model routing and basic fallback.
- Credit wallet and prepaid top-up.
- Usage metering and dashboard.
- Basic billing ledger.
- Stripe or equivalent card payment integration.
- Admin console for users, models, prices, credits, and usage.
- Metadata manager UI for model/provider/capability/pricing/routing metadata.
- Documentation with quickstart examples.
- Basic status/health monitoring.

### 7.2 MVP Should Have

- Image generation async job API.
- Image generation workspace UI.
- File upload for chat with supported text/PDF/image files.
- Model playground.
- Coupons or promotional credits.
- Budget limits and email alerts.
- Provider health dashboard.
- Webhooks for async jobs.
- Basic referral tracking.

### 7.3 MVP Can Defer

- Full enterprise SSO/SCIM.
- Private deployment.
- Reserved capacity.
- Provider self-service portal.
- GPU-backed first-party model hosting.
- Advanced model rankings.
- Complex multi-region data residency.
- Revenue share automation.
- Advanced fraud ML models.

## 8. Suggested Architecture

```text
Public Site / Dashboard / Docs / Provider Portal
                  |
              Web Frontend
                  |
       Backend Control Plane APIs
                  |
     Auth | Catalog | Billing | Usage | Admin
                  |
          API Gateway / Router
                  |
       Provider Adapter Services
                  |
  OpenAI | Anthropic | Google | xAI | Replicate | Custom | Self-hosted

Data Stores:
Postgres | Redis | Object Storage | Queue | Analytics DB | Search Index | Secret Manager
```

## 9. Risk Areas

- Billing correctness under retries, streaming, provider errors, and async jobs.
- Provider price changes and inconsistent usage reporting.
- Provider outages causing customer-impacting failures.
- Abuse, fraud, and credential sharing.
- Legal and policy differences across model providers.
- Margin leakage from incorrect unit conversion or stale pricing.
- Model output safety and customer data handling.
- Scaling usage analytics without slowing API traffic.

## 10. Open Decisions

- Initial target market: global English developer market, China market, or both.
- Concrete implementation choices for frontend framework, Go backend framework, ORM/query library, queue, auth library, object storage, and payment integration.
- Initial provider list.
- Whether to start with LLM-only or include image/video from day one.
- Prepaid-only MVP or prepaid plus enterprise invoice support.
- Whether to support OpenAI compatibility only or add native multimodal APIs immediately.
- Data retention default.
- Cloud provider and region strategy.
- Brand positioning: developer router, multimodal creator API, enterprise MaaS platform, or hybrid.
- Detailed API contract, database schema, migration plan, and service-level implementation roadmap.

## 11. Reference Observations

- Atlas Cloud presents itself as a full-modal inference platform with model categories, developer console concepts, API keys, billing, usage history, and per-unit pricing across image and video models.
- OpenRouter emphasizes a unified LLM interface, one API key, no subscription requirement, credit purchase flow, OpenAI-compatible API, provider fallback, cost-aware routing, model catalog, rankings, apps, and enterprise/data policy features.
- SiliconFlow positions itself as a MaaS platform covering language, voice, image, and video APIs, reserved instances, inference acceleration, private deployment, enterprise SLA, monitoring, cost optimization, BYOC, security isolation, and industry solutions.
