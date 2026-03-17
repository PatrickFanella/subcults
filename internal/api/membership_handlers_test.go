package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/onnwee/subcults/internal/audit"
	"github.com/onnwee/subcults/internal/membership"
	"github.com/onnwee/subcults/internal/middleware"
	"github.com/onnwee/subcults/internal/scene"
)

func TestRequestMembership_Success(t *testing.T) {
	membershipRepo := membership.NewInMemoryMembershipRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewMembershipHandlers(membershipRepo, sceneRepo, auditRepo)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-123",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "u4pruydqqvj",
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("Failed to insert test scene: %v", err)
	}

	// Create request
	req := httptest.NewRequest("POST", "/scenes/scene-123/membership/request", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:requester")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.RequestMembership(w, req)

	// Verify response
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var result membership.Membership
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.SceneID != "scene-123" {
		t.Errorf("Expected scene_id 'scene-123', got %s", result.SceneID)
	}

	if result.UserDID != "did:plc:requester" {
		t.Errorf("Expected user_did 'did:plc:requester', got %s", result.UserDID)
	}

	if result.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", result.Status)
	}

	if result.Role != "member" {
		t.Errorf("Expected role 'member', got %s", result.Role)
	}

	if result.TrustWeight != 0.5 {
		t.Errorf("Expected trust_weight 0.5, got %f", result.TrustWeight)
	}
}

func TestRequestMembership_DuplicatePending(t *testing.T) {
	membershipRepo := membership.NewInMemoryMembershipRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewMembershipHandlers(membershipRepo, sceneRepo, auditRepo)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-123",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "u4pruydqqvj",
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("Failed to insert test scene: %v", err)
	}

	// Create initial pending membership
	initialMembership := &membership.Membership{
		SceneID:     "scene-123",
		UserDID:     "did:plc:requester",
		Role:        "member",
		Status:      "pending",
		TrustWeight: 0.5,
	}
	if _, err := membershipRepo.Upsert(initialMembership); err != nil {
		t.Fatalf("Failed to create initial membership: %v", err)
	}

	// Try to create another request
	req := httptest.NewRequest("POST", "/scenes/scene-123/membership/request", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:requester")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.RequestMembership(w, req)

	// Verify 409 Conflict
	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d. Body: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeConflict {
		t.Errorf("Expected error code %s, got %s", ErrCodeConflict, errResp.Error.Code)
	}

	if !strings.Contains(errResp.Error.Message, "Pending membership request already exists") {
		t.Errorf("Expected error message about pending request, got: %s", errResp.Error.Message)
	}
}

func TestRequestMembership_OwnerCannotRequest(t *testing.T) {
	membershipRepo := membership.NewInMemoryMembershipRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewMembershipHandlers(membershipRepo, sceneRepo, auditRepo)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-123",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "u4pruydqqvj",
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("Failed to insert test scene: %v", err)
	}

	// Owner tries to request membership
	req := httptest.NewRequest("POST", "/scenes/scene-123/membership/request", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:owner")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.RequestMembership(w, req)

	// Verify 409 Conflict
	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d. Body: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if !strings.Contains(errResp.Error.Message, "owner cannot request membership") {
		t.Errorf("Expected error message about owner, got: %s", errResp.Error.Message)
	}
}

func TestRequestMembership_RejectedCanReapply(t *testing.T) {
	membershipRepo := membership.NewInMemoryMembershipRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewMembershipHandlers(membershipRepo, sceneRepo, auditRepo)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-123",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "u4pruydqqvj",
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("Failed to insert test scene: %v", err)
	}

	// Create rejected membership
	rejectedMembership := &membership.Membership{
		SceneID:     "scene-123",
		UserDID:     "did:plc:requester",
		Role:        "member",
		Status:      "rejected",
		TrustWeight: 0.5,
	}
	if _, err := membershipRepo.Upsert(rejectedMembership); err != nil {
		t.Fatalf("Failed to create rejected membership: %v", err)
	}

	// Try to create new request (should succeed)
	req := httptest.NewRequest("POST", "/scenes/scene-123/membership/request", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:requester")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.RequestMembership(w, req)

	// Verify success
	if w.Code != http.StatusCreated {
		t.Errorf("Expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	var result membership.Membership
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Status != "pending" {
		t.Errorf("Expected status 'pending', got %s", result.Status)
	}
}

func TestApproveMembership_Success(t *testing.T) {
	membershipRepo := membership.NewInMemoryMembershipRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewMembershipHandlers(membershipRepo, sceneRepo, auditRepo)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-123",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "u4pruydqqvj",
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("Failed to insert test scene: %v", err)
	}

	// Create pending membership
	pendingMembership := &membership.Membership{
		SceneID:     "scene-123",
		UserDID:     "did:plc:requester",
		Role:        "member",
		Status:      "pending",
		TrustWeight: 0.5,
	}
	result, err := membershipRepo.Upsert(pendingMembership)
	if err != nil {
		t.Fatalf("Failed to create pending membership: %v", err)
	}

	// Owner approves the membership
	req := httptest.NewRequest("POST", "/scenes/scene-123/membership/did:plc:requester/approve", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:owner")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.ApproveMembership(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var approved membership.Membership
	if err := json.NewDecoder(w.Body).Decode(&approved); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if approved.Status != "active" {
		t.Errorf("Expected status 'active', got %s", approved.Status)
	}

	if approved.Since.IsZero() {
		t.Error("Expected since timestamp to be set")
	}

	// Verify in repository
	retrieved, err := membershipRepo.GetByID(result.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve membership: %v", err)
	}

	if retrieved.Status != "active" {
		t.Errorf("Expected status 'active' in repo, got %s", retrieved.Status)
	}
}

func TestApproveMembership_Unauthorized(t *testing.T) {
	membershipRepo := membership.NewInMemoryMembershipRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewMembershipHandlers(membershipRepo, sceneRepo, auditRepo)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-123",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "u4pruydqqvj",
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("Failed to insert test scene: %v", err)
	}

	// Create pending membership
	pendingMembership := &membership.Membership{
		SceneID:     "scene-123",
		UserDID:     "did:plc:requester",
		Role:        "member",
		Status:      "pending",
		TrustWeight: 0.5,
	}
	if _, err := membershipRepo.Upsert(pendingMembership); err != nil {
		t.Fatalf("Failed to create pending membership: %v", err)
	}

	// Non-owner tries to approve
	req := httptest.NewRequest("POST", "/scenes/scene-123/membership/did:plc:requester/approve", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:attacker")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.ApproveMembership(w, req)

	// Verify 403 Forbidden
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d. Body: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeForbidden {
		t.Errorf("Expected error code %s, got %s", ErrCodeForbidden, errResp.Error.Code)
	}
}

func TestApproveMembership_NotPending(t *testing.T) {
	membershipRepo := membership.NewInMemoryMembershipRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewMembershipHandlers(membershipRepo, sceneRepo, auditRepo)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-123",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "u4pruydqqvj",
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("Failed to insert test scene: %v", err)
	}

	// Create active membership (not pending)
	activeMembership := &membership.Membership{
		SceneID:     "scene-123",
		UserDID:     "did:plc:requester",
		Role:        "member",
		Status:      "active",
		TrustWeight: 0.5,
	}
	if _, err := membershipRepo.Upsert(activeMembership); err != nil {
		t.Fatalf("Failed to create active membership: %v", err)
	}

	// Owner tries to approve active membership
	req := httptest.NewRequest("POST", "/scenes/scene-123/membership/did:plc:requester/approve", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:owner")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.ApproveMembership(w, req)

	// Verify 409 Conflict
	if w.Code != http.StatusConflict {
		t.Errorf("Expected status 409, got %d. Body: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if !strings.Contains(errResp.Error.Message, "pending membership") {
		t.Errorf("Expected error about pending membership, got: %s", errResp.Error.Message)
	}
}

func TestRejectMembership_Success(t *testing.T) {
	membershipRepo := membership.NewInMemoryMembershipRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewMembershipHandlers(membershipRepo, sceneRepo, auditRepo)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-123",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "u4pruydqqvj",
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("Failed to insert test scene: %v", err)
	}

	// Create pending membership
	now := time.Now()
	pendingMembership := &membership.Membership{
		SceneID:     "scene-123",
		UserDID:     "did:plc:requester",
		Role:        "member",
		Status:      "pending",
		TrustWeight: 0.5,
		Since:       now,
	}
	result, err := membershipRepo.Upsert(pendingMembership)
	if err != nil {
		t.Fatalf("Failed to create pending membership: %v", err)
	}

	// Owner rejects the membership
	req := httptest.NewRequest("POST", "/scenes/scene-123/membership/did:plc:requester/reject", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:owner")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.RejectMembership(w, req)

	// Verify response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	var rejected membership.Membership
	if err := json.NewDecoder(w.Body).Decode(&rejected); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if rejected.Status != "rejected" {
		t.Errorf("Expected status 'rejected', got %s", rejected.Status)
	}

	// Verify since timestamp was NOT changed
	if !rejected.Since.Equal(now) {
		t.Errorf("Expected since to remain %v, got %v", now, rejected.Since)
	}

	// Verify in repository
	retrieved, err := membershipRepo.GetByID(result.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve membership: %v", err)
	}

	if retrieved.Status != "rejected" {
		t.Errorf("Expected status 'rejected' in repo, got %s", retrieved.Status)
	}
}

func TestRejectMembership_Unauthorized(t *testing.T) {
	membershipRepo := membership.NewInMemoryMembershipRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewMembershipHandlers(membershipRepo, sceneRepo, auditRepo)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-123",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "u4pruydqqvj",
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("Failed to insert test scene: %v", err)
	}

	// Create pending membership
	pendingMembership := &membership.Membership{
		SceneID:     "scene-123",
		UserDID:     "did:plc:requester",
		Role:        "member",
		Status:      "pending",
		TrustWeight: 0.5,
	}
	if _, err := membershipRepo.Upsert(pendingMembership); err != nil {
		t.Fatalf("Failed to create pending membership: %v", err)
	}

	// Non-owner tries to reject
	req := httptest.NewRequest("POST", "/scenes/scene-123/membership/did:plc:requester/reject", nil)
	ctx := middleware.SetUserDID(req.Context(), "did:plc:attacker")
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	handlers.RejectMembership(w, req)

	// Verify 403 Forbidden
	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d. Body: %s", w.Code, w.Body.String())
	}

	var errResp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}

	if errResp.Error.Code != ErrCodeForbidden {
		t.Errorf("Expected error code %s, got %s", ErrCodeForbidden, errResp.Error.Code)
	}
}

func TestMembershipEnumerationProtection(t *testing.T) {
	membershipRepo := membership.NewInMemoryMembershipRepository()
	sceneRepo := scene.NewInMemorySceneRepository()
	auditRepo := audit.NewInMemoryRepository()
	handlers := NewMembershipHandlers(membershipRepo, sceneRepo, auditRepo)

	// Create a test scene
	testScene := &scene.Scene{
		ID:            "scene-123",
		Name:          "Test Scene",
		OwnerDID:      "did:plc:owner",
		CoarseGeohash: "u4pruydqqvj",
	}
	if err := sceneRepo.Insert(testScene); err != nil {
		t.Fatalf("Failed to insert test scene: %v", err)
	}

	// Test 1: Non-existent user should return same error as forbidden
	req1 := httptest.NewRequest("POST", "/scenes/scene-123/membership/did:plc:nonexistent/approve", nil)
	ctx1 := middleware.SetUserDID(req1.Context(), "did:plc:attacker")
	req1 = req1.WithContext(ctx1)

	w1 := httptest.NewRecorder()
	handlers.ApproveMembership(w1, req1)

	// Test 2: Try to approve with wrong owner
	pendingMembership := &membership.Membership{
		SceneID:     "scene-123",
		UserDID:     "did:plc:requester",
		Role:        "member",
		Status:      "pending",
		TrustWeight: 0.5,
	}
	if _, err := membershipRepo.Upsert(pendingMembership); err != nil {
		t.Fatalf("Failed to create pending membership: %v", err)
	}

	req2 := httptest.NewRequest("POST", "/scenes/scene-123/membership/did:plc:requester/approve", nil)
	ctx2 := middleware.SetUserDID(req2.Context(), "did:plc:attacker")
	req2 = req2.WithContext(ctx2)

	w2 := httptest.NewRecorder()
	handlers.ApproveMembership(w2, req2)

	// Both should return 403 with similar response format
	if w1.Code != http.StatusForbidden {
		t.Errorf("Test 1: Expected status 403, got %d", w1.Code)
	}

	if w2.Code != http.StatusForbidden {
		t.Errorf("Test 2: Expected status 403, got %d", w2.Code)
	}

	// Verify timing consistency by ensuring both use uniform error messages
	var err1, err2 ErrorResponse
	if e := json.NewDecoder(w1.Body).Decode(&err1); e != nil {
		t.Fatalf("Failed to decode error 1: %v", e)
	}
	if e := json.NewDecoder(w2.Body).Decode(&err2); e != nil {
		t.Fatalf("Failed to decode error 2: %v", e)
	}

	// Both should have same error code to prevent enumeration
	if err1.Error.Code != err2.Error.Code {
		t.Errorf("Error codes differ: %s vs %s (potential enumeration)", err1.Error.Code, err2.Error.Code)
	}
}
