# Payment Status Transitions

## Overview
The payment system enforces strict status transitions to ensure data integrity and prevent invalid state changes.

## Status Constants
- `pending`: Initial state when payment session is created
- `succeeded`: Payment completed successfully
- `failed`: Payment failed (e.g., card declined, insufficient funds)
- `canceled`: Payment was canceled by user or system
- `refunded`: Successful payment was refunded

## Valid Transitions

### From `pending`
- ✅ `pending` → `succeeded` (via `MarkCompleted`)
- ✅ `pending` → `failed` (via `MarkFailed`)
- ✅ `pending` → `canceled`

### From `succeeded`
- ✅ `succeeded` → `refunded`

### Invalid Transitions
All other transitions are **not allowed**:
- ❌ `succeeded` → `pending` (cannot un-complete)
- ❌ `succeeded` → `failed` (cannot fail after success)
- ❌ `failed` → `succeeded` (cannot succeed after failure)
- ❌ `failed` → `pending` (cannot reset failed payment)
- ❌ `canceled` → any status (canceled is terminal)
- ❌ `refunded` → any status (refunded is terminal)

## Repository Methods

### `CreatePending(record *PaymentRecord) error`
Creates a new payment record in pending status.
- **Returns**: `ErrDuplicateSessionID` if session_id already exists
- **Idempotency**: N/A (always creates new record)
- **Thread-safe**: Yes

### `MarkCompleted(sessionID, paymentIntentID string) error`
Transitions a payment from pending to succeeded.
- **Returns**: 
  - `ErrPaymentRecordNotFound` if session doesn't exist
  - `ErrInvalidStatusTransition` if not in pending status
  - `ErrPaymentIntentMismatch` if already succeeded with a different payment intent ID
- **Idempotency**: Yes (returns nil if already succeeded with same payment intent ID)
- **Thread-safe**: Yes
- **Side effects**: Stores `payment_intent_id`, updates `updated_at`

### `MarkFailed(sessionID, reason string) error`
Transitions a payment from pending to failed.
- **Returns**:
  - `ErrPaymentRecordNotFound` if session doesn't exist
  - `ErrInvalidStatusTransition` if not in pending status
- **Idempotency**: Yes (returns nil if already failed with same reason)
- **Thread-safe**: Yes
- **Side effects**: 
  - Stores `failure_reason`, updates `updated_at`
  - Updates reason if already failed with different reason

### `MarkCanceled(sessionID string) error`
Transitions a payment from pending to canceled.
- **Returns**:
  - `ErrPaymentRecordNotFound` if session doesn't exist
  - `ErrInvalidStatusTransition` if not in pending status
- **Idempotency**: Yes (returns nil if already canceled)
- **Thread-safe**: Yes
- **Side effects**: Updates `updated_at`

### `MarkRefunded(sessionID string) error`
Transitions a payment from succeeded to refunded.
- **Returns**:
  - `ErrPaymentRecordNotFound` if session doesn't exist
  - `ErrInvalidStatusTransition` if not in succeeded status
- **Idempotency**: Yes (returns nil if already refunded)
- **Thread-safe**: Yes
- **Side effects**: Updates `updated_at`

## Example Usage

```go
repo := payment.NewInMemoryPaymentRepository()

// Create pending payment
record := &payment.PaymentRecord{
    SessionID: "cs_test_abc123",
    Amount:    1000, // $10.00
    Fee:       100,  // $1.00 platform fee
    Currency:  "usd",
    UserDID:   "did:plc:user123",
    SceneID:   "scene-uuid",
}
err := repo.CreatePending(record)

// Mark as completed
err = repo.MarkCompleted("cs_test_abc123", "pi_test_xyz789")

// Or mark as failed
err = repo.MarkFailed("cs_test_abc123", "card_declined")
```

## Database Schema
See `migrations/000017_create_payment_records.up.sql` for the complete schema definition.

Key features:
- `session_id` has UNIQUE constraint (prevents duplicates at DB level)
- `status` has CHECK constraint (validates enum values at DB level)
- Partial index on `status='pending'` for efficient pending payment queries
- Foreign keys to `scenes` and `events` with appropriate cascading

## Testing
All status transitions are thoroughly tested in `internal/payment/repository_test.go`:
- Table-driven tests for all 20 possible transitions
- Idempotency tests for `MarkCompleted` and `MarkFailed`
- Duplicate session_id rejection tests
- Thread-safety verified with race detector

Coverage: 70.3%
