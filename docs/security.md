# Security Guide

This document describes the security approach for Model Market. It covers the Phase 1 implementation and the security requirements that must continue through later phases.

## Security Principles

- Do not store model weights or provider proprietary internals.
- Do not log secrets, raw API keys, OAuth tokens, payment data, or provider credentials.
- Use least privilege for database users, provider credentials, payment credentials, and admin accounts.
- Keep customer-facing billing records, API usage, and provider-cost records auditable.
- Treat admin metadata changes, pricing changes, routing changes, and credit changes as sensitive actions.
- Prefer secure defaults for qa/staging/prod, especially database SSL and secret handling.

## Database Security

PostgreSQL is the implemented database for Phase 1.

Environment variables use the `MM_` prefix:

```sh
MM_DB_DRIVER=postgres
MM_DB_HOST=localhost
MM_DB_PORT=5432
MM_DB_NAME=model_market
MM_DB_USER=model_market
MM_DB_PASSWORD=model_market
MM_DB_SSL_MODE=disable
MM_DATABASE_URL=
```

Default SSL behavior:

- `dev`, `test`, `local`: `MM_DB_SSL_MODE=disable`
- `qa`, `staging`, `prod`: `MM_DB_SSL_MODE=require`

For strict production verification:

```sh
MM_DB_SSL_MODE=verify-full
```

Production requirements:

- Use SSL/TLS for database connections.
- Store DB passwords in a secret manager, not in source-controlled files.
- Use a dedicated app database user with only required privileges.
- Use a separate migration/admin user if production migration permissions need to be elevated.
- Enable backups and point-in-time recovery.
- Restrict database network access to backend services.

## SQL Injection Prevention

Backend SQL calls must use parameterized queries.

Correct pattern:

```go
db.QueryContext(ctx, "select id from api_keys where key_hash = $1", keyHash)
```

Avoid:

```go
db.QueryContext(ctx, "select id from api_keys where key_hash = '" + keyHash + "'")
```

Rules:

- Never concatenate user input into SQL strings.
- Use bound parameters for all user/API/admin input.
- Treat IDs, model names, provider names, filters, sort fields, and pagination as untrusted input.
- For dynamic sort/filter fields, use allowlists.
- Keep migration SQL reviewed because it defines access and data boundaries.

Phase 1 note: `devSummary` builds count queries from a hardcoded internal table list, not user input.

## API Keys

API keys are for external developer/API access, not browser dashboard access.

Use cases:

- external apps calling `/api/v1/chat/completions`
- SDK users
- service-to-service API integrations

Rules:

- Store only API key hashes in the database.
- Show raw API keys only once at creation time.
- Prefix keys for identification, but never use prefixes as authentication secrets.
- Support revoke/rotate.
- Scope keys by project and permission.
- Rate-limit by API key.
- Log key ID/prefix where useful, never raw key value.

Phase 1 dev test key:

```text
mk_dev_test_key
```

This key is only for local development and smoke testing.

## Frontend Session Auth vs API Key Auth

There are two different auth paths:

- Frontend/dashboard/workbench:
  - user login/session/OAuth
  - browser calls backend as logged-in user

- Public API:
  - API key in `Authorization: Bearer ...`
  - used by external apps and SDKs

The frontend should not rely on developer API keys for normal dashboard access.

## OAuth Login

Initial OAuth providers:

- Google
- Facebook

OAuth requirements:

- Store OAuth provider secrets only in environment variables or a secret manager.
- Validate OAuth state/nonce to prevent CSRF.
- Validate redirect URIs.
- Link OAuth accounts to users only through verified email or explicit account-linking flow.
- Store provider account ID and metadata, not raw OAuth access/refresh tokens unless required.
- If tokens must be stored later, encrypt them and restrict access.
- Log login events for audit and abuse detection.

## Passwordless and Sessions

Production-mode sessions use random bearer tokens, store only SHA-256 token
hashes in `sys_sessions`, enforce expiration, and can be revoked through the
logout endpoint. The frontend attaches the session token to protected API
requests. New and changed production passwords use bcrypt; successful login
upgrades legacy development hashes.

Remaining passwordless/session requirements:

- Use signed, expiring session tokens.
- Hash session tokens before storing them if persistent sessions are used.
- Use secure cookie flags in production:
  - `HttpOnly`
  - `Secure`
  - `SameSite=Lax` or stricter where possible
- Rotate sessions after privilege changes.
- Support logout and session revocation.

## Secrets Management

Secrets include:

- DB passwords.
- OAuth client secrets.
- Provider API keys.
- Payment provider keys.
- Webhook signing secrets.
- JWT/session signing keys.

Rules:

- Do not commit real secrets.
- Use `.env.example` only for placeholders.
- Use environment variables for local dev.
- Use a secret manager for qa/staging/prod.
- Rotate secrets after suspected exposure.
- Keep provider credential references in the database, not raw secrets, where practical.

## Logging and PII

Structured logs should include:

- request ID
- user ID
- organization ID
- project ID
- API key ID/prefix
- model/provider
- route
- status
- latency
- error class

Never log:

- raw API keys
- OAuth tokens
- provider API keys
- payment card data
- webhook signing secrets
- private file contents
- full sensitive prompts unless explicitly allowed by retention policy

## Payment Security

Payment requirements:

- Use a payment provider such as Stripe or equivalent.
- Do not store raw card numbers.
- Store payment method metadata only, such as provider ID, method type, last four digits, status, and billing email.
- Verify payment webhook signatures.
- Use idempotency keys for payment events, credit grants, refunds, and ledger entries.
- Keep ledger transactions immutable after posting.
- Audit manual credit adjustments and refunds.

Inference charges use a reservation lifecycle. The backend locks the project
wallet and deducts an estimated charge before provider dispatch. On success it
captures the provider-reported final charge and releases any unused credits; on
failure it restores the reservation. Reservation, capture, and release entries
use unique request-based idempotency keys in the immutable ledger.

## Webhook Security

Webhook requirements:

- Sign outbound webhooks.
- Verify inbound payment webhooks.
- Include event IDs and timestamps.
- Prevent replay with event ID/idempotency tracking.
- Retry with backoff.
- Store delivery attempts and status.
- Do not include secrets or unnecessary sensitive content in webhook payloads.

## Rate Limiting and Abuse Protection

Rate limiting dimensions:

- IP address.
- user ID.
- organization ID.
- project ID.
- API key ID.
- endpoint.
- model/provider.
- payment risk state.

Protect:

- login/signup
- OAuth callbacks
- API key creation
- chat completions
- file uploads
- payment actions
- webhooks
- admin endpoints
- metadata manager writes

Production should use edge protection:

- CDN/WAF
- load balancer throttles
- bot controls for public forms
- emergency global throttle for provider outages or attacks

## Admin and Metadata Manager Security

Sensitive admin actions:

- model/provider metadata changes
- model profile changes
- system prompt/default parameter changes
- pricing changes
- provider endpoint changes
- routing/fallback changes
- credit adjustments
- refunds
- account suspension/unblock

Requirements:

- Require admin role checks.
- Record audit logs.
- Support publish/rollback for high-risk metadata.
- Add confirmation for destructive or financial actions.
- Restrict provider credential access.

## File Upload Security

Future file upload requirements:

- Validate file type and size.
- Store files outside the database.
- Use object storage paths, not raw binary DB columns.
- Scan or quarantine risky files where appropriate.
- Prevent path traversal.
- Do not execute uploaded content.
- Apply retention/deletion policies.
- Restrict file access by project/org permissions.

## XSS and Frontend Security

Frontend requirements:

- Escape user-generated content.
- Do not inject raw HTML from prompts, model output, metadata fields, or uploaded files.
- Use a Content Security Policy before production.
- Keep auth tokens out of local storage where possible.
- Avoid exposing provider/payment/OAuth secrets to browser code.
- Validate admin form inputs on both frontend and backend.

## SSRF Protection

Provider endpoints, webhook URLs, and file-processing URLs can create SSRF risk.

Requirements:

- Validate provider endpoint URLs.
- Restrict admin ability to configure internal/private network destinations.
- Block localhost, link-local, and private IP ranges for user-configurable outbound URLs unless explicitly allowed.
- Resolve and validate DNS before outbound calls.
- Apply outbound allowlists for production provider integrations where practical.

## CI Security Checks

CI/release should include:

- Go tests.
- Frontend typecheck/build.
- Dependency vulnerability scanning.
- Container image scanning.
- Secret scanning.
- Static application security testing.
- Migration checks.
- API contract tests.
- Performance smoke tests for critical paths.

High/critical security findings should block production release unless explicitly waived.

## Current Phase 1 Status

Implemented now:

- API key hashing.
- API key auth for public API smoke test.
- Environment-driven DB config with `MM_DB_*`.
- QA/prod DB SSL defaults.
- Generic SQL schema files.
- Local dev test data.
- Unit tests for config and handlers.
- No raw provider/payment/OAuth secrets required for dev.

Not implemented yet:

- real OAuth flow.
- real payment provider/webhook verification.
- production session cookie hardening.
- WAF/CDN integration.
- file upload scanning.
- full admin authorization model.
- automated dependency/container/secret scanning.

These belong in later phases before production launch.
