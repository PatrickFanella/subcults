package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/onnwee/subcults/internal/idempotency"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/payment"
	"github.com/onnwee/subcults/internal/scene"
	"github.com/stripe/stripe-go/v81"
)

// TestCreateCheckoutSession_WithIdempotency tests that duplicate requests with the same idempotency key
// return the same response and only create one payment record.
func TestCreateCheckoutSession_WithIdempotency(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	paymentRepo := payment.NewInMemoryPaymentRepository()
	idempotencyRepo := idempotency.NewInMemoryRepository()

	// Track how many times checkout session is created
	createCount := 0
	mockClient := &mockStripeClient{
		createCheckoutSessionFunc: func(params *payment.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
			createCount++
			return &stripe.CheckoutSession{
				ID:  "cs_test123",
				URL: "https://checkout.stripe.com/pay/cs_test123",
			}, nil
		},
	}

	handlers := NewPaymentHandlers(
		sceneRepo,
		paymentRepo,
		mockClient,
		"https://example.com/return",
		"https://example.com/refresh",
		5.0,
	)

	// Create a test scene with connected account
	existingAccountID := "acct_test123"
	testScene := &scene.Scene{
		ID:                 "scene-1",
		Name:               "Test Scene",
		OwnerDID:           "did:plc:owner123",
		CoarseGeohash:      "dr5regw",
		ConnectedAccountID: &existingAccountID,
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to create test scene: %v", err)
	}

	// Setup idempotency middleware
	routes := map[string]bool{"/payments/checkout": true}
	idempotencyMW := middleware.IdempotencyMiddleware(idempotencyRepo, routes)
	handler := idempotencyMW(http.HandlerFunc(handlers.CreateCheckoutSession))

	reqBody := CheckoutSessionRequest{
		SceneID: "scene-1",
		Items: []CheckoutItemRequest{
			{PriceID: "price_test123", Quantity: 2},
		},
		SuccessURL: "https://example.com/success",
		CancelURL:  "https://example.com/cancel",
	}
	body, _ := json.Marshal(reqBody)

	// First request
	req1 := httptest.NewRequest(http.MethodPost, "/payments/checkout", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(middleware.IdempotencyKeyHeader, "test-idempotency-key-1")
	ctx1 := middleware.SetUserDID(req1.Context(), "did:plc:owner123")
	req1 = req1.WithContext(ctx1)

	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("first request: expected status 200, got %d: %s", w1.Code, w1.Body.String())
	}

	var response1 CheckoutSessionResponse
	if err := json.NewDecoder(w1.Body).Decode(&response1); err != nil {
		t.Fatalf("failed to decode first response: %v", err)
	}

	if response1.SessionID != "cs_test123" {
		t.Errorf("expected session ID cs_test123, got %s", response1.SessionID)
	}

	// Second request with same idempotency key
	body2, _ := json.Marshal(reqBody)
	req2 := httptest.NewRequest(http.MethodPost, "/payments/checkout", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(middleware.IdempotencyKeyHeader, "test-idempotency-key-1")
	ctx2 := middleware.SetUserDID(req2.Context(), "did:plc:owner123")
	req2 = req2.WithContext(ctx2)

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("second request: expected status 200, got %d: %s", w2.Code, w2.Body.String())
	}

	var response2 CheckoutSessionResponse
	if err := json.NewDecoder(w2.Body).Decode(&response2); err != nil {
		t.Fatalf("failed to decode second response: %v", err)
	}

	// Responses should be identical
	if response1.SessionID != response2.SessionID {
		t.Errorf("session IDs don't match: %s vs %s", response1.SessionID, response2.SessionID)
	}
	if response1.SessionURL != response2.SessionURL {
		t.Errorf("session URLs don't match: %s vs %s", response1.SessionURL, response2.SessionURL)
	}

	// Stripe checkout session should only be created once
	if createCount != 1 {
		t.Errorf("expected Stripe checkout session to be created once, but was created %d times", createCount)
	}

	// Only one payment record should exist
	paymentRecord, err := paymentRepo.GetBySessionID("cs_test123")
	if err != nil {
		t.Fatalf("expected payment record to exist: %v", err)
	}
	if paymentRecord.SessionID != "cs_test123" {
		t.Errorf("expected payment record session ID cs_test123, got %s", paymentRecord.SessionID)
	}
}

// TestCreateCheckoutSession_MissingIdempotencyKey tests that requests without idempotency key are rejected.
func TestCreateCheckoutSession_MissingIdempotencyKey(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	paymentRepo := payment.NewInMemoryPaymentRepository()
	idempotencyRepo := idempotency.NewInMemoryRepository()
	mockClient := &mockStripeClient{}

	handlers := NewPaymentHandlers(
		sceneRepo,
		paymentRepo,
		mockClient,
		"https://example.com/return",
		"https://example.com/refresh",
		5.0,
	)

	// Create a test scene with connected account
	existingAccountID := "acct_test123"
	testScene := &scene.Scene{
		ID:                 "scene-1",
		Name:               "Test Scene",
		OwnerDID:           "did:plc:owner123",
		CoarseGeohash:      "dr5regw",
		ConnectedAccountID: &existingAccountID,
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to create test scene: %v", err)
	}

	// Setup idempotency middleware
	routes := map[string]bool{"/payments/checkout": true}
	idempotencyMW := middleware.IdempotencyMiddleware(idempotencyRepo, routes)
	handler := idempotencyMW(http.HandlerFunc(handlers.CreateCheckoutSession))

	reqBody := CheckoutSessionRequest{
		SceneID: "scene-1",
		Items: []CheckoutItemRequest{
			{PriceID: "price_test123", Quantity: 2},
		},
		SuccessURL: "https://example.com/success",
		CancelURL:  "https://example.com/cancel",
	}
	body, _ := json.Marshal(reqBody)

	// Request without idempotency key
	req := httptest.NewRequest(http.MethodPost, "/payments/checkout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// Note: NOT setting Idempotency-Key header
	ctx := middleware.SetUserDID(req.Context(), "did:plc:owner123")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var errorResp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&errorResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResp["error"] != "missing_idempotency_key" {
		t.Errorf("expected error code 'missing_idempotency_key', got %v", errorResp["error"])
	}
}

// TestCreateCheckoutSession_IdempotencyKeyTooLong tests that overly long idempotency keys are rejected.
func TestCreateCheckoutSession_IdempotencyKeyTooLong(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	paymentRepo := payment.NewInMemoryPaymentRepository()
	idempotencyRepo := idempotency.NewInMemoryRepository()
	mockClient := &mockStripeClient{}

	handlers := NewPaymentHandlers(
		sceneRepo,
		paymentRepo,
		mockClient,
		"https://example.com/return",
		"https://example.com/refresh",
		5.0,
	)

	// Create a test scene with connected account
	existingAccountID := "acct_test123"
	testScene := &scene.Scene{
		ID:                 "scene-1",
		Name:               "Test Scene",
		OwnerDID:           "did:plc:owner123",
		CoarseGeohash:      "dr5regw",
		ConnectedAccountID: &existingAccountID,
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to create test scene: %v", err)
	}

	// Setup idempotency middleware
	routes := map[string]bool{"/payments/checkout": true}
	idempotencyMW := middleware.IdempotencyMiddleware(idempotencyRepo, routes)
	handler := idempotencyMW(http.HandlerFunc(handlers.CreateCheckoutSession))

	reqBody := CheckoutSessionRequest{
		SceneID: "scene-1",
		Items: []CheckoutItemRequest{
			{PriceID: "price_test123", Quantity: 2},
		},
		SuccessURL: "https://example.com/success",
		CancelURL:  "https://example.com/cancel",
	}
	body, _ := json.Marshal(reqBody)

	// Request with key that's too long (>64 chars)
	longKey := strings.Repeat("a", idempotency.MaxKeyLength+1)

	req := httptest.NewRequest(http.MethodPost, "/payments/checkout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set(middleware.IdempotencyKeyHeader, longKey)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:owner123")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d: %s", w.Code, w.Body.String())
	}

	var errorResp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&errorResp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}

	if errorResp["error"] != "idempotency_key_too_long" {
		t.Errorf("expected error code 'idempotency_key_too_long', got %v", errorResp["error"])
	}
}

// TestCreateCheckoutSession_DifferentIdempotencyKeys tests that different idempotency keys create separate records.
func TestCreateCheckoutSession_DifferentIdempotencyKeys(t *testing.T) {
	sceneRepo := scene.NewInMemorySceneRepository()
	paymentRepo := payment.NewInMemoryPaymentRepository()
	idempotencyRepo := idempotency.NewInMemoryRepository()

	sessionCounter := 0
	mockClient := &mockStripeClient{
		createCheckoutSessionFunc: func(params *payment.CheckoutSessionParams) (*stripe.CheckoutSession, error) {
			sessionCounter++
			sessionID := "cs_test" + string(rune('0'+sessionCounter))
			return &stripe.CheckoutSession{
				ID:  sessionID,
				URL: "https://checkout.stripe.com/pay/" + sessionID,
			}, nil
		},
	}

	handlers := NewPaymentHandlers(
		sceneRepo,
		paymentRepo,
		mockClient,
		"https://example.com/return",
		"https://example.com/refresh",
		5.0,
	)

	// Create a test scene with connected account
	existingAccountID := "acct_test123"
	testScene := &scene.Scene{
		ID:                 "scene-1",
		Name:               "Test Scene",
		OwnerDID:           "did:plc:owner123",
		CoarseGeohash:      "dr5regw",
		ConnectedAccountID: &existingAccountID,
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("failed to create test scene: %v", err)
	}

	// Setup idempotency middleware
	routes := map[string]bool{"/payments/checkout": true}
	idempotencyMW := middleware.IdempotencyMiddleware(idempotencyRepo, routes)
	handler := idempotencyMW(http.HandlerFunc(handlers.CreateCheckoutSession))

	reqBody := CheckoutSessionRequest{
		SceneID: "scene-1",
		Items: []CheckoutItemRequest{
			{PriceID: "price_test123", Quantity: 2},
		},
		SuccessURL: "https://example.com/success",
		CancelURL:  "https://example.com/cancel",
	}

	// First request with key1
	body1, _ := json.Marshal(reqBody)
	req1 := httptest.NewRequest(http.MethodPost, "/payments/checkout", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set(middleware.IdempotencyKeyHeader, "key-1")
	ctx1 := middleware.SetUserDID(req1.Context(), "did:plc:owner123")
	req1 = req1.WithContext(ctx1)

	w1 := httptest.NewRecorder()
	handler.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("first request: expected status 200, got %d", w1.Code)
	}

	// Second request with different key (key-2)
	body2, _ := json.Marshal(reqBody)
	req2 := httptest.NewRequest(http.MethodPost, "/payments/checkout", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set(middleware.IdempotencyKeyHeader, "key-2")
	ctx2 := middleware.SetUserDID(req2.Context(), "did:plc:owner123")
	req2 = req2.WithContext(ctx2)

	w2 := httptest.NewRecorder()
	handler.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("second request: expected status 200, got %d", w2.Code)
	}

	// Responses should be different (different session IDs)
	var response1, response2 CheckoutSessionResponse
	json.NewDecoder(w1.Body).Decode(&response1)
	json.NewDecoder(w2.Body).Decode(&response2)

	if response1.SessionID == response2.SessionID {
		t.Errorf("different idempotency keys should create different sessions, but both got %s", response1.SessionID)
	}

	// Two Stripe sessions should have been created
	if sessionCounter != 2 {
		t.Errorf("expected 2 Stripe sessions to be created, got %d", sessionCounter)
	}
}
