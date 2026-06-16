# Payments

Model Market supports two payment provider modes:

- `mock`: local/dev mode. The backend pretends payment succeeded and immediately posts credits.
- `stripe`: production-style mode. The backend creates a Stripe Checkout Session and only posts credits after a signed Stripe webhook confirms payment.

## Credit Ratio

The current configured ratio is:

```text
1 USD = 100 credits
```

This is stored in `sys_config`:

```text
usd_to_credit_ratio = 100
```

The UI sends purchase amounts as `amount_cents`. The backend calculates credits from the configured ratio and records money as integer cents.

## Configuration

Database config:

```text
payment_provider_mode = mock | stripe
payment_mock_enabled = true | false
usd_to_credit_ratio = 100
```

Environment variables:

```text
PAYMENT_PROVIDER_MODE=mock
STRIPE_SECRET_KEY=
STRIPE_WEBHOOK_SECRET=
PUBLIC_URL=http://localhost:3000
```

`payment_provider_mode` is read from `sys_config`. If the config row is missing, the backend falls back to `PAYMENT_PROVIDER_MODE`, then `mock`.

## Mock Flow

`POST /api/v1/credits/purchase`

Request:

```json
{
  "user_id": "user-yong-zhao",
  "amount_cents": 1000,
  "payment_method": "credit_card"
}
```

When `payment_provider_mode=mock`, the backend:

- Creates `user_credit_purchases` with `status='posted'`.
- Creates `user_payments` with `status='succeeded'`.
- Creates a posted `user_ledger_transactions` row.
- Adds credits to `user_wallets.paid_credits`.

## Stripe Flow

When `payment_provider_mode=stripe`, `POST /api/v1/credits/purchase`:

- Creates a Stripe Checkout Session.
- Stores pending `user_credit_purchases` and `user_payments` rows.
- Returns `checkout_url`.
- Does not update wallet credits yet.

The frontend redirects the user to `checkout_url`.

Stripe sends `checkout.session.completed` to:

```text
POST /api/v1/payments/stripe/webhook
```

The webhook handler:

- Verifies `Stripe-Signature` using `STRIPE_WEBHOOK_SECRET`.
- Ignores non-completed events.
- Ignores unpaid checkout sessions.
- Posts credits once using a ledger idempotency key based on the Stripe Checkout Session ID.
- Updates the pending purchase/payment rows to posted/succeeded.
- Updates `user_wallets.paid_credits`.

## Stripe Webhook Metadata

Checkout Sessions include these metadata keys:

```text
purchase_id
wallet_id
user_id
credits
amount_cents
```

The webhook uses this metadata to post the ledger transaction and credit the correct wallet.

## Notes

- Do not collect or store card numbers in the app. Stripe Checkout handles payment details.
- Do not credit the wallet from the initial purchase request in Stripe mode.
- Webhook processing must remain idempotent because Stripe can retry events.
