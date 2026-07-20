# Model Market Feature Backlog

This backlog compares the current implementation with the product scope in
`system_requirement.md` and `implementation_plan.md`. Items are ordered by
priority. A checked item means the repository's current MVP scope is complete;
larger production and enterprise extensions can still be tracked separately.

- [ ] 1. Production authentication and authorization: persistent sessions or
  signed tokens, request authentication, project isolation, RBAC, and a
  password KDF such as Argon2id or bcrypt.
- [ ] 2. Real OAuth login: Google and Facebook authorization callbacks,
  identity verification, and account linking.
- [ ] 3. Tenant and project data isolation across projects, conversations,
  assets, usage, company views, and admin APIs.
- [ ] 4. Transactional billing: estimate, reserve, capture, release/refund,
  immutable ledger entries, failure handling, and concurrency safety.
- [ ] 5. Provider fallback and reliability routing: ordered provider attempts,
  retry policy, health-aware selection, and single customer settlement.
- [x] 6. Streaming responses: OpenAI-compatible SSE output, cancellation, and
  final usage settlement.
- [ ] 7. Full OpenAI API compatibility: structured messages and content,
  tools/function calls, response formats, finish reasons, usage, and standard
  error envelopes.
- [ ] 8. Additional real provider adapters for OpenAI, Anthropic, xAI,
  Replicate, and configurable OpenAI-compatible endpoints.
- [ ] 9. Async generation jobs with a queue, worker, progress, retry,
  cancellation, credit reservations, and status APIs.
- [ ] 10. Real object storage and secure uploads with presigned URLs,
  ownership checks, size limits, malware scanning, retention, and private
  downloads.
- [ ] 11. File understanding and context management: extraction, chunks,
  embeddings/retrieval, summaries, pinned messages, and token-aware packing.
- [ ] 12. Conversation editing and collaboration: edit, regenerate, delete,
  export, share, folders, tags, permissions, and retention controls.
- [ ] 13. Model comparison and advanced workbench controls, presets, cost
  estimates, and request/response inspection.
- [ ] 14. Complete metadata administration with CRUD, validation, publishing,
  versioning, rollback, audit history, pricing schedules, and route editing.
- [ ] 15. Team and organization administration: invitations, membership and
  role management, project permissions, organization budgets, and quotas.
- [x] 16. Complete API-key management: expiration, rotation, scope
  enforcement, environment restrictions, budgets, IP restrictions, and rate
  limits.
- [ ] 17. Rate limiting and abuse controls for login, API key/project traffic,
  concurrent requests, uploads, suspensions, and abnormal usage.
- [ ] 18. Payment lifecycle: successful, failed, and expired checkouts;
  refunds, disputes, receipts/invoices, saved methods, automatic top-up, and
  promotional-credit expiration.
- [ ] 19. Outgoing webhook delivery with signing, retries, delivery logs,
  replay, and management APIs.
- [ ] 20. Observability and operations: metrics, traces, provider dashboards,
  queue visibility, anomaly alerts, status pages, incidents, and SLA reports.
- [ ] 21. Rich search, discovery, model details, benchmark rankings,
  availability, regional support, changelogs, samples, and safety policies.
- [ ] 22. SDKs and developer tooling for TypeScript, Python, Go, Java, CLI,
  generated contracts, interactive docs, sandbox mode, and error references.
- [ ] 23. Enterprise capabilities: SAML/SSO, SCIM, MFA, IP allowlists, audit
  exports, zero-data-retention, private endpoints, contracts, and compliance.
- [ ] 24. Provider portal for model submission, endpoint setup, analytics,
  health reports, revenue share, and settlement reporting.
- [ ] 25. Production deployment and security hardening: security scanning,
  migration checks, staging/prod delivery, load tests, disaster recovery, and
  object-storage backups.

## July 2026 implementation progress

The requested MVP work delivered in the current implementation includes:

- Item 4: atomic, row-locked inference settlement that consumes promotional
  credits before paid credits and creates immutable, idempotent ledger rows.
  Credit reservation/release before provider dispatch remains open.
- Item 5: ordered retry to a second active route, latency-aware fallback
  selection, and persisted failed provider attempts when fallback succeeds.
  Circuit breakers and multi-provider health scoring remain open.
- Item 6: OpenAI-compatible `text/event-stream` completion chunks, terminal
  usage, and `[DONE]` framing.
- Item 7: standard error envelopes, OpenAI top-level parameter aliases,
  response format/tool fields, finish reasons, and array-based multimodal text
  content. Actual tool execution and provider-specific structured-output
  enforcement remain open.
- Item 15: organization member listing, role updates/removal, and invitation
  creation/listing. Invitation acceptance, project-specific permissions, and
  organization budget management remain open.
- Item 16: configurable scopes, expiration, rotation, environment, monthly
  budget, IP allowlist, rate limit, last-used tracking, and revocation.
- Item 17: API-key request limiting, scope enforcement, expiration,
  environment and IP restrictions, and monthly budget blocking. Redis-backed
  distributed counters, login/upload throttles, and suspension workflows
  remain open.
- Item 18: payment history, atomic full/partial credit refunds, failed and
  expired Stripe checkout handling, and idempotent successful Stripe posting.
  Provider-side Stripe refund submission, invoices/tax, saved methods,
  automatic top-up execution, disputes, and promotional expiry remain open.

## User-interface launch checklist

- [x] Verify login and signup behavior with backend handler tests.
- [x] Verify project, conversation, branch, message, and asset behavior with
  backend handler tests.
- [x] Support image, audio, and video uploads and library rendering.
- [x] Add organization member role and removal controls to Company Admin.
- [x] Add organization invitation creation and listing to Company Admin.
- [x] Keep company and individual credit-usage views connected to live data.
- [x] Add mock payment history and partial/full refund controls.
- [x] Improve loading, empty, success, and error states for the new workflows.
- [x] Remove public API/API-key navigation and user-facing placeholder copy.
- [x] Make management forms and wide data tables usable on mobile screens.
- [x] Pass backend tests, frontend typecheck/build, Go vet, and diff checks.
- [ ] Run the database-backed demo smoke test. Blocked until PostgreSQL and the
  backend debug service are running.
- [ ] Visually verify the deployed debug UI. Blocked while
  `http://100.98.0.64:3000` is unreachable.

## Independent feature batch

- [x] Add a 25 MB upload limit and explicit safe image/audio/video MIME list.
- [x] Add login throttling with a 15-minute lock after five failed attempts.
- [x] Add conversation rename, permanent delete, and JSON export.
- [x] Add asset deletion with attachment, extraction, and local-file cleanup.
- [x] Add organization invitation acceptance with email verification.
- [x] Add project prompt presets for model, prompt, and output parameters.
- [x] Add Prometheus-style request, error, and latency metrics at `/metrics`.
- [x] Add response request IDs and structured success/rejection/failure logs.
- [x] Add catalog sorting by name, newest, cheapest, provider, and modality.
- [x] Add context window, lifecycle, added date, capabilities, and stored
  descriptions to model details.
- [x] Add dry-run/apply retention cleanup for expired conversations, assets,
  related rows, and local objects.
- [x] Add backend coverage for organization invitations and roles, invitation
  acceptance, refunds, metrics, throttling, upload rejection, presets,
  conversation rename, and asset deletion.
