package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/payment"
	"github.com/onnwee/subcults/internal/scene"
)

// generateStripeSignature generates a valid Stripe webhook signature for testing.
func generateStripeSignature(payload []byte, secret string, timestamp int64) string {
	// Stripe signature format: t=timestamp,v1=signature
	signedPayload := fmt.Sprintf("%d.%s", timestamp, payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	signature := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", timestamp, signature)
}

// TestHandleStripeWebhook_InvalidSignature tests that invalid signatures are rejected.
func TestHandleStripeWebhook_InvalidSignature(t *testing.T) {
	webhookSecret := "whsec_test_secret"
	paymentRepo := payment.NewInMemoryPaymentRepository()
	webhookRepo := payment.NewInMemoryWebhookRepository()
	sceneRepo := scene.NewInMemorySceneRepository()

	handlers := NewWebhookHandlers(webhookSecret, paymentRepo, webhookRepo, sceneRepo)

	// Create a test event
	event := map[string]interface{}{
		"id":   "evt_test123",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "cs_test123",
			},
		},
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/internal/stripe", bytes.NewReader(body))
	// Use an invalid signature
	req.Header.Set("Stripe-Signature", "t=1234567890,v1=invalidsignature")

	w := httptest.NewRecorder()
	handlers.HandleStripeWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeBadRequest {
		t.Errorf("expected error code %s, got %s", ErrCodeBadRequest, errResp.Error.Code)
	}
}

// TestHandleStripeWebhook_MissingSignature tests that missing signature header is rejected.
func TestHandleStripeWebhook_MissingSignature(t *testing.T) {
	webhookSecret := "whsec_test_secret"
	paymentRepo := payment.NewInMemoryPaymentRepository()
	webhookRepo := payment.NewInMemoryWebhookRepository()
	sceneRepo := scene.NewInMemorySceneRepository()

	handlers := NewWebhookHandlers(webhookSecret, paymentRepo, webhookRepo, sceneRepo)

	event := map[string]interface{}{
		"id":   "evt_test123",
		"type": "checkout.session.completed",
	}
	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/internal/stripe", bytes.NewReader(body))
	// No Stripe-Signature header

	w := httptest.NewRecorder()
	handlers.HandleStripeWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

// TestHandleStripeWebhook_ValidSignature tests that valid signatures are accepted.
func TestHandleStripeWebhook_ValidSignature(t *testing.T) {
	webhookSecret := "whsec_test_secret"
	paymentRepo := payment.NewInMemoryPaymentRepository()
	webhookRepo := payment.NewInMemoryWebhookRepository()
	sceneRepo := scene.NewInMemorySceneRepository()

	handlers := NewWebhookHandlers(webhookSecret, paymentRepo, webhookRepo, sceneRepo)

	// Create a test event
	event := map[string]interface{}{
		"id":   "evt_test123",
		"type": "payment_intent.succeeded",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "pi_test123",
				"metadata": map[string]interface{}{
					"session_id": "cs_test123",
				},
			},
		},
	}
	body, _ := json.Marshal(event)

	// Generate valid signature
	timestamp := time.Now().Unix()
	signature := generateStripeSignature(body, webhookSecret, timestamp)

	// Create a payment record to update
	paymentRecord := &payment.PaymentRecord{
		SessionID: "cs_test123",
		Amount:    10000,
		Fee:       500,
		UserDID:   "did:plc:test",
		SceneID:   "scene-1",
		Status:    payment.StatusPending,
	}
	if err := paymentRepo.CreatePending(paymentRecord); err != nil {
		t.Fatalf("failed to create payment record: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/internal/stripe", bytes.NewReader(body))
	req.Header.Set("Stripe-Signature", signature)

	w := httptest.NewRecorder()
	handlers.HandleStripeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify payment was marked as completed
	updated, err := paymentRepo.GetBySessionID("cs_test123")
	if err != nil {
		t.Fatalf("failed to get updated payment: %v", err)
	}

	if updated.Status != payment.StatusSucceeded {
		t.Errorf("expected status %s, got %s", payment.StatusSucceeded, updated.Status)
	}
}

// TestHandleStripeWebhook_Idempotency tests that duplicate events are ignored.
func TestHandleStripeWebhook_Idempotency(t *testing.T) {
	webhookSecret := "whsec_test_secret"
	paymentRepo := payment.NewInMemoryPaymentRepository()
	webhookRepo := payment.NewInMemoryWebhookRepository()
	sceneRepo := scene.NewInMemorySceneRepository()

	handlers := NewWebhookHandlers(webhookSecret, paymentRepo, webhookRepo, sceneRepo)

	// Create a test event
	event := map[string]interface{}{
		"id":   "evt_test123",
		"type": "payment_intent.succeeded",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "pi_test123",
				"metadata": map[string]interface{}{
					"session_id": "cs_test123",
				},
			},
		},
	}
	body, _ := json.Marshal(event)

	timestamp := time.Now().Unix()
	signature := generateStripeSignature(body, webhookSecret, timestamp)

	// Create a payment record
	paymentRecord := &payment.PaymentRecord{
		SessionID: "cs_test123",
		Amount:    10000,
		Fee:       500,
		UserDID:   "did:plc:test",
		SceneID:   "scene-1",
		Status:    payment.StatusPending,
	}
	if err := paymentRepo.CreatePending(paymentRecord); err != nil {
		t.Fatalf("failed to create payment record: %v", err)
	}

	// First request - should process
	req1 := httptest.NewRequest(http.MethodPost, "/internal/stripe", bytes.NewReader(body))
	req1.Header.Set("Stripe-Signature", signature)
	w1 := httptest.NewRecorder()
	handlers.HandleStripeWebhook(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("first request: expected status 200, got %d", w1.Code)
	}

	// Verify payment was marked as completed
	updated, err := paymentRepo.GetBySessionID("cs_test123")
	if err != nil {
		t.Fatalf("failed to get updated payment: %v", err)
	}
	if updated.Status != payment.StatusSucceeded {
		t.Errorf("expected status %s, got %s", payment.StatusSucceeded, updated.Status)
	}

	// Second request with same event - should be ignored (idempotent)
	req2 := httptest.NewRequest(http.MethodPost, "/internal/stripe", bytes.NewReader(body))
	req2.Header.Set("Stripe-Signature", signature)
	w2 := httptest.NewRecorder()
	handlers.HandleStripeWebhook(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("second request: expected status 200, got %d", w2.Code)
	}

	// Verify payment status hasn't changed
	stillUpdated, err := paymentRepo.GetBySessionID("cs_test123")
	if err != nil {
		t.Fatalf("failed to get payment after replay: %v", err)
	}
	if stillUpdated.Status != payment.StatusSucceeded {
		t.Errorf("status changed after replay: expected %s, got %s", payment.StatusSucceeded, stillUpdated.Status)
	}
}

// TestHandleStripeWebhook_CheckoutSessionCompleted tests checkout.session.completed event handling.
func TestHandleStripeWebhook_CheckoutSessionCompleted(t *testing.T) {
	webhookSecret := "whsec_test_secret"
	paymentRepo := payment.NewInMemoryPaymentRepository()
	webhookRepo := payment.NewInMemoryWebhookRepository()
	sceneRepo := scene.NewInMemorySceneRepository()

	handlers := NewWebhookHandlers(webhookSecret, paymentRepo, webhookRepo, sceneRepo)

	// Create a payment record
	paymentRecord := &payment.PaymentRecord{
		SessionID: "cs_test123",
		Amount:    10000,
		Fee:       500,
		UserDID:   "did:plc:test",
		SceneID:   "scene-1",
		Status:    payment.StatusPending,
	}
	if err := paymentRepo.CreatePending(paymentRecord); err != nil {
		t.Fatalf("failed to create payment record: %v", err)
	}

	// Create checkout.session.completed event
	event := map[string]interface{}{
		"id":   "evt_session_completed",
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "cs_test123",
				"payment_intent": map[string]interface{}{
					"id": "pi_test123",
				},
			},
		},
	}
	body, _ := json.Marshal(event)

	timestamp := time.Now().Unix()
	signature := generateStripeSignature(body, webhookSecret, timestamp)

	req := httptest.NewRequest(http.MethodPost, "/internal/stripe", bytes.NewReader(body))
	req.Header.Set("Stripe-Signature", signature)

	w := httptest.NewRecorder()
	handlers.HandleStripeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the event was recorded
	hasProcessed, err := webhookRepo.HasProcessed("evt_session_completed")
	if err != nil {
		t.Fatalf("failed to check if event was processed: %v", err)
	}
	if !hasProcessed {
		t.Error("event should have been recorded as processed")
	}

	// Note: In the current implementation, checkout.session.completed doesn't
	// change the status - we wait for payment_intent.succeeded for that
}

// TestHandleStripeWebhook_PaymentIntentFailed tests payment_intent.payment_failed event handling.
func TestHandleStripeWebhook_PaymentIntentFailed(t *testing.T) {
	webhookSecret := "whsec_test_secret"
	paymentRepo := payment.NewInMemoryPaymentRepository()
	webhookRepo := payment.NewInMemoryWebhookRepository()
	sceneRepo := scene.NewInMemorySceneRepository()

	handlers := NewWebhookHandlers(webhookSecret, paymentRepo, webhookRepo, sceneRepo)

	// Create a payment record
	paymentRecord := &payment.PaymentRecord{
		SessionID: "cs_test456",
		Amount:    10000,
		Fee:       500,
		UserDID:   "did:plc:test",
		SceneID:   "scene-1",
		Status:    payment.StatusPending,
	}
	if err := paymentRepo.CreatePending(paymentRecord); err != nil {
		t.Fatalf("failed to create payment record: %v", err)
	}

	// Create payment_intent.payment_failed event
	event := map[string]interface{}{
		"id":   "evt_payment_failed",
		"type": "payment_intent.payment_failed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "pi_test456",
				"metadata": map[string]interface{}{
					"session_id": "cs_test456",
				},
				"last_payment_error": map[string]interface{}{
					"code":    "card_declined",
					"message": "Your card was declined",
				},
			},
		},
	}
	body, _ := json.Marshal(event)

	timestamp := time.Now().Unix()
	signature := generateStripeSignature(body, webhookSecret, timestamp)

	req := httptest.NewRequest(http.MethodPost, "/internal/stripe", bytes.NewReader(body))
	req.Header.Set("Stripe-Signature", signature)

	w := httptest.NewRecorder()
	handlers.HandleStripeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify payment was marked as failed
	updated, err := paymentRepo.GetBySessionID("cs_test456")
	if err != nil {
		t.Fatalf("failed to get updated payment: %v", err)
	}

	if updated.Status != payment.StatusFailed {
		t.Errorf("expected status %s, got %s", payment.StatusFailed, updated.Status)
	}

	if updated.FailureReason == nil || *updated.FailureReason != "card_declined" {
		t.Errorf("expected failure reason 'card_declined', got %v", updated.FailureReason)
	}
}

// TestHandleStripeWebhook_AccountUpdated tests account.updated event handling.
func TestHandleStripeWebhook_AccountUpdated(t *testing.T) {
	webhookSecret := "whsec_test_secret"
	paymentRepo := payment.NewInMemoryPaymentRepository()
	webhookRepo := payment.NewInMemoryWebhookRepository()
	sceneRepo := scene.NewInMemorySceneRepository()

	handlers := NewWebhookHandlers(webhookSecret, paymentRepo, webhookRepo, sceneRepo)

	// Create account.updated event with active capabilities
	event := map[string]interface{}{
		"id":   "evt_account_updated",
		"type": "account.updated",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id":                "acct_test789",
				"charges_enabled":   true,
				"details_submitted": true,
				"capabilities": map[string]interface{}{
					"transfers": "active",
				},
			},
		},
	}
	body, _ := json.Marshal(event)

	timestamp := time.Now().Unix()
	signature := generateStripeSignature(body, webhookSecret, timestamp)

	req := httptest.NewRequest(http.MethodPost, "/internal/stripe", bytes.NewReader(body))
	req.Header.Set("Stripe-Signature", signature)

	w := httptest.NewRecorder()
	handlers.HandleStripeWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the event was recorded
	hasProcessed, err := webhookRepo.HasProcessed("evt_account_updated")
	if err != nil {
		t.Fatalf("failed to check if event was processed: %v", err)
	}
	if !hasProcessed {
		t.Error("event should have been recorded as processed")
	}

	// Note: Since we don't have a connected_account_status field in the Scene model yet,
	// this test just verifies the event is processed without errors
}

// TestHandleStripeWebhook_UnknownEventType tests that unknown event types are handled gracefully.
func TestHandleStripeWebhook_UnknownEventType(t *testing.T) {
	webhookSecret := "whsec_test_secret"
	paymentRepo := payment.NewInMemoryPaymentRepository()
	webhookRepo := payment.NewInMemoryWebhookRepository()
	sceneRepo := scene.NewInMemorySceneRepository()

	handlers := NewWebhookHandlers(webhookSecret, paymentRepo, webhookRepo, sceneRepo)

	// Create an unknown event type
	event := map[string]interface{}{
		"id":   "evt_unknown",
		"type": "some.unknown.event",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "obj_test",
			},
		},
	}
	body, _ := json.Marshal(event)

	timestamp := time.Now().Unix()
	signature := generateStripeSignature(body, webhookSecret, timestamp)

	req := httptest.NewRequest(http.MethodPost, "/internal/stripe", bytes.NewReader(body))
	req.Header.Set("Stripe-Signature", signature)

	w := httptest.NewRecorder()
	handlers.HandleStripeWebhook(w, req)

	// Should still return 200 (acknowledge receipt)
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	// Verify the event was still recorded for idempotency
	hasProcessed, err := webhookRepo.HasProcessed("evt_unknown")
	if err != nil {
		t.Fatalf("failed to check if event was processed: %v", err)
	}
	if !hasProcessed {
		t.Error("unknown event should still be recorded as processed")
	}
}

// Helper to create a properly formatted Stripe event JSON for testing
func createStripeEventJSON(eventID, eventType string, dataObject map[string]interface{}) []byte {
	event := map[string]interface{}{
		"id":      eventID,
		"type":    eventType,
		"created": time.Now().Unix(),
		"data": map[string]interface{}{
			"object": dataObject,
		},
	}
	body, _ := json.Marshal(event)
	return body
}

// TestWebhookSignatureGeneration validates our test signature generation matches Stripe's format.
func TestWebhookSignatureGeneration(t *testing.T) {
	secret := "whsec_test"
	payload := []byte(`{"id":"evt_test","type":"test"}`)
	timestamp := int64(1234567890)

	sig := generateStripeSignature(payload, secret, timestamp)

	// Signature should have format: t=timestamp,v1=signature
	if len(sig) == 0 {
		t.Error("signature should not be empty")
	}

	// Signature should start with 't='
	if !strings.HasPrefix(sig, "t=") {
		t.Error("signature should start with 't='")
	}

	// Should contain ',v1=' separator
	if !strings.Contains(sig, ",v1=") {
		t.Error("signature should contain ',v1=' component")
	}

	// Parse the signature to extract and validate timestamp
	parts := strings.Split(sig, ",")
	if len(parts) < 2 {
		t.Error("signature should have at least timestamp and v1 components")
	}

	// Extract timestamp from first part (format: t=1234567890)
	if len(parts[0]) < 3 {
		t.Error("timestamp part should be at least 't=X'")
	}
	timestampStr := parts[0][2:] // Remove "t=" prefix
	parsedTimestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		t.Errorf("failed to parse timestamp from signature: %v", err)
	}
	if parsedTimestamp != timestamp {
		t.Errorf("timestamp mismatch: expected %d, got %d", timestamp, parsedTimestamp)
	}

	// Validate v1 signature part exists and is hex-encoded
	v1Found := false
	for _, part := range parts {
		if strings.HasPrefix(part, "v1=") {
			v1Found = true
			sigHex := part[3:] // Remove "v1=" prefix
			// Should be a valid hex string (64 chars for SHA256)
			if len(sigHex) != 64 {
				t.Errorf("v1 signature should be 64 hex chars (SHA256), got %d", len(sigHex))
			}
			// Validate it's valid hex
			if _, err := hex.DecodeString(sigHex); err != nil {
				t.Errorf("v1 signature should be valid hex: %v", err)
			}
			break
		}
	}
	if !v1Found {
		t.Error("signature should contain v1= component")
	}
}
