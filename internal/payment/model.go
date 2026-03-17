// Package payment provides models and services for payment processing.
package payment

import "time"

// PaymentStatus represents the status of a payment record.
const (
	StatusPending   = "pending"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCanceled  = "canceled"
	StatusRefunded  = "refunded"
)

// PaymentRecord represents a provisional payment record for a Stripe Checkout Session.
type PaymentRecord struct {
	ID                 string     `json:"id"`
	SessionID          string     `json:"session_id"`                     // Stripe Checkout Session ID
	Status             string     `json:"status"`                         // pending, succeeded, failed, canceled, refunded
	Amount             int64      `json:"amount"`                         // Total amount in cents
	Fee                int64      `json:"fee"`                            // Platform fee in cents
	Currency           string     `json:"currency,omitempty"`             // ISO 4217 currency code (defaults to 'usd' if empty)
	UserDID            string     `json:"user_did"`                       // User making the payment
	SceneID            string     `json:"scene_id"`                       // Scene receiving payment
	EventID            *string    `json:"event_id,omitempty"`             // Optional event ID
	ConnectedAccountID *string    `json:"connected_account_id,omitempty"` // Stripe Connect account ID
	PaymentIntentID    *string    `json:"payment_intent_id,omitempty"`    // Stripe Payment Intent ID
	FailureReason      *string    `json:"failure_reason,omitempty"`       // Reason for failure
	CreatedAt          *time.Time `json:"created_at,omitempty"`
	UpdatedAt          *time.Time `json:"updated_at,omitempty"`
}

// DeepCopy creates a deep copy of the PaymentRecord, including all pointer fields.
func (p *PaymentRecord) DeepCopy() *PaymentRecord {
	if p == nil {
		return nil
	}

	copied := &PaymentRecord{
		ID:        p.ID,
		SessionID: p.SessionID,
		Status:    p.Status,
		Amount:    p.Amount,
		Fee:       p.Fee,
		Currency:  p.Currency,
		UserDID:   p.UserDID,
		SceneID:   p.SceneID,
	}

	if p.EventID != nil {
		eventID := *p.EventID
		copied.EventID = &eventID
	}
	if p.ConnectedAccountID != nil {
		accountID := *p.ConnectedAccountID
		copied.ConnectedAccountID = &accountID
	}
	if p.PaymentIntentID != nil {
		intentID := *p.PaymentIntentID
		copied.PaymentIntentID = &intentID
	}
	if p.FailureReason != nil {
		reason := *p.FailureReason
		copied.FailureReason = &reason
	}
	if p.CreatedAt != nil {
		createdAt := *p.CreatedAt
		copied.CreatedAt = &createdAt
	}
	if p.UpdatedAt != nil {
		updatedAt := *p.UpdatedAt
		copied.UpdatedAt = &updatedAt
	}

	return copied
}
