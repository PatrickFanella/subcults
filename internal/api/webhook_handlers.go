package api

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/payment"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/webhook"
)

// Stripe webhook event type constants.
const (
	eventCheckoutSessionCompleted = "checkout.session.completed"
	eventPaymentIntentSucceeded   = "payment_intent.succeeded"
	eventPaymentIntentFailed      = "payment_intent.payment_failed"
	eventAccountUpdated           = "account.updated"
)

// WebhookHandlers holds dependencies for webhook-related HTTP handlers.
type WebhookHandlers struct {
	webhookSecret string
	paymentRepo   payment.PaymentRepository
	webhookRepo   payment.WebhookRepository
	sceneRepo     scene.SceneRepository
}

// NewWebhookHandlers creates a new WebhookHandlers instance.
func NewWebhookHandlers(
	webhookSecret string,
	paymentRepo payment.PaymentRepository,
	webhookRepo payment.WebhookRepository,
	sceneRepo scene.SceneRepository,
) *WebhookHandlers {
	return &WebhookHandlers{
		webhookSecret: webhookSecret,
		paymentRepo:   paymentRepo,
		webhookRepo:   webhookRepo,
		sceneRepo:     sceneRepo,
	}
}

// HandleStripeWebhook processes Stripe webhook events with signature verification.
// POST /internal/stripe
func (h *WebhookHandlers) HandleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read the request body (limit to 64KB to prevent abuse)
	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		ctx = middleware.SetErrorCode(ctx, ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "failed to read request body")
		return
	}

	// Get the Stripe signature from the header
	signature := r.Header.Get("Stripe-Signature")
	if signature == "" {
		ctx = middleware.SetErrorCode(ctx, ErrCodeBadRequest)
		WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "missing Stripe-Signature header")
		return
	}

	// Verify the webhook signature
	event, err := webhook.ConstructEvent(body, signature, h.webhookSecret)
	if err != nil {
		// Stripe may deliver events using a newer API version than stripe-go expects.
		// Keep cryptographic signature verification strict, but allow API-version mismatch
		// to avoid unnecessary webhook downtime when endpoint version drift occurs.
		event, err = webhook.ConstructEventWithOptions(body, signature, h.webhookSecret, webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		})
		if err != nil {
			slog.WarnContext(ctx, "webhook signature verification failed", "error", err)
			ctx = middleware.SetErrorCode(ctx, ErrCodeBadRequest)
			WriteError(w, ctx, http.StatusBadRequest, ErrCodeBadRequest, "invalid signature")
			return
		}

		slog.WarnContext(ctx, "webhook accepted with API version mismatch", "error", err)
	}

	// Log minimal event info (type and ID only, not full payload)
	slog.InfoContext(ctx, "webhook event received", "event_type", event.Type, "event_id", event.ID)

	// Check idempotency - has this event already been processed?
	if err := h.webhookRepo.RecordEvent(event.ID, string(event.Type)); err != nil {
		if err == payment.ErrEventAlreadyProcessed {
			slog.InfoContext(ctx, "webhook event already processed, ignoring", "event_id", event.ID)
			// Return 200 to acknowledge receipt even though we're ignoring it
			w.WriteHeader(http.StatusOK)
			return
		}
		// Other errors recording the event
		slog.ErrorContext(ctx, "failed to record webhook event", "event_id", event.ID, "error", err)
		ctx = middleware.SetErrorCode(ctx, ErrCodeInternal)
		WriteError(w, ctx, http.StatusInternalServerError, ErrCodeInternal, "failed to process webhook")
		return
	}

	// Route to appropriate handler based on event type
	switch event.Type {
	case eventCheckoutSessionCompleted:
		h.handleCheckoutSessionCompleted(ctx, event)
	case eventPaymentIntentSucceeded:
		h.handlePaymentIntentSucceeded(ctx, event)
	case eventPaymentIntentFailed:
		h.handlePaymentIntentFailed(ctx, event)
	case eventAccountUpdated:
		h.handleAccountUpdated(ctx, event)
	default:
		// Unknown event type - log and ignore
		slog.InfoContext(ctx, "ignoring unhandled webhook event type", "event_type", event.Type, "event_id", event.ID)
	}

	// Always return 200 to acknowledge receipt
	w.WriteHeader(http.StatusOK)
}

// handleCheckoutSessionCompleted processes checkout.session.completed events.
func (h *WebhookHandlers) handleCheckoutSessionCompleted(ctx context.Context, event stripe.Event) {
	var session stripe.CheckoutSession
	if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
		slog.ErrorContext(ctx, "failed to parse checkout session", "event_id", event.ID, "error", err)
		return
	}

	// Verify the payment record exists by session ID
	_, err := h.paymentRepo.GetBySessionID(session.ID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to get payment record", "session_id", session.ID, "error", err)
		return
	}

	// NOTE:
	// We intentionally do NOT persist any state here. The source of truth for payment
	// status is the payment_intent.succeeded event, which we handle in
	// handlePaymentIntentSucceeded. That handler is responsible for marking the
	// payment record as completed based on the payment intent ID.
	//
	// Historically this handler attempted to "record" the payment intent ID by
	// mutating an in-memory copy of the payment record returned from the
	// repository, but that mutation was never persisted and therefore had no
	// effect. We avoid that pattern now and only use this hook for observability.

	// For payment mode, the payment intent is often created immediately and
	// attached to the checkout session. We log that relationship for debugging
	// and tracing, but defer any state changes to payment_intent.succeeded.
	if session.PaymentIntent != nil && session.PaymentIntent.ID != "" {
		slog.InfoContext(ctx, "checkout session completed, payment intent available",
			"session_id", session.ID,
			"payment_intent_id", session.PaymentIntent.ID)
	} else {
		slog.InfoContext(ctx, "checkout session completed without immediate payment intent",
			"session_id", session.ID)
	}

	// If the mode requires immediate finalization (e.g., for certain payment methods),
	// we could mark as completed here, but typically we wait for the payment_intent.succeeded event.
}

// handlePaymentIntentSucceeded processes payment_intent.succeeded events.
func (h *WebhookHandlers) handlePaymentIntentSucceeded(ctx context.Context, event stripe.Event) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		slog.ErrorContext(ctx, "failed to parse payment intent", "event_id", event.ID, "error", err)
		return
	}

	// Get the checkout session ID from metadata.
	// Note: The metadata.session_id field must be set when creating the checkout session.
	// Since Stripe creates the PaymentIntent after the checkout session is created,
	// we cannot set metadata at session creation time. Instead, in a production
	// implementation with a database, we would query:
	//   SELECT * FROM payment_records WHERE payment_intent_id = paymentIntent.ID
	//
	// For the in-memory implementation, we require session_id in metadata.
	// This is a known limitation documented in stripe.go.
	sessionID := ""
	if paymentIntent.Metadata != nil {
		sessionID = paymentIntent.Metadata["session_id"]
	}

	if sessionID == "" {
		slog.ErrorContext(ctx, "payment intent succeeded but session_id not found in metadata",
			"payment_intent_id", paymentIntent.ID,
			"event_id", event.ID,
			"help", "PaymentIntent must include session_id in metadata, or use database query by payment_intent_id")
		return
	}

	// Mark payment as completed
	if err := h.paymentRepo.MarkCompleted(sessionID, paymentIntent.ID); err != nil {
		if err == payment.ErrPaymentRecordNotFound {
			slog.WarnContext(ctx, "payment record not found for payment intent",
				"session_id", sessionID,
				"payment_intent_id", paymentIntent.ID)
			return
		}
		slog.ErrorContext(ctx, "failed to mark payment completed",
			"session_id", sessionID,
			"payment_intent_id", paymentIntent.ID,
			"error", err)
		return
	}

	slog.InfoContext(ctx, "payment marked as completed",
		"session_id", sessionID,
		"payment_intent_id", paymentIntent.ID,
		"amount", paymentIntent.Amount,
		"currency", paymentIntent.Currency)
}

// handlePaymentIntentFailed processes payment_intent.payment_failed events.
func (h *WebhookHandlers) handlePaymentIntentFailed(ctx context.Context, event stripe.Event) {
	var paymentIntent stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &paymentIntent); err != nil {
		slog.ErrorContext(ctx, "failed to parse payment intent", "event_id", event.ID, "error", err)
		return
	}

	// Get the session ID from metadata (same requirements as handlePaymentIntentSucceeded).
	// See documentation in handlePaymentIntentSucceeded for details on metadata requirements.
	sessionID := ""
	if paymentIntent.Metadata != nil {
		sessionID = paymentIntent.Metadata["session_id"]
	}

	if sessionID == "" {
		slog.ErrorContext(ctx, "payment intent failed but session_id not found in metadata",
			"payment_intent_id", paymentIntent.ID,
			"event_id", event.ID,
			"help", "PaymentIntent must include session_id in metadata, or use database query by payment_intent_id")
		return
	}

	// Extract failure reason
	failureReason := "unknown"
	if paymentIntent.LastPaymentError != nil {
		if paymentIntent.LastPaymentError.Code != "" {
			failureReason = string(paymentIntent.LastPaymentError.Code)
		} else if paymentIntent.LastPaymentError.Msg != "" {
			failureReason = paymentIntent.LastPaymentError.Msg
		}
	}

	// Mark payment as failed
	if err := h.paymentRepo.MarkFailed(sessionID, failureReason); err != nil {
		if err == payment.ErrPaymentRecordNotFound {
			slog.WarnContext(ctx, "payment record not found for failed payment intent",
				"session_id", sessionID,
				"payment_intent_id", paymentIntent.ID)
			return
		}
		slog.ErrorContext(ctx, "failed to mark payment as failed",
			"session_id", sessionID,
			"payment_intent_id", paymentIntent.ID,
			"error", err)
		return
	}

	slog.InfoContext(ctx, "payment marked as failed",
		"session_id", sessionID,
		"payment_intent_id", paymentIntent.ID,
		"reason", failureReason)
}

// handleAccountUpdated processes account.updated events for Connect onboarding completion.
func (h *WebhookHandlers) handleAccountUpdated(ctx context.Context, event stripe.Event) {
	var account stripe.Account
	if err := json.Unmarshal(event.Data.Raw, &account); err != nil {
		slog.ErrorContext(ctx, "failed to parse account", "event_id", event.ID, "error", err)
		return
	}

	// Check if capabilities are now active
	// For Express accounts, we check if transfers capability is active
	transfersActive := account.Capabilities != nil &&
		account.Capabilities.Transfers == stripe.AccountCapabilityStatusActive

	if !transfersActive {
		// Capabilities not yet active, no action needed
		slog.InfoContext(ctx, "account capabilities not yet active",
			"account_id", account.ID,
			"transfers_active", transfersActive)
		return
	}

	// Capabilities are active - find and update associated scenes
	slog.InfoContext(ctx, "account capabilities activated, searching for associated scene",
		"account_id", account.ID,
		"details_submitted", account.DetailsSubmitted,
		"charges_enabled", account.ChargesEnabled)

	scenes, err := h.sceneRepo.ListByConnectedAccountID(account.ID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to query scenes by connected_account_id",
			"account_id", account.ID,
			"error", err)
		return
	}

	if len(scenes) == 0 {
		slog.WarnContext(ctx, "no scenes found for connected account",
			"account_id", account.ID)
		return
	}

	now := time.Now()
	for _, s := range scenes {
		s.ConnectedAccountStatus = "active"
		s.AccountOnboardedAt = &now
		if updateErr := h.sceneRepo.Update(s); updateErr != nil {
			slog.ErrorContext(ctx, "failed to update scene onboarding status",
				"account_id", account.ID,
				"scene_id", s.ID,
				"error", updateErr)
			continue
		}
		slog.InfoContext(ctx, "scene onboarding status updated to active",
			"account_id", account.ID,
			"scene_id", s.ID)
	}
}
