# Inference billing

Inference prices in `sys_price_rules.price` are credits per configured unit.
Provider prices are USD per unit and are converted using
`usd_to_credit_ratio`. Token rules use units of 1,000 tokens. Wallet balances,
ledger changes, inference charges, message charges, usage and asset charges
store up to eight decimal places rather than rounding each request up to a
whole credit. For example, 1,000 input tokens at 0.025 credits/1k tokens plus
100 output tokens at 0.15 credits/1k tokens costs 0.04 credits.

## Existing database upgrade

Before deploying this backend against an existing database, apply
`db/migrations/20261017_fractional_billing.sql` with your normal database
administration process. This migration changes column precision and adds API
key creator attribution. It preserves existing balances and records without
rescaling them. Fresh installations use the updated `db/init_db.sql`.

Do not use the development schema reset to upgrade a database whose data you
need to preserve. The migration has not been applied automatically.

## Price selection and settlement

- Requests without active prices are rejected before provider execution.
- Future prices and superseded effective versions are excluded. A request
  keeps its selected rules and conversion ratio through final settlement.
- Profile prices override model prices for the same billable dimension.
  Alternative resolution/quality prices are selected, not added together.
  Ambiguous or unmatched alternatives are rejected. `pricing_variant` can
  explicitly select a variant; resolution, quality and size metadata also
  constrain selection.
- Positive integer quantities are required. Mock image counts are limited to
  four; the current Gemini image adapter supports exactly one image.
- Credits are reserved before execution, with a separate reservation when
  falling back to another route. Actual reported token usage determines the
  final charge. Real providers must return usage rather than silently falling
  back to message counts.
- Final capture, inference records, provider attempts, conversation messages,
  usage events and generated-asset database records commit together. A failure
  rolls back settlement and the handler attempts to release the reservation.
  Failed releases are logged.
- Available credit comes directly from the remaining wallet balance. Historical
  usage is not deducted a second time.
- New API keys record their creator; their requests are attributed to that
  user. In development, the frontend supplies the creator when making the key.
  Production derives it from the authenticated session. Existing keys without
  an owner remain visible in project totals but do not invent a user identity.
  Old unattributed usage is not backfilled. Empty user histories stay empty.
- The UI shows the selected project's actual remaining wallet balance and never
  generates fake usage or purchases on API failures. History charts show posted
  purchases minus recorded usage within the selected period, starting at zero;
  they are not historical wallet balances and exclude refunds, promotional
  adjustments and reservations.

## Remaining limitations

This does not establish billing support for every catalog model. Real media
adapters beyond the implemented Gemini image path, provider-specific cached,
reasoning and audio token rate breakdowns, and reconciliation against actual
provider invoices still require work. Unsupported token price types and units
are rejected rather than guessed. Configured provider costs are estimates
based on stored rates, not invoice amounts.

Object bytes live outside the database transaction; a later database rollback
can leave an unreferenced stored object. Automatic recovery of reservations
after a process crash, client request idempotency, and uncertain database
commit outcomes also require a durable reconciliation workflow. Credit
calculations use Go floating-point values rounded to eight decimal places;
the database stores fixed-scale numeric values. Historical whole-credit
charges are retained unchanged.
