# Stripe Webhook Handler

## Overview

The Stripe webhook handler processes payment-related events from Stripe with signature verification and idempotency guarantees.

## Endpoint

**POST** `/internal/stripe`

- **Authentication**: Stripe signature verification (via `Stripe-Signature` header)
- **Content-Type**: `application/json`
- **Response**: Always returns 200 OK to acknowledge receipt

## Configuration

Required environment variable:
- `STRIPE_WEBHOOK_SECRET`: Webhook signing secret from Stripe Dashboard

## Supported Events

### 1. `checkout.session.completed`
- Triggered when a checkout session is completed
- Records the payment intent ID for tracking
- Does not change payment status (waits for `payment_intent.succeeded`)

### 2. `payment_intent.succeeded`
- Triggered when payment successfully processes
- Marks payment record as `succeeded`
- Logs amount and currency for observability

### 3. `payment_intent.payment_failed`
- Triggered when payment fails
- Marks payment record as `failed`
- Captures failure reason code from `last_payment_error`

### 4. `account.updated`
- Triggered when Connect account capabilities change
- Logs when account becomes active (transfers capability enabled)
- Future: Will update `connected_account_status` when field is added

## Idempotency

All webhook events are tracked in the `webhook_events` table to prevent duplicate processing:

- Event IDs are stored with timestamps
- Duplicate events return 200 OK without processing
- Thread-safe for concurrent webhook deliveries

## Security

### Signature Verification

All requests must include a valid `Stripe-Signature` header:

```
Stripe-Signature: t=1234567890,v1=<signature_hex>
```

The handler:
1. Reads the raw request body
2. Extracts the signature from headers
3. Verifies using `webhook.ConstructEvent()` with the secret
4. Rejects invalid signatures with 400 Bad Request

### Privacy

- Only minimal event info is logged (type and ID)
- Full event payloads are never logged
- Follows project privacy-first principles

## Error Handling

| Scenario | Response | Description |
|----------|----------|-------------|
| Missing signature | 400 Bad Request | `Stripe-Signature` header required |
| Invalid signature | 400 Bad Request | Signature verification failed |
| Duplicate event | 200 OK | Event already processed (idempotent) |
| Unknown event type | 200 OK | Event acknowledged but not handled |
| Processing error | 200 OK | Error logged, but always ack to Stripe |

## Testing

### Unit Tests

Located in `internal/api/webhook_handlers_test.go`:

- `TestHandleStripeWebhook_InvalidSignature` - Reject invalid signatures
- `TestHandleStripeWebhook_MissingSignature` - Reject missing signatures
- `TestHandleStripeWebhook_ValidSignature` - Accept valid signatures
- `TestHandleStripeWebhook_Idempotency` - Replay protection
- `TestHandleStripeWebhook_CheckoutSessionCompleted` - Session completion flow
- `TestHandleStripeWebhook_PaymentIntentFailed` - Failed payment handling
- `TestHandleStripeWebhook_AccountUpdated` - Account activation tracking
- `TestHandleStripeWebhook_UnknownEventType` - Graceful unknown event handling

### Testing Locally

1. Use [Stripe CLI](https://stripe.com/docs/stripe-cli) to forward events:
   ```bash
   stripe listen --forward-to localhost:8080/internal/stripe
   ```

2. Get the webhook signing secret:
   ```bash
   stripe listen --print-secret
   ```

3. Set environment variable:
   ```bash
   export STRIPE_WEBHOOK_SECRET=whsec_...
   ```

4. Trigger test events:
   ```bash
   stripe trigger payment_intent.succeeded
   stripe trigger payment_intent.payment_failed
   ```

## Database Schema

### `webhook_events` Table

```sql
CREATE TABLE webhook_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id VARCHAR(255) NOT NULL UNIQUE,
    event_type VARCHAR(100) NOT NULL,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Migration: `000018_create_webhook_events.up.sql`

## Future Enhancements

1. **Database Query Support**: Currently payment intents must include `session_id` in metadata. With database support, we can query by `payment_intent_id` directly.

2. **Connected Account Status Field**: Add `connected_account_status` to Scene model to track onboarding state beyond just presence of `connected_account_id`.

3. **Webhook Retry Logic**: Consider exponential backoff for failed processing (currently always returns 200).

4. **Webhook Event Pruning**: Implement cleanup job to remove old webhook events after retention period.

5. **Additional Events**: Support for refunds, disputes, and other payment lifecycle events.

## Troubleshooting

### Webhooks not being received

1. Check webhook endpoint is configured in Stripe Dashboard
2. Verify `STRIPE_WEBHOOK_SECRET` matches the endpoint secret
3. Ensure firewall allows incoming requests from Stripe IPs

### Signature verification failing

1. Verify clock synchronization on server (required for timestamp validation)
2. Check that raw body is passed to verification (no JSON parsing before verification)
3. Confirm secret matches the webhook endpoint (not the API key)

### Payment status not updating

1. Verify payment record exists with matching `session_id`
2. Check logs for event processing errors
3. Confirm event is not being blocked by idempotency check (duplicate event ID)
4. Ensure `metadata.session_id` is set on PaymentIntent (required for lookup)

## References

- [Stripe Webhooks Guide](https://stripe.com/docs/webhooks)
- [Webhook Security](https://stripe.com/docs/webhooks/signatures)
- [Testing Webhooks](https://stripe.com/docs/webhooks/test)
- Issue: subculture-collective/subcults#<issue-number>
